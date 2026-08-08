import { type Ref, ref, watch } from 'vue'
import type { Kyou } from '@/classes/datas/kyou'
import { build_cascade_delete_failed_error, build_deleted_kyou_stub, cascade_delete_kyou } from '@/classes/cascade-delete-kyou'
import type { ConfirmDeleteKyouViewProps } from '@/pages/views/confirm-delete-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useConfirmDeleteKyouView(options: {
    props: ConfirmDeleteKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const show_kyou: Ref<boolean> = ref(true)

    // ── Watchers ──
    watch(() => props.kyou, () => cloned_kyou.value = props.kyou.clone())

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Delete logic ──
    //
    // 何があってもダイアログを閉じる。削除リクエストはサーバに届いているのに例外で
    // クローズまで到達せず、「消えているのに閉じない」状態になるのを防ぐため、
    // クローズはfinallyに置く（エラー時も閉じるのは元の挙動どおり）。
    async function delete_kyou(): Promise<void> {
        if (is_requested_submit.value) {
            return
        }
        is_requested_submit.value = true
        try {
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
        } catch (err: unknown) {
            console.error(err)
            emits('received_errors', [build_cascade_delete_failed_error()])
        } finally {
            is_requested_submit.value = false
            emits('requested_close_dialog')
        }
    }

    // ── Return ──
    return {
        // State
        is_requested_submit,
        cloned_kyou,
        show_kyou,

        // Methods
        delete_kyou,

        // Event relay objects
        crudRelayHandlers,
    }
}

