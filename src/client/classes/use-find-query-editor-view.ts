import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { deep_equals } from '@/classes/deep-equals'
import moment from 'moment'
import { CheckState } from '@/pages/views/check-state'
import type { FindQueryEditorViewProps } from '@/pages/views/find-query-editor-view-props'
import type { FindQueryEditorViewEmits } from '@/pages/views/find-query-editor-view-emits'
import type KeywordQuery from '@/pages/views/keyword-query.vue'
import type TimeIsQuery from '@/pages/views/time-is-query.vue'
import type RepQuery from '@/pages/views/rep-query.vue'
import type TagQuery from '@/pages/views/tag-query.vue'
import type MapQuery from '@/pages/views/map-query.vue'
import type PeriodOfTimeQuery from '@/pages/views/period-of-time-query.vue'

export function useFindQueryEditorView(options: {
    props: FindQueryEditorViewProps,
    emits: FindQueryEditorViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const keyword_query = ref<InstanceType<typeof KeywordQuery> | null>(null)
    const timeis_query = ref<InstanceType<typeof TimeIsQuery> | null>(null)
    const rep_query = ref<InstanceType<typeof RepQuery> | null>(null)
    const tag_query = ref<InstanceType<typeof TagQuery> | null>(null)
    const map_query = ref<InstanceType<typeof MapQuery> | null>(null)
    const period_of_time_query = ref<InstanceType<typeof PeriodOfTimeQuery> | null>(null)

    // ── State refs ──
    const default_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const is_mounted = ref(false)
    nextTick(() => is_mounted.value = true)

    const loading: Ref<boolean> = ref(true)
    const inited_keyword_query_for_query_sidebar = ref(true)
    const inited_timeis_query_for_query_sidebar = ref(false)
    const inited_rep_query_for_query_sidebar = ref(false)
    const inited_tag_query_for_query_sidebar = ref(false)
    const inited_map_query_for_query_sidebar = ref(false)
    const inited_period_of_time_query_for_query_sidebar = ref(true)

    // ── Computed ──
    const loading_class = computed(() => loading.value ? "loading_find_query_editor_view" : "")
    const inited = computed(() => {
        if (!is_mounted.value) {
            return false
        }

        return inited_keyword_query_for_query_sidebar.value &&
            inited_rep_query_for_query_sidebar.value &&
            inited_tag_query_for_query_sidebar.value &&
            inited_timeis_query_for_query_sidebar.value &&
            inited_map_query_for_query_sidebar.value &&
            inited_period_of_time_query_for_query_sidebar.value
    })

    // ── Watchers ──
    watch(() => inited.value, (new_value: boolean, old_value: boolean) => {
        if (old_value !== new_value && new_value) {
            // 初期値の規則: 値がセットされていれば(query_idが空でなければ)それを優先し、
            // 無ければApplicationConfigのデフォルト検索条件を適用する。
            // 以前はgenerate_query()(=まだ空の子UIの写し)を既定にしていたため、
            // 新規作成時にApplicationConfig既定が一切効かなかった
            default_query.value = get_default_query()
            nextTick(() => {
                if (props.find_kyou_query.query_id === "") {
                    query.value = default_query.value
                } else {
                    query.value = props.find_kyou_query
                }
                loading.value = false
                nextTick(() => emits('inited'))
            })
        }
    })

    watch(() => props.find_kyou_query, (new_value: FindKyouQuery, old_value: FindKyouQuery) => {
        if (deep_equals(new_value, old_value)) {
            return
        }
        query.value = new_value
    })

    // ── Business logic ──
    function get_default_query(): FindKyouQuery {
        const q = FindKyouQuery.generate_default_query_for_rykv(props.application_config)
        q.query_id = props.gkill_api.generate_uuid()
        if (props.application_config.rykv_default_period !== -1) {
            q.calendar_start_date = moment(moment().add(-props.application_config.rykv_default_period, "days").format("YYYY-MM-DD 00:00:00 ZZ")).toDate()
            q.calendar_end_date = moment(moment().format("YYYY-MM-DD 00:00:00 ZZ")).add(1, "days").add(-1, "milliseconds").toDate()
        }
        return q
    }

    function generate_query(query_id?: string): FindKyouQuery {
        const find_query = new FindKyouQuery()
        if (query_id) {
            find_query.query_id = query_id
        }

        find_query.is_focus_kyou_in_list_view = props.find_kyou_query! ? props.find_kyou_query.is_focus_kyou_in_list_view : false
        find_query.is_image_only = props.find_kyou_query ? props.find_kyou_query.is_image_only : false

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

    // ── Template event handlers ──
    function emits_current_query(): void {
        // クエリ変更はSave時のみ反映
    }

    function emits_cleard_keyword_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        const d = get_default_query()
        find_query.words = d.words === null ? null : d.words.concat()
        find_query.not_words = d.not_words === null ? null : d.not_words.concat()
        find_query.keywords = d.keywords.concat()
        find_query.words_and = d.words_and
        query.value = find_query
    }

    function emits_cleard_timeis_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        const d = get_default_query()
        find_query.timeis_words = d.timeis_words === null ? null : d.timeis_words.concat()
        find_query.timeis_not_words = d.timeis_not_words === null ? null : d.timeis_not_words.concat()
        find_query.timeis_keywords = d.timeis_keywords.concat()
        find_query.timeis_words_and = d.timeis_words_and
        find_query.timeis_tags = d.timeis_tags === null ? null : d.timeis_tags.concat()
        find_query.timeis_tags_and = d.timeis_tags_and
        query.value = find_query
        timeis_query.value?.update_check(find_query.timeis_tags ?? [], CheckState.checked, true, true)
    }

    function emits_cleard_rep_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        const d = get_default_query()
        find_query.reps = d.reps === null ? null : d.reps.concat()
        find_query.devices_in_sidebar = d.devices_in_sidebar.concat()
        find_query.rep_types_in_sidebar = d.rep_types_in_sidebar.concat()
        query.value = find_query
    }

    function emits_cleard_tag_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        const d = get_default_query()
        find_query.tags = d.tags === null ? null : d.tags.concat()
        find_query.tags_and = d.tags_and
        query.value = find_query
        tag_query.value?.update_check(find_query.tags ?? [], CheckState.checked, true, true)
    }

    function emits_cleard_map_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        const d = get_default_query()
        find_query.map_latitude = d.map_latitude
        find_query.map_longitude = d.map_longitude
        find_query.is_enable_map_circle_in_sidebar = d.is_enable_map_circle_in_sidebar
        find_query.map_radius = d.map_radius
        query.value = find_query
    }

    function emits_cleard_period_of_time_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        const d = default_query.value
        find_query.period_of_time_start_time_second = d.period_of_time_start_time_second
        find_query.period_of_time_end_time_second = d.period_of_time_end_time_second
        find_query.period_of_time_week_of_days = d.period_of_time_week_of_days === null ? null : d.period_of_time_week_of_days.concat()
        query.value = find_query
        emits('updated_query', find_query)
    }

    async function emits_default_query(): Promise<void> {
        const find_query = get_default_query().clone()
        find_query.query_id = props.gkill_api.generate_uuid()
        await tag_query.value?.update_check(find_query.tags ?? [], CheckState.checked, true, true)
        await timeis_query.value?.update_check(find_query.timeis_tags ?? [], CheckState.checked, true, true)
        query.value = find_query
    }

    function onRepQueryRequestUpdateCheckedReps(_reps: string[], is_by_user: boolean): void {
        if (is_by_user) emits_current_query()
    }

    function onTagQueryRequestUpdateCheckedTags(_tags: string[], is_by_user: boolean): void {
        if (is_by_user) emits_current_query()
    }

    function onTimeisQueryRequestUpdateCheckedTimeisTags(_tags: string[], is_by_user: boolean): void {
        if (is_by_user) emits_current_query()
    }

    function onInitedKeyword(): void {
        inited_keyword_query_for_query_sidebar.value = true
    }

    function onInitedTimeis(): void {
        inited_timeis_query_for_query_sidebar.value = true
    }

    function onInitedRep(): void {
        inited_rep_query_for_query_sidebar.value = true
    }

    function onInitedTag(): void {
        inited_tag_query_for_query_sidebar.value = true
    }

    function onInitedMap(): void {
        inited_map_query_for_query_sidebar.value = true
    }

    function onSaveClicked(): void {
        emits('requested_apply', generate_query(props.gkill_api.generate_uuid()))
        emits('requested_close_dialog')
    }

    // ── Return ──
    return {
        // Template refs
        keyword_query,
        timeis_query,
        rep_query,
        tag_query,
        map_query,
        period_of_time_query,

        // State
        default_query,
        query,
        is_mounted,
        loading,
        inited_keyword_query_for_query_sidebar,
        inited_timeis_query_for_query_sidebar,
        inited_rep_query_for_query_sidebar,
        inited_tag_query_for_query_sidebar,
        inited_map_query_for_query_sidebar,
        inited_period_of_time_query_for_query_sidebar,

        // Computed
        loading_class,
        inited,

        // Exposed methods (for defineExpose)
        generate_query,
        get_default_query,

        // Template event handlers
        emits_current_query,
        emits_cleard_keyword_query,
        emits_cleard_timeis_query,
        emits_cleard_rep_query,
        emits_cleard_tag_query,
        emits_cleard_map_query,
        emits_cleard_period_of_time_query,
        emits_default_query,
        onRepQueryRequestUpdateCheckedReps,
        onTagQueryRequestUpdateCheckedTags,
        onTimeisQueryRequestUpdateCheckedTimeisTags,
        onInitedKeyword,
        onInitedTimeis,
        onInitedRep,
        onInitedTag,
        onInitedMap,
        onSaveClicked,
    }
}
