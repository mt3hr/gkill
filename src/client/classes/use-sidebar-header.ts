import type { SidebarHeaderProps } from '@/pages/views/sidebar-header-props'
import type { SidebarHeaderEmits } from '@/pages/views/sidebar-header-emits'

export function useSidebarHeader(options: {
    props: SidebarHeaderProps,
    emits: SidebarHeaderEmits,
}) {
    const { props: _props, emits: _emits } = options

    return {
    }
}
