/**
 * audioStore.ts
 *
 * Central Zustand store for all DSP state.
 * Each setter mirrors a Go method exposed via Wails:
 *   window.go.main.App.<MethodName>(payload)
 *
 * Pattern:
 *   1. Optimistic local update  (instant UI response)
 *   2. Fire-and-forget Go call  (hardware / native DSP layer)
 *   3. On error: revert + surface toast (future)
 */
import { create } from "zustand";
import { subscribeWithSelector } from 'zustand/middleware';

// ── Wails runtime helper ──────────────────────────────────────────────────────

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetState():                             Promise<DSPState>;
          ResetState():                           Promise<DSPState>;
          CheckDriver():                          Promise<boolean>;
          SetDriverStatus(status: boolean):       Promise<boolean>;
          SetMode(mode: string):                  Promise<void>;
          SetEqBand(index: number, db: number):   Promise<void>;
          SetFullEq(bands: number[]):             Promise<void>;
          ResetEq():                              Promise<void>;
          SetPower(on: boolean):                  Promise<void>;
          SetPreVolume(db: number):               Promise<void>;
          SetPostVolume(db: number):              Promise<void>;
          SetXBass(s: XBassState):                Promise<void>;
          SetXClarity(s: XClarityState):          Promise<void>;
          SetSurround3D(s: Surround3DState):      Promise<void>;
          SetReverb(p: ReverbParams):             Promise<void>;
          SetReverbPanel(p: ReverbPanelState):    Promise<void>;
          SavePreset(name: string):               Promise<void>;
          LoadPreset(name: string):               Promise<DSPState>;
          ListPresets():                          Promise<string[]>;
        };
      };
    };
  }
}

/** Safely call a Wails Go method. Returns false on failure. */
async function go<T>(fn: () => Promise<T>, onError?: () => void): Promise<T | null> {
  try {
    return await fn();
  } catch (e) {
    console.error("[Wails bridge]", e);
    if (onError) onError();
    return null;
  }
}

// ── State types ───────────────────────────────────────────────────────────────

export interface MasterState {
  power:   boolean;
  preVol:  number;
  postVol: number;
}

export interface XBassState {
  on:          boolean;
  speakerSize: number;
  level:       number;
  mode:        "Natural Bass" | "Pure Bass";
}

export interface XClarityState {
  on:    boolean;
  level: number;
  mode:  "Natural" | "OZone+" | "X-HiFi";
}

export interface Surround3DState {
  on:        boolean;
  spaceSize: number;
  roomSize:  string;
  imageSize: number;
}

export interface ReverbParams {
  on:        boolean;
  roomSize:  number;
  damping:   number;
  density:   number;
  bandwidth: number;
  decay:     number;
  preDelay:  number;
  earlyMix:  number;
  wetMix:    number;
}

export interface ReverbPanelState {
  on:       boolean;
  roomSize: string;
  size:     number;
  wetMix:   number;
}

export interface DSPState {
  master:      MasterState;
  xBass:       XBassState;
  xClarity:    XClarityState;
  surround3D:  Surround3DState;
  reverb:      ReverbParams;
  reverbPanel: ReverbPanelState;
  mode:        "music" | "movie" | "freestyle";
}

// ── Default state (mirrors app.go defaultState) ────────────────────────────

const DEFAULT_STATE: DSPState = {
  mode: "freestyle",
  master:     { power: false, preVol: 0, postVol: 12.00 },
  xBass:      { on: true, speakerSize: 5, level: 0, mode: "Natural Bass" },
  xClarity:   { on: true, level: 0, mode: "X-HiFi" },
  surround3D: { on: false, spaceSize: 5, roomSize: "Smallest Room", imageSize: 2 },
  reverb: {
    on: false, roomSize: 500, damping: 1.03, density: 12.2,
    bandwidth: 44, decay: 13, preDelay: 0, earlyMix: 91, wetMix: 50,
  },
  reverbPanel: { on: false, roomSize: "Smallest Room", size: 40, wetMix: 50 },
};

// ── Store ─────────────────────────────────────────────────────────────────────

interface AudioStore extends DSPState {
  /** True once Go state has been loaded */
  ready: boolean;
  presets: string[];
  isDriverInstalled: boolean;
  equalizer: number[];
  setEqBand: (index: number, db: number) => void;
  resetEq: () => void;
  checkDriverStatus: () => Promise<void>;

  /** Hydrate store from Go on app start */
  init(): Promise<void>;

  // Master
  setDriverStatus(status: boolean): void;
  setPower(on: boolean): void;
  setPreVol(db: number): void;
  setPostVol(db: number): void;

  // Mode
  setMode(mode: DSPState["mode"]): void;

  // Modules
  setXBass(patch: Partial<XBassState>): void;
  setXClarity(patch: Partial<XClarityState>): void;
  setSurround3D(patch: Partial<Surround3DState>): void;
  setReverb(patch: Partial<ReverbParams>): void;
  setReverbPanel(patch: Partial<ReverbPanelState>): void;

  // Presets
  savePreset(name: string): Promise<void>;
  loadPreset(name: string): Promise<void>;
  refreshPresets(): Promise<void>;
}

export const useAudioStore = create<AudioStore>()(
  subscribeWithSelector((set, get) => ({
  ...DEFAULT_STATE,
  ready: false,
  presets: [],
  isDriverInstalled: false,
  equalizer: Array(18).fill(0),

  // ── Init ──────────────────────────────────────────────────────────────────

  async init() {
    const [state, driverOk] = await Promise.all([
      go(() => window.go.main.App.GetState()),
      go(() => window.go.main.App.CheckDriver())
    ]);
    if (state) {
      set({ ...state, isDriverInstalled: !!driverOk, ready: true });
    } else {
      // Wails not available (browser dev mode) — use defaults
      set({ ready: true, isDriverInstalled: !!driverOk });
    }
    get().refreshPresets();
  },
  
  // ── Driver ────────────────────────────────────────────────────────────────
  setDriverStatus(status) {
    set((s) => ({ isDriverInstalled: status }));
    go(() => window.go.main.App.SetDriverStatus(status));
  },
  checkDriverStatus: async () => {
    const installed = await window.go.main.App.CheckDriver();
    set({ isDriverInstalled: installed });
  },

  // ── Master ────────────────────────────────────────────────────────────────

  setPower(on) {
    set((s) => ({ master: { ...s.master, power: on } }));
    go(() => window.go.main.App.SetPower(on));
  },

  setPreVol(db) {
    set((s) => ({ master: { ...s.master, preVol: db } }));
    go(() => window.go.main.App.SetPreVolume(db));
  },

  setPostVol(db) {
    set((s) => ({ master: { ...s.master, postVol: db } }));
    go(() => window.go.main.App.SetPostVolume(db));
  },

  // ── Mode ──────────────────────────────────────────────────────────────────

  setMode(mode) {
    set({ mode });
    go(() => window.go.main.App.SetMode(mode));
  },

    // ── Eq ──────────────────────────────────────────────────────────────────
  setEqBand(index, db) {
    set((s) => {
      const prevEq = [...s.equalizer];
      const nextEq = [...s.equalizer];
      nextEq[index] = db;
      
      // Enviamos el cambio a Go de forma optimista
      go(() => window.go.main.App.SetEqBand(index, db), () => {
        // Rollback on error
        set({ equalizer: prevEq });
      });
      
      return { equalizer: nextEq };
    });
  },
  resetEq() {
    const flatEq = Array(18).fill(0);
    set({ equalizer: flatEq });
    go(() => window.go.main.App.ResetEq());
  },

  // ── XBass ─────────────────────────────────────────────────────────────────

  setXBass(patch) {
    set((s) => {
      const next = { ...s.xBass, ...patch };
      go(() => window.go.main.App.SetXBass(next));
      return { xBass: next };
    });
  },

  // ── XClarity ──────────────────────────────────────────────────────────────

  setXClarity(patch) {
    set((s) => {
      const next = { ...s.xClarity, ...patch };
      go(() => window.go.main.App.SetXClarity(next));
      return { xClarity: next };
    });
  },

  // ── Surround 3D ───────────────────────────────────────────────────────────

  setSurround3D(patch) {
    set((s) => {
      const next = { ...s.surround3D, ...patch };
      go(() => window.go.main.App.SetSurround3D(next));
      return { surround3D: next };
    });
  },

  // ── Reverb ────────────────────────────────────────────────────────────────

  setReverb(patch) {
    set((s) => {
      const next = { ...s.reverb, ...patch };
      go(() => window.go.main.App.SetReverb(next));
      return { reverb: next };
    });
  },

  setReverbPanel(patch) {
    set((s) => {
      const next = { ...s.reverbPanel, ...patch };
      go(() => window.go.main.App.SetReverbPanel(next));
      return { reverbPanel: next };
    });
  },

  // ── Presets ───────────────────────────────────────────────────────────────

  async savePreset(name) {
    await go(() => window.go.main.App.SavePreset(name));
    get().refreshPresets();
  },

  async loadPreset(name) {
    const state = await go(() => window.go.main.App.LoadPreset(name));
    if (state) set({ ...state });
  },

  async refreshPresets() {
    const list = await go(() => window.go.main.App.ListPresets());
    if (list) set({ presets: list });
  },
}))
);

useAudioStore.subscribe(
  (state) => state.isDriverInstalled, // Argumento 1: Selector
  (isInstalled) => {                 // Argumento 2: Callback
    if (!isInstalled) {
      useAudioStore.getState().setPower(false);
      console.error("⚠️ CRITICAL: Driver de audio no detectado. Power OFF preventivo.");
    }
  }
);