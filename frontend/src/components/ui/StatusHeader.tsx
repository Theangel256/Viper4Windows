import { useAudioStore } from "../../store/audioStore";
import { RotateCcw } from "lucide-react";

export function StatusHeader() {
  // Extraemos el booleano y la función de refresco del store
  const { isDriverInstalled, checkDriverStatus } = useAudioStore();

  // Convertimos el booleano al string que espera tu diseño
  const systemStatus = isDriverInstalled ? 'REGISTERED' : 'DRIVER MISSING';

  const onRefreshStatus = async () => {
    // Esto llamará a Go y actualizará isDriverInstalled en todo el sistema
    await checkDriverStatus();
  };

  return (
    <header className="flex justify-end items-center gap-4 mb-4 pt-8">
      <div className="flex items-center gap-2 bg-white dark:bg-zinc-900 px-3 py-1.5 rounded-full shadow-sm border border-zinc-200 dark:border-zinc-800">
        <span className={`text-[9px] font-black uppercase tracking-widest ${
          isDriverInstalled ? 'text-green-500' : 'text-red-500'
        }`}>
          {systemStatus}
        </span>
        <div className={`w-2 h-2 rounded-full ${
          isDriverInstalled ? 'bg-green-500 animate-pulse' : 'bg-red-500'
        }`} />
      </div>
      
      <button 
        onClick={onRefreshStatus} 
        className="text-[10px] text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200 transition-colors flex items-center gap-1 font-bold"
      >
        <RotateCcw size={12} /> RECHECK
      </button>
    </header>
  );
}