import { ref, type Ref } from 'vue'
import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import type { KFTLTemplateStructContextMenuProps } from '@/pages/views/kftl-template-struct-context-menu-props'
import type { KFTLTemplateStructContextMenuEmits } from '@/pages/views/kftl-template-struct-context-menu-emits'

export function useKFTLTemplateStructContextMenu(options: {
    props: KFTLTemplateStructContextMenuProps,
    emits: KFTLTemplateStructContextMenuEmits,
}) {
    const { emits } = options

    const id: Ref<string> = ref("")
    const { is_show, menu_target, open_at } = useContextMenuPosition()

    async function show(e: MouseEvent, kftl_template_id: string): Promise<void> {
        id.value = kftl_template_id
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
