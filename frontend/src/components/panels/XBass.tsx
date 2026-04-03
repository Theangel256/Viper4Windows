import { useAudioStore } from "../../store/audioStore";
import { FeatureCard } from "../ui/FeatureCard";
import { DSPSlider } from "../ui/DSPSlider";
import { DSPOptionList } from "../ui/DSPOptionList";

const BASS_MODES = ["Natural Bass", "Pure Bass"] as const;

export function XBass() {
  const { xBass, setXBass } = useAudioStore();

  return (
    <FeatureCard
      title="XBass"
      value={xBass.on}
      onToggle={(on) => setXBass({ on })}
    >
      <DSPSlider
        label="Speaker Size"
        min={0} max={10} step={1}
        value={xBass.speakerSize}
        unit=""
        onChange={(v) => setXBass({ speakerSize: v })}
      />
      <DSPSlider
        label="Bass Level"
        min={-12} max={12} step={0.05}
        value={xBass.level}
        unit="dB"
        onChange={(v) => setXBass({ level: v })}
      />
      <DSPOptionList
        options={[...BASS_MODES]}
        value={xBass.mode}
        onChange={(v) => setXBass({ mode: v as typeof xBass.mode })}
      />
    </FeatureCard>
  );
}