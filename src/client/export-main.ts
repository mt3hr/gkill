// 「HTML で出力」したファイルのエントリ。本体の main.ts とは別物で、
// vite.export.config.ts だけがこれをビルドする。
//
// **本体の main.ts から写すときに持ち込んではいけないもの:**
//   - router（単独ファイルなのでルートが無い）。**import しただけで壊れる。**
//     `@/router` は module 評価時に createRouter(createWebHistory(BASE_URL)) を走らせ、
//     createWebHistory がその場で history.replaceState を URL 付きで呼ぶ。
//     file:// で開くとそれが file:///C:/C:/… に化け、画面が出る前にブラウザの
//     「ファイルにアクセスできませんでした」になる（export-router-stub.ts に詳しい）。
//     ここから import していなくても、画面側（use-rykv-view.ts など）が import しているので、
//     **vite.export.config.ts の alias で差し替えるのが唯一の防御**
//   - registerSW / virtual:pwa-register（Service Worker を取りに行く = 通信する）
//   - GkillAPI.apply_locale()（Cookie を読んでロケールのメッセージを動的 import する。
//     出力 HTML は焼き込まれた1言語だけを持つので、use-exported-rykv-page.ts が
//     焼き込みから当てる）
//
// 握られなかった例外のフィードは本体と同じものを使う（通信はしない）。

import { createApp } from "vue"
// gkill-api を**画面より先に**評価する。データクラス（InfoBase 系）と req_res は
// gkill-api と循環 import になっており、評価順が逆になると
// `class extends value undefined` で mount の前に落ちる。
// 単一チャンクへ畳む出力ビルドでは、エントリの import 順がそのまま評価順になる
// （__tests__/unit/classes/mermaid-render.test.ts も同じ理由で先に import している）
import "./classes/api/gkill-api"
import ExportedRykvPage from "./pages/exported-rykv-page.vue"
import vuetify from "./plugins/vuetify"
// gkill のグローバル CSS。本体は App.vue が持ち込むが、出力 HTML は App.vue を
// 使わないので**ここで明示的に import する**。これが無いと画面の高さ・
// Kyou の画像サイズ・Lantana のサイズ・ダイアログの見た目が全部効かない
import '@/styles/gkill-global.css'
import '@mdi/font/css/materialdesignicons.css'
import { i18n } from "./export-i18n"
import { v_long_press } from "./classes/long-press"
import { onUnhandledRejection, onVueError, onWindowError } from "./classes/global-exception-feed"

window.addEventListener("unhandledrejection", onUnhandledRejection)
window.addEventListener("error", onWindowError)

const app = createApp(ExportedRykvPage)
app.config.errorHandler = (err, _instance, info) => onVueError(err, info)
app.directive('long-press', v_long_press)
app.use(i18n)
app.use(vuetify)
app.mount("#app")
