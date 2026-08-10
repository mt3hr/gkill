import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { collect_inited_tag_names } from '@/classes/api/find_query/collect-inited-tag-names'
import { type Ref, ref, watch } from 'vue'
import { CheckState } from '@/pages/views/check-state'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { apply_check_state_to_struct } from '@/classes/foldable-struct-check'
import type { TimeIsQueryEmits } from '@/pages/views/time-is-query-emits'
import type { TimeIsQueryProps } from '@/pages/views/time-is-query-props'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'

export function useTimeIsQuery(options: {
    props: TimeIsQueryProps,
    emits: TimeIsQueryEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const foldable_struct = ref<ComponentRef | null>(null)

    // ── State refs ──
    const old_cloned_query: Ref<FindKyouQuery | null> = ref(null)
    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())
    const cloned_query: Ref<FindKyouQuery> = ref(props.find_kyou_query.clone())
    // チェックボックスのUI状態。クエリ上は timeis_words / timeis_tags の null 判定が担うため、
    // ローカルrefに分離してprops片方向同期する（use-period-of-time-queryと同じパターン）
    const use_timeis: Ref<boolean> = ref(props.find_kyou_query.timeis_words !== null)
    const use_timeis_tags: Ref<boolean> = ref(props.find_kyou_query.timeis_tags !== null)
    // 同一query_id内の null 着信（チェックオフ）ではローカルのタグ選択・キーワードを保持し、
    // query_id が変わったときだけ着信値でリセットするための同期済みquery_id
    const last_synced_query_id: Ref<string | null> = ref(null)

    // タグツリーへ反映するチェック集合。timeis_tags=null（グループ未使用）のときは
    // 初期チェックタグ集合へフォールバックする（既定クエリ生成と同じ集合）
    function timeis_tags_for_tree(timeis_tags: Array<string> | null): Array<string> {
        return timeis_tags ?? collect_inited_tag_names(cloned_application_config.value.tag_struct)
    }

    // ── Watchers ──
    watch(() => props.application_config, async () => {
        cloned_query.value = props.find_kyou_query
        cloned_application_config.value = props.application_config.clone()
        if (props.inited) {
            // props同期はユーザー操作ではないのでemitしない(disable_emits)。
            // タイミングフラグ(nextTickで倒す方式)はマイクロタスクの順序次第で
            // すり抜けるため使わない
            update_check(timeis_tags_for_tree(cloned_query.value.timeis_tags), CheckState.checked, true, true)
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
        old_cloned_query.value = old_value
        const query_id_changed = last_synced_query_id.value !== new_value.query_id
        last_synced_query_id.value = new_value.query_id
        const local_timeis_keywords = cloned_query.value.timeis_keywords
        cloned_query.value = props.find_kyou_query.clone()
        use_timeis.value = new_value.timeis_words !== null
        use_timeis_tags.value = new_value.timeis_tags !== null
        if (!query_id_changed && new_value.timeis_words === null) {
            // 同一query_id内のオフ着信（クエリ上はnull）ではローカルのキーワードを保持する。
            // 即時トグルで値が復活する（query_idが変わったら着信値でリセット）
            cloned_query.value.timeis_keywords = local_timeis_keywords
        }
        if (!query_id_changed && new_value.timeis_tags === null) {
            // 同一query_id内のオフ着信ではツリーのローカルタグ選択も保持する
            // （チェックrefだけ上で更新済み）
            return
        }
        // props同期はユーザー操作ではないのでemitしない(disable_emits=true)。
        // RepQuery/TagQueryの同期経路と同じ扱い。これをis_by_user=trueで流すと、
        // フォーカス切替のたびにサイドバーが実検索を発火してループする。
        // pre_uncheck_all=trueで「列のクエリの写し」に置き換える
        // (falseだと列をまたいでチェックが和集合に累積し、生成クエリが列クエリと一致しなくなる)
        await update_check(timeis_tags_for_tree(cloned_query.value.timeis_tags), CheckState.checked, true, true)
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
        return use_timeis.value
    }
    function get_use_timeis_tags(): boolean {
        return use_timeis_tags.value
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
        apply_check_state_to_struct(cloned_application_config.value.tag_struct, items, is_checked, pre_uncheck_all)

        const checked_items = foldable_struct.value?.get_selected_items()
        if (checked_items) {
            if (!disable_emits) {
                emits('request_update_checked_timeis_tags', checked_items, true)
            }
        }
        foldable_struct.value?.update_check()
    }

    // ── Template event handlers ──
    function onChangeUseTimeis(): void {
        emits('request_update_use_timeis_query', use_timeis.value)
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
        use_timeis_tags.value = !use_timeis_tags.value
        emits('request_update_use_timeis_query', use_timeis_tags.value)
    }

    // ── Event relay objects ──
    const foldableStructHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    return {
        // Template refs
        foldable_struct,

        // State
        cloned_application_config,
        cloned_query,
        use_timeis,
        use_timeis_tags,

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
