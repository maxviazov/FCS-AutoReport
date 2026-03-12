package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"fcs-autoreport/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	dataDir := "."
	if dir, err := os.UserConfigDir(); err == nil {
		dataDir = filepath.Join(dir, "FCS-AutoReport")
	}

	database, store, err := app.Bootstrap(dataDir)
	if err != nil {
		log.Fatal("bootstrap:", err)
	}
	defer database.Close()

	svc := app.NewReportService(database, store)
	if err := svc.LoadDictionariesToMemory(); err != nil {
		log.Println("предупреждение: загрузка справочников:", err)
	}

	wailsApp := app.NewWailsApp(svc)

	err = wails.Run(&options.App{
		Title:  "FCS AutoReport (сборка 12.03.2026)",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: wailsApp.Startup,
		Bind:      []interface{}{wailsApp},
	})
	if err != nil {
		log.Fatal(err)
	}
}
