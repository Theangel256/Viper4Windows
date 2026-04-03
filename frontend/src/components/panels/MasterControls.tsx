import { useAudioStore } from "../../store/audioStore";
import { DSPSlider } from "../ui/DSPSlider";
import { Power } from "lucide-react";
import { PowerButton } from "../ui/PowerButton";
/**
 * MasterControls — Pre/Post volume + power.
 * Reads/writes via useAudioStore (→ Go via Wails).
 */
export function MasterControls() {
  const { master, setPower, setPreVol, setPostVol } = useAudioStore();
  return (
<div className="bg-white dark:bg-zinc-900 rounded-3xl p-6 shadow-sm border border-zinc-200 dark:border-zinc-800 w-full">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-zinc-800 dark:text-zinc-100 tracking-tight">
          Master Controls
        </h2>
        <PowerButton />
      </div>
      <div className="grid grid-cols-2 gap-6">
        <DSPSlider
          label="Pre-Volume"
          min={-18} max={0} step={0.05}
          value={master.preVol}
          unit="dB"
          onChange={setPreVol}
        />
        <DSPSlider
          label="Post-Volume"
          min={0} max={18} step={0.05}
          value={master.postVol}
          unit="dB"
          onChange={setPostVol}
        />
      </div>
    </div>
  );
}