package protocol

import (
	"math"
	"testing"

	"viper4windows/internal/domain/models"
)

func TestParamBufferRoundTrip(t *testing.T) {
	state := models.NewDefaultState()
	state.Equalizer[0] = 6.5
	state.Equalizer[3] = -3.25

	buf := EncodeDSPState(state)
	data := buf.Bytes()

	if len(data) != models.SharedMemBytes {
		t.Fatalf("Bytes(): expected %d bytes, got %d", models.SharedMemBytes, len(data))
	}

	decoded, err := DecodeParamBuffer(data)
	if err != nil {
		t.Fatalf("DecodeParamBuffer: %v", err)
	}
	if decoded != buf {
		t.Fatalf("round-trip mismatch: encoded and decoded ParamBuffer differ")
	}

	if got := decoded[models.IdxEnabled]; got != 1 {
		t.Errorf("IdxEnabled = %v, want 1 (Master.Power default true)", got)
	}
	if got := decoded[models.IdxEQBands]; math.Abs(float64(got)-6.5) > 1e-6 {
		t.Errorf("IdxEQBands[0] = %v, want 6.5", got)
	}
	if got := decoded[models.IdxEQBands+3]; math.Abs(float64(got)+3.25) > 1e-6 {
		t.Errorf("IdxEQBands[3] = %v, want -3.25", got)
	}
}

func TestDecodeParamBufferRejectsWrongSize(t *testing.T) {
	if _, err := DecodeParamBuffer(make([]byte, 10)); err == nil {
		t.Fatal("expected an error for a buffer of the wrong size, got nil")
	}
}

func TestDispatchNamespaceOffset(t *testing.T) {
	state := models.NewDefaultState()

	hp := BuildDispatchCommands(state, Headphone)
	spk := BuildDispatchCommands(state, Speaker)

	if len(hp) == 0 {
		t.Fatal("BuildDispatchCommands(Headphone) returned no commands")
	}

	// Every HP command except SpeakerCorrection (not emitted for HP at
	// all) must have a matching SPK command exactly 0x200 higher.
	spkParams := make(map[int]bool, len(spk))
	for _, c := range spk {
		spkParams[c.Param] = true
	}
	for _, c := range hp {
		if !spkParams[c.Param+namespaceOffset] {
			t.Errorf("HP param 0x%X has no SPK twin at 0x%X", c.Param, c.Param+namespaceOffset)
		}
	}

	// SpeakerCorrection must appear only in the Speaker list, using the
	// real (unshifted) SPK-only code from ViPERParams.h.
	foundInSpk := false
	for _, c := range spk {
		if c.Param == paramSpeakerCorrectionEnable {
			foundInSpk = true
		}
	}
	if !foundInSpk {
		t.Error("expected paramSpeakerCorrectionEnable in the Speaker command list")
	}
	for _, c := range hp {
		if c.Param == paramSpeakerCorrectionEnable {
			t.Error("paramSpeakerCorrectionEnable must not appear in the Headphone command list")
		}
	}
}

func TestDispatchFETUsesBoolCenti(t *testing.T) {
	state := models.NewDefaultState()
	state.FETCompressor.On = true
	state.FETCompressor.NoClip = true

	cmds := BuildDispatchCommands(state, Headphone)

	var enableVal, noClipVal int32 = -1, -1
	for _, c := range cmds {
		switch c.Param {
		case paramFETEnable:
			enableVal = c.Val1
		case paramFETNoClip:
			noClipVal = c.Val1
		}
	}
	if enableVal != 100 {
		t.Errorf("FET enable = %d, want 100 (boolCenti convention)", enableVal)
	}
	if noClipVal != 100 {
		t.Errorf("FET no-clip = %d, want 100 (boolCenti convention)", noClipVal)
	}
}

func TestDispatchEQOneBandAtATime(t *testing.T) {
	state := models.NewDefaultState()
	state.Equalizer[2] = 4.0

	cmds := BuildDispatchCommands(state, Headphone)

	found := false
	for _, c := range cmds {
		if c.Param == paramEQBandLevel && c.Val1 == 2 {
			found = true
			if c.Val2 != 400 {
				t.Errorf("EQ band 2 centi value = %d, want 400", c.Val2)
			}
		}
		if c.Param == paramEQBandLevel && c.ArrSize != 0 {
			t.Error("EQ band dispatch must not carry a bulk payload — it's one band per call")
		}
	}
	if !found {
		t.Fatal("expected a per-band EQ dispatch command for band index 2")
	}
}

func TestDDCRequiresBothCoefficientSets(t *testing.T) {
	state := models.NewDefaultState()
	state.DDC.On = true
	state.DDC.Coeffs44100 = []float64{1, 2, 3, 4, 5}
	// Coeffs48000 intentionally left empty.

	cmds := BuildDispatchCommands(state, Headphone)
	for _, c := range cmds {
		if c.Param == paramDDCEnable && c.Val1 != 0 {
			t.Error("DDC must stay disabled when only one coefficient set is present")
		}
	}
}
