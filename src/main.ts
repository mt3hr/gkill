import { createApp } from "vue"
import App from "./App.vue"
import router from "./router"
import vuetify from "./plugins/vuetify"
import '@mdi/font/css/materialdesignicons.css'
import { registerSW } from 'virtual:pwa-register'
import { createI18n } from "vue-i18n"

import ja from './locales/ja.json'
import original from './locales/original.json'

const i18n = createI18n({
    legacy: false,
    locale: 'ja',
    fallbackLocale: 'original',
    messages: {
        original,
        ja,
    }
})

registerSW()
const app = createApp(App)
app.use(i18n)
app.use(router)
app.use(vuetify)
app.mount("#app")