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
