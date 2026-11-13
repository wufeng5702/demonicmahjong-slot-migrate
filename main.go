package main

import (
	"embed"

	"github.com/wailsapp/wails/v3/pkg/application"

	"demonicmahjong-slot-migrate/internal/android"
	"demonicmahjong-slot-migrate/internal/migrate"
	"demonicmahjong-slot-migrate/internal/steam"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "demonicmahjong-slot-migrate",
		Description: "将安卓版《我在地府打麻将》的存档迁移到 Steam 版本",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	// Wire services after creating the app so that services can access
	// the application instance (Dialog, Logger, Events, etc.).
	app.RegisterService(application.NewService(android.NewService(app)))
	app.RegisterService(application.NewService(migrate.NewService(app)))
	app.RegisterService(application.NewService(steam.NewService(app)))

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "我在地府打麻将 · 存档迁移",
		Width:  1024,
		Height: 768,
	})

	_ = window

	err := app.Run()
	if err != nil {
		panic(err)
	}
}
