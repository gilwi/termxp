package main

import (
	"context"
	"embed"

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
	// Create instances of the app structures
	app := NewApp()
	terminalService := NewTerminalService()
	historyService := NewHistoryService()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "TermXP - Premium Terminal",
		Width:     1020,
		Height:    680,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 15, B: 20, A: 0},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			terminalService.SetContext(ctx)
		},
		OnShutdown: func(ctx context.Context) {
			terminalService.CleanupAllSessions()
		},
		Bind: []interface{}{
			app,
			terminalService,
			historyService,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Mica,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHidden(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
		Linux: &linux.Options{
			WindowIsTranslucent: true,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
