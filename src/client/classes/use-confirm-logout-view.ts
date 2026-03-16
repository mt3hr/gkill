import type { ConfirmLogoutViewEmits } from '@/pages/views/confirm-logout-view-emits'
import type { ConfirmLogoutViewProps } from '@/pages/views/confirm-logout-view-props'

export function useConfirmLogoutView(options: {
    props: ConfirmLogoutViewProps,
    emits: ConfirmLogoutViewEmits,
}) {
    const { props, emits } = options

    // ── Methods ──
    async function confirm_logout(): Promise<void> {
        emits('requested_logout', props.close_database)
    }

    // ── Return ──
    return {
        // Methods
        confirm_logout,
    }
}
