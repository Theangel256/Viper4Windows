import { useAudioStore } from "../../store/audioStore";
import { useState } from "react";
import { Power, AlertCircle } from "lucide-react";

export function PowerButton() {
  const { master, setPower, isDriverInstalled, installDriver } = useAudioStore();
  const [installing, setInstalling] = useState(false);

  const handleAction = async () => {
    if (!isDriverInstalled) {
      setInstalling(true);
      try {
        await installDriver();
      } catch (err) {
        console.error("Error installing driver:", err);
      } finally {
        setInstalling(false);
      }
      return;
    }

    setPower(!master.power);
  };

  return (
    <button
      onClick={handleAction}
      disabled={installing}
      className={`flex items-center gap-2 px-4 py-1.5 rounded-full transition-all border text-sm font-bold disabled:opacity-60 ${
        !isDriverInstalled
          ? "bg-amber-50 text-amber-600 border-amber-200 dark:bg-amber-900/10 dark:border-amber-900/30"
          : master.power
            ? "bg-purple-500/10 text-purple-400 border-purple-500/20 shadow-[0_0_15px_rgba(168,85,247,0.15)]"
            : "bg-zinc-800 text-zinc-500 border-zinc-700"
      }`}
    >
      {!isDriverInstalled ? <AlertCircle size={14} /> : <Power size={14} />}
      {!isDriverInstalled
        ? installing
          ? "INSTALLING…"
          : "INSTALL DRIVER"
        : master.power
          ? "POWER ON"
          : "POWER OFF"}
    </button>
  );
}
