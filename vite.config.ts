import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import { VitePWA } from 'vite-plugin-pwa'
import vuetify from 'vite-plugin-vuetify'
import type { Plugin } from 'vite'
import package_json from './package.json'

/**
 * Material Design Icons のWebフォントを woff2 のみに絞る。
 *
 * @mdi/font の CSS は @font-face で eot/woff2/woff/ttf の4形式を列挙しているため、
 * 何もしないと Vite が4ファイル全部 (計3.6MB) を出力してしまう。
 * gkill の対象は PWA / WebView (Android minSdk 26) なので woff2 非対応環境は無い。
 * url() の参照を消せば Vite はそのアセットを出力しない。
 */
function mdiWoff2Only(): Plugin {
  return {
    name: 'gkill-mdi-woff2-only',
    enforce: 'pre',
    transform: {
      // filter を付けないと全モジュール (3900件超) でフック呼び出しが発生してビルドが遅くなる
      filter: { id: '**/@mdi/font/**/*.css' },
      handler(code: string) {
        const replaced = code
          // 単独の eot 指定 (IE互換用の1行目)
          .replace(/src:\s*url\([^)]*\.eot[^)]*\);/g, '')
          // src リスト内の eot / woff / ttf エントリ
          .replace(/url\([^)]*\.eot[^)]*\)\s*format\(["']embedded-opentype["']\),?\s*/g, '')
          .replace(/,?\s*url\([^)]*\.woff\?[^)]*\)\s*format\(["']woff["']\)/g, '')
          .replace(/,?\s*url\([^)]*\.ttf[^)]*\)\s*format\(["']truetype["']\)/g, '')
        return replaced === code ? null : { code: replaced, map: null }
      },
    },
  }
}

/**
 * PWA の precache から除外するチャンク。
 *
 * mermaid は classes/mermaid-render.ts で動的 import しており、mermaid記法を含む記録を
 * 開いたときだけ必要。precache に入れると全ユーザがインストール時に3.3MB落とすことになる。
 */
const precacheGlobIgnores = [
  '**/node_modules/**/*',
  'assets/*Diagram-*.js',
  'assets/*diagram-*.js',
  'assets/diagram-*.js',
  'assets/chunk-*.js',
  'assets/cytoscape.esm-*.js',
  'assets/katex-*.js',
  'assets/dagre-*.js',
  'assets/graphlib-*.js',
  'assets/rough.esm-*.js',
  'assets/mermaid*.js',
  'assets/*-definition-*.js',
  'assets/swimlanes-*.js',
  'assets/cose-bilkent-*.js',
  'assets/purify.es-*.js',
  'assets/marked.esm-*.js',
  // ja以外のロケール (i18n.ts で動的import)。使う1言語だけランタイムキャッシュされればよい
  'assets/en-*.js',
  'assets/zh-*.js',
  'assets/ko-*.js',
  'assets/es-*.js',
  'assets/fr-*.js',
  'assets/de-*.js',
]

// https://vitejs.dev/config/
export default defineConfig(() => {
  // minifyは既定で有効。デバッグで読めるJSが欲しいときだけ MINIFY=false を指定する。
  // 既定offだと install_server / install_app (= prepare_install -> build) が
  // 未minifyの5MB近いJSをそのままGoバイナリに埋め込んでしまう。
  const minify = (process.env.MINIFY ?? 'true') === 'true'
  // GKILL_API_PROXY_TARGET でproxy先を上書きできる
  // (`npm run dev -- --api=<url>` がこれを設定する。E2Eが空きポートで立てたサーバ向けにも使う)
  const api_target = process.env.GKILL_API_PROXY_TARGET ?? 'http://localhost:9999'
  return {
    build: {
      minify: minify,
    },
    plugins: [
      mdiWoff2Only(),
      vue({
        script: {
          propsDestructure: true,
        },
      }),
      // テンプレートで実際に使われている Vuetify コンポーネントだけを import する。
      // これが無いと plugins/vuetify.ts の一括登録で全コンポーネント + 全CSSがバンドルされる
      vuetify({ autoImport: true }),
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
          globIgnores: precacheGlobIgnores,
        },
        workbox: {
          maximumFileSizeToCacheInBytes: 10 * 1024 ** 2,
          globIgnores: precacheGlobIgnores,
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
