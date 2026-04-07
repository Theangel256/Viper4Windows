package main

import (
	"embed"
	"fmt"
	"strings"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	windowsOpts "github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// DriverManager handles the registration and lifecycle of the APO.
// Corresponds to the 'Installer' logic in ViPERDSP/Alpha.
// IsElevated checks if the process has administrative privileges
func (dm *DriverManager) IsElevated() bool {
	// Method 1: Token elevation (primary)
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err == nil {
		defer token.Close()

		var elevation TOKEN_ELEVATION
		var returnedLen uint32
		err = windows.GetTokenInformation(
			token,
			windows.TokenElevation,
			(*byte)(unsafe.Pointer(&elevation)),
			uint32(unsafe.Sizeof(elevation)),
			&returnedLen,
		)
		if err == nil {
			return elevation.TokenIsElevated != 0
		}
	}

	// Method 2: Fallback — try opening a protected registry key
	// If we can write to HKLM, we're elevated
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		registry.SET_VALUE,
	)
	if err == nil {
		k.Close()
		return true
	}

	return false
}

// RequireAdmin validates administrative privileges before operations
func (dm *DriverManager) RequireAdmin() error {
	if !dm.IsElevated() {
		return fmt.Errorf("ACCESS_DENIED: Administrator privileges required.\nRight-click the application and select 'Run as Administrator'")
	}
	return nil
}

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
		Title:         "Viper4Windows — Audio DSP",
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
