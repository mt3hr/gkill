import { computed, type Ref, ref, watch } from 'vue'
import { i18n } from '@/i18n'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { ReKyouViewProps } from '@/pages/views/re-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { is_row_height } from '@/classes/kyou-row-height'
import { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'

export function useReKyouView(options: {
    props: ReKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const context_menu = ref<ComponentRef | null>(null)

    // 一覧の行として描かれているか。行では通知やリクエストが行数ぶん暴発するので抑える
    const is_row = computed(() => is_row_height(props.height))

    // ── State refs ──
    const target_kyou: Ref<Kyou> = ref(new Kyou())
    // 取得済みのtarget_id。仮想スクロールで行を使い回すときに同じ参照先を引き直さないために持つ
    let loaded_target_id = ''

    // ── Watchers ──
    watch(() => props.kyou, () => get_target_kyou())
    watch(() => props.rekyou, () => get_target_kyou())

    // ── Business logic ──
    async function get_target_kyou() {
        // 仮想スクロールの行使い回しでpropsだけ差し替わることがある。参照先が同じなら引き直さない
        if (loaded_target_id === props.rekyou.target_id) {
            return
        }
        loaded_target_id = props.rekyou.target_id

        const req = new GetKyouRequest()
        req.id = props.rekyou.target_id
        const res = await props.gkill_api.get_kyou(req)
        if (res.errors && res.errors.length !== 0) {
            // 一覧では行数ぶんスナックバーが出てしまうので、行では黙って諦める
            if (!is_row.value) {
                emits('received_errors', res.errors)
            }
            return
        }
        // 参照先が消えているとundefinedが入ってしまうので、空なら何もしない
        if (!res.kyou_histories || res.kyou_histories.length < 1) {
            if (!is_row.value) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.not_found_rekyou_target
                error.error_message = i18n.global.t('NOT_FOUND_REKYOU_TARGET_ERROR_MESSAGE')
                emits('received_errors', [error])
            }
            return
        }
        target_kyou.value = res.kyou_histories[0]
    }

    function show_context_menu(e: PointerEvent): void {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    // ── Init ──
    get_target_kyou()

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
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    // ── Return ──
    return {
        // Template refs
        context_menu,

        // State
        target_kyou,

        // Computed
        is_row,

        // Business logic
        show_context_menu,
        get_target_kyou,

        // Event relay objects
        crudRelayHandlers,
    }
}
