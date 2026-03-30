import { i18n } from '@/i18n'
import { computed, ref, type Ref } from 'vue'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { GkillError } from '@/classes/api/gkill-error'
import moment from 'moment'
import { Lantana } from '@/classes/datas/lantana'
import type { AddLantanaViewProps } from '@/pages/views/add-lantana-view-props'
import { AddLantanaRequest } from '@/classes/api/req_res/add-lantana-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { ComponentRef } from '@/classes/component-ref'

export function useAddLantanaView(options: {
    props: AddLantanaViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const edit_lantana_flowers = ref<ComponentRef | null>(null)

    // ── State refs ──
    const is_requested_submit = ref(false)

    const lantana: Ref<Lantana> = ref((() => {
        const lantana = new Lantana()
        lantana.related_time = new Date(Date.now())
        return lantana
    })())
    const mood: Ref<number> = ref(lantana.value.mood)
    const related_date_typed: Ref<Date> = ref(moment().toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment().format("HH:mm:ss"))
    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Business logic ──
    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            if (!lantana.value) {
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
            if (lantana.value.mood === await edit_lantana_flowers.value?.get_mood()) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.lantana_is_no_update
                error.error_message = i18n.global.t("LANTANA_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 追加するLantana情報を用意する
            const new_lantana = lantana.value.clone()
            new_lantana.id = props.gkill_api.generate_uuid()
            new_lantana.mood = await edit_lantana_flowers.value!.get_mood()
            new_lantana.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
            new_lantana.create_app = "gkill"
            new_lantana.create_device = props.application_config.device
            new_lantana.create_time = new Date(Date.now())
            new_lantana.create_user = props.application_config.user_id
            new_lantana.update_app = "gkill"
            new_lantana.update_device = props.application_config.device
            new_lantana.update_time = new Date(Date.now())
            new_lantana.update_user = props.application_config.user_id

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_lantana.id)
            const req = new AddLantanaRequest()
            req.lantana = new_lantana
            req.want_response_kyou = true
            const res = await props.gkill_api.add_lantana(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits('registered_kyou', res.added_kyou!)
            emits('requested_reload_list')
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    function reset_related_date_time(): void {
        related_date_typed.value = moment(lantana.value.related_time).toDate()
        related_time_string.value = moment(lantana.value.related_time).format("HH:mm:ss")
    }

    function now_to_related_date_time(): void {
        related_date_typed.value = moment().toDate()
        related_time_string.value = moment().format("HH:mm:ss")
    }

    function reset(): void {
        mood.value = lantana.value.mood
        related_date_typed.value = moment().toDate()
        related_time_string.value = moment().format("HH:mm:ss")
    }

    // ── Template event handlers ──
    function onCloseDateMenu(): void {
        show_related_date_menu.value = false
    }

    function onCloseTimeMenu(): void {
        show_related_time_menu.value = false
    }

    return {
        // Template refs
        edit_lantana_flowers,

        // State
        is_requested_submit,
        mood,
        related_date_typed,
        related_date_string,
        related_time_string,
        show_related_date_menu,
        show_related_time_menu,

        // Business logic
        save,
        reset_related_date_time,
        now_to_related_date_time,
        reset,

        // Template event handlers
        onCloseDateMenu,
        onCloseTimeMenu,
    }
}
