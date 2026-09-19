package ports

// DLLPort manages direct DLL loading and dispatch to ViPERDSP.dll
type DLLPort interface {
	// LoadDLLPath loads ViPERDSP.dll from the given path
	LoadDLLPath(dllPath string) error

	// Close releases DLL resources
	Close() error

	// IsLoaded returns true if DLL is loaded and ready
	IsLoaded() bool

	// Dispatch sends a parameter dispatch command
	Dispatch(param, val1, val2, val3, val4 int)

	// DispatchPayload sends a parameter with array payload
	DispatchPayload(param, val1, val2, val3, val4, arrSize int, payload []byte)

	// SetSampleRate sets the DSP sample rate
	SetSampleRate(rate uint32)

	// Reset resets all DSP effect states
	Reset()
}
