import { ref, type Ref } from 'vue'
import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import type { RepStructContextMenuProps } from '@/pages/views/rep-struct-context-menu-emits'
import type { RepStructContextMenuEmits } from '@/pages/views/rep-struct-context-menu-props'

export function useRepStructContextMenu(options: {
    props: RepStructContextMenuProps,
    emits: RepStructContextMenuEmits,
}) {
    const { emits } = options

    const id: Ref<string> = ref("")
    const { is_show, menu_target, open_at } = useContextMenuPosition()

    async function show(e: MouseEvent, rep_id: string): Promise<void> {
        id.value = rep_id
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
