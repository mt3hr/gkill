import { i18n } from '@/i18n'
import { type Ref, ref, watch } from 'vue'
import type { EditTagViewProps } from '@/pages/views/edit-tag-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { UpdateTagRequest } from '@/classes/api/req_res/update-tag-request'
import { GkillError } from '@/classes/api/gkill-error'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'

export function useEditTagView(options: {
    props: EditTagViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const cloned_tag: Ref<Tag> = ref(props.tag.clone())
    const tag_name: Ref<string> = ref(props.tag.tag)
    const show_kyou: Ref<boolean> = ref(false)

    // ── Watchers ──
    watch([() => props.kyou, () => props.tag], () => load())

    // ── Business logic ──
    async function load(): Promise<void> {
        cloned_kyou.value = props.kyou.clone()
        await cloned_kyou.value.reload(false, true)
        await cloned_kyou.value.load_typed_datas()
        await cloned_kyou.value.load_all()
        cloned_tag.value = props.tag.clone()
        tag_name.value = cloned_tag.value.tag
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // 値がなかったらエラーメッセージを出力する
            if (tag_name.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.tag_is_blank
                error.error_message = i18n.global.t("TAG_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新がなかったらエラーメッセージを出力する
            if (cloned_tag.value.tag === tag_name.value) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.tag_is_no_update
                error.error_message = i18n.global.t("TAG_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新後タグ情報を用意する
            const updated_tag = cloned_tag.value.clone()
            updated_tag.tag = tag_name.value
            updated_tag.update_app = "gkill"
            updated_tag.update_device = props.application_config.device
            updated_tag.update_time = new Date(Date.now())
            updated_tag.update_user = props.application_config.user_id

            // 更新リクエストを飛ばす
            await delete_gkill_kyou_cache(updated_tag.id)
            await delete_gkill_kyou_cache(updated_tag.target_id)
            const req = new UpdateTagRequest()
            req.tag = updated_tag
            const res = await props.gkill_api.update_tag(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits("updated_tag", res.updated_tag)
            emits('requested_reload_kyou', props.kyou)
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
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
        // State
        is_requested_submit,
        cloned_kyou,
        cloned_tag,
        tag_name,
        show_kyou,

        // Business logic
        save,

        // Event relay objects
        crudRelayHandlers,
    }
}
