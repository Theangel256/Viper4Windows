package services

import (
	"fmt"

	"viper4windows/internal/domain/models"
	"viper4windows/internal/domain/ports"
	"viper4windows/internal/dsp/protocol"
	"viper4windows/internal/utils"
)

// ═══════════════════════════════════════════════════════════════════════════
// DSP Parameter Service
// ═══════════════════════════════════════════════════════════════════════════
// Handles encoding, validation, and normalization of DSP parameters.
// Separates business logic from IPC concerns.
//
// EncodeState used to hand-roll the same 256×float32 shared-memory layout
// that internal/dsp/protocol.EncodeDSPState already implements (same
// models.Idx* constants, same bytes on the wire) across ~20 private
// encode* methods. That duplication is gone — this now just calls the
// protocol package, which is also the one with round-trip tests.

type DSPParameterService struct {
	logger ports.Logger
}

func NewDSPParameterService(logger ports.Logger) *DSPParameterService {
	return &DSPParameterService{
		logger: logger.WithContext("DSPParameters"),
	}
}

// EncodeState converts DSPState to binary format for shared memory
func (s *DSPParameterService) EncodeState(state models.DSPState) ([]byte, error) {
	return protocol.EncodeDSPState(state).Bytes(), nil
}

// ValidateState checks if all parameters are within valid ranges.
//
// The eight validate* methods for AGC/DynamicSystem/SpectrumExtension/
// FieldSurround/DiffSurround/FETCompressor/TubeSimulator/DDC already
// existed in this file but were never called from here — dead code that
// meant those seven effects (DDC has nothing to validate) could silently
// hold out-of-range values. They're wired in now; behavior for the six
// fields validated below is unchanged.
func (s *DSPParameterService) ValidateState(state *models.DSPState) error {
	if err := s.validateMaster(&state.Master); err != nil {
		return fmt.Errorf("master: %w", err)
	}
	if err := s.validateOutput(&state.Output); err != nil {
		return fmt.Errorf("output: %w", err)
	}

	if len(state.Equalizer) != models.EqBands {
		return fmt.Errorf("equalizer: expected %d bands, got %d",
			models.EqBands, len(state.Equalizer))
	}
	for i, val := range state.Equalizer {
		if val < models.MinEqBand || val > models.MaxEqBand {
			return fmt.Errorf("equalizer band %d: value %.2f out of range [%.2f, %.2f]",
				i, val, models.MinEqBand, models.MaxEqBand)
		}
	}

	if err := s.validateXBass(&state.XBass); err != nil {
		return fmt.Errorf("xBass: %w", err)
	}
	if err := s.validateXClarity(&state.XClarity); err != nil {
		return fmt.Errorf("xClarity: %w", err)
	}
	if err := s.validateSurround3D(&state.Surround3D); err != nil {
		return fmt.Errorf("surround3D: %w", err)
	}
	if err := s.validateAGC(&state.AGC); err != nil {
		return fmt.Errorf("agc: %w", err)
	}
	if err := s.validateDynamicSystem(&state.DynamicSystem); err != nil {
		return fmt.Errorf("dynamicSystem: %w", err)
	}
	if err := s.validateSpectrumExtension(&state.SpectrumExtension); err != nil {
		return fmt.Errorf("spectrumExtension: %w", err)
	}
	if err := s.validateFieldSurround(&state.FieldSurround); err != nil {
		return fmt.Errorf("fieldSurround: %w", err)
	}
	if err := s.validateDiffSurround(&state.DiffSurround); err != nil {
		return fmt.Errorf("diffSurround: %w", err)
	}
	if err := s.validateFETCompressor(&state.FETCompressor); err != nil {
		return fmt.Errorf("fetCompressor: %w", err)
	}
	if err := s.validateTubeSimulator(&state.TubeSimulator); err != nil {
		return fmt.Errorf("tubeSimulator: %w", err)
	}
	if err := s.validateDDC(&state.DDC); err != nil {
		return fmt.Errorf("ddc: %w", err)
	}

	return nil
}

// NormalizeState clamps all parameters to valid ranges
func (s *DSPParameterService) NormalizeState(state models.DSPState) models.DSPState {
	state.Master.PreVol = utils.Clamp(state.Master.PreVol, models.MinPreVol, models.MaxPreVol)
	state.Master.PostVol = utils.Clamp(state.Master.PostVol, models.MinPostVol, models.MaxPostVol)

	state.Output.Pan = utils.Clamp(state.Output.Pan, models.MinPan, models.MaxPan)
	state.Output.Limiter = utils.Clamp(state.Output.Limiter, models.MinLimiter, models.MaxLimiter)

	if len(state.Equalizer) != models.EqBands {
		eq := make([]float64, models.EqBands)
		copy(eq, state.Equalizer)
		state.Equalizer = eq
	}
	for i := range state.Equalizer {
		state.Equalizer[i] = utils.Clamp(state.Equalizer[i], models.MinEqBand, models.MaxEqBand)
	}

	state.XBass.Level = utils.Clamp(state.XBass.Level, models.MinXBassLevel, models.MaxXBassLevel)
	state.XBass.SpeakerSize = utils.ClampInt(state.XBass.SpeakerSize, models.MinSpeakerSize, models.MaxSpeakerSize)

	state.XClarity.Level = utils.Clamp(state.XClarity.Level, models.MinXClarityLevel, models.MaxXClarityLevel)

	state.Surround3D.SpaceSize = utils.ClampInt(state.Surround3D.SpaceSize, models.MinSpaceSize, models.MaxSpaceSize)
	state.Surround3D.ImageSize = utils.ClampInt(state.Surround3D.ImageSize, models.MinImageSize, models.MaxImageSize)

	state.AGC.Ratio = utils.Clamp(state.AGC.Ratio, 0.5, 20.0)
	state.AGC.Volume = utils.Clamp(state.AGC.Volume, -12.0, 12.0)
	state.AGC.MaxScaler = utils.Clamp(state.AGC.MaxScaler, 1.0, 10.0)

	state.DynamicSystem.SideGainX = utils.Clamp(state.DynamicSystem.SideGainX, 0.0, 2.0)
	state.DynamicSystem.SideGainY = utils.Clamp(state.DynamicSystem.SideGainY, 0.0, 2.0)
	state.DynamicSystem.Strength = utils.Clamp(state.DynamicSystem.Strength, 0.0, 1.0)

	state.SpectrumExtension.ReferenceFrequency = utils.ClampInt(state.SpectrumExtension.ReferenceFrequency, 1000, 20000)
	state.SpectrumExtension.Exciter = utils.Clamp(state.SpectrumExtension.Exciter, 0.0, 1.0)

	state.FieldSurround.Widening = utils.Clamp(state.FieldSurround.Widening, 0.0, 1.0)
	state.FieldSurround.MidImage = utils.Clamp(state.FieldSurround.MidImage, 0.0, 1.0)
	state.FieldSurround.Depth = utils.ClampInt(state.FieldSurround.Depth, 0, 100)

	state.DiffSurround.Delay = utils.Clamp(state.DiffSurround.Delay, 0.0, 0.5)

	state.FETCompressor.Threshold = utils.Clamp(state.FETCompressor.Threshold, -60.0, 0.0)
	state.FETCompressor.Ratio = utils.Clamp(state.FETCompressor.Ratio, 1.0, 20.0)
	state.FETCompressor.Knee = utils.Clamp(state.FETCompressor.Knee, 0.0, 12.0)
	state.FETCompressor.Gain = utils.Clamp(state.FETCompressor.Gain, -12.0, 12.0)
	state.FETCompressor.Attack = utils.Clamp(state.FETCompressor.Attack, 0.01, 100.0)
	state.FETCompressor.Release = utils.Clamp(state.FETCompressor.Release, 10.0, 1000.0)

	return state
}

// ═══════════════════════════════════════════════════════════════════════════
// Private Validation Methods
// ═══════════════════════════════════════════════════════════════════════════

func (s *DSPParameterService) validateMaster(m *models.MasterState) error {
	if m.PreVol < models.MinPreVol || m.PreVol > models.MaxPreVol {
		return fmt.Errorf("preVol %.2f out of range [%.2f, %.2f]",
			m.PreVol, models.MinPreVol, models.MaxPreVol)
	}
	if m.PostVol < models.MinPostVol || m.PostVol > models.MaxPostVol {
		return fmt.Errorf("postVol %.2f out of range [%.2f, %.2f]",
			m.PostVol, models.MinPostVol, models.MaxPostVol)
	}
	return nil
}

func (s *DSPParameterService) validateOutput(o *models.OutputState) error {
	if o.Pan < models.MinPan || o.Pan > models.MaxPan {
		return fmt.Errorf("pan %.2f out of range [%.2f, %.2f]",
			o.Pan, models.MinPan, models.MaxPan)
	}
	if o.Limiter < models.MinLimiter || o.Limiter > models.MaxLimiter {
		return fmt.Errorf("limiter %.2f out of range [%.2f, %.2f]",
			o.Limiter, models.MinLimiter, models.MaxLimiter)
	}
	return nil
}

func (s *DSPParameterService) validateXBass(x *models.XBassState) error {
	if x.Level < models.MinXBassLevel || x.Level > models.MaxXBassLevel {
		return fmt.Errorf("level %.2f out of range [%.2f, %.2f]",
			x.Level, models.MinXBassLevel, models.MaxXBassLevel)
	}
	if x.SpeakerSize < models.MinSpeakerSize || x.SpeakerSize > models.MaxSpeakerSize {
		return fmt.Errorf("speakerSize %d out of range [%d, %d]",
			x.SpeakerSize, models.MinSpeakerSize, models.MaxSpeakerSize)
	}
	return nil
}

func (s *DSPParameterService) validateXClarity(x *models.XClarityState) error {
	if x.Level < models.MinXClarityLevel || x.Level > models.MaxXClarityLevel {
		return fmt.Errorf("level %.2f out of range [%.2f, %.2f]",
			x.Level, models.MinXClarityLevel, models.MaxXClarityLevel)
	}
	return nil
}

func (s *DSPParameterService) validateSurround3D(s3d *models.Surround3DState) error {
	if s3d.SpaceSize < models.MinSpaceSize || s3d.SpaceSize > models.MaxSpaceSize {
		return fmt.Errorf("spaceSize %d out of range [%d, %d]",
			s3d.SpaceSize, models.MinSpaceSize, models.MaxSpaceSize)
	}
	if s3d.ImageSize < models.MinImageSize || s3d.ImageSize > models.MaxImageSize {
		return fmt.Errorf("imageSize %d out of range [%d, %d]",
			s3d.ImageSize, models.MinImageSize, models.MaxImageSize)
	}
	return nil
}

func (s *DSPParameterService) validateDDC(ddc *models.DDCState) error {
	// DDC coefficients are optional, no validation needed beyond type safety
	return nil
}

func (s *DSPParameterService) validateAGC(agc *models.AGCState) error {
	if agc.Ratio < 0.5 || agc.Ratio > 20.0 {
		return fmt.Errorf("ratio %.2f out of range [0.5, 20.0]", agc.Ratio)
	}
	if agc.Volume < -12.0 || agc.Volume > 12.0 {
		return fmt.Errorf("volume %.2f out of range [-12.0, 12.0]", agc.Volume)
	}
	if agc.MaxScaler < 1.0 || agc.MaxScaler > 10.0 {
		return fmt.Errorf("maxScaler %.2f out of range [1.0, 10.0]", agc.MaxScaler)
	}
	return nil
}

func (s *DSPParameterService) validateDynamicSystem(ds *models.DynamicSystemState) error {
	if ds.SideGainX < 0.0 || ds.SideGainX > 2.0 {
		return fmt.Errorf("sideGainX %.2f out of range [0.0, 2.0]", ds.SideGainX)
	}
	if ds.SideGainY < 0.0 || ds.SideGainY > 2.0 {
		return fmt.Errorf("sideGainY %.2f out of range [0.0, 2.0]", ds.SideGainY)
	}
	if ds.Strength < 0.0 || ds.Strength > 1.0 {
		return fmt.Errorf("strength %.2f out of range [0.0, 1.0]", ds.Strength)
	}
	return nil
}

func (s *DSPParameterService) validateSpectrumExtension(se *models.SpectrumExtensionState) error {
	if se.ReferenceFrequency < 1000 || se.ReferenceFrequency > 20000 {
		return fmt.Errorf("referenceFrequency %d out of range [1000, 20000]", se.ReferenceFrequency)
	}
	if se.Exciter < 0.0 || se.Exciter > 1.0 {
		return fmt.Errorf("exciter %.2f out of range [0.0, 1.0]", se.Exciter)
	}
	return nil
}

func (s *DSPParameterService) validateFieldSurround(fs *models.FieldSurroundState) error {
	if fs.Widening < 0.0 || fs.Widening > 1.0 {
		return fmt.Errorf("widening %.2f out of range [0.0, 1.0]", fs.Widening)
	}
	if fs.MidImage < 0.0 || fs.MidImage > 1.0 {
		return fmt.Errorf("midImage %.2f out of range [0.0, 1.0]", fs.MidImage)
	}
	if fs.Depth < 0 || fs.Depth > 100 {
		return fmt.Errorf("depth %d out of range [0, 100]", fs.Depth)
	}
	return nil
}

func (s *DSPParameterService) validateDiffSurround(ds *models.DiffSurroundState) error {
	if ds.Delay < 0.0 || ds.Delay > 0.5 {
		return fmt.Errorf("delay %.2f out of range [0.0, 0.5]", ds.Delay)
	}
	return nil
}

func (s *DSPParameterService) validateFETCompressor(fet *models.FETCompressorState) error {
	if fet.Threshold < -60.0 || fet.Threshold > 0.0 {
		return fmt.Errorf("threshold %.2f out of range [-60.0, 0.0]", fet.Threshold)
	}
	if fet.Ratio < 1.0 || fet.Ratio > 20.0 {
		return fmt.Errorf("ratio %.2f out of range [1.0, 20.0]", fet.Ratio)
	}
	if fet.Knee < 0.0 || fet.Knee > 12.0 {
		return fmt.Errorf("knee %.2f out of range [0.0, 12.0]", fet.Knee)
	}
	if fet.Gain < -12.0 || fet.Gain > 12.0 {
		return fmt.Errorf("gain %.2f out of range [-12.0, 12.0]", fet.Gain)
	}
	if fet.Attack < 0.01 || fet.Attack > 100.0 {
		return fmt.Errorf("attack %.2f out of range [0.01, 100.0]", fet.Attack)
	}
	if fet.Release < 10.0 || fet.Release > 1000.0 {
		return fmt.Errorf("release %.2f out of range [10.0, 1000.0]", fet.Release)
	}
	return nil
}

func (s *DSPParameterService) validateTubeSimulator(tube *models.TubeSimulatorState) error {
	// Tube simulator is a simple on/off effect, no additional validation
	return nil
}
