import { ref } from 'vue'
import type KCContextMenu from '@/pages/views/kc-context-menu.vue'
import type { KCViewProps } from '@/pages/views/kc-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useKCView(options: {
    props: KCViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    const context_menu = ref<InstanceType<typeof KCContextMenu> | null>(null)

    async function show_context_menu(e: PointerEvent): Promise<void> {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    return {
        context_menu,
        show_context_menu,
    }
}
