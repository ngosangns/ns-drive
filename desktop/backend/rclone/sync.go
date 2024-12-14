package rclone

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"

	"log"

	be "desktop/backend"

	env "github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	// import fs drivers
	_ "github.com/rclone/rclone/backend/drive"
	_ "github.com/rclone/rclone/backend/local"
	"github.com/rclone/rclone/cmd"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/accounting"
	"github.com/rclone/rclone/fs/cache"
	"github.com/rclone/rclone/fs/filter"
	"github.com/rclone/rclone/fs/fserrors"
	"github.com/rclone/rclone/fs/rc"
	"github.com/rclone/rclone/fs/rc/rcserver"
	fssync "github.com/rclone/rclone/fs/sync"
	"github.com/rclone/rclone/lib/atexit"
	"github.com/rclone/rclone/lib/exitcode"
	"github.com/rclone/rclone/lib/terminal"

	"github.com/rclone/rclone/fs/config/configfile"
	fslog "github.com/rclone/rclone/fs/log"
)

func initConfig(ctx context.Context) context.Context {
	// Set the global options from the flags
	err := fs.GlobalOptionsInit()
	if err != nil {
		fs.Fatalf(nil, "Failed to initialise global options: %v", err)
	}

	// Start the logger
	fslog.InitLogging()

	// Load the config
	// config.SetConfigPath(filepath.Join(dir, "rclone.conf"))
	configfile.Install()

	// Start accounting
	accounting.Start(ctx)

	// Initialize the global config
	fsConfig := fs.GetConfig(ctx)

	// Configure console
	if fsConfig.NoConsole {
		// Hide the console window
		terminal.HideConsole()
	} else {
		// Enable color support on stdout if possible.
		// This enables virtual terminal processing on Windows 10,
		// adding native support for ANSI/VT100 escape sequences.
		terminal.EnableColorsStdout()
	}

	// Write the args for debug purposes
	fs.Debugf("rclone", "Version %q starting with parameters %q", fs.Version, os.Args)

	// Inform user about systemd log support now that we have a logger
	if fslog.Opt.LogSystemdSupport {
		fs.Debugf("rclone", "systemd logging support activated")
	}

	// Start the metrics server if configured
	_, err = rcserver.MetricsStart(ctx, &rc.Opt)
	if err != nil {
		fs.Fatalf(nil, "Failed to start metrics server: %v", err)

	}

	err = fsConfig.StatsLogLevel.Set("INFO") // EMERGENCY ALERT CRITICAL ERROR WARNING NOTICE INFO DEBUG
	if err != nil {
		log.Fatalf("Failed to set stats log level: %v", err)
	}

	err = fsConfig.LogLevel.Set("INFO")
	if err != nil {
		log.Fatalf("Failed to set log level: %v", err)
	}

	// Setup the default filters
	filterOpts := filter.Options{
		MinAge:  fs.DurationOff, // These have to be set here as the options are parsed once before the defaults are set
		MaxAge:  fs.DurationOff,
		MinSize: fs.SizeSuffix(-1),
		MaxSize: fs.SizeSuffix(-1),
	}
	filterConfig, err := filter.NewFilter(&filterOpts)
	if err != nil {
		log.Fatalf("Failed to load filter file: %v", err)
	}
	ctx = filter.ReplaceConfig(ctx, filterConfig)

	// Setup CPU profiling if desired
	cpuProfile := ""
	// cpuProfile := "Debugging"
	if cpuProfile != "" {
		fs.Infof(nil, "Creating CPU profile %q\n", cpuProfile)
		f, err := os.Create(cpuProfile)
		if err != nil {
			err = fs.CountError(err)
			fs.Fatal(nil, fmt.Sprint(err))
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			err = fs.CountError(err)
			fs.Fatal(nil, fmt.Sprint(err))
		}
		atexit.Register(func() {
			pprof.StopCPUProfile()
			err := f.Close()
			if err != nil {
				err = fs.CountError(err)
				fs.Fatal(nil, fmt.Sprint(err))
			}
		})
	}

	// Setup memory profiling if desired
	memProfile := ""
	// memProfile := "Debugging"
	if memProfile != "" {
		atexit.Register(func() {
			fs.Infof(nil, "Saving Memory profile %q\n", memProfile)
			f, err := os.Create(memProfile)
			if err != nil {
				err = fs.CountError(err)
				fs.Fatal(nil, fmt.Sprint(err))
			}
			err = pprof.WriteHeapProfile(f)
			if err != nil {
				err = fs.CountError(err)
				fs.Fatal(nil, fmt.Sprint(err))
			}
			err = f.Close()
			if err != nil {
				err = fs.CountError(err)
				fs.Fatal(nil, fmt.Sprint(err))
			}
		})
	}

	return ctx
}

func Run(ctx context.Context, Retry bool, showStats bool, cb func() error) {
	ci := fs.GetConfig(ctx)
	var cmdErr error
	stopStats := func() {}
	if !showStats && cmd.ShowStats() {
		showStats = true
	}
	if ci.Progress {
		stopStats = StartProgress()
	} else if showStats {
		stopStats = cmd.StartStats()
	}
	cmd.SigInfoHandler()
	for try := 1; try <= ci.Retries; try++ {
		cmdErr = cb()
		cmdErr = fs.CountError(cmdErr)
		lastErr := accounting.GlobalStats().GetLastError()
		if cmdErr == nil {
			cmdErr = lastErr
		}
		if !Retry || !accounting.GlobalStats().Errored() {
			if try > 1 {
				fs.Errorf(nil, "Attempt %d/%d succeeded", try, ci.Retries)
			}
			break
		}
		if accounting.GlobalStats().HadFatalError() {
			fs.Errorf(nil, "Fatal error received - not attempting retries")
			break
		}
		if accounting.GlobalStats().Errored() && !accounting.GlobalStats().HadRetryError() {
			fs.Errorf(nil, "Can't retry any of the errors - not attempting retries")
			break
		}
		if retryAfter := accounting.GlobalStats().RetryAfter(); !retryAfter.IsZero() {
			d := time.Until(retryAfter)
			if d > 0 {
				fs.Logf(nil, "Received retry after error - sleeping until %s (%v)", retryAfter.Format(time.RFC3339Nano), d)
				time.Sleep(d)
			}
		}
		if lastErr != nil {
			fs.Errorf(nil, "Attempt %d/%d failed with %d errors and: %v", try, ci.Retries, accounting.GlobalStats().GetErrors(), lastErr)
		} else {
			fs.Errorf(nil, "Attempt %d/%d failed with %d errors", try, ci.Retries, accounting.GlobalStats().GetErrors())
		}
		if try < ci.Retries {
			accounting.GlobalStats().ResetErrors()
		}
		if ci.RetriesInterval > 0 {
			time.Sleep(ci.RetriesInterval)
		}
	}
	stopStats()
	if showStats && (accounting.GlobalStats().Errored() || statsInterval > 0) {
		accounting.GlobalStats().Log()
	}
	fs.Debugf(nil, "%d go routines active\n", runtime.NumGoroutine())

	if ci.Progress && ci.ProgressTerminalTitle {
		// Clear terminal title
		terminal.WriteTerminalTitle("")
	}

	// dump all running go-routines
	if ci.Dump&fs.DumpGoRoutines != 0 {
		err := pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		if err != nil {
			fs.Errorf(nil, "Failed to dump goroutines: %v", err)
		}
	}

	// dump open files
	if ci.Dump&fs.DumpOpenFiles != 0 {
		c := exec.Command("lsof", "-p", strconv.Itoa(os.Getpid()))
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		err := c.Run()
		if err != nil {
			fs.Errorf(nil, "Failed to list open files: %v", err)
		}
	}

	// clear cache and shutdown backends
	cache.Clear()
	if lastErr := accounting.GlobalStats().GetLastError(); cmdErr == nil {
		cmdErr = lastErr
	}

	// Log the final error message and exit
	if cmdErr != nil {
		nerrs := accounting.GlobalStats().GetErrors()
		if nerrs <= 1 {
			fs.Logf(nil, "Failed: %v", cmdErr)
		} else {
			fs.Logf(nil, "Failed with %d errors: last error was: %v", nerrs, cmdErr)
		}
	}
	resolveExitCode(cmdErr)
}

var (
	errorCommandNotFound    = errors.New("command not found")
	errorNotEnoughArguments = errors.New("not enough arguments")
	errorTooManyArguments   = errors.New("too many arguments")
)

func resolveExitCode(err error) {
	ctx := context.Background()
	ci := fs.GetConfig(ctx)
	atexit.Run()
	if err == nil {
		if ci.ErrorOnNoTransfer {
			if accounting.GlobalStats().GetTransfers() == 0 {
				os.Exit(exitcode.NoFilesTransferred)
			}
		}
		os.Exit(exitcode.Success)
	}

	switch {
	case errors.Is(err, fs.ErrorDirNotFound):
		os.Exit(exitcode.DirNotFound)
	case errors.Is(err, fs.ErrorObjectNotFound):
		os.Exit(exitcode.FileNotFound)
	case errors.Is(err, accounting.ErrorMaxTransferLimitReached):
		os.Exit(exitcode.TransferExceeded)
	case errors.Is(err, fssync.ErrorMaxDurationReached):
		os.Exit(exitcode.DurationExceeded)
	case fserrors.ShouldRetry(err):
		os.Exit(exitcode.RetryError)
	case fserrors.IsNoRetryError(err), fserrors.IsNoLowLevelRetryError(err):
		os.Exit(exitcode.NoRetryError)
	case fserrors.IsFatalError(err):
		os.Exit(exitcode.FatalError)
	case errors.Is(err, errorCommandNotFound), errors.Is(err, errorNotEnoughArguments), errors.Is(err, errorTooManyArguments):
		os.Exit(exitcode.UsageError)
	default:
		os.Exit(exitcode.UncategorizedError)
	}
}

var initial sync.Once
var config Config

func Initial() {
	initial.Do(func() {
		// Load the .env file
		err := godotenv.Load(".env")
		if err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}

		// Parse environment variables into the struct
		if err := env.Parse(&config); err != nil {
			log.Fatalf("Failed to parse env variables: %v", err)
		}
	})
}

func Sync() {
	// Initialize the config
	ctx := initConfig(context.Background())
	fsConfig := fs.GetConfig(ctx)

	var err error

	srcFs, err := fs.NewFs(ctx, config.FromFs)
	be.HandleError(err, "Failed to initialize source filesystem", true, nil, nil)

	dstFs, err := fs.NewFs(ctx, config.ToFs)
	be.HandleError(err, "Failed to initialize destination filesystem", true, nil, nil)

	// Set up filter rules
	if config.FilterFile != "" {
		filterConfig := filter.GetConfig(ctx)
		err = filterConfig.AddFile(config.FilterFile)
		be.HandleError(err, "Add filter file error", false, nil, func() {
			ctx = filter.ReplaceConfig(ctx, filterConfig)
		})
	}

	// Set bandwidth limit
	if config.Bandwidth != "" {
		be.HandleError(fsConfig.BwLimit.Set(config.Bandwidth), "Failed to set bandwidth limit", false, nil, nil)
	}

	// Set parallel transfers
	fsConfig.Transfers = config.Parallel

	fsConfig.Progress = true
	fsConfig.ProgressTerminalTitle = true
	fsConfig.Reload(ctx)

	Run(ctx, true, false, func() error {
		// Perform the sync
		err := be.HandleError(fssync.Sync(ctx, dstFs, srcFs, false), "Sync failed", true, func(err error) {
			log.Fatalf("Sync failed: %v", err)
		}, nil)
		if err != nil {
			return err
		}

		log.Println("Sync completed successfully.")
		return nil
	})
}
