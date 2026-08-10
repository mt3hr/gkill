import { i18n } from '@/i18n'
import { computed, nextTick, type Ref, ref, watch } from 'vue'
import type { MiBoardQueryEmits } from '@/pages/views/mi-board-query-emits'
import type { MiBoardQueryProps } from '@/pages/views/mi-board-query-props'

export function useMiBoardQuery(options: {
    props: MiBoardQueryProps,
    emits: MiBoardQueryEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const mi_board_struct = computed(() => props.application_config.mi_board_struct)
    const board_name: Ref<string> = ref(i18n.global.t("MI_ALL_BOARD_NAME_TITLE"))
    const use_board = ref(true)

    // ── Watchers ──
    // 板選択はフォーカス列の検索条件に追随させる。追随しないと最後にクリックした板名が
    // サイドバーに残り続け、generate_query経由で別列の検索条件に混入する
    watch(() => props.find_kyou_query, () => {
        load_board_name()
    })

    // ── Lifecycle ──
    nextTick(() => {
        load_board_name()
        emits('inited')
    })

    // ── Methods ──
    function load_board_name(): void {
        if (!props.find_kyou_query) {
            return
        }
        // クエリ上 mi_board_name=null は「すべて」。
        // 番兵文字列MI_ALL_BOARD_NAME_TITLEは表示層（ここ）だけが使う
        board_name.value = props.find_kyou_query.mi_board_name
            ?? i18n.global.t("MI_ALL_BOARD_NAME_TITLE")
    }

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
