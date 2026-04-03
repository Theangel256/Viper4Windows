import { useAudioStore } from "../../store/audioStore";
import { Power, AlertCircle, Settings } from "lucide-react";

export function PowerButton() {
  const { master, setPower, isDriverInstalled, checkDriverStatus } = useAudioStore();
  const handleAction = async () => {
    if (!isDriverInstalled) {
      // Si no hay driver, intentamos instalarlo
      try {
        const success = await window.go.main.App.SetDriverStatus(true);
        if (success) {
          alert("¡Driver instalado! Reiniciando estado...");
          checkDriverStatus(); // Actualizamos el store sin recargar toda la página
        } else {
          alert("Error: Asegúrate de ejecutar como Administrador.");
        }
      } catch (err) {
        console.error("Error en FixDriver:", err);
      }
      return;
    }
    
    // Si hay driver, simplemente toggle del power
    setPower(!master.power);
  };
  return (
    <button 
      onClick={handleAction}
      className={`flex items-center gap-2 px-4 py-1.5 rounded-full transition-all border text-sm font-bold ${
        !isDriverInstalled 
          ? 'bg-amber-50 text-amber-600 border-amber-200 dark:bg-amber-900/10 dark:border-amber-900/30' 
          : master.power 
            ? 'bg-purple-500/10 text-purple-400 border-purple-500/20 shadow-[0_0_15px_rgba(168,85,247,0.15)]'
            : 'bg-zinc-800 text-zinc-500 border-zinc-700'
      }`}
    >
      {!isDriverInstalled ? <AlertCircle size={14} /> : <Power size={14} />}
      {!isDriverInstalled ? 'INSTALL DRIVER' : master.power ? 'POWER ON' : 'POWER OFF'}
    </button>
  );
}