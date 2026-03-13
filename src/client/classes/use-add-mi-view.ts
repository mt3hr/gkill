import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import { Mi } from '@/classes/datas/mi'
import moment from 'moment'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import { GkillError } from '@/classes/api/gkill-error'
import { AddMiRequest } from '@/classes/api/req_res/add-mi-request'
import { Kyou } from '@/classes/datas/kyou'
import { Notification } from '@/classes/datas/notification'
import { AddNotificationRequest } from '@/classes/api/req_res/add-notification-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { AddMiViewProps } from '@/pages/views/add-mi-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type NewBoardNameDialog from '@/pages/dialogs/new-board-name-dialog.vue'

export function useAddMiView(options: {
    props: AddMiViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const new_board_name_dialog = ref<InstanceType<typeof NewBoardNameDialog> | null>(null)
    const add_notification_views = ref<any>(null)

    // ── State refs ──
    const is_requested_submit = ref(false)

    const id: Ref<string> = ref(props.gkill_api.generate_uuid())
    const kyou: Ref<Kyou> = computed(() => {
        const k = new Kyou()
        k.id = id.value
        return k
    })
    const mi: Ref<Mi> = ref((() => {
        const m = new Mi()
        m.id = id.value
        return m
    })())
    const mi_board_names: Ref<Array<string>> = ref(props.application_config.mi_default_board !== "" ? [props.application_config.mi_default_board] : [])

    const mi_title: Ref<string> = ref(mi.value ? mi.value.title : "")
    const mi_board_name: Ref<string> = ref(props.application_config.mi_default_board !== "" ? props.application_config.mi_default_board : "Inbox")
    const mi_estimate_start_date_typed: Ref<Date | null> = ref(mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).toDate() : null)
    const mi_estimate_start_date_string: Ref<string> = computed(() => mi_estimate_start_date_typed.value ? moment(mi_estimate_start_date_typed.value).format("YYYY-MM-DD") : "")
    const mi_estimate_start_time_string: Ref<string> = ref(mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("HH:mm:ss") : "")
    const mi_estimate_end_date_typed: Ref<Date | null> = ref(mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).toDate() : null)
    const mi_estimate_end_date_string: Ref<string> = computed(() => mi_estimate_end_date_typed.value ? moment(mi_estimate_end_date_typed.value).format("YYYY-MM-DD") : "")
    const mi_estimate_end_time_string: Ref<string> = ref(mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("HH:mm:ss") : "")
    const mi_limit_date_typed: Ref<Date | null> = ref(mi.value && mi.value.limit_time ? moment(mi.value.limit_time).toDate() : null)
    const mi_limit_date_string: Ref<string> = computed(() => mi_limit_date_typed.value ? moment(mi_limit_date_typed.value).format("YYYY-MM-DD") : "")
    const mi_limit_time_string: Ref<string> = ref(mi.value && mi.value.limit_time ? moment(mi.value.limit_time).format("HH:mm:ss") : "")

    const notifications: Ref<Array<Notification>> = ref(new Array<Notification>())
    const show_start_date_menu = ref(false)
    const show_start_time_menu = ref(false)
    const show_end_date_menu = ref(false)
    const show_end_time_menu = ref(false)
    const show_limit_date_menu = ref(false)
    const show_limit_time_menu = ref(false)

    // ── Watchers ──
    watch(() => props.application_config, () => load_mi_board_names())

    // ── Init ──
    load_mi_board_names()

    // ── Business logic ──
    async function load_mi_board_names(): Promise<void> {
        const req = new GetMiBoardRequest()

        const res = await props.gkill_api.get_mi_board_list(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            // emits('received_messages', res.messages)
        }

        let is_contain_default_board = false
        res.boards.forEach((board_name) => {
            if (board_name === props.application_config.mi_default_board) {
                is_contain_default_board = true
            }
        })
        if (!is_contain_default_board) {
            res.boards.push(props.application_config.mi_default_board)
        }

        mi_board_names.value = res.boards
    }

    function update_board_name(board_name: string): void {
        mi_board_names.value.push(board_name)
        mi_board_name.value = board_name
    }

    function show_new_board_name_dialog(): void {
        new_board_name_dialog.value?.show()
    }

    function clear_estimate_start_date_time(): void {
        mi_estimate_start_date_typed.value = null
        mi_estimate_start_time_string.value = ""
    }

    function clear_estimate_end_date_time(): void {
        mi_estimate_end_date_typed.value = null
        mi_estimate_end_time_string.value = ""
    }

    function clear_limit_date_time(): void {
        mi_limit_date_typed.value = null
        mi_limit_time_string.value = ""
    }

    function now_to_estimate_start_date_time(): void {
        mi_estimate_start_date_typed.value = moment().toDate()
        mi_estimate_start_time_string.value = moment().format("HH:mm:ss")
    }

    function now_to_estimate_end_date_time(): void {
        mi_estimate_end_date_typed.value = moment().toDate()
        mi_estimate_end_time_string.value = moment().format("HH:mm:ss")
    }

    function now_to_limit_date_time(): void {
        mi_limit_date_typed.value = moment().toDate()
        mi_limit_time_string.value = moment().format("HH:mm:ss")
    }

    function reset(): void {
        mi_title.value = mi.value.title
        mi_board_name.value = props.application_config.mi_default_board
        mi_estimate_start_date_typed.value = mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).toDate() : null
        mi_estimate_start_time_string.value = mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("HH:mm:ss") : ""
        mi_estimate_end_date_typed.value = mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).toDate() : null
        mi_estimate_end_time_string.value = mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("HH:mm:ss") : ""
        mi_limit_date_typed.value = mi.value && mi.value.limit_time ? moment(mi.value.limit_time).toDate() : null
        mi_limit_time_string.value = mi.value && mi.value.limit_time ? moment(mi.value.limit_time).format("HH:mm:ss") : ""
        notifications.value.splice(0)
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // Notification チェック
            // おかしかったらnullが戻ってくるので中断する
            const notificationResults = new Array<Notification>()
            if (add_notification_views.value) {
                for (let i = 0; i < add_notification_views.value.length; i++) {
                    const notification = await add_notification_views.value[i].get_notification()
                    if (!notification) {
                        return
                    }
                    notificationResults.push(notification)
                }
            }

            // Mi チェック
            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            if (!mi.value) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_mi_is_null
                error.error_message = i18n.global.t("CLIENT_MI_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // タイトルの入力チェック
            if (mi_title.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.mi_title_is_blank
                error.error_message = i18n.global.t("MI_TITLE_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 開始日時 片方だけ入力されていたらエラーチェック
            if (mi_estimate_start_date_string.value === "" || mi_estimate_start_time_string.value === "") {//どっちも入力されていなければOK。nullとして扱う
                if ((mi_estimate_start_date_string.value === "" && mi_estimate_start_time_string.value !== "") ||
                    (mi_estimate_start_date_string.value !== "" && mi_estimate_start_time_string.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
                    const error = new GkillError()
                    error.error_code = GkillErrorCodes.mi_estimate_start_time_is_blank
                    error.error_message = i18n.global.t("MI_START_DATE_TIME_IS_BLANK_MESSAGE")
                    const errors = new Array<GkillError>()
                    errors.push(error)
                    emits('received_errors', errors)
                    return
                }
            }

            // 終了日時 片方だけ入力されていたらエラーチェック
            if (mi_estimate_end_date_string.value === "" || mi_estimate_end_time_string.value === "") {//どっちも入力されていなければOK。nullとして扱う
                if ((mi_estimate_end_date_string.value === "" && mi_estimate_end_time_string.value !== "") ||
                    (mi_estimate_end_date_string.value !== "" && mi_estimate_end_time_string.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
                    const error = new GkillError()
                    error.error_code = GkillErrorCodes.mi_estimate_end_time_is_blank
                    error.error_message = i18n.global.t("MI_END_DATE_TIME_IS_BLANK_MESSAGE")
                    const errors = new Array<GkillError>()
                    errors.push(error)
                    emits('received_errors', errors)
                    return
                }
            }

            // 期限日時 片方だけ入力されていたらエラーチェック
            if (mi_limit_date_string.value === "" || mi_limit_time_string.value === "") {//どっちも入力されていなければOK。nullとして扱う
                if ((mi_limit_date_string.value === "" && mi_limit_time_string.value !== "") ||
                    (mi_limit_date_string.value !== "" && mi_limit_time_string.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
                    const error = new GkillError()
                    error.error_code = GkillErrorCodes.mi_limit_time_is_blank
                    error.error_message = i18n.global.t("MI_LIMIT_DATE_TIME_IS_BLANK_MESSAGE")
                    const errors = new Array<GkillError>()
                    errors.push(error)
                    emits('received_errors', errors)
                    return
                }
            }

            // 更新がなかったらエラーメッセージを出力する
            if (mi.value.title === mi_title.value &&
                moment(mi.value.estimate_start_time).valueOf() === moment(mi_estimate_start_date_string.value + " " + mi_estimate_start_time_string.value).valueOf() &&
                moment(mi.value.estimate_end_time).valueOf() === moment(mi_estimate_end_date_string.value + " " + mi_estimate_end_time_string.value).valueOf() &&
                moment(mi.value.limit_time).valueOf() === moment(mi_limit_date_string.value + " " + mi_limit_time_string.value).valueOf()
            ) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.mi_is_no_update
                error.error_message = i18n.global.t("MI_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // Mi 追加

            // 作成するMi情報を用意する
            let estimate_start_time: Date | null = null
            let estimate_end_time: Date | null = null
            let limit_time: Date | null = null
            if (mi_estimate_start_date_string.value !== "" && mi_estimate_start_time_string.value !== "") {
                estimate_start_time = moment(mi_estimate_start_date_string.value + " " + mi_estimate_start_time_string.value).toDate()
            }
            if (mi_estimate_end_date_string.value !== "" && mi_estimate_end_time_string.value !== "") {
                estimate_end_time = moment(mi_estimate_end_date_string.value + " " + mi_estimate_end_time_string.value).toDate()
            }
            if (mi_limit_date_string.value !== "" && mi_limit_time_string.value !== "") {
                limit_time = moment(mi_limit_date_string.value + " " + mi_limit_time_string.value).toDate()
            }
            const new_mi = mi.value.clone()
            new_mi.id = mi.value.id
            new_mi.title = mi_title.value
            new_mi.board_name = mi_board_name.value
            new_mi.estimate_start_time = estimate_start_time
            new_mi.estimate_end_time = estimate_end_time
            new_mi.limit_time = limit_time
            new_mi.create_app = "gkill"
            new_mi.create_device = props.application_config.device
            new_mi.create_time = new Date(Date.now())
            new_mi.create_user = props.application_config.user_id
            new_mi.update_app = "gkill"
            new_mi.update_device = props.application_config.device
            new_mi.update_time = new Date(Date.now())
            new_mi.update_user = props.application_config.user_id

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_mi.id)
            const req = new AddMiRequest()
            req.mi = new_mi
            req.want_response_kyou = true
            const res = await props.gkill_api.add_mi(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }

            // Notification 追加
            for (let i = 0; i < notificationResults.length; i++) {
                // 追加リクエストを飛ばす
                await delete_gkill_kyou_cache(notificationResults[i].id)
                const notifReq = new AddNotificationRequest()
                notifReq.notification = notificationResults[i]
                const notifRes = await props.gkill_api.add_notification(notifReq)
                if (notifRes.errors && notifRes.errors.length !== 0) {
                    emits('received_errors', notifRes.errors)
                    return
                }
                if (notifRes.messages && notifRes.messages.length !== 0) {
                    emits('received_messages', notifRes.messages)
                }
            }
            emits("registered_kyou", res.added_kyou!)
            emits('requested_reload_list')
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    function add_notification(): void {
        const notification = new Notification()
        notification.id = props.gkill_api.generate_uuid()
        notification.target_id = id.value
        notification.content = mi_title.value
        notification.notification_time = new Date(0)
        if (mi_estimate_start_date_string.value !== "" && mi_estimate_start_time_string.value !== "") {
            notification.notification_time = moment(mi_estimate_start_date_string.value + " " + mi_estimate_start_time_string.value).toDate()
        }
        notifications.value.push(notification)
    }

    function delete_notification(index: number): void {
        notifications.value.splice(index, 1)
    }

    // ── CRUD relay handlers ──
    const crudRelayHandlers = {
        'received_errors': (...args: any[]) => emits('received_errors', args[0] as Array<GkillError>),
        'received_messages': (...args: any[]) => emits('received_messages', args[0] as Array<GkillMessage>),
    }

    // ── Second init call ──
    load_mi_board_names()

    // ── Return ──
    return {
        // Template refs
        new_board_name_dialog,
        add_notification_views,

        // State
        is_requested_submit,
        id,
        kyou,
        mi,
        mi_board_names,
        mi_title,
        mi_board_name,
        mi_estimate_start_date_typed,
        mi_estimate_start_date_string,
        mi_estimate_start_time_string,
        mi_estimate_end_date_typed,
        mi_estimate_end_date_string,
        mi_estimate_end_time_string,
        mi_limit_date_typed,
        mi_limit_date_string,
        mi_limit_time_string,
        notifications,
        show_start_date_menu,
        show_start_time_menu,
        show_end_date_menu,
        show_end_time_menu,
        show_limit_date_menu,
        show_limit_time_menu,

        // Business logic / template handlers
        load_mi_board_names,
        update_board_name,
        show_new_board_name_dialog,
        clear_estimate_start_date_time,
        clear_estimate_end_date_time,
        clear_limit_date_time,
        now_to_estimate_start_date_time,
        now_to_estimate_end_date_time,
        now_to_limit_date_time,
        reset,
        save,
        add_notification,
        delete_notification,

        // Event relay objects
        crudRelayHandlers,
    }
}
