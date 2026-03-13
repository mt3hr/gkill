import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import type { EditLantanaViewProps } from '@/pages/views/edit-lantana-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateLantanaRequest } from '@/classes/api/req_res/update-lantana-request'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'

export function useEditLantanaView(options: {
    props: EditLantanaViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const edit_lantana_flowers = ref<any>(null)

    // ── State refs ──
    const is_requested_submit = ref(false)

    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const mood: Ref<Number> = ref(props.kyou.typed_lantana?.mood ?? 0)
    const related_date_typed: Ref<Date> = ref(moment(props.kyou.related_time).toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment(props.kyou.related_time).format("HH:mm:ss"))
    const show_kyou: Ref<boolean> = ref(false)

    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Watchers ──
    watch(() => props.kyou, () => load())

    // ── Business logic ──
    async function load(): Promise<void> {
        cloned_kyou.value = props.kyou.clone()
        await cloned_kyou.value.reload(false, true)
        await cloned_kyou.value.load_typed_datas()
        await cloned_kyou.value.load_all()
        mood.value = cloned_kyou.value.typed_lantana ? cloned_kyou.value.typed_lantana!.mood : 0
        related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
        related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
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

            // 更新がなかったらエラーメッセージを出力する
            if (lantana.mood === await edit_lantana_flowers.value?.get_mood()) {
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
            updated_lantana.mood = await edit_lantana_flowers.value!.get_mood()
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
    const crudRelayHandlers = {
        'deleted_kyou': (...args: any[]) => emits('deleted_kyou', args[0] as Kyou),
        'deleted_tag': (...args: any[]) => emits('deleted_tag', args[0] as Tag),
        'deleted_text': (...args: any[]) => emits('deleted_text', args[0] as Text),
        'deleted_notification': (...args: any[]) => emits('deleted_notification', args[0] as Notification),
        'registered_kyou': (...args: any[]) => emits('registered_kyou', args[0] as Kyou),
        'registered_tag': (...args: any[]) => emits('registered_tag', args[0] as Tag),
        'registered_text': (...args: any[]) => emits('registered_text', args[0] as Text),
        'registered_notification': (...args: any[]) => emits('registered_notification', args[0] as Notification),
        'updated_kyou': (...args: any[]) => emits('updated_kyou', args[0] as Kyou),
        'updated_tag': (...args: any[]) => emits('updated_tag', args[0] as Tag),
        'updated_text': (...args: any[]) => emits('updated_text', args[0] as Text),
        'updated_notification': (...args: any[]) => emits('updated_notification', args[0] as Notification),
        'received_errors': (...args: any[]) => emits('received_errors', args[0] as Array<GkillError>),
        'received_messages': (...args: any[]) => emits('received_messages', args[0] as Array<GkillMessage>),
        'requested_reload_kyou': (...args: any[]) => emits('requested_reload_kyou', args[0] as Kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (...args: any[]) => emits('requested_update_check_kyous', args[0] as Array<Kyou>, args[1] as boolean),
    }

    // ── Init calls ──
    load()

    // ── Return ──
    return {
        // Template refs
        edit_lantana_flowers,

        // State
        is_requested_submit,
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
