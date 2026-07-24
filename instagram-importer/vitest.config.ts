import react from '@vitejs/plugin-react'
import { configDefaults, defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    exclude: [...configDefaults.exclude, 'e2e/**'],
    setupFiles: ['./src/test/setup.ts'],
    restoreMocks: true,
    testTimeout: 10_000,
    coverage: {
      reporter: ['text', 'lcov'],
    },
  },
})
