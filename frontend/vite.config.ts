import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'
import IstanbulPlugin from 'vite-plugin-istanbul'
import { VitePWA } from 'vite-plugin-pwa'

const coverage = process.env.E2E_COVERAGE === '1'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    VitePWA({
      registerType: 'autoUpdate',
      manifest: false,
      // Do not inject registerSW.js. A self-destroying worker still emits
      // /sw.js so an older caching worker can update and retire; registering
      // it on every load calls client.navigate() and aborts in-flight unlock.
      injectRegister: false,
      selfDestroying: true,
    }),
    ...(coverage
      ? [
          IstanbulPlugin({
            include: 'src/*',
            exclude: ['node_modules', 'e2e', 'src/api/types.ts'],
            extension: ['.js', '.ts', '.vue'],
            requireEnv: false,
            cypress: false,
            checkProd: false,
            forceBuildInstrument: true,
          }),
        ]
      : []),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    strictPort: false,
    proxy: {
      '/api': {
        target: process.env.GN_DRIVE_DEV_PORT
          ? `http://127.0.0.1:${process.env.GN_DRIVE_DEV_PORT}`
          : 'http://127.0.0.1:53241',
        changeOrigin: false,
        ws: false,
        // SSE (/api/v1/events) must not be buffered by the dev proxy.
        timeout: 0,
        proxyTimeout: 0,
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: true,
  },
})
