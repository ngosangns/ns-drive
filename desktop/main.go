package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist/browser
var assets embed.FS

func main() {

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "ngosangns-drive",
		Width:  768,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 128, G: 128, B: 128, A: 1}, // Gray color
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		EnumBind: []interface{}{
			[]struct {
				Value  Platform
				TSName string
			}{
				{Windows, Windows.String()},
				{Darwin, Darwin.String()},
				{Linux, Linux.String()},
			},
			[]struct {
				Value  Environment
				TSName string
			}{
				{Development, Development.String()},
				{Production, Production.String()},
			},
			[]struct {
				Value  Command
				TSName string
			}{
				{CommandStoped, CommandStoped.String()},
				{CommandOutput, CommandOutput.String()},
				{CommandStarted, CommandStarted.String()},
				{Error, Error.String()},
			},
			[]CommandDTO{
				NewCommandStoppedDTO(0),
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
