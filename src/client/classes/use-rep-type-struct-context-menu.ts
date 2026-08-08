import { ref, type Ref } from 'vue'
import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import type { RepTypeStructContextMenuProps } from '@/pages/views/rep-type-struct-context-menu-props'
import type { RepTypeStructContextMenuEmits } from '@/pages/views/rep-type-struct-context-menu-emits'

export function useRepTypeStructContextMenu(options: {
    props: RepTypeStructContextMenuProps,
    emits: RepTypeStructContextMenuEmits,
}) {
    const { emits } = options

    const id: Ref<string> = ref("")
    const { is_show, menu_target, open_at } = useContextMenuPosition()

    async function show(e: MouseEvent, rep_type_id: string): Promise<void> {
        id.value = rep_type_id
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
