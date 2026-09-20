import { type Ref, ref } from 'vue'
import type { EditMiBoardStructElementViewEmits } from '@/pages/views/edit-mi-board-struct-element-view-emits'
import type { EditMiBoardStructElementViewProps } from '@/pages/views/edit-mi-board-struct-element-view-props'
import { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'

/**
 * 板構造の要素編集。
 *
 * 板名は実データ(Mi/MiReKyou のレコード)由来なので変えられない。
 * ここで編集できるのは説明（MCP へ渡す運用メモ）だけ。
 */
export function useEditMiBoardStructElementView(options: {
    props: EditMiBoardStructElementViewProps,
    emits: EditMiBoardStructElementViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    // 説明（運用メモ）。古い保存データには欄が無いので undefined を空文字に倒す
    const description: Ref<string> = ref(props.struct_obj.description ?? "")

    // ── Methods ──
    async function apply(): Promise<void> {
        const mi_board_struct = new MiBoardStructElementData()
        mi_board_struct.id = props.struct_obj.id
        mi_board_struct.check_when_inited = props.struct_obj.check_when_inited
        mi_board_struct.description = description.value
        mi_board_struct.children = props.struct_obj.children
        mi_board_struct.indeterminate = false
        mi_board_struct.is_dir = props.struct_obj.is_dir
        mi_board_struct.key = props.struct_obj.board_name
        mi_board_struct.board_name = props.struct_obj.board_name
        mi_board_struct.name = props.struct_obj.board_name

        emits('requested_update_mi_board_struct', mi_board_struct)
        emits('requested_close_dialog')
    }

    // ── Return ──
    return {
        // State
        description,

        // Methods
        apply,
    }
}
