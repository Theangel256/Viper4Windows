package protocol

import (
	"encoding/binary"
	"math"
	"strings"

	"viper4windows/internal/domain/models"
	"viper4windows/internal/utils"
)

// DispatchCommand mirrors the exact signature of ViPER::DispatchCommand
// (ViPERDSP/viper/ViPER.h) and of the viper_dispatch bridge export
// (ViPERDSP/viper/viper_bridge.cpp):
//
//	void DispatchCommand(int param, int val1, int val2, int val3,
//	                      int val4, uint32_t arrSize, signed char *arr);
//
// Every field/scaling choice below (centi vs boolCenti vs plain int,
// which effect maps to which PARAM_* code) is copied verbatim from
// the currently-shipped buildDLLCommands() in dspmanager.go — this is
// NOT a redesign of the wire semantics, only a namespaced, testable,
// duplication-free version of the same logic so it can also target
// PARAM_SPK_* (which the current app never dispatches at all).
type DispatchCommand struct {
	Param   int
	Val1    int32
	Val2    int32
	Val3    int32
	Val4    int32
	ArrSize uint32
	Payload []byte // nil when ArrSize == 0
}

// ── scaling helpers, copied from dspmanager.go verbatim ─────────────

func centi(v float64) int32 { return int32(math.Round(v * 100.0)) }

func dbLinearCenti(db float64) int32 {
	linear := math.Pow(10.0, db/20.0)
	return int32(math.Round(linear * 100.0))
}

func boolInt(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// boolCenti is the (odd, but real) convention buildDLLCommands() uses
// for FETCompressor's enable/auto/no-clip flags: 0 or 100, not 0/1.
func boolCenti(b bool) int32 {
	if b {
		return 100
	}
	return 0
}

// clampF/clampI used to duplicate utils.Clamp/ClampInt's bodies
// verbatim under different names. They're kept as thin aliases (rather
// than rewriting every call site below to say utils.Clamp) so this
// package still reads as "protocol logic", with exactly one real
// implementation living in internal/utils.
func clampF(v, lo, hi float64) float64 { return utils.Clamp(v, lo, hi) }
func clampI(v, lo, hi int) int         { return utils.ClampInt(v, lo, hi) }

func cmd(param int, val1 int32) DispatchCommand { return DispatchCommand{Param: param, Val1: val1} }

func float32Payload(values []float32) []byte {
	out := make([]byte, 4*len(values))
	for i, v := range values {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(v))
	}
	return out
}

// BuildDispatchCommands renders a full models.DSPState into the ordered
// list of DispatchCommand calls needed to bring the DSP instance for
// the given Namespace (Headphone or Speaker) up to date.
//
// The shipped app only ever calls the Headphone-shaped version of this
// (hardcoded PARAM_HP_* everywhere) regardless of whether the active
// endpoint is actually headphones or speakers — Namespace lets a
// caller apply the same state to both blocks, or route it based on
// the endpoint's real form factor once that's wired up.
func BuildDispatchCommands(state models.DSPState, ns Namespace) []DispatchCommand {
	cmds := make([]DispatchCommand, 0, 96)

	// ── Master output ────────────────────────────────────────────
	var postVolume int32
	if state.Master.Power {
		postVolume = dbLinearCenti(state.Master.PostVol)
	}
	cmds = append(cmds, cmd(code(paramOutputVolume, ns), postVolume))
	cmds = append(cmds, cmd(code(paramChannelPan, ns), centi(clampF(state.Output.Pan, -1.0, 1.0))))
	cmds = append(cmds, cmd(code(paramLimiter, ns), centi(clampF(state.Output.Limiter, 0.0, 1.0))))

	// ── Equalizer (dispatched one band at a time: val1=index, val2=centi value) ──
	cmds = append(cmds, cmd(code(paramEQEnable, ns), boolInt(state.EqOn)))
	cmds = append(cmds, cmd(code(paramEQBandCount, ns), int32(models.EqBands)))
	for i := 0; i < len(state.Equalizer) && i < models.EqBands; i++ {
		cmds = append(cmds, DispatchCommand{
			Param: code(paramEQBandLevel, ns),
			Val1:  int32(i),
			Val2:  centi(state.Equalizer[i]),
		})
	}

	// ── Bass (XBass) ─────────────────────────────────────────────
	cmds = append(cmds, cmd(code(paramBassEnable, ns), boolInt(state.XBass.On)))
	if state.XBass.On {
		cmds = append(cmds,
			cmd(code(paramBassMode, ns), bassModeCode(state.XBass.Mode)),
			cmd(code(paramBassFrequency, ns), speakerSizeHz(state.XBass.SpeakerSize)),
			cmd(code(paramBassGain, ns), centi(state.XBass.Level)),
			cmd(code(paramBassAntiPop, ns), 1),
		)
	}

	// ── Bass Mono (XBassMono) — previously built but never sent for SPK ──
	cmds = append(cmds, cmd(code(paramBassMonoEnable, ns), boolInt(state.XBassMono.On)))
	if state.XBassMono.On {
		cmds = append(cmds,
			cmd(code(paramBassMonoMode, ns), bassModeCode(state.XBassMono.Mode)),
			cmd(code(paramBassMonoFrequency, ns), speakerSizeHz(state.XBassMono.SpeakerSize)),
			cmd(code(paramBassMonoGain, ns), centi(state.XBassMono.Level)),
			cmd(code(paramBassMonoAntiPop, ns), 1),
		)
	}

	// ── Clarity (XClarity) ───────────────────────────────────────
	cmds = append(cmds, cmd(code(paramClarityEnable, ns), boolInt(state.XClarity.On)))
	if state.XClarity.On {
		cmds = append(cmds,
			cmd(code(paramClarityMode, ns), clarityModeCode(state.XClarity.Mode)),
			cmd(code(paramClarityGain, ns), centi(state.XClarity.Level)),
		)
	}

	// "Headphone surround" block: driven by Surround3D per the shipped
	// mapping (paramHeadphoneSurrEnable/Str, NOT FieldSurround). The
	// name in ViPERParams.h is misleading — it exists once per
	// namespace, so it is not literally headphone-only.
	cmds = append(cmds, cmd(code(paramHeadphoneSurrEnable, ns), boolInt(state.Surround3D.On)))
	if state.Surround3D.On {
		cmds = append(cmds, cmd(code(paramHeadphoneSurrStr, ns), int32(clampI(state.Surround3D.SpaceSize*10, 0, 100))))
	}

	// ── Reverb ───────────────────────────────────────────────────
	cmds = append(cmds, cmd(code(paramReverbEnable, ns), boolInt(state.Reverb.On)))
	if state.Reverb.On {
		wet := int32(clampI(int(math.Round(state.Reverb.WetMix)), 0, 100))
		cmds = append(cmds,
			cmd(code(paramReverbRoomSize, ns), int32(clampI(int(math.Round(state.Reverb.RoomSize/10.0)), 0, 100))),
			cmd(code(paramReverbDampening, ns), int32(clampI(int(math.Round(state.Reverb.Damping)), 0, 100))),
			cmd(code(paramReverbWet, ns), wet),
			cmd(code(paramReverbDry, ns), 100-wet),
			cmd(code(paramReverbRoomWidth, ns), 100),
		)
	}

	// ── Convolver ────────────────────────────────────────────────
	convolverEnabled := state.Convolver.On && strings.TrimSpace(state.Convolver.KernelPath) != ""
	cmds = append(cmds, cmd(code(paramConvolverEnable, ns), boolInt(convolverEnabled)))
	cmds = append(cmds, cmd(code(paramConvolverCrossChan, ns), centi(clampF(state.Convolver.CrossChannel, 0.0, 1.0))))
	if convolverEnabled {
		payload := []byte(strings.TrimSpace(state.Convolver.KernelPath))
		cmds = append(cmds, DispatchCommand{
			Param:   code(paramConvolverSetKernel, ns),
			ArrSize: uint32(len(payload)),
			Payload: payload,
		})
	}

	// ── DDC ──────────────────────────────────────────────────────
	ddcPayload, ddcCount := ddcCoefficientPayload(state.DDC)
	ddcEnabled := state.DDC.On && ddcCount > 0
	cmds = append(cmds, cmd(code(paramDDCEnable, ns), boolInt(ddcEnabled)))
	if ddcEnabled {
		cmds = append(cmds, DispatchCommand{
			Param:   code(paramDDCCoefficients, ns),
			ArrSize: uint32(ddcCount),
			Payload: ddcPayload,
		})
	}

	// ── AGC ──────────────────────────────────────────────────────
	cmds = append(cmds, cmd(code(paramAGCEnable, ns), boolInt(state.AGC.On)))
	if state.AGC.On {
		cmds = append(cmds,
			cmd(code(paramAGCRatio, ns), centi(state.AGC.Ratio)),
			cmd(code(paramAGCVolume, ns), centi(state.AGC.Volume)),
			cmd(code(paramAGCMaxScaler, ns), centi(state.AGC.MaxScaler)),
		)
	}

	// ── Dynamic System ───────────────────────────────────────────
	cmds = append(cmds, cmd(code(paramDynSysEnable, ns), boolInt(state.DynamicSystem.On)))
	if state.DynamicSystem.On {
		cmds = append(cmds,
			DispatchCommand{Param: code(paramDynSysXCoeffs, ns), Val1: int32(state.DynamicSystem.XCoeffsLow), Val2: int32(state.DynamicSystem.XCoeffsHigh)},
			DispatchCommand{Param: code(paramDynSysYCoeffs, ns), Val1: int32(state.DynamicSystem.YCoeffsLow), Val2: int32(state.DynamicSystem.YCoeffsHigh)},
			DispatchCommand{Param: code(paramDynSysSideGain, ns), Val1: centi(state.DynamicSystem.SideGainX), Val2: centi(state.DynamicSystem.SideGainY)},
			cmd(code(paramDynSysStrength, ns), centi(state.DynamicSystem.Strength)),
		)
	}

	// ── Spectrum extension ───────────────────────────────────────
	cmds = append(cmds, cmd(code(paramSpectrumEnable, ns), boolInt(state.SpectrumExtension.On)))
	if state.SpectrumExtension.On {
		cmds = append(cmds,
			cmd(code(paramSpectrumBark, ns), int32(state.SpectrumExtension.ReferenceFrequency)),
			cmd(code(paramSpectrumBarkRecon, ns), centi(state.SpectrumExtension.Exciter)),
		)
	}

	// ── Field surround ───────────────────────────────────────────
	cmds = append(cmds, cmd(code(paramFieldEnable, ns), boolInt(state.FieldSurround.On)))
	if state.FieldSurround.On {
		cmds = append(cmds,
			cmd(code(paramFieldWidening, ns), centi(state.FieldSurround.Widening)),
			cmd(code(paramFieldMidImage, ns), centi(state.FieldSurround.MidImage)),
			cmd(code(paramFieldDepth, ns), int32(state.FieldSurround.Depth)),
		)
	}

	// ── Diff surround ────────────────────────────────────────────
	cmds = append(cmds, cmd(code(paramDiffEnable, ns), boolInt(state.DiffSurround.On)))
	if state.DiffSurround.On {
		cmds = append(cmds, cmd(code(paramDiffDelay, ns), centi(state.DiffSurround.Delay)))
	}

	// ── Cure ─────────────────────────────────────────────────────
	cmds = append(cmds, cmd(code(paramCureEnable, ns), boolInt(state.Cure.On)))
	if state.Cure.On {
		cmds = append(cmds, cmd(code(paramCureStrength, ns), int32(clampI(state.Cure.StrengthPreset, 0, 2))))
	}

	// ── Tube simulator ───────────────────────────────────────────
	cmds = append(cmds, cmd(code(paramTubeEnable, ns), boolInt(state.TubeSimulator.On)))

	// ── AnalogX ──────────────────────────────────────────────────
	cmds = append(cmds, cmd(code(paramAnalogXEnable, ns), boolInt(state.AnalogX.On)))
	if state.AnalogX.On {
		cmds = append(cmds, cmd(code(paramAnalogXMode, ns), int32(state.AnalogX.Mode)))
	}

	// ── FET Compressor (note: enable/auto/no-clip flags use boolCenti,
	// i.e. 0/100 — that's the real upstream convention, not a typo) ──
	cmds = append(cmds, cmd(code(paramFETEnable, ns), boolCenti(state.FETCompressor.On)))
	if state.FETCompressor.On {
		cmds = append(cmds,
			cmd(code(paramFETThreshold, ns), centi(state.FETCompressor.Threshold)),
			cmd(code(paramFETRatio, ns), centi(state.FETCompressor.Ratio)),
			cmd(code(paramFETKnee, ns), centi(state.FETCompressor.Knee)),
			cmd(code(paramFETAutoKnee, ns), boolCenti(state.FETCompressor.AutoKnee)),
			cmd(code(paramFETGain, ns), centi(state.FETCompressor.Gain)),
			cmd(code(paramFETAutoGain, ns), boolCenti(state.FETCompressor.AutoGain)),
			cmd(code(paramFETAttack, ns), centi(state.FETCompressor.Attack)),
			cmd(code(paramFETAutoAttack, ns), boolCenti(state.FETCompressor.AutoAttack)),
			cmd(code(paramFETRelease, ns), centi(state.FETCompressor.Release)),
			cmd(code(paramFETAutoRelease, ns), boolCenti(state.FETCompressor.AutoRelease)),
			cmd(code(paramFETKneeMulti, ns), centi(state.FETCompressor.KneeMulti)),
			cmd(code(paramFETMaxAttack, ns), centi(state.FETCompressor.MaxAttack)),
			cmd(code(paramFETMaxRelease, ns), centi(state.FETCompressor.MaxRelease)),
			cmd(code(paramFETCrest, ns), centi(state.FETCompressor.Crest)),
			cmd(code(paramFETAdapt, ns), centi(state.FETCompressor.Adapt)),
			cmd(code(paramFETNoClip, ns), boolCenti(state.FETCompressor.NoClip)),
		)
	}

	// ── Speaker correction: SPK-only in the real header. The shipped
	// app currently sends this fixed SPK code unconditionally, even
	// while building "headphone" commands. Here it is only emitted
	// for ns == Speaker, which is the behavior you actually want once
	// HP/SPK are dispatched separately.
	if ns == Speaker {
		cmds = append(cmds, cmd(paramSpeakerCorrectionEnable, boolInt(state.SpeakerCorrection.On)))
	}

	return cmds
}

func bassModeCode(mode models.XBassMode) int32 {
	if mode == models.XBassPure {
		return 1
	}
	return 0
}

func clarityModeCode(mode models.XClarityMode) int32 {
	switch mode {
	case models.XClarityOZone:
		return 1
	case models.XClarityXHiFi:
		return 2
	default:
		return 0
	}
}

func speakerSizeHz(size int) int32 {
	idx := clampI(size, 0, len(models.SpeakerSizeToHz)-1)
	return int32(models.SpeakerSizeToHz[idx])
}

// ddcCoefficientPayload mirrors serializeDDCPayload() in dspmanager.go:
// both coefficient sets are first trimmed down to a multiple of 5
// (5 biquad coefficients per band), then truncated to the shorter of
// the two, then interleaved as [44100 band..., 48000 band...].
func ddcCoefficientPayload(ddc models.DDCState) ([]byte, int) {
	count44100 := (len(ddc.Coeffs44100) / 5) * 5
	count48000 := (len(ddc.Coeffs48000) / 5) * 5
	if count44100 == 0 || count48000 == 0 {
		return nil, 0
	}
	count := count44100
	if count48000 < count {
		count = count48000
	}

	values := make([]float32, 0, count*2)
	for i := 0; i < count; i++ {
		values = append(values, float32(ddc.Coeffs44100[i]))
	}
	for i := 0; i < count; i++ {
		values = append(values, float32(ddc.Coeffs48000[i]))
	}
	return float32Payload(values), count
}
