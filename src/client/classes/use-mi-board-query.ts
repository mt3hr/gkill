import { i18n } from '@/i18n'
import { computed, nextTick, type Ref, ref } from 'vue'
import type { miBoardQueryEmits } from '@/pages/views/mi-board-query-emits'
import type { miBoardQueryProps } from '@/pages/views/mi-board-query-props'

export function useMiBoardQuery(options: {
    props: miBoardQueryProps,
    emits: miBoardQueryEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const mi_board_struct = computed(() => props.application_config.mi_board_struct)
    const board_name: Ref<string> = ref(i18n.global.t("MI_ALL_TITLE"))
    const use_board = ref(true)

    // ── Lifecycle ──
    nextTick(() => emits('inited'))

    // ── Methods ──
    function get_board_name(): string {
        return board_name.value
    }

    // ── Return ──
    return {
        // State
        mi_board_struct,
        board_name,
        use_board,

        // Methods
        get_board_name,
    }
}
