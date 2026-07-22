import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import { VitePWA } from 'vite-plugin-pwa'
import package_json from './package.json'

// https://vitejs.dev/config/
export default defineConfig(() => {
  const minify = (process.env.MINIFY ?? 'false') === 'true'
  // GKILL_API_PROXY_TARGET でproxy先を上書きできる
  // (`npm run dev -- --api=<url>` がこれを設定する。E2Eが空きポートで立てたサーバ向けにも使う)
  const api_target = process.env.GKILL_API_PROXY_TARGET ?? 'http://localhost:9999'
  return {
    build: {
      minify: minify,
    },
    plugins: [
      vue({
        script: {
          propsDestructure: true,
        },
      }),
      ...(process.env.NODE_ENV !== 'production' ? [vueDevTools()] : []),
      VitePWA({
        registerType: 'autoUpdate',
        injectRegister: 'auto',
        strategies: 'injectManifest',
        srcDir: 'src/client',
        filename: 'serviceWorker.ts',
        manifest: {
          version: package_json.version,
          icons: [{
            src: "favicon.png",
            sizes: "144x144",
            type: "image/png",
            purpose: "any"
          }],
          share_target: {
            action: '/share-target',
            method: 'POST',
            enctype: 'multipart/form-data',
            params: {
              title: 'title',
              text: 'text',
              url: 'url'
            }
          }
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        } as any,
        injectManifest: {
          maximumFileSizeToCacheInBytes: 10 * 1024 ** 2,
        },
        workbox: {
          maximumFileSizeToCacheInBytes: 10 * 1024 ** 2,
        },
      }),
    ],
    server: {
      // gkill_serverが配信する非SPAパスをまとめてproxyする。
      // SPAルート (/rykv, /kftl, /mi など) はvue-routerが処理するのでproxyしない
      proxy: {
        '/api': api_target,
        '/files': api_target, // IDFファイル・サムネイル
        '/zip_cache': api_target, // ZIP展開キャッシュ
        '/resources/manual': api_target, // ヘルプHTML
      },
    },
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src/client', import.meta.url))
      }
    },
    define: {
      __VUE_PROD_DEVTOOLS__: true,
    },
  }
})
