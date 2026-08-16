import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { deep_equals } from '@/classes/deep-equals'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import moment from 'moment'
import { CheckState } from '@/pages/views/check-state'
import type { MiQueryEditorSidebarEmits } from '@/pages/views/mi-query-editor-sidebar-emits'
import type { MiQueryEditorSidebarProps } from '@/pages/views/mi-query-editor-sidebar-props'
import type SidebarHeader from '@/pages/views/sidebar-header.vue'
import type KeywordQuery from '@/pages/views/keyword-query.vue'
import type TagQuery from '@/pages/views/tag-query.vue'
import type CalendarQuery from '@/pages/views/calendar-query.vue'
import type miExtractCheckStateQuery from '@/pages/views/mi-extract-check-state-query.vue'
import type miSortTypeQuery from '@/pages/views/mi-sort-type-query.vue'
import type MiBoardQuery from '@/pages/views/mi-board-query.vue'
import { SavedFindQueryConfig, type SavedFindQueryItem } from '@/classes/datas/config/saved-find-query-config'
import { MI_ALL_BOARD_KEY } from '@/classes/mi-board-names'

export function useMiQueryEditorSidebar(options: {
    props: MiQueryEditorSidebarProps,
    emits: MiQueryEditorSidebarEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const sidebar_header = ref<InstanceType<typeof SidebarHeader> | null>(null)
    const keyword_query = ref<InstanceType<typeof KeywordQuery> | null>(null)
    const tag_query = ref<InstanceType<typeof TagQuery> | null>(null)
    const calendar_query = ref<InstanceType<typeof CalendarQuery> | null>(null)
    const check_state_query = ref<InstanceType<typeof miExtractCheckStateQuery> | null>(null)
    const sort_type_query = ref<InstanceType<typeof miSortTypeQuery> | null>(null)
    const board_query = ref<InstanceType<typeof MiBoardQuery> | null>(null)

    // ── State refs ──
    const default_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const is_mounted = ref(false)
    nextTick(() => is_mounted.value = true)

    const inited_sidebar_header_for_query_sidebar = ref(true)
    const inited_keyword_query_for_query_sidebar = ref(true)
    const inited_tag_query_for_query_sidebar = ref(false)
    const inited_calendar_query_for_query_sidebar = ref(false)
    const inited_check_state_query_for_query_sidebar = ref(false)
    const inited_sort_query_for_query_sidebar = ref(false)
    const inited_board_query_for_query_sidebar = ref(false)

    // ── Computed ──
    const header_margin = computed(() => props.application_config.is_show_share_footer ? 12 : 6)
    const header_height: Ref<number> = computed(() => 36 * (props.application_config.is_show_share_footer ? 2 : 1) + header_margin.value.valueOf())
    const sidebar_height = computed(() => (props.app_content_height.valueOf() - header_height.value).toString().concat("px"))
    const header_top_px = computed(() => (props.app_content_height.valueOf() - header_height.value).toString().concat("px"))
    const sidebar_top_px = computed(() => (header_height.value * -1).toString().concat("px"))
    // 保存済み検索条件(タスク用)。0件なら呼び出しFABごと非表示にする
    const saved_find_querys = computed(() => SavedFindQueryConfig.parse(props.application_config.saved_find_query_json_data).saved_mi_find_kyou_querys)

    const inited = computed(() => {
        if (!is_mounted.value) {
            return false
        }

        // ここに載せてよいのは実際に @inited を発火する子だけ。
        // 画面から消した子のフラグを残すと永久にfalseのままで inited が立たず、
        // mi画面が init() を走らせられずスピナーのまま固まる
        return inited_keyword_query_for_query_sidebar.value &&
            inited_tag_query_for_query_sidebar.value &&
            inited_calendar_query_for_query_sidebar.value &&
            inited_check_state_query_for_query_sidebar.value &&
            inited_sort_query_for_query_sidebar.value &&
            inited_board_query_for_query_sidebar.value
    })

    // ── Watchers ──
    watch(() => inited.value, async (new_value: boolean, old_value: boolean) => {
        if (old_value !== new_value && new_value) {
            nextTick(() => { emits('inited') })
        }
    })

    // immediate必須。マウント時点で設定が既に渡っているとwatcherは発火せず、
    // default_queryが空のFindKyouQueryのまま列追加・クリアに使われてしまう
    watch(() => props.application_config, () => {
        default_query.value = FindKyouQuery.generate_default_query_for_mi(props.application_config)
        default_query.value.query_id = props.gkill_api.generate_uuid()
        // mi_board_name はコンストラクタ既定の null（=「すべて」）のまま。
        // 番兵(MI_ALL_BOARD_KEY)はサイドバーだけが使い、クエリへは持ち込まない
    }, { immediate: true })

    watch(() => props.find_kyou_query, (new_value: FindKyouQuery, old_value: FindKyouQuery) => {
        if (deep_equals(new_value, old_value)) {
            return
        }
        query.value = new_value
    })

    // ── Business logic ──
    function get_default_query(): FindKyouQuery {
        const q = default_query.value.clone()
        q.query_id = props.gkill_api.generate_uuid()
        if (props.application_config.mi_default_period !== -1) {
            q.calendar_start_date = moment(moment().add(-props.application_config.mi_default_period, "days").format("YYYY-MM-DD 00:00:00 ZZ")).toDate()
            q.calendar_end_date = moment(moment().format("YYYY-MM-DD 00:00:00 ZZ")).add(1, "days").add(-1, "milliseconds").toDate()
        }
        return q
    }

    function generate_query(query_id?: string): FindKyouQuery {
        const find_query = new FindKyouQuery()
        if (query_id) {
            find_query.query_id = query_id
        }
        find_query.for_mi = true

        find_query.is_focus_kyou_in_list_view = props.find_kyou_query ? props.find_kyou_query.is_focus_kyou_in_list_view : false
        find_query.is_image_only = query.value.is_image_only

        if (keyword_query.value) {
            // 有効時は未パースプレースホルダの[]（パースは送信直前のcloneで行う）、無効時はnull
            const use_words = keyword_query.value.get_use_words()
            find_query.words = use_words ? [] : null
            find_query.not_words = use_words ? [] : null
            find_query.words_and = keyword_query.value.get_use_word_and_search()
            find_query.keywords = keyword_query.value.get_keywords().concat()
        }

        if (board_query.value) {
            // 「すべて」の番兵→null変換はここ1点のみ（番兵はサイドバー専用。クエリ上のnull=「すべて」）。
            // 番兵はロケール非依存のMI_ALL_BOARD_KEY。ツリーが emit するのはノードのkeyで、
            // それは append_all_mi_board() が入れた "すべて" 固定なので、i18nの訳語と比べてはいけない
            const board_name = board_query.value.get_board_name()
            find_query.mi_board_name = board_name === MI_ALL_BOARD_KEY ? null : board_name
        }

        find_query.reps = get_default_query().reps
        find_query.rep_types = ["mi"]

        if (check_state_query.value) {
            find_query.mi_check_state = check_state_query.value.get_update_extract_check_state()
        }

        if (sort_type_query.value) {
            find_query.mi_sort_type = sort_type_query.value.get_sort_type()
            find_query.include_create_mi = true
            switch (find_query.mi_sort_type) {
                case MiSortType.create_time:
                    break
                case MiSortType.estimate_end_time:
                    find_query.include_end_mi = true
                    break
                case MiSortType.estimate_start_time:
                    find_query.include_start_mi = true
                    break
                case MiSortType.limit_time:
                    find_query.include_limit_mi = true
                    break
            }
        }

        if (tag_query.value) {
            const tags = tag_query.value.get_tags()?.concat()
            if (tags) {
                find_query.tags = tags
            }
            find_query.tags_and = tag_query.value.get_is_and_search()
        }

        if (calendar_query.value) {
            const use_calendar = calendar_query.value.get_use_calendar()
            find_query.calendar_start_date = use_calendar ? calendar_query.value.get_start_date() : null
            find_query.calendar_end_date = use_calendar ? calendar_query.value.get_end_date() : null
        }

        // 状況(TimeIs)・時間帯・場所はこのサイドバーから外したので、
        // それらのフィールドは FindKyouQuery の既定値（すべてOFF）のままにする

        find_query.apply_hide_tags(props.application_config)

        return find_query
    }

    // ── Template event handlers ──
    function emits_current_query(): void {
        const current_query = generate_query(query.value.query_id)
        // フォーカス切替等でfind_kyou_queryを差し替えると、子ビューのprops同期の残響が
        // このハンドラへ機械的に届く(1tickより遅れて来るのでタイミングでは抑止できない)。
        // 機械的な発火は再生成結果が同期済みクエリと同値になるので、値比較で捨てる。
        // これを外すと「検索中の列をクリック→飛行中の検索がabortされ最初からやり直し」が再発する
        if (deep_equals(current_query, query.value)) {
            return
        }
        emits('updated_query', current_query)
    }

    function emits_cleard_sort_type_query(): void {
        const find_query = generate_query()
        find_query.query_id = query.value.query_id
        find_query.mi_sort_type = get_default_query().mi_sort_type
        query.value = find_query
        emits('updated_query', find_query)
    }

    function emits_cleard_check_state(): void {
        const find_query = generate_query()
        find_query.query_id = query.value.query_id
        find_query.mi_check_state = get_default_query().mi_check_state
        query.value = find_query
        emits('updated_query', find_query)
    }

    function emits_cleard_keyword_query(): void {
        const find_query = generate_query()
        find_query.query_id = query.value.query_id
        const d = get_default_query()
        find_query.words = d.words === null ? null : d.words.concat()
        find_query.not_words = d.not_words === null ? null : d.not_words.concat()
        find_query.keywords = d.keywords.concat()
        find_query.words_and = d.words_and
        query.value = find_query
        emits('updated_query', find_query)
    }

    function emits_cleard_tag_query(): void {
        const find_query = generate_query()
        find_query.query_id = query.value.query_id
        const d = get_default_query()
        find_query.tags = d.tags === null ? null : d.tags.concat()
        find_query.tags_and = d.tags_and
        query.value = find_query
        tag_query.value?.update_check(find_query.tags ?? [], CheckState.checked, true, true)
        emits('updated_query', find_query)
    }

    function emits_cleard_calendar_query(): void {
        const find_query = generate_query()
        find_query.query_id = query.value.query_id
        const d = get_default_query()
        find_query.calendar_start_date = d.calendar_start_date
        find_query.calendar_end_date = d.calendar_end_date
        query.value = find_query
        emits('updated_query', find_query)
    }

    async function emits_default_query(): Promise<void> {
        const board_name = props.find_kyou_query.mi_board_name
        const find_query = get_default_query().clone()
        find_query.query_id = query.value.query_id
        find_query.mi_board_name = board_name
        query.value = find_query
        tag_query.value?.update_check(find_query.tags ?? [], CheckState.checked, true, true)
        emits('updated_query', find_query)
    }

    // 保存済み検索条件をサイドバーへ適用する（手で編集したのと同じ扱い。
    // ホットリロードONなら親が自動検索し、OFFなら検索ボタンで実行される）。
    // query_id は列の恒久IDなので、保存条件のものは使わず必ず現在の列のIDを維持する。
    // 板名はクリアと違って保持しない（保存された条件の板名も条件の一部として勝つ）
    async function apply_saved_query(item: SavedFindQueryItem): Promise<void> {
        const find_query = item.find_kyou_query.clone()
        find_query.query_id = query.value.query_id
        query.value = find_query
        tag_query.value?.update_check(find_query.tags ?? [], CheckState.checked, true, true)
        emits('updated_query', find_query)
    }

    async function show_manage_share_kyou_dialog(): Promise<void> {
        throw new Error('Not implemented')
    }

    async function show_share_kyou_dialog(): Promise<void> {
        throw new Error('Not implemented')
    }

    function onRequestSearchFalse(): void {
        emits('requested_search', false)
    }

    function onRequestSearchTrue(): void {
        emits('requested_search', true)
    }

    function onRequestOpenFocusBoard(board_name: string): void {
        emits('request_open_focus_board', board_name)
    }

    function onReceivedMessages(messages: Array<GkillMessage>): void {
        emits('received_messages', messages)
    }

    function onReceivedErrors(errors: Array<GkillError>): void {
        emits('received_errors', errors)
    }

    function onTagQueryRequestUpdateCheckedTags(_tags: string[], is_by_user: boolean): void {
        if (is_by_user) emits_current_query()
    }

    function onInitedTag(): void {
        inited_tag_query_for_query_sidebar.value = true
    }

    function onInitedCalendar(): void {
        inited_calendar_query_for_query_sidebar.value = true
    }

    function onInitedCheckState(): void {
        inited_check_state_query_for_query_sidebar.value = true
    }

    function onInitedSort(): void {
        inited_sort_query_for_query_sidebar.value = true
    }

    function onInitedBoard(): void {
        inited_board_query_for_query_sidebar.value = true
    }

    function onInitedKeyword(): void {
        inited_keyword_query_for_query_sidebar.value = true
    }

    // ── Return ──
    return {
        // Template refs
        sidebar_header,
        keyword_query,
        tag_query,
        calendar_query,
        check_state_query,
        sort_type_query,
        board_query,

        // State
        default_query,
        query,
        is_mounted,
        inited_sidebar_header_for_query_sidebar,
        inited_keyword_query_for_query_sidebar,
        inited_tag_query_for_query_sidebar,
        inited_calendar_query_for_query_sidebar,
        inited_check_state_query_for_query_sidebar,
        inited_sort_query_for_query_sidebar,
        inited_board_query_for_query_sidebar,

        // Computed
        header_margin,
        header_height,
        sidebar_height,
        header_top_px,
        sidebar_top_px,
        saved_find_querys,
        inited,

        // Exposed methods (for defineExpose)
        generate_query,
        get_default_query,
        apply_saved_query,

        // Template event handlers
        emits_current_query,
        emits_cleard_sort_type_query,
        emits_cleard_check_state,
        emits_cleard_keyword_query,
        emits_cleard_tag_query,
        emits_cleard_calendar_query,
        emits_default_query,
        show_manage_share_kyou_dialog,
        show_share_kyou_dialog,
        onRequestSearchFalse,
        onRequestSearchTrue,
        onRequestOpenFocusBoard,
        onReceivedMessages,
        onReceivedErrors,
        onTagQueryRequestUpdateCheckedTags,
        onInitedTag,
        onInitedCalendar,
        onInitedCheckState,
        onInitedSort,
        onInitedBoard,
        onInitedKeyword,
    }
}
