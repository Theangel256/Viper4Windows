import { useAudioStore } from "../../store/audioStore";
import { Power, AlertCircle, Settings } from "lucide-react";

export function PowerButton() {
  const { master, setPower, isDriverInstalled } = useAudioStore();

  const handlePowerClick = () => {
    if (!isDriverInstalled) {
      // Aquí es donde disparas tu lógica de "Configurador"
      // Por ejemplo: abrir un modal de instalación o ir a la pestaña de drivers
      console.log("Acción bloqueada: Instala el driver primero.");
      return;
    }
    setPower(!master.power);
  };

  // 1. Estado: Driver no instalado
  if (!isDriverInstalled) {
    return (
      <button 
        onClick={handlePowerClick}
        className="flex items-center gap-2 px-4 py-1.5 rounded-full transition-all border 
                   bg-amber-50 text-amber-600 border-amber-200 
                   dark:bg-amber-900/10 dark:text-amber-500 dark:border-amber-900/30 
                   hover:bg-amber-100 cursor-help font-bold text-sm"
        title="Haz clic para configurar el driver de audio"
      >
        <AlertCircle size={14} /> DRIVER REQUIRED
      </button>
    );
  }

  // 2. Estado: Driver OK (Tu botón original con lógica segura)
  return (
    <button 
      onClick={handlePowerClick}
      className={`flex items-center gap-2 px-4 py-1.5 rounded-full transition-all border text-sm font-bold ${
        master.power 
        ? 'bg-accent/10 text-accent border-accent/20 dark:bg-purple-500/10 dark:text-purple-400 dark:border-purple-500/20 shadow-[0_0_15px_rgba(168,85,247,0.15)]' 
        : 'bg-zinc-100 text-zinc-400 border-zinc-200 dark:bg-zinc-800 dark:border-zinc-700'
      }`}
    >
      <Power size={14} /> 
      {master.power ? 'POWER ON' : 'POWER OFF'}
    </button>
  );
}