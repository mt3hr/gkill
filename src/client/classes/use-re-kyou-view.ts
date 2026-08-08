import { computed, type Ref, ref, watch } from 'vue'
import { i18n } from '@/i18n'
import type { ReKyouViewProps } from '@/pages/views/re-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { is_row_height } from '@/classes/kyou-row-height'
import { Kyou } from '@/classes/datas/kyou'
import type { ComponentRef } from '@/classes/component-ref'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

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
    // 参照先が見つからなかったか。終端状態として持たないと、
    // 中身の入らないKyouViewが読み込み中表示のまま止まってしまう
    const is_target_not_found: Ref<boolean> = ref(false)
    // 取得済みのtarget_id。仮想スクロールで行を使い回すときに同じ参照先を引き直さないために持つ
    let loaded_target_id = ''

    // ── Watchers ──
    watch(() => props.kyou, () => get_target_kyou())
    watch(() => props.rekyou, () => get_target_kyou())

    // ── Business logic ──
    async function get_target_kyou() {
        // target_idが空だと下の使い回しガード(初期値'')に引っかかってリクエストすら飛ばず、
        // 中身の入らないKyouViewが読み込み中表示のまま止まる。見つからなかった扱いにして終端させる
        if (props.rekyou.target_id === '') {
            is_target_not_found.value = true
            return
        }
        // 仮想スクロールの行使い回しでpropsだけ差し替わることがある。参照先が同じなら引き直さない
        if (loaded_target_id === props.rekyou.target_id) {
            return
        }
        loaded_target_id = props.rekyou.target_id
        is_target_not_found.value = false

        const requested_target_id = props.rekyou.target_id
        const req = new GetKyouRequest()
        req.id = requested_target_id
        const res = await props.gkill_api.get_kyou(req)
        // 応答が返るまでに行が別の参照先へ使い回されていたら捨てる。
        // loaded_target_idは連続する同一idしか抑制しないので、A→B→Aで応答が入れ替わりうる
        if (loaded_target_id !== requested_target_id) {
            return
        }
        if (res.errors && res.errors.length !== 0) {
            is_target_not_found.value = true
            // 一覧では行数ぶんスナックバーが出てしまうので、行では黙って諦める
            if (!is_row.value) {
                emits('received_errors', res.errors)
            }
            return
        }
        // 参照先が消えているとundefinedが入ってしまうので、空なら見つからなかった扱いにする
        if (!res.kyou_histories || res.kyou_histories.length < 1) {
            is_target_not_found.value = true
            if (!is_row.value) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.not_found_rekyou_target
                error.error_message = i18n.global.t('NOT_FOUND_REKYOU_TARGET_ERROR_MESSAGE')
                emits('received_errors', [error])
            }
            return
        }
        is_target_not_found.value = false
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
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Return ──
    return {
        // Template refs
        context_menu,

        // State
        target_kyou,
        is_target_not_found,

        // Computed
        is_row,

        // Business logic
        show_context_menu,
        get_target_kyou,

        // Event relay objects
        crudRelayHandlers,
    }
}

