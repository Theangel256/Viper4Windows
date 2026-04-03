package main

// ── Comunicación de Bajo Nivel con el APO ────────────────────────────────────

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")

	// Cargamos los procedimientos específicos que necesitas
	procOpenFileMapping = modKernel32.NewProc("OpenFileMappingW") // 'W' es para Wide (UTF16/Unicode)
	procMapViewOfFile   = modKernel32.NewProc("MapViewOfFile")
	procUnmapViewOfFile = modKernel32.NewProc("UnmapViewOfFile")
)

const (
	// Nombre del objeto de memoria compartida que usa ViPER4Windows
	// Según el RE, suele ser algo como "Global\ViPER4Windows_SharedMem"
	SharedMemName = "Global\\ViPER4Windows_SharedMemory"

	// El tamaño aproximado de la estructura de parámetros
	SharedMemSize = 4096
)

// Esta estructura debe replicar EXACTAMENTE el orden de ViPER4Windows
// según el archivo /ViPERDSP/DSP.h del repo de RE.
type VIPER_DSP_PARAMS struct {
	Enabled float32
	PreVol  float32
	PostVol float32

	// El bloque del EQ de 18 bandas
	// Frecuencias: 65, 93, 131, 185, 262, 370, 523, 740,
	// 1k, 1.5k, 2.1k, 3k, 4.2k, 6k, 8.4k, 11.8k, 16.7k, 20k
	EqEnabled float32
	EqBands   [18]float32

	// XBass (Siguiente bloque en memoria)
	XBassEnabled float32
	XBassSize    float32 // Speaker Size (2, 5, etc)
	XBassLevel   float32 // Bass Level dB

	// XClarity
	XClarityEnabled float32
	XClarityLevel   float32
	XClarityMode    float32 // 0: Natural, 1: OZone+, 2: X-HiFi
}

func (a *App) writeToSharedMemory(params VIPER_DSP_PARAMS) error {
	namePtr, _ := windows.UTF16PtrFromString("Global\\ViPER4Windows_SharedMem")

	// 1. OpenFileMappingW
	// Argumentos: dwDesiredAccess, bInheritHandle, lpName
	handle, _, err := procOpenFileMapping.Call(
		uintptr(windows.FILE_MAP_WRITE),
		0,
		uintptr(unsafe.Pointer(namePtr)),
	)

	// En las llamadas a DLL, el "err" siempre devuelve algo (incluso si es 0/success)
	// Por eso verificamos si el handle es 0 (NULL)
	if handle == 0 {
		return fmt.Errorf("no se pudo abrir el mapa de memoria (APO inactivo): %v", err)
	}
	defer windows.CloseHandle(windows.Handle(handle))

	// 2. MapViewOfFile
	// Argumentos: hFileMappingObject, dwDesiredAccess, dwFileOffsetHigh, dwFileOffsetLow, dwNumberOfBytesToMap
	addr, _, err := procMapViewOfFile.Call(
		handle,
		uintptr(windows.FILE_MAP_WRITE),
		0,
		0,
		0,
	)
	if addr == 0 {
		return fmt.Errorf("error al mapear vista: %v", err)
	}
	// IMPORTANTE: El desmapeo debe ocurrir DESPUÉS de usar los datos
	defer procUnmapViewOfFile.Call(addr)

	// 3. Inyectar datos (Forma recomendada para evitar avisos del linter)
	// Al envolverlo así, Go trata la operación como una copia de memoria pura
	destination := (*VIPER_DSP_PARAMS)(unsafe.Pointer(addr))
	*destination = params

	return nil
}

func (a *App) ApplyToDriver() {
	// 1. Convertimos nuestro estado de Go (DSPState) al formato de la DLL
	params := VIPER_DSP_PARAMS{
		Enabled:   boolToFloat(a.state.Master.Power),
		PreVol:    float32(a.state.Master.PreVol),
		PostVol:   float32(a.state.Master.PostVol),
		EqEnabled: 1.0, // Siempre on si hay cambios
	}

	// Copiamos las 18 bandas de nuestro array a la estructura de C
	for i := 0; i < 18; i++ {
		params.EqBands[i] = float32(a.state.Equalizer[i])
	}

	// 2. Escribir en la Memoria Compartida (Shared Memory)
	// El nombre suele ser "Global\ViPER4Windows_SharedMem" o similar en el RE
	a.writeToSharedMemory(params)
}

func boolToFloat(b bool) float32 {
	if b {
		return 1.0
	}
	return 0.0
}
