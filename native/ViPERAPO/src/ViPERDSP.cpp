#include "../include/ViPERDSP.h"
#include <cmath>
#include <vector>
#include <mutex>
#include <cstring>

// Minimal C++ port of a subset of the Go `dsp_core` functionality so the
// shim can exercise EQ and Bass processing when an external ViPERDSP is not
// available. This is not a full implementation, but it follows the same
// structure (biquad filters, 18-band EQ, simple bass modes) and is suitable
// for prototyping.

struct BiquadCoefficients {
    float B0, B1, B2;
    float A1, A2;
};

struct BiquadFilter {
    BiquadCoefficients Coefs{};
    float Z1{0.0f}, Z2{0.0f};

    inline float ProcessSample(float in) {
        float out = Coefs.B0 * in + Z1;
        Z1 = Coefs.B1 * in - Coefs.A1 * out + Z2;
        Z2 = Coefs.B2 * in - Coefs.A2 * out;
        return out;
    }

    inline void Reset() { Z1 = Z2 = 0.0f; }
};

static BiquadCoefficients CalculatePeakingEQ(float freq, float sampleRate, float gainDB, float q) {
    float A = std::pow(10.0f, gainDB/40.0f);
    float omega = 2.0f * static_cast<float>(M_PI) * freq / sampleRate;
    float sinOmega = std::sin(omega);
    float cosOmega = std::cos(omega);
    float alpha = sinOmega / (2.0f * q);

    float b0 = 1 + alpha * A;
    float b1 = -2 * cosOmega;
    float b2 = 1 - alpha * A;
    float a0 = 1 + alpha / A;
    float a1 = -2 * cosOmega;
    float a2 = 1 - alpha / A;

    BiquadCoefficients c;
    c.B0 = b0 / a0;
    c.B1 = b1 / a0;
    c.B2 = b2 / a0;
    c.A1 = a1 / a0;
    c.A2 = a2 / a0;
    return c;
}

static BiquadCoefficients CalculateLowShelf(float freq, float sampleRate, float gainDB, float q) {
    float A = std::pow(10.0f, gainDB/40.0f);
    float omega = 2.0f * static_cast<float>(M_PI) * freq / sampleRate;
    float sinOmega = std::sin(omega);
    float cosOmega = std::cos(omega);
    float alpha = sinOmega / (2.0f * q);
    float sqrtA = std::sqrt(A);

    float b0 = A * ((A + 1) - (A - 1) * cosOmega + 2 * sqrtA * alpha);
    float b1 = 2 * A * ((A - 1) - (A + 1) * cosOmega);
    float b2 = A * ((A + 1) - (A - 1) * cosOmega - 2 * sqrtA * alpha);
    float a0 = (A + 1) + (A - 1) * cosOmega + 2 * sqrtA * alpha;
    float a1 = -2 * ((A - 1) + (A + 1) * cosOmega);
    float a2 = (A + 1) + (A - 1) * cosOmega - 2 * sqrtA * alpha;

    BiquadCoefficients c;
    c.B0 = b0 / a0;
    c.B1 = b1 / a0;
    c.B2 = b2 / a0;
    c.A1 = a1 / a0;
    c.A2 = a2 / a0;
    return c;
}

static BiquadCoefficients CalculateLowPass(float freq, float sampleRate, float q) {
    float omega = 2.0f * static_cast<float>(M_PI) * freq / sampleRate;
    float sinOmega = std::sin(omega);
    float cosOmega = std::cos(omega);
    float alpha = sinOmega / (2.0f * q);

    float b0 = (1 - cosOmega) / 2;
    float b1 = 1 - cosOmega;
    float b2 = (1 - cosOmega) / 2;
    float a0 = 1 + alpha;
    float a1 = -2 * cosOmega;
    float a2 = 1 - alpha;

    BiquadCoefficients c;
    c.B0 = b0 / a0;
    c.B1 = b1 / a0;
    c.B2 = b2 / a0;
    c.A1 = a1 / a0;
    c.A2 = a2 / a0;
    return c;
}

// 18-band equalizer (simplified port)
class Equalizer18Band {
public:
    Equalizer18Band(float sampleRate) : sampleRate(sampleRate) {
        const float freqs[18] = {65, 92, 131, 185, 262, 370, 523, 740, 1047, 1480, 2093, 2960, 4186, 5920, 8372, 11840, 16744, 20000};
        for (int i = 0; i < 18; ++i) frequencies[i] = freqs[i];
        for (int i = 0; i < 18; ++i) {
            bands[i].Coefs = CalculatePeakingEQ(frequencies[i], sampleRate, 0.0f, 1.41f);
            bands[i].Reset();
        }
    }

    void SetBandGain(int band, float gainDB) {
        if (band < 0 || band >= 18) return;
        bands[band].Coefs = CalculatePeakingEQ(frequencies[band], sampleRate, gainDB, 1.41f);
    }

    void Reset() {
        for (int i = 0; i < 18; ++i) bands[i].Reset();
    }

    void ProcessBuffer(float* buffer, int frames, int channels) {
        if (!buffer || frames <= 0) return;
        if (channels == 1) {
            for (int i = 0; i < frames; ++i) {
                float s = buffer[i];
                for (int b = 0; b < 18; ++b) s = bands[b].ProcessSample(s);
                buffer[i] = s;
            }
        } else {
            for (int i = 0; i < frames; ++i) {
                int idx = i * channels;
                // Left
                float left = buffer[idx];
                for (int b = 0; b < 18; ++b) left = bands[b].ProcessSample(left);
                buffer[idx] = left;
                // Right (if present)
                if (channels > 1) {
                    float right = buffer[idx + 1];
                    for (int b = 0; b < 18; ++b) right = bands[b].ProcessSample(right);
                    buffer[idx + 1] = right;
                }
            }
        }
    }

private:
    BiquadFilter bands[18];
    float frequencies[18];
    float sampleRate;
};

class ViPERBass {
public:
    ViPERBass(float sampleRate): sampleRate(sampleRate) { UpdateFilters(); }

    void UpdateFilters() {
        float cutoff = SpeakerSize;
        if (Mode == 0) {
            shelf.Coefs = CalculateLowShelf(cutoff, sampleRate, Gain, 0.707f);
        } else {
            lpf.Coefs = CalculateLowPass(cutoff, sampleRate, 0.707f);
            harmonic.Coefs = CalculatePeakingEQ(cutoff/2.0f, sampleRate, Gain, 2.0f);
        }
    }

    float ProcessSample(float in) {
        if (Mode == 0) {
            return shelf.ProcessSample(in);
        } else {
            float low = lpf.ProcessSample(in);
            float distorted = low * low;
            if (low < 0) distorted = -distorted;
            float sub = harmonic.ProcessSample(distorted);
            return in + sub * 0.5f;
        }
    }

    int Mode{0};
    float SpeakerSize{60.0f};
    float Gain{0.0f};

private:
    float sampleRate{44100.0f};
    BiquadFilter lpf, shelf, harmonic;
};

// Engine globals
static std::mutex g_engine_lock;
static Equalizer18Band* g_eq = nullptr;
static ViPERBass* g_bass = nullptr;
static VIPER_DSP_PARAMS_C g_params{};
static int g_sampleRate = 48000;
static int g_channels = 2;

extern "C" __declspec(dllexport) int ViperDSP_Init(int sampleRate, int channels) {
    std::lock_guard<std::mutex> lk(g_engine_lock);
    g_sampleRate = sampleRate > 0 ? sampleRate : 48000;
    g_channels = channels > 0 ? channels : 2;
    if (!g_eq) g_eq = new Equalizer18Band((float)g_sampleRate);
    if (!g_bass) g_bass = new ViPERBass((float)g_sampleRate);
    return 0;
}

extern "C" __declspec(dllexport) int ViperDSP_UpdateParams(const VIPER_DSP_PARAMS_C* params) {
    if (!params) return -1;
    std::lock_guard<std::mutex> lk(g_engine_lock);
    // copy snapshot
    std::memcpy(&g_params, params, sizeof(VIPER_DSP_PARAMS_C));

    // EQ
    if (g_eq && g_params.EqEnabled > 0.5f) {
        for (int b = 0; b < 18; ++b) g_eq->SetBandGain(b, g_params.EqBands[b]);
    }

    // Bass
    if (g_bass) {
        g_bass->Mode = static_cast<int>(g_params.BassMode);
        g_bass->SpeakerSize = g_params.BassSpkSize > 0 ? g_params.BassSpkSize : 60.0f;
        g_bass->Gain = g_params.BassGain;
        g_bass->UpdateFilters();
    }

    return 0;
}

extern "C" __declspec(dllexport) int ViperDSP_Process(float* buffer, int frames, int channels) {
    if (!buffer || frames <= 0 || channels <= 0) return -1;
    std::lock_guard<std::mutex> lk(g_engine_lock);
    // Apply pre volume
    float pre = g_params.PreVol;
    float post = g_params.PostVol;

    int total = frames * channels;
    for (int i = 0; i < total; ++i) buffer[i] *= pre;

    // EQ
    if (g_eq && g_params.EqEnabled > 0.5f) {
        g_eq->ProcessBuffer(buffer, frames, channels);
    }

    // Bass (per-sample)
    if (g_bass && g_params.BassEnabled > 0.5f) {
        for (int f = 0; f < frames; ++f) {
            int idx = f * channels;
            for (int c = 0; c < channels; ++c) {
                float s = buffer[idx + c];
                s = g_bass->ProcessSample(s);
                buffer[idx + c] = s;
            }
        }
    }

    // Post volume
    for (int i = 0; i < total; ++i) buffer[i] *= post;

    return 0;
}
