import { useAudioStore } from "../../store/audioStore";
import { FeatureCard } from "../ui/FeatureCard";
import { DSPSlider } from "../ui/DSPSlider";
import { DSPOptionList } from "../ui/DSPOptionList";

const CLARITY_MODES = ["Natural", "OZone+", "X-HiFi"];

export function XClarity() {
  const { xClarity, setXClarity } = useAudioStore();

  return (
    <FeatureCard
      title="XClarity"
      value={xClarity.on}
      onToggle={(on) => setXClarity({ on })}
    >
      <DSPSlider
        label="Clarity"
        min={-12} max={12} step={0.05}
        value={xClarity.level}
        unit="dB"
        onChange={(v) => setXClarity({ level: v })}
      />
      <div className="flex flex-col gap-1.5">
        <span className="text-[11px] font-medium text-zinc-500 uppercase tracking-wider">
          Options
        </span>
        <DSPOptionList
          options={[...CLARITY_MODES]}
          value={xClarity.mode}
          onChange={(v) => setXClarity({ mode: v as typeof xClarity.mode })}
        />
      </div>
    </FeatureCard>
  );
}