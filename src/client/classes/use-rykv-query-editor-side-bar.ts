import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { deep_equals } from '@/classes/deep-equals'
import { computed, type Ref, ref, watch } from 'vue'
import { CheckState } from '@/pages/views/check-state'
import moment from 'moment'
import type { RykvQueryEditorSidebarEmits } from '@/pages/views/rykv-query-editor-sidebar-emits'
import type { RykvQueryEditorSidebarProps } from '@/pages/views/rykv-query-editor-sidebar-props'
import type SidebarHeader from '@/pages/views/sidebar-header.vue'
import type KeywordQuery from '@/pages/views/keyword-query.vue'
import type TimeIsQuery from '@/pages/views/time-is-query.vue'
import type RepQuery from '@/pages/views/rep-query.vue'
import type TagQuery from '@/pages/views/tag-query.vue'
import type CalendarQuery from '@/pages/views/calendar-query.vue'
import type MapQuery from '@/pages/views/map-query.vue'
import type PeriodOfTimeQuery from '@/pages/views/period-of-time-query.vue'
import { SavedFindQueryConfig, type SavedFindQueryItem } from '@/classes/datas/config/saved-find-query-config'

export function useRykvQueryEditorSideBar(options: {
    props: RykvQueryEditorSidebarProps,
    emits: RykvQueryEditorSidebarEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const sidebar_header = ref<InstanceType<typeof SidebarHeader> | null>(null)
    const keyword_query = ref<InstanceType<typeof KeywordQuery> | null>(null)
    const timeis_query = ref<InstanceType<typeof TimeIsQuery> | null>(null)
    const rep_query = ref<InstanceType<typeof RepQuery> | null>(null)
    const tag_query = ref<InstanceType<typeof TagQuery> | null>(null)
    const calendar_query = ref<InstanceType<typeof CalendarQuery> | null>(null)
    const map_query = ref<InstanceType<typeof MapQuery> | null>(null)
    const period_of_time_query = ref<InstanceType<typeof PeriodOfTimeQuery> | null>(null)

    // ── Computed ──
    const header_margin = computed(() => props.application_config.is_show_share_footer ? 12 : 6)
    const header_height: Ref<number> = computed(() => 36 * (props.application_config.is_show_share_footer ? 2 : 1) + header_margin.value.valueOf())
    const sidebar_height = computed(() => (props.app_content_height.valueOf() - header_height.value).toString().concat("px"))
    const header_top_px = computed(() => (props.app_content_height.valueOf() - header_height.value).toString().concat("px"))
    const sidebar_top_px = computed(() => (header_height.value * -1).toString().concat("px"))
    // 保存済み検索条件(ライフログ用)。0件なら呼び出しFABごと非表示にする
    const saved_find_querys = computed(() => SavedFindQueryConfig.parse(props.application_config.saved_find_query_json_data).saved_rykv_find_kyou_querys)

    // ── State refs ──
    const default_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

    // 各節の「描けた」フラグ。子へ :inited prop として降り、子は
    // 「初回同期か再同期か」の判定に使う(use-rep-query.ts / use-tag-query.ts /
    // use-time-is-query.ts / use-calendar-query.ts)。消してはいけない。
    // かつてはこれらの AND を親への @inited イベントにしていたが、
    // 「設定が来た」を表していたのは immediate の付いていない
    // application_config watch から emit する子がいるという偶然だった。
    // 画面の初期化は use-rykv-view.ts が application_config.is_loaded を
    // 直接 watch して起こすので、集約もイベントも不要になった。
    const inited_sidebar_header_for_query_sidebar = ref(true)
    const inited_keyword_query_for_query_sidebar = ref(true)
    const inited_timeis_query_for_query_sidebar = ref(false)
    const inited_rep_query_for_query_sidebar = ref(false)
    const inited_tag_query_for_query_sidebar = ref(false)
    const inited_calendar_query_for_query_sidebar = ref(false)
    const inited_map_query_for_query_sidebar = ref(false)
    const inited_period_of_time_query_for_query_sidebar = ref(true)

    // ── Watchers ──
    watch(() => props.find_kyou_query, (new_value: FindKyouQuery, old_value: FindKyouQuery) => {
        if (deep_equals(new_value, old_value)) {
            return
        }
        query.value = new_value
    })

    // immediate必須。マウント時点で設定が既に渡っているとwatcherは発火せず、
    // default_queryが空のFindKyouQueryのまま列追加・クリアに使われてしまう
    watch(() => props.application_config, () => {
        default_query.value = FindKyouQuery.generate_default_query_for_rykv(props.application_config)
    }, { immediate: true })

    // ── Business logic ──
    function get_default_query(): FindKyouQuery {
        const q = default_query.value?.clone()
        q.query_id = props.gkill_api.generate_uuid()
        if (props.application_config.rykv_default_period !== -1) {
            q.calendar_start_date = moment(moment().add(-props.application_config.rykv_default_period, "days").format("YYYY-MM-DD 00:00:00 ZZ")).toDate()
            q.calendar_end_date = moment(moment().format("YYYY-MM-DD 00:00:00 ZZ")).add(1, "days").add(-1, "milliseconds").toDate()
        }
        return q
    }

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

    function generate_query(query_id?: string): FindKyouQuery {
        const find_query = new FindKyouQuery()
        if (query_id) {
            find_query.query_id = query_id
        }

        find_query.is_focus_kyou_in_list_view = props.find_kyou_query ? props.find_kyou_query.is_focus_kyou_in_list_view : false
        find_query.is_image_only = query.value.is_image_only
        // Mi射影フラグはサイドバーにUIが無いので列のクエリから引き継ぐ。
        // 以前は5つともtrue固定で、既定列(create のみ)と必ず不一致になり、
        // 「無変更の再生成をdeep_equalsで吸収する」安全網が一度も効かなかった。
        // ??の既定値はFindKyouQueryコンストラクタと同値にする
        find_query.include_create_mi = props.find_kyou_query?.include_create_mi ?? true
        find_query.include_check_mi = props.find_kyou_query?.include_check_mi ?? false
        find_query.include_limit_mi = props.find_kyou_query?.include_limit_mi ?? false
        find_query.include_start_mi = props.find_kyou_query?.include_start_mi ?? false
        find_query.include_end_mi = props.find_kyou_query?.include_end_mi ?? false

        if (keyword_query.value) {
            // 有効時は未パースプレースホルダの[]（パースは送信直前のcloneで行う）、無効時はnull
            const use_words = keyword_query.value.get_use_words()
            find_query.words = use_words ? [] : null
            find_query.not_words = use_words ? [] : null
            find_query.words_and = keyword_query.value.get_use_word_and_search()
            find_query.keywords = keyword_query.value.get_keywords().concat()
        }

        if (timeis_query.value) {
            // グループ有効時は未パースプレースホルダの[]、無効時はnull。
            // timeis_tags はグループ有効かつタグ絞り込み有効のときだけ非null
            const use_timeis = timeis_query.value.get_use_timeis()
            find_query.timeis_words = use_timeis ? [] : null
            find_query.timeis_not_words = use_timeis ? [] : null
            find_query.timeis_keywords = timeis_query.value.get_timeis_keywords().concat()
            find_query.timeis_words_and = timeis_query.value.get_use_and_search_timeis_words()
            find_query.timeis_tags = use_timeis && timeis_query.value.get_use_timeis_tags()
                ? timeis_query.value.get_timeis_tags().concat()
                : null
            find_query.timeis_tags_and = timeis_query.value.get_use_and_search_timeis_tags()
        }

        if (rep_query.value) {
            const reps = rep_query.value.get_checked_reps()?.concat()
            const devices = rep_query.value.get_checked_devices()?.concat()
            const rep_types = rep_query.value.get_checked_rep_types()?.concat()
            if (reps) {
                find_query.reps = reps
            }
            if (devices) {
                find_query.devices_in_sidebar = devices
            }
            if (rep_types) {
                find_query.rep_types_in_sidebar = rep_types
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

        if (map_query.value) {
            const use_map = map_query.value.get_use_map()
            find_query.map_latitude = use_map ? map_query.value.get_latitude() : null
            find_query.map_longitude = use_map ? map_query.value.get_longitude() : null
            find_query.map_radius = use_map ? map_query.value.get_radius() : null
            find_query.is_enable_map_circle_in_sidebar = map_query.value.get_is_enable_circle()
        }

        if (period_of_time_query.value) {
            const use_period_of_time = period_of_time_query.value.get_use_period_of_time()
            find_query.period_of_time_start_time_second = use_period_of_time ? period_of_time_query.value.get_period_of_time_start_time_second() : null
            find_query.period_of_time_end_time_second = use_period_of_time ? period_of_time_query.value.get_period_of_time_end_time_second() : null
            find_query.period_of_time_week_of_days = use_period_of_time ? period_of_time_query.value.get_period_of_time_week_of_days() : null
        }

        find_query.apply_hide_tags(props.application_config)

        return find_query
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

    function emits_cleard_timeis_query(): void {
        const find_query = generate_query()
        find_query.query_id = query.value.query_id
        const d = get_default_query()
        find_query.timeis_words = d.timeis_words === null ? null : d.timeis_words.concat()
        find_query.timeis_not_words = d.timeis_not_words === null ? null : d.timeis_not_words.concat()
        find_query.timeis_keywords = d.timeis_keywords.concat()
        find_query.timeis_words_and = d.timeis_words_and
        find_query.timeis_tags = d.timeis_tags === null ? null : d.timeis_tags.concat()
        find_query.timeis_tags_and = d.timeis_tags_and
        query.value = find_query
        timeis_query.value?.update_check(find_query.timeis_tags ?? [], CheckState.checked, true, true)
        emits('updated_query', find_query)
    }

    function emits_cleard_rep_query(): void {
        const find_query = generate_query()
        find_query.query_id = query.value.query_id
        const d = get_default_query()
        find_query.reps = d.reps === null ? null : d.reps.concat()
        find_query.devices_in_sidebar = d.devices_in_sidebar.concat()
        find_query.rep_types_in_sidebar = d.rep_types_in_sidebar.concat()
        query.value = find_query
        rep_query.value?.update_check_devices(find_query.devices_in_sidebar, CheckState.checked, true, true)
        rep_query.value?.update_check_rep_types(find_query.rep_types_in_sidebar, CheckState.checked, true, true)
        rep_query.value?.update_check_reps(find_query.reps ?? [], CheckState.checked, true, true)
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

    function emits_cleard_map_query(): void {
        const find_query = generate_query()
        find_query.query_id = query.value.query_id
        const d = get_default_query()
        find_query.map_latitude = d.map_latitude
        find_query.map_longitude = d.map_longitude
        find_query.is_enable_map_circle_in_sidebar = d.is_enable_map_circle_in_sidebar
        find_query.map_radius = d.map_radius
        query.value = find_query
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

    function emits_cleard_period_of_time_query(): void {
        const find_query = generate_query()
        find_query.query_id = query.value.query_id
        const d = default_query.value
        find_query.period_of_time_start_time_second = d.period_of_time_start_time_second
        find_query.period_of_time_end_time_second = d.period_of_time_end_time_second
        find_query.period_of_time_week_of_days = d.period_of_time_week_of_days === null ? null : d.period_of_time_week_of_days.concat()
        query.value = find_query
        emits('updated_query', find_query)
    }

    async function emits_default_query(): Promise<void> {
        const find_query = get_default_query().clone()
        find_query.query_id = query.value.query_id
        query.value = find_query
        timeis_query.value?.update_check(find_query.timeis_tags ?? [], CheckState.checked, true, true)
        rep_query.value?.update_check_devices(find_query.devices_in_sidebar, CheckState.checked, true, true)
        rep_query.value?.update_check_rep_types(find_query.rep_types_in_sidebar, CheckState.checked, true, true)
        rep_query.value?.update_check_reps(find_query.reps ?? [], CheckState.checked, true, true)
        tag_query.value?.update_check(find_query.tags ?? [], CheckState.checked, true, true)
        emits('updated_query', find_query)
    }

    // 保存済み検索条件をサイドバーへ適用する（手で編集したのと同じ扱い。
    // ホットリロードONなら親が自動検索し、OFFなら検索ボタンで実行される）。
    // query_id は列の恒久IDなので、保存条件のものは使わず必ず現在の列のIDを維持する
    async function apply_saved_query(item: SavedFindQueryItem): Promise<void> {
        const find_query = item.find_kyou_query.clone()
        find_query.query_id = query.value.query_id
        query.value = find_query
        timeis_query.value?.update_check(find_query.timeis_tags ?? [], CheckState.checked, true, true)
        rep_query.value?.update_check_devices(find_query.devices_in_sidebar, CheckState.checked, true, true)
        rep_query.value?.update_check_rep_types(find_query.rep_types_in_sidebar, CheckState.checked, true, true)
        rep_query.value?.update_check_reps(find_query.reps ?? [], CheckState.checked, true, true)
        tag_query.value?.update_check(find_query.tags ?? [], CheckState.checked, true, true)
        emits('updated_query', find_query)
    }

    async function show_manage_share_kyou_dialog(): Promise<void> {
        throw new Error('Not implemented')
    }

    async function show_share_kyou_dialog(): Promise<void> {
        throw new Error('Not implemented')
    }

    // ── Template event handlers ──
    function onSidebarHeaderRequestedSearch(): void {
        emits('requested_search', false)
    }

    function onSidebarHeaderRequestedSearchWithUpdateCache(): void {
        emits('requested_search', true)
    }

    function onRepQueryRequestUpdateCheckedReps(_reps: Array<string>, is_by_user: boolean): void {
        if (is_by_user) emits_current_query()
    }

    function onTagQueryRequestUpdateCheckedTags(_tags: Array<string>, is_by_user: boolean): void {
        if (is_by_user) emits_current_query()
    }

    function onTimeIsQueryRequestUpdateCheckedTimeisTags(_tags: Array<string>, is_by_user: boolean): void {
        if (is_by_user) emits_current_query()
    }

    function onKeywordQueryInited(): void {
        inited_keyword_query_for_query_sidebar.value = true
    }

    function onRepQueryInited(): void {
        inited_rep_query_for_query_sidebar.value = true
    }

    function onTagQueryInited(): void {
        inited_tag_query_for_query_sidebar.value = true
    }

    function onTimeIsQueryInited(): void {
        inited_timeis_query_for_query_sidebar.value = true
    }

    function onCalendarQueryInited(): void {
        inited_calendar_query_for_query_sidebar.value = true
    }

    function onMapQueryInited(): void {
        inited_map_query_for_query_sidebar.value = true
    }

    // ── Return ──
    return {
        // Template refs
        sidebar_header,
        keyword_query,
        timeis_query,
        rep_query,
        tag_query,
        calendar_query,
        map_query,
        period_of_time_query,

        // State
        query,
        inited_sidebar_header_for_query_sidebar,
        inited_keyword_query_for_query_sidebar,
        inited_timeis_query_for_query_sidebar,
        inited_rep_query_for_query_sidebar,
        inited_tag_query_for_query_sidebar,
        inited_calendar_query_for_query_sidebar,
        inited_map_query_for_query_sidebar,
        inited_period_of_time_query_for_query_sidebar,

        // Computed
        header_margin,
        header_height,
        sidebar_height,
        header_top_px,
        sidebar_top_px,
        saved_find_querys,

        // Business logic / exposed
        generate_query,
        get_default_query,
        apply_saved_query,

        // Template event handlers
        emits_current_query,
        emits_cleard_keyword_query,
        emits_cleard_timeis_query,
        emits_cleard_rep_query,
        emits_cleard_tag_query,
        emits_cleard_map_query,
        emits_cleard_calendar_query,
        emits_cleard_period_of_time_query,
        emits_default_query,
        show_manage_share_kyou_dialog,
        show_share_kyou_dialog,
        onSidebarHeaderRequestedSearch,
        onSidebarHeaderRequestedSearchWithUpdateCache,
        onRepQueryRequestUpdateCheckedReps,
        onTagQueryRequestUpdateCheckedTags,
        onTimeIsQueryRequestUpdateCheckedTimeisTags,
        onKeywordQueryInited,
        onRepQueryInited,
        onTagQueryInited,
        onTimeIsQueryInited,
        onCalendarQueryInited,
        onMapQueryInited,
    }
}
