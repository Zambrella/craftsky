import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import process from 'node:process'

export default defineConfig({
  base: '/admin/',
  plugins: [react()],
  server: { port: 5173, strictPort: true, proxy: Object.fromEntries(['/v1/admin', '/oauth/admin'].map(path => [path, {
    target: process.env.APPVIEW_ORIGIN ?? 'http://127.0.0.1:18080',
    changeOrigin: true,
    configure(proxy) {
      proxy.on('proxyReq', (outgoing, incoming) => {
        // Only the exact local Vite origin is translated. Cross-origin requests
        // keep their Origin and are rejected by AppView's normal CSRF boundary.
        if (incoming.headers.origin === 'http://127.0.0.1:5173') outgoing.setHeader('Origin', process.env.APPVIEW_ORIGIN ?? 'http://127.0.0.1:18080')
      })
    },
  }])) },
})
