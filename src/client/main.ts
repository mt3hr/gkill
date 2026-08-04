import { createApp } from "vue"
import App from "./App.vue"
import router from "./router"
import vuetify from "./plugins/vuetify"
import '@mdi/font/css/materialdesignicons.css'
import { registerSW } from 'virtual:pwa-register'
import { i18n } from "./i18n"
import { vLongPress } from "./classes/long-press"
import { GkillAPI } from "./classes/api/gkill-api"

registerSW()

// abort系エラーの判定。unhandledrejectionでabortを握りつぶすために使う
export function isAbortError(err: unknown): boolean {
  if (!err) return false
  if (err instanceof Error && err.name === "AbortError") return true

  const msg = String((err instanceof Error ? err.message : err) ?? "")
  return (
    msg.includes("signal is aborted without reason") ||
    msg.includes("user aborted a request") ||
    msg.includes("The operation was aborted")
  )
}

window.addEventListener("unhandledrejection", (event) => {
    if (isAbortError(event.reason)) {
        event.preventDefault()
    }
})

// ロケールのメッセージ(ja以外)は動的importなので、mount前にロードを終わらせる。
// App.vue の setup 内で呼ぶと、ロード完了までの間だけ日本語が表示されてしまう
await GkillAPI.get_gkill_api().apply_locale()

const app = createApp(App)
app.directive('long-press', vLongPress)
app.use(i18n)
app.use(router)
app.use(vuetify)
app.mount("#app")