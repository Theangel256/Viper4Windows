package services

import (
	"bytes"
	"testing"

	"viper4windows/internal/domain/models"
	"viper4windows/internal/dsp/protocol"
	"viper4windows/internal/utils"
)

// ── fakes ────────────────────────────────────────────────────────────
//
// ports.Logger is satisfied by utils.NewLogger("test") directly, so
// tests below use that instead of a hand-rolled fake.

type fakeDLL struct {
	loaded   bool
	dispatch []int
	payloads int
}

func (f *fakeDLL) LoadDLLPath(string) error { return nil }
func (f *fakeDLL) Close() error             { return nil }
func (f *fakeDLL) IsLoaded() bool           { return f.loaded }
func (f *fakeDLL) Dispatch(param, _, _, _, _ int) {
	f.dispatch = append(f.dispatch, param)
}
func (f *fakeDLL) DispatchPayload(param, _, _, _, _, _ int, _ []byte) {
	f.dispatch = append(f.dispatch, param)
	f.payloads++
}
func (f *fakeDLL) SetSampleRate(uint32) {}
func (f *fakeDLL) Reset()               {}

type fakeSHM struct {
	ready   bool
	written [][]byte
}

func (f *fakeSHM) Open() error                       { return nil }
func (f *fakeSHM) Close() error                      { return nil }
func (f *fakeSHM) WriteParams(models.DSPState) error { return nil }
func (f *fakeSHM) Write(data []byte) error {
	f.written = append(f.written, append([]byte(nil), data...))
	return nil
}
func (f *fakeSHM) ReadAPOStatus() (models.AudioEngineStatus, error) {
	return models.AudioEngineStatus{}, nil
}
func (f *fakeSHM) IsConnected() bool { return f.ready }
func (f *fakeSHM) IsReady() bool     { return f.ready }
func (f *fakeSHM) Signal() error     { return nil }

// ── DSPParameterService ─────────────────────────────────────────────

func TestEncodeStateMatchesProtocolPackage(t *testing.T) {
	svc := NewDSPParameterService(utils.NewLogger("test"))
	state := models.NewDefaultState()
	state.Equalizer[5] = 3.5

	got, err := svc.EncodeState(state)
	if err != nil {
		t.Fatalf("EncodeState: %v", err)
	}

	want := protocol.EncodeDSPState(state).Bytes()
	if !bytes.Equal(got, want) {
		t.Fatal("EncodeState no longer matches protocol.EncodeDSPState — the delegation is broken")
	}
}

func TestValidateStateNowChecksPreviouslyDeadFields(t *testing.T) {
	svc := NewDSPParameterService(utils.NewLogger("test"))
	state := models.NewDefaultState()

	// Before this pass, validateAGC existed but ValidateState never
	// called it — an out-of-range AGC.Ratio passed silently. Confirm
	// it's actually caught now.
	state.AGC.Ratio = 999.0
	if err := svc.ValidateState(&state); err == nil {
		t.Fatal("expected ValidateState to reject AGC.Ratio = 999.0, got nil error")
	}

	state = models.NewDefaultState()
	state.FETCompressor.Threshold = 500.0
	if err := svc.ValidateState(&state); err == nil {
		t.Fatal("expected ValidateState to reject FETCompressor.Threshold = 500.0, got nil error")
	}

	// A fully default state must still validate cleanly.
	state = models.NewDefaultState()
	if err := svc.ValidateState(&state); err != nil {
		t.Fatalf("default state should validate cleanly, got: %v", err)
	}
}

func TestNormalizeStateClamps(t *testing.T) {
	svc := NewDSPParameterService(utils.NewLogger("test"))
	state := models.NewDefaultState()
	state.Master.PreVol = -999
	state.AGC.Ratio = 999

	normalized := svc.NormalizeState(state)
	if normalized.Master.PreVol != models.MinPreVol {
		t.Errorf("PreVol = %v, want clamped to %v", normalized.Master.PreVol, models.MinPreVol)
	}
	if normalized.AGC.Ratio != 20.0 {
		t.Errorf("AGC.Ratio = %v, want clamped to 20.0", normalized.AGC.Ratio)
	}
}

// ── DSPManagerService ────────────────────────────────────────────────

func TestApplyChangesUsesBothPathsWhenReady(t *testing.T) {
	dll := &fakeDLL{loaded: true}
	shm := &fakeSHM{ready: true}
	paramSvc := NewDSPParameterService(utils.NewLogger("test"))
	mgr := NewDSPManagerService(utils.NewLogger("test"), dll, shm, paramSvc)

	if !mgr.IsDLLReady() || !mgr.IsSHMReady() {
		t.Fatal("expected both paths ready")
	}

	if err := mgr.ApplyChanges(models.NewDefaultState()); err != nil {
		t.Fatalf("ApplyChanges: %v", err)
	}
	if len(dll.dispatch) == 0 {
		t.Error("expected at least one DLL dispatch call")
	}
	if len(shm.written) != 1 {
		t.Errorf("expected exactly one SHM write, got %d", len(shm.written))
	}
}

func TestShouldDispatchPayloadSkipsUnchangedPayload(t *testing.T) {
	dll := &fakeDLL{loaded: true}
	mgr := NewDSPManagerService(utils.NewLogger("test"), dll, &fakeSHM{}, NewDSPParameterService(utils.NewLogger("test")))

	state := models.NewDefaultState()
	state.Convolver.On = true
	state.Convolver.KernelPath = "C:\\kernels\\room.wav"

	if err := mgr.applyViaDLL(state); err != nil {
		t.Fatalf("applyViaDLL (1st): %v", err)
	}
	firstPayloadCount := dll.payloads
	if firstPayloadCount == 0 {
		t.Fatal("expected the convolver kernel path to dispatch as a payload at least once")
	}

	if err := mgr.applyViaDLL(state); err != nil {
		t.Fatalf("applyViaDLL (2nd, unchanged): %v", err)
	}
	if dll.payloads != firstPayloadCount {
		t.Errorf("payload dispatch count changed on an unchanged kernel path: %d -> %d", firstPayloadCount, dll.payloads)
	}

	state.Convolver.KernelPath = "C:\\kernels\\hall.wav"
	if err := mgr.applyViaDLL(state); err != nil {
		t.Fatalf("applyViaDLL (3rd, changed): %v", err)
	}
	if dll.payloads != firstPayloadCount+1 {
		t.Errorf("expected exactly one more payload dispatch after changing the kernel path, got %d -> %d", firstPayloadCount, dll.payloads)
	}
}
