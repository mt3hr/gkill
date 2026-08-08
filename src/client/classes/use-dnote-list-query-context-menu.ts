import { ref, type Ref } from 'vue'
import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import type { DnoteListQueryContextMenuProps } from '@/pages/views/dnote-list-query-context-menu-props'
import type { DnoteListQueryContextMenuEmits } from '@/pages/views/dnote-list-query-context-menu-emits'

export function useDnoteListQueryContextMenu(options: {
    props: DnoteListQueryContextMenuProps,
    emits: DnoteListQueryContextMenuEmits,
}) {
    const { emits } = options

    const id: Ref<string> = ref("")
    const { is_show, menu_target, open_at } = useContextMenuPosition()

    async function show(e: MouseEvent, dnote_list_query_id: string): Promise<void> {
        id.value = dnote_list_query_id
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
