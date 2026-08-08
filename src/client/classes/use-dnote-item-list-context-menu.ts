import { ref, type Ref } from 'vue'
import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import type { DnoteItemListContextMenuProps } from '@/pages/views/dnote-item-list-context-menu-props'
import type { DnoteItemListContextMenuEmits } from '@/pages/views/dnote-item-list-context-menu-emits'

export function useDnoteItemListContextMenu(options: {
    props: DnoteItemListContextMenuProps,
    emits: DnoteItemListContextMenuEmits,
}) {
    const { emits } = options

    const id: Ref<string> = ref("")
    const { is_show, menu_target, open_at } = useContextMenuPosition()

    async function show(e: MouseEvent, dnote_item_id: string): Promise<void> {
        id.value = dnote_item_id
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
