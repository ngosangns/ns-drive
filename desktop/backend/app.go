package backend

import (
	"context"
	"desktop/backend/models"
	"desktop/backend/utils"
	"os"

	fsConfig "github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	oc         chan []byte
	ConfigInfo models.ConfigInfo
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

	config, err := utils.LoadConfigFromEnv()
	if err != nil {
		utils.LogErrorAndExit(err)
	}

	// Load working directory
	wd, err := os.Getwd()
	if err != nil {
		utils.LogErrorAndExit(err)
	}
	a.ConfigInfo.WorkingDir = wd

	// Load profiles
	err = a.ConfigInfo.ReadFromFile(*config)
	if err != nil {
		utils.LogErrorAndExit(err)
	}

	// Setup event channel for sending messages to the frontend
	a.oc = make(chan []byte)
	go func() {
		for data := range a.oc {
			runtime.EventsEmit(a.ctx, "tofe", string(data))
		}
	}()

	// Load Rclone config
	fsConfig.SetConfigPath(config.RcloneFilePath)
	configfile.Install()
}
