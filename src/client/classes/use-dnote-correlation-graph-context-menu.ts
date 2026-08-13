import { ref, type Ref } from 'vue'
import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import type { DnoteCorrelationGraphContextMenuProps } from '@/pages/views/dnote-correlation-graph-context-menu-props'
import type { DnoteCorrelationGraphContextMenuEmits } from '@/pages/views/dnote-correlation-graph-context-menu-emits'

export function useDnoteCorrelationGraphContextMenu(options: {
    props: DnoteCorrelationGraphContextMenuProps,
    emits: DnoteCorrelationGraphContextMenuEmits,
}) {
    const { emits } = options

    const id: Ref<string> = ref("")
    const { is_show, menu_target, open_at } = useContextMenuPosition()

    async function show(e: MouseEvent, dnote_correlation_graph_query_id: string): Promise<void> {
        id.value = dnote_correlation_graph_query_id
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
