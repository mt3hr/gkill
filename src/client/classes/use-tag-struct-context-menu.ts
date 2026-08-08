import { ref, type Ref } from 'vue'
import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import type { TagStructContextMenuProps } from '@/pages/views/tag-struct-context-menu-props'
import type { TagStructContextMenuEmits } from '@/pages/views/tag-struct-context-menu-emits'

export function useTagStructContextMenu(options: {
    props: TagStructContextMenuProps,
    emits: TagStructContextMenuEmits,
}) {
    const { emits } = options

    const id: Ref<string> = ref("")
    const { is_show, menu_target, open_at } = useContextMenuPosition()

    async function show(e: MouseEvent, tag_id: string): Promise<void> {
        id.value = tag_id
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
