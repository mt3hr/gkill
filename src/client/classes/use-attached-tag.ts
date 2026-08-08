import { computed, ref } from 'vue'
import type { AttachedTagProps } from '@/pages/views/attached-tag-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { ComponentRef } from '@/classes/component-ref'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useAttachedTag(options: {
    props: AttachedTagProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const context_menu = ref<ComponentRef | null>(null)

    // ── Computed ──
    const tag_class = computed(() => {
        let highlighted = false;
        for (let i = 0; i < props.highlight_targets.length; i++) {
            if (props.highlight_targets[i].id === props.tag.id
                && props.highlight_targets[i].create_time.getTime() === props.tag.create_time.getTime()
                && props.highlight_targets[i].update_time.getTime() === props.tag.update_time.getTime()) {
                highlighted = true
                break
            }
        }
        if (highlighted) {
            return "highlighted_tag"
        }
        return "tag"
    })

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

        // Computed
        tag_class,

        // Business logic
        show_context_menu,

        // Event relay objects
        crudRelayHandlers,
    }
}

