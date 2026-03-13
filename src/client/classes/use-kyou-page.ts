import { i18n } from '@/i18n'
import router from '@/router'
import { GkillAPI } from '@/classes/api/gkill-api'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { type Ref, ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { InfoIdentifier } from '@/classes/datas/info-identifier'
import { Kyou } from '@/classes/datas/kyou'
import { GetGkillNotificationPublicKeyRequest } from '@/classes/api/req_res/get-gkill-notification-public-key-request'
import { RegisterGkillNotificationRequest } from '@/classes/api/req_res/register-gkill-notification-request'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import { useTheme } from 'vuetify'
import { useRoute } from 'vue-router'
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'

export function useKyouPage() {
    const theme = useTheme()

    // ── Template refs ──
    const application_config_dialog = ref<any>(null)

    // ── State refs ──
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

    const is_loading = ref(true)

    // ── Computed ──
    const page_list = computed(() => [
        { app_name: i18n.global.t('RYKV_APP_NAME'), page_name: 'rykv' },
        { app_name: i18n.global.t('MI_APP_NAME'), page_name: 'mi' },
        { app_name: i18n.global.t('KFTL_APP_NAME'), page_name: 'kftl' },
        { app_name: i18n.global.t('PLAING_TIMEIS_APP_NAME'), page_name: 'plaing' },
        { app_name: i18n.global.t('MKFL_APP_NAME'), page_name: 'mkfl' },
        { app_name: i18n.global.t('SAIHATE_APP_NAME'), page_name: 'saihate' },
    ])

    // ── Messages ──
    const messages: Ref<Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>> = ref([])

    // ── Watchers ──
    watch(() => application_config.value, () => {
        is_loading.value = false
    })

    // ── Lifecycle ──
    onMounted(async () => {
        await resetDialogHistory()
    })

    const onResize = () => {
        resize_content()
    }
    window.addEventListener('resize', onResize)
    onUnmounted(() => {
        window.removeEventListener('resize', onResize)
    })

    // ── Internal helpers ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

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
        for (let j = 0; j < received_errors.length; j++) {
            const auto_close_duration_milli_seconds = received_errors[j].auto_close_duration_milli_seconds
            if (auto_close_duration_milli_seconds) {
                sleep(auto_close_duration_milli_seconds).then(() => {
                    close_message(received_errors[j].id)
                })
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
        for (let j = 0; j < received_messages.length; j++) {
            const auto_close_duration_milli_seconds = received_messages[j].auto_close_duration_milli_seconds
            if (auto_close_duration_milli_seconds) {
                sleep(auto_close_duration_milli_seconds).then(() => {
                    close_message(received_messages[j].id)
                })
            }
        }
    }

    function close_message(message_id: string): void {
        for (let i = 0; i < messages.value.length; i++) {
            if (messages.value[i].id === message_id) {
                messages.value.splice(i, 1)
                return
            }
        }
    }

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
        if (!res.kyou_histories || res.kyou_histories.length === 0) {
            return
        }
        kyou.value = res.kyou_histories[0]
        await kyou.value.load_all()
    }

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

    // ── Template event handlers ──
    async function navigateToPage(page_name: string): Promise<void> {
        await resetDialogHistory()
        router.replace('/' + page_name + '?loaded=true')
    }

    // ── Initialization ──
    resize_content()
    load_application_config().then(() => {
        load_kyou()
    })

    nextTick(() => register_gkill_task_notification())

    // ── Return ──
    return {
        // Template refs
        application_config_dialog,

        // State
        enable_context_menu,
        enable_dialog,
        actual_height,
        app_title_bar_height,
        gkill_api,
        application_config,
        app_content_height,
        app_content_width,
        is_show_application_config_dialog,
        hightlight_targets,
        is_image_view,
        kyou,
        is_loading,
        messages,

        // Computed
        page_list,

        // Template event handlers
        navigateToPage,
        write_errors,
        write_messages,
        close_message,
        load_application_config,
        show_application_config_dialog,
    }
}
