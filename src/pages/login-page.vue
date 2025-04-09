<template>
    <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" app color="primary" flat>
        <v-toolbar-title>{{ $t("LOGIN_TITLE") }}</v-toolbar-title>
        <v-spacer />
        <span class="gkill_version">{{ $t("VERSION_TITLE") }}: {{ gkill_version }}</span>
    </v-app-bar>
    <v-main class="main">
        <LoginView :gkill_api="gkill_api" :app_content_height="app_content_height"
            :app_content_width="app_content_width" @received_errors="write_errors" @received_messages="write_messages"
            @successed_login="handle_success_login" />
        <div class="alert_container">
            <v-slide-y-transition group>
                <v-alert v-for="message in messages" theme="dark" :key="message.id">
                    {{ message.message }}
                </v-alert>
            </v-slide-y-transition>
        </div>
    </v-main>
</template>

<script lang="ts" setup>
import { computed, ref, type Ref } from 'vue'
import { GkillAPI } from '@/classes/api/gkill-api'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import LoginView from './views/login-view.vue'
import package_json from '../../package.json'
import router from '@/router'
import { useI18n } from 'vue-i18n'

import { i18n } from '@/i18n'
const actual_height: Ref<Number> = ref(0)
const element_height: Ref<Number> = ref(0)
const browser_url_bar_height: Ref<Number> = ref(0)
const app_title_bar_height: Ref<Number> = ref(50)
const app_title_bar_height_px = computed(() => app_title_bar_height.value.toString().concat("px"))
const gkill_api = computed(() => GkillAPI.get_instance())
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

async function handle_success_login(session_id: string): Promise<void> {
    GkillAPI.get_instance().set_session_id(session_id)
    await sleep(1500)

    // ログインに成功したらrykv画面に遷移
    router.replace('rykv')
}

const messages: Ref<Array<{ message: string, id: string, show_snackbar: boolean }>> = ref([])

async function write_errors(errors: Array<GkillError>) {
    const received_messages = new Array<{ message: string, id: string, show_snackbar: boolean }>()
    for (let i = 0; i < errors.length; i++) {
        received_messages.push({
            message: errors[i].error_message,
            id: GkillAPI.get_instance().generate_uuid(),
            show_snackbar: true,
        })
    }
    messages.value.push(...received_messages)
    sleep(2500).then(() => {
        for (let i = 0; i < received_messages.length; i++) {
            messages.value.splice(0, 1)
        }
    })
}

async function write_messages(messages_: Array<GkillMessage>) {
    const received_messages = new Array<{ message: string, id: string, show_snackbar: boolean }>()
    for (let i = 0; i < messages_.length; i++) {
        received_messages.push({
            message: messages_[i].message,
            id: GkillAPI.get_instance().generate_uuid(),
            show_snackbar: true,
        })
    }
    messages.value.push(...received_messages)
    sleep(2500).then(() => {
        for (let i = 0; i < received_messages.length; i++) {
            messages.value.splice(0, 1)
        }
    })
}

const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

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

.alert_container {
    position: fixed;
    top: 60px;
    right: 10px;
    display: grid;
    grid-gap: .5em;
    z-index: 99;
}
</style>