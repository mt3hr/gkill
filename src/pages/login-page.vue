<template>
    <v-app-bar :height="app_title_bar_height" class="app_bar" app color="primary" flat >
        <v-toolbar-title>gkill ログイン</v-toolbar-title>
        <v-spacer />
        <span class="gkill_version">version: {{ gkill_version }}</span>
    </v-app-bar>
    <v-main class="main">
        <LoginView :gkill_api="gkill_api" :app_content_height="app_content_height"
            :app_content_width="app_content_width" @received_errors="write_errors" @received_messages="write_messages"
            @successed_login="handle_success_login" />
    </v-main>
</template>

<script lang="ts" setup>
import { computed, ref, type Ref } from 'vue'
import { GkillAPI } from '@/classes/api/gkill-api'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import LoginView from './views/login-view.vue'
import package_json from '../../package.json'

const actual_height: Ref<Number> = ref(0)
const element_height: Ref<Number> = ref(0)
const browser_url_bar_height: Ref<Number> = ref(0)
const app_title_bar_height: Ref<Number> = ref(50)
const app_title_bar_height_px = computed(() => app_title_bar_height.value.toString().concat("px"))
const gkill_api: Ref<GkillAPI> = ref(GkillAPI.get_instance())
const app_content_height: Ref<Number> = ref(0)
const app_content_width: Ref<Number> = ref(0)
const gkill_version: Ref<string> = ref(package_json.version)

async function resize_content(): Promise<void> {
    const inner_element = document.querySelector('#control-height')
    actual_height.value = window.innerHeight
    element_height.value = inner_element ? inner_element.clientHeight : actual_height.value
    browser_url_bar_height.value = Number(element_height.value) - Number(actual_height.value)
    app_content_height.value = Number(element_height.value) - (Number(browser_url_bar_height.value) + Number(app_title_bar_height.value))
    app_content_width.value = window.innerWidth
}

async function write_errors(errors: Array<GkillError>) {
    //TODO エラーメッセージを画面に出力するように
    errors.forEach(error => {
        console.log(error)
    })
}

async function write_messages(messages: Array<GkillMessage>) {
    //TODO メッセージを画面に出力するように
    messages.forEach(message => {
        console.log(message)
    })
}

async function handle_success_login(session_id: string): Promise<void> {
    //TODO webStorageにセッションIDを入れる
    //TODO webStorageの構造モデル起こし
    //TODO rykv画面に繊維
}

window.addEventListener('resize', () => {
    resize_content()
})


resize_content()
</script>

<style lang="css">
html {
    overflow-y: hidden !important;
}
</style>
<style lang="css" scoped>
.main {
    height: calc(100vh - v-bind(app_title_bar_height_px));
    padding-top: v-bind(app_title_bar_height_px);
    top: v-bind(app_title_bar_height_px)
}

.gkill_version {
    font-size: small;
    margin-right: 15px;
}
</style>