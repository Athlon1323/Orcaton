package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	err := wails.Run(&options.App{
		Title:            "Orcaton",
		Width:            1440,
		Height:           900,
		MinWidth:         1100,
		MinHeight:        680,
		BackgroundColour: &options.RGBA{R: 0x1b, G: 0x1c, B: 0x1e, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown,
		DragAndDrop:      &options.DragAndDrop{EnableFileDrop: true, DisableWebViewDrop: true},
		Bind:             []interface{}{app},
		Windows:          &windows.Options{Theme: windows.Dark},
		Mac: &mac.Options{Appearance: mac.NSAppearanceNameDarkAqua,
			About: &mac.AboutInfo{Title: "Orcaton", Message: "Multicam audio sync (FCP7 XML, DRT)"}},
		Linux: &linux.Options{ProgramName: "orcaton"},
	})
	if err != nil {
		log.Fatal(err)
	}
}
