// ViperAPO.h — public API for the Hydrogen_Inst.dll shim
#pragma once

#ifdef __cplusplus
extern "C" {
#endif

// Installs the APO on a specific device GUID (render/capture endpoint id)
// deviceId: UTF-8 string with the GUID folder name used under MMDevices\Audio\Render\{GUID}
// Returns 0 on success, non-zero on error.
__declspec(dllexport) int InstallOnDeviceA(const char* deviceId);
__declspec(dllexport) int UninstallFromDeviceA(const char* deviceId);

// SetEqualizer: pass an array of floats (18 bands expected)
__declspec(dllexport) int SetEqualizer(const float* bands, int count);

// Commit changes (hint to reload/flush settings). Return 0 on success.
__declspec(dllexport) int CommitChanges();

// Start/Stop an internal shared-memory monitor (prototype shim)
// pollMs: polling interval in milliseconds (0 uses 100ms)
__declspec(dllexport) int StartSharedMemMonitor(int pollMs);
__declspec(dllexport) int StopSharedMemMonitor();

#ifdef __cplusplus
}
#endif
