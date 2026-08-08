import { ref } from 'vue'
import type { KyouHistoriesViewProps } from '@/pages/views/kyou-histories-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { Kyou } from '@/classes/datas/kyou'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useKyouHistoriesView(options: {
    props: KyouHistoriesViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const cloned_kyou = ref(new Kyou())

    // ── Init ──
    load_cloned_kyou()

    // ── Business logic ──
    async function load_cloned_kyou() {
        const cloned_kyou_value = props.kyou.clone()
        await cloned_kyou_value.load_attached_histories()
        for (let i = 0; i < cloned_kyou.value.attached_histories.length; i++) {
            cloned_kyou_value.attached_histories[i].related_time = cloned_kyou_value.attached_histories[i].update_time
        }
        cloned_kyou.value = cloned_kyou_value
    }

    // ── Event relay objects ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Return ──
    return {
        // State
        cloned_kyou,

        // Event relay objects
        crudRelayHandlers,
    }
}

