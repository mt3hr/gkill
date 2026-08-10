'use strict'

import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { deep_equals } from '@/classes/deep-equals'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import { CheckState } from '@/pages/views/check-state'
import type { MiFindQueryEditorViewProps } from '@/pages/views/mi-find-query-editor-view-props'
import type { MiFindQueryEditorViewEmits } from '@/pages/views/mi-find-query-editor-view-emits'
import type KeywordQuery from '@/pages/views/keyword-query.vue'
import type TagQuery from '@/pages/views/tag-query.vue'
import type miExtractCheckStateQuery from '@/pages/views/mi-extract-check-state-query.vue'
import type miSortTypeQuery from '@/pages/views/mi-sort-type-query.vue'

export function useMiFindQueryEditorView(options: {
    props: MiFindQueryEditorViewProps,
    emits: MiFindQueryEditorViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const keyword_query = ref<InstanceType<typeof KeywordQuery> | null>(null)
    const tag_query = ref<InstanceType<typeof TagQuery> | null>(null)
    const check_state_query = ref<InstanceType<typeof miExtractCheckStateQuery> | null>(null)
    const sort_type_query = ref<InstanceType<typeof miSortTypeQuery> | null>(null)

    // ── State refs ──
    const default_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const is_mounted = ref(false)
    nextTick(() => is_mounted.value = true)

    const loading: Ref<boolean> = ref(true)
    const inited_keyword_query_for_query_sidebar = ref(true)
    const inited_tag_query_for_query_sidebar = ref(false)
    const inited_check_state_query_for_query_sidebar = ref(false)
    const inited_sort_query_for_query_sidebar = ref(false)

    // ── Computed ──
    const loading_class = computed(() => loading.value ? "loading_mi_find_query_editor_view" : "")

    const inited = computed(() => {
        if (!is_mounted.value) {
            return false
        }
        // ここに載せてよいのは実際に @inited を発火する子だけ。
        // 画面から消した子のフラグを残すと永久にfalseのままで loading が晴れない
        return inited_keyword_query_for_query_sidebar.value &&
            inited_tag_query_for_query_sidebar.value &&
            inited_check_state_query_for_query_sidebar.value &&
            inited_sort_query_for_query_sidebar.value
    })

    // ── Watchers ──
    watch(() => inited.value, (new_value: boolean, old_value: boolean) => {
        if (old_value !== new_value && new_value) {
            // 初期値の規則: 値がセットされていれば(query_idが空でなければ)それを優先し、
            // 無ければApplicationConfig既定(mi用)を適用する。
            // 以前はgenerate_query()(=まだ空の子UIの写し)を既定にしていたため、
            // 新規作成時にApplicationConfig既定がrepsしか効かなかった
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
        const q = FindKyouQuery.generate_default_query_for_mi(props.application_config)
        q.query_id = props.gkill_api.generate_uuid()
        // mi_board_name はコンストラクタ既定の null（=「すべて」）のまま。番兵文字列は表示層専用
        q.rep_types = ["mi"]
        return q
    }

    function generate_query(query_id?: string): FindKyouQuery {
        const find_query = new FindKyouQuery()
        if (query_id) {
            find_query.query_id = query_id
        }
        find_query.for_mi = true
        // このエディタに板選択UIは無いので常に「すべて」（=null）
        find_query.mi_board_name = null
        find_query.reps = get_default_query().reps
        find_query.rep_types = ["mi"]

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

        // 状況(TimeIs)・時間帯・場所はこのエディタから外したので、
        // それらのフィールドは FindKyouQuery の既定値（すべてOFF）のままにする

        find_query.apply_hide_tags(props.application_config)

        return find_query
    }

    // ── Template event handlers ──
    function emits_current_query(): void {
        // クエリ変更はSave時のみ反映
    }

    function emits_cleard_check_state(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        find_query.mi_check_state = get_default_query().mi_check_state
        query.value = find_query
    }

    function emits_cleard_sort_type_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        find_query.mi_sort_type = get_default_query().mi_sort_type
        query.value = find_query
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

    function emits_cleard_tag_query(): void {
        const find_query = generate_query()
        find_query.query_id = props.gkill_api.generate_uuid()
        const d = get_default_query()
        find_query.tags = d.tags === null ? null : d.tags.concat()
        find_query.tags_and = d.tags_and
        query.value = find_query
        tag_query.value?.update_check(find_query.tags ?? [], CheckState.checked, true, true)
    }

    async function emits_default_query(): Promise<void> {
        const find_query = get_default_query().clone()
        find_query.query_id = props.gkill_api.generate_uuid()
        await tag_query.value?.update_check(find_query.tags ?? [], CheckState.checked, true, true)
        query.value = find_query
    }

    function onTagQueryRequestUpdateCheckedTags(_tags: string[], is_by_user: boolean): void {
        if (is_by_user) emits_current_query()
    }

    function onInitedKeyword(): void {
        inited_keyword_query_for_query_sidebar.value = true
    }

    function onInitedTag(): void {
        inited_tag_query_for_query_sidebar.value = true
    }

    function onInitedCheckState(): void {
        inited_check_state_query_for_query_sidebar.value = true
    }

    function onInitedSort(): void {
        inited_sort_query_for_query_sidebar.value = true
    }

    function onSaveClicked(): void {
        emits('requested_apply', generate_query(props.gkill_api.generate_uuid()))
        emits('requested_close_dialog')
    }

    // ── Return ──
    return {
        // Template refs
        keyword_query,
        tag_query,
        check_state_query,
        sort_type_query,

        // State
        default_query,
        query,
        is_mounted,
        loading,
        inited_keyword_query_for_query_sidebar,
        inited_tag_query_for_query_sidebar,
        inited_check_state_query_for_query_sidebar,
        inited_sort_query_for_query_sidebar,

        // Computed
        loading_class,
        inited,

        // Exposed methods
        generate_query,
        get_default_query,

        // Template event handlers
        emits_current_query,
        emits_cleard_check_state,
        emits_cleard_sort_type_query,
        emits_cleard_keyword_query,
        emits_cleard_tag_query,
        emits_default_query,
        onTagQueryRequestUpdateCheckedTags,
        onInitedKeyword,
        onInitedTag,
        onInitedCheckState,
        onInitedSort,
        onSaveClicked,
    }
}
