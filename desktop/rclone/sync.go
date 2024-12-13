package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"log"

	"github.com/joho/godotenv"
	_ "github.com/rclone/rclone/backend/drive"
	_ "github.com/rclone/rclone/backend/local"
	"github.com/rclone/rclone/cmd"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/accounting"
	"github.com/rclone/rclone/fs/cache"
	"github.com/rclone/rclone/fs/filter"
	"github.com/rclone/rclone/fs/fserrors"
	"github.com/rclone/rclone/fs/operations"
	"github.com/rclone/rclone/fs/rc"
	"github.com/rclone/rclone/fs/rc/rcserver"
	fssync "github.com/rclone/rclone/fs/sync"
	"github.com/rclone/rclone/lib/atexit"
	"github.com/rclone/rclone/lib/exitcode"
	"github.com/rclone/rclone/lib/terminal"

	"github.com/rclone/rclone/fs/config/configfile"
	fslog "github.com/rclone/rclone/fs/log"
)

const (
	// interval between progress prints
	defaultProgressInterval = 500 * time.Millisecond
	// time format for logging
	logTimeFormat = "2006/01/02 15:04:05"
)

var (
	statsInterval = time.Minute * 1
)

// startProgress starts the progress bar printing
//
// It returns a func which should be called to stop the stats.
func startProgress() func() {
	stopStats := make(chan struct{})
	oldLogOutput := fs.LogOutput
	oldSyncPrint := operations.SyncPrintf

	if !fslog.Redirected() {
		// Intercept the log calls if not logging to file or syslog
		fs.LogOutput = func(level fs.LogLevel, text string) {
			printProgress(fmt.Sprintf("%s %-6s: %s", time.Now().Format(logTimeFormat), level, text))

		}
	}

	// Intercept output from functions such as HashLister to stdout
	operations.SyncPrintf = func(format string, a ...interface{}) {
		printProgress(fmt.Sprintf(format, a...))
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		progressInterval := defaultProgressInterval
		if cmd.ShowStats() && statsInterval > 0 {
			progressInterval = statsInterval
		}
		ticker := time.NewTicker(progressInterval)
		for {
			select {
			case <-ticker.C:
				printProgress("")
			case <-stopStats:
				ticker.Stop()
				printProgress("")
				fs.LogOutput = oldLogOutput
				operations.SyncPrintf = oldSyncPrint
				fmt.Println("")
				return
			}
		}
	}()
	return func() {
		close(stopStats)
		wg.Wait()
	}
}

// state for the progress printing
var (
	nlines = 0 // number of lines in the previous stats block
)

// printProgress prints the progress with an optional log
func printProgress(logMessage string) {
	operations.StdoutMutex.Lock()
	defer operations.StdoutMutex.Unlock()

	var buf bytes.Buffer
	w, _ := terminal.GetSize()
	stats := strings.TrimSpace(accounting.GlobalStats().String())
	logMessage = strings.TrimSpace(logMessage)

	out := func(s string) {
		buf.WriteString(s)
	}

	if logMessage != "" {
		out("\n")
		out(terminal.MoveUp)
	}
	// Move to the start of the block we wrote erasing all the previous lines
	for i := 0; i < nlines-1; i++ {
		out(terminal.EraseLine)
		out(terminal.MoveUp)
	}
	out(terminal.EraseLine)
	out(terminal.MoveToStartOfLine)
	if logMessage != "" {
		out(terminal.EraseLine)
		out(logMessage + "\n")
	}
	fixedLines := strings.Split(stats, "\n")
	nlines = len(fixedLines)
	for i, line := range fixedLines {
		if len(line) > w {
			line = line[:w]
		}
		out(line)
		if i != nlines-1 {
			out("\n")
		}
	}
	terminal.Write(buf.Bytes())
}

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

	// // Setup CPU profiling if desired
	// cpuProfile := "Debugging"
	// if cpuProfile != "" {
	// 	fs.Infof(nil, "Creating CPU profile %q\n", cpuProfile)
	// 	f, err := os.Create(cpuProfile)
	// 	if err != nil {
	// 		err = fs.CountError(err)
	// 		fs.Fatal(nil, fmt.Sprint(err))
	// 	}
	// 	err = pprof.StartCPUProfile(f)
	// 	if err != nil {
	// 		err = fs.CountError(err)
	// 		fs.Fatal(nil, fmt.Sprint(err))
	// 	}
	// 	atexit.Register(func() {
	// 		pprof.StopCPUProfile()
	// 		err := f.Close()
	// 		if err != nil {
	// 			err = fs.CountError(err)
	// 			fs.Fatal(nil, fmt.Sprint(err))
	// 		}
	// 	})
	// }

	// // Setup memory profiling if desired
	// memProfile := "Debugging"
	// if memProfile != "" {
	// 	atexit.Register(func() {
	// 		fs.Infof(nil, "Saving Memory profile %q\n", memProfile)
	// 		f, err := os.Create(memProfile)
	// 		if err != nil {
	// 			err = fs.CountError(err)
	// 			fs.Fatal(nil, fmt.Sprint(err))
	// 		}
	// 		err = pprof.WriteHeapProfile(f)
	// 		if err != nil {
	// 			err = fs.CountError(err)
	// 			fs.Fatal(nil, fmt.Sprint(err))
	// 		}
	// 		err = f.Close()
	// 		if err != nil {
	// 			err = fs.CountError(err)
	// 			fs.Fatal(nil, fmt.Sprint(err))
	// 		}
	// 	})
	// }

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
		stopStats = startProgress()
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

func main() {
	// Change the working directory to the root of the project
	os.Chdir("../../")

	// Load the .env file
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	clientFromPath := os.Getenv("CLIENT_FROM_PATH")
	clientToPath := os.Getenv("CLIENT_PATH")
	clientFilterPath := os.Getenv("CLIENT_FILTER_PATH")
	clientLimitBandwidth := os.Getenv("CLIENT_LIMIT_BANDWIDTH")
	clientParallel := os.Getenv("CLIENT_PARALLEL")

	// Initialize the config
	ctx := initConfig(context.Background())
	fsConfig := fs.GetConfig(ctx)

	// Set up filter rules
	if clientFilterPath != "" {
		filterConfig := filter.GetConfig(ctx)
		filterConfig.AddFile(clientFilterPath)
		ctx = filter.ReplaceConfig(ctx, filterConfig)
	}

	// Configure the FS (FileSystem) for source and destination
	srcFs, err := fs.NewFs(ctx, clientFromPath)
	if err != nil {
		log.Fatalf("Failed to initialize source filesystem: %v", err)
	}

	dstFs, err := fs.NewFs(ctx, clientToPath)
	if err != nil {
		log.Fatalf("Failed to initialize destination filesystem: %v", err)
	}

	// Set bandwidth limit
	if clientLimitBandwidth != "" {
		err = fsConfig.BwLimit.Set(clientLimitBandwidth)
		if err != nil {
			log.Fatalf("Failed to set bandwidth limit: %v", err)
		}
	}

	// Set parallel transfers
	if clientParallel != "" {
		parallel, err := strconv.Atoi(clientParallel)
		if err != nil {
			log.Fatalf("Failed to parse CLIENT_PARALLEL: %v", err)
		}
		fsConfig.Transfers = parallel
	}

	fsConfig.Progress = true
	fsConfig.ProgressTerminalTitle = true
	fsConfig.Reload(ctx)

	Run(ctx, true, false, func() error {
		// Perform the sync
		err := fssync.Sync(ctx, dstFs, srcFs, false)
		if err != nil {
			log.Fatalf("Sync failed: %v", err)
			return err
		}

		log.Println("Sync completed successfully.")
		return nil
	})
}
