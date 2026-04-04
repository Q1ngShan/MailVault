package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"mailvault/internal/db"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	database, err := db.Init()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	mailService := NewMailService(database)

	app := application.New(application.Options{
		Name:        "MailVault",
		Description: "Outlook mail account manager",
		Services: []application.Service{
			application.NewService(mailService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "MailVault",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		URL:       "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
