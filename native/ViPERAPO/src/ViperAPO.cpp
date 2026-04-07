#include <windows.h>
#include <string>
#include <vector>
#include <iostream>
#include "../include/ViperAPO.h"
#include "../include/ViPERDSP.h"

// NOTE: This file is a lightweight scaffolding for a ViPER APO DLL.
// It provides registry-based registration helpers and simple exported
// functions so the rest of the app can call into a consistent API.
//
// Important: This is NOT a full APO implementation. To be a functioning
// Windows Audio Processing Object you must implement the APO COM interfaces
// (IAudioProcessingObject, etc.) using the Windows SDK/WDK. See README.

static const char* VIPER_CLSID = "{DA2FB532-3014-4B93-AD05-21B2C620F9C2}";

// Helper: get path to this DLL
static std::string GetModulePath()
{
    char buf[MAX_PATH];
    HMODULE hMod = NULL;
    // Portable way to obtain the current module handle from an address
    if (!GetModuleHandleExA(GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS,
                           reinterpret_cast<LPCSTR>(&GetModulePath), &hMod)) {
        return std::string();
    }
    DWORD n = GetModuleFileNameA(hMod, buf, MAX_PATH);
    if (n == 0) return std::string();
    return std::string(buf, n);
}

// DllMain (minimal)
BOOL APIENTRY DllMain(HMODULE hModule, DWORD  ul_reason_for_call, LPVOID lpReserved)
{
    switch (ul_reason_for_call)
    {
    case DLL_PROCESS_ATTACH:
        DisableThreadLibraryCalls(hModule);
        break;
    case DLL_PROCESS_DETACH:
        break;
    }
    return TRUE;
}

// DllRegisterServer: create basic AudioProcessingObjects + CLSID entries
extern "C" HRESULT __stdcall DllRegisterServer()
{
    std::string modulePath = GetModulePath();
    if (modulePath.empty()) return E_FAIL;

    // 1) HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\{CLSID}
    std::string apoKey = std::string("SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\AudioEngine\\AudioProcessingObjects\\") + VIPER_CLSID;
    HKEY hKey = nullptr;
    LONG r = RegCreateKeyExA(HKEY_LOCAL_MACHINE, apoKey.c_str(), 0, NULL, REG_OPTION_NON_VOLATILE, KEY_WRITE, NULL, &hKey, NULL);
    if (r != ERROR_SUCCESS) return HRESULT_FROM_WIN32(r);

    const char* friendly = "ViPER4Windows APO";
    RegSetValueExA(hKey, "FriendlyName", 0, REG_SZ, (const BYTE*)friendly, (DWORD)strlen(friendly) + 1);
    RegSetValueExA(hKey, "Library", 0, REG_SZ, (const BYTE*)modulePath.c_str(), (DWORD)modulePath.size() + 1);
    DWORD major = 1; RegSetValueExA(hKey, "MajorVersion", 0, REG_DWORD, (const BYTE*)&major, sizeof(major));
    DWORD minor = 0; RegSetValueExA(hKey, "MinorVersion", 0, REG_DWORD, (const BYTE*)&minor, sizeof(minor));
    DWORD flags = 13; RegSetValueExA(hKey, "Flags", 0, REG_DWORD, (const BYTE*)&flags, sizeof(flags));
    RegCloseKey(hKey);

    // 2) HKLM\SOFTWARE\Classes\CLSID\{CLSID}\InprocServer32 = modulePath, ThreadingModel = Both
    std::string clsidKey = std::string("SOFTWARE\\Classes\\CLSID\\") + VIPER_CLSID + "\\InprocServer32";
    HKEY hC = nullptr;
    r = RegCreateKeyExA(HKEY_LOCAL_MACHINE, clsidKey.c_str(), 0, NULL, REG_OPTION_NON_VOLATILE, KEY_WRITE, NULL, &hC, NULL);
    if (r != ERROR_SUCCESS) return HRESULT_FROM_WIN32(r);
    RegSetValueExA(hC, NULL, 0, REG_SZ, (const BYTE*)modulePath.c_str(), (DWORD)modulePath.size() + 1);
    const char* threading = "Both";
    RegSetValueExA(hC, "ThreadingModel", 0, REG_SZ, (const BYTE*)threading, (DWORD)strlen(threading) + 1);
    RegCloseKey(hC);

    return S_OK;
}

extern "C" HRESULT __stdcall DllUnregisterServer()
{
    // Remove the keys we created in DllRegisterServer. Use RegDeleteTree if available.
    std::string apoKey = std::string("SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\AudioEngine\\AudioProcessingObjects\\") + VIPER_CLSID;
    RegDeleteTreeA(HKEY_LOCAL_MACHINE, apoKey.c_str());

    std::string clsidPath = std::string("SOFTWARE\\Classes\\CLSID\\") + VIPER_CLSID;
    RegDeleteTreeA(HKEY_LOCAL_MACHINE, clsidPath.c_str());

    return S_OK;
}

// Exported helper: InstallOnDeviceA — creates FxProperties entries for a device
extern "C" __declspec(dllexport) int InstallOnDeviceA(const char* deviceId)
{
    if (!deviceId) return -1;
    std::string base = "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\MMDevices\\Audio\\Render\\";
    std::string fxPath = base + deviceId + "\\FxProperties";

    HKEY hk = nullptr;
    LONG r = RegCreateKeyExA(HKEY_LOCAL_MACHINE, fxPath.c_str(), 0, NULL, REG_OPTION_NON_VOLATILE, KEY_WRITE, NULL, &hk, NULL);
    if (r != ERROR_SUCCESS) return (int)r;

    const char* clsid = VIPER_CLSID;
    const char* keys[] = {
        "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},5",
        "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},6",
        "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},7",
    };
    for (const char* k : keys) {
        RegSetValueExA(hk, k, 0, REG_SZ, (const BYTE*)clsid, (DWORD)strlen(clsid) + 1);
    }

    RegSetValueExA(hk, "{b725f130-47ef-101a-a5f1-02608c9eebac},10", 0, REG_SZ, (const BYTE*)"ViPER4Windows APO", (DWORD)strlen("ViPER4Windows APO") + 1);

    RegCloseKey(hk);
    return 0;
}

extern "C" __declspec(dllexport) int UninstallFromDeviceA(const char* deviceId)
{
    if (!deviceId) return -1;
    std::string base = "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\MMDevices\\Audio\\Render\\";
    std::string fxPath = base + deviceId + "\\FxProperties";
    // Open key and delete known values
    HKEY hk = nullptr;
    LONG r = RegOpenKeyExA(HKEY_LOCAL_MACHINE, fxPath.c_str(), 0, KEY_SET_VALUE, &hk);
    if (r != ERROR_SUCCESS) return (int)r;

    const char* keys[] = {
        "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},5",
        "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},6",
        "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},7",
        "{b725f130-47ef-101a-a5f1-02608c9eebac},10",
    };
    for (const char* k : keys) {
        RegDeleteValueA(hk, k);
    }
    RegCloseKey(hk);
    return 0;
}

// Stub: SetEqualizer — for now just print values to debug output
extern "C" __declspec(dllexport) int SetEqualizer(const float* bands, int count)
{
    if (!bands || count <= 0) return -1;
    // Build a temporary params struct and update the internal DSP engine.
    VIPER_DSP_PARAMS_C params{};
    params.EqEnabled = 1.0f;
    int c = (count > 18) ? 18 : count;
    for (int i = 0; i < c; ++i) params.EqBands[i] = bands[i];
    ViperDSP_UpdateParams(&params);
    return 0;
}

// Stub: CommitChanges — hint for flushing state to the real APO
extern "C" __declspec(dllexport) int CommitChanges()
{
    // Ensure the DSP engine is initialized (no-op if already initialized)
    ViperDSP_Init(48000, 2);
    return 0;
}
