import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { MiReKyouViewProps } from '@/pages/views/mi-re-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import { UpdateMiReKyouRequest } from '@/classes/api/req_res/update-mi-re-kyou-request'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { Kyou } from '@/classes/datas/kyou'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'

export function useMiReKyouView(options: {
    props: MiReKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const context_menu = ref<ComponentRef | null>(null)

    // タッチデバイス（モバイル）ではドラッグを無効にする。
    // ロングプレスでcontextmenuイベントを発火させるため。
    const is_mobile = 'ontouchstart' in window || navigator.maxTouchPoints > 0
    const effective_draggable = computed(() => is_mobile ? false : (props.draggable ?? false))

    // ── State refs ──
    const target_kyou: Ref<Kyou> = ref(new Kyou())
    const is_checked_mi: Ref<boolean> = ref(props.mirekyou.is_checked)

    // ── Watchers ──
    watch(() => props.kyou, () => get_target_kyou())
    watch(() => props.mirekyou, () => {
        is_checked_mi.value = props.mirekyou.is_checked
        get_target_kyou()
    })

    // ── Business logic ──
    async function get_target_kyou() {
        const req = new GetKyouRequest()
        req.id = props.mirekyou.target_id
        const res = await props.gkill_api.get_kyou(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (!res.kyou_histories || res.kyou_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_mi_rekyou_target
            error.error_message = i18n.global.t('NOT_FOUND_MI_REKYOU_TARGET_ERROR_MESSAGE')
            emits('received_errors', [error])
            return
        }
        target_kyou.value = res.kyou_histories[0]
    }

    function show_context_menu(e: PointerEvent): void {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    async function clicked_mi_check(): Promise<void> {
        // 読み取り専用表示だったら何もしない
        if (props.is_readonly_mi_check) {
            return
        }

        is_checked_mi.value = !is_checked_mi.value

        // 更新がなかったらエラーメッセージを出力する
        if (props.mirekyou.is_checked === is_checked_mi.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.mi_rekyou_is_no_update
            error.error_message = i18n.global.t("MI_REKYOU_IS_NO_UPDATE_MESSAGE")
            emits('received_errors', [error])
            return
        }

        // 更新後mirekyou情報を用意する
        const updated_mirekyou = props.mirekyou.clone()
        updated_mirekyou.is_checked = is_checked_mi.value
        updated_mirekyou.update_app = "gkill"
        updated_mirekyou.update_device = props.application_config.device
        updated_mirekyou.update_time = new Date(Date.now())
        updated_mirekyou.update_user = props.application_config.user_id

        // 更新リクエストを飛ばす
        await delete_gkill_kyou_cache(updated_mirekyou.id)
        const req = new UpdateMiReKyouRequest()
        req.mirekyou = updated_mirekyou
        req.want_response_kyou = true

        const res = await props.gkill_api.update_mirekyou(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits('requested_reload_kyou', props.kyou)
        return
    }

    function on_drag_start(e: DragEvent) {
        e.dataTransfer!.setData("gkill_mi_re_kyou", JSON.stringify(props.mirekyou))
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
        is_checked_mi,
        effective_draggable,

        // Business logic
        show_context_menu,
        get_target_kyou,
        clicked_mi_check,
        on_drag_start,

        // Event relay objects
        crudRelayHandlers,
    }
}
