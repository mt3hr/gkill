import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { deepEquals } from '@/classes/deep-equals'
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
            default_query.value = generate_query().clone()
            default_query.value.query_id = props.gkill_api.generate_uuid()
            nextTick(() => {
                if (props.find_kyou_query.query_id === "") {
                    query.value = default_query.value
                }
                loading.value = false
                nextTick(() => emits('inited'))
            })
        }
    })

    watch(() => props.find_kyou_query, (new_value: FindKyouQuery, old_value: FindKyouQuery) => {
        if (deepEquals(new_value, old_value)) {
            return
        }
        query.value = new_value
    })

    // ── Business logic ──
    function get_default_query(): FindKyouQuery {
        const q = FindKyouQuery.generate_default_query_for_rykv(props.application_config)
        q.query_id = props.gkill_api.generate_uuid()
        if (props.application_config.rykv_default_period !== -1) {
            q.use_calendar = true
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
            find_query.use_words = keyword_query.value.get_use_words()
            find_query.words_and = keyword_query.value.get_use_word_and_search()
            find_query.keywords = keyword_query.value.get_keywords().concat()
        }

        if (timeis_query.value) {
            find_query.use_timeis = timeis_query.value.get_use_timeis()
            find_query.timeis_keywords = timeis_query.value.get_timeis_keywords().concat()
            find_query.timeis_words_and = timeis_query.value.get_use_and_search_timeis_words()
            find_query.use_timeis_tags = timeis_query.value.get_use_timeis_tags()
            find_query.timeis_tags = timeis_query.value.get_timeis_tags().concat()
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
            find_query.use_map = map_query.value.get_use_map()
            find_query.map_latitude = map_query.value.get_latitude()
            find_query.map_longitude = map_query.value.get_longitude()
            find_query.map_radius = map_query.value.get_radius()
            find_query.is_enable_map_circle_in_sidebar = map_query.value.get_is_enable_circle()
        }

        if (period_of_time_query.value) {
            find_query.use_period_of_time = period_of_time_query.value.get_use_period_of_time()
            find_query.period_of_time_start_time_second = period_of_time_query.value.get_period_of_time_start_time_second()
            find_query.period_of_time_end_time_second = period_of_time_query.value.get_period_of_time_end_time_second()
            find_query.period_of_time_week_of_days = period_of_time_query.value.get_period_of_time_week_of_days()
        }

        find_query.apply_hide_tags(props.application_config)

        return find_query
    }

    // ── Template event handlers ──
    function emits_current_query(): void {
        // const currentt_query = generate_query(query.value.query_id)
        // emits('updated_query', currentt_query)
    }

    function emits_cleard_keyword_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        find_query.use_words = get_default_query().use_words
        find_query.keywords = get_default_query().keywords.concat()
        find_query.words_and = get_default_query().words_and
        query.value = find_query
    }

    function emits_cleard_timeis_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        find_query.use_timeis = get_default_query().use_timeis
        find_query.use_timeis_tags = get_default_query().use_timeis_tags
        find_query.timeis_keywords = get_default_query().timeis_keywords.concat()
        find_query.timeis_words_and = get_default_query().timeis_words_and
        find_query.use_timeis_tags = get_default_query().use_timeis_tags
        find_query.timeis_tags = get_default_query().timeis_tags.concat()
        find_query.timeis_tags_and = get_default_query().timeis_tags_and
        query.value = find_query
        timeis_query.value?.update_check(find_query.tags, CheckState.checked, true, true)
    }

    function emits_cleard_rep_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        find_query.reps = get_default_query().reps.concat()
        find_query.devices_in_sidebar = get_default_query().devices_in_sidebar.concat()
        find_query.rep_types_in_sidebar = get_default_query().rep_types_in_sidebar.concat()
        query.value = find_query
    }

    function emits_cleard_tag_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        find_query.tags = get_default_query().tags.concat()
        find_query.tags_and = get_default_query().tags_and
        query.value = find_query
        tag_query.value?.update_check(find_query.tags, CheckState.checked, true, true)
    }

    function emits_cleard_map_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        find_query.use_map = get_default_query().use_map
        find_query.map_latitude = get_default_query().map_latitude
        find_query.map_longitude = get_default_query().map_longitude
        find_query.is_enable_map_circle_in_sidebar = get_default_query().is_enable_map_circle_in_sidebar
        find_query.map_radius = get_default_query().map_radius
        query.value = find_query
    }

    function emits_cleard_period_of_time_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        find_query.use_calendar = get_default_query().use_calendar
        find_query.use_period_of_time = default_query.value.use_period_of_time
        find_query.period_of_time_start_time_second = default_query.value.period_of_time_start_time_second
        find_query.period_of_time_end_time_second = default_query.value.period_of_time_end_time_second
        find_query.period_of_time_week_of_days = default_query.value.period_of_time_week_of_days.concat()
        query.value = find_query
        emits('updated_query', find_query)
    }

    async function emits_default_query(): Promise<void> {
        const find_query = get_default_query().clone()
        find_query.query_id = props.gkill_api.generate_uuid()
        await tag_query.value?.update_check(find_query.tags, CheckState.checked, true, true)
        await timeis_query.value?.update_check(find_query.tags, CheckState.checked, true, true)
        query.value = find_query
    }

    function onRepQueryRequestUpdateCheckedReps(_reps: any, is_by_user: boolean): void {
        if (is_by_user) emits_current_query()
    }

    function onTagQueryRequestUpdateCheckedTags(_tags: any, is_by_user: boolean): void {
        if (is_by_user) emits_current_query()
    }

    function onTimeisQueryRequestUpdateCheckedTimeisTags(...params: any[]): void {
        const is_by_user = params[1]
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
