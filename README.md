<p align="center">
   <a href="https://github.com/Theangel256/Viper4Windows/releases">
    <!-- <img width="256" height="256" src="" /> -->
    <h1 align="center">Viper4Windows</h1>
   </a>
</p>

**Viper4Windows** is an advanced Digital Signal Processing (DSP) workstation designed for Windows. By combining the power and efficiency of **Go** on the backend with an ultra-modern interface built using **Astro** and **React**, Viper4Windows offers granular control over your system's auditory experience.


## ✨ Key Features

* **High-Performance DSP Engine**: Real-time audio processing powered by the refurbished ViperFX core for modern architectures.
* **Adaptive Interface**: A "Glassmorphism" and "Light Mode" design optimized for both productivity and aesthetics.
* **Advanced Audio Modules**:
    * **XBass Dynamic**: Depth control and speaker size adjustment for powerful low-ends.
    * **XClarity**: Sharpness algorithm for crystal-clear vocals and high frequencies.
    * **3D Surround**: Acoustic space emulation ranging from small rooms to large auditoriums.
    * **Intelligent Reverberation**: Full control over density, decay, and pre-delay for environmental depth.
* **Listening Modes**: Specialized presets for Music, Movie, and a flexible "Freestyle" mode.


## 🛠️ Tech Stack

The project utilizes a cutting-edge hybrid architecture:

* **Backend**: [Go](https://go.dev/) (Golang) for system logic and audio driver communication.
* **Frontend**: [Astro](https://astro.build/) + [React](https://reactjs.org/) for a lightning-fast UI and reactive components.
* **Runtime**: [Wails v2](https://wails.io/) to package the web interface as a native desktop application.
* **Styling**: [Tailwind CSS](https://tailwindcss.com/) + [Framer Motion](https://www.framer.com/motion/) for fluid animations.
* **Package Manager**: [Bun](https://bun.sh/) for ultra-fast frontend installations and builds.


## 🚀 Installation & Development

### Prerequisites

* **Go** (v1.18+)
* **Wails CLI** (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
* **Bun** (Recommended)

### Environment Setup

1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/Theangel256/Viper4Windows.git](https://github.com/Theangel256/Viper4Windows.git)
    cd Viper4Windows
    ```

2.  **Install frontend dependencies:**
    ```bash
    cd frontend
    bun install
    ```

3.  **Run in development mode:**
    ```bash
    wails dev
    ```

## 📦 Compilation (Build)

To generate the final Windows executable (`.exe`):

1.  Ensure that `wails.json` correctly points to the Astro output directory (`frontend/dist`).
2.  Run the build command:
    ```bash
    wails build -clean
    ```

## 📂 Project Structure

```text
Viper4Windows/
├── frontend/             # Interface source code (Astro + React)
│   ├── src/components/   # UI Components
│   ├── src/store/        # State management (audioStore.ts)
│   └── dist/             # Static production build
├── app.go                # Main application logic (Go)
├── main.go               # Wails entry point
├── go.mod                # Go module definition
└── wails.json            # Wails build configuration
```