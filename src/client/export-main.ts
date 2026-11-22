// 「HTML で出力」したファイルのエントリ。本体の main.ts とは別物で、
// vite.export.config.ts だけがこれをビルドする。
//
// **本体の main.ts から写すときに持ち込んではいけないもの:**
//   - router（単独ファイルなのでルートが無い。vue-router を入れると @vue/devtools-api も付いてくる）
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
