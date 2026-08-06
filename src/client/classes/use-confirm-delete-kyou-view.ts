import { type Ref, ref, watch } from 'vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { GkillError } from '@/classes/api/gkill-error'
import { build_deleted_kyou_stub, cascade_delete_kyou } from '@/classes/cascade-delete-kyou'
import type { ConfirmDeleteKyouViewProps } from '@/pages/views/confirm-delete-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'

export function useConfirmDeleteKyouView(options: {
    props: ConfirmDeleteKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const show_kyou: Ref<boolean> = ref(true)

    // ── Watchers ──
    watch(() => props.kyou, () => cloned_kyou.value = props.kyou.clone())

    // ── CRUD relay handlers ──
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

    // ── Delete logic ──
    async function delete_kyou(): Promise<void> {
        // Kyou自身だけでなく、付随するTag/Text/Notificationと、
        // それを参照しているReKyou/MiReKyouも連鎖して消す
        const result = await cascade_delete_kyou({
            kyou: cloned_kyou.value,
            gkill_api: props.gkill_api,
            application_config: props.application_config,
        })

        if (result.errors.length !== 0) {
            emits('received_errors', result.errors)
        }

        // 消したid分だけ画面から取り除く。全列再検索(requested_reload_list)には頼らない。
        // 受け手はidしか見ないので、Kyou自身以外はidだけのstubで足りる
        for (let i = 0; i < result.deleted_ids.length; i++) {
            const deleted_id = result.deleted_ids[i]
            if (deleted_id === cloned_kyou.value.id) {
                emits('deleted_kyou', cloned_kyou.value)
                continue
            }
            emits('deleted_kyou', build_deleted_kyou_stub(deleted_id))
        }

        emits('requested_close_dialog')
    }

    // ── Return ──
    return {
        // State
        cloned_kyou,
        show_kyou,

        // Methods
        delete_kyou,

        // Event relay objects
        crudRelayHandlers,
    }
}
