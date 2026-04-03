interface DSPToggleProps {
  value:     boolean;
  onChange?: (on: boolean) => void;
  size?:     "sm" | "md";
}

export function DSPToggle({ value, onChange, size = "md" }: DSPToggleProps) {
  const sizes = {
    sm: { track: "w-8 h-4",  thumb: "w-3 h-3",  on: "translate-x-4"  },
    md: { track: "w-11 h-6", thumb: "w-5 h-5",  on: "translate-x-5"  },
  };
  const s = sizes[size];

  return (
    <button
      role="switch"
      aria-checked={value}
      onClick={() => onChange?.(!value)}
      className={`
        relative inline-flex items-center shrink-0 rounded-full
        transition-colors duration-200 ease-in-out
        focus:outline-none focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2
        ${s.track}
        ${value ? "bg-red-500" : "bg-zinc-300 dark:bg-zinc-600"}
      `}
    >
      <span
        className={`
          inline-block rounded-full bg-white shadow-sm
          transform transition-transform duration-200 ease-in-out
          ${s.thumb}
          ${value ? s.on : "translate-x-0.5"}
        `}
      />
    </button>
  );
}
