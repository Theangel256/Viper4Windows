//go:build windows

package main

import "testing"

func TestDefaultStateIncludesAdvancedDefaults(t *testing.T) {
	state := defaultState()

	if !state.Master.Power {
		t.Fatalf("master power default = false, want true")
	}
	if state.Output.Limiter != 1.0 {
		t.Fatalf("output limiter default = %v, want 1.0", state.Output.Limiter)
	}
	if state.Convolver.On {
		t.Fatalf("convolver default should be off")
	}
	if len(state.DDC.Coeffs44100) != 0 || len(state.DDC.Coeffs48000) != 0 {
		t.Fatalf("ddc defaults should be empty")
	}
	if state.DynamicSystem.XCoeffsLow != 120 || state.DynamicSystem.YCoeffsHigh != 200 {
		t.Fatalf("dynamic system defaults not preserved: %+v", state.DynamicSystem)
	}
	if state.Cure.StrengthPreset != 0 {
		t.Fatalf("cure preset default = %d, want 0", state.Cure.StrengthPreset)
	}
	if state.AnalogX.Mode != 0 {
		t.Fatalf("analogx mode default = %d, want 0", state.AnalogX.Mode)
	}
	if len(state.Equalizer) != 18 {
		t.Fatalf("equalizer len = %d, want 18", len(state.Equalizer))
	}
}

func TestDecodePresetStateMergesDefaultsAndSanitizes(t *testing.T) {
	data := []byte(`{
		"master": {"power": false},
		"output": {"pan": 2, "limiter": -1},
		"ddc": {
			"on": true,
			"coeffs44100": [1,2,3,4,5,6],
			"coeffs48000": [7,8,9,10,11]
		},
		"fetCompressor": {"gain": 30, "ratio": 25},
		"equalizer": [1,2]
	}`)

	state, err := decodePresetState(data)
	if err != nil {
		t.Fatalf("decodePresetState returned error: %v", err)
	}

	if state.Mode != "freestyle" {
		t.Fatalf("mode = %q, want freestyle", state.Mode)
	}
	if state.Master.Power {
		t.Fatalf("master power should have loaded as false")
	}
	if state.Master.PostVol != 12.0 {
		t.Fatalf("post volume default merge failed: got %v", state.Master.PostVol)
	}
	if state.Output.Pan != 1.0 || state.Output.Limiter != 0.0 {
		t.Fatalf("output sanitize failed: %+v", state.Output)
	}
	if len(state.DDC.Coeffs44100) != 5 || len(state.DDC.Coeffs48000) != 5 {
		t.Fatalf("ddc sanitize failed: 44k=%d 48k=%d", len(state.DDC.Coeffs44100), len(state.DDC.Coeffs48000))
	}
	if state.FETCompressor.Gain != 24.0 || state.FETCompressor.Ratio != 20.0 {
		t.Fatalf("fet sanitize failed: %+v", state.FETCompressor)
	}
	if len(state.Equalizer) != 18 {
		t.Fatalf("equalizer len = %d, want 18", len(state.Equalizer))
	}
	if state.Equalizer[0] != 1 || state.Equalizer[1] != 2 || state.Equalizer[2] != 0 {
		t.Fatalf("equalizer merge failed: %v", state.Equalizer[:3])
	}
}
