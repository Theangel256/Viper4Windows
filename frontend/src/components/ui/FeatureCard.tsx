import { DSPToggle } from "./DSPToggle";

interface FeatureCardProps {
  title:     string;
  value:     boolean;
  onToggle?: (on: boolean) => void;
  children:  React.ReactNode;
  className?: string;
}

export function FeatureCard({ title, value, onToggle, children, className = "" }: FeatureCardProps) {
  return (
    <div
      className={`
        bg-white dark:bg-zinc-900
        border border-zinc-200 dark:border-zinc-800
        rounded-2xl px-4 py-3.5 flex flex-col gap-2.5
        shadow-card
        ${className}
      `}
    >
      <div className="flex items-center justify-between">
        <span className="text-[14px] font-semibold text-zinc-800 dark:text-zinc-100 tracking-tight">
          {title}
        </span>
        <DSPToggle value={value} onChange={onToggle} />
      </div>
      <div className="flex flex-col gap-2.5">{children}</div>
    </div>
  );
}
