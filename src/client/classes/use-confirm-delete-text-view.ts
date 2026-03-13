import { computed, type Ref, ref } from 'vue'
import type { InfoIdentifier } from '@/classes/datas/info-identifier'
import { UpdateTextRequest } from '@/classes/api/req_res/update-text-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { ConfirmDeleteTextViewProps } from '@/pages/views/confirm-delete-text-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useConfirmDeleteTextView(options: {
    props: ConfirmDeleteTextViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const show_kyou: Ref<boolean> = ref(true)

    const text_highlight_targets = computed<Array<InfoIdentifier>>(() => {
        const info_identifer = props.text.generate_info_identifer()
        return [info_identifer]
    })

    // ── CRUD relay handlers ──
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

    // ── Delete logic ──
    async function delete_text(): Promise<void> {
        // 更新後テキスト情報を用意する
        const updated_text = props.text.clone()
        updated_text.is_deleted = true
        updated_text.update_app = "gkill"
        updated_text.update_device = props.application_config.device
        updated_text.update_time = new Date(Date.now())
        updated_text.update_user = props.application_config.user_id

        // 更新リクエストを飛ばす
        await delete_gkill_kyou_cache(updated_text.id)
        await delete_gkill_kyou_cache(updated_text.target_id)
        const req = new UpdateTextRequest()
        req.text = updated_text
        const res = await props.gkill_api.update_text(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits('deleted_text', res.updated_text)
        emits('requested_reload_kyou', props.kyou)
        emits('requested_close_dialog')
        return
    }

    // ── Return ──
    return {
        // State
        show_kyou,
        text_highlight_targets,

        // Methods
        delete_text,

        // Event relay objects
        crudRelayHandlers,
    }
}
