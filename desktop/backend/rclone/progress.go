package rclone

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rclone/rclone/cmd"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/accounting"
	fslog "github.com/rclone/rclone/fs/log"
	"github.com/rclone/rclone/fs/operations"
)

var (
	// state for the progress printing
	nlines = 0 // number of lines in the previous stats block
)

const (
	statsInterval = time.Minute * 1
	// interval between progress prints
	defaultProgressInterval = 500 * time.Millisecond
	// time format for logging
	logTimeFormat = "2006/01/02 15:04:05"
)

// startProgress starts the progress bar printing
//
// It returns a func which should be called to stop the stats.
func StartProgress(outLog chan string) func() {
	stopStats := make(chan struct{})
	oldLogOutput := fs.LogOutput
	oldSyncPrint := operations.SyncPrintf

	if !fslog.Redirected() {
		// Intercept the log calls if not logging to file or syslog
		fs.LogOutput = func(level fs.LogLevel, text string) {
			outLog <- FormatToProgress(fmt.Sprintf("%s %-6s: %s", time.Now().Format(logTimeFormat), level, text))
		}
	}

	// Intercept output from functions such as HashLister to stdout
	operations.SyncPrintf = func(format string, a ...interface{}) {
		outLog <- FormatToProgress(fmt.Sprintf(format, a...))
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
				outLog <- FormatToProgress("")
			case <-stopStats:
				ticker.Stop()
				outLog <- FormatToProgress("")
				fs.LogOutput = oldLogOutput
				operations.SyncPrintf = oldSyncPrint
				return
			}
		}
	}()
	return func() {
		close(stopStats)
		close(outLog)
		wg.Wait()
	}
}

// printProgress prints the progress with an optional log
func FormatToProgress(logMessage string) string {
	operations.StdoutMutex.Lock()
	defer operations.StdoutMutex.Unlock()

	var buf bytes.Buffer
	// w, _ := terminal.GetSize()
	stats := strings.TrimSpace(accounting.GlobalStats().String())
	logMessage = strings.TrimSpace(logMessage)

	out := func(s string) {
		buf.WriteString(s)
	}

	// if logMessage != "" {
	// 	out("\n")
	// 	out(terminal.MoveUp)
	// }
	// // Move to the start of the block we wrote erasing all the previous lines
	// for i := 0; i < nlines-1; i++ {
	// 	out(terminal.EraseLine)
	// 	out(terminal.MoveUp)
	// }
	// out(terminal.EraseLine)
	// out(terminal.MoveToStartOfLine)
	// if logMessage != "" {
	// 	out(terminal.EraseLine)
	// 	out(logMessage + "\n")
	// }

	fixedLines := strings.Split(stats, "\n")
	nlines = len(fixedLines)
	for i, line := range fixedLines {
		// if len(line) > w {
		// 	line = line[:w]
		// }
		out(line)
		if i != nlines-1 {
			out("\n")
		}
	}
	// terminal.Write(buf.Bytes())

	return buf.String()
}
