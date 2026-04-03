import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { 
  Power, 
  RotateCcw, 
  Music, 
  Film, 
  Wand2, 
  Download, 
  Save
} from "lucide-react";

// Importación de paneles
import { MasterControls } from "./panels/MasterControls";
import { XBass }          from "./panels/XBass";
import { XClarity }       from "./panels/XClarity";
import { Surround3D }     from "./panels/Surround3D";
import { ReverbSidebar }  from "./panels/ReverbSidebar";
import { ReverbPanel }    from "./panels/ReverbPanel";
import { DSPButton }      from "./ui/DSPButton";
import { useAudioStore }  from "../store/audioStore";
import { Sidebar } from "./Sidebar";
import { Equalizer } from "./Equalizer";

interface AudioDSPProps {
  systemStatus: string;
  onRefreshStatus: () => void;
}

export function AudioDSP({ systemStatus, onRefreshStatus }: AudioDSPProps) {
  const [activeMode, setActiveMode] = useState('Freestyle');
  const setDriverStatus = useAudioStore((state) => state.setDriverStatus);
  const modes = [
    { id: 'Music Mode', icon: Music },
    { id: 'Movie Mode', icon: Film },
    { id: 'Freestyle', icon: Wand2 },
  ];

  // Variantes para animar la entrada de los paneles
  const containerVariants = {
    hidden: { opacity: 0 },
    show: {
      opacity: 1,
      transition: { staggerChildren: 0.125 }
    }
  };

  const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    show: { 
      opacity: 1, 
      y: 0, 
      transition: { 
        type: "spring" as const, // <-- "Fíjalo" como constante
        stiffness: 300, 
        damping: 24 
      } 
    }
  };

  return (
    // Contenedor principal: Light Mode, Flex Row para el Sidebar
    <div id="App" className="h-screen w-screen flex bg-[#f8f9fa] text-zinc-900 p-4 md:p-6 select-none overflow-hidden relative font-sans gap-6">

      {/* --- SIDEBAR IZQUIERDO --- */}
      <motion.aside 
        initial={{ opacity: 0, x: -20 }}
        animate={{ opacity: 1, x: 0 }}
        className="w-64 flex flex-col justify-between shrink-0 z-10"
      >
        <Sidebar ></Sidebar>
      </motion.aside>

      {/* --- CONTENIDO PRINCIPAL --- */}
      <main className="flex-1 flex flex-col z-10 overflow-y-auto pr-2 custom-scrollbar"> 
        {/* Barra superior de estado del driver */}
        <header className="flex justify-end items-center gap-4 mb-4">
           <div className="flex items-center gap-2 bg-white px-3 py-1.5 rounded-full shadow-sm border border-zinc-200">
             <span className={`text-[9px] font-black uppercase tracking-widest ${systemStatus === 'REGISTERED' ? 'text-green-500' : 'text-red-500'}`}>
                {systemStatus}
             </span>
             <div className={`w-2 h-2 rounded-full animate-pulse ${systemStatus === 'REGISTERED' ? 'bg-green-500' : 'bg-red-500'}`} />
           </div>
           <button onClick={onRefreshStatus} className="text-[10px] text-zinc-400 hover:text-zinc-800 transition-colors flex items-center gap-1 font-bold">
             <RotateCcw size={12} /> RECHECK
           </button>
        </header>

        {/* Grid animado de Paneles */}
        <motion.div 
          variants={containerVariants}
          initial="hidden"
          animate="show"
          className="flex flex-col gap-5 pb-6"
        >
          {/* Master Controls */}
          <motion.section variants={itemVariants} className="w-full">
            <MasterControls />
          </motion.section>

          {/* Grid de 4 Columnas (Efectos) */}
          <motion.section variants={itemVariants} className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-5">
            <XBass />
            <XClarity />
            <Surround3D />
            <ReverbSidebar />
          </motion.section>

          {/* Panel inferior de Reverb */}
          <motion.section variants={itemVariants} className="bg-white rounded-3xl p-5 shadow-sm border border-zinc-200">
          <ReverbPanel />
          </motion.section>

          {/* Footer: EQ y Compressor */}
          <motion.footer variants={itemVariants} className="grid grid-cols-2 gap-5 mt-2">
          <Equalizer />
          </motion.footer>

        </motion.div>
      </main>
      </div>
  );
}