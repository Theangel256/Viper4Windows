import { memo } from "react";
import { useAudioStore } from "../store/audioStore";
import { DSPSlider } from "./ui/DSPSlider";
import { Activity, RotateCcw, Save } from "lucide-react";

// Definimos las frecuencias estándar para un ecualizador de 18 bandas
const BANDS = [
  "65", "93", "131", "185", "262", "370", "523", "740", 
  "1.0k", "1.5k", "2.1k", "3.0k", "4.2k", "6.0k", "8.4k", 
  "11.8k", "16.7k", "20k"
];

export const Equalizer = memo(() => {
  // Nota: Deberás añadir 'equalizer' y 'setEqBand' a tu interface AudioStore
  const { equalizer, setEqBand, resetEq } = useAudioStore();

  return (
    <section className="bg-white dark:bg-zinc-900 rounded-3xl p-6 shadow-sm border border-zinc-200 dark:border-zinc-800 w-full relative">
      {/* Header del Ecualizador */}
      <div className="flex items-center justify-between mb-8">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-purple-500/10 rounded-xl">
            <Activity className="text-purple-500" size={20} />
          </div>
          <div>
            <h2 className="text-lg font-bold text-zinc-800 dark:text-zinc-100 leading-none">
              Equalizer
            </h2>
            <p className="text-[10px] text-zinc-400 uppercase tracking-widest mt-1 font-bold">
              18-Band Precision Control
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button 
            onClick={() => resetEq?.()}
            className="p-2 text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 transition-colors"
            title="Reset to 0dB"
          >
            <RotateCcw size={18} />
          </button>
          <button className="flex items-center gap-2 px-4 py-2 bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 rounded-2xl text-xs font-bold hover:opacity-90 transition-opacity">
            <Save size={14} /> SAVE PRESET
          </button>
        </div>
      </div>

      {/* Grid de Sliders Verticales */}
      <div className="flex flex-row justify-between items-end h-[320px] gap-2 px-2 overflow-x-auto pb-4 custom-scrollbar">
        {BANDS.map((freq, index) => (
          <div key={freq} className="flex flex-col items-center gap-4 h-full min-w-[32px] group">
            {/* Valor en dB (solo visible al interactuar o sutil) */}
            <span className="text-[9px] font-bold tabular-nums text-zinc-400 group-hover:text-purple-500 transition-colors">
              {equalizer?.[index]?.toFixed(1) || "0.0"}
            </span>

            {/* Contenedor del Slider Vertical */}
            <div className="relative flex-1 w-6 flex items-center justify-center">
              {/* Aquí rotamos el DSPSlider 270 grados para hacerlo vertical.
                  Ajustamos el ancho al alto del contenedor.
              */}
              <div className="absolute w-[240px] -rotate-90 origin-center pointer-events-auto">
                <DSPSlider
                  min={-12}
                  max={12}
                  step={0.1}
                  value={equalizer?.[index] || 0}
                  onChange={(val) => setEqBand?.(index, val)}
                  showValue={false} // Ocultamos el label interno para usar el nuestro
                  compact
                />
              </div>
            </div>

            {/* Etiqueta de Frecuencia */}
            <div className="flex flex-col items-center">
              <div className="w-1 h-1 bg-zinc-200 dark:bg-zinc-700 rounded-full mb-2" />
              <span className="text-[10px] font-bold text-zinc-500 dark:text-zinc-400 orientation-vertical">
                {freq}
              </span>
            </div>
          </div>
        ))}
      </div>

      {/* Línea de referencia de 0dB */}
      <div className="absolute left-0 right-0 top-[calc(50%+12px)] h-[1px] bg-zinc-100 dark:bg-zinc-800/50 pointer-events-none z-0" />
    </section>
  );
});

Equalizer.displayName = "Equalizer";