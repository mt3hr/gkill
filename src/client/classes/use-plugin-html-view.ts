import { ref } from 'vue'
import type { PluginHtmlViewProps } from '@/pages/views/plugin-html-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { ComponentRef } from '@/classes/component-ref'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function usePluginHtmlView(options: {
    props: PluginHtmlViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const context_menu = ref<ComponentRef | null>(null)

    // ── Business logic ──
    async function show_context_menu(e: PointerEvent): Promise<void> {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    // ── Event relay objects ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Return ──
    return {
        // Template refs
        context_menu,

        // Business logic
        show_context_menu,

        // Event relay objects
        crudRelayHandlers,
    }
}

