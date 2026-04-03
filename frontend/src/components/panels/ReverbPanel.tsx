import { useAudioStore } from "../../store/audioStore";
import { DSPSlider } from "../ui/DSPSlider";
import { DSPToggle } from "../ui/DSPToggle";
import { DSPButton } from "../ui/DSPButton";

const ROOM_SIZES = ["Smallest Room", "Small Room", "Medium Room", "Large Room", "Hall"];

export function ReverbPanel() {
  const { reverbPanel, setReverbPanel } = useAudioStore();

  const cycleRoom = () => {
    const idx = ROOM_SIZES.indexOf(reverbPanel.roomSize);
    setReverbPanel({ roomSize: ROOM_SIZES[(idx + 1) % ROOM_SIZES.length] });
  };

  return (
    <div className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl px-5 py-4 shadow-card">
      <div className="flex items-center justify-between mb-3">
        <span className="text-[15px] font-semibold text-zinc-800 dark:text-zinc-100 tracking-tight">
          Reverberation
        </span>
        <DSPToggle
          value={reverbPanel.on}
          onChange={(on) => setReverbPanel({ on })}
        />
      </div>

      <div className="flex items-center gap-4">
        <DSPButton variant="danger" active onClick={cycleRoom} className="shrink-0">
          {reverbPanel.roomSize}
        </DSPButton>

        <DSPSlider
          min={0} max={100} step={1}
          value={reverbPanel.size}
          unit=""
          showValue={false}
          className="flex-1"
          onChange={(v) => setReverbPanel({ size: v })}
        />
        <span className="text-[13px] font-semibold tabular-nums text-zinc-700 dark:text-zinc-300 w-7 text-right">
          {Math.round(reverbPanel.size)}
        </span>

        <DSPSlider
          min={0} max={100} step={1}
          value={reverbPanel.wetMix}
          unit=""
          showValue={false}
          className="flex-1"
          onChange={(v) => setReverbPanel({ wetMix: v })}
        />
        <span className="text-[13px] font-semibold tabular-nums text-zinc-700 dark:text-zinc-300 w-7 text-right">
          {Math.round(reverbPanel.wetMix)}
        </span>
      </div>
    </div>
  );
}
