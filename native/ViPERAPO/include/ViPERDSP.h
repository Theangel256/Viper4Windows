// ViPERDSP.h — minimal C API + param struct for the local DSP engine
#pragma once

#include <cstddef>

#ifdef __cplusplus
extern "C" {
#endif

// Shared mem size and struct must match Go side
static constexpr size_t SHARED_MEM_SIZE = 8192;

// C struct that mirrors VIPER_DSP_PARAMS in Go (packed as floats)
typedef struct VIPER_DSP_PARAMS_C {
    float Enabled;
    float PreVol;
    float PostVol;

    float EqEnabled;
    float EqBands[18];

    float BassEnabled;
    float BassMode;
    float BassSpkSize;
    float BassGain;

    float ClarityEnabled;
    float ClarityMode;
    float ClarityGain;

    float SurroundEnabled;
    float SurroundSize;

    float ConvolverEnabled;

    float ReverbEnabled;
    float ReverbRoom;
    float ReverbDamp;
    float ReverbMix;

    float CureTechEnabled;
    float CureTechLevel;

    float AnalogXEnabled;
    float AnalogXMode;

    float SpeakerOptEnabled;
    float SpeakerOptMode;
    float SpeakerOptGain;

    float LimiterEnabled;
    float LimiterThreshold;

    float _padding[2003];
} VIPER_DSP_PARAMS_C;

static_assert(sizeof(VIPER_DSP_PARAMS_C) == SHARED_MEM_SIZE, "Shared struct size mismatch");

// Simple C API for the DSP engine used by the shim
// Initialize with sample rate and channel count. Returns 0 on success.
__declspec(dllexport) int ViperDSP_Init(int sampleRate, int channels);

// Update all DSP parameters from a VIPER_DSP_PARAMS_C snapshot.
__declspec(dllexport) int ViperDSP_UpdateParams(const VIPER_DSP_PARAMS_C* params);

// Process an interleaved float buffer in-place. `frames` is number of frames,
// `channels` is channels per frame. Returns 0 on success.
__declspec(dllexport) int ViperDSP_Process(float* buffer, int frames, int channels);

#ifdef __cplusplus
}
#endif
