//go:build windows

package main

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestCenti(t *testing.T) {
	if got := centi(1.234); got != 123 {
		t.Fatalf("centi(1.234) = %d, want 123", got)
	}
	if got := centi(-0.255); got != -26 {
		t.Fatalf("centi(-0.255) = %d, want -26", got)
	}
}

func TestDBLinearCenti(t *testing.T) {
	if got := dbLinearCenti(0); got != 100 {
		t.Fatalf("dbLinearCenti(0) = %d, want 100", got)
	}
	if got := dbLinearCenti(6); got != 200 {
		t.Fatalf("dbLinearCenti(6) = %d, want 200", got)
	}
}

func TestParseModes(t *testing.T) {
	if got := parseBassMode("Natural Bass"); got != 0 {
		t.Fatalf("parseBassMode Natural Bass = %d, want 0", got)
	}
	if got := parseBassMode("Pure Bass"); got != 1 {
		t.Fatalf("parseBassMode Pure Bass = %d, want 1", got)
	}
	if got := parseClarityMode("Natural"); got != 0 {
		t.Fatalf("parseClarityMode Natural = %d, want 0", got)
	}
	if got := parseClarityMode("OZone+"); got != 1 {
		t.Fatalf("parseClarityMode OZone+ = %d, want 1", got)
	}
	if got := parseClarityMode("X-HiFi"); got != 2 {
		t.Fatalf("parseClarityMode X-HiFi = %d, want 2", got)
	}
}

func TestSerializeDDCPayload(t *testing.T) {
	payload, arrSize := serializeDDCPayload(DDCState{
		On:          true,
		Coeffs44100: []float64{1, 2, 3, 4, 5, 6},
		Coeffs48000: []float64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
	})

	if arrSize != 5 {
		t.Fatalf("arrSize = %d, want 5", arrSize)
	}
	if len(payload) != 40 {
		t.Fatalf("payload len = %d, want 40", len(payload))
	}

	got0 := math.Float32frombits(binary.LittleEndian.Uint32(payload[0:4]))
	got5 := math.Float32frombits(binary.LittleEndian.Uint32(payload[20:24]))
	if got0 != 1 {
		t.Fatalf("first coeff = %v, want 1", got0)
	}
	if got5 != 11 {
		t.Fatalf("first 48k coeff = %v, want 11", got5)
	}
}

func TestFillParamBuf(t *testing.T) {
	state := defaultState()
	state.Master.Power = true
	state.Master.PreVol = -3
	state.Master.PostVol = 9
	state.EqOn = true
	state.Equalizer[0] = 1.5
	state.XBass = XBassState{On: true, SpeakerSize: 6, Level: 2.5, Mode: "Pure Bass"}
	state.XClarity = XClarityState{On: true, Level: 1.25, Mode: "OZone+"}
	state.Surround3D = Surround3DState{On: true, SpaceSize: 7, RoomSize: "Smallest Room", ImageSize: 2}
	state.Reverb = ReverbParams{On: true, RoomSize: 640, Damping: 8, WetMix: 42}
	state.Convolver = ConvolverState{On: true, KernelPath: "C:\\impulse.wav"}
	state.Cure = CureState{On: true, StrengthPreset: 2}
	state.AnalogX = AnalogXState{On: true, Mode: 3}
	state.SpeakerCorrection = SpeakerCorrectionState{On: true}
	state.Output = OutputState{Pan: 0, Limiter: 0.75}

	var buf paramBuf
	fillParamBuf(&buf, state)

	if buf[idxEnabled] != 1 {
		t.Fatalf("enabled = %v, want 1", buf[idxEnabled])
	}
	if buf[idxPreVol] != -3 {
		t.Fatalf("prevol = %v, want -3", buf[idxPreVol])
	}
	if buf[idxPostVol] != 9 {
		t.Fatalf("postvol = %v, want 9", buf[idxPostVol])
	}
	if buf[idxEQEnabled] != 1 || buf[idxEQBands] != 1.5 {
		t.Fatalf("eq buffer not populated correctly: enabled=%v band0=%v", buf[idxEQEnabled], buf[idxEQBands])
	}
	if buf[idxXBassMode] != 1 || buf[idxXBassSpkSize] != 6 || buf[idxXBassGain] != 2.5 {
		t.Fatalf("xbass buffer mismatch: mode=%v size=%v gain=%v", buf[idxXBassMode], buf[idxXBassSpkSize], buf[idxXBassGain])
	}
	if buf[idxXClarMode] != 1 || buf[idxXClarGain] != 1.25 {
		t.Fatalf("xclarity buffer mismatch: mode=%v gain=%v", buf[idxXClarMode], buf[idxXClarGain])
	}
	if buf[idxSurrEnabled] != 1 || buf[idxSurrSize] != 7 {
		t.Fatalf("surround buffer mismatch: enabled=%v size=%v", buf[idxSurrEnabled], buf[idxSurrSize])
	}
	if buf[idxRevEnabled] != 1 || buf[idxRevRoom] != 640 || buf[idxRevMix] != 42 {
		t.Fatalf("reverb buffer mismatch: enabled=%v room=%v wet=%v", buf[idxRevEnabled], buf[idxRevRoom], buf[idxRevMix])
	}
	if buf[idxConvolverEnabled] != 1 {
		t.Fatalf("convolver flag = %v, want 1", buf[idxConvolverEnabled])
	}
	if buf[idxCureEnabled] != 1 || buf[idxCureStrength] != 2 {
		t.Fatalf("cure buffer mismatch: enabled=%v strength=%v", buf[idxCureEnabled], buf[idxCureStrength])
	}
	if buf[idxAnalogXEnabled] != 1 || buf[idxAnalogXMode] != 3 {
		t.Fatalf("analogx buffer mismatch: enabled=%v mode=%v", buf[idxAnalogXEnabled], buf[idxAnalogXMode])
	}
	if buf[idxSpeakerCorrectionEnabled] != 1 {
		t.Fatalf("speaker correction flag = %v, want 1", buf[idxSpeakerCorrectionEnabled])
	}
	if buf[idxLimiterEnabled] != 1 || buf[idxLimiterValue] != 0.75 {
		t.Fatalf("limiter buffer mismatch: enabled=%v value=%v", buf[idxLimiterEnabled], buf[idxLimiterValue])
	}
}

func TestBuildDLLCommands(t *testing.T) {
	state := defaultState()
	state.Master.Power = true
	state.Master.PostVol = 6
	state.Output = OutputState{Pan: -0.25, Limiter: 0.8}
	state.Equalizer[0] = 1.5
	state.XBass = XBassState{On: true, SpeakerSize: 4, Level: 3.5, Mode: "Pure Bass"}
	state.XClarity = XClarityState{On: true, Level: 2, Mode: "X-HiFi"}
	state.Surround3D = Surround3DState{On: true, SpaceSize: 5}
	state.Reverb = ReverbParams{On: true, RoomSize: 500, Damping: 10, WetMix: 25}
	state.Convolver = ConvolverState{On: true, KernelPath: "C:\\irs\\room.wav", CrossChannel: 0.3}
	state.DDC = DDCState{
		On:          true,
		Coeffs44100: []float64{1, 2, 3, 4, 5},
		Coeffs48000: []float64{6, 7, 8, 9, 10},
	}
	state.Cure = CureState{On: true, StrengthPreset: 2}
	state.AnalogX = AnalogXState{On: true, Mode: 4}
	state.SpeakerCorrection = SpeakerCorrectionState{On: true}

	cmds := buildDLLCommands(state)

	assertCommand := func(param int, want1 int, want2 ...int) {
		t.Helper()
		for _, cmd := range cmds {
			if cmd.param != param {
				continue
			}
			if cmd.val1 != want1 {
				t.Fatalf("param 0x%x val1 = %d, want %d", param, cmd.val1, want1)
			}
			if len(want2) > 0 && cmd.val2 != want2[0] {
				t.Fatalf("param 0x%x val2 = %d, want %d", param, cmd.val2, want2[0])
			}
			return
		}
		t.Fatalf("param 0x%x not found", param)
	}

	assertPayload := func(param int, wantArrSize int, wantKey string) {
		t.Helper()
		for _, cmd := range cmds {
			if cmd.param == param {
				if cmd.arrSize != wantArrSize {
					t.Fatalf("param 0x%x arrSize = %d, want %d", param, cmd.arrSize, wantArrSize)
				}
				if cmd.onlyOnChange != wantKey {
					t.Fatalf("param 0x%x onlyOnChange = %q, want %q", param, cmd.onlyOnChange, wantKey)
				}
				if len(cmd.payload) == 0 {
					t.Fatalf("param 0x%x payload empty", param)
				}
				return
			}
		}
		t.Fatalf("payload param 0x%x not found", param)
	}

	assertCommand(paramHPOutputVolume, dbLinearCenti(6))
	assertCommand(paramHPChannelPan, centi(-0.25))
	assertCommand(paramHPLimiter, centi(0.8))
	assertCommand(paramHPEQEnable, 1)
	assertCommand(paramHPEQBandCount, 18)
	assertCommand(paramHPEQBandLevel, 0, centi(1.5))
	assertCommand(paramHPBassMode, 1)
	assertCommand(paramHPBassFrequency, speakerHz(4))
	assertCommand(paramHPClarityMode, 2)
	assertCommand(paramHPHeadphoneStrength, 50)
	assertCommand(paramHPReverbWet, 25)
	assertCommand(paramHPConvolverEnable, 1)
	assertCommand(paramHPConvolverCrossChan, centi(0.3))
	assertPayload(paramHPConvolverSetKernel, len([]byte("C:\\irs\\room.wav")), "convolver_kernel")
	assertCommand(paramHPDDCEnable, 1)
	assertPayload(paramHPDDCCoefficients, 5, "ddc_coeffs")
	assertCommand(paramHPCureStrength, 2)
	assertCommand(paramHPAnalogXMode, 4)
	assertCommand(paramHPSpeakerCorrection, 1)
}

func TestCloseIsIdempotent(t *testing.T) {
	var dm DSPManager
	dm.Close()
	dm.Close()
}
