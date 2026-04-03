/** @type {import('tailwindcss').Config} */
export default {
    // 1. IMPORTANTE: Astro usa archivos .astro. Si no los añades, 
    // las clases en tus layouts no funcionarán.
    content: [
      "./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}",
    ],
    darkMode: "class",
    theme: {
      container: {
        center: true,
        padding: "2rem",
        screens: { "2xl": "1400px" },
      },
      extend: {
        // 2. Colores compatibles con Shadcn Luma
        colors: {
          border: "hsl(var(--border))",
          input: "hsl(var(--input))",
          ring: "hsl(var(--ring))",
          background: "hsl(var(--background))",
          foreground: "hsl(var(--foreground))",
          primary: {
            DEFAULT: "hsl(var(--primary))",
            foreground: "hsl(var(--primary-foreground))",
          },
          // Tu acento personalizado (Red Luma)
          accent: {
            DEFAULT: "#dc2626",
            50:  "#fef2f2",
            100: "#fee2e2",
            200: "#fecaca",
            400: "#f87171",
            500: "#ef4444",
            600: "#dc2626",
            700: "#b91c1c",
            800: "#991b1b",
            900: "#7f1d1d",
          },
        },
        fontFamily: {
          sans: ["Geist", "Inter", "system-ui", "sans-serif","sf-pro-rounded"],
          mono: ["Geist Mono", "JetBrains Mono", "monospace"],
        },
        borderRadius: {
          lg: "var(--radius)",
          md: "calc(var(--radius) - 2px)",
          sm: "calc(var(--radius) - 4px)",
          "2xl": "1rem",
          "3xl": "1.25rem",
        },
        boxShadow: {
          card: "0 1px 3px 0 rgb(0 0 0 / 0.04), 0 1px 2px -1px rgb(0 0 0 / 0.04)",
          "card-hover": "0 4px 12px 0 rgb(0 0 0 / 0.08)",
        },
      },
    },
    // 3. Shadcn suele requerir tailwindcss-animate
    plugins: [require("tailwindcss-animate")],
  };