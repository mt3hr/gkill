import { ref } from 'vue'
import type LantanaContextMenu from '@/pages/views/lantana-context-menu.vue'
import type { LantanaViewProps } from '@/pages/views/lantana-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useLantanaView(options: {
    props: LantanaViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits: _emits } = options

    const context_menu = ref<InstanceType<typeof LantanaContextMenu> | null>(null)

    function show_context_menu(e: PointerEvent): void {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    return {
        context_menu,
        show_context_menu,
    }
}
