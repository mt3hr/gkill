import { ref, watch, type Ref } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { KeywordQueryEmits } from '@/pages/views/keyword-query-emits'
import type { KeywordQueryProps } from '@/pages/views/keyword-query-props'

export function useKeywordQuery(options: {
    props: KeywordQueryProps,
    emits: KeywordQueryEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const cloned_find_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

    // ── Watchers ──
    watch(() => props.find_kyou_query, () => {
        if (!props.find_kyou_query) {
            return
        }
        cloned_find_query.value = props.find_kyou_query.clone()
        emits('inited')
    })

    // ── Business logic ──
    function get_keywords(): string {
        return cloned_find_query.value.keywords
    }
    function get_use_words(): boolean {
        return cloned_find_query.value.use_words
    }
    function get_use_word_and_search(): boolean {
        return cloned_find_query.value.words_and
    }

    // ── Return ──
    return {
        // State
        cloned_find_query,

        // Business logic
        get_keywords,
        get_use_words,
        get_use_word_and_search,
    }
}
