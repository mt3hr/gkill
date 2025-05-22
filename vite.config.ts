import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import { VitePWA } from 'vite-plugin-pwa'
import package_json from './package.json'

// https://vitejs.dev/config/
export default defineConfig({
  build: {
    minify: false,
  },
  plugins: [
    vue({
      script: {
        propsDestructure: true,
      },
    }),
    vueDevTools(),
    VitePWA({
      registerType: 'autoUpdate',
      injectRegister: 'auto',
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'serviceWorker.ts',
      manifest: {
        version: package_json.version,
        icons: [{
          src: "favicon.png",
          sizes: "144x144",
          type: "image/png",
          purpose: "any"
        }]
      } as any,
      injectManifest: {
        maximumFileSizeToCacheInBytes: 5 * 1024 ** 2,
      },
      workbox: {
        maximumFileSizeToCacheInBytes: 5 * 1024 ** 2,
        runtimeCaching: [
          {
            // eslint-disable-next-line no-useless-escape
            urlPattern: "^.*\/api\/get_gkill_info$",
            handler: 'StaleWhileRevalidate',
          },
          {
            // eslint-disable-next-line no-useless-escape
            urlPattern: "^.*\/api\/get_all_rep_names$",
            handler: 'StaleWhileRevalidate',
          },
          {
            // eslint-disable-next-line no-useless-escape
            urlPattern: "^.*\/api\/get_all_tag_names$",
            handler: 'StaleWhileRevalidate',
          },
          {
            // eslint-disable-next-line no-useless-escape
            urlPattern: "^.*\/api\/get_mi_board_list$",
            handler: 'StaleWhileRevalidate',
          }
        ]
      },
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  define: {
    __VUE_PROD_DEVTOOLS__: true,
  },
})
