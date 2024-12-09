package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

type Message chan []byte

// Support command output
func (m Message) Write(p []byte) (n int, err error) {
	m <- p
	return len(p), nil
}

var Im Message
var Om Message

var AllCommands = []struct {
	Value  Command
	TSName string
}{
	{StopCommand, StopCommand.String()},
	{CommandStoped, CommandStoped.String()},
	{ErrorPrefix, ErrorPrefix.String()},
}

var AllPlatforms = []struct {
	Value  Platform
	TSName string
}{
	{Windows, Windows.String()},
	{Darwin, Darwin.String()},
	{Linux, Linux.String()},
}

var AllEnvironments = []struct {
	Value  Environment
	TSName string
}{
	{Development, Development.String()},
	{Production, Production.String()},
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// Pull syncs from the client source path to the client destination path
func (a *App) Pull() error {
	return a.RunRcloneSync("pull")
}

// Push syncs from the client destination path to the client source path
func (a *App) Push() error {
	return a.RunRcloneSync("push")
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	if err := a.LoadEnv(); err != nil {
		a.LogErrorAndExit(err)
	}

	if err := a.CdToNormalizeWorkingDir(); err != nil {
		a.LogErrorAndExit(err)
	}

	Im = make(chan []byte)
	Om = make(chan []byte)

	runtime.EventsOn(a.ctx, "tobe", func(data ...interface{}) {
		Im <- []byte(fmt.Sprintf("%v", data))
	})

	go func() {
		for data := range Om {
			runtime.EventsEmit(a.ctx, "tofe", string(data))
		}
	}()
}
