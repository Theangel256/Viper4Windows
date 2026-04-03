import { useAudioStore } from "../../store/audioStore";
import { FeatureCard } from "../ui/FeatureCard";
import { DSPSlider } from "../ui/DSPSlider";
import { DSPButton } from "../ui/DSPButton";

const ROOM_SIZES = ["Smallest Room", "Small Room", "Medium Room", "Large Room", "Hall"];

export function Surround3D() {
  const { surround3D, setSurround3D } = useAudioStore();

  const cycleRoom = () => {
    const idx = ROOM_SIZES.indexOf(surround3D.roomSize);
    setSurround3D({ roomSize: ROOM_SIZES[(idx + 1) % ROOM_SIZES.length] });
  };

  return (
    <FeatureCard
      title="3D Surround"
      value={surround3D.on}
      onToggle={(on) => setSurround3D({ on })}
    >
      <DSPSlider
        label="Space Size"
        min={0} max={10} step={1}
        value={surround3D.spaceSize}
        unit=""
        onChange={(v) => setSurround3D({ spaceSize: v })}
      />

      <div className="flex flex-col gap-1.5">
        <span className="text-[11px] font-medium text-zinc-500 uppercase tracking-wider">
          Room Size
        </span>
        <DSPButton
          active
          variant="danger"
          className="w-full justify-center"
          onClick={cycleRoom}
        >
          {surround3D.roomSize}
        </DSPButton>
      </div>

      <DSPSlider
        label="Image Size"
        min={0} max={10} step={1}
        value={surround3D.imageSize}
        unit=""
        onChange={(v) => setSurround3D({ imageSize: v })}
      />
    </FeatureCard>
  );
}
