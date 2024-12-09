package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
var Oc chan []byte

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	if err := a.CdToNormalizeWorkingDir(); err != nil {
		a.LogErrorAndExit(err)
	}

	if err := a.LoadEnv(); err != nil {
		a.LogErrorAndExit(err)
	}

	// Send events to the frontend
	Oc = make(chan []byte)
	go func() {
		for data := range Oc {
			runtime.EventsEmit(a.ctx, "tofe", string(data))
		}
	}()
}
