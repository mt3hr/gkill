import { i18n } from '@/i18n'
import router from '@/router'
import { computed, onMounted, onUnmounted, ref, type Ref } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import type { GkillError } from '@/classes/api/gkill-error'
import { GkillMessage } from '@/classes/api/gkill-message'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { DashboardConfig } from '@/classes/datas/config/dashboard-config'
import { Kyou } from '@/classes/datas/kyou'
import { useTheme } from 'vuetify'
import { useRoute } from 'vue-router'
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'
import { LogoutRequest } from '@/classes/api/req_res/logout-request'
import type { ComponentRef } from '@/classes/component-ref'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import moment from 'moment'
import { useScopedEnterForKFTL } from '@/classes/use-scoped-enter-for-kftl'
import { useScopedCtrlVForClipboard } from '@/classes/use-scoped-ctrl-v-for-clipboard'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'

export function useDashboardPage() {
    const theme = useTheme()

    // ── Template refs ──
    const dashboard_root = ref<HTMLElement | null>(null)
    const confirm_logout_dialog = ref<ComponentRef | null>(null)
    const add_mi_dialog = ref<ComponentRef | null>(null)
    const add_nlog_dialog = ref<ComponentRef | null>(null)
    const add_lantana_dialog = ref<ComponentRef | null>(null)
    const add_timeis_dialog = ref<ComponentRef | null>(null)
    const add_urlog_dialog = ref<ComponentRef | null>(null)
    const kftl_dialog = ref<ComponentRef | null>(null)
    const add_kc_dialog = ref<ComponentRef | null>(null)
    const mkfl_dialog = ref<ComponentRef | null>(null)
    const upload_file_dialog = ref<ComponentRef | null>(null)
    const save_clipboard_to_file_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const actual_height: Ref<number> = ref(0)
    const element_height: Ref<number> = ref(0)
    const browser_url_bar_height: Ref<number> = ref(0)
    const app_title_bar_height: Ref<number> = ref(50)
    const gkill_api = computed(() => GkillAPI.get_instance())
    const application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
    const app_content_height: Ref<number> = ref(0)
    const app_content_width: Ref<number> = ref(0)
    const messages: Ref<Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>> = ref([])

    const selected_date: Ref<Date> = ref(moment().startOf('day').toDate())
    const checked_kyous: Ref<Array<Kyou>> = ref([])
    const mi_kyous: Ref<Array<Kyou>> = ref([])
    const dnote_kyous: Ref<Array<Kyou>> = ref([])
    let mi_kyous_fetch_epoch = 0
    let current_mi_abort_controller: AbortController | null = null
    let current_dnote_abort_controller: AbortController | null = null
    const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])

    // ── Computed ──
    const page_list = computed(() => [
        { app_name: i18n.global.t('RYKV_APP_NAME'), page_name: 'rykv' },
        { app_name: i18n.global.t('MI_APP_NAME'), page_name: 'mi' },
        { app_name: i18n.global.t('KFTL_APP_NAME'), page_name: 'kftl' },
        { app_name: i18n.global.t('PLAING_TIMEIS_APP_NAME'), page_name: 'plaing' },
        { app_name: i18n.global.t('MKFL_APP_NAME'), page_name: 'mkfl' },
        { app_name: i18n.global.t('DASHBOARD_APP_NAME'), page_name: 'dashboard' },
        { app_name: i18n.global.t('SAIHATE_APP_NAME'), page_name: 'saihate' },
    ])

    const panel_height = computed<number>(() => Math.max(400, app_content_height.value - 350))

    const target_date_start = computed<Date>(() => moment(selected_date.value).startOf('day').toDate())
    const target_date_end = computed<Date>(() => moment(selected_date.value).endOf('day').toDate())

    const date_label = computed<string>(() => {
        const weekdayKeys = [
            'SUNDAY_TITLE', 'MONDAY_TITLE', 'TUESDAY_TITLE', 'WEDNESDAY_TITLE',
            'THURSDAY_TITLE', 'FRIDAY_TITLE', 'SATURDAY_TITLE',
        ]
        const dayOfWeek = i18n.global.t(weekdayKeys[moment(selected_date.value).day()])
        return `${moment(selected_date.value).format('YYYY/M/D')}(${dayOfWeek})`
    })

    const date_picker_model = computed<Date>({
        get: () => selected_date.value,
        set: (value: Date) => {
            selected_date.value = moment(value).startOf('day').toDate()
        },
    })

    const dnote_query = computed<FindKyouQuery>(() => {
        const base_query = new FindKyouQuery()
        base_query.use_reps = false
        base_query.use_tags = false
        if (application_config.value.dashboard_json_data) {
            const config = DashboardConfig.parse(application_config.value.dashboard_json_data)
            if (config.dashboard_dnote_find_kyou_query) {
                const saved = config.dashboard_dnote_find_kyou_query
                base_query.use_tags = saved.use_tags
                base_query.tags = saved.tags.concat()
                base_query.tags_and = saved.tags_and
                base_query.use_reps = saved.use_reps
                base_query.reps = saved.reps.concat()
                base_query.use_words = saved.use_words
                base_query.keywords = saved.keywords
                base_query.words = saved.words.concat()
                base_query.not_words = saved.not_words.concat()
            }
        }
        base_query.use_calendar = true
        base_query.calendar_start_date = target_date_start.value
        base_query.calendar_end_date = target_date_end.value
        base_query.apply_hide_tags(application_config.value)
        return base_query
    })

    const mi_kyou_query = computed<FindKyouQuery>(() => {
        const query = new FindKyouQuery()
        if (application_config.value.dashboard_json_data) {
            const config = DashboardConfig.parse(application_config.value.dashboard_json_data)
            if (config.dashboard_mi_find_kyou_query) {
                const saved = config.dashboard_mi_find_kyou_query
                query.use_tags = saved.use_tags
                query.tags = saved.tags.concat()
                query.tags_and = saved.tags_and
                query.use_words = saved.use_words
                query.keywords = saved.keywords
                query.words = saved.words.concat()
                query.not_words = saved.not_words.concat()
                query.use_mi_check_state = saved.use_mi_check_state
                query.mi_check_state = saved.mi_check_state
            }
        }
        query.for_mi = true
        query.use_reps = false
        query.use_calendar = true
        query.calendar_start_date = target_date_start.value
        query.calendar_end_date = target_date_end.value
        query.apply_hide_tags(application_config.value)
        return query
    })

    // ── Loading state ──
    const is_loading = ref(true)

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

    // ── Business logic ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

    async function load_application_config(): Promise<void> {
        const req = new GetApplicationConfigRequest()
        const loaded_raw_value = useRoute().query.loaded
        const loaded = loaded_raw_value && (loaded_raw_value == 'true')
        req.force_reget = !loaded
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

    function abort_all_fetches(): void {
        if (current_mi_abort_controller) {
            current_mi_abort_controller.abort()
            current_mi_abort_controller = null
        }
        if (current_dnote_abort_controller) {
            current_dnote_abort_controller.abort()
            current_dnote_abort_controller = null
        }
        mi_kyous_fetch_epoch++
    }

    function clear_dashboard_datas(): void {
        checked_kyous.value.splice(0)
        dnote_kyous.value.splice(0)
        mi_kyous.value.splice(0)
    }

    async function fetch_mi_kyous(): Promise<void> {
        const current_epoch = ++mi_kyous_fetch_epoch
        const req = new GetKyousRequest()
        req.query = mi_kyou_query.value
        current_mi_abort_controller = req.abort_controller

        req.query.include_create_mi = req.query.mi_sort_type === MiSortType.create_time
        req.query.include_check_mi = false
        req.query.include_limit_mi = req.query.mi_sort_type === MiSortType.limit_time
        req.query.include_start_mi = req.query.mi_sort_type === MiSortType.estimate_start_time
        req.query.include_end_mi = req.query.mi_sort_type === MiSortType.estimate_end_time

        try {
            const res = await gkill_api.value.get_kyous(req)
            if (current_epoch !== mi_kyous_fetch_epoch) {
                return
            }
            if (res.errors && res.errors.length !== 0) {
                write_errors(res.errors)
                return
            }
            mi_kyous.value = res.kyous
        } catch (_e: unknown) {
            // abort時は無視
        }
    }

    async function fetch_dnote_kyous(): Promise<Array<Kyou>> {
        if (current_dnote_abort_controller) {
            current_dnote_abort_controller.abort()
        }
        const req = new GetKyousRequest()
        req.query = dnote_query.value
        current_dnote_abort_controller = req.abort_controller
        try {
            const res = await gkill_api.value.get_kyous(req)
            if (res.errors && res.errors.length !== 0) {
                write_errors(res.errors)
                return []
            }
            dnote_kyous.value = res.kyous
            return res.kyous
        } catch (_e: unknown) {
            // abort時は無視
            return []
        }
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

    // ── Rykv dialog ──
    function open_rykv_dialog(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload): void {
        const dialog_id = gkill_api.value.generate_uuid()
        opened_dialogs.value.push({
            id: dialog_id,
            kind,
            kyou: kyou.clone(),
            payload: payload ?? null,
            opened_at: Date.now(),
        });
        (async (): Promise<void> => {
            const updated_kyou = kyou.clone()
            await updated_kyou.reload(false, true, mi_kyou_query.value)
            updated_kyou.is_typed_data_loaded = false
            await updated_kyou.load_all(mi_kyou_query.value, true)
            for (let i = 0; i < opened_dialogs.value.length; i++) {
                if (opened_dialogs.value[i].id === dialog_id) {
                    opened_dialogs.value[i] = { ...opened_dialogs.value[i], kyou: updated_kyou }
                    break
                }
            }
        })()
    }

    function close_rykv_dialog(dialog_id: string): void {
        for (let i = 0; i < opened_dialogs.value.length; i++) {
            if (opened_dialogs.value[i].id === dialog_id) {
                opened_dialogs.value.splice(i, 1)
                break
            }
        }
    }

    // ── Navigation ──
    async function navigateToPage(page_name: string): Promise<void> {
        await resetDialogHistory()
        router.replace('/' + page_name + '?loaded=true')
    }

    // ── Floating button ──
    function floatingActionButtonStyle() {
        return {
            'bottom': '60px',
            'right': '10px',
            'height': '50px',
            'width': '50px',
        }
    }

    // ── Dialog show methods ──
    function show_confirm_logout_dialog(close_database: boolean): void {
        confirm_logout_dialog.value?.show(close_database)
    }

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

    function show_save_clipboard_to_file_dialog(): void {
        save_clipboard_to_file_dialog.value?.show()
    }

    // ── Date navigation ──
    function go_prev_day(): void {
        selected_date.value = moment(selected_date.value).subtract(1, 'day').startOf('day').toDate()
    }
    function go_next_day(): void {
        selected_date.value = moment(selected_date.value).add(1, 'day').startOf('day').toDate()
    }
    function go_today(): void {
        selected_date.value = moment().startOf('day').toDate()
    }

    // ── Logout ──
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

    // ── Keyboard shortcuts ──
    const enable_enter_shortcut = ref(true)
    useScopedEnterForKFTL(dashboard_root, show_kftl_dialog, enable_enter_shortcut)
    useScopedCtrlVForClipboard(dashboard_root, show_save_clipboard_to_file_dialog, enable_enter_shortcut)

    // ── Initialize ──
    resize_content()
    load_application_config()

    // ── Return ──
    return {
        // Template refs
        dashboard_root,
        confirm_logout_dialog,
        add_mi_dialog,
        add_nlog_dialog,
        add_lantana_dialog,
        add_timeis_dialog,
        add_urlog_dialog,
        kftl_dialog,
        add_kc_dialog,
        mkfl_dialog,
        upload_file_dialog,
        save_clipboard_to_file_dialog,

        // State
        is_loading,
        actual_height,
        app_title_bar_height,
        gkill_api,
        application_config,
        app_content_height,
        app_content_width,
        messages,
        selected_date,
        checked_kyous,
        mi_kyous,
        dnote_kyous,
        opened_dialogs,

        // Computed
        panel_height,
        page_list,
        target_date_start,
        target_date_end,
        date_label,
        date_picker_model,
        dnote_query,
        mi_kyou_query,

        // Methods
        write_errors,
        write_messages,
        close_message,
        navigateToPage,
        abort_all_fetches,
        clear_dashboard_datas,
        fetch_mi_kyous,
        fetch_dnote_kyous,
        load_application_config,
        go_prev_day,
        go_next_day,
        go_today,
        floatingActionButtonStyle,
        show_confirm_logout_dialog,
        show_kftl_dialog,
        show_add_kc_dialog,
        show_mkfl_dialog,
        show_timeis_dialog,
        show_mi_dialog,
        show_nlog_dialog,
        show_lantana_dialog,
        show_urlog_dialog,
        show_upload_file_dialog,
        show_save_clipboard_to_file_dialog,
        logout,
        open_rykv_dialog,
        close_rykv_dialog,
    }
}
