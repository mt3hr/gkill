import { ref, type Ref } from 'vue'
import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import type { DeviceStructContextMenuProps } from '@/pages/views/device-struct-context-menu-props'
import type { DeviceStructContextMenuEmits } from '@/pages/views/device-struct-context-menu-emits'

export function useDeviceStructContextMenu(options: {
    props: DeviceStructContextMenuProps,
    emits: DeviceStructContextMenuEmits,
}) {
    const { emits } = options

    const id: Ref<string> = ref("")
    const { is_show, menu_target, open_at } = useContextMenuPosition()

    async function show(e: MouseEvent, device_id: string): Promise<void> {
        id.value = device_id
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
