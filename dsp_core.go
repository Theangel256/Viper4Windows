// Package dsp provides pure Go implementations of ViPER audio processing algorithms.
// These implementations are based on reverse-engineered C++ code from ViPERFX_RE.
//
// References:
// - ViPERFX_RE: https://github.com/likelikeslike/ViPERFX_RE
// - Audio EQ Cookbook: Robert Bristow-Johnson
// - Freeverb: Jezar at Dreampoint
package main

import "math"

// ═══════════════════════════════════════════════════════════════════════════
// BIQUAD FILTER (Foundation for all EQ/filters)
// ═══════════════════════════════════════════════════════════════════════════

// BiquadCoefficients represents a second-order IIR filter (biquad).
// This is the foundation for all EQ bands, bass, and clarity filters.
type BiquadCoefficients struct {
	B0, B1, B2 float32 // Numerator coefficients
	A1, A2     float32 // Denominator coefficients (a0 normalized to 1.0)
}

// BiquadFilter is a stateful biquad filter using Direct Form I topology.
// Ref: ViPERFX_RE/src/effects/biquad.cpp
type BiquadFilter struct {
	Coefs      BiquadCoefficients
	Z1, Z2     float32 // Delay line (state variables)
	SampleRate float32
}

// ProcessSample processes a single sample through the biquad filter.
// This is the hot path - optimized for minimal CPU cycles.
//
// Direct Form I implementation:
// y[n] = b0*x[n] + b1*x[n-1] + b2*x[n-2] - a1*y[n-1] - a2*y[n-2]
func (f *BiquadFilter) ProcessSample(in float32) float32 {
	out := f.Coefs.B0*in + f.Z1
	f.Z1 = f.Coefs.B1*in - f.Coefs.A1*out + f.Z2
	f.Z2 = f.Coefs.B2*in - f.Coefs.A2*out
	return out
}

// Reset clears the filter state (for discontinuities or format changes)
func (f *BiquadFilter) Reset() {
	f.Z1 = 0
	f.Z2 = 0
}

// ═══════════════════════════════════════════════════════════════════════════
// FILTER COEFFICIENT CALCULATIONS (Audio EQ Cookbook)
// ═══════════════════════════════════════════════════════════════════════════

// CalculatePeakingEQ generates coefficients for a parametric peaking filter.
// Used for: Equalizer bands
//
// Parameters:
//   - freq: Center frequency (Hz)
//   - sampleRate: Sample rate (Hz)
//   - gain: Boost/cut in dB (positive = boost, negative = cut)
//   - q: Quality factor (bandwidth control, typically 0.7-2.0)
//
// Ref: Audio EQ Cookbook by Robert Bristow-Johnson
func CalculatePeakingEQ(freq, sampleRate, gain, q float32) BiquadCoefficients {
	A := float32(math.Pow(10, float64(gain/40.0))) // Convert dB to linear
	omega := 2 * math.Pi * freq / sampleRate
	sinOmega := float32(math.Sin(float64(omega)))
	cosOmega := float32(math.Cos(float64(omega)))
	alpha := sinOmega / (2 * q)

	b0 := 1 + alpha*A
	b1 := -2 * cosOmega
	b2 := 1 - alpha*A
	a0 := 1 + alpha/A
	a1 := -2 * cosOmega
	a2 := 1 - alpha/A

	return BiquadCoefficients{
		B0: b0 / a0,
		B1: b1 / a0,
		B2: b2 / a0,
		A1: a1 / a0,
		A2: a2 / a0,
	}
}

// CalculateLowShelf generates coefficients for a low-frequency shelf filter.
// Used for: ViPER Bass (Natural Mode)
//
// Ref: Audio EQ Cookbook
func CalculateLowShelf(freq, sampleRate, gain, q float32) BiquadCoefficients {
	A := float32(math.Pow(10, float64(gain/40.0)))
	omega := 2 * math.Pi * freq / sampleRate
	sinOmega := float32(math.Sin(float64(omega)))
	cosOmega := float32(math.Cos(float64(omega)))
	alpha := sinOmega / (2 * q)
	sqrtA := float32(math.Sqrt(float64(A)))

	b0 := A * ((A + 1) - (A-1)*cosOmega + 2*sqrtA*alpha)
	b1 := 2 * A * ((A - 1) - (A+1)*cosOmega)
	b2 := A * ((A + 1) - (A-1)*cosOmega - 2*sqrtA*alpha)
	a0 := (A + 1) + (A-1)*cosOmega + 2*sqrtA*alpha
	a1 := -2 * ((A - 1) + (A+1)*cosOmega)
	a2 := (A + 1) + (A-1)*cosOmega - 2*sqrtA*alpha

	return BiquadCoefficients{
		B0: b0 / a0,
		B1: b1 / a0,
		B2: b2 / a0,
		A1: a1 / a0,
		A2: a2 / a0,
	}
}

// CalculateLowPass generates coefficients for a low-pass filter.
// Used for: ViPER Bass (Pure Mode), isolation
//
// Ref: Audio EQ Cookbook
func CalculateLowPass(freq, sampleRate, q float32) BiquadCoefficients {
	omega := 2 * math.Pi * freq / sampleRate
	sinOmega := float32(math.Sin(float64(omega)))
	cosOmega := float32(math.Cos(float64(omega)))
	alpha := sinOmega / (2 * q)

	b0 := (1 - cosOmega) / 2
	b1 := 1 - cosOmega
	b2 := (1 - cosOmega) / 2
	a0 := 1 + alpha
	a1 := -2 * cosOmega
	a2 := 1 - alpha

	return BiquadCoefficients{
		B0: b0 / a0,
		B1: b1 / a0,
		B2: b2 / a0,
		A1: a1 / a0,
		A2: a2 / a0,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// EQUALIZER (18-Band Precision)
// ═══════════════════════════════════════════════════════════════════════════

// Equalizer18Band implements ViPER's 18-band parametric equalizer.
// Each band uses a cascaded biquad peaking filter.
//
// Ref: ViPERFX_RE/src/effects/equalizer.cpp
type Equalizer18Band struct {
	Bands       [18]BiquadFilter
	Frequencies [18]float32
	SampleRate  float32
}

// NewEqualizer18Band creates an equalizer with ISO standard frequencies.
// Frequency distribution: 65Hz to 20kHz (logarithmic spacing)
func NewEqualizer18Band(sampleRate float32) *Equalizer18Band {
	eq := &Equalizer18Band{
		Frequencies: [18]float32{
			65, 92, 131, 185, 262, 370, 523, 740,
			1047, 1480, 2093, 2960, 4186, 5920, 8372, 11840, 16744, 20000,
		},
		SampleRate: sampleRate,
	}

	// Initialize all bands with 0 dB gain (flat response)
	for i := 0; i < 18; i++ {
		eq.Bands[i].Coefs = CalculatePeakingEQ(
			eq.Frequencies[i],
			sampleRate,
			0.0,  // gain in dB
			1.41, // Q factor (slightly wide for musical EQ)
		)
		eq.Bands[i].SampleRate = sampleRate
	}

	return eq
}

// SetBandGain adjusts the gain of a specific band.
// This recalculates the filter coefficients.
//
// Parameters:
//   - band: Band index (0-17)
//   - gainDB: Gain in dB (-12 to +12 typical range)
func (eq *Equalizer18Band) SetBandGain(band int, gainDB float32) {
	if band < 0 || band >= 18 {
		return
	}
	eq.Bands[band].Coefs = CalculatePeakingEQ(
		eq.Frequencies[band],
		eq.SampleRate,
		gainDB,
		1.41,
	)
}

// ProcessBuffer processes a buffer of audio samples (stereo interleaved).
// For mono, call with channels=1.
//
// PERFORMANCE: This is the critical hot path. Optimize for:
// - Cache locality (sequential access)
// - No allocations (pre-allocated buffers)
// - SIMD-friendly (consider asm in production)
func (eq *Equalizer18Band) ProcessBuffer(buffer []float32, channels int) {
	if channels == 1 {
		// Mono: process each sample through all 18 bands
		for i := range buffer {
			sample := buffer[i]
			for b := 0; b < 18; b++ {
				sample = eq.Bands[b].ProcessSample(sample)
			}
			buffer[i] = sample
		}
	} else {
		// Stereo: process left and right separately (but use same coefficients)
		// NOTE: For true stereo EQ, need separate state per channel
		for i := 0; i < len(buffer); i += 2 {
			// Left channel
			left := buffer[i]
			for b := 0; b < 18; b++ {
				left = eq.Bands[b].ProcessSample(left)
			}
			buffer[i] = left

			// Right channel
			if i+1 < len(buffer) {
				right := buffer[i+1]
				for b := 0; b < 18; b++ {
					right = eq.Bands[b].ProcessSample(right)
				}
				buffer[i+1] = right
			}
		}
	}
}

// Reset clears all filter states (e.g., on format change)
func (eq *Equalizer18Band) Reset() {
	for i := 0; i < 18; i++ {
		eq.Bands[i].Reset()
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// VIPER BASS (Natural + Pure Mode)
// ═══════════════════════════════════════════════════════════════════════════

// ViPERBass implements ViPER's bass enhancement algorithm.
// Two modes:
//   - Natural: Low-frequency shelf boost (clean)
//   - Pure: Sub-harmonic synthesis (aggressive)
//
// Ref: ViPERFX_RE/src/effects/viper_bass.cpp
type ViPERBass struct {
	Mode        int     // 0 = Natural, 1 = Pure
	SpeakerSize float32 // Cutoff frequency (Hz) - simulates speaker size
	Gain        float32 // Boost in dB

	// Internal filters
	lpf        BiquadFilter // Low-pass for isolation
	shelf      BiquadFilter // Shelf for Natural mode
	harmonic   BiquadFilter // Sub-harmonic generator for Pure mode
	SampleRate float32
}

// NewViPERBass creates a ViPER Bass processor.
func NewViPERBass(sampleRate float32) *ViPERBass {
	vb := &ViPERBass{
		Mode:        0,
		SpeakerSize: 60.0,
		Gain:        0.0,
		SampleRate:  sampleRate,
	}
	vb.UpdateFilters()
	return vb
}

// UpdateFilters recalculates filter coefficients when parameters change.
func (vb *ViPERBass) UpdateFilters() {
	cutoff := vb.SpeakerSize

	if vb.Mode == 0 {
		// Natural Bass: Use shelf filter for gentle boost
		vb.shelf.Coefs = CalculateLowShelf(cutoff, vb.SampleRate, vb.Gain, 0.707)
	} else {
		// Pure Bass: Generate sub-harmonics
		vb.lpf.Coefs = CalculateLowPass(cutoff, vb.SampleRate, 0.707)
		vb.harmonic.Coefs = CalculatePeakingEQ(cutoff/2, vb.SampleRate, vb.Gain, 2.0)
	}
}

// ProcessSample processes a single sample.
//
// Natural Mode: Simple shelf filter
// Pure Mode: Extracts low frequencies, generates harmonics via non-linearity
func (vb *ViPERBass) ProcessSample(in float32) float32 {
	if vb.Mode == 0 {
		// Natural: Just apply shelf
		return vb.shelf.ProcessSample(in)
	} else {
		// Pure: Low-pass → square → harmonic filter → mix
		lowFreq := vb.lpf.ProcessSample(in)
		// Non-linear distortion (x² generates harmonics)
		distorted := lowFreq * lowFreq
		if lowFreq < 0 {
			distorted = -distorted // Preserve sign
		}
		subHarmonic := vb.harmonic.ProcessSample(distorted)
		return in + subHarmonic*0.5 // Mix original + enhanced bass
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// REVERB (Freeverb Algorithm)
// ═══════════════════════════════════════════════════════════════════════════

// CombFilter is a feedback comb filter (IIR delay line).
type CombFilter struct {
	buffer      []float32
	index       int
	feedback    float32
	damping     float32
	filterStore float32
}

// Process processes a sample through the comb filter.
func (cf *CombFilter) Process(in float32) float32 {
	output := cf.buffer[cf.index]
	cf.filterStore = (output * (1 - cf.damping)) + (cf.filterStore * cf.damping)
	cf.buffer[cf.index] = in + (cf.filterStore * cf.feedback)
	cf.index = (cf.index + 1) % len(cf.buffer)
	return output
}

// AllpassFilter is a feedforward-feedback allpass filter.
type AllpassFilter struct {
	buffer []float32
	index  int
}

// Process processes a sample through the allpass filter.
func (ap *AllpassFilter) Process(in float32) float32 {
	bufOut := ap.buffer[ap.index]
	ap.buffer[ap.index] = in + (bufOut * 0.5)
	ap.index = (ap.index + 1) % len(ap.buffer)
	return bufOut - in
}

// Reverb implements the Freeverb algorithm (simplified).
// Classic algorithm used in ViPER's reverb effect.
//
// Ref: Freeverb by Jezar at Dreampoint
type Reverb struct {
	RoomSize float32 // 0.0 - 1.0
	Damping  float32 // 0.0 - 1.0
	WetMix   float32 // 0.0 - 1.0

	combFilters    [8]CombFilter
	allpassFilters [4]AllpassFilter
	SampleRate     float32
}

// NewReverb creates a Freeverb-based reverb processor.
func NewReverb(sampleRate float32) *Reverb {
	// Freeverb standard delay line sizes (scaled for 44.1kHz)
	combSizes := [8]int{1116, 1188, 1277, 1356, 1422, 1491, 1557, 1617}
	allpassSizes := [4]int{556, 441, 341, 225}

	// Scale for sample rate (if not 44100)
	scale := sampleRate / 44100.0
	for i := range combSizes {
		combSizes[i] = int(float32(combSizes[i]) * scale)
	}
	for i := range allpassSizes {
		allpassSizes[i] = int(float32(allpassSizes[i]) * scale)
	}

	r := &Reverb{
		RoomSize:   0.5,
		Damping:    0.5,
		WetMix:     0.3,
		SampleRate: sampleRate,
	}

	for i := 0; i < 8; i++ {
		r.combFilters[i] = CombFilter{
			buffer:   make([]float32, combSizes[i]),
			feedback: 0.84,
			damping:  0.2,
		}
	}

	for i := 0; i < 4; i++ {
		r.allpassFilters[i] = AllpassFilter{
			buffer: make([]float32, allpassSizes[i]),
		}
	}

	return r
}

// ProcessSample processes a single sample through the reverb.
func (r *Reverb) ProcessSample(in float32) float32 {
	// Parallel comb filters
	combOut := float32(0)
	for i := 0; i < 8; i++ {
		combOut += r.combFilters[i].Process(in)
	}
	combOut /= 8 // Average

	// Series allpass filters (diffusion)
	allpassOut := combOut
	for i := 0; i < 4; i++ {
		allpassOut = r.allpassFilters[i].Process(allpassOut)
	}

	// Mix dry/wet
	return in*(1-r.WetMix) + allpassOut*r.WetMix
}

// UpdateParameters updates reverb parameters dynamically.
func (r *Reverb) UpdateParameters(roomSize, damping, wetMix float32) {
	r.RoomSize = roomSize
	r.Damping = damping
	r.WetMix = wetMix

	// Update comb filter parameters
	for i := 0; i < 8; i++ {
		r.combFilters[i].feedback = 0.7 + (roomSize * 0.28) // 0.7-0.98
		r.combFilters[i].damping = damping
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// UTILITY FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════

// Clamp constrains a value between min and max.
func Clamp(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// dBToLinear converts decibels to linear gain.
func dBToLinear(dB float32) float32 {
	return float32(math.Pow(10, float64(dB/20.0)))
}

// LinearTodB converts linear gain to decibels.
func LinearTodB(linear float32) float32 {
	if linear <= 0 {
		return -100.0 // Negative infinity
	}
	return 20.0 * float32(math.Log10(float64(linear)))
}
