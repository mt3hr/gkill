import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { MiKyouViewProps } from '@/pages/views/mi-kyou-view-props'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { ComponentRef } from '@/classes/component-ref'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'
import { useDeviceKind } from '@/classes/use-device-kind'

export function useMiKyouView(options: {
    props: MiKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const context_menu = ref<ComponentRef | null>(null)

    // ドラッグ&ドロップはPCでのみ有効にする。
    // タブレット・スマートフォンでは長押しでcontextmenuイベントを発火させるため。
    const { is_pc } = useDeviceKind()
    const effective_draggable = computed(() => is_pc.value && (props.draggable ?? false))

    // ── State refs ──
    const is_requested_submit = ref(false)
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const is_checked_mi: Ref<boolean> = ref(props.kyou.typed_mi ? props.kyou.typed_mi.is_checked : false)

    // ── Watchers ──
    watch(() => props.kyou, async () => {
        await load_cloned_kyou()
        is_checked_mi.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.is_checked : false
    })

    // ── Business logic ──
    async function load_cloned_kyou() {
        const kyou = props.kyou.clone()
        await kyou.load_typed_datas()
        cloned_kyou.value = kyou
    }

    function show_context_menu(e: PointerEvent): void {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    // 一覧上のチェックはそのままサーバ更新に繋がる。連打で同じ更新が重なるのを防ぐ
    async function clicked_mi_check(): Promise<void> {
        // 読み取り専用表示だったら何もしない
        if (props.is_readonly_mi_check) {
            return
        }
        if (is_requested_submit.value) {
            return
        }
        is_requested_submit.value = true
        try {
            await update_mi_check()
        } finally {
            is_requested_submit.value = false
        }
    }

    async function update_mi_check(): Promise<void> {
        is_checked_mi.value = !is_checked_mi.value

        cloned_kyou.value = props.kyou.clone()
        await cloned_kyou.value.load_typed_datas()

        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        const mi = cloned_kyou.value.typed_mi
        if (!mi) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.client_mi_is_null
            error.error_message = i18n.global.t("CLIENT_MI_IS_NULL_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 更新がなかったらエラーメッセージを出力する
        if (mi.is_checked === is_checked_mi.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.mi_is_no_update
            error.error_message = i18n.global.t("MI_IS_NO_UPDATE_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 更新後mi情報を用意する
        const updated_mi = mi.clone()
        updated_mi.is_checked = is_checked_mi.value
        updated_mi.update_app = "gkill"
        updated_mi.update_device = props.application_config.device
        updated_mi.update_time = new Date(Date.now())
        updated_mi.update_user = props.application_config.user_id

        // 更新リクエストを飛ばす
        await delete_gkill_kyou_cache(updated_mi.id)
        const req = new UpdateMiRequest()
        req.mi = updated_mi
        req.want_response_kyou = true

        const res = await props.gkill_api.update_mi(req)
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

    function onDragStart(e: DragEvent) {
        e.dataTransfer!.setData("gkill_mi", JSON.stringify(props.kyou.typed_mi))
    }

    // ── Event relay objects ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Return ──
    return {
        // Template refs
        context_menu,

        // State
        cloned_kyou,
        is_requested_submit,
        is_checked_mi,
        effective_draggable,

        // Business logic
        show_context_menu,
        clicked_mi_check,
        onDragStart,

        // Event relay objects
        crudRelayHandlers,
    }
}

