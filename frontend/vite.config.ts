import { readFileSync } from 'node:fs'
import { loadEnv } from 'vite'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

// package.json is the single source of truth for the released version; the
// About page reads it back through import.meta.env.VITE_APP_VERSION.
const pkg = JSON.parse(readFileSync(new URL('./package.json', import.meta.url), 'utf8')) as { version: string }
const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, repositoryRoot, '')
  const dataDir = resolve(env.SUBFLOW_DATA_DIR || join(repositoryRoot, 'backend', 'pb_data'))
  let localTLS: { key: Buffer; cert: Buffer } | undefined
  if (mode === 'development') {
    try {
      const tlsDir = join(dataDir, '.subflow', 'tls')
      const active = readFileSync(join(tlsDir, 'active'), 'utf8').trim()
      const versionDir = active === 'self-signed' ? join(tlsDir, active) : join(tlsDir, 'versions', active)
      localTLS = { key: readFileSync(join(versionDir, 'key.pem')), cert: readFileSync(join(versionDir, 'cert.pem')) }
    } catch {
      throw new Error('Start SubFlow once to create its local HTTPS certificate before running Vite.')
    }
  }
  const backend = `https://127.0.0.1:${env.SUBFLOW_HTTP_PORT || '8080'}`
  return {
    define: { 'import.meta.env.VITE_APP_VERSION': JSON.stringify(pkg.version) },
    plugins: [
      vue(),
      VitePWA({
        registerType: 'prompt',
        includeAssets: ['favicon.svg', 'favicon.png'],
        manifest: {
          name: 'SubFlow 共享財務管理',
          short_name: 'SubFlow',
          description: 'SubFlow 共享財務管理',
          theme_color: '#7357ff',
          background_color: '#101525',
          display: 'standalone',
          start_url: '/',
          icons: [
            { src: '/pwa-192x192.png', sizes: '192x192', type: 'image/png' },
            { src: '/pwa-512x512.png', sizes: '512x512', type: 'image/png' },
            { src: '/pwa-512x512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
          ],
        },
        workbox: {
          navigateFallback: '/index.html',
          navigateFallbackDenylist: [/^\/api\//, /^\/_\//],
        },
      }),
    ],
    server: {
      https: localTLS,
      proxy: {
        // Vite is a local-only development proxy; the backend is HTTPS and
        // uses its generated certificate unless a development certificate is installed.
        '/api': { target: backend, changeOrigin: true, ws: true, secure: false },
        '/_': { target: backend, changeOrigin: true, ws: true, secure: false },
      },
    },
    test: {
      exclude: ['e2e/**', 'node_modules/**', 'dist/**'],
      coverage: {
        provider: 'v8',
        reporter: ['text', 'lcov'],
      },
    },
  }
})

