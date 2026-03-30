import { computed, nextTick, ref, watch, type Ref } from 'vue'
import type { TagQueryEmits } from '@/pages/views/tag-query-emits'
import type { TagQueryProps } from '@/pages/views/tag-query-props'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { CheckState } from '@/pages/views/check-state'
import type { FoldableStructModel } from '@/pages/views/foldable-struct-model'

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
        update_check(cloned_query.value.tags, CheckState.checked, true, true)
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
        if (pre_uncheck_all) {
            let f = (_struct: FoldableStructModel) => { }
            const func = (struct: FoldableStructModel) => {
                struct.is_checked = false
                struct.indeterminate = false
                if (struct.children) {
                    struct.children.forEach(child => {
                        f(child)
                    })
                }
            }
            f = func
            f(cloned_application_config.value.tag_struct)
        }

        for (let i = 0; i < items.length; i++) {
            const key_name = items[i]
            let f = (_struct: FoldableStructModel) => { }
            const func = (struct: FoldableStructModel) => {
                if (struct.key === key_name) {
                    switch (is_checked) {
                        case CheckState.checked:
                            struct.is_checked = true
                            struct.indeterminate = false
                            break
                        case CheckState.unchecked:
                            struct.is_checked = false
                            struct.indeterminate = false
                            break
                        case CheckState.indeterminate:
                            struct.is_checked = false
                            struct.indeterminate = true
                            break
                    }
                }
                if (struct.children) {
                    struct.children.forEach(child => {
                        f(child)
                    })
                }
            }
            f = func
            f(cloned_application_config.value.tag_struct)
        }

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
