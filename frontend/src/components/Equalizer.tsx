import { memo, useState } from "react";
import { createPortal } from "react-dom";
import { useAudioStore } from "../store/audioStore";
import { DSPSlider } from "./ui/DSPSlider";
import { DSPButton } from "./ui/DSPButton";
import { Activity, RotateCcw, Save, X } from "lucide-react";

// Definimos las frecuencias estándar para un ecualizador de 18 bandas
const BANDS = [
  "65", "93", "131", "185", "262", "370", "523", "740", 
  "1.0k", "1.5k", "2.1k", "3.0k", "4.2k", "6.0k", "8.4k", 
  "11.8k", "16.7k", "20k"
];

interface EqualizerProps {
  onClose?: () => void;
  overlay?: boolean;
}

export const Equalizer = memo(({ onClose, overlay = false }: EqualizerProps) => {
  const { equalizer, setEqBand, resetEq, savePreset } = useAudioStore();
  const [showSaveModal, setShowSaveModal] = useState(false);
  const [presetName, setPresetName] = useState("");

  const handleSavePreset = () => {
    if (presetName.trim()) {
      savePreset(presetName.trim());
      setPresetName("");
      setShowSaveModal(false);
    }
  };

  return (
    <>
      <section
        className={`relative w-full border border-zinc-200 bg-white shadow-sm dark:border-zinc-800 dark:bg-zinc-900 ${
          overlay
            ? "mx-auto max-w-[1320px] rounded-[30px] px-4 py-4 sm:px-5 lg:px-6 lg:py-5"
            : "rounded-[28px] px-5 py-4"
        }`}
      >
        {/* Header del Ecualizador */}
        <div className={`flex items-center justify-between ${overlay ? "mb-4 sm:mb-5" : "mb-5"}`}>
          <div className="flex items-center gap-3">
            <div className="p-2 bg-purple-500/10 rounded-xl">
              <Activity className="text-purple-500" size={18} />
            </div>
            <div>
              <h2 className="text-base font-bold text-zinc-800 dark:text-zinc-100 leading-none">
                Equalizer
              </h2>
              <p className="text-[10px] text-zinc-400 uppercase tracking-widest mt-1 font-bold">
                18-Band Precision Control
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {onClose && (
              <button
                onClick={onClose}
                className="p-1.5 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200 transition-colors"
                title="Close equalizer"
              >
                <X size={16} />
              </button>
            )}
            <button 
              onClick={() => resetEq?.()}
              className="p-1.5 text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 transition-colors"
              title="Reset to 0dB"
            >
              <RotateCcw size={16} />
            </button>
            <button 
              onClick={() => setShowSaveModal(true)}
              className="flex items-center gap-2 px-3.5 py-1.5 bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 rounded-2xl text-[11px] font-bold hover:opacity-90 transition-opacity"
            >
              <Save size={14} /> SAVE PRESET
            </button>
        </div>
      </div>

      {/* Grid de Sliders Verticales */}
      <div
        className={`flex flex-row items-end gap-1.5 overflow-x-auto custom-scrollbar ${
          overlay
            ? "h-[min(58vh,440px)] justify-start sm:justify-between px-1 pb-2 sm:pb-3"
            : "h-[268px] justify-between px-1 pb-3"
        }`}
      >
        {BANDS.map((freq, index) => (
          <div key={freq} className="flex flex-col items-center gap-2.5 h-full min-w-[28px] group">
            {/* Valor en dB (solo visible al interactuar o sutil) */}
            <span className="text-[9px] font-bold tabular-nums text-zinc-400 group-hover:text-purple-500 transition-colors">
              {equalizer?.[index]?.toFixed(1) || "0.0"}
            </span>

            {/* Contenedor del Slider Vertical */}
            <div className="relative flex-1 w-5 flex items-center justify-center">
              {/* Aquí rotamos el DSPSlider 270 grados para hacerlo vertical.
                  Ajustamos el ancho al alto del contenedor.
              */}
              <div
                className={`absolute -rotate-90 origin-center pointer-events-auto ${
                  overlay ? "w-[min(34vw,320px)] sm:w-[230px] lg:w-[250px]" : "w-[204px]"
                }`}
              >
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
              <div className="w-1 h-1 bg-zinc-200 dark:bg-zinc-700 rounded-full mb-1.5" />
              <span className="text-[9px] font-bold text-zinc-500 dark:text-zinc-400 orientation-vertical">
                {freq}
              </span>
            </div>
          </div>
        ))}
      </div>

      {/* Línea de referencia de 0dB */}
      <div
        className={`absolute left-0 right-0 h-[1px] bg-zinc-100 dark:bg-zinc-800/50 pointer-events-none z-0 ${
          overlay ? "top-[calc(50%+10px)]" : "top-[calc(50%+4px)]"
        }`}
      />
    </section>

    {/* Save Preset Modal */}
    {showSaveModal && typeof document !== 'undefined' && createPortal(
      <div className="fixed inset-0 z-[100] flex items-center justify-center">
        <div className="absolute inset-0 bg-zinc-950/40 transition-opacity" onClick={() => setShowSaveModal(false)} />
        <div className="relative bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl p-6 w-[320px] shadow-2xl animate-in fade-in zoom-in duration-300">
          <h3 className="text-sm font-bold text-zinc-800 dark:text-zinc-100 mb-4">Save Equalizer Preset</h3>
          <input
            autoFocus
            type="text"
            placeholder="Enter preset name..."
            value={presetName}
            onChange={(e) => setPresetName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSavePreset()}
            className="w-full px-4 py-3 rounded-xl border border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800 text-sm outline-none focus:ring-2 focus:ring-purple-500/20 focus:border-purple-500 transition-all mb-4"
          />
          <div className="flex gap-3">
            <button className="flex-1 py-2.5 text-sm font-medium text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded-xl transition-colors" onClick={() => setShowSaveModal(false)}>Cancel</button>
            <DSPButton variant="default" className="flex-1 justify-center" onClick={handleSavePreset}>Save</DSPButton>
          </div>
        </div>
      </div>,
      document.body
    )}
  </>
  );
});

Equalizer.displayName = "Equalizer";
