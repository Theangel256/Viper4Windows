ViPERAPO (Windows APO scaffolding)
=================================

What this is
------------
This folder contains a minimal scaffolding project to build a Windows DLL named
`Hydrogen_Inst.dll`. The DLL exposes a small set of helper functions and
provides `DllRegisterServer`/`DllUnregisterServer` registry helpers so you can
register the APO with the audio engine.

Important: this is a *scaffolding*, not a full APO implementation. To be used
by Windows Audio Engine the DLL must implement the APO COM interfaces (using
the Windows SDK / WDK). Use the files here as a starting point.

Build (recommended)
-------------------
Requirements:
- Windows with Visual Studio (MSVC) and CMake
- (optional) ViPERDSP sources placed at the repository root `..\..\ViPERDSP`

Example:

```powershell
mkdir build
cd build
cmake -G "Visual Studio 17 2022" -A x64 ..\native\ViPERAPO
cmake --build . --config Release
```

The produced DLL will be `Hydrogen_Inst.dll` in the build output folder.

Next steps to implement a real APO
----------------------------------
1. Implement the APO COM class that exposes the Windows APO interfaces
   (IAudioProcessingObject and related). See the Windows SDK and WDK docs.
2. Wire the APO to call into the ViPERDSP engine for audio processing.
   The `ViPERDSP` project (separate repo) can be added as a subdirectory
   and linked to this target.
3. Implement proper thread-safe parameter updates (shared memory, IPC, or
   the APO parameter APIs). The Go backend can continue to write shared
   memory; the APO must read it on the audio thread.
4. Thoroughly test with real audio playback and multiple devices.

Automated registration
----------------------
The provided `DllRegisterServer` creates the minimal registry keys under:

- `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\{CLSID}`
- `HKLM\SOFTWARE\Classes\CLSID\{CLSID}\InprocServer32`

However, to be fully integrated you must ensure the APO exposes the correct
APO interfaces and the `APOInterface0`/`IID` values match the implementation.

Security / privileges
---------------------
Registering the APO and modifying `HKLM` requires Administrator privileges.
Use caution and always back up the registry before making system-wide changes.

If you want, I can:
- extend the scaffolding with an APO COM skeleton implementing the
  `IAudioProcessingObject` methods (more code, requires WDK/headers), or
- implement a lightweight shim that writes/reads the shared memory layout
  used by the Go backend so you can prototype quickly without a full COM APO.
