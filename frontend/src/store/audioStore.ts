/**
 * audioStore.ts
 *
 * Central Zustand store for all DSP state.
 * Each setter mirrors a Go method exposed via Wails:
 *   window.go.main.App.<MethodName>(payload)
 *
 * Pattern:
 *   1. Optimistic local update
 *   2. Fire-and-forget Go call
 *   3. On error: rollback
 */
import { create } from "zustand";
import { subscribeWithSelector } from "zustand/middleware";

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetState(): Promise<DSPState>;
          ResetState(): Promise<DSPState>;
          CheckDriver(): Promise<boolean>;
          SetDriverStatus(status: boolean): Promise<boolean>;
          SetMode(mode: string): Promise<void>;
          SetEqBand(index: number, db: number): Promise<void>;
          SetFullEq(bands: number[]): Promise<void>;
          ResetEq(): Promise<void>;
          SetPower(on: boolean): Promise<void>;
          SetPreVolume(db: number): Promise<void>;
          SetPostVolume(db: number): Promise<void>;
          SetXBass(s: XBassState): Promise<void>;
          SetXBassMono(s: XBassMonoState): Promise<void>;
          SetXClarity(s: XClarityState): Promise<void>;
          SetSurround3D(s: Surround3DState): Promise<void>;
          SetReverb(p: ReverbParams): Promise<void>;
          SetReverbPanel(p: ReverbPanelState): Promise<void>;
          SetOutput(s: OutputState): Promise<void>;
          SetConvolver(s: ConvolverState): Promise<void>;
          SetDDC(s: DDCState): Promise<void>;
          SetAGC(s: AGCState): Promise<void>;
          SetDynamicSystem(s: DynamicSystemState): Promise<void>;
          SetSpectrumExtension(s: SpectrumExtensionState): Promise<void>;
          SetFieldSurround(s: FieldSurroundState): Promise<void>;
          SetDiffSurround(s: DiffSurroundState): Promise<void>;
          SetCure(s: CureState): Promise<void>;
          SetTubeSimulator(s: TubeSimulatorState): Promise<void>;
          SetAnalogX(s: AnalogXState): Promise<void>;
          SetFETCompressor(s: FETCompressorState): Promise<void>;
          SetSpeakerCorrection(s: SpeakerCorrectionState): Promise<void>;
          CommitDSPChanges(): Promise<void>;
          CommitDSPChangesAsync(): Promise<void>;
          SavePreset(name: string): Promise<void>;
          LoadPreset(name: string): Promise<DSPState>;
          ListPresets(): Promise<string[]>;
        };
      };
    };
  }
}

async function go<T>(fn: () => Promise<T>, onError?: () => void): Promise<T | null> {
  try {
    return await fn();
  } catch (e) {
    console.error("[Wails bridge]", e);
    if (onError) onError();
    return null;
  }
}

let commitTimer: number | undefined;

function scheduleCommit() {
  if (commitTimer) {
    clearTimeout(commitTimer);
  }
  commitTimer = window.setTimeout(() => {
    go(() => window.go.main.App.CommitDSPChangesAsync()).catch(() => {});
  }, 150);
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
  agc: { on: false, ratio: 1, volume: 1, maxScaler: 1 },
  dynamicSystem: {
    on: false,
    xCoeffsLow: 120,
    xCoeffsHigh: 120,
    yCoeffsLow: 200,
    yCoeffsHigh: 200,
    sideGainX: 0,
    sideGainY: 0,
    strength: 0,
  },
  spectrumExtension: { on: false, referenceFrequency: 7600, exciter: 0 },
  fieldSurround: { on: false, widening: 0, midImage: 0, depth: 0 },
  diffSurround: { on: false, delay: 0 },
  cure: { on: false, strengthPreset: 0 },
  tubeSimulator: { on: false },
  analogX: { on: false, mode: 0 },
  fetCompressor: {
    on: false,
    threshold: 0,
    ratio: 1,
    knee: 0,
    autoKnee: false,
    gain: 0,
    autoGain: false,
    attack: 0,
    autoAttack: false,
    release: 0,
    autoRelease: false,
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
  init(): Promise<void>;
  checkDriverStatus(): Promise<void>;
  setDriverStatus(status: boolean): void;
  setPower(on: boolean): void;
  setPreVol(db: number): void;
  setPostVol(db: number): void;
  setMode(mode: DSPState["mode"]): void;
  setEqBand(index: number, db: number): void;
  setFullEq(bands: number[]): void;
  resetEq(): void;
  setXBass(patch: Partial<XBassState>): void;
  setXBassMono(patch: Partial<XBassMonoState>): void;
  setXClarity(patch: Partial<XClarityState>): void;
  setSurround3D(patch: Partial<Surround3DState>): void;
  setReverb(patch: Partial<ReverbParams>): void;
  setReverbPanel(patch: Partial<ReverbPanelState>): void;
  setOutput(patch: Partial<OutputState>): void;
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
  refreshPresets(): Promise<void>;
}

export const useAudioStore = create<AudioStore>()(
  subscribeWithSelector((set, get) => {
    const patchModule = <K extends keyof DSPState>(
      key: K,
      patch: Partial<DSPState[K]>,
      persist: (next: DSPState[K]) => Promise<unknown>,
    ) => {
      const prev = get()[key] as DSPState[K];
      const next = { ...(prev as object), ...(patch as object) } as DSPState[K];
      set({ [key]: next } as Pick<AudioStore, K>);
      go(() => persist(next), () => {
        set({ [key]: prev } as Pick<AudioStore, K>);
      });
      scheduleCommit();
    };

    return {
      ...DEFAULT_STATE,
      ready: false,
      presets: [],
      isDriverInstalled: false,

      async init() {
        const [state, driverOk] = await Promise.all([
          go(() => window.go.main.App.GetState()),
          go(() => window.go.main.App.CheckDriver()),
        ]);
        if (state) {
          set({ ...state, isDriverInstalled: !!driverOk, ready: true });
        } else {
          set({ ready: true, isDriverInstalled: !!driverOk });
        }
        get().refreshPresets();
      },

      async checkDriverStatus() {
        const installed = await window.go.main.App.CheckDriver();
        set({ isDriverInstalled: installed });
      },

      setDriverStatus(status) {
        const prev = get().isDriverInstalled;
        set({ isDriverInstalled: status });
        go(() => window.go.main.App.SetDriverStatus(status), () => {
          set({ isDriverInstalled: prev });
        });
      },

      setPower(on) {
        const prev = get().master;
        const next = { ...prev, power: on };
        set({ master: next });
        go(() => window.go.main.App.SetPower(on), () => {
          set({ master: prev });
        });
        scheduleCommit();
      },

      setPreVol(db) {
        const prev = get().master;
        const next = { ...prev, preVol: db };
        set({ master: next });
        go(() => window.go.main.App.SetPreVolume(db), () => {
          set({ master: prev });
        });
        scheduleCommit();
      },

      setPostVol(db) {
        const prev = get().master;
        const next = { ...prev, postVol: db };
        set({ master: next });
        go(() => window.go.main.App.SetPostVolume(db), () => {
          set({ master: prev });
        });
        scheduleCommit();
      },

      setMode(mode) {
        const prev = get().mode;
        set({ mode });
        go(() => window.go.main.App.SetMode(mode), () => {
          set({ mode: prev });
        });
        scheduleCommit();
      },

      setEqBand(index, db) {
        const prevEq = get().equalizer;
        const nextEq = [...prevEq];
        nextEq[index] = db;
        set({ equalizer: nextEq });
        go(() => window.go.main.App.SetEqBand(index, db), () => {
          set({ equalizer: prevEq });
        });
        scheduleCommit();
      },

      resetEq() {
        const flatEq = Array(18).fill(0);
        const prevEq = get().equalizer;
        set({ equalizer: flatEq });
        go(() => window.go.main.App.ResetEq(), () => {
          set({ equalizer: prevEq });
        });
        scheduleCommit();
      },

      setFullEq(bands) {
        const prevEq = get().equalizer;
        set({ equalizer: bands });
        go(() => window.go.main.App.SetFullEq(bands), () => {
          set({ equalizer: prevEq });
        });
        scheduleCommit();
      },

      setXBass(patch) {
        patchModule("xBass", patch, (next) => window.go.main.App.SetXBass(next as XBassState));
      },

      setXBassMono(patch) {
        patchModule("xBassMono", patch, (next) => window.go.main.App.SetXBassMono(next as XBassMonoState));
      },

      setXClarity(patch) {
        patchModule("xClarity", patch, (next) => window.go.main.App.SetXClarity(next as XClarityState));
      },

      setSurround3D(patch) {
        patchModule("surround3D", patch, (next) => window.go.main.App.SetSurround3D(next as Surround3DState));
      },

      setReverb(patch) {
        patchModule("reverb", patch, (next) => window.go.main.App.SetReverb(next as ReverbParams));
      },

      setReverbPanel(patch) {
        patchModule("reverbPanel", patch, (next) => window.go.main.App.SetReverbPanel(next as ReverbPanelState));
      },

      setOutput(patch) {
        patchModule("output", patch, (next) => window.go.main.App.SetOutput(next as OutputState));
      },

      setConvolver(patch) {
        patchModule("convolver", patch, (next) => window.go.main.App.SetConvolver(next as ConvolverState));
      },

      setDDC(patch) {
        patchModule("ddc", patch, (next) => window.go.main.App.SetDDC(next as DDCState));
      },

      setAGC(patch) {
        patchModule("agc", patch, (next) => window.go.main.App.SetAGC(next as AGCState));
      },

      setDynamicSystem(patch) {
        patchModule("dynamicSystem", patch, (next) => window.go.main.App.SetDynamicSystem(next as DynamicSystemState));
      },

      setSpectrumExtension(patch) {
        patchModule("spectrumExtension", patch, (next) => window.go.main.App.SetSpectrumExtension(next as SpectrumExtensionState));
      },

      setFieldSurround(patch) {
        patchModule("fieldSurround", patch, (next) => window.go.main.App.SetFieldSurround(next as FieldSurroundState));
      },

      setDiffSurround(patch) {
        patchModule("diffSurround", patch, (next) => window.go.main.App.SetDiffSurround(next as DiffSurroundState));
      },

      setCure(patch) {
        patchModule("cure", patch, (next) => window.go.main.App.SetCure(next as CureState));
      },

      setTubeSimulator(patch) {
        patchModule("tubeSimulator", patch, (next) => window.go.main.App.SetTubeSimulator(next as TubeSimulatorState));
      },

      setAnalogX(patch) {
        patchModule("analogX", patch, (next) => window.go.main.App.SetAnalogX(next as AnalogXState));
      },

      setFETCompressor(patch) {
        patchModule("fetCompressor", patch, (next) => window.go.main.App.SetFETCompressor(next as FETCompressorState));
      },

      setSpeakerCorrection(patch) {
        patchModule("speakerCorrection", patch, (next) => window.go.main.App.SetSpeakerCorrection(next as SpeakerCorrectionState));
      },

      async savePreset(name) {
        await go(() => window.go.main.App.SavePreset(name));
        get().refreshPresets();
      },

      async loadPreset(name) {
        const state = await go(() => window.go.main.App.LoadPreset(name));
        if (state) {
          set({ ...state });
        }
      },

      async refreshPresets() {
        const list = await go(() => window.go.main.App.ListPresets());
        if (list) {
          set({ presets: list });
        }
      },
    };
  }),
);

useAudioStore.subscribe(
  (state) => state.isDriverInstalled,
  (isInstalled) => {
    if (!isInstalled) {
      useAudioStore.getState().setPower(false);
      console.error("CRITICAL: Driver de audio no detectado. Power OFF preventivo.");
    }
  },
);
