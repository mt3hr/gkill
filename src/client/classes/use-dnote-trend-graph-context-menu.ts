import { ref, type Ref } from 'vue'
import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import type { DnoteTrendGraphContextMenuProps } from '@/pages/views/dnote-trend-graph-context-menu-props'
import type { DnoteTrendGraphContextMenuEmits } from '@/pages/views/dnote-trend-graph-context-menu-emits'

export function useDnoteTrendGraphContextMenu(options: {
    props: DnoteTrendGraphContextMenuProps,
    emits: DnoteTrendGraphContextMenuEmits,
}) {
    const { emits } = options

    const id: Ref<string> = ref("")
    const { is_show, menu_target, open_at } = useContextMenuPosition()

    async function show(e: MouseEvent, dnote_trend_graph_query_id: string): Promise<void> {
        id.value = dnote_trend_graph_query_id
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
