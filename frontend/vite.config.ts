import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('plotly')) return 'plotly.min'
        },
        chunkFileNames(chunkInfo) {
          if (chunkInfo.name === 'plotly.min') return 'assets/plotly.min.js'
          return 'assets/[name]-[hash].js'
        },
      },
    },
  },
})