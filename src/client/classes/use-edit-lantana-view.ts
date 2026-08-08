import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import type { EditLantanaViewProps } from '@/pages/views/edit-lantana-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateLantanaRequest } from '@/classes/api/req_res/update-lantana-request'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { ComponentRef } from '@/classes/component-ref'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useEditLantanaView(options: {
    props: EditLantanaViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const edit_lantana_flowers = ref<ComponentRef | null>(null)

    // ── State refs ──
    const is_loading = ref(true)
    const is_requested_submit = ref(false)
    const is_busy = computed(() => is_loading.value || is_requested_submit.value)

    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const mood: Ref<number> = ref(props.kyou.typed_lantana?.mood ?? 0)
    const related_date_typed: Ref<Date> = ref(moment(props.kyou.related_time).toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment(props.kyou.related_time).format("HH:mm:ss"))
    const show_kyou: Ref<boolean> = ref(false)

    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Business logic ──
    async function load(): Promise<void> {
        try {
            is_loading.value = true
            cloned_kyou.value = props.kyou.clone()
            await cloned_kyou.value.load_typed_datas()
            mood.value = cloned_kyou.value.typed_lantana ? cloned_kyou.value.typed_lantana!.mood : 0
            related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
            related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
        } finally {
            is_loading.value = false
        }
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            cloned_kyou.value.abort_controller.abort()
            cloned_kyou.value.abort_controller = new AbortController()

            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            const lantana = cloned_kyou.value.typed_lantana
            if (!lantana) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_lantana_is_null
                error.error_message = i18n.global.t("CLIENT_LANTANA_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 日時必須入力チェック
            if (related_date_string.value === "" || related_time_string.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.lantana_related_time_is_blank
                error.error_message = i18n.global.t("LANTANA_DATE_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新がなかったらエラーメッセージを出力する。
            // 関連日時もこの画面で編集できるので比較に含める（含めないと日時だけ変えても保存できない）
            if (lantana.mood === await edit_lantana_flowers.value?.get_mood() &&
                moment(lantana.related_time).toDate().getTime() === moment(related_date_string.value + " " + related_time_string.value).toDate().getTime()) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.lantana_is_no_update
                error.error_message = i18n.global.t("LANTANA_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新後Lantana情報を用意する
            const updated_lantana = lantana.clone()
            // refがnullでも例外にしない。throwするとダイアログが閉じないまま固まる
            updated_lantana.mood = await edit_lantana_flowers.value?.get_mood() ?? lantana.mood
            updated_lantana.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
            updated_lantana.update_app = "gkill"
            updated_lantana.update_device = props.application_config.device
            updated_lantana.update_time = new Date(Date.now())
            updated_lantana.update_user = props.application_config.user_id

            // 更新リクエストを飛ばす
            await delete_gkill_kyou_cache(updated_lantana.id)
            const req = new UpdateLantanaRequest()
            req.want_response_kyou = true
            req.lantana = updated_lantana

            const res = await props.gkill_api.update_lantana(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits('updated_kyou', res.updated_kyou!)
            emits('requested_reload_kyou', props.kyou)
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    function now_to_related_date_time(): void {
        related_date_typed.value = moment().toDate()
        related_time_string.value = moment().format("HH:mm:ss")
    }

    function reset_related_date_time(): void {
        related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
        related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
    }

    function reset(): void {
        mood.value = cloned_kyou.value.typed_lantana ? cloned_kyou.value.typed_lantana.mood : 0
        related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
        related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
    }

    // ── Event relay objects ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Init calls ──
    load()

    // ── Return ──
    return {
        // Template refs
        edit_lantana_flowers,

        // State
        is_loading,
        is_requested_submit,
        is_busy,
        cloned_kyou,
        mood,
        related_date_typed,
        related_date_string,
        related_time_string,
        show_kyou,
        show_related_date_menu,
        show_related_time_menu,

        // Business logic
        now_to_related_date_time,
        reset_related_date_time,
        reset,
        save,

        // Event relay objects
        crudRelayHandlers,
    }
}

