package backend

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetPlatform() string {
	env := runtime.Environment(a.ctx)
	return env.Platform // windows, darwin, linux
}

func (a *App) GetCurrentEnvName() string {
	if runtime.Environment(a.ctx).BuildType == "dev" {
		return Development.String()
	}
	return Production.String()
}

func (a *App) GetPATH() (string, error) {
	cmd := exec.Command("/bin/zsh", "-l", "-c", "source ~/.zshrc && echo $PATH")
	output, err := cmd.Output()
	return string(output), err
}

func (a *App) CdToNormalizeWorkingDir() error {
	var wd string
	var err error

	if a.GetCurrentEnvName() == Development.String() {
		wd, err = os.Getwd()
		if err != nil {
			return err
		}
		wd = filepath.Clean(wd + "/../")
	} else {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		wd = filepath.Dir(exePath)
		if a.GetPlatform() != Windows.String() {
			wd = filepath.Clean(wd + "/../../../")
		}
	}

	return os.Chdir(wd)
}

var (
	cmdStore      = make(map[int]*exec.Cmd)
	cmdStoreMutex sync.RWMutex
)

func AddCmd(pid int, cmd *exec.Cmd) {
	cmdStoreMutex.Lock()
	defer cmdStoreMutex.Unlock()
	cmdStore[pid] = cmd
}

func GetCmd(pid int) (*exec.Cmd, bool) {
	cmdStoreMutex.RLock()
	defer cmdStoreMutex.RUnlock()
	cmd, exists := cmdStore[pid]
	return cmd, exists
}

func RemoveCmd(pid int) {
	cmdStoreMutex.Lock()
	defer cmdStoreMutex.Unlock()
	delete(cmdStore, pid)
}

// runRcloneSync runs the rclone sync command with the provided arguments
func (a *App) RunRcloneSync(args ...string) error {
	cmdArr := append([]string{"task"}, args...)
	cmd := exec.Command(cmdArr[0], cmdArr[1:]...)

	stdout := RcloneStdout{c: Oc, pid: 0}
	cmd.Stdout = stdout
	cmd.Stderr = stdout

	e := cmd.Start()
	if e != nil {
		a.LogError(e)
		j, _ := NewCommandErrorDTO(0, e).ToJSON()
		Oc <- j
		return e
	}

	// pid := cmd.Process.Pid
	pid := a.GetRandomPid()
	AddCmd(pid, cmd)

	j, _ := NewCommandStartedDTO(pid, cmdArr[1]).ToJSON()
	Oc <- j

	// Wait for the command to finish
	go func() {
		err := cmd.Wait()
		if err != nil {
			a.LogError(err)
			j, _ := NewCommandErrorDTO(pid, err).ToJSON()
			Oc <- j
		} else {
			j, _ := NewCommandStoppedDTO(pid).ToJSON()
			Oc <- j
		}
		RemoveCmd(pid)
	}()

	return e
}

func (a *App) GetRandomPid() int {
	return os.Getpid() + int(time.Now().UnixNano()%1000)
}

// logError logs the error to a file on the desktop, including stack trace and error line number
func (a *App) LogError(inErr error) {
	if wd, err := os.Getwd(); err == nil {
		if f, fileErr := os.OpenFile(filepath.Join(wd, "desktop.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); fileErr == nil {
			defer f.Close()
			logger := log.New(f, "", log.LstdFlags)
			logger.Printf("Error: %v\nStack Trace:\n%s", inErr, debug.Stack())
		} else {
			log.Printf("Failed to open log file: %v", fileErr)
		}
	} else {
		log.Printf("Failed to get working directory: %v", err)
	}
}

func (a *App) LogErrorAndExit(err error) {
	a.LogError(err)
	log.Fatal(err)
}

func HandleError(err error, context string, fatal bool, onError func(error), onClear func()) error {
	if err != nil {
		if onError != nil {
			onError(err)
		}
		if fatal {
			log.Fatal(err)
		} else {
			log.Printf("Context: %s, Error: %s\n", context, err.Error())
		}
	} else {
		if onClear != nil {
			onClear()
		}
	}

	return err
}
