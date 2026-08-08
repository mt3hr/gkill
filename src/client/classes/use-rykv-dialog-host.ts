import type { Kyou } from '@/classes/datas/kyou'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'

interface RykvDialogHostEmits extends KyouViewEmits {
    (e: 'closed', id: string): void
}

export function useRykvDialogHost(options: {
    emits: RykvDialogHostEmits,
}) {
    const { emits } = options

    // ── Event relay objects ──
    const crudRelayHandlers = {
        ...build_kyou_dialog_relay(emits, {
            // クリックはフォーカス移動も伴う
            'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
        }),
        // closed はダイアログ固有なので共通束には含まれない
        'closed': (value: string) => emits('closed', value),
    }

    // ── Return ──
    return {
        // Event relay objects
        crudRelayHandlers,
    }
}
