#include <windows.h>
#include <iostream>

int main() {
    const wchar_t* name = L"ViPER4Windows_SharedMem";
    const size_t size = 8192;
    HANDLE hMap = CreateFileMappingW(INVALID_HANDLE_VALUE, NULL, PAGE_READWRITE, 0, (DWORD)size, name);
    if (!hMap) {
        std::cerr << "CreateFileMappingW failed: " << GetLastError() << std::endl;
        return 1;
    }
    LPVOID addr = MapViewOfFile(hMap, FILE_MAP_WRITE, 0, 0, size);
    if (!addr) {
        std::cerr << "MapViewOfFile failed: " << GetLastError() << std::endl;
        CloseHandle(hMap);
        return 2;
    }

    float* p = reinterpret_cast<float*>(addr);
    const int floats = (int)(size / sizeof(float));
    for (int i = 0; i < floats; ++i) p[i] = 0.0f;

    int idx = 0;
    p[idx++] = 1.0f; // Enabled
    p[idx++] = 1.0f; // PreVol
    p[idx++] = 1.0f; // PostVol
    p[idx++] = 1.0f; // EqEnabled
    for (int b = 0; b < 18; ++b) p[idx++] = 6.0f; // Eq bands +6dB

    p[idx++] = 1.0f; // BassEnabled
    p[idx++] = 0.0f; // BassMode
    p[idx++] = 60.0f; // BassSpkSize
    p[idx++] = 6.0f; // BassGain

    std::cout << "Wrote shared memory snapshot (" << idx << " floats written)\n";

    UnmapViewOfFile(addr);
    CloseHandle(hMap);
    return 0;
}
