import { createApp } from "vue"
import App from "./App.vue"
import router from "./router"
import vuetify from "./plugins/vuetify"
import '@mdi/font/css/materialdesignicons.css'
import { registerSW } from 'virtual:pwa-register'
import { i18n } from "./i18n"
import { vLongPress } from "./classes/long-press"

registerSW()

/* abortは握りつぶす */
// src/utils/isAbortError.ts (例)
export function isAbortError(err: any): boolean {
  if (!err) return false
  if (err.name === "AbortError") return true

  const msg = String(err?.message ?? "")
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

const app = createApp(App)
app.directive('long-press', vLongPress)
app.use(i18n)
app.use(router)
app.use(vuetify)
app.mount("#app")