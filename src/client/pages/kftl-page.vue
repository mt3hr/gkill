<template>
    <div class="kftl_view_wrap">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-btn icon="mdi-menu" :ripple="false" link="false" :style="{ opacity: 0, cursor: 'unset', }" />
            <v-toolbar-title>
                <div>
                    <span>
                        {{ i18n.global.t("KFTL_APP_NAME") }}
                    </span>
                    <v-menu activator="parent">
                        <v-list>
                            <v-list-item :key="index" :value="index" v-for="page, index in [
                                { app_name: i18n.global.t('RYKV_APP_NAME'), page_name: 'rykv' },
                                { app_name: i18n.global.t('MI_APP_NAME'), page_name: 'mi' },
                                { app_name: i18n.global.t('KFTL_APP_NAME'), page_name: 'kftl' },
                                { app_name: i18n.global.t('PLAING_TIMEIS_APP_NAME'), page_name: 'plaing' },
                                { app_name: i18n.global.t('MKFL_APP_NAME'), page_name: 'mkfl' },
                                { app_name: i18n.global.t('SAIHATE_APP_NAME'), page_name: 'saihate' },
                            ]">
                                <v-list-item-title
                                    @click="async () => { await resetDialogHistory(); router.replace('/' + page.page_name + '?loaded=true') }">
                                    {{ page.app_name }}</v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-menu>
                </div>
            </v-toolbar-title>
            <v-spacer />
            <v-divider vertical />
            <v-btn icon="mdi-cog" :disabled="!application_config.is_loaded" @click="show_application_config_dialog()" />
        </v-app-bar>
        <v-main class="main">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
            <kftlView :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)"
                ref="kftl_view" />
            <ApplicationConfigDialog :application_config="application_config" :gkill_api="gkill_api"
                :app_content_height="app_content_height" :app_content_width="app_content_width"
                :is_show="is_show_application_config_dialog"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)"
                @requested_reload_application_config="load_application_config" ref="application_config_dialog" />
        </v-main>
        <div class="alert_container">
            <v-slide-y-transition group>
                <v-tooltip :text="(message.is_error ? 'エラーコード' : 'メッセージコード') + ':' + message.code"
                    v-for="message in messages" :key="message.id">
                    <template v-slot:activator="{ props }">
                        <v-alert v-bind="props" :color="message.is_error ? 'error' : undefined"
                            :closable="message.closable" @click:close="close_message(message.id)">
                            {{ message.message }}
                        </v-alert>
                    </template>
                </v-tooltip>
            </v-slide-y-transition>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
'use strict'
import router from '@/router'
import { computed, nextTick, onMounted, ref, watch, type Ref } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

import ApplicationConfigDialog from './dialogs/application-config-dialog.vue'
import kftlView from './views/kftl-view.vue'
import { GetGkillNotificationPublicKeyRequest } from '@/classes/api/req_res/get-gkill-notification-public-key-request'
import { RegisterGkillNotificationRequest } from '@/classes/api/req_res/register-gkill-notification-request'
import { useTheme } from 'vuetify'
import { useRoute } from 'vue-router'
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'

const theme = useTheme()

const application_config_dialog = ref<InstanceType<typeof ApplicationConfigDialog> | null>(null);
const kftl_view = ref<InstanceType<typeof kftlView> | null>(null);

const actual_height: Ref<Number> = ref(0)
const element_height: Ref<Number> = ref(0)
const browser_url_bar_height: Ref<Number> = ref(0)
const app_title_bar_height: Ref<Number> = ref(50)
const gkill_api = computed(() => GkillAPI.get_instance())
const application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
const app_content_height: Ref<Number> = ref(0)
const app_content_width: Ref<Number> = ref(0)

const is_show_application_config_dialog: Ref<boolean> = ref(false)

onMounted(async () => {
    await resetDialogHistory()
})

async function load_application_config(): Promise<void> {
    const req = new GetApplicationConfigRequest()
    const loaded_raw_value = useRoute().query.loaded
    const loaded = loaded_raw_value && (loaded_raw_value == 'true')
    req.force_reget = !loaded // メニューから遷移したときにはApplicationConfig再取得はしない（キャッシュから取得する）
    return gkill_api.value.get_application_config(req)
        .then(res => {
            if (res.errors && res.errors.length != 0) {
                write_errors(res.errors)
                return
            }

            const use_dark_theme = res.application_config.use_dark_theme
            if (use_dark_theme) {
                theme.global.name.value = 'gkill_dark_theme'
            } else {
                theme.global.name.value = 'gkill_theme'
            }
            gkill_api.value.set_use_dark_theme(use_dark_theme)

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

const messages: Ref<Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>> = ref([])

async function write_errors(errors_: Array<GkillError>) {
    const received_errors = new Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>()
    for (let i = 0; i < errors_.length; i++) {
        if (errors_[i] && errors_[i].error_message) {
            received_errors.push({
                code: errors_[i].error_code,
                message: errors_[i].error_message,
                id: GkillAPI.get_instance().generate_uuid(),
                show_snackbar: true,
                closable: errors_[i].show_keep,
                auto_close_duration_milli_seconds: errors_[i].show_keep ? null : 2500,
                is_error: true,
            })
        }
    }
    messages.value.push(...received_errors)
    for (let i = 0; i < received_errors.length; i++) {
        for (let j = 0; j < received_errors.length; j++) {
            const auto_close_duration_milli_seconds = received_errors[j].auto_close_duration_milli_seconds
            if (auto_close_duration_milli_seconds) {
                sleep(auto_close_duration_milli_seconds).then(() => {
                    close_message(received_errors[j].id)
                })
            }
        }
    }
}

async function write_messages(messages_: Array<GkillMessage>) {
    const received_messages = new Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>()
    for (let i = 0; i < messages_.length; i++) {
        if (messages_[i] && messages_[i].message) {
            received_messages.push({
                code: messages_[i].message_code,
                message: messages_[i].message,
                id: GkillAPI.get_instance().generate_uuid(),
                show_snackbar: true,
                closable: messages_[i].show_keep,
                auto_close_duration_milli_seconds: messages_[i].show_keep ? null : 2500,
                is_error: false,
            })
        }
    }
    messages.value.push(...received_messages)
    for (let i = 0; i < received_messages.length; i++) {
        for (let j = 0; j < received_messages.length; j++) {
            const auto_close_duration_milli_seconds = received_messages[j].auto_close_duration_milli_seconds
            if (auto_close_duration_milli_seconds) {
                sleep(auto_close_duration_milli_seconds).then(() => {
                    close_message(received_messages[j].id)
                })
            }
        }
    }
}

function close_message(message_id: string): void {
    for (let i = 0; i < messages.value.length; i++) {
        if (messages.value[i].id === message_id) {
            messages.value.splice(i, 1)
        }
    }
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
load_application_config().then(() => kftl_view.value?.focus_kftl_text_area())

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

function show_application_config_dialog(): void {
    application_config_dialog.value?.show()
}
</script>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: v-bind("is_loading == true ? '0px' : 'calc(100vw)'");
}
</style>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>