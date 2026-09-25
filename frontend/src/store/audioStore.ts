/**
 * audioStore.ts
 *
 * Central Zustand store for all DSP state.
 *
 * FIX (this pass): every action in this store used to call
 * `window.go.main.App.<Method>` — a hand-typed, invented API surface
 * (SetPreVolume, SetReverb, SetAGC, CommitDSPChangesAsync, CheckDriver,
 * ...) that never matched the real Go bindings. Two separate bugs:
 *
 *   1. Wrong namespace: the real App lives at `window.go.app.App`
 *      (Go package `app`), not `window.go.main.App`.
 *   2. Most of those methods were never implemented on the Go side at
 *      all — only Master (power/preVol/postVol), EQ, XBass, XClarity,
 *      and Surround3D have real per-field Set* methods today.
 *
 * Every call here now goes through the generated bindings in
 * ../wailsjs/go/app/App (typed, always in sync with the real Go
 * struct — regenerated on every `wails dev`/`wails build`) instead of
 * a hand-maintained global `window.go` interface that could silently
 * drift from reality again.
 *
 * Modules with no backend Set* method yet (reverb, reverbPanel,
 * convolver, ddc, agc, dynamicSystem, spectrumExtension,
 * fieldSurround, diffSurround, cure, tubeSimulator, analogX,
 * fetCompressor, speakerCorrection, output, xBassMono, mode) still
 * update local UI state so their panels stay interactive/previewable,
 * but deliberately do NOT call Go — there is nothing real to call.
 * GetState()/LoadPreset() DO hydrate these from the backend correctly
 * (DSPState carries all of them), so opening a saved preset shows the
 * right values; changing a slider just won't persist until each one
 * gets a real Set* method on App (see RESTRUCTURE_NOTES.md). Search
 * for "LOCAL-ONLY" below to find exactly which ones.
 */
import { create } from "zustand";
import { subscribeWithSelector } from "zustand/middleware";
import * as Go from "../wailsjs/go/app/App";

async function call<T>(fn: () => Promise<T>, onError?: () => void): Promise<T | null> {
  try {
    return await fn();
  } catch (e) {
    console.error("[Wails bridge]", e);
    if (onError) onError();
    return null;
  }
}

export interface MasterState {
  power: boolean;
  preVol: number;
  postVol: number;
}

export interface OutputState {
  pan: number;
  limiter: number;
}

export interface XBassState {
  on: boolean;
  speakerSize: number;
  level: number;
  mode: "Natural Bass" | "Pure Bass";
}

export interface XBassMonoState {
  on: boolean;
  speakerSize: number;
  level: number;
  mode: "Natural Bass" | "Pure Bass";
}

export interface XClarityState {
  on: boolean;
  level: number;
  mode: "Natural" | "OZone+" | "X-HiFi";
}

export interface Surround3DState {
  on: boolean;
  spaceSize: number;
  roomSize: string;
  imageSize: number;
}

export interface ReverbParams {
  on: boolean;
  roomSize: number;
  damping: number;
  density: number;
  bandwidth: number;
  decay: number;
  preDelay: number;
  earlyMix: number;
  wetMix: number;
}

export interface ReverbPanelState {
  on: boolean;
  roomSize: string;
  size: number;
  wetMix: number;
}

export interface ConvolverState {
  on: boolean;
  kernelPath: string;
  crossChannel: number;
}

export interface DDCState {
  on: boolean;
  coeffs44100: number[];
  coeffs48000: number[];
}

export interface AGCState {
  on: boolean;
  ratio: number;
  volume: number;
  maxScaler: number;
}

export interface DynamicSystemState {
  on: boolean;
  xCoeffsLow: number;
  xCoeffsHigh: number;
  yCoeffsLow: number;
  yCoeffsHigh: number;
  sideGainX: number;
  sideGainY: number;
  strength: number;
}

export interface SpectrumExtensionState {
  on: boolean;
  referenceFrequency: number;
  exciter: number;
}

export interface FieldSurroundState {
  on: boolean;
  widening: number;
  midImage: number;
  depth: number;
}

export interface DiffSurroundState {
  on: boolean;
  delay: number;
}

export interface CureState {
  on: boolean;
  strengthPreset: number;
}

export interface TubeSimulatorState {
  on: boolean;
}

export interface AnalogXState {
  on: boolean;
  mode: number;
}

export interface FETCompressorState {
  on: boolean;
  threshold: number;
  ratio: number;
  knee: number;
  autoKnee: boolean;
  gain: number;
  autoGain: boolean;
  attack: number;
  autoAttack: boolean;
  release: number;
  autoRelease: boolean;
  kneeMulti: number;
  maxAttack: number;
  maxRelease: number;
  crest: number;
  adapt: number;
  noClip: boolean;
}

export interface SpeakerCorrectionState {
  on: boolean;
}

export interface DSPState {
  master: MasterState;
  output: OutputState;
  xBass: XBassState;
  xBassMono: XBassMonoState;
  xClarity: XClarityState;
  surround3D: Surround3DState;
  reverb: ReverbParams;
  reverbPanel: ReverbPanelState;
  convolver: ConvolverState;
  ddc: DDCState;
  agc: AGCState;
  dynamicSystem: DynamicSystemState;
  spectrumExtension: SpectrumExtensionState;
  fieldSurround: FieldSurroundState;
  diffSurround: DiffSurroundState;
  cure: CureState;
  tubeSimulator: TubeSimulatorState;
  analogX: AnalogXState;
  fetCompressor: FETCompressorState;
  speakerCorrection: SpeakerCorrectionState;
  mode: "music" | "movie" | "freestyle";
  eqOn: boolean;
  equalizer: number[];
}

const DEFAULT_STATE: DSPState = {
  mode: "freestyle",
  eqOn: true,
  equalizer: Array(18).fill(0),
  master: { power: true, preVol: 0, postVol: 12.0 },
  output: { pan: 0, limiter: 1 },
  xBass: { on: true, speakerSize: 5, level: 0, mode: "Natural Bass" },
  xBassMono: { on: false, speakerSize: 5, level: 0, mode: "Natural Bass" },
  xClarity: { on: true, level: 0, mode: "X-HiFi" },
  surround3D: { on: true, spaceSize: 5, roomSize: "Smallest Room", imageSize: 2 },
  reverb: {
    on: true,
    roomSize: 500,
    damping: 1.03,
    density: 12.2,
    bandwidth: 44,
    decay: 13,
    preDelay: 0,
    earlyMix: 91,
    wetMix: 50,
  },
  reverbPanel: { on: false, roomSize: "Smallest Room", size: 40, wetMix: 50 },
  convolver: { on: false, kernelPath: "", crossChannel: 0 },
  ddc: { on: false, coeffs44100: [], coeffs48000: [] },
  agc: { on: false, ratio: 2, volume: 0, maxScaler: 3 },
  dynamicSystem: {
    on: false,
    xCoeffsLow: 0,
    xCoeffsHigh: 0,
    yCoeffsLow: 0,
    yCoeffsHigh: 0,
    sideGainX: 1,
    sideGainY: 1,
    strength: 0.5,
  },
  spectrumExtension: { on: false, referenceFrequency: 8000, exciter: 0.3 },
  fieldSurround: { on: false, widening: 0.3, midImage: 0.3, depth: 20 },
  diffSurround: { on: false, delay: 0.02 },
  cure: { on: false, strengthPreset: 1 },
  tubeSimulator: { on: false },
  analogX: { on: false, mode: 0 },
  fetCompressor: {
    on: false,
    threshold: -20,
    ratio: 4,
    knee: 2,
    autoKnee: true,
    gain: 0,
    autoGain: true,
    attack: 5,
    autoAttack: true,
    release: 100,
    autoRelease: true,
    kneeMulti: 1,
    maxAttack: 0,
    maxRelease: 0,
    crest: 0,
    adapt: 0,
    noClip: false,
  },
  speakerCorrection: { on: false },
};

interface AudioStore extends DSPState {
  ready: boolean;
  presets: string[];
  isDriverInstalled: boolean;
  isDriverAttached: boolean;
  isElevated: boolean;
  driverVersion: string;
  driverArchitecture: string;
  driverDllPath: string;
  audioEngineRunning: boolean;

  init(): Promise<void>;
  refreshAPOStatus(): Promise<void>;
  refreshAudioEngineStatus(): Promise<void>;
  installDriver(): Promise<void>;
  uninstallDriver(): Promise<void>;
  restartAudioEngine(): Promise<void>;

  setPower(on: boolean): void;
  setPreVol(db: number): void;
  setPostVol(db: number): void;

  setEqEnabled(on: boolean): void;
  setEqBand(index: number, db: number): void;
  setFullEq(bands: number[]): void;
  resetEq(): void;

  setXBass(patch: Partial<XBassState>): void;
  setXClarity(patch: Partial<XClarityState>): void;
  setSurround3D(patch: Partial<Surround3DState>): void;

  // LOCAL-ONLY — no Go Set* method exists for these yet. See the file
  // header comment. Kept as plain local setters (no `call()`, no
  // optimistic rollback — there is nothing to roll back from) so
  // these panels stay interactive without lying about being saved.
  setMode(mode: DSPState["mode"]): void;
  setOutput(patch: Partial<OutputState>): void;
  setXBassMono(patch: Partial<XBassMonoState>): void;
  setReverb(patch: Partial<ReverbParams>): void;
  setReverbPanel(patch: Partial<ReverbPanelState>): void;
  setConvolver(patch: Partial<ConvolverState>): void;
  setDDC(patch: Partial<DDCState>): void;
  setAGC(patch: Partial<AGCState>): void;
  setDynamicSystem(patch: Partial<DynamicSystemState>): void;
  setSpectrumExtension(patch: Partial<SpectrumExtensionState>): void;
  setFieldSurround(patch: Partial<FieldSurroundState>): void;
  setDiffSurround(patch: Partial<DiffSurroundState>): void;
  setCure(patch: Partial<CureState>): void;
  setTubeSimulator(patch: Partial<TubeSimulatorState>): void;
  setAnalogX(patch: Partial<AnalogXState>): void;
  setFETCompressor(patch: Partial<FETCompressorState>): void;
  setSpeakerCorrection(patch: Partial<SpeakerCorrectionState>): void;

  savePreset(name: string): Promise<void>;
  loadPreset(name: string): Promise<void>;
  deletePreset(name: string): Promise<void>;
  refreshPresets(): Promise<void>;
}

export const useAudioStore = create<AudioStore>()(
  subscribeWithSelector((set, get) => {
    // Persisted patch: optimistic local update, call Go, roll back on error.
    const patchModule = <K extends keyof DSPState>(
      key: K,
      patch: Partial<DSPState[K]>,
      persist: (next: DSPState[K]) => Promise<unknown>,
    ) => {
      const prev = get()[key] as DSPState[K];
      const next = { ...(prev as object), ...(patch as object) } as DSPState[K];
      set({ [key]: next } as Pick<AudioStore, K>);
      call(() => persist(next), () => {
        set({ [key]: prev } as Pick<AudioStore, K>);
      });
    };

    // LOCAL-ONLY patch: no Go call, nothing to roll back. See header.
    const patchLocal = <K extends keyof DSPState>(key: K, patch: Partial<DSPState[K]>) => {
      const prev = get()[key] as DSPState[K];
      const next = { ...(prev as object), ...(patch as object) } as DSPState[K];
      set({ [key]: next } as Pick<AudioStore, K>);
    };

    return {
      ...DEFAULT_STATE,
      ready: false,
      presets: [],
      isDriverInstalled: false,
      isDriverAttached: false,
      isElevated: false,
      driverVersion: "",
      driverArchitecture: "",
      driverDllPath: "",
      audioEngineRunning: false,

      async init() {
        const [state, apoStatus, elevated, engineStatus] = await Promise.all([
          call(() => Go.GetState()),
          call(() => Go.GetAPOStatus()),
          call(() => Go.IsElevated()),
          call(() => Go.GetAudioEngineStatus()),
        ]);
        if (state) {
          set({ ...(state as unknown as DSPState) });
        }
        set({
          ready: true,
          isDriverInstalled: !!apoStatus?.isInstalled,
          isDriverAttached: !!apoStatus?.isAttached,
          driverVersion: apoStatus?.version ?? "",
          driverArchitecture: apoStatus?.architecture ?? "",
          driverDllPath: apoStatus?.dllPath ?? "",
          isElevated: !!elevated,
          audioEngineRunning: !!engineStatus?.isRunning,
        });
        get().refreshPresets();
      },

      async refreshAPOStatus() {
        const status = await call(() => Go.GetAPOStatus());
        set({
          isDriverInstalled: !!status?.isInstalled,
          isDriverAttached: !!status?.isAttached,
          driverVersion: status?.version ?? "",
          driverArchitecture: status?.architecture ?? "",
          driverDllPath: status?.dllPath ?? "",
        });
      },

      async refreshAudioEngineStatus() {
        const status = await call(() => Go.GetAudioEngineStatus());
        set({ audioEngineRunning: !!status?.isRunning });
      },

      // InstallDriver on the Go side does the full real sequence:
      // register the APO CLSID with Windows, attach it to every render
      // device, then restart the audio engine so audiodg.exe actually
      // picks it up. Go also emits an "app:toast" event on
      // success/failure — AudioDSP.tsx already listens for that, so no
      // extra toast call is needed here.
      async installDriver() {
        await call(() => Go.InstallDriver());
        await get().refreshAPOStatus();
        await get().refreshAudioEngineStatus();
      },

      async uninstallDriver() {
        await call(() => Go.UninstallDriver());
        await get().refreshAPOStatus();
        await get().refreshAudioEngineStatus();
      },

      async restartAudioEngine() {
        await call(() => Go.RestartAudioEngine());
        await get().refreshAudioEngineStatus();
      },

      setPower(on) {
        const prev = get().master;
        set({ master: { ...prev, power: on } });
        call(() => Go.SetPower(on), () => set({ master: prev }));
      },

      setPreVol(db) {
        const prev = get().master;
        set({ master: { ...prev, preVol: db } });
        call(() => Go.SetPreVol(db), () => set({ master: prev }));
      },

      setPostVol(db) {
        const prev = get().master;
        set({ master: { ...prev, postVol: db } });
        call(() => Go.SetPostVol(db), () => set({ master: prev }));
      },

      setEqEnabled(on) {
        const prev = get().eqOn;
        set({ eqOn: on });
        call(() => Go.SetEqEnabled(on), () => set({ eqOn: prev }));
      },

      setEqBand(index, db) {
        const prevEq = get().equalizer;
        const nextEq = [...prevEq];
        nextEq[index] = db;
        set({ equalizer: nextEq });
        call(() => Go.SetEqBand(index, db), () => set({ equalizer: prevEq }));
      },

      resetEq() {
        const prevEq = get().equalizer;
        const flatEq = Array(18).fill(0);
        set({ equalizer: flatEq });
        call(() => Go.ResetEq(), () => set({ equalizer: prevEq }));
      },

      setFullEq(bands) {
        const prevEq = get().equalizer;
        set({ equalizer: bands });
        call(() => Go.SetFullEq(bands), () => set({ equalizer: prevEq }));
      },

      setXBass(patch) {
        patchModule("xBass", patch, (next) =>
          Go.SetXBass(next.on, next.speakerSize, next.level, next.mode),
        );
      },

      setXClarity(patch) {
        patchModule("xClarity", patch, (next) => Go.SetXClarity(next.on, next.level, next.mode));
      },

      setSurround3D(patch) {
        patchModule("surround3D", patch, (next) =>
          Go.SetSurround3D(next.on, next.spaceSize, next.roomSize, next.imageSize),
        );
      },

      // ── LOCAL-ONLY (no backend Set* method yet) ──────────────────
      setMode(mode) {
        set({ mode });
      },
      setOutput(patch) {
        patchLocal("output", patch);
      },
      setXBassMono(patch) {
        patchLocal("xBassMono", patch);
      },
      setReverb(patch) {
        patchLocal("reverb", patch);
      },
      setReverbPanel(patch) {
        patchLocal("reverbPanel", patch);
      },
      setConvolver(patch) {
        patchLocal("convolver", patch);
      },
      setDDC(patch) {
        patchLocal("ddc", patch);
      },
      setAGC(patch) {
        patchLocal("agc", patch);
      },
      setDynamicSystem(patch) {
        patchLocal("dynamicSystem", patch);
      },
      setSpectrumExtension(patch) {
        patchLocal("spectrumExtension", patch);
      },
      setFieldSurround(patch) {
        patchLocal("fieldSurround", patch);
      },
      setDiffSurround(patch) {
        patchLocal("diffSurround", patch);
      },
      setCure(patch) {
        patchLocal("cure", patch);
      },
      setTubeSimulator(patch) {
        patchLocal("tubeSimulator", patch);
      },
      setAnalogX(patch) {
        patchLocal("analogX", patch);
      },
      setFETCompressor(patch) {
        patchLocal("fetCompressor", patch);
      },
      setSpeakerCorrection(patch) {
        patchLocal("speakerCorrection", patch);
      },

      // ── Presets ───────────────────────────────────────────────────
      async savePreset(name) {
        await call(() => Go.SavePreset(name));
        get().refreshPresets();
      },

      async loadPreset(name) {
        const state = await call(() => Go.LoadPreset(name));
        if (state) {
          set({ ...(state as unknown as DSPState) });
        }
      },

      async deletePreset(name) {
        await call(() => Go.DeletePreset(name));
        get().refreshPresets();
      },

      async refreshPresets() {
        const list = await call(() => Go.ListPresets());
        if (list) {
          set({ presets: list });
        }
      },
    };
  }),
);
