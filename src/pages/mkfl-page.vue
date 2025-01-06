<template>
    <v-overlay v-model="is_loading" class="align-center justify-center" persistent>
        <v-progress-circular indeterminate color="primary" />
    </v-overlay>
    <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
        <v-btn icon="mdi-menu" :ripple="false" link="false" :style="{ opacity: 0, cursor: 'unset', }" />
        <v-toolbar-title>
            <div>
                <span>
                    MKFL
                </span>
                <v-menu activator="parent">
                    <v-list>
                        <v-list-item v-for="page, index in ['rykv', 'mi', 'kftl', 'plaing', 'mkfl', 'saihate']"
                            :key="index" :value="index">
                            <v-list-item-title @click="router.replace('/' + page)">{{ page }}</v-list-item-title>
                        </v-list-item>
                    </v-list>
                </v-menu>
            </div>
        </v-toolbar-title>
        <v-spacer />
        <v-divider vertical />
        <v-btn icon="mdi-cog" @click="is_show_application_config_dialog = true" />
    </v-app-bar>
    <v-main class="main">
        <kftlView :app_content_height="app_content_height.valueOf() / 2" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api" @received_errors="write_errors"
            @deleted_kyou="reload_plaing_timeis_view()" @registered_kyou="reload_plaing_timeis_view()"
            @updated_kyou="reload_plaing_timeis_view()" @received_messages="write_messages"
            @saved_kyou_by_kftl="reload_plaing_timeis_view()" />
        <PlaingTimeisView :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="app_content_height.valueOf() / 2" :app_content_width="app_content_width"
            @received_errors="write_errors" @received_messages="write_messages" ref="plaing_timeis_view" />
        <ApplicationConfigDialog :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="app_content_height" :app_content_width="app_content_width"
            :is_show="is_show_application_config_dialog" @received_errors="write_errors"
            @received_messages="write_messages" @requested_reload_application_config="load_application_config" />
    </v-main>
    <div class="alert_container">
        <v-slide-y-transition group>
            <v-alert v-for="message in messages" theme="dark" :key="message.id">
                {{ message.message }}
            </v-alert>
        </v-slide-y-transition>
    </div>
</template>
<script lang="ts" setup>
import router from '@/router'
import { GkillAPI } from '@/classes/api/gkill-api'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { type Ref, ref, computed, watch, nextTick } from 'vue'
import ApplicationConfigDialog from './dialogs/application-config-dialog.vue'
import kftlView from './views/kftl-view.vue'
import PlaingTimeisView from './views/plaing-timeis-view.vue'
import { GetGkillNotificationPublicKeyRequest } from '@/classes/api/req_res/get-gkill-notification-public-key-request'
import { RegisterGkillNotificationRequest } from '@/classes/api/req_res/register-gkill-notification-request'
const plaing_timeis_view = ref<InstanceType<typeof PlaingTimeisView> | null>(null);

const actual_height: Ref<Number> = ref(0)
const element_height: Ref<Number> = ref(0)
const browser_url_bar_height: Ref<Number> = ref(0)
const app_title_bar_height: Ref<Number> = ref(50)
const gkill_api = computed(() => GkillAPI.get_instance())
const application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
const app_content_height: Ref<Number> = ref(0)
const app_content_width: Ref<Number> = ref(0)

const is_show_application_config_dialog: Ref<boolean> = ref(false)

async function reload_plaing_timeis_view(): Promise<void> {
    plaing_timeis_view.value?.reload_list(false)
}

async function load_application_config(): Promise<void> {
    const req = new GetApplicationConfigRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()

    return gkill_api.value.get_application_config(req)
        .then(res => {
            if (res.errors && res.errors.length != 0) {
                write_errors(res.errors)
                return
            }

            application_config.value = res.application_config
            GkillAPI.get_instance().set_saved_application_config(res.application_config)

            if (res.messages && res.messages.length != 0) {
                write_messages(res.messages)
                return
            }
        })
}

async function resize_content(): Promise<void> {
    const inner_element = document.querySelector('#control-height')
    actual_height.value = window.innerHeight
    element_height.value = inner_element ? inner_element.clientHeight : actual_height.value
    browser_url_bar_height.value = Number(element_height.value) - Number(actual_height.value)
    app_content_height.value = Number(element_height.value) - (Number(browser_url_bar_height.value) + Number(app_title_bar_height.value))
    app_content_width.value = window.innerWidth
}

const messages: Ref<Array<{ message: string, id: string, show_snackbar: boolean }>> = ref([])

async function write_errors(errors: Array<GkillError>) {
    const received_messages = new Array<{ message: string, id: string, show_snackbar: boolean }>()
    for (let i = 0; i < errors.length; i++) {
        if (errors[i] && errors[i].error_message) {
            received_messages.push({
                message: errors[i].error_message,
                id: GkillAPI.get_instance().generate_uuid(),
                show_snackbar: true,
            })
        }
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
        if (messages_[i] && messages_[i].message) {
            received_messages.push({
                message: messages_[i].message,
                id: GkillAPI.get_instance().generate_uuid(),
                show_snackbar: true,
            })
        }
    }
    messages.value.push(...received_messages)
    sleep(2500).then(() => {
        for (let i = 0; i < received_messages.length; i++) {
            messages.value.splice(0, 1)
        }
    })
}

const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

const is_loading = ref(true)
watch(() => application_config.value, () => {
    is_loading.value = false
})

window.addEventListener('resize', () => {
    resize_content()
})

resize_content()
load_application_config()

// プッシュ通知登録用
async function subscribe(vapidPublicKey: string) {
    if (!vapidPublicKey || vapidPublicKey === "") {
        return
    }
    await navigator.serviceWorker.ready
        .then(function (registration) {
            return registration.pushManager.subscribe({
                userVisibleOnly: true,
                applicationServerKey: urlBase64ToUint8Array(vapidPublicKey),
            });
        })
        .then(async function (subscription) {
            const req = new RegisterGkillNotificationRequest()
            req.session_id = GkillAPI.get_gkill_api().get_session_id()
            req.subscription = subscription
            req.public_key = vapidPublicKey
            const res = await GkillAPI.get_gkill_api().register_gkill_notification(req)
            if (res.errors && res.errors.length !== 0) {
                write_errors(res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                write_messages(res.messages)
            }
        })
        .catch(err => console.error(err));
}
// プッシュ通知登録用
function urlBase64ToUint8Array(base64String: string) {
    const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
    /* eslint no-useless-escape: 0 */
    const base64 = (base64String + padding).replace(/\-/g, '+').replace(/_/g, '/');
    const rawData = window.atob(base64);
    return Uint8Array.from([...rawData].map(char => char.charCodeAt(0)));
}

nextTick(() => register_gkill_task_notification())

// プッシュ通知登録用
async function register_gkill_task_notification(): Promise<void> {
    if ('serviceWorker' in navigator) {
        await navigator.serviceWorker.ready
            .then(function (registration) {
                return registration.pushManager.getSubscription();
            })
            .then(async function (subscription) {
                if (!subscription) {
                    const req = new GetGkillNotificationPublicKeyRequest()
                    req.session_id = GkillAPI.get_gkill_api().get_session_id()
                    const res = await GkillAPI.get_gkill_api().get_gkill_notification_public_key(req)
                    if (res.errors && res.errors.length !== 0) {
                        write_errors(res.errors)
                        return
                    }
                    if (res.messages && res.messages.length !== 0) {
                        write_messages(res.messages)
                    }
                    subscribe(res.gkill_notification_public_key)
                }
            })
    }
}

</script>
<style lang="css">
/* 不要なスクロールバーを消す */
body,
.v-application--wrap,
.v-navigation-drawer--open {
    overflow-y: scroll !important;
    overflow-x: auto !important;
    height: calc(actual_height) !important;
    min-height: calc(actual_height) !important;
    max-height: calc(actual_height) !important;
}

body {
    overflow-y: hidden !important;
}

/* メッセージ、エラーメッセージ */
.alert_container {
    position: fixed;
    top: 60px;
    right: 10px;
    display: grid;
    grid-gap: .5em;
    z-index: 100000000;
}

/* ダイアログ */
.kyou_detail_view,
.kyou_list_view,
.v-dialog .v-card {
    overflow-y: scroll;
}

/* スクロールバー */
.v-navigation-drawer__content::-webkit-scrollbar,
.kyou_detail_view::-webkit-scrollbar,
.kyou_list_view::-webkit-scrollbar,
.kyou_list_view_image::-webkit-scrollbar,
.kftl_text_area::-webkit-scrollbar,
.v-dialog .v-card::-webkit-scrollbar {
    margin-left: 1px;
    width: 8px;
}

.v-navigation-drawer__content::-webkit-scrollbar-thumb,
.kyou_detail_view::-webkit-scrollbar-thumb,
.kyou_list_view::-webkit-scrollbar-thumb,
.kyou_list_view_image::-webkit-scrollbar-thumb,
.kftl_text_area::-webkit-scrollbar-thumb,
.v-dialog .v-card::-webkit-scrollbar-thumb {
    background: rgb(var(--v-theme-primary));
    width: 6px;
    border-radius: 5px;
}

/* テーブルの隙間埋め */
table,
tr,
td {
    border-spacing: 0 !important;
}
</style>