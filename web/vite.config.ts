import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Dev proxy forwards API calls to the locally running panel-api so the SPA can
// be developed against real endpoints without CORS setup.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/healthz': 'http://127.0.0.1:8080',
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
  },
})
