import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const backend = env.VITE_BACKEND_URL || 'http://localhost:8080'
  return {
    plugins: [vue()],
    server: {
      proxy: {
        '/api': { target: backend, changeOrigin: true, ws: true },
      },
    },
  }
})
