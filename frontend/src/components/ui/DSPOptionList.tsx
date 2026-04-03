import { useState, memo } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { ChevronDown } from "lucide-react"; // O usa un SVG si prefieres

interface DSPOptionListProps {
  options:   string[];
  value:     string;
  onChange?: (val: string) => void;
}

export const DSPOptionList = memo(({ options, value, onChange }: DSPOptionListProps) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="flex flex-col rounded-xl border border-zinc-200 dark:border-zinc-800 overflow-hidden bg-zinc-50 dark:bg-zinc-900/50 transition-colors">
      
      {/* Botón de la Opción Seleccionada (Trigger) */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center justify-between px-4 py-2.5 text-[13px] font-medium text-zinc-900 dark:text-zinc-100 bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-800 transition-all"
      >
        <span>{value}</span>
        <motion.div
          animate={{ rotate: isOpen ? 180 : 0 }}
          transition={{ duration: 0.2, ease: "easeInOut" }}
        >
          <ChevronDown size={14} className="text-zinc-400" />
        </motion.div>
      </button>

      {/* Lista Desplegable Animada */}
      <AnimatePresence>
        {isOpen && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.25, ease: [0.4, 0, 0.2, 1] }} // "Standard easing" de UI
            className="overflow-hidden border-t border-zinc-100 dark:border-zinc-800"
          >
            <div className="flex flex-col bg-zinc-50/50 dark:bg-zinc-800/30">
              {options.map((opt) => {
                const active = opt === value;
                return (
                  <button
                    key={opt}
                    onClick={() => {
                      onChange?.(opt);
                      setIsOpen(false); // Se cierra al seleccionar
                    }}
                    className={`
                      flex items-center justify-between px-4 py-2.5 text-[13px]
                      transition-all text-left hover:pl-5
                      ${active
                        ? "text-red-500 font-bold bg-red-50/30 dark:bg-red-500/5"
                        : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200"
                      }
                    `}
                  >
                    <span>{opt}</span>
                    {active && (
                      <motion.svg 
                        initial={{ scale: 0 }}
                        animate={{ scale: 1 }}
                        className="w-3.5 h-3.5 text-red-500 shrink-0" 
                        fill="none" 
                        stroke="currentColor" 
                        strokeWidth={3} 
                        viewBox="0 0 24 24"
                      >
                        <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                      </motion.svg>
                    )}
                  </button>
                );
              })}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
});

DSPOptionList.displayName = "DSPOptionList";