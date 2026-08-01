import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import moment from 'moment'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import type { GkillPropsBase } from '@/pages/views/gkill-props-base'
import type { ComponentRef } from '@/classes/component-ref'

/**
 * MiReKyouの板名・開始/終了/期限のフォーム状態をまとめたもの。
 * 追加ダイアログと編集ダイアログで同じ入力欄を使うため共通化している。
 * MiReKyouはタイトルを持たないため、タイトル欄は扱わない。
 */
export function useMiReKyouScheduleFields(options: {
    props: GkillPropsBase,
    emits: {
        (e: 'received_errors', errors: Array<GkillError>): void
    },
    /** 板名の初期値。編集時は既存のMiReKyouの板名を渡す */
    default_board_name?: string,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const new_board_name_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const mi_board_names: Ref<Array<string>> = ref(props.application_config.mi_default_board !== "" ? [props.application_config.mi_default_board] : [])
    const mi_board_name: Ref<string> = ref(options.default_board_name ?? (props.application_config.mi_default_board !== "" ? props.application_config.mi_default_board : "Inbox"))

    const mi_estimate_start_date_typed: Ref<Date | null> = ref(null)
    const mi_estimate_start_date_string: Ref<string> = computed(() => mi_estimate_start_date_typed.value ? moment(mi_estimate_start_date_typed.value).format("YYYY-MM-DD") : "")
    const mi_estimate_start_time_string: Ref<string> = ref("")
    const mi_estimate_end_date_typed: Ref<Date | null> = ref(null)
    const mi_estimate_end_date_string: Ref<string> = computed(() => mi_estimate_end_date_typed.value ? moment(mi_estimate_end_date_typed.value).format("YYYY-MM-DD") : "")
    const mi_estimate_end_time_string: Ref<string> = ref("")
    const mi_limit_date_typed: Ref<Date | null> = ref(null)
    const mi_limit_date_string: Ref<string> = computed(() => mi_limit_date_typed.value ? moment(mi_limit_date_typed.value).format("YYYY-MM-DD") : "")
    const mi_limit_time_string: Ref<string> = ref("")

    const show_start_date_menu = ref(false)
    const show_start_time_menu = ref(false)
    const show_end_date_menu = ref(false)
    const show_end_time_menu = ref(false)
    const show_limit_date_menu = ref(false)
    const show_limit_time_menu = ref(false)

    // ── Business logic ──
    async function load_mi_board_names(): Promise<void> {
        const req = new GetMiBoardRequest()

        const res = await props.gkill_api.get_mi_board_list(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
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

    /** 既存のMiReKyouの値をフォームへ反映する */
    function apply_times(times: { estimate_start_time: Date | null, estimate_end_time: Date | null, limit_time: Date | null }): void {
        mi_estimate_start_date_typed.value = times.estimate_start_time ? moment(times.estimate_start_time).toDate() : null
        mi_estimate_start_time_string.value = times.estimate_start_time ? moment(times.estimate_start_time).format("HH:mm:ss") : ""
        mi_estimate_end_date_typed.value = times.estimate_end_time ? moment(times.estimate_end_time).toDate() : null
        mi_estimate_end_time_string.value = times.estimate_end_time ? moment(times.estimate_end_time).format("HH:mm:ss") : ""
        mi_limit_date_typed.value = times.limit_time ? moment(times.limit_time).toDate() : null
        mi_limit_time_string.value = times.limit_time ? moment(times.limit_time).format("HH:mm:ss") : ""
    }

    /**
     * 日付と時刻の片方だけ入力されていないか検証する。
     * 問題があればエラーをemitしてfalseを返す。
     */
    function validate(): boolean {
        const checks: Array<{ date: string, time: string, code: GkillErrorCodes, message_key: string }> = [
            { date: mi_estimate_start_date_string.value, time: mi_estimate_start_time_string.value, code: GkillErrorCodes.mi_estimate_start_time_is_blank, message_key: "MI_START_DATE_TIME_IS_BLANK_MESSAGE" },
            { date: mi_estimate_end_date_string.value, time: mi_estimate_end_time_string.value, code: GkillErrorCodes.mi_estimate_end_time_is_blank, message_key: "MI_END_DATE_TIME_IS_BLANK_MESSAGE" },
            { date: mi_limit_date_string.value, time: mi_limit_time_string.value, code: GkillErrorCodes.mi_limit_time_is_blank, message_key: "MI_LIMIT_DATE_TIME_IS_BLANK_MESSAGE" },
        ]
        for (const check of checks) {
            // どちらも未入力ならnull扱いでOK。片方だけならエラー
            if ((check.date === "" && check.time !== "") || (check.date !== "" && check.time === "")) {
                const error = new GkillError()
                error.error_code = check.code
                error.error_message = i18n.global.t(check.message_key)
                emits('received_errors', [error])
                return false
            }
        }
        return true
    }

    /** フォームの入力値から日時を組み立てる */
    function resolve_times(): { estimate_start_time: Date | null, estimate_end_time: Date | null, limit_time: Date | null } {
        return {
            estimate_start_time: (mi_estimate_start_date_string.value !== "" && mi_estimate_start_time_string.value !== "")
                ? moment(mi_estimate_start_date_string.value + " " + mi_estimate_start_time_string.value).toDate() : null,
            estimate_end_time: (mi_estimate_end_date_string.value !== "" && mi_estimate_end_time_string.value !== "")
                ? moment(mi_estimate_end_date_string.value + " " + mi_estimate_end_time_string.value).toDate() : null,
            limit_time: (mi_limit_date_string.value !== "" && mi_limit_time_string.value !== "")
                ? moment(mi_limit_date_string.value + " " + mi_limit_time_string.value).toDate() : null,
        }
    }

    return {
        // Template refs
        new_board_name_dialog,

        // State
        mi_board_names,
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
        load_mi_board_names,
        update_board_name,
        show_new_board_name_dialog,
        clear_estimate_start_date_time,
        clear_estimate_end_date_time,
        clear_limit_date_time,
        now_to_estimate_start_date_time,
        now_to_estimate_end_date_time,
        now_to_limit_date_time,
        apply_times,
        validate,
        resolve_times,
    }
}
