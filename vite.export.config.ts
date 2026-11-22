import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'
import { mdiWoff2Only } from './vite.config'

/**
 * 「HTML で出力」のガワを作る第2ビルド。
 *
 * 本体ビルド（vite.config.ts）とは**config を分ける**。同じビルドの2エントリにすると、
 * VitePWA の injectRegister が出力側の HTML にも Service Worker 登録を注いでしまい、
 * 「出力した HTML は通信をしない」という約束を破る（VitePWA は command で分岐していないので、
 * プラグイン配列を共有している限り必ず入る）。
 *
 * ここで作るのは `dist_export/export.html` と `dist_export/assets/*`。
 * これを1枚の HTML へ畳むのは src/tools/build_export_template.mjs の担当で、
 * 外部参照が1つでも残っていればビルドを落とす。
 */
export default defineConfig(() => {
  // 本体ビルドと同じ既定。デバッグで読める JS が欲しいときだけ MINIFY=false
  const minify = (process.env.MINIFY ?? 'true') === 'true'
  return {
    build: {
      minify: minify,
      outDir: 'dist_export',
      emptyOutDir: true,
      // CSS を1枚に寄せる。コンポーネントごとに割れると、畳む側が
      // 「どの <link> を拾えばよいか」をチャンクグラフから読まないといけなくなる
      cssCodeSplit: false,
      rollupOptions: {
        input: fileURLToPath(new URL('./export.html', import.meta.url)),
        output: {
          // 動的 import も含めて1チャンクへ畳む。エントリが1本なので成立する
          // （本体ビルドでは成立しない）。これが効かないと、出力 HTML が
          // 別チャンクを fetch しようとして通信ゼロの約束を破る。
          // 旧 inlineDynamicImports は Vite 8（rolldown）で非推奨
          codeSplitting: false,
        },
      },
    },
    plugins: [
      mdiWoff2Only(),
      vue({
        script: {
          propsDestructure: true,
        },
      }),
      vuetify({ autoImport: true }),
      // VitePWA は入れない（上のコメントの理由）。vueDevTools も入れない
    ],
    resolve: {
      alias: [
        // i18n は「出力時の1言語だけ」を持つ版へ差し替える。本体の i18n.ts は
        // ja を静的・他6言語を動的 import しており、inlineDynamicImports で畳むと
        // 7言語ぶんのメッセージがガワに常駐する（数百KB級。vite.config.ts の
        // precacheGlobIgnores のコメントが同じ理由でロケールチャンクを除外している）
        {
          find: /^@\/i18n$/,
          replacement: fileURLToPath(new URL('./src/client/export-i18n.ts', import.meta.url)),
        },
        { find: '@', replacement: fileURLToPath(new URL('./src/client', import.meta.url)) },
      ],
    },
    define: {
      __VUE_PROD_DEVTOOLS__: false,
    },
  }
})
