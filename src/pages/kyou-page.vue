<template>
    <div class="kyou_view_wrap">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-btn icon="mdi-menu" :ripple="false" link="false" :style="{ opacity: 0, cursor: 'unset', }" />
            <v-toolbar-title>
                <div>
                    <span>
                        {{ i18n.global.t("KYOU_APP_NAME") }}
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
                                <v-list-item-title @click="router.replace('/' + page.page_name)">
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
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="hightlight_targets" :is_image_view="is_image_view" :kyou="kyou" :last_added_tag="''"
                :show_checkbox="false" :show_content_only="false" :show_mi_create_time="true"
                :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true" :height="'fit-content'"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :show_attached_timeis="true"
                :show_update_time="false" :show_related_time="true" :width="'fit-content'" :is_readonly_mi_check="false"
                :show_rep_name="true" :force_show_latest_kyou_info="true" @received_errors="write_errors"
                @received_messages="write_messages" />
            <ApplicationConfigDialog :application_config="application_config" :gkill_api="gkill_api"
                :app_content_height="app_content_height" :app_content_width="app_content_width"
                :is_show="is_show_application_config_dialog" @received_errors="write_errors"
                @received_messages="write_messages" @requested_reload_application_config="load_application_config"
                ref="application_config_dialog" />
        </v-main>
        <div class="alert_container">
            <v-slide-y-transition group>
                <v-alert v-for="message in messages" theme="dark" :key="message.id">
                    {{ message.message }}
                </v-alert>
            </v-slide-y-transition>
        </div>
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import router from '@/router'
import { GkillAPI } from '@/classes/api/gkill-api'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { type Ref, ref, computed, watch, nextTick } from 'vue'
import ApplicationConfigDialog from './dialogs/application-config-dialog.vue'
import KyouView from './views/kyou-view.vue'
import { InfoIdentifier } from '@/classes/datas/info-identifier'
import { Kyou } from '@/classes/datas/kyou'
import { GetGkillNotificationPublicKeyRequest } from '@/classes/api/req_res/get-gkill-notification-public-key-request'
import { RegisterGkillNotificationRequest } from '@/classes/api/req_res/register-gkill-notification-request'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import { useTheme } from 'vuetify'




const theme = useTheme()

const application_config_dialog = ref<InstanceType<typeof ApplicationConfigDialog> | null>(null);

const enable_context_menu = ref(true)
const enable_dialog = ref(true)

const actual_height: Ref<Number> = ref(0)
const element_height: Ref<Number> = ref(0)
const browser_url_bar_height: Ref<Number> = ref(0)
const app_title_bar_height: Ref<Number> = ref(50)
const gkill_api = computed(() => GkillAPI.get_instance())
const application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
const app_content_height: Ref<Number> = ref(0)
const app_content_width: Ref<Number> = ref(0)

const is_show_application_config_dialog: Ref<boolean> = ref(false)
const hightlight_targets: Ref<Array<InfoIdentifier>> = ref(new Array<InfoIdentifier>())
const is_image_view: Ref<boolean> = ref(false)
const kyou: Ref<Kyou> = ref(new Kyou())


async function load_kyou(): Promise<void> {
    let kyou_id = new URL(location.href).searchParams.get('kyou_id')
    if (!kyou_id || kyou_id === "") {
        return
    }
    const req = new GetKyouRequest()
    req.id = kyou_id
    const res = await gkill_api.value.get_kyou(req)
    if (res.errors && res.errors.length !== 0) {
        write_errors(res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        write_messages(res.messages)
    }
    kyou.value = res.kyou_histories[0]
    kyou.value.load_all()
}

async function load_application_config(): Promise<void> {
    const req = new GetApplicationConfigRequest()

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
load_application_config().then(() => {
    load_kyou()
})

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

body::-webkit-scrollbar {
    display: none;
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
.tag_struct_root::-webkit-scrollbar,
.rep_struct_root::-webkit-scrollbar,
.rep_type_struct_root::-webkit-scrollbar,
.device_struct_root::-webkit-scrollbar,
.kftl_template_struct_root::-webkit-scrollbar,
.v-navigation-drawer__content::-webkit-scrollbar,
.kyou_detail_view::-webkit-scrollbar,
.kyou_list_view::-webkit-scrollbar,
.kyou_list_view_image::-webkit-scrollbar,
.aggregate_amount_list::-webkit-scrollbar,
.aggregate_location_list::-webkit-scrollbar,
.aggregate_people_list::-webkit-scrollbar,
.kftl_text_area::-webkit-scrollbar,
.v-dialog .v-card::-webkit-scrollbar {
    margin-left: 1px;
    width: 8px;
}

.tag_struct_root::-webkit-scrollbar-thumb,
.rep_struct_root::-webkit-scrollbar-thumb,
.rep_type_struct_root::-webkit-scrollbar-thumb,
.device_struct_root::-webkit-scrollbar-thumb,
.kftl_template_struct_root::-webkit-scrollbar-thumb,
.v-navigation-drawer__content::-webkit-scrollbar-thumb,
.kyou_detail_view::-webkit-scrollbar-thumb,
.ryuu_view::-webkit-scrollbar-thumb,
.kyou_list_view::-webkit-scrollbar-thumb,
.kyou_list_view_image::-webkit-scrollbar-thumb,
.aggregate_amount_list::-webkit-scrollbar-thumb,
.aggregate_location_list::-webkit-scrollbar-thumb,
.aggregate_people_list::-webkit-scrollbar-thumb,
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
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: calc(100vw);
}
</style>