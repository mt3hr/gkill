import { computed, ref } from 'vue'
import type { AttachedTimeIsPlayingProps } from '@/pages/views/attached-time-is-playing-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useAttachedTimeIsPlaying(options: {
    props: AttachedTimeIsPlayingProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const context_menu = ref()

    // ── Computed ──
    const playing_class = computed(() => {
        if (!props.kyou) {
            return ""
        }
        let highlighted = false;
        for (let i = 0; i < props.highlight_targets.length; i++) {
            if (props.highlight_targets[i].id === props.kyou.id
                && props.highlight_targets[i].create_time.getTime() === props.timeis_kyou.create_time.getTime()
                && props.highlight_targets[i].update_time.getTime() === props.timeis_kyou.update_time.getTime()) {
                highlighted = true
                break
            }
        }
        if (highlighted) {
            return "highlighted_playing"
        }
        return "playing"
    })

    // ── Methods ──
    async function show_context_menu(e: PointerEvent): Promise<void> {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    function show_kyou_dialog(): void {
        if (props.enable_dialog) {
            emits('requested_open_rykv_dialog', 'kyou', props.timeis_kyou)
        }
    }

    // ── Return ──
    // ── Event relay objects ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    return {
        // Event relay objects
        crudRelayHandlers,
        // Template refs
        context_menu,

        // State
        playing_class,

        // Methods used in template
        show_context_menu,
        show_kyou_dialog,
    }
}
