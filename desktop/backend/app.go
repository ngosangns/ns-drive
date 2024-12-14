package backend

import (
	"context"
	"desktop/backend/rclone"
	"desktop/backend/utils"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
	oc  chan []byte
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	if err := utils.CdToNormalizeWorkingDir(a.ctx); err != nil {
		utils.LogErrorAndExit(err)
	}

	// Send events to the frontend
	a.oc = make(chan []byte)
	go func() {
		for data := range a.oc {
			runtime.EventsEmit(a.ctx, "tofe", string(data))
		}
	}()

	rclone.Initial()
}
