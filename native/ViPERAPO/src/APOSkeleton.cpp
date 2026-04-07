#include <windows.h>
#include <objbase.h>
#include <atomic>
#include <iostream>
#include "../include/APOSkel.h"
#include "../include/ViPERDSP.h"

// Simple global counters for server locks / objects
static std::atomic_long g_serverLocks{0};
static std::atomic_long g_serverObjects{0};

// ViperAPO COM object
class ViperAPO : public IViperAPO {
public:
    ViperAPO() : m_ref(1), m_sampleRate(48000), m_channels(2) {
        g_serverObjects.fetch_add(1);
        ViperDSP_Init(m_sampleRate, m_channels);
    }

    virtual ~ViperAPO() {
        g_serverObjects.fetch_sub(1);
    }

    // IUnknown
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** ppvObject) override {
        if (!ppvObject) return E_POINTER;
        *ppvObject = NULL;
        if (IsEqualIID(riid, IID_IUnknown) || IsEqualIID(riid, IID_IViperAPO)) {
            *ppvObject = static_cast<IViperAPO*>(this);
            AddRef();
            return S_OK;
        }
        return E_NOINTERFACE;
    }

    ULONG STDMETHODCALLTYPE AddRef(void) override {
        return (ULONG)InterlockedIncrement(&m_ref);
    }

    ULONG STDMETHODCALLTYPE Release(void) override {
        LONG r = InterlockedDecrement(&m_ref);
        if (r == 0) delete this;
        return (ULONG)r;
    }

    // IViperAPO
    HRESULT STDMETHODCALLTYPE Initialize(UINT32 sampleRate, UINT32 channels) override {
        m_sampleRate = sampleRate > 0 ? sampleRate : 48000;
        m_channels = channels > 0 ? channels : 2;
        ViperDSP_Init((int)m_sampleRate, (int)m_channels);
        return S_OK;
    }

    HRESULT STDMETHODCALLTYPE Process(float* buffer, UINT32 frames, UINT32 channels) override {
        if (!buffer || frames == 0 || channels == 0) return E_INVALIDARG;
        // Forward to the shared engine
        int r = ViperDSP_Process(buffer, (int)frames, (int)channels);
        return r == 0 ? S_OK : E_FAIL;
    }

    HRESULT STDMETHODCALLTYPE ReleaseResources() override {
        // No-op for now
        return S_OK;
    }

private:
    LONG m_ref;
    UINT32 m_sampleRate;
    UINT32 m_channels;
};

// Class factory
class ViperAPOClassFactory : public IClassFactory {
public:
    ViperAPOClassFactory() : m_ref(1) { }
    virtual ~ViperAPOClassFactory() { }

    // IUnknown
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** ppvObject) override {
        if (!ppvObject) return E_POINTER;
        *ppvObject = NULL;
        if (IsEqualIID(riid, IID_IUnknown) || IsEqualIID(riid, IID_IClassFactory)) {
            *ppvObject = static_cast<IClassFactory*>(this);
            AddRef();
            return S_OK;
        }
        return E_NOINTERFACE;
    }

    ULONG STDMETHODCALLTYPE AddRef(void) override {
        return (ULONG)InterlockedIncrement(&m_ref);
    }

    ULONG STDMETHODCALLTYPE Release(void) override {
        LONG r = InterlockedDecrement(&m_ref);
        if (r == 0) delete this;
        return (ULONG)r;
    }

    // IClassFactory
    HRESULT STDMETHODCALLTYPE CreateInstance(IUnknown* pUnkOuter, REFIID riid, void** ppvObject) override {
        if (!ppvObject) return E_POINTER;
        *ppvObject = NULL;
        if (pUnkOuter != NULL) return CLASS_E_NOAGGREGATION;
        ViperAPO* obj = new (std::nothrow) ViperAPO();
        if (!obj) return E_OUTOFMEMORY;
        HRESULT hr = obj->QueryInterface(riid, ppvObject);
        obj->Release(); // QueryInterface gave a ref or not
        return hr;
    }

    HRESULT STDMETHODCALLTYPE LockServer(BOOL fLock) override {
        if (fLock) g_serverLocks.fetch_add(1);
        else g_serverLocks.fetch_sub(1);
        return S_OK;
    }

private:
    LONG m_ref;
};

// Dll exports
extern "C" __declspec(dllexport) HRESULT __stdcall DllGetClassObject(REFCLSID rclsid, REFIID riid, LPVOID* ppv) {
    if (!ppv) return E_POINTER;
    *ppv = NULL;
    if (IsEqualCLSID(rclsid, CLSID_ViperAPO)) {
        ViperAPOClassFactory* factory = new (std::nothrow) ViperAPOClassFactory();
        if (!factory) return E_OUTOFMEMORY;
        HRESULT hr = factory->QueryInterface(riid, ppv);
        factory->Release();
        return hr;
    }
    return CLASS_E_CLASSNOTAVAILABLE;
}

extern "C" __declspec(dllexport) HRESULT __stdcall DllCanUnloadNow(void) {
    return (g_serverLocks.load() == 0 && g_serverObjects.load() == 0) ? S_OK : S_FALSE;
}

// C helper to create an instance without COM plumbing (convenience for tests)
extern "C" __declspec(dllexport) HRESULT CreateAPOInstance(IViperAPO** ppOut) {
    if (!ppOut) return E_POINTER;
    *ppOut = nullptr;
    ViperAPO* obj = new (std::nothrow) ViperAPO();
    if (!obj) return E_OUTOFMEMORY;
    HRESULT hr = obj->QueryInterface(IID_IViperAPO, (void**)ppOut);
    obj->Release();
    return hr;
}

// Direct audio-thread-friendly helper
extern "C" __declspec(dllexport) int APO_ProcessBuffer(float* buffer, int frames, int channels) {
    return ViperDSP_Process(buffer, frames, channels);
}
