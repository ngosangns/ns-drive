package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"

	"github.com/joho/godotenv"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Environment int

const (
	Development Environment = iota
	Production
)

func (e Environment) String() string {
	return [...]string{"development", "production"}[e]
}

type Platform int

const (
	Windows Platform = iota
	Darwin
	Linux
)

func (p Platform) String() string {
	return [...]string{"windows", "darwin", "linux"}[p]
}

type Command string

const (
	StopCommand   Command = "stop_command"
	CommandStoped Command = "command_stoped"
	ErrorPrefix   Command = "error__"
)

func (c Command) String() string {
	return string(c)
}

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

func (a *App) GetWorkingDir() (string, error) {
	if a.GetCurrentEnvName() == Development.String() {
		if wd, err := os.Getwd(); err == nil {
			return filepath.Clean(wd + "/../"), nil
		} else {
			return "", err
		}
	}

	if exePath, err := os.Executable(); err == nil {
		if a.GetPlatform() == Windows.String() {
			return filepath.Dir(exePath), nil
		}
		return filepath.Clean(filepath.Dir(exePath) + "/../../../"), nil
	} else {
		return "", err
	}
}

func (a *App) LoadEnv() error {
	if a.GetPlatform() == Darwin.String() {
		if path, err := a.GetPATH(); err == nil {
			if err := os.Setenv("PATH", path); err != nil {
				a.LogError(err)
				return err
			}
		} else {
			a.LogError(err)
			return err
		}
	}

	if wd, err := a.GetWorkingDir(); err == nil {
		return godotenv.Load(wd + "/.env")
	} else {
		return err
	}
}

func (a *App) CdToNormalizeWorkingDir() error {
	if wd, err := a.GetWorkingDir(); err == nil {
		return os.Chdir(wd)
	} else {
		return err
	}
}

// runRcloneSync runs the rclone sync command with the provided arguments
func (a *App) RunRcloneSync(args ...string) error {
	cmdArr := append([]string{"task"}, args...)

	cmd := exec.Command(cmdArr[0], cmdArr[1:]...)
	cmd.Stdout = Om
	cmd.Stderr = Om

	go func() {
		for data := range Im {
			if string(data) == StopCommand.String() {
				if err := cmd.Process.Kill(); err != nil {
					Om <- []byte(ErrorPrefix.String() + err.Error())
					a.LogError(err)
				}

				Om <- []byte(CommandStoped)
				break
			} else if string(data) == CommandStoped.String() {
				break
			}
		}
	}()

	err := cmd.Run()
	if err != nil {
		Om <- []byte(ErrorPrefix.String() + err.Error())
		a.LogError(err)
	}
	Om <- []byte(CommandStoped)
	Im <- []byte(CommandStoped)
	return err
}

// logError logs the error to a file on the desktop, including stack trace and error line number
func (a *App) LogError(inErr error) {
	if wd, err := a.GetWorkingDir(); err == nil {
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
