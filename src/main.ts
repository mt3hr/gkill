import { createApp } from "vue"
import App from "./App.vue"
import router from "./router"
import vuetify from "./plugins/vuetify"
import '@mdi/font/css/materialdesignicons.css'
import { registerSW } from 'virtual:pwa-register'
import { i18n } from "./i18n"
import { vLongPress } from "./classes/long-press"
import { isAbortError } from "@/utils/isAbortError"

registerSW()

/* abortは握りつぶす */
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