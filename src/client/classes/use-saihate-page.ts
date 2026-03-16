import { i18n } from '@/i18n'
import router from '@/router'
import { computed, onMounted, onUnmounted, ref, watch, type Ref } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import type { GkillError } from '@/classes/api/gkill-error'
import { GkillMessage } from '@/classes/api/gkill-message'
import { Kyou } from '@/classes/datas/kyou'
import { useTheme } from 'vuetify'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { useScopedEnterForKFTL } from '@/classes/use-scoped-enter-for-kftl'
import { useRoute } from 'vue-router'
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'
import { LogoutRequest } from '@/classes/api/req_res/logout-request'
import { ReloadRepositoriesRequest } from '@/classes/api/req_res/reload-repositories-request'
import { delete_gkill_config_cache } from '@/classes/delete-gkill-cache'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'

export function useSaihatePage() {
    const theme = useTheme()

    // ── Template refs ──
    const add_mi_dialog = ref<any>(null)
    const add_nlog_dialog = ref<any>(null)
    const add_lantana_dialog = ref<any>(null)
    const add_timeis_dialog = ref<any>(null)
    const add_urlog_dialog = ref<any>(null)
    const kftl_dialog = ref<any>(null)
    const add_kc_dialog = ref<any>(null)
    const mkfl_dialog = ref<any>(null)
    const upload_file_dialog = ref<any>(null)
    const confirm_logout_dialog = ref<any>(null)
    const saihate_root = ref<HTMLElement | null>(null)

    // ── State refs ──
    const enable_context_menu = ref(true)
    const enable_dialog = ref(false)

    const actual_height: Ref<Number> = ref(0)
    const element_height: Ref<Number> = ref(0)
    const browser_url_bar_height: Ref<Number> = ref(0)
    const app_title_bar_height: Ref<Number> = ref(50)
    const gkill_api = computed(() => GkillAPI.get_instance())
    const application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
    const app_content_height: Ref<Number> = ref(0)
    const app_content_width: Ref<Number> = ref(0)

    const position_x: Ref<Number> = ref(0)
    const position_y: Ref<Number> = ref(0)
    const add_kyou_menu_style = computed(() => `{ position: absolute; left: ${position_x.value}px; top: ${position_y.value}px; }`)

    const is_loading = ref(true)

    const messages: Ref<Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>> = ref([])

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

    // ── Watchers ──
    watch(() => application_config.value, () => {
        is_loading.value = false
    })
    watch(() => is_loading.value, (new_value: boolean, old_value: boolean) => {
        if (old_value !== new_value && !new_value) {
            show_dialog()
        }
    })

    // ── Business logic ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

    async function show_dialog(): Promise<void> {
        const dialog = new URL(location.href).searchParams.get('dialog')
        const is_saved = new URL(location.href).searchParams.get('is_saved')
        if (is_saved && parseBoolLoose(is_saved)) {
            const message = new GkillMessage()
            message.message = i18n.global.t("SAVED_MESSAGE")
            message.message_code = GkillMessageCodes.saved_shared_data
            write_messages([message])

            await sleep(2500)
            await resetDialogHistory()
            await router.replace('/saihate')
            window.close()
        }
        switch (dialog) {
            case 'kc':
                show_add_kc_dialog()
                break
            case 'timeis':
                show_timeis_dialog()
                break
            case 'mi':
                show_mi_dialog()
                break
            case 'nlog':
                show_nlog_dialog()
                break
            case 'lantana':
                show_lantana_dialog()
                break
            case 'urlog':
                show_urlog_dialog()
                break
            case 'kftl':
                show_kftl_dialog()
                break
            case 'mkfl':
                show_mkfl_dialog()
                break
            case 'file':
                show_upload_file_dialog()
                break
            default:
                break
        }
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

    function floatingActionButtonStyle() {
        return {
            'bottom': '60px',
            'right': '10px',
            'height': '50px',
            'width': '50px'
        }
    }

    // ── Dialog show methods ──
    function show_kftl_dialog(): void {
        kftl_dialog.value?.show()
    }

    function show_add_kc_dialog(): void {
        add_kc_dialog.value?.show()
    }

    function show_mkfl_dialog(): void {
        mkfl_dialog.value?.show()
    }

    function show_timeis_dialog(): void {
        add_timeis_dialog.value?.show()
    }

    function show_mi_dialog(): void {
        add_mi_dialog.value?.show()
    }

    function show_nlog_dialog(): void {
        add_nlog_dialog.value?.show()
    }

    function show_lantana_dialog(): void {
        add_lantana_dialog.value?.show()
    }

    function show_urlog_dialog(): void {
        add_urlog_dialog.value?.show()
    }

    function show_upload_file_dialog(): void {
        upload_file_dialog.value?.show()
    }

    function show_confirm_logout_dialog(close_database: boolean): void {
        confirm_logout_dialog.value?.show(close_database)
    }

    function parseBoolLoose(value: unknown): boolean {
        if (typeof value === "boolean") return value
        if (typeof value === "number") return value !== 0
        if (typeof value === "string") {
            const v = value.trim().toLowerCase()
            if (["true", "1", "yes", "y"].includes(v)) return true
            if (["false", "0", "no", "n"].includes(v)) return false
        }
        throw new SyntaxError(`Boolean expected, got ${JSON.stringify(value)}`)
    }

    async function reload_repositories(clear_thumb_cache: boolean): Promise<void> {
        const requested_reload_message = new GkillMessage()
        requested_reload_message.message = i18n.global.t("REQUESTED_RELOAD_TITLE")
        requested_reload_message.message_code = GkillMessageCodes.requested_reload
        requested_reload_message.show_keep = true
        write_messages([requested_reload_message])

        is_loading.value = true

        const req = new ReloadRepositoriesRequest()
        req.clear_thumb_cache = clear_thumb_cache
        const res = await gkill_api.value.reload_repositories(req)

        await delete_gkill_config_cache()
        await delete_gkill_kyou_cache(null)

        if (res.errors && res.errors.length !== 0) {
            write_errors(res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            write_messages(res.messages)
        }

        is_loading.value = false

        const page_reload_message = new GkillMessage()
        page_reload_message.message = i18n.global.t("DO_RELOAD_TITLE")
        page_reload_message.message_code = GkillMessageCodes.do_reload
        write_messages([page_reload_message])
        await sleep(1500)

        location.reload()
    }

    async function logout(close_database: boolean): Promise<void> {
        const req = new LogoutRequest()
        req.close_database = close_database
        const res = await gkill_api.value.logout(req)
        if (res.errors && res.errors.length !== 0) {
            write_errors(res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            write_messages(res.messages)
        }
        await sleep(1500)
        gkill_api.value.set_session_id("")
        await resetDialogHistory()
        router.replace("/")
    }

    // ── Initialize ──
    resize_content()
    load_application_config()

    // ── Keyboard shortcut ──
    const enable_enter_shortcut = ref(true)
    useScopedEnterForKFTL(saihate_root, show_kftl_dialog, enable_enter_shortcut)

    // ── Return ──
    return {
        // Template refs
        saihate_root,
        add_mi_dialog,
        add_nlog_dialog,
        add_lantana_dialog,
        add_timeis_dialog,
        add_urlog_dialog,
        kftl_dialog,
        add_kc_dialog,
        mkfl_dialog,
        upload_file_dialog,
        confirm_logout_dialog,

        // State
        enable_context_menu,
        enable_dialog,
        actual_height,
        app_title_bar_height,
        gkill_api,
        application_config,
        app_content_height,
        app_content_width,
        is_loading,
        messages,
        add_kyou_menu_style,

        // Methods
        write_errors,
        write_messages,
        close_message,
        floatingActionButtonStyle,

        // Dialog show methods
        show_kftl_dialog,
        show_mkfl_dialog,
        show_add_kc_dialog,
        show_urlog_dialog,
        show_timeis_dialog,
        show_mi_dialog,
        show_nlog_dialog,
        show_lantana_dialog,
        show_upload_file_dialog,
        show_confirm_logout_dialog,

        // Reload
        reload_repositories,

        // Logout
        logout,
    }
}
