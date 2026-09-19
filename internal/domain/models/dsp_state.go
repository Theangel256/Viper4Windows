package models

// ═══════════════════════════════════════════════════════════════════════════
// DSP State Models
// ═══════════════════════════════════════════════════════════════════════════
// Clean, domain-focused models following the reference architecture.
// Each model represents a distinct DSP module with clear responsibilities.

// MasterState controls global power and volume settings
type MasterState struct {
	Power   bool    `json:"power"`
	PreVol  float64 `json:"preVol"`  // Pre-amplification volume (-12 to 0 dB)
	PostVol float64 `json:"postVol"` // Post-amplification volume (0 to 12 dB)
}

// OutputState manages stereo output and limiting
type OutputState struct {
	Pan     float64 `json:"pan"`     // Stereo pan (-1.0 left to 1.0 right)
	Limiter float64 `json:"limiter"` // Output limiter (0.0 to 1.0)
}

// XBassState represents the bass enhancement module
type XBassState struct {
	On          bool      `json:"on"`
	SpeakerSize int       `json:"speakerSize"` // 0-10 (maps to Hz via lookup table)
	Level       float64   `json:"level"`       // Bass boost level in dB
	Mode        XBassMode `json:"mode"`        // "Natural Bass" or "Pure Bass"
}

// XBassMonoState handles mono bass enhancement
type XBassMonoState struct {
	On          bool      `json:"on"`
	SpeakerSize int       `json:"speakerSize"`
	Level       float64   `json:"level"`
	Mode        XBassMode `json:"mode"`
}

// XClarityState manages clarity enhancement
type XClarityState struct {
	On    bool         `json:"on"`
	Level float64      `json:"level"` // Clarity boost level in dB
	Mode  XClarityMode `json:"mode"`  // "Natural", "OZone+", or "X-HiFi"
}

// Surround3DState controls 3D spatial effects
type Surround3DState struct {
	On        bool   `json:"on"`
	SpaceSize int    `json:"spaceSize"` // 0-10
	RoomSize  string `json:"roomSize"`  // "Smallest Room", etc.
	ImageSize int    `json:"imageSize"` // 0-10
}

// ReverbParams defines reverb effect parameters
type ReverbParams struct {
	On        bool    `json:"on"`
	RoomSize  float64 `json:"roomSize"`  // Room dimensions
	Damping   float64 `json:"damping"`   // High-frequency damping
	Density   float64 `json:"density"`   // Echo density
	Bandwidth float64 `json:"bandwidth"` // Frequency bandwidth
	Decay     float64 `json:"decay"`     // Reverb decay time
	PreDelay  float64 `json:"preDelay"`  // Initial delay before reverb
	EarlyMix  float64 `json:"earlyMix"`  // Early reflections mix
	WetMix    float64 `json:"wetMix"`    // Wet signal mix
}

// ReverbPanelState is a simplified reverb control interface
type ReverbPanelState struct {
	On       bool    `json:"on"`
	RoomSize string  `json:"roomSize"` // Preset room size
	Size     float64 `json:"size"`     // Custom size parameter
	WetMix   float64 `json:"wetMix"`   // Wet signal amount
}

// ConvolverState manages convolution reverb
type ConvolverState struct {
	On           bool    `json:"on"`
	KernelPath   string  `json:"kernelPath"`   // Path to IR file
	CrossChannel float64 `json:"crossChannel"` // Cross-channel mixing
}

// DDCState represents Digital Dynamic Compression
type DDCState struct {
	On          bool      `json:"on"`
	Coeffs44100 []float64 `json:"coeffs44100"` // Coefficients for 44.1kHz
	Coeffs48000 []float64 `json:"coeffs48000"` // Coefficients for 48kHz
}

// AGCState manages Automatic Gain Control
type AGCState struct {
	On        bool    `json:"on"`
	Ratio     float64 `json:"ratio"`     // Compression ratio
	Volume    float64 `json:"volume"`    // Target volume
	MaxScaler float64 `json:"maxScaler"` // Maximum gain scaling
}

// DynamicSystemState controls dynamic processing
type DynamicSystemState struct {
	On          bool    `json:"on"`
	XCoeffsLow  int     `json:"xCoeffsLow"`
	XCoeffsHigh int     `json:"xCoeffsHigh"`
	YCoeffsLow  int     `json:"yCoeffsLow"`
	YCoeffsHigh int     `json:"yCoeffsHigh"`
	SideGainX   float64 `json:"sideGainX"`
	SideGainY   float64 `json:"sideGainY"`
	Strength    float64 `json:"strength"`
}

// SpectrumExtensionState extends high-frequency content
type SpectrumExtensionState struct {
	On                 bool    `json:"on"`
	ReferenceFrequency int     `json:"referenceFrequency"` // Reference freq in Hz
	Exciter            float64 `json:"exciter"`            // Exciter strength
}

// FieldSurroundState creates stereo field widening
type FieldSurroundState struct {
	On       bool    `json:"on"`
	Widening float64 `json:"widening"` // Stereo width (0.0 to 1.0)
	MidImage float64 `json:"midImage"` // Mid/side image balance
	Depth    int     `json:"depth"`    // Effect depth (0-100)
}

// DiffSurroundState applies differential surround
type DiffSurroundState struct {
	On    bool    `json:"on"`
	Delay float64 `json:"delay"` // Delay time for effect
}

// CureState manages cure/restoration processing
type CureState struct {
	On             bool `json:"on"`
	StrengthPreset int  `json:"strengthPreset"` // 0, 1, or 2
}

// TubeSimulatorState simulates tube amp warmth
type TubeSimulatorState struct {
	On bool `json:"on"`
}

// AnalogXState provides analog coloration
type AnalogXState struct {
	On   bool `json:"on"`
	Mode int  `json:"mode"` // 0-8, different analog models
}

// FETCompressorState models a FET-style compressor
type FETCompressorState struct {
	On          bool    `json:"on"`
	Threshold   float64 `json:"threshold"`   // Compression threshold in dB
	Ratio       float64 `json:"ratio"`       // Compression ratio
	Knee        float64 `json:"knee"`        // Knee width
	AutoKnee    bool    `json:"autoKnee"`    // Auto-adjust knee
	Gain        float64 `json:"gain"`        // Makeup gain in dB
	AutoGain    bool    `json:"autoGain"`    // Auto makeup gain
	Attack      float64 `json:"attack"`      // Attack time in ms
	AutoAttack  bool    `json:"autoAttack"`  // Auto attack time
	Release     float64 `json:"release"`     // Release time in ms
	AutoRelease bool    `json:"autoRelease"` // Auto release time
	KneeMulti   float64 `json:"kneeMulti"`   // Knee multiplier
	MaxAttack   float64 `json:"maxAttack"`   // Max attack time
	MaxRelease  float64 `json:"maxRelease"`  // Max release time
	Crest       float64 `json:"crest"`       // Crest factor
	Adapt       float64 `json:"adapt"`       // Adaptive parameter
	NoClip      bool    `json:"noClip"`      // Prevent output clipping
}

// SpeakerCorrectionState enables speaker correction
type SpeakerCorrectionState struct {
	On bool `json:"on"`
}

// ═══════════════════════════════════════════════════════════════════════════
// Complete DSP State
// ═══════════════════════════════════════════════════════════════════════════

// DSPState is the complete serializable state of the DSP engine.
// This is the single source of truth for all DSP parameters.
type DSPState struct {
	// Core Controls
	Master MasterState `json:"master"`
	Output OutputState `json:"output"`
	Mode   DSPMode     `json:"mode"` // "music", "movie", or "freestyle"

	// Equalizer
	EqOn      bool      `json:"eqOn"`
	Equalizer []float64 `json:"equalizer"` // 18-band EQ values

	// Enhancement Effects
	XBass     XBassState     `json:"xBass"`
	XBassMono XBassMonoState `json:"xBassMono"`
	XClarity  XClarityState  `json:"xClarity"`

	// Spatial Effects
	Surround3D    Surround3DState    `json:"surround3D"`
	Reverb        ReverbParams       `json:"reverb"`
	ReverbPanel   ReverbPanelState   `json:"reverbPanel"`
	Convolver     ConvolverState     `json:"convolver"`
	FieldSurround FieldSurroundState `json:"fieldSurround"`
	DiffSurround  DiffSurroundState  `json:"diffSurround"`

	// Dynamic Processing
	DDC           DDCState           `json:"ddc"`
	AGC           AGCState           `json:"agc"`
	DynamicSystem DynamicSystemState `json:"dynamicSystem"`
	FETCompressor FETCompressorState `json:"fetCompressor"`

	// Tone Shaping
	SpectrumExtension SpectrumExtensionState `json:"spectrumExtension"`
	Cure              CureState              `json:"cure"`
	TubeSimulator     TubeSimulatorState     `json:"tubeSimulator"`
	AnalogX           AnalogXState           `json:"analogX"`
	SpeakerCorrection SpeakerCorrectionState `json:"speakerCorrection"`
}

// NewDefaultState returns factory-reset DSP parameters
func NewDefaultState() DSPState {
	return DSPState{
		Equalizer: make([]float64, EqBands),
		EqOn:      true,
		Mode:      ModeFreestyle,

		Master: MasterState{
			Power:   true,
			PreVol:  0.0,
			PostVol: 12.0,
		},

		Output: OutputState{
			Pan:     0.0,
			Limiter: 1.0,
		},

		XBass: XBassState{
			On:          true,
			SpeakerSize: 5,
			Level:       0.0,
			Mode:        XBassNatural,
		},

		XBassMono: XBassMonoState{
			On:          false,
			SpeakerSize: 5,
			Level:       0.0,
			Mode:        XBassNatural,
		},

		XClarity: XClarityState{
			On:    true,
			Level: 0.0,
			Mode:  XClarityXHiFi,
		},

		Surround3D: Surround3DState{
			On:        true,
			SpaceSize: 5,
			RoomSize:  "Smallest Room",
			ImageSize: 2,
		},

		Reverb: ReverbParams{
			On:        true,
			RoomSize:  500,
			Damping:   1.03,
			Density:   12.2,
			Bandwidth: 44,
			Decay:     13,
			PreDelay:  0,
			EarlyMix:  91,
			WetMix:    48,
		},

		ReverbPanel: ReverbPanelState{
			On:       true,
			RoomSize: "Medium Hall",
			Size:     0.5,
			WetMix:   0.3,
		},

		Convolver: ConvolverState{
			On:           false,
			KernelPath:   "",
			CrossChannel: 0.0,
		},

		DDC: DDCState{
			On:          false,
			Coeffs44100: []float64{},
			Coeffs48000: []float64{},
		},

		AGC: AGCState{
			On:        false,
			Ratio:     3.0,
			Volume:    10.0,
			MaxScaler: 5.0,
		},

		DynamicSystem: DynamicSystemState{
			On:          false,
			XCoeffsLow:  0,
			XCoeffsHigh: 0,
			YCoeffsLow:  0,
			YCoeffsHigh: 0,
			SideGainX:   0.0,
			SideGainY:   0.0,
			Strength:    0.0,
		},

		SpectrumExtension: SpectrumExtensionState{
			On:                 false,
			ReferenceFrequency: 8000,
			Exciter:            0.0,
		},

		FieldSurround: FieldSurroundState{
			On:       false,
			Widening: 0.0,
			MidImage: 0.0,
			Depth:    0,
		},

		DiffSurround: DiffSurroundState{
			On:    false,
			Delay: 0.0,
		},

		Cure: CureState{
			On:             false,
			StrengthPreset: 0,
		},

		TubeSimulator: TubeSimulatorState{
			On: false,
		},

		AnalogX: AnalogXState{
			On:   false,
			Mode: 0,
		},

		FETCompressor: FETCompressorState{
			On:          false,
			Threshold:   -20.0,
			Ratio:       4.0,
			Knee:        6.0,
			AutoKnee:    false,
			Gain:        0.0,
			AutoGain:    true,
			Attack:      10.0,
			AutoAttack:  false,
			Release:     100.0,
			AutoRelease: false,
			KneeMulti:   1.0,
			MaxAttack:   50.0,
			MaxRelease:  500.0,
			Crest:       12.0,
			Adapt:       0.5,
			NoClip:      true,
		},

		SpeakerCorrection: SpeakerCorrectionState{
			On: false,
		},
	}
}
