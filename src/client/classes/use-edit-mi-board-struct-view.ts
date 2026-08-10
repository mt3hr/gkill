import { nextTick, type Ref, ref, watch } from 'vue'
import type { EditMiBoardStructViewEmits } from '@/pages/views/edit-mi-board-struct-view-emits'
import type { EditMiBoardStructViewProps } from '@/pages/views/edit-mi-board-struct-view-props'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'
import { move_struct_up, move_struct_down } from '@/classes/foldable-struct-move'

/**
 * Mi の板構造の編集。
 *
 * 板はフォルダ分けも表示名の変更もしない ―― フラットな一覧を並べ替えて、
 * 使わなくなった板を消すだけ。板名は実データ(Mi/MiReKyou のレコード)由来なので
 * ここでは触らない。
 */
export function useEditMiBoardStructView(options: {
    props: EditMiBoardStructViewProps,
    emits: EditMiBoardStructViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const foldable_struct = ref<ComponentRef | null>(null)
    const mi_board_struct_context_menu = ref<ComponentRef | null>(null)
    const confirm_delete_mi_board_struct_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

    // ── Watchers ──
    watch(() => props.application_config, () => reload_cloned_application_config())

    // ── Business logic ──
    async function reload_cloned_application_config(): Promise<void> {
        cloned_application_config.value = props.application_config.clone()
        // 実在する板でノードが無いものを補う。
        // append_all_mi_board は呼ばない ―― load_all で実行済みだし、
        // ダイアログを開くたびに「すべて」を unshift すると並び順が戻ってしまう
        await cloned_application_config.value.append_not_found_mi_boards()
    }

    function show_mi_board_contextmenu(e: MouseEvent, id: string | null): void {
        if (id) {
            mi_board_struct_context_menu.value?.show(e, id)
        }
    }

    function find_mi_board_struct(id: string): MiBoardStructElementData | null {
        let found: MiBoardStructElementData | null = null
        const walk = (board: MiBoardStructElementData): void => {
            if (board.id === id) {
                found = board
                return
            }
            board.children?.forEach(child => {
                if (child) {
                    walk(child)
                }
            })
        }
        walk(cloned_application_config.value.mi_board_struct)
        return found
    }

    async function apply(): Promise<void> {
        emits('requested_apply_mi_board_struct', cloned_application_config.value.mi_board_struct)
        nextTick(() => emits('requested_close_dialog'))
    }

    function show_confirm_delete_mi_board_struct_dialog(id: string): void {
        const target_struct_object = find_mi_board_struct(id)
        if (!target_struct_object) {
            return
        }
        confirm_delete_mi_board_struct_dialog.value?.show(target_struct_object)
    }

    function delete_mi_board_struct(id: string): void {
        let walk = (_board: MiBoardStructElementData): boolean => false
        walk = (board: MiBoardStructElementData): boolean => {
            const children = board.children
            if (board.id === id) {
                return true
            } else if (children) {
                for (let i = 0; i < children.length; i++) {
                    if (walk(children[i])) {
                        children.splice(i, 1)
                        return false
                    }
                }
            }
            return false
        }
        walk(cloned_application_config.value.mi_board_struct)
    }

    function move_mi_board_struct_up(id: string): void {
        move_struct_up(cloned_application_config.value.mi_board_struct, id)
    }

    function move_mi_board_struct_down(id: string): void {
        move_struct_down(cloned_application_config.value.mi_board_struct, id)
    }

    // ── Template event handlers ──
    function onRequestedCloseDialog(): void {
        emits('requested_close_dialog')
    }

    // ── Event relay objects ──
    const errorMessageRelayHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    // ── Return ──
    return {
        // Template refs
        foldable_struct,
        mi_board_struct_context_menu,
        confirm_delete_mi_board_struct_dialog,

        // State
        cloned_application_config,

        // Business logic
        reload_cloned_application_config,
        show_mi_board_contextmenu,
        apply,
        show_confirm_delete_mi_board_struct_dialog,
        delete_mi_board_struct,
        move_mi_board_struct_up,
        move_mi_board_struct_down,

        // Template event handlers
        onRequestedCloseDialog,

        // Event relay objects
        errorMessageRelayHandlers,
    }
}
