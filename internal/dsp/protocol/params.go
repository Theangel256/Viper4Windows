// Package protocol is the single source of truth for talking to the
// vendored ViPERDSP engine (submodule commit currently pinned in this
// repo — see internal/dsp/protocol/README.md for how to tell which one).
//
// Every constant in this file was transcribed by hand from
// ViPERDSP/include/ViPERParams.h as it exists in THIS repo's submodule
// today. That header does not define a C++ struct — it only defines
// int #define codes for a single-parameter dispatch call:
//
//	void DispatchCommand(int param, int val1, int val2, int val3,
//	                      int val4, uint32_t arrSize, signed char *arr);
//
// If you upgrade the submodule to a newer likelikeslike/ViPERDSP commit
// that ships a typed ViPERParams struct (their v2.0.0 release notes
// mention exactly that), THIS FILE GOES AWAY and gets replaced by a
// generated codec — see tools/cpp/dump_viper_params_layout.cpp for how
// to produce the real ABI offsets from whatever header you end up
// vendoring. Nobody should hand-type struct offsets; the compiler
// that builds ViPERDSP.dll is the only thing allowed to define them.
package protocol

// Namespace selects which endpoint-class parameter block a command
// targets. The header defines two completely parallel blocks — every
// PARAM_SPK_* code sits exactly 0x200 above its PARAM_HP_* twin — with
// one documented exception (SpeakerCorrection has no headphone twin).
type Namespace int

const (
	Headphone Namespace = iota
	Speaker
)

// namespaceOffset is the verified, constant delta between the HP and
// SPK blocks in ViPERParams.h (0x10100 -> 0x10300, 0x10110 -> 0x10310,
// etc.). It is NOT valid for SpeakerCorrection, which only exists in
// the SPK block.
const namespaceOffset = 0x200

// HP effect params: 0x10100 - 0x102FF (verbatim from ViPERParams.h)
const (
	paramConvolverEnable     = 0x10100
	paramConvolverSetKernel  = 0x10101
	paramConvolverCrossChan  = 0x10105
	paramDDCEnable           = 0x10110
	paramDDCCoefficients     = 0x10111
	paramEQEnable            = 0x10120
	paramEQBandLevel         = 0x10121
	paramEQBandCount         = 0x10122
	paramReverbEnable        = 0x10130
	paramReverbRoomSize      = 0x10131
	paramReverbRoomWidth     = 0x10132
	paramReverbDampening     = 0x10133
	paramReverbWet           = 0x10134
	paramReverbDry           = 0x10135
	paramAGCEnable           = 0x10140
	paramAGCRatio            = 0x10141
	paramAGCVolume           = 0x10142
	paramAGCMaxScaler        = 0x10143
	paramDynSysEnable        = 0x10150
	paramDynSysXCoeffs       = 0x10151
	paramDynSysYCoeffs       = 0x10152
	paramDynSysSideGain      = 0x10153
	paramDynSysStrength      = 0x10154
	paramBassEnable          = 0x10160
	paramBassMode            = 0x10161
	paramBassFrequency       = 0x10162
	paramBassGain            = 0x10163
	paramBassMonoEnable      = 0x10164
	paramBassMonoMode        = 0x10165
	paramBassMonoFrequency   = 0x10166
	paramBassMonoGain        = 0x10167
	paramBassAntiPop         = 0x10168
	paramBassMonoAntiPop     = 0x10169
	paramClarityEnable       = 0x10170
	paramClarityMode         = 0x10171
	paramClarityGain         = 0x10172
	paramHeadphoneSurrEnable = 0x10180
	paramHeadphoneSurrStr    = 0x10181
	paramSpectrumEnable      = 0x10190
	paramSpectrumBark        = 0x10191
	paramSpectrumBarkRecon   = 0x10192
	paramFieldEnable         = 0x101A0
	paramFieldWidening       = 0x101A1
	paramFieldMidImage       = 0x101A2
	paramFieldDepth          = 0x101A3
	paramDiffEnable          = 0x101B0
	paramDiffDelay           = 0x101B1
	paramCureEnable          = 0x101C0
	paramCureStrength        = 0x101C1
	paramTubeEnable          = 0x101D0
	paramAnalogXEnable       = 0x101E0
	paramAnalogXMode         = 0x101E1
	paramOutputVolume        = 0x101F0
	paramChannelPan          = 0x101F1
	paramLimiter             = 0x101F2
	paramFETEnable           = 0x10200
	paramFETThreshold        = 0x10201
	paramFETRatio            = 0x10202
	paramFETKnee             = 0x10203
	paramFETAutoKnee         = 0x10204
	paramFETGain             = 0x10205
	paramFETAutoGain         = 0x10206
	paramFETAttack           = 0x10207
	paramFETAutoAttack       = 0x10208
	paramFETRelease          = 0x10209
	paramFETAutoRelease      = 0x1020A
	paramFETKneeMulti        = 0x1020B
	paramFETMaxAttack        = 0x1020C
	paramFETMaxRelease       = 0x1020D
	paramFETCrest            = 0x1020E
	paramFETAdapt            = 0x1020F
	paramFETNoClip           = 0x10210
)

// SPK-only: no headphone equivalent exists in ViPERParams.h.
const paramSpeakerCorrectionEnable = 0x10420

// code resolves a shared HP-block constant to the right namespace.
// Do NOT call this with a param that has no SPK twin — the only one
// today is SpeakerCorrection, which has its own accessor below.
func code(hpParam int, ns Namespace) int {
	switch ns {
	case Headphone:
		return hpParam

	case Speaker:
		return hpParam + namespaceOffset

	default:
		panic("protocol: invalid namespace")
	}
}
