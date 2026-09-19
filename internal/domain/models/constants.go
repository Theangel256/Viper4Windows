package models

// ═══════════════════════════════════════════════════════════════════════════
// DSP Parameter Constraints
// ═══════════════════════════════════════════════════════════════════════════

const (
	// Volume Constraints
	MinPreVol  = -12.0
	MaxPreVol  = 0.0
	MinPostVol = 0.0
	MaxPostVol = 12.0

	// Equalizer Constraints
	MinEqBand = -12.0
	MaxEqBand = 12.0
	EqBands   = 18

	// XBass Constraints
	MinXBassLevel = -12.0
	MaxXBassLevel = 12.0

	// XClarity Constraints
	MinXClarityLevel = -12.0
	MaxXClarityLevel = 12.0

	// Spatial Constraints
	MinSpeakerSize = 0
	MaxSpeakerSize = 10
	MinSpaceSize   = 0
	MaxSpaceSize   = 10
	MinImageSize   = 0
	MaxImageSize   = 10

	// Output Constraints
	MinPan     = -1.0
	MaxPan     = 1.0
	MinLimiter = 0.0
	MaxLimiter = 1.0
)

// Note: the PARAM_HP_*/PARAM_SPK_* dispatch codes used to be duplicated
// here (as ParamHP*, exported) AND in internal/dsp/protocol/params.go
// (as paramXxx, private, generated per-namespace via code()). This
// file's copy was the only one dsp_manager_service.go ever read, and
// that file has since been rewritten to call
// protocol.BuildDispatchCommands instead of building commands by hand
// — so the 73 constants that used to live in this block are gone.
// internal/dsp/protocol/params.go is now the single source of truth
// for wire-level dispatch codes; this file only holds domain-level
// constraints and the shared-memory buffer layout below.

// ═══════════════════════════════════════════════════════════════════════════
// Shared Memory Layout Constants
// ═══════════════════════════════════════════════════════════════════════════

const (
	SharedMemName   = "Global\\ViPER4Windows_SharedMemory"
	SharedEventName = "Global\\ViPER4Windows_Event"
	SharedMemBytes  = 1024
	ParamCount      = SharedMemBytes / 4

	// Buffer Indices
	IdxEnabled                  = 0
	IdxPreVol                   = 1
	IdxPostVol                  = 2
	IdxEQEnabled                = 4
	IdxEQBands                  = 5
	IdxXBassEnabled             = 23
	IdxXBassMode                = 24
	IdxXBassSpkSize             = 25
	IdxXBassGain                = 26
	IdxXClarEnabled             = 27
	IdxXClarMode                = 28
	IdxXClarGain                = 29
	IdxSurrEnabled              = 30
	IdxSurrSize                 = 31
	IdxRevEnabled               = 32
	IdxRevRoom                  = 33
	IdxRevDamp                  = 34
	IdxRevMix                   = 35
	IdxConvolverEnabled         = 36
	IdxCureEnabled              = 37
	IdxCureStrength             = 38
	IdxAnalogXEnabled           = 39
	IdxAnalogXMode              = 40
	IdxSpeakerCorrectionEnabled = 41
	IdxLimiterEnabled           = 44
	IdxLimiterValue             = 45

	// Additional indices for effects not in original fillParamBuf
	// These fill gaps in the buffer layout
	IdxDDCEnabled        = 46
	IdxAGCEnabled        = 48
	IdxDynamicSysEnabled = 50
	IdxSpectrumEnabled   = 52
	IdxFieldEnabled      = 54
	IdxDiffEnabled       = 56
	IdxFETEnabled        = 58
	IdxTubeEnabled       = 60
)

// ═══════════════════════════════════════════════════════════════════════════
// Lookup Tables
// ═══════════════════════════════════════════════════════════════════════════

var SpeakerSizeToHz = [11]int{
	20, 30, 40, 55, 70, 90, 115, 150, 200, 280, 380,
}

// ═══════════════════════════════════════════════════════════════════════════
// Mode Enums
// ═══════════════════════════════════════════════════════════════════════════

type DSPMode string

const (
	ModeMusic     DSPMode = "music"
	ModeMovie     DSPMode = "movie"
	ModeFreestyle DSPMode = "freestyle"
)

type XBassMode string

const (
	XBassNatural XBassMode = "Natural Bass"
	XBassPure    XBassMode = "Pure Bass"
)

type XClarityMode string

const (
	XClarityNatural XClarityMode = "Natural"
	XClarityOZone   XClarityMode = "OZone+"
	XClarityXHiFi   XClarityMode = "X-HiFi"
)
