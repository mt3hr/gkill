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
export default defineConfig(({ command }) => {
  // minifyは既定で有効。デバッグで読めるJSが欲しいときだけ MINIFY=false を指定する。
  // 既定offだと install_server / install_app (= prepare_install -> build) が
  // 未minifyの5MB近いJSをそのままGoバイナリに埋め込んでしまう。
  const minify = (process.env.MINIFY ?? 'true') === 'true'
  // GKILL_API_PROXY_TARGET でproxy先を上書きできる
  // (`npm run dev -- --api=<url>` がこれを設定する。E2Eが空きポートで立てたサーバ向けにも使う)
  const api_target = process.env.GKILL_API_PROXY_TARGET ?? 'http://localhost:9999'
  // gkillのTLS証明書は自己署名なので、https宛のproxyは既定 (secure: true) だと
  // 「self-signed certificate」で全リクエストが落ちる。devサーバのproxy限定の設定で、
  // ビルド成果物には影響しない (targetがhttpのときは無視される)
  const api_proxy = { target: api_target, secure: false }
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
      // devサーバのときだけ入れる。判定はViteが渡す command で行う
      // (process.env.NODE_ENV はViteがいつ設定するかに依存するので、
      // 「本番ビルドにdev用プラグインが混ざる」形で静かに壊れうる)
      ...(command === 'serve' ? [vueDevTools()] : []),
      VitePWA({
        registerType: 'autoUpdate',
        injectRegister: 'auto',
        strategies: 'injectManifest',
        srcDir: 'src/client',
        filename: 'serviceWorker.ts',
        manifest: {
          version: package_json.version,
          // 以下3つは vite-plugin-pwa の既定値 ('#42b883' / '#ffffff' / 'en') と
          // マージされる。既定のまま放置すると「ビルドは通るが表示だけ違う」形で出るので、
          // 既定と同じ値になるものも含めて明示的に固定すること。
          //
          // Android Chrome のPWAはステータスバー色をここから取る (index.htmlの
          // <meta name="theme-color"> はstandalone起動では見られない)。
          // 未指定だと既定の '#42b883' (Vueのブランド色) が入り、
          // 画面最上部だけがアプリバー (color="primary") と違う緑になる
          theme_color: '#2672ed',
          // 起動スプラッシュの背景。manifestはテーマ切り替えに追従できないので、
          // 既定テーマ (gkill_theme) の background に合わせた固定値にする。
          // ダークテーマ利用時は起動時だけ白くなるが、これは避けられない
          background_color: '#ffffff',
          // i18n.ts の locale / fallbackLocale と index.html の <html lang> に合わせる
          lang: 'ja',
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
      // **E2E の成果物を監視対象から外すこと。** Playwright は実行中に test-results/ と
      // playwright-report/ へトレース・スクリーンショット・HTML レポートを書く。
      // dev サーバの監視はこれらを拾って開いているページを再読み込みさせるので、
      // 別のワーカーが遷移中だと page.goto が net::ERR_ABORTED で落ちる。
      // しかも失敗が成果物を増やす → 再読み込みが増える → さらに失敗、と連鎖する。
      // .gitignore には入っているが、dev サーバの監視は .gitignore を見ない。
      watch: {
        ignored: ['**/test-results/**', '**/playwright-report/**', '**/release/**'],
      },
      // gkill_serverが配信する非SPAパスをまとめてproxyする。
      // SPAルート (/rykv, /kftl, /mi など) はvue-routerが処理するのでproxyしない
      proxy: {
        '/api': api_proxy,
        '/files': api_proxy, // IDFファイル・サムネイル
        '/zip_cache': api_proxy, // ZIP展開キャッシュ
        '/resources/manual': api_proxy, // ヘルプHTML
      },
    },
    // **動的 import するものはここに挙げること。** mermaid と dompurify は
    // 図を含む Markdown を開いたときだけ読む（mermaid-render.ts の await import）。
    // 事前バンドルから漏れると、最初にその画面を開いた瞬間に dev サーバが
    // 依存を検出し直して再バンドルし、開いている全ページを強制再読み込みする。
    // E2E だと遷移中のワーカーが巻き添えで落ちる。
    optimizeDeps: {
      include: ['mermaid', 'dompurify'],
    },
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src/client', import.meta.url))
      }
    },
    define: {
      // 本番ビルドへ Vue DevTools 連携コードを残すかどうか。true にすると
      // vue-router が同梱する @vue/devtools-api (コンポーネントインスペクタの
      // オーバーレイ一式) がそのまま本番バンドルに載る。
      // デバッグで本番ビルドにDevToolsを繋ぎたいときだけ一時的に true に戻すこと
      __VUE_PROD_DEVTOOLS__: false,
    },
  }
})
