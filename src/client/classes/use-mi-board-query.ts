import { computed, nextTick, type Ref, ref, watch } from 'vue'
import type { MiBoardQueryEmits } from '@/pages/views/mi-board-query-emits'
import type { MiBoardQueryProps } from '@/pages/views/mi-board-query-props'
import { CheckState } from '@/pages/views/check-state'
import { MI_ALL_BOARD_KEY, resolve_clicked_mi_board_names } from '@/classes/mi-board-names'

export function useMiBoardQuery(options: {
    props: MiBoardQueryProps,
    emits: MiBoardQueryEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const mi_board_struct = computed(() => props.application_config.mi_board_struct)
    // 番兵はロケール非依存の MI_ALL_BOARD_KEY。ツリーが emit するのは key なので、
    // i18n の訳語と比較すると日本語以外のロケールで「すべて」が効かなくなる
    const board_name: Ref<string> = ref(MI_ALL_BOARD_KEY)
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
        // 番兵MI_ALL_BOARD_KEYはサイドバー（ここ）だけが使う
        board_name.value = props.find_kyou_query.mi_board_name
            ?? MI_ALL_BOARD_KEY
    }

    function get_board_name(): string {
        return board_name.value
    }

    // ── Template event handlers ──
    // ツリーの行がクリックされたとき。開く板の決定は
    // resolve_clicked_mi_board_names() に閉じている（ルート行のクリックでは何も開かない）
    function onClickedItems(_e: MouseEvent, items: Array<string>, check_state: CheckState, is_by_user: boolean): void {
        if (!is_by_user || check_state !== CheckState.checked) {
            return
        }
        resolve_clicked_mi_board_names(items, props.application_config.mi_board_struct).forEach(board => {
            board_name.value = board
            emits('request_open_focus_board', board)
        })
    }

    // ── Return ──
    return {
        // State
        mi_board_struct,
        board_name,
        use_board,

        // Methods
        get_board_name,

        // Template event handlers
        onClickedItems,
    }
}
