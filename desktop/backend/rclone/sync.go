package rclone

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"sync"

	"log"

	beConfig "desktop/backend/config"
	"desktop/backend/utils"

	env "github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/accounting"
	"github.com/rclone/rclone/fs/filter"
	"github.com/rclone/rclone/fs/rc"
	"github.com/rclone/rclone/fs/rc/rcserver"
	fssync "github.com/rclone/rclone/fs/sync"
	"github.com/rclone/rclone/lib/atexit"
	"github.com/rclone/rclone/lib/terminal"

	"github.com/rclone/rclone/fs/config/configfile"
	fslog "github.com/rclone/rclone/fs/log"

	// import fs drivers
	_ "github.com/rclone/rclone/backend/drive"
	_ "github.com/rclone/rclone/backend/local"
)

func InitConfig(ctx context.Context) context.Context {
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

var initial sync.Once
var config beConfig.Config

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

func Sync(ctx context.Context, outLog chan string) {
	// Initialize the config
	fsConfig := fs.GetConfig(ctx)

	var err error

	srcFs, err := fs.NewFs(ctx, config.FromFs)
	utils.HandleError(err, "Failed to initialize source filesystem", nil, nil)

	dstFs, err := fs.NewFs(ctx, config.ToFs)
	utils.HandleError(err, "Failed to initialize destination filesystem", nil, nil)

	// Set up filter rules
	if config.FilterFile != "" {
		filterConfig := filter.GetConfig(ctx)
		err = filterConfig.AddFile(config.FilterFile)
		utils.HandleError(err, "Add filter file error", nil, func() {
			ctx = filter.ReplaceConfig(ctx, filterConfig)
		})
	}

	// Set bandwidth limit
	if config.Bandwidth != "" {
		utils.HandleError(fsConfig.BwLimit.Set(config.Bandwidth), "Failed to set bandwidth limit", nil, nil)
	}

	// Set parallel transfers
	fsConfig.Transfers = config.Parallel

	fsConfig.Progress = true
	fsConfig.Reload(ctx)

	// ctx, cancel := context.WithCancel(ctx)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go (func() {
		go (func() {
			for {
				logEntry, ok := <-outLog
				if !ok { // channel is closed
					break
				}
				fmt.Println(logEntry)
			}
			wg.Done()
		})()

		runErr := utils.RunRcloneWithRetryAndStats(ctx, true, false, outLog, func() error {
			return utils.HandleError(fssync.Sync(ctx, dstFs, srcFs, false), "Sync failed", nil, nil)
		})

		if runErr != nil {
			outLog <- fmt.Sprintf("RunRcloneWithRetryAndStats failed: %v", runErr)
		} else {
			outLog <- "Sync completed successfully"
		}

		wg.Done()
	})()

	wg.Wait()
}
