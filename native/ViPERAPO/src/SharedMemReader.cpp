#include <windows.h>
#include <iostream>
#include <thread>
#include <atomic>
#include <vector>
#include <chrono>
#include <cstring>
#include "../include/ViPERDSP.h"

// Shared mem name & size must match Go side
static constexpr const wchar_t* SHARED_MEM_NAME = L"Global\\ViPER4Windows_SharedMem";

// Monitor state
static std::atomic_bool g_running{false};
static std::thread g_thread;

// Utility: pretty-print small summary of params
static void DumpSummary(const VIPER_DSP_PARAMS_C* p) {
    if (!p) return;
    std::cout << "[ViperSHIM] Enabled=" << p->Enabled
              << " PreVol=" << p->PreVol
              << " PostVol=" << p->PostVol << "\n";
    std::cout << "  EQ[0..5]: ";
    for (int i = 0; i < 6; ++i) std::cout << p->EqBands[i] << " ";
    std::cout << "\n";
    std::cout << "  Bass: enabled=" << p->BassEnabled << " gain=" << p->BassGain << " spksize=" << p->BassSpkSize << " mode=" << p->BassMode << "\n";
}

// Polling loop: open mapping, read & detect changes
static void MonitorLoop(int pollMs) {
    std::vector<char> prev(SHARED_MEM_SIZE);

    while (g_running.load()) {
        HANDLE hMap = OpenFileMappingW(FILE_MAP_READ, FALSE, SHARED_MEM_NAME);
        if (!hMap) {
            // Try without the Global\ prefix as a fallback for interactive sessions
            hMap = OpenFileMappingW(FILE_MAP_READ, FALSE, L"ViPER4Windows_SharedMem");
        }
        if (!hMap) {
            // Not present yet — wait and retry
            std::this_thread::sleep_for(std::chrono::milliseconds(500));
            continue;
        }

        LPVOID addr = MapViewOfFile(hMap, FILE_MAP_READ, 0, 0, 0);
        if (!addr) {
            CloseHandle(hMap);
            std::this_thread::sleep_for(std::chrono::milliseconds(500));
            continue;
        }

        // initial read
        memcpy(prev.data(), addr, SHARED_MEM_SIZE);
        const VIPER_DSP_PARAMS_C* params = reinterpret_cast<const VIPER_DSP_PARAMS_C*>(addr);
        DumpSummary(params);

        // Initialize the internal DSP engine and push initial params
        ViperDSP_Init(48000, 2);
        ViperDSP_UpdateParams(params);

        while (g_running.load()) {
            std::this_thread::sleep_for(std::chrono::milliseconds(pollMs));
            // Copy current snapshot
            std::vector<char> cur(SHARED_MEM_SIZE);
            memcpy(cur.data(), addr, SHARED_MEM_SIZE);
            if (memcmp(cur.data(), prev.data(), SHARED_MEM_SIZE) != 0) {
                // Changes detected
                const VIPER_DSP_PARAMS_C* pcur = reinterpret_cast<const VIPER_DSP_PARAMS_C*>(cur.data());
                DumpSummary(pcur);
                // Update engine parameters so effects are applied
                ViperDSP_UpdateParams(pcur);
                prev.swap(cur);
            }
        }

        UnmapViewOfFile(addr);
        CloseHandle(hMap);
    }
}

// Start/Stop API (callable from monitor or DLL export)
extern "C" __declspec(dllexport) int StartSharedMemMonitor(int pollMs) {
    if (g_running.load()) return 0;
    g_running.store(true);
    g_thread = std::thread(MonitorLoop, pollMs > 0 ? pollMs : 100);
    return 0;
}

extern "C" __declspec(dllexport) int StopSharedMemMonitor() {
    if (!g_running.load()) return 0;
    g_running.store(false);
    if (g_thread.joinable()) g_thread.join();
    return 0;
}

// Helper for the console monitor to do a blocking run
int RunMonitorBlocking(int pollMs) {
    StartSharedMemMonitor(pollMs);
    std::cout << "Press Ctrl-C to stop...\n";
    while (g_running.load()) std::this_thread::sleep_for(std::chrono::seconds(1));
    StopSharedMemMonitor();
    return 0;
}
