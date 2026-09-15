import { createApp } from "vue"
import App from "./App.vue"
import router from "./router"
import vuetify from "./plugins/vuetify"
import '@mdi/font/css/materialdesignicons.css'
import { registerSW } from 'virtual:pwa-register'
import { i18n } from "./i18n"
import { v_long_press } from "./classes/long-press"
import { GkillAPI } from "./classes/api/gkill-api"
import { push_client_exception } from "./classes/use-gkill-message-feed"

registerSW()

// abort系エラーの判定。unhandledrejectionでabortを握りつぶすために使う
export function is_abort_error(err: unknown): boolean {
  if (!err) return false
  if (err instanceof Error && err.name === "AbortError") return true

  const msg = String((err instanceof Error ? err.message : err) ?? "")
  return (
    msg.includes("signal is aborted without reason") ||
    msg.includes("user aborted a request") ||
    msg.includes("The operation was aborted")
  )
}

// 握られなかった例外は画面右上のフィードにも出す（コンソールには従来どおり出る）。
// 以前はコンソールにしか出ず、利用者には「押しても何も起きない」にしか見えなかった。
window.addEventListener("unhandledrejection", (event) => {
    if (is_abort_error(event.reason)) {
        event.preventDefault()
        return
    }
    push_client_exception(event.reason)
})

window.addEventListener("error", (event) => {
    // リソース読込失敗（error が無い）と ResizeObserver の無害な通知は出さない
    if (!event.error || is_resize_observer_loop_notice(event.message)) {
        return
    }
    push_client_exception(event.error)
})

export function is_resize_observer_loop_notice(message: unknown): boolean {
    return typeof message === "string" && message.includes("ResizeObserver loop")
}

// ロケールのメッセージ(ja以外)は動的importなので、mount前にロードを終わらせる。
// App.vue の setup 内で呼ぶと、ロード完了までの間だけ日本語が表示されてしまう
await GkillAPI.get_gkill_api().apply_locale()

const app = createApp(App)
// コンポーネントの中で投げられた例外。errorHandler を置くと Vue 自身の console 出力が止まるので、ここで出し直す
app.config.errorHandler = (err, _instance, info) => {
    console.error(err, info)
    push_client_exception(err)
}
app.directive('long-press', v_long_press)
app.use(i18n)
app.use(router)
app.use(vuetify)
app.mount("#app")