import { i18n } from '@/i18n'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { nextTick, type Ref, ref, watch } from 'vue'
import { CheckState } from '@/pages/views/check-state'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { FoldableStructModel } from '@/pages/views/foldable-struct-model'
import type { TimeIsQueryEmits } from '@/pages/views/time-is-query-emits'
import type { TimeIsQueryProps } from '@/pages/views/time-is-query-props'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

export function useTimeIsQuery(options: {
    props: TimeIsQueryProps,
    emits: TimeIsQueryEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const foldable_struct = ref<any>(null)

    // ── State refs ──
    const old_cloned_query: Ref<FindKyouQuery | null> = ref(null)
    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())
    const cloned_query: Ref<FindKyouQuery> = ref(props.find_kyou_query.clone())

    const loading = ref(false)
    const skip_emits_this_tick = ref(false)

    // ── Watchers ──
    watch(() => loading.value, async (new_value: boolean, old_value: boolean) => {
        if (new_value !== old_value && new_value) {
            const tags = cloned_query.value.tags
            if (tags) {
                await update_check(tags, CheckState.checked, true)
            }
        }
    })

    watch(() => props.application_config, async () => {
        loading.value = true
        cloned_query.value = props.find_kyou_query
        cloned_application_config.value = props.application_config.clone()
        if (props.inited) {
            skip_emits_this_tick.value = true
            nextTick(() => skip_emits_this_tick.value = false)
            update_check(cloned_query.value.timeis_tags, CheckState.checked, true)
            return
        }
        if (!props.inited) {
            emits('inited')
        }
    })

    watch(() => props.find_kyou_query, async (new_value: FindKyouQuery, old_value: FindKyouQuery) => {
        if (!new_value) {
            return
        }
        loading.value = true
        old_cloned_query.value = old_value
        cloned_query.value = new_value.clone()
        cloned_query.value = props.find_kyou_query.clone()
        await update_check_state(cloned_query.value.timeis_tags, CheckState.checked)
        const checked_items = foldable_struct.value?.get_selected_items()
        if (checked_items) {
            emits('request_update_checked_timeis_tags', checked_items, false)
        }
    })

    // ── Business logic ──
    async function clicked_items(_e: MouseEvent, items: Array<string>, check_state: CheckState): Promise<void> {
        update_check(items, check_state, true)
    }

    function get_use_timeis(): boolean {
        return cloned_query.value.use_timeis
    }
    function get_use_timeis_tags(): boolean {
        return cloned_query.value.use_timeis_tags
    }
    function get_use_and_search_timeis_words(): boolean {
        return cloned_query.value.timeis_words_and
    }
    function get_use_and_search_timeis_tags(): boolean {
        return cloned_query.value.timeis_tags_and
    }
    function get_timeis_keywords(): string {
        return cloned_query.value.timeis_keywords
    }
    function get_timeis_tags(): Array<string> {
        const tags = foldable_struct.value?.get_selected_items()
        if (tags) {
            return tags
        }
        return new Array<string>()
    }

    async function update_check_state(items: Array<string>, is_checked: CheckState): Promise<void> {
        await update_check(items, is_checked, false)
    }

    async function update_check(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean, disable_emits?: boolean): Promise<void> {
        if (pre_uncheck_all) {
            let f = (_struct: FoldableStructModel) => { }
            let func = (struct: FoldableStructModel) => {
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
            let func = (struct: FoldableStructModel) => {
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
                emits('request_update_checked_timeis_tags', checked_items, true)
            }
        }
        foldable_struct.value?.update_check()
    }

    // ── Template event handlers ──
    function onChangeUseTimeis(): void {
        emits('request_update_use_timeis_query', cloned_query.value.use_timeis)
    }

    function onClickClear(): void {
        emits('request_clear_timeis_query')
    }

    function onToggleTimeisWordsAnd(): void {
        cloned_query.value.timeis_words_and = !cloned_query.value.timeis_words_and
        emits('request_update_and_search_timeis_word', cloned_query.value.timeis_words_and)
    }

    function onChangeTimeisKeywords(): void {
        emits('request_update_timeis_keywords', cloned_query.value.timeis_keywords)
    }

    function onToggleTimeisTagsAnd(): void {
        cloned_query.value.timeis_tags_and = !cloned_query.value.timeis_tags_and
        emits('request_update_and_search_timeis_tags', cloned_query.value.timeis_tags_and)
    }

    function onClickUseTimeisTags(): void {
        cloned_query.value.use_timeis_tags = !cloned_query.value.use_timeis_tags
        emits('request_update_use_timeis_query', cloned_query.value.use_timeis_tags)
    }

    // ── Event relay objects ──
    const foldableStructHandlers = {
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
    }

    return {
        // Template refs
        foldable_struct,

        // State
        cloned_application_config,
        cloned_query,

        // Business logic
        get_use_timeis,
        get_use_and_search_timeis_words,
        get_use_and_search_timeis_tags,
        get_timeis_keywords,
        get_use_timeis_tags,
        get_timeis_tags,
        update_check,
        clicked_items,
        update_check_state,

        // Template event handlers
        onChangeUseTimeis,
        onClickClear,
        onToggleTimeisWordsAnd,
        onChangeTimeisKeywords,
        onToggleTimeisTagsAnd,
        onClickUseTimeisTags,

        // Event relay objects
        foldableStructHandlers,
    }
}
