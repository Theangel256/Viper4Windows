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
import { StatusHeader } from "./ui/StatusHeader";

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
    <div id="App" className="h-screen w-screen flex bg-background overflow-hidden relative">
      {/* --- SIDEBAR IZQUIERDO --- */}
      <Sidebar />
      {/* --- CONTENIDO PRINCIPAL --- */}
      <main className="flex-1 flex flex-col z-10 overflow-y-auto p-8 custom-scrollbar">
        {/* Barra superior de estado del driver */}
        <StatusHeader />

        {/* Grid animado de Paneles */}
        <motion.div 
          variants={containerVariants}
          initial="hidden"
          animate="show"
          className="flex flex-col gap-5 pb-6 mt-8 max-w-[1600px] mx-auto w-full"
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
          {/* Panel inferior de Reverb
          <motion.section variants={itemVariants} className="bg-white rounded-3xl p-5 shadow-sm border border-zinc-200">
          <ReverbPanel />
          </motion.section>*/}
          {/* Footer: EQ y Compressor */}
          <motion.footer variants={itemVariants} className="grid grid-cols-2 gap-5 mt-2">
          <Equalizer />
          </motion.footer>

        </motion.div>
      </main>
      </div>
  );
}