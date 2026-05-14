package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"fcs-autoreport/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend
var assets embed.FS

// windowTitle — заголовок окна; дата сборки из метаданных Go (vcs.time), если сборка из репозитория.
func windowTitle() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.time" && s.Value != "" {
				if t, err := time.Parse(time.RFC3339, s.Value); err == nil {
					return fmt.Sprintf("FCS AutoReport (сборка %s)", t.Format("02.01.2006"))
				}
			}
		}
	}
	return "FCS AutoReport"
}

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
		Title:  windowTitle(),
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
