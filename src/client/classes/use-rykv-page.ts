import { computed, nextTick, onMounted, onUnmounted, ref, type Ref } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { GetGkillNotificationPublicKeyRequest } from '@/classes/api/req_res/get-gkill-notification-public-key-request'
import { RegisterGkillNotificationRequest } from '@/classes/api/req_res/register-gkill-notification-request'
import { useTheme } from 'vuetify'
import { useRoute } from 'vue-router'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'
import { Tag } from '@/classes/datas/tag'
import { GetAllTagNamesRequest } from '@/classes/api/req_res/get-all-tag-names-request'
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'
import type { ComponentRef } from '@/classes/component-ref'

export function useRykvPage() {
    const theme = useTheme()

    // ── Template refs ──
    const application_config_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const actual_height: Ref<number> = ref(0)
    const element_height: Ref<number> = ref(0)
    const browser_url_bar_height: Ref<number> = ref(0)
    const app_title_bar_height: Ref<number> = ref(50)
    const gkill_api = computed(() => GkillAPI.get_instance())
    const application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
    const app_content_height: Ref<number> = ref(0)
    const app_content_width: Ref<number> = ref(0)
    const is_show_application_config_dialog: Ref<boolean> = ref(false)

    const messages: Ref<Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>> = ref([])

    // ── 連打/連続登録で二重に通信しないため ──
    let tagStructRefreshPromise: Promise<void> | null = null

    // ── Helpers ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

    function resize_content(): void {
        const inner_element = document.querySelector('#control-height')
        actual_height.value = window.innerHeight
        element_height.value = inner_element ? inner_element.clientHeight : actual_height.value
        browser_url_bar_height.value = Number(element_height.value) - Number(actual_height.value)
        app_content_height.value = Number(element_height.value) - (Number(browser_url_bar_height.value) + Number(app_title_bar_height.value))
        app_content_width.value = window.innerWidth
    }

    async function load_application_config(): Promise<void> {
        const req = new GetApplicationConfigRequest()
        const loaded_raw_value = useRoute().query.loaded
        const loaded = loaded_raw_value && (loaded_raw_value == 'true')
        req.force_reget = !loaded // メニューから遷移したときにはApplicationConfig再取得はしない（キャッシュから取得する）
        return gkill_api.value.get_application_config(req)
            .then(async res => {
                if (res.errors && res.errors.length !== 0) {
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

                if (res.messages && res.messages.length !== 0) {
                    write_messages(res.messages)
                    return
                }
            })
    }

    function write_errors(errors_: Array<GkillError>): void {
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

    function write_messages(messages_: Array<GkillMessage>): void {
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

    function show_application_config_dialog(): void {
        application_config_dialog.value?.show()
    }

    function tagStructHas(tag_struct: TagStructElementData, tagName: string): boolean {
        if (tag_struct.tag_name === tagName) return true
        for (const c of (tag_struct.children ?? [])) {
            if (tagStructHas(c, tagName)) return true
        }
        return false
    }

    async function check_tag_update(tag: Tag): Promise<void> {
        const name = tag.tag
        if (!name) return

        const req = new GetAllTagNamesRequest()
        req.force_reget = true
        await gkill_api.value.get_all_tag_names(req)

        if (tagStructHas(application_config.value.tag_struct, name)) return

        // すでに更新中ならそれに乗る
        if (tagStructRefreshPromise) {
            await tagStructRefreshPromise
            return
        }

        tagStructRefreshPromise = (async () => {
            const errors = await application_config.value.append_not_found_tags()
            if (errors && errors.length) {
                write_errors(errors)
                return
            }

            application_config.value = application_config.value.clone()

            gkill_api.value.set_saved_application_config(application_config.value)
        })()

        try {
            await tagStructRefreshPromise
        } finally {
            tagStructRefreshPromise = null
        }
    }

    // プッシュ通知登録用
    function urlBase64ToUint8Array(base64String: string): Uint8Array {
        const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
        /* eslint no-useless-escape: 0 */
        const base64 = (base64String + padding).replace(/\-/g, '+').replace(/_/g, '/');
        const rawData = window.atob(base64);
        return Uint8Array.from([...rawData].map(char => char.charCodeAt(0)));
    }

    // プッシュ通知登録用
    async function subscribe(vapidPublicKey: string): Promise<void> {
        if (!vapidPublicKey || vapidPublicKey === "") {
            return
        }
        await navigator.serviceWorker.ready
            .then(function (registration) {
                return registration.pushManager.subscribe({
                    userVisibleOnly: true,
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    applicationServerKey: urlBase64ToUint8Array(vapidPublicKey) as any,
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

    // ── RykvView event handlers ──
    function onDeletedKyou(): void {
        // no-op in page
    }

    function onDeletedTag(): void {
        // no-op in page
    }

    function onDeletedText(): void {
        // no-op in page
    }

    function onDeletedNotification(): void {
        // no-op in page
    }

    function onRegisteredKyou(): void {
        // no-op in page
    }

    function onRegisteredTag(tag: Tag): void {
        check_tag_update(tag)
    }

    function onRegisteredText(): void {
        // no-op in page
    }

    function onRegisteredNotification(): void {
        // no-op in page
    }

    function onUpdatedKyou(): void {
        // no-op in page
    }

    function onUpdatedTag(tag: Tag): void {
        check_tag_update(tag)
    }

    function onUpdatedText(): void {
        // no-op in page
    }

    function onRequestedShowApplicationConfigDialog(): void {
        show_application_config_dialog()
    }

    function onReceivedErrors(errors: Array<GkillError>): void {
        write_errors(errors)
    }

    function onReceivedMessages(messages: Array<GkillMessage>): void {
        write_messages(messages)
    }

    function onRequestedReloadApplicationConfig(): void {
        load_application_config()
    }

    function onCloseMessage(message_id: string): void {
        close_message(message_id)
    }

    // ── CRUD relay for RykvView ──
    const rykvViewHandlers = {
        'deleted_kyou': () => onDeletedKyou(),
        'deleted_tag': () => onDeletedTag(),
        'deleted_text': () => onDeletedText(),
        'deleted_notification': () => onDeletedNotification(),
        'registered_kyou': () => onRegisteredKyou(),
        'registered_tag': (tag: Tag) => onRegisteredTag(tag),
        'registered_text': () => onRegisteredText(),
        'registered_notification': () => onRegisteredNotification(),
        'updated_kyou': () => onUpdatedKyou(),
        'updated_tag': (tag: Tag) => onUpdatedTag(tag),
        'updated_text': () => onUpdatedText(),
        'requested_show_application_config_dialog': () => onRequestedShowApplicationConfigDialog(),
        'received_errors': (errors: Array<GkillError>) => onReceivedErrors(errors),
        'received_messages': (messages: Array<GkillMessage>) => onReceivedMessages(messages),
        'requested_reload_application_config': () => onRequestedReloadApplicationConfig(),
    }

    // ── beforeunload guard ──
    function handleBeforeUnload(e: BeforeUnloadEvent) {
        if (is_show_application_config_dialog.value) {
            e.preventDefault()
        }
    }

    // ── Lifecycle ──
    const onResize = () => {
        resize_content()
    }
    window.addEventListener('resize', onResize)
    window.addEventListener('beforeunload', handleBeforeUnload)
    onMounted(async () => {
        await resetDialogHistory()
    })
    onUnmounted(() => {
        window.removeEventListener('resize', onResize)
        window.removeEventListener('beforeunload', handleBeforeUnload)
    })

    // ── Init ──
    resize_content()
    load_application_config()
    nextTick(() => register_gkill_task_notification())

    return {
        // Template refs
        application_config_dialog,

        // State
        actual_height,
        app_title_bar_height,
        gkill_api,
        application_config,
        app_content_height,
        app_content_width,
        is_show_application_config_dialog,
        messages,

        // Event handlers
        onCloseMessage,
        onReceivedErrors,
        onReceivedMessages,
        onRequestedReloadApplicationConfig,

        // CRUD relay
        rykvViewHandlers,
    }
}
