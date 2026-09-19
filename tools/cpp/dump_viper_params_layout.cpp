// tools/cpp/dump_viper_params_layout.cpp
//
// WHAT THIS IS FOR
// -----------------
// The ViPERDSP submodule currently vendored in this repo does NOT
// define a typed C++ struct for its parameters — only #define PARAM_*
// dispatch codes (see ViPERDSP/include/ViPERParams.h). Some upstream
// forks (likelikeslike/ViPERDSP, per its own release notes) have since
// moved to a typed struct with fields like bass/eq/convolver/etc.
//
// If and when you update the submodule to a commit that ships that
// struct, the ONE thing you must never do is take an AI's word (mine,
// ChatGPT's, anyone's) for what the byte offsets/sizes are. C++
// struct layout depends on the compiler, target, padding, and pragma
// pack state — the only source of truth is offsetof()/sizeof()
// evaluated by the actual compiler that builds ViPERDSP.dll, on the
// actual header you vendored.
//
// This file does exactly that and nothing more. It prints JSON you
// can feed straight into a Go code generator for internal/dsp/protocol.
//
// HOW TO USE IT
// -------------
// 1. Update the ViPERDSP submodule.
// 2. Open the new ViPERParams.h and copy its real member names into
//    the FIELDS(...) macro list below, replacing the worked
//    `DemoParams` example (which is NOT ViPERParams — it's a
//    self-contained demonstration you can compile and run right now,
//    before touching the submodule, to see the tool actually works).
// 3. #include the real header instead of defining DemoParams by hand.
// 4. Rebuild and run: it will print real offsets/sizes as JSON.
//
// Build (no external deps, just a C++17 compiler):
//   g++ -std=c++17 -O0 dump_viper_params_layout.cpp -o dump_layout
//   ./dump_layout > viper_params_layout.json
//
// Do NOT enable struct packing pragmas here that ViPERDSP.dll itself
// doesn't also use for this struct — the whole point is to match the
// real binary, not to produce a smaller one.

#include <cstddef>
#include <cstdint>
#include <iostream>
#include <string>
//#include "ViPERParams.h"  // <-- swap in the real ViPERParams.h here once you update the submodule

// ─────────────────────────────────────────────────────────────────
// DEMO STRUCT — delete this block once you swap in the real header.
// output today, with zero claims about ViPERDSP's actual layout.
// ─────────────────────────────────────────────────────────────────
struct DemoBassParams {
    bool enable;
    int32_t mode;
    uint32_t frequency;
    float gain;
};

struct DemoParams {
    bool masterEnable;
    float masterGain;
    DemoBassParams bass;
    float eqBandLevels[31];
};
// ─────────────────────────────────────────────────────────────────

// A tiny hand-rolled JSON emitter — deliberately dependency-free so
// this compiles with nothing but a C++17 toolchain on a bare CI
// runner or your own machine, no vcpkg/conan/CMake package required.
struct FieldPrinter {
    bool first = true;
    void field(const char *name, std::size_t offset, std::size_t size) {
        if (!first) std::cout << ",\n";
        first = false;
        std::cout << "    { \"name\": \"" << name << "\", "
                   << "\"offset\": " << offset << ", "
                   << "\"size\": " << size << " }";
    }
};

int main() {
    std::cout << "{\n";
    std::cout << "  \"struct\": \"DemoParams\",\n";
    std::cout << "  \"sizeofStruct\": " << sizeof(DemoParams) << ",\n";
    std::cout << "  \"fields\": [\n";

    FieldPrinter p;
    // NOTE: replace this hand-written list with one entry per real
    // top-level member of viper::ViPERParams once you've swapped in
    // the real header. offsetof() requires a standard-layout type —
    // if the real struct has constructors/virtuals, you may need
    // reinterpret_cast-based offset computation instead; flag that
    // in a comment here if you hit it.
    p.field("masterEnable", offsetof(DemoParams, masterEnable), sizeof(DemoParams::masterEnable));
    p.field("masterGain", offsetof(DemoParams, masterGain), sizeof(DemoParams::masterGain));
    p.field("bass", offsetof(DemoParams, bass), sizeof(DemoParams::bass));
    p.field("eqBandLevels", offsetof(DemoParams, eqBandLevels), sizeof(DemoParams::eqBandLevels));

    std::cout << "\n  ]\n";
    std::cout << "}\n";

    // Also dump the nested DemoBassParams block on its own, since
    // that's the granularity your Go encoder actually needs for each
    // effect (top-level struct offset + per-block internal layout).
    std::cerr << "// DemoBassParams (nested block) — same idea, one level down:\n";
    std::cerr << "// sizeof(DemoBassParams) = " << sizeof(DemoBassParams) << "\n";
    std::cerr << "// offsetof(enable)    = " << offsetof(DemoBassParams, enable) << "\n";
    std::cerr << "// offsetof(mode)      = " << offsetof(DemoBassParams, mode) << "\n";
    std::cerr << "// offsetof(frequency) = " << offsetof(DemoBassParams, frequency) << "\n";
    std::cerr << "// offsetof(gain)      = " << offsetof(DemoBassParams, gain) << "\n";

    return 0;
}
