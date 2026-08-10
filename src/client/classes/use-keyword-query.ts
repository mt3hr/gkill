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
    // チェックボックスのUI状態。クエリ上は words の null 判定が担うため、
    // ローカルrefに分離してprops片方向同期する（use-period-of-time-queryと同じパターン）
    const use_words: Ref<boolean> = ref(props.find_kyou_query ? props.find_kyou_query.words !== null : false)

    // ── Watchers ──
    watch(() => props.find_kyou_query, () => {
        if (!props.find_kyou_query) {
            return
        }
        cloned_find_query.value = props.find_kyou_query.clone()
        use_words.value = props.find_kyou_query.words !== null
        emits('inited')
    })

    // ── Business logic ──
    function get_keywords(): string {
        return cloned_find_query.value.keywords
    }
    function get_use_words(): boolean {
        return use_words.value
    }
    function get_use_word_and_search(): boolean {
        return cloned_find_query.value.words_and
    }

    // ── Return ──
    return {
        // State
        cloned_find_query,
        use_words,

        // Business logic
        get_keywords,
        get_use_words,
        get_use_word_and_search,
    }
}
