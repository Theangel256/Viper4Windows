<p align="center">
   <a href="https://github.com/Theangel256/Viper4Windows/releases">
        <img width="256" height="256" alt="appicon" src="https://github.com/user-attachments/assets/ba1a0c3b-5d8b-458a-9e44-6fe0f76d5fcd" />
    <h1 align="center">Viper4Windows</h1>
   </a>
</p>

Viper4Windows is a Windows desktop app for driving the ViPER audio stack from a modern UI without losing the low-level control that makes the original ecosystem interesting in the first place.

The app uses a Go backend with Wails for the native shell and an Astro + React frontend for the interface. Under the hood it can talk to the audio stack in two ways:

- through the installed APO/driver path using shared memory
- through a direct `ViPERDSP.dll` bridge exported from the ViPERDSP submodule

That split is intentional. It makes day-to-day testing easier, keeps development flexible, and gives the project a clean path for working both with the installed driver and the rebuilt DSP engine.

## Current Status

This project is in early alpha.

What already works well:

- native desktop shell with Wails
- driver/APO install and detach helpers for audio endpoints
- shared-memory sync with the installed ViPER APO
- direct loading of `ViPERDSP.dll`
- persistent DSP state and JSON presets
- modern frontend state management for audio controls
- a growing set of backend tests around DSP state and dispatch behavior

What is still moving:

- polishing the UI flow and visual consistency
- expanding parity between shared-memory control and direct-DLL control
- packaging and first-run setup
- broader testing across different audio devices and Windows setups

## Features

### Core DSP Control

- master power, pre volume and post volume
- 18-band equalizer with full state sync
- output pan and limiter controls
- preset save and load support

### ViPER Modules

- XBass
- XBass Mono
- XClarity
- 3D Surround / VHE
- Reverb
- Convolver
- DDC
- AGC / Playback Gain
- Dynamic System
- Spectrum Extension
- Field Surround
- Diff Surround
- Cure
- Tube Simulator
- AnalogX
- FET Compressor
- Speaker Correction

### Driver Integration

- detect whether the APO is installed
- attach the APO to a selected render or capture endpoint
- attach to the default endpoint
- remove the APO from a device
- restart the Windows audio engine after install or uninstall actions

## **Screenshots**


![Main Window](https://github.com/user-attachments/assets/017c23e7-074a-473f-ab4a-5162ba8f92b5)


### Equalizer

![Equalizer](https://github.com/user-attachments/assets/e278ecc9-e011-47ad-8a42-288f5872f6e5)

### Advanced DSP

![Advanced DSP](https://github.com/user-attachments/assets/f0926dcd-487d-4948-b696-da6607cc32f4)

## Tech Stack

- Go for backend logic and Windows integration
- Wails v2 for the desktop app shell
- Astro + React for the frontend
- Zustand for frontend state
- Tailwind CSS for styling
- ViPERDSP as the rebuilt DSP engine submodule

## Development

### Requirements

- Go
- Wails CLI
- Bun
- Visual Studio Build Tools or Visual Studio with MSVC
- CMake

### Run The App

Install frontend dependencies:

```bash
cd frontend
bun install
```

Start the app in development mode:

```bash
cd ..
wails dev
```

### Build The DSP DLL

Build the ViPERDSP bridge DLL:

```powershell
cmake --build .\ViPERDSP --config Debug
```

For local development with `wails dev`, `ViPERDSP.dll` should be available next to the generated executable. In this repo the practical place is:

```text
build/bin/ViPERDSP.dll
```

The backend loads it from the executable directory at startup.

## Project Layout

```text
Viper4Windows/
├── frontend/                # Astro + React UI
├── ViPERDSP/                # DSP engine submodule
├── build/                   # Wails build output
├── app.go                   # App state and backend methods
├── driveManager.go          # Driver/APO installation helpers
├── dspmanager.go            # Shared memory + direct DLL DSP control
├── dsp_core.go              # DSP core model/types
├── main.go                  # Wails entry point
└── wails.json               # Wails config
```

## Notes

- The installed APO path and the direct DLL path are both useful. They solve different problems and the app currently supports both.
- The direct DLL path depends on the exported bridge functions from the ViPERDSP submodule.
- The APO/shared-memory path is the easiest way to test against an already installed driver setup.