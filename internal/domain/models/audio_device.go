package models

// ═══════════════════════════════════════════════════════════════════════════
// Audio Device Models
// ═══════════════════════════════════════════════════════════════════════════

// AudioDevice represents a Windows audio endpoint
type AudioDevice struct {
	ID            string `json:"id"`            // Unique device identifier
	Name          string `json:"name"`          // Friendly display name
	DeviceType    string `json:"deviceType"`    // "render" or "capture"
	IsDefault     bool   `json:"isDefault"`     // Whether this is the default device
	IsEnabled     bool   `json:"isEnabled"`     // Device enabled status
	IconPath      string `json:"iconPath"`      // System icon path
	Description   string `json:"description"`   // Full device description
	State         int    `json:"state"`         // Device state (1=active, etc.)
	HasAPO        bool   `json:"hasAPO"`        // Whether the ViPER APO is currently attached
	LegacyResidue bool   `json:"legacyResidue"` // PreMix points at a foreign/old CLSID, not ours
	LegacyCLSID   string `json:"legacyClsid"`   // That foreign CLSID, if any (empty otherwise)
}

// DeviceRole represents the type of audio endpoint
type DeviceRole string

const (
	DeviceRoleRender  DeviceRole = "render"  // Playback devices
	DeviceRoleCapture DeviceRole = "capture" // Recording devices
)

// APOStatus represents the current status of the Audio Processing Object
type APOStatus struct {
	IsInstalled  bool   `json:"isInstalled"`  // APO is registered in Windows
	IsAttached   bool   `json:"isAttached"`   // APO is attached to device
	Version      string `json:"version"`      // APO version string
	Architecture string `json:"architecture"` // "x64" or "x86"
	DllPath      string `json:"dllPath"`      // Path to ViPERDSP.dll
}

// AudioEngineStatus represents the state of the Windows audio engine
type AudioEngineStatus struct {
	IsRunning   bool  `json:"isRunning"`   // Audio service is active
	SampleRate  int   `json:"sampleRate"`  // Current sample rate (Hz)
	ProcessTime int64 `json:"processTime"` // Processing time in ms
	BufferSize  int   `json:"bufferSize"`  // Audio buffer size
}
