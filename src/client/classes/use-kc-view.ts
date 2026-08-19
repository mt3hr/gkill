import { ref } from 'vue'
import type KCContextMenu from '@/pages/views/kc-context-menu.vue'
import type { KCViewProps } from '@/pages/views/kc-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

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

    // ── Event relay objects ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    return {
        // Event relay objects
        crudRelayHandlers,
        context_menu,
        show_context_menu,
    }
}
