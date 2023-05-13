package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	//"github.com/eugenepentland/VectorPB/functions"
	//"github.com/nlpodyssey/cybertron/pkg/tasks"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/cmd"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed frontend/build/*
var assets embed.FS

func main() {
	// Create an instance of the app structure
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()
	ui := NewApp(db)

	// Create application with options
	// Create application with options
	err = wails.Run(&options.App{
		Title:             "Local VectorDB",
		Width:             1020,
		Height:            768,
		MinWidth:          1024,
		MinHeight:         768,
		MaxWidth:          1280,
		MaxHeight:         800,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		Assets:            assets,
		Menu:              nil,
		Logger:            nil,
		LogLevel:          logger.DEBUG,
		OnStartup:         ui.startup,
		OnDomReady:        ui.domReady,
		OnBeforeClose:     ui.beforeClose,
		OnShutdown:        ui.shutdown,
		WindowStartState:  options.Normal,
		Bind: []interface{}{
			ui,
		},
		// Windows platform specific options
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			// DisableFramelessWindowDecorations: false,
			WebviewUserDataPath: "",
		},
		// Mac platform specific options
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: false,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			Appearance:           mac.NSAppearanceNameAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "sveltekit",
				Message: "",
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	app := pocketbase.New()
	cmd.NewServeCommand(app, true)
	app.Bootstrap()
	os.Args = append(os.Args, "serve")
	err = app.Start()
	if err != nil {
		fmt.Println(err)
	}
}
