import { computed, nextTick, ref, watch, type Ref } from 'vue'
import type { TagQueryEmits } from '@/pages/views/tag-query-emits'
import type { TagQueryProps } from '@/pages/views/tag-query-props'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { CheckState } from '@/pages/views/check-state'
import { apply_check_state_to_struct } from '@/classes/foldable-struct-check'

export function useTagQuery(options: {
    props: TagQueryProps,
    emits: TagQueryEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const foldable_struct = ref()

    // ── State refs ──
    const use_tag: Ref<boolean> = ref(true)
    const is_and_search: Ref<boolean> = ref(false)
    const old_cloned_query: Ref<FindKyouQuery | null> = ref(null)
    const cloned_query: Ref<FindKyouQuery> = ref(props.find_kyou_query.clone())
    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())
    const skip_emits_this_tick = ref(false)

    // ── Computed ──
    const tag_struct = computed(() => cloned_application_config.value.tag_struct)

    // ── Internal helpers ──
    async function init_tag_struct() {
        cloned_application_config.value = props.application_config.clone()
        skip_emits_this_tick.value = true
        nextTick(() => skip_emits_this_tick.value = false)
        // tags は null（フィルタ未使用。plaing検索の既定クエリ等）でありうる
        update_check(cloned_query.value.tags ?? [], CheckState.checked, true, true)
        if (!props.inited) {
            emits('inited')
        }
    }

    // ── Watchers ──
    watch(() => props.application_config.tag_struct, () => init_tag_struct())

    watch(() => props.find_kyou_query, async (new_value: FindKyouQuery, old_value: FindKyouQuery) => {
        if (!new_value) return

        old_cloned_query.value = old_value
        cloned_query.value = new_value.clone()
        is_and_search.value = props.find_kyou_query.tags_and

        await nextTick()

        update_check(cloned_query.value.tags ?? [], CheckState.checked, true, true)

        const checked_items = foldable_struct.value?.get_selected_items()
        if (checked_items) {
            emits('request_update_checked_tags', checked_items, false)
        }
    })

    // ── Initialization ──
    init_tag_struct()

    // ── Methods ──
    async function clicked_items(e: MouseEvent, items: Array<string>, is_checked: CheckState): Promise<void> {
        update_check(items, is_checked, true, false)
    }

    function update_check_state(items: Array<string>, is_checked: CheckState) {
        update_check(items, is_checked, false, false)
    }

    function update_check(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean, disable_emits?: boolean) {
        apply_check_state_to_struct(cloned_application_config.value.tag_struct, items, is_checked, pre_uncheck_all)

        const checked_items = foldable_struct.value?.get_selected_items()
        if (checked_items) {
            if (!skip_emits_this_tick.value && !disable_emits) {
                emits('request_update_checked_tags', checked_items, true)
            }
        }
        foldable_struct.value?.update_check()
    }

    function get_use_tag(): boolean {
        return use_tag.value
    }

    function get_tags(): Array<string> | null {
        const tags = foldable_struct.value?.get_selected_items()
        if (!tags) {
            return null
        }
        return tags
    }

    function get_is_and_search(): boolean {
        return is_and_search.value
    }

    // ── Return ──
    return {
        // Template refs
        foldable_struct,

        // State
        use_tag,
        is_and_search,
        tag_struct,

        // Methods used in template
        clicked_items,
        update_check_state,

        // Exposed methods
        get_use_tag,
        get_tags,
        get_is_and_search,
        update_check,
    }
}
