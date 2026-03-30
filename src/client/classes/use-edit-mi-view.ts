import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import type { EditMiViewProps } from '@/pages/views/edit-mi-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import moment from 'moment'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { ComponentRef } from '@/classes/component-ref'

export function useEditMiView(options: {
    props: EditMiViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const new_board_name_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const is_requested_submit = ref(false)

    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const show_kyou: Ref<boolean> = ref(false)
    const mi_board_names: Ref<Array<string>> = ref(props.application_config.mi_default_board !== "" ? [props.application_config.mi_default_board] : [])

    const mi_title: Ref<string> = ref(cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.title : "")
    const mi_board_name: Ref<string> = ref(cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.board_name : "")
    const mi_estimate_start_date_typed: Ref<Date | null> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).toDate() : null)
    const mi_estimate_start_date_string: Ref<string> = computed(() => mi_estimate_start_date_typed.value ? moment(mi_estimate_start_date_typed.value).format("YYYY-MM-DD") : "")
    const mi_estimate_start_time_string: Ref<string> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("HH:mm:ss") : "")
    const mi_estimate_end_date_typed: Ref<Date | null> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).toDate() : null)
    const mi_estimate_end_date_string: Ref<string> = computed(() => mi_estimate_end_date_typed.value ? moment(mi_estimate_end_date_typed.value).format("YYYY-MM-DD") : "")
    const mi_estimate_end_time_string: Ref<string> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("HH:mm:ss") : "")
    const mi_limit_date_typed: Ref<Date | null> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).toDate() : null)
    const mi_limit_date_string: Ref<string> = computed(() => mi_limit_date_typed.value ? moment(mi_limit_date_typed.value).format("YYYY-MM-DD") : "")
    const mi_limit_time_string: Ref<string> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("HH:mm:ss") : "")

    const show_start_date_menu = ref(false)
    const show_start_time_menu = ref(false)
    const show_end_date_menu = ref(false)
    const show_end_time_menu = ref(false)
    const show_limit_date_menu = ref(false)
    const show_limit_time_menu = ref(false)

    // ── Watchers ──
    watch(() => props.kyou, () => load())

    // ── Business logic ──
    async function load(): Promise<void> {
        cloned_kyou.value = props.kyou.clone()
        await cloned_kyou.value.reload(false, true)
        await cloned_kyou.value.load_typed_datas()
        await cloned_kyou.value.load_all()
        mi_title.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.title : ""
        mi_board_name.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.board_name : ""
        mi_estimate_start_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).toDate() : null
        mi_estimate_start_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("HH:mm:ss") : ""
        mi_estimate_end_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).toDate() : null
        mi_estimate_end_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("HH:mm:ss") : ""
        mi_limit_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).toDate() : null
        mi_limit_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("HH:mm:ss") : ""
    }

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

    function reset_estimate_start_date_time(): void {
        mi_estimate_start_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).toDate() : null
        mi_estimate_start_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("HH:mm:ss") : ""
    }

    function reset_estimate_end_date_time(): void {
        mi_estimate_end_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).toDate() : null
        mi_estimate_end_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("HH:mm:ss") : ""
    }

    function reset_limit_date_time(): void {
        mi_limit_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).toDate() : null
        mi_limit_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("HH:mm:ss") : ""
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
        mi_title.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.title : ""
        mi_board_name.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.board_name : ""
        mi_estimate_start_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).toDate() : null
        mi_estimate_start_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("HH:mm:ss") : ""
        mi_estimate_end_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).toDate() : null
        mi_estimate_end_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("HH:mm:ss") : ""
        mi_limit_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).toDate() : null
        mi_limit_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("HH:mm:ss") : ""
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            cloned_kyou.value.abort_controller.abort()
            cloned_kyou.value.abort_controller = new AbortController()

            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            const mi = cloned_kyou.value.typed_mi
            if (!mi) {
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
            if (mi.title === mi_title.value &&
                mi.board_name === mi_board_name.value &&
                (moment(mi.estimate_start_time).toDate().getTime() === moment(mi_estimate_start_date_string.value + " " + mi_estimate_start_time_string.value).toDate().getTime() || (mi.estimate_start_time == null && mi_estimate_start_date_string.value === "" && mi_estimate_start_time_string.value === "")) &&
                (moment(mi.estimate_end_time).toDate().getTime() === moment(mi_estimate_end_date_string.value + " " + mi_estimate_end_time_string.value).toDate().getTime() || (mi.estimate_end_time == null && mi_estimate_end_date_string.value === "" && mi_estimate_end_time_string.value === "")) &&
                (moment(mi.limit_time).toDate().getTime() === moment(mi_limit_date_string.value + " " + mi_limit_time_string.value).toDate().getTime() || (mi.limit_time == null && mi_limit_date_string.value === "" && mi_limit_time_string.value === ""))) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.mi_is_no_update
                error.error_message = i18n.global.t("MI_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新後Mi情報を用意する
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
            const updated_mi = mi.clone()
            updated_mi.title = mi_title.value
            updated_mi.board_name = mi_board_name.value
            updated_mi.estimate_start_time = estimate_start_time
            updated_mi.estimate_end_time = estimate_end_time
            updated_mi.limit_time = limit_time
            updated_mi.update_app = "gkill"
            updated_mi.update_device = props.application_config.device
            updated_mi.update_time = new Date(Date.now())
            updated_mi.update_user = props.application_config.user_id

            // 更新リクエストを飛ばす
            await delete_gkill_kyou_cache(updated_mi.id)
            const req = new UpdateMiRequest()
            req.mi = updated_mi
            req.want_response_kyou = true
            const res = await props.gkill_api.update_mi(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits("updated_kyou", res.updated_kyou!)
            emits('requested_reload_kyou', props.kyou)
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    // ── Event relay objects ──
    const crudRelayHandlers = {
        'deleted_kyou': (kyou: Kyou) => emits('deleted_kyou', kyou),
        'deleted_tag': (tag: Tag) => emits('deleted_tag', tag),
        'deleted_text': (text: Text) => emits('deleted_text', text),
        'deleted_notification': (notification: Notification) => emits('deleted_notification', notification),
        'registered_kyou': (kyou: Kyou) => emits('registered_kyou', kyou),
        'registered_tag': (tag: Tag) => emits('registered_tag', tag),
        'registered_text': (text: Text) => emits('registered_text', text),
        'registered_notification': (notification: Notification) => emits('registered_notification', notification),
        'updated_kyou': (kyou: Kyou) => emits('updated_kyou', kyou),
        'updated_tag': (tag: Tag) => emits('updated_tag', tag),
        'updated_text': (text: Text) => emits('updated_text', text),
        'updated_notification': (notification: Notification) => emits('updated_notification', notification),
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_reload_kyou': (kyou: Kyou) => emits('requested_reload_kyou', kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (kyous: Array<Kyou>, checked: boolean) => emits('requested_update_check_kyous', kyous, checked),
    }

    const newBoardNameDialogHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'setted_new_board_name': (value: string) => update_board_name(value),
    }

    // ── Init calls ──
    load()
    load_mi_board_names()

    // ── Return ──
    return {
        // Template refs
        new_board_name_dialog,

        // State
        is_requested_submit,
        cloned_kyou,
        show_kyou,
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
        show_start_date_menu,
        show_start_time_menu,
        show_end_date_menu,
        show_end_time_menu,
        show_limit_date_menu,
        show_limit_time_menu,

        // Business logic
        show_new_board_name_dialog,
        clear_estimate_start_date_time,
        clear_estimate_end_date_time,
        clear_limit_date_time,
        reset_estimate_start_date_time,
        reset_estimate_end_date_time,
        reset_limit_date_time,
        now_to_estimate_start_date_time,
        now_to_estimate_end_date_time,
        now_to_limit_date_time,
        reset,
        save,

        // Event relay objects
        crudRelayHandlers,
        newBoardNameDialogHandlers,
    }
}
