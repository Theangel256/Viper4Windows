import { useState, useEffect, memo } from "react";
import { createPortal } from "react-dom";
import { motion } from "framer-motion";
import { useAudioStore } from "../store/audioStore";
import { DSPButton } from "./ui/DSPButton";
import { ThemeToggle } from "./ui/ThemeToggle.jsx"

// --- COMPONENTES DE ICONOS (Memorizados) ---
const MusicIcon = memo(() => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={1.8} viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" d="M9 9l10.5-3m0 6.553v3.75a2.25 2.25 0 01-1.632 2.163l-1.32.377a1.803 1.803 0 11-.99-3.467l2.31-.66a2.25 2.25 0 001.632-2.163zm0 0V2.25L9 5.25v10.303m0 0v3.75a2.25 2.25 0 01-1.632 2.163l-1.32.377a1.803 1.803 0 01-.99-3.467l2.31-.66A2.25 2.25 0 009 15.553z" />
  </svg>
));

const MovieIcon = memo(() => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={1.8} viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" d="M3.375 19.5h17.25m-17.25 0a1.125 1.125 0 01-1.125-1.125M3.375 19.5h7.5c.621 0 1.125-.504 1.125-1.125m-9.75 0V5.625m0 12.75v-1.5c0-.621.504-1.125 1.125-1.125m18.375 2.625V5.625m0 12.75c0 .621-.504 1.125-1.125 1.125m1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125m0 3.75h-7.5A1.125 1.125 0 0112 18.375m9.75-12.75c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125m19.5 0v1.5c0 .621-.504 1.125-1.125 1.125M2.25 5.625v1.5c0 .621.504 1.125 1.125 1.125m0 0h17.25m-17.25 0h7.5c.621 0 1.125.504 1.125 1.125M3.375 8.25c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125h.75" />
  </svg>
));

const FreestyleIcon = memo(() => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={1.8} viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" d="M9.53 16.122a3 3 0 00-5.78 1.128 2.25 2.25 0 01-2.4 2.245 4.5 4.5 0 008.4-2.245c0-.399-.078-.78-.22-1.128zm0 0a15.998 15.998 0 003.388-1.62m-5.043-.025a15.994 15.994 0 011.622-3.395m3.42 3.42a15.995 15.995 0 004.764-4.648l3.876-5.814a1.151 1.151 0 00-1.597-1.597L14.146 6.32a15.996 15.996 0 00-4.649 4.763m3.42 3.42a6.776 6.776 0 00-3.42-3.42" />
  </svg>
));

const SavePresetIcon = memo(() => (
  <svg className="w-4 h-4 text-zinc-500" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
  </svg>
));

const LoadPresetIcon = memo(() => (
  <svg className="w-4 h-4 text-zinc-500" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
  </svg>
));

const MODES = [
  { id: "music", label: "Music Mode", Icon: MusicIcon },
  { id: "movie", label: "Movie Mode", Icon: MovieIcon },
  { id: "freestyle", label: "Freestyle", Icon: FreestyleIcon },
] as const;

// --- MODALES ---
const SavePresetModal = ({ isOpen, onClose, onSave }: any) => {
  const [name, setName] = useState("");
  if (!isOpen) return null;
  return createPortal(
    <div className="fixed inset-0 z-[100] flex items-center justify-center">
      <div className="absolute inset-0 bg-zinc-950/40 transition-opacity" onClick={onClose} />
      <div className="relative bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl p-6 w-[320px] shadow-2xl animate-in fade-in zoom-in duration-300">
        <h3 className="text-sm font-bold text-zinc-800 dark:text-zinc-100 mb-4">Save Current Preset</h3>
        <input
          autoFocus
          type="text"
          placeholder="Enter preset name..."
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full px-4 py-3 rounded-xl border border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800 text-sm outline-none focus:ring-2 focus:ring-red-500/20 focus:border-red-500 transition-all mb-4"
        />
        <div className="flex gap-3">
          <button className="flex-1 py-2.5 text-sm font-medium text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded-xl transition-colors" onClick={onClose}>Cancel</button>
          <DSPButton variant="danger" className="flex-1 justify-center" onClick={() => { onSave(name); setName(""); }}>Save</DSPButton>
        </div>
      </div>
    </div>,
    document.body
  );
};

const LoadPresetModal = ({ isOpen, onClose, presets, onLoad }: any) => {
  if (!isOpen) return null;
  return createPortal(
    <div className="fixed inset-0 z-[100] flex items-center justify-center">
      <div className="absolute inset-0 bg-zinc-950/40 transition-opacity" onClick={onClose} />
      <div className="relative bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-3xl p-6 w-[340px] shadow-2xl animate-in fade-in zoom-in duration-300">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-bold text-zinc-800 dark:text-zinc-100">Load Preset</h3>
          <button onClick={onClose} className="text-zinc-400 hover:text-zinc-600 transition-colors">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div className="flex flex-col gap-1.5 max-h-[300px] overflow-y-auto pr-1 custom-scrollbar">
          {presets.length === 0 ? (
            <p className="text-center py-4 text-zinc-400 text-sm">No presets found</p>
          ) : (
            presets.map((p: string) => (
              <button key={p} onClick={() => onLoad(p)} className="group text-left px-4 py-3 rounded-xl text-sm text-zinc-600 dark:text-zinc-400 hover:bg-red-50 dark:hover:bg-red-900/10 hover:text-red-600 transition-all flex items-center justify-between">
                {p}
                <svg className="w-4 h-4 opacity-0 group-hover:opacity-100 transition-opacity" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path d="M9 5l7 7-7 7" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              </button>
            ))
          )}
        </div>
      </div>
    </div>,
    document.body
  );
};

// --- SIDEBAR PRINCIPAL ---
export function Sidebar() {
  const { mode, presets, setMode, savePreset, loadPreset, isDriverInstalled } = useAudioStore();
  const [showSaveModal, setShowSaveModal] = useState(false);
  const [showLoadModal, setShowLoadModal] = useState(false);
  const [isMounted, setIsMounted] = useState(false);
  const [isCollapsed, setIsCollapsed] = useState(false);

  useEffect(() => { setIsMounted(true); }, []);

  if (!isMounted) return <aside className="w-[230px] h-screen bg-transparent border-r border-zinc-200/50" />;

  return (
    <>
      <motion.aside 
        initial={{ opacity: 0, x: -20 }} 
        animate={{ 
          opacity: 1, 
          x: 0,
          width: isCollapsed ? 90 : 230 
        }}
        transition={{ duration: 0.3, ease: "easeInOut" }} 
        className="flex flex-col h-screen shrink-0 py-8 px-5 bg-transparent border-r border-zinc-200/50 dark:border-zinc-800/50 overflow-hidden z-10"
      >
        {/* Logo y Botón de Contraer */}
        <div className={`flex items-center mb-10 transition-all duration-300 ${isCollapsed ? "flex-col gap-6" : "justify-between"}`}>
          <div className={`${!isDriverInstalled ? "logo-error" : ""} h-14 w-14 bg-white dark:bg-zinc-900 rounded-[20px] shadow-lg border border-zinc-200/60 dark:border-zinc-800 flex items-center justify-center text-purple-600 font-black text-3xl shrink-0 transition-all duration-300 hover:scale-105 active:scale-95`}>
            V
          </div>
          
          <div className={`flex items-center gap-1 ${isCollapsed ? "flex-col gap-4" : ""}`}>
            <ThemeToggle />
            <button onClick={() => setIsCollapsed(!isCollapsed)} className="p-2 rounded-xl hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors">
              <svg className={`w-6 h-6 text-zinc-500 transition-transform duration-300 ${isCollapsed ? "rotate-180" : ""}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16m-7 6h7" />
              </svg>
            </button>
          </div>
        </div>

        {/* Navegación de Modos */}
        <nav className={`flex flex-col gap-2 ${isCollapsed ? "px-0" : "px-1"}`}>
          {MODES.map(({ id, label, Icon }) => {
            const active = mode === id;
            return (
              <button
                key={id}
                onClick={() => setMode(id as any)}
                title={isCollapsed ? label : ""}
                className={`flex items-center rounded-xl text-[14px] font-medium transition-all duration-300 text-left w-full overflow-hidden
                  ${isCollapsed ? "justify-center p-0 py-3.5" : "justify-start px-4 py-3"}
                  ${active 
                    ? "bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-50 border border-red-200 dark:border-red-900/30 shadow-sm" 
                    : "text-zinc-500 dark:text-zinc-400 hover:bg-white/60 dark:hover:bg-zinc-800/60 border border-transparent"
                  }`}
              >
                <span className={`shrink-0 ${active ? "text-red-500" : "text-zinc-400"}`}>
                  <Icon />
                </span>
                
                {/* Animación del texto: max-width + opacity */}
                <span className={`overflow-hidden whitespace-nowrap transition-all duration-300 ease-in-out ${
                  isCollapsed ? "max-w-0 opacity-0 ml-0" : "max-w-[130px] opacity-100 ml-3"
                }`}>
                  {label}
                </span>
              </button>
            );
          })}
        </nav>
        
        {/* Barra Freestyle */}
        <div className={`transition-all duration-300 overflow-hidden ${mode === "freestyle" && !isCollapsed ? "max-h-20 opacity-100 mt-4 px-5" : "max-h-0 opacity-0"}`}>
          <div className="h-1 rounded-full bg-zinc-200 dark:bg-zinc-800 overflow-hidden">
            <div className="h-full w-2/5 rounded-full bg-red-400 shadow-[0_0_8px_rgba(248,113,113,0.4)]" />
          </div>
        </div>

        <div className="flex-1" />
        
        {/* Botones Inferiores (Load/Save) - Centrado perfeccionado */}
        <div className={`flex flex-col gap-3 mt-auto ${isCollapsed ? "items-center px-0" : "px-1"}`}>
          <DSPButton 
            title={isCollapsed ? "Load Preset" : ""}
            className={`shadow-sm bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 transition-all duration-300 overflow-hidden ${
              isCollapsed ? "w-12 h-12 p-0 justify-center rounded-2xl mx-auto" : "w-full justify-start px-4 py-3"
            }`} 
            onClick={() => setShowLoadModal(true)}
          >
            <span className="shrink-0"><LoadPresetIcon /></span>
            <span className={`overflow-hidden whitespace-nowrap transition-all duration-300 ease-in-out ${
              isCollapsed ? "max-w-0 opacity-0 ml-0" : "max-w-[120px] opacity-100 ml-2.5"
            }`}>
              Load Preset
            </span>
          </DSPButton>
          
          <DSPButton 
            title={isCollapsed ? "Save Preset" : ""}
            className={`shadow-sm bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 transition-all duration-300 overflow-hidden ${
              isCollapsed ? "w-12 h-12 p-0 justify-center rounded-2xl mx-auto" : "w-full justify-start px-4 py-3"
            }`} 
            onClick={() => setShowSaveModal(true)}
          >
            <span className="shrink-0"><SavePresetIcon /></span>
            <span className={`overflow-hidden whitespace-nowrap transition-all duration-300 ease-in-out ${
              isCollapsed ? "max-w-0 opacity-0 ml-0" : "max-w-[120px] opacity-100 ml-2.5"
            }`}>
              Save Preset
            </span>
          </DSPButton>
        </div>
      </motion.aside>

      {/* Portales para Modales */}
      <SavePresetModal 
        isOpen={showSaveModal} 
        onClose={() => setShowSaveModal(false)} 
        onSave={async (name: string) => {
          if (name.trim()) {
            await savePreset(name);
            setShowSaveModal(false);
          }
        }} 
      />
      
      <LoadPresetModal 
        isOpen={showLoadModal} 
        onClose={() => setShowLoadModal(false)} 
        presets={presets}
        onLoad={(p: string) => {
          loadPreset(p);
          setShowLoadModal(false);
        }}
      />
    </>
  );
}