import { useCallback, memo } from "react";

interface DSPSliderProps {
  label?:     string;
  min?:       number;
  max?:       number;
  step?:      number;
  value:      number;
  unit?:      string;
  showValue?: boolean;
  compact?:   boolean;
  onChange?:  (val: number) => void;
  className?: string;
}

export const DSPSlider = memo(({
  label = "",
  min = -24,
  max = 24,
  step = 0.01,
  value,
  unit = "dB",
  showValue = true,
  compact = false,
  onChange,
  className = "",
}: DSPSliderProps) => {
  
  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onChange?.(parseFloat(e.target.value));
    },
    [onChange]
  );

  // Evitamos errores de división por cero si min === max
  const range = max - min;
  const pct = range === 0 ? 0 : ((value - min) / range) * 100;
  
  const decimals = step < 0.1 ? 2 : step < 1 ? 1 : 0;
  const displayVal = value.toFixed(decimals);

  return (
    <div className={`w-full flex flex-col gap-1.5 ${className}`}>
      {showValue && !compact && (
        <div className="flex items-center justify-between w-full px-0.5">
          <span className="text-[10px] font-bold text-zinc-400 dark:text-zinc-500 uppercase tracking-widest">
            {label}
          </span>
          <span className="text-[11px] font-bold tabular-nums text-zinc-700 dark:text-zinc-200">
            {displayVal}<span className="text-[9px] ml-0.5 opacity-60">{unit}</span>
          </span>
        </div>
      )}

      <div className="relative w-full h-6 flex items-center group">
        {/* 1. CAPA VISUAL (Fondo y Progreso) - pointer-events-none para que no bloqueen el clic */}
        <div className="absolute w-full h-1.5 bg-zinc-100 dark:bg-zinc-800/60 rounded-full pointer-events-none" />
        
        <div 
          className="absolute h-1.5 bg-zinc-900 dark:bg-zinc-100 rounded-full pointer-events-none" 
          style={{ width: `${pct}%` }}
        />

        {/* 2. EL PUNTO ROJO VISUAL - Sincronizado con el porcentaje */}
        <div 
          className="absolute w-4 h-4 bg-white border-[3px] border-red-500 rounded-full shadow-md pointer-events-none z-0"
          style={{ 
            left: `${pct}%`,
            transform: "translateX(-50%)" 
          }}
        />

        {/* 3. EL INPUT (EL "MOTOR") - Debe estar ARRIBA (z-20) y ser transparente */}
        <input
          type="range"
          min={min}
          max={max}
          step={step}
          value={value}
          onChange={handleChange}
          className="absolute w-full h-full appearance-none bg-transparent cursor-pointer z-20
            [&::-webkit-slider-thumb]:appearance-none 
            [&::-webkit-slider-thumb]:w-6 
            [&::-webkit-slider-thumb]:h-6 
            [&::-webkit-slider-thumb]:rounded-full 
            [&::-webkit-slider-thumb]:bg-transparent
            
            [&::-moz-range-thumb]:w-6
            [&::-moz-range-thumb]:h-6
            [&::-moz-range-thumb]:bg-transparent
            [&::-moz-range-thumb]:border-none
          "
        />
      </div>
    </div>
  );
});

DSPSlider.displayName = "DSPSlider";