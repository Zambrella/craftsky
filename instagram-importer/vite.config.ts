import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '127.0.0.1',
    warmup: {
      clientFiles: [
        './src/main.tsx',
        './src/worker/client.ts',
        './src/worker/archive.worker.ts',
      ],
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    target: 'es2022',
  },
  worker: {
    format: 'es',
  },
})
