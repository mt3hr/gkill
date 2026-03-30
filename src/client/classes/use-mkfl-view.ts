import { ref } from 'vue'
import type PlaingTimeisView from '@/pages/views/plaing-timeis-view.vue'
import type { MKFLViewEmits } from '@/pages/views/mkfl-view-emits'

export function useMkflView(options: {
    emits: MKFLViewEmits,
    plaing_timeis_view: ReturnType<typeof ref<InstanceType<typeof PlaingTimeisView> | null>>,
}) {
    const { emits: _emits, plaing_timeis_view } = options

    // ── Methods ──
    async function reload_plaing_timeis_view(): Promise<void> {
        plaing_timeis_view.value?.reload_list(false)
    }

    // ── Return ──
    return {
        // Methods
        reload_plaing_timeis_view,
    }
}
