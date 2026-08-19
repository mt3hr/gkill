'use strict'

import { computed, ref } from 'vue'
import type { AttachedTextProps } from '@/pages/views/attached-text-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import AttachedTextContextMenu from '@/pages/views/attached-text-context-menu.vue'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useAttachedText(options: { props: AttachedTextProps, emits: KyouViewEmits }) {
    const { props, emits } = options

    const context_menu = ref<InstanceType<typeof AttachedTextContextMenu> | null>(null)

    const text_class = computed(() => {
        let highlighted = false;
        for (let i = 0; i < props.highlight_targets.length; i++) {
            if (props.highlight_targets[i].id === props.text.id
                && props.highlight_targets[i].create_time.getTime() === props.text.create_time.getTime()
                && props.highlight_targets[i].update_time.getTime() === props.text.update_time.getTime()) {
                highlighted = true
                break
            }
        }
        if (highlighted) {
            return "highlighted_text"
        }
        return "text"
    })

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
        text_class,
        show_context_menu,
    }
}
