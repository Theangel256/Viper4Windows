package utils

// Clamp restricts v to the inclusive range [lo, hi].
//
// This used to be reimplemented privately in at least two places
// (internal/services/dsp_manager_service.go's clamp/clampInt, and
// internal/dsp/protocol/dispatch.go's clampF/clampI) with identical
// logic under different names. This is now the one copy; the protocol
// package's clampF/clampI call into this instead of having their own
// bodies — see internal/dsp/protocol/dispatch.go.
func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ClampInt is Clamp for integers.
func ClampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
