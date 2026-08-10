import { ref, type Ref } from 'vue'
import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import type { MiBoardStructContextMenuProps } from '@/pages/views/mi-board-struct-context-menu-props'
import type { MiBoardStructContextMenuEmits } from '@/pages/views/mi-board-struct-context-menu-emits'

export function useMiBoardStructContextMenu(options: {
    props: MiBoardStructContextMenuProps,
    emits: MiBoardStructContextMenuEmits,
}) {
    const { emits } = options

    const id: Ref<string> = ref("")
    const { is_show, menu_target, open_at } = useContextMenuPosition()

    async function show(e: MouseEvent, mi_board_id: string): Promise<void> {
        id.value = mi_board_id
        open_at(e)
    }

    async function hide(): Promise<void> {
        is_show.value = false
    }

    return {
        id,
        is_show,
        menu_target,
        show,
        hide,
        emits,
    }
}
