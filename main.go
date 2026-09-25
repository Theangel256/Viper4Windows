package main

import (
	"embed"
	"os"
	"path/filepath"

	"viper4windows/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	windowsOpts "github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	presetsDir := resolvePresetsDir()
	dllPath := resolveDllPath()
	viperApp := app.NewApp(presetsDir, dllPath)

	err := wails.Run(&options.App{
		Title:         "Viper4Windows — Audio DSP",
		Width:         1200,
		Height:        680,
		MinWidth:      900,
		MinHeight:     580,
		DisableResize: false,
		Frameless:     false,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "67730e9c-0e89-47ef-8360-57ecd90aa1c2",
			OnSecondInstanceLaunch: viperApp.OnSecondInstanceLaunch,
		},
		BackgroundColour: &options.RGBA{R: 244, G: 244, B: 245, A: 255}, // zinc-100
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: viperApp.Startup,
		// The old main.go never wired OnShutdown at all, even though
		// App.Shutdown() exists and does real cleanup (closes shared
		// memory) — Wails was just never told to call it.
		OnShutdown: viperApp.Shutdown,
		Bind: []interface{}{
			viperApp,
		},
		Windows: &windowsOpts.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisablePinchZoom:     true,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}

// resolvePresetsDir mirrors the pre-refactor app's convention
// (filepath.Join(exeDir, "presets")) rather than inventing a new one.
func resolvePresetsDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "presets"
	}
	return filepath.Join(filepath.Dir(exePath), "presets")
}

// resolveDllPath is the same "next to the exe" convention the
// original pre-refactor app.go used for ViPERDSP.dll.
func resolveDllPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return "ViPERDSP.dll"
	}
	return filepath.Join(filepath.Dir(exePath), "ViPERDSP.dll")
}
