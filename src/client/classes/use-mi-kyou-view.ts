import { i18n } from '@/i18n'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import { nextTick, type Ref, ref, watch } from 'vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { miKyouViewProps } from '@/pages/views/mi-kyou-view-props'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'

export function useMiKyouView(options: {
    props: miKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const context_menu = ref<ComponentRef | null>(null)

    // ── State refs ──
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const is_checked_mi: Ref<boolean> = ref(props.kyou.typed_mi ? props.kyou.typed_mi.is_checked : false)
    const mi_title_style = ref({
        maxWidth: 'calc(100% - 0px)'
    })

    // ── Init ──
    nextTick(() => update_mi_title_style())

    // ── Watchers ──
    watch(() => props.kyou, async () => {
        await load_cloned_kyou()
        nextTick(() => update_mi_title_style())
        is_checked_mi.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.is_checked : false
    })

    // ── Business logic ──
    function update_mi_title_style(): void {
        mi_title_style.value.maxWidth = `calc(100% - ${get_board_name_text_width_px()}px)`
    }

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

    async function clicked_mi_check(): Promise<void> {
        // 読み取り専用表示だったら何もしない
        if (props.is_readonly_mi_check) {
            return
        }

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

    function get_board_name_text_width_px(): number {
        const mi_board_name_element = document.querySelector(".kyou_".concat(props.kyou.id).concat(" ").concat(".mi_board_name"))
        if (mi_board_name_element == null) {
            return 0
        }
        const text_width = get_text_width(props.kyou.typed_mi?.board_name ?? '', get_canvas_font(mi_board_name_element as HTMLElement)).valueOf() + 16 + 16 + 4 // padding + padding + 4px
        return text_width
    }

    function get_text_width(text: string, font: string): number {
        const fn = get_text_width as unknown as Record<string, HTMLCanvasElement>
        const canvas: HTMLCanvasElement = fn.canvas || (fn.canvas = document.createElement("canvas"))
        const context = canvas.getContext("2d")!
        context.font = font
        const metrics = context.measureText(text)
        return metrics.width
    }

    function get_css_style(element: Element, prop: string): string {
        return window.getComputedStyle(element, null).getPropertyValue(prop)
    }

    function get_canvas_font(element = document.body): string {
        const fontWeight = get_css_style(element, 'font-weight') || 'normal'
        const fontSize = get_css_style(element, 'font-size') || '16px'
        const fontFamily = get_css_style(element, 'font-family') || 'Times New Roman'
        return `${fontWeight} ${fontSize} ${fontFamily}`
    }

    function on_drag_start(e: DragEvent) {
        e.dataTransfer!.setData("gkill_mi", JSON.stringify(props.kyou.typed_mi))
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
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    // ── Return ──
    return {
        // Template refs
        context_menu,

        // State
        cloned_kyou,
        is_checked_mi,
        mi_title_style,

        // Business logic
        show_context_menu,
        clicked_mi_check,
        on_drag_start,

        // Event relay objects
        crudRelayHandlers,
    }
}
