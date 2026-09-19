package protocol

import (
	"encoding/binary"
	"fmt"
	"math"

	"viper4windows/internal/domain/models"
)

// ParamBuffer is a typed view over the shared-memory layout that is
// ACTUALLY in this repo today (models.SharedMemBytes == 1024, i.e.
// 256 float32 slots, see CreateFileMapping(..., sharedMemBytes, ...)
// in the pre-refactor dspmanager.go and SharedMemoryService in the
// refactored tree). It is a drop-in replacement for the ad-hoc
// []float32 + Idx* indexing in DSPParameterService.EncodeState: same
// bytes on the wire, same indices, just named field access plus a
// round-trip Decode so state can be verified/tested without a live
// APO on the other end.
//
// This is deliberately NOT the 1144-byte "typed ViPERParams struct"
// some upstream forks are moving to — the ViPERDSP submodule vendored
// in this repo right now doesn't define one (see
// ViPERDSP/include/ViPERParams.h, which only has PARAM_HP_*/PARAM_SPK_*
// dispatch codes). When you update the submodule to a commit that does
// ship that struct, this file is what gets replaced — and its
// replacement must be generated from the real header via
// tools/cpp/dump_viper_params_layout.cpp, not hand-typed.
type ParamBuffer [models.ParamCount]float32

// EncodeDSPState renders a models.DSPState into the flat parameter
// buffer using the exact same indices as fillParamBuf() in the
// pre-refactor dspmanager.go / DSPParameterService.EncodeState in the
// refactored tree.
func EncodeDSPState(state models.DSPState) ParamBuffer {
	var buf ParamBuffer

	if state.Master.Power {
		buf[models.IdxEnabled] = 1
	}
	buf[models.IdxPreVol] = float32(state.Master.PreVol)
	buf[models.IdxPostVol] = float32(state.Master.PostVol)

	if state.EqOn {
		buf[models.IdxEQEnabled] = 1
	}
	for i := 0; i < len(state.Equalizer) && i < models.EqBands; i++ {
		buf[models.IdxEQBands+i] = float32(state.Equalizer[i])
	}

	if state.XBass.On {
		buf[models.IdxXBassEnabled] = 1
		buf[models.IdxXBassMode] = bassModeFloat(state.XBass.Mode)
		buf[models.IdxXBassSpkSize] = float32(state.XBass.SpeakerSize)
		buf[models.IdxXBassGain] = float32(state.XBass.Level)
	}

	if state.XClarity.On {
		buf[models.IdxXClarEnabled] = 1
		buf[models.IdxXClarMode] = clarityModeFloat(state.XClarity.Mode)
		buf[models.IdxXClarGain] = float32(state.XClarity.Level)
	}

	if state.Surround3D.On {
		buf[models.IdxSurrEnabled] = 1
		buf[models.IdxSurrSize] = float32(state.Surround3D.SpaceSize)
	}

	if state.Reverb.On {
		buf[models.IdxRevEnabled] = 1
		buf[models.IdxRevRoom] = float32(state.Reverb.RoomSize)
		buf[models.IdxRevDamp] = float32(state.Reverb.Damping)
		buf[models.IdxRevMix] = float32(state.Reverb.WetMix)
	}

	if state.Convolver.On {
		buf[models.IdxConvolverEnabled] = 1
	}

	if state.Cure.On {
		buf[models.IdxCureEnabled] = 1
		buf[models.IdxCureStrength] = float32(clampI(state.Cure.StrengthPreset, 0, 2))
	}

	if state.AnalogX.On {
		buf[models.IdxAnalogXEnabled] = 1
		buf[models.IdxAnalogXMode] = float32(state.AnalogX.Mode)
	}

	if state.SpeakerCorrection.On {
		buf[models.IdxSpeakerCorrectionEnabled] = 1
	}

	if state.Output.Limiter < 1.0 {
		buf[models.IdxLimiterEnabled] = 1
		buf[models.IdxLimiterValue] = float32(clampF(state.Output.Limiter, 0.0, 1.0))
	}

	if state.DDC.On && len(state.DDC.Coeffs44100) > 0 && len(state.DDC.Coeffs48000) > 0 {
		buf[models.IdxDDCEnabled] = 1
	}
	if state.AGC.On {
		buf[models.IdxAGCEnabled] = 1
	}
	if state.DynamicSystem.On {
		buf[models.IdxDynamicSysEnabled] = 1
	}
	if state.SpectrumExtension.On {
		buf[models.IdxSpectrumEnabled] = 1
	}
	if state.FieldSurround.On {
		buf[models.IdxFieldEnabled] = 1
	}
	if state.DiffSurround.On {
		buf[models.IdxDiffEnabled] = 1
	}
	if state.FETCompressor.On {
		buf[models.IdxFETEnabled] = 1
	}
	if state.TubeSimulator.On {
		buf[models.IdxTubeEnabled] = 1
	}

	return buf
}

// Bytes returns the little-endian wire representation written into
// shared memory (exactly what writeSharedMemory() copies today).
func (b ParamBuffer) Bytes() []byte {
	out := make([]byte, models.SharedMemBytes)
	for i, v := range b {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(v))
	}
	return out
}

// DecodeParamBuffer parses raw shared-memory bytes back into a typed
// ParamBuffer. Used by tests (and by anything that wants to read back
// what a peer just wrote, e.g. a debug/inspection tool).
func DecodeParamBuffer(data []byte) (ParamBuffer, error) {
	var buf ParamBuffer
	if len(data) != models.SharedMemBytes {
		return buf, fmt.Errorf("protocol: expected %d bytes, got %d", models.SharedMemBytes, len(data))
	}
	for i := range buf {
		buf[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return buf, nil
}

func bassModeFloat(mode models.XBassMode) float32 {
	if mode == models.XBassPure {
		return 1
	}
	return 0
}

func clarityModeFloat(mode models.XClarityMode) float32 {
	switch mode {
	case models.XClarityOZone:
		return 1
	case models.XClarityXHiFi:
		return 2
	default:
		return 0
	}
}
