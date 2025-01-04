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
      injectRegister: 'auto',
      manifest: {
        icons: [{
          src: "favicon.png",
          sizes: "144x144",
          type: "image/png",
          purpose: "any"
        }]
      },
      workbox: {
        maximumFileSizeToCacheInBytes: 5 * 1024 ** 2, // 5 MB or set to something else
      },
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  }
})
