import { createApp } from "vue"
import App from "./App.vue"
import router from "./router"
import vuetify from "./plugins/vuetify"
import '@mdi/font/css/materialdesignicons.css'
import { registerSW } from 'virtual:pwa-register'

registerSW()
const app = createApp(App)
app.use(router)
app.use(vuetify)
app.mount("#app")