// APOSkel.h — simple COM APO skeleton types and GUIDs
#pragma once

#include <windows.h>
#include <unknwn.h>

// CLSID for the APO (matches DriverManager ViPER_CLSID)
static const GUID CLSID_ViperAPO =
    {0xDA2FB532, 0x3014, 0x4B93, {0xAD,0x05,0x21,0xB2,0xC6,0x20,0xF9,0xC2}};

// IID for the APO interface (matches DriverManager ViPER_IID)
static const GUID IID_IViperAPO =
    {0xFD7F2B29, 0x24D0, 0x4B5C, {0xB1,0x77,0x59,0x2C,0x39,0xF9,0xCA,0x10}};

// Minimal APO interface used by the shim / test harness.
// This is intentionally small: Initialize + Process.
struct IViperAPO : public IUnknown {
    virtual HRESULT STDMETHODCALLTYPE Initialize(UINT32 sampleRate, UINT32 channels) = 0;
    virtual HRESULT STDMETHODCALLTYPE Process(float* buffer, UINT32 frames, UINT32 channels) = 0;
    virtual HRESULT STDMETHODCALLTYPE ReleaseResources() = 0;
};

// C helper exported by the DLL to allow easy instantiation from test code
extern "C" __declspec(dllexport) HRESULT CreateAPOInstance(IViperAPO** ppOut);

// Lightweight direct process helper (for audio thread wiring convenience)
extern "C" __declspec(dllexport) int APO_ProcessBuffer(float* buffer, int frames, int channels);
