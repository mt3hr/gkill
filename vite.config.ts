import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import { VitePWA } from 'vite-plugin-pwa'

// https://vitejs.dev/config/
export default defineConfig({
  build: {
    minify: false,
  },
  plugins: [
    vue({
      script: {
        propsDestructure: true,
      }
    }),
    vueDevTools(),
    VitePWA({
      registerType: 'autoUpdate',
      injectRegister: 'auto',
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'serviceWorker.ts',
      manifest: {
        icons: [{
          src: "favicon.png",
          sizes: "144x144",
          type: "image/png",
          purpose: "any"
        }]
      },
      injectManifest: {
        maximumFileSizeToCacheInBytes: 5 * 1024 ** 2,
      },
      workbox: {
        maximumFileSizeToCacheInBytes: 5 * 1024 ** 2,
      },
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  }
})
