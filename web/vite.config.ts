import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// В dev Vite отдаёт фронтенд, а /api проксирует на запущенный tasky
// (по умолчанию 127.0.0.1:9110; переопределить TASKY_API_URL).
const apiTarget = process.env.TASKY_API_URL || 'http://127.0.0.1:9110'

export default defineConfig({
  base: './',
  plugins: [react()],
  server: {
    proxy: {
      '/api': { target: apiTarget, changeOrigin: true },
    },
  },
  build: {
    outDir: 'dist',
  },
})
