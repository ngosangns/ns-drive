package main

import (
	be "desktop/backend"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist/browser
var assets embed.FS

func main() {

	// Create an instance of the app structure
	app := be.NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "ngosangns-drive",
		Width:  768,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 128, G: 128, B: 128, A: 1}, // Gray color
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
		EnumBind: []interface{}{
			[]struct {
				Value  be.Platform
				TSName string
			}{
				{be.Windows, be.Windows.String()},
				{be.Darwin, be.Darwin.String()},
				{be.Linux, be.Linux.String()},
			},
			[]struct {
				Value  be.Environment
				TSName string
			}{
				{be.Development, be.Development.String()},
				{be.Production, be.Production.String()},
			},
			[]struct {
				Value  be.Command
				TSName string
			}{
				{be.CommandStoped, be.CommandStoped.String()},
				{be.CommandOutput, be.CommandOutput.String()},
				{be.CommandStarted, be.CommandStarted.String()},
				{be.Error, be.Error.String()},
			},
			[]be.CommandDTO{
				be.NewCommandStoppedDTO(0),
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
