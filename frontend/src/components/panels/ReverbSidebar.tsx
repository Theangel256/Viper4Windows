import { useAudioStore } from "../../store/audioStore";
import { DSPToggle } from "../ui/DSPToggle";
import { DSPSlider } from "../ui/DSPSlider";
import { DSPButton } from "../ui/DSPButton";
import type { ReverbParams } from "../../store/audioStore";

type ReverbKey = keyof Omit<ReverbParams, "on">;

interface ParamDef {
  key: ReverbKey;
  label: string;
  min: number;
  max: number;
  step: number;
  unit: string;
}

const PARAMS: ParamDef[] = [
  { key: "roomSize",  label: "Room Size",  min: 0,   max: 1000, step: 1,    unit: ""   },
  { key: "damping",   label: "Damping",    min: 0,   max: 5,    step: 0.01, unit: ""   },
  { key: "density",   label: "Density",    min: 0,   max: 100,  step: 0.1,  unit: ""   },
  { key: "bandwidth", label: "Bandwidth",  min: 0,   max: 100,  step: 1,    unit: ""   },
  { key: "decay",     label: "Decay",      min: 0,   max: 100,  step: 0.1,  unit: ""   },
  { key: "preDelay",  label: "Pre-delay",  min: 0,   max: 500,  step: 1,    unit: "ms" },
  { key: "earlyMix",  label: "Early Mix",  min: 0,   max: 100,  step: 1,    unit: ""   },
];

export function ReverbSidebar() {
  const { reverb, setReverb } = useAudioStore();

  return (
    <div className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl p-4 flex flex-col gap-2 shadow-card">
      {/* Header */}
      <div className="flex items-center justify-between mb-1">
        <span className="text-[15px] font-semibold text-zinc-800 dark:text-zinc-100 tracking-tight">
          Reverberation
        </span>
        <DSPToggle value={reverb.on} onChange={(on) => setReverb({ on })} />
      </div>

      {/* Parameter rows */}
      <div className="flex flex-col gap-2">
        {PARAMS.map((p) => (
          <div
            key={p.key}
            className="grid items-center gap-2"
            style={{ gridTemplateColumns: "76px 1fr 34px" }}
          >
            <span className="text-[11px] text-zinc-500 dark:text-zinc-400 truncate">
              {p.label}
            </span>
            <DSPSlider
              min={p.min}
              max={p.max}
              step={p.step}
              value={reverb[p.key] as number}
              unit={p.unit}
              showValue={false}
              compact
              onChange={(v) => setReverb({ [p.key]: v })}
            />
            <span className="text-[11px] font-semibold tabular-nums text-zinc-700 dark:text-zinc-300 text-right">
              {(reverb[p.key] as number).toFixed(p.step < 1 ? (p.step < 0.1 ? 2 : 1) : 0)}
              {p.unit}
            </span>
          </div>
        ))}
      </div>

      {/* Wet Mix row */}
      <div className="flex items-center justify-between pt-2 border-t border-zinc-100 dark:border-zinc-800 mt-1">
        <span className="text-[11px] text-zinc-500">Wet Mix</span>
        <DSPButton size="sm">Preset</DSPButton>
      </div>
    </div>
  );
}