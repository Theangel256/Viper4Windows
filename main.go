package main

import (
	"embed"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func (a *App) onSecondInstanceLaunch(secondInstanceData options.SecondInstanceData) {
	// Notificamos al frontend los argumentos si es necesario
	secondInstanceArgs := secondInstanceData.Args

	// 1. Restaurar si está minimizada
	runtime.WindowUnminimise(a.ctx)

	// 2. Mostrar la ventana
	runtime.WindowShow(a.ctx)

	// 3. Forzar el foco para que la ventana existente sea la protagonista
	runtime.EventsEmit(a.ctx, "launchArgs", secondInstanceArgs)

	// Opcional: imprimir en consola para debug
	println("Segunda instancia bloqueada. Argumentos:", strings.Join(secondInstanceArgs, " "))
}

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:         "Luma — Audio DSP",
		Width:         1200,
		Height:        680,
		MinWidth:      900,
		MinHeight:     580,
		DisableResize: false,
		Frameless:     false,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "67730e9c-0e89-47ef-8360-57ecd90aa1c2", // Usa un string único
			OnSecondInstanceLaunch: app.onSecondInstanceLaunch,             // Tu función
		},
		BackgroundColour: &options.RGBA{R: 244, G: 244, B: 245, A: 255}, // zinc-100
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisablePinchZoom:     true,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
