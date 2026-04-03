// @ts-check
import { defineConfig } from 'astro/config';

import react from '@astrojs/react';

// https://astro.build/config

export default defineConfig({
  integrations: [react()],
  output: "static",
  adapter: undefined, // Astro dev server
  vite: {
    build: {
      outDir: "../build", // Wails expects frontend bundle here
      emptyOutDir: true,
    },
    server: {
      port: 34115,
      strictPort: true,
    },
    plugins: []
  },
});