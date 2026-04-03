/**
 * DSPButton — Ghost-outline button, active state uses red border.
 * Props:
 *   children  : ReactNode
 *   active    : bool
 *   onClick   : fn
 *   size      : "sm" | "md" | "lg"
 *   variant   : "default" | "danger" | "ghost"
 *   className : string
 */

// 1. Definimos la "forma" de los datos (Interface)
interface DSPButtonProps {
  children: React.ReactNode;
  onClick?: () => void;      // El '?' lo hace opcional para que no de error
  active?: boolean;          // Opcional
  size?: "sm" | "md" | "lg"; 
  variant?: "default" | "danger" | "ghost";
  className?: string;       // Opcional
  [x: string]: any;         // Permite pasar otras props como 'id', 'style', etc.
}

// 2. Usamos la Interface y asignamos valores por defecto al mismo tiempo
export function DSPButton({
  children,
  active = false,       // Valor por defecto si no se envía
  onClick = () => {},   // Función vacía por defecto si no se envía
  size = "md",
  variant = "default",
  className = "",
  ...props
}: DSPButtonProps) {
  const sizes = {
    sm: "px-2.5 py-1 text-[12px]",
    md: "px-3.5 py-2 text-[13px]",
    lg: "px-5 py-2.5 text-[14px]",
  };

  const variants = {
    default: active
      ? "border-red-400 text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-950/30"
      : "border-zinc-300 dark:border-zinc-700 text-zinc-600 dark:text-zinc-300 hover:border-zinc-400 hover:bg-zinc-50 dark:hover:bg-zinc-800",
    danger:
      "border-red-300 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30",
    ghost:
      "border-transparent text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800",
  };

  return (
    <button
      onClick={onClick}
      className={`
        inline-flex items-center justify-center gap-2 rounded-xl
        border font-medium transition-all duration-150
        focus:outline-none focus-visible:ring-2 focus-visible:ring-red-400
        active:scale-[0.97]
        ${sizes[size]}
        ${variants[variant]}
        ${className}
      `}
      {...props}
    >
      {children}
    </button>
  );
}
