import type { LantanaFlowerProps } from '@/pages/views/lantana-flower-props'
import type { LantanaFlowerEmits } from '@/pages/views/lantana-flower-emits'

export function useLantanaFlower(options: {
    props: LantanaFlowerProps,
    emits: LantanaFlowerEmits,
}) {
    const { props, emits } = options

    // ── Business logic ──
    function emit_clicked_left() {
        if (props.editable) {
            emits('clicked_left')
        }
    }

    function emit_clicked_right() {
        if (props.editable) {
            emits('clicked_right')
        }
    }

    // ── Return ──
    return {
        // Business logic
        emit_clicked_left,
        emit_clicked_right,
    }
}
