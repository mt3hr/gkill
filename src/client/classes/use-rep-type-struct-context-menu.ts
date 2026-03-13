import { computed, ref, type Ref } from 'vue'
import type { RepTypeStructContextMenuProps } from '@/pages/views/rep-type-struct-context-menu-props'
import type { RepTypeStructContextMenuEmits } from '@/pages/views/rep-type-struct-context-menu-emits'

export function useRepTypeStructContextMenu(options: {
    props: RepTypeStructContextMenuProps,
    emits: RepTypeStructContextMenuEmits,
}) {
    const { emits } = options

    const id: Ref<string> = ref("")
    const is_show: Ref<boolean> = ref(false)
    const position_x: Ref<Number> = ref(0)
    const position_y: Ref<Number> = ref(0)
    const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(Math.max(50, document.defaultView!.innerHeight - ( + 8 + (48 * 2))), position_y.value.valueOf())}px; }`)

    async function show(e: MouseEvent, rep_type_id: string): Promise<void> {
        id.value = rep_type_id
        position_x.value = e.clientX
        position_y.value = e.clientY
        is_show.value = true
    }

    async function hide(): Promise<void> {
        is_show.value = false
    }

    return {
        id,
        is_show,
        position_x,
        position_y,
        context_menu_style,
        show,
        hide,
        emits,
    }
}
