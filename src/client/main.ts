import { createApp } from "vue"
import App from "./App.vue"
import router from "./router"
import vuetify from "./plugins/vuetify"
import '@mdi/font/css/materialdesignicons.css'
import { registerSW } from 'virtual:pwa-register'
import { i18n } from "./i18n"
import { v_long_press } from "./classes/long-press"
import { GkillAPI } from "./classes/api/gkill-api"
import { onUnhandledRejection, onVueError, onWindowError } from "./classes/global-exception-feed"

registerSW()

// 握られなかった例外は画面右上のフィードにも出す（コンソールには従来どおり出る）。
// 判定と配線は classes/global-exception-feed.ts（テストから import できる形）にある
window.addEventListener("unhandledrejection", onUnhandledRejection)
window.addEventListener("error", onWindowError)

// ロケールのメッセージ(ja以外)は動的importなので、mount前にロードを終わらせる。
// App.vue の setup 内で呼ぶと、ロード完了までの間だけ日本語が表示されてしまう
await GkillAPI.get_gkill_api().apply_locale()

const app = createApp(App)
// コンポーネントの中で投げられた例外。errorHandler を置くと Vue 自身の console 出力が止まるので、ここで出し直す
app.config.errorHandler = (err, _instance, info) => onVueError(err, info)
app.directive('long-press', v_long_press)
app.use(i18n)
app.use(router)
app.use(vuetify)
app.mount("#app")