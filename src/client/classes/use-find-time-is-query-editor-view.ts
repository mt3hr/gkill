'use strict'

import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { deep_equals } from '@/classes/deep-equals'
import { CheckState } from '@/pages/views/check-state'
import type { FindTimeIsQueryEditorViewProps } from '@/pages/views/find-time-is-query-editor-view-props'
import type { FindTimeIsQueryEditorViewEmits } from '@/pages/views/find-time-is-query-editor-view-emits'
import type KeywordQuery from '@/pages/views/keyword-query.vue'
import type TagQuery from '@/pages/views/tag-query.vue'

// plaing検索（実行中TimeIs）のカスタム検索条件エディタ。
// 編集面はキーワード・タグ絞り込みトグル・タグの3ブロックだけ。
// 記録保管場所と記録タイプは選ばせない（plaing検索は常にTimeIsのrepに固定されるため）。
// ここで書き込むフィールドは generate_plaing_timeis_query のコピーリストと
// 1:1で対応させること（片方だけ増やすと「設定したのに効かない」になる）。
export function useFindTimeIsQueryEditorView(options: {
    props: FindTimeIsQueryEditorViewProps,
    emits: FindTimeIsQueryEditorViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const keyword_query = ref<InstanceType<typeof KeywordQuery> | null>(null)
    const tag_query = ref<InstanceType<typeof TagQuery> | null>(null)

    // ── State refs ──
    const default_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    // タグ絞り込みトグル（UI状態）。クエリ上は tags の null 判定が担う。
    // 従来のplaing検索はタグ絞り込みなし（tags=null）なので既定OFF。
    // これを置かないと FindKyouQuery コンストラクタ既定の tags=[]（有効・0件）が保存され、
    // 「保存しただけでタグ絞り込みが勝手にONになる」事故が起きる
    const use_tag_filter = ref(false)
    const is_mounted = ref(false)
    nextTick(() => is_mounted.value = true)

    const loading: Ref<boolean> = ref(true)
    const inited_keyword_query_for_query_sidebar = ref(true)
    const inited_tag_query_for_query_sidebar = ref(false)

    // ── Computed ──
    const loading_class = computed(() => loading.value ? "loading_find_time_is_query_editor_view" : "")

    const inited = computed(() => {
        if (!is_mounted.value) {
            return false
        }
        return inited_keyword_query_for_query_sidebar.value &&
            inited_tag_query_for_query_sidebar.value
    })

    // ── Watchers ──
    watch(() => inited.value, (new_value: boolean, old_value: boolean) => {
        if (old_value !== new_value && new_value) {
            // 初期値の規則: 値がセットされていれば(query_idが空でなければ)それを優先し、
            // 無ければApplicationConfig既定(未設定時のplaing検索と同じ条件)を適用する。
            // 以前はgenerate_query()(=まだ空の子UIの写し)を既定にしていた
            default_query.value = get_default_query()
            nextTick(() => {
                if (props.find_kyou_query.query_id === "") {
                    query.value = default_query.value
                } else {
                    query.value = props.find_kyou_query
                }
                use_tag_filter.value = query.value.tags !== null
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
        use_tag_filter.value = new_value.tags !== null
    })

    // ── Business logic ──
    function get_default_query(): FindKyouQuery {
        // 未設定時のplaing検索と同じ条件（全rep + use_tags=false）。
        // エディタの初期表示・クリアが「未設定の挙動」と定義上一致する
        const q = FindKyouQuery.generate_default_query_for_plaing_timeis(props.application_config)
        q.query_id = props.gkill_api.generate_uuid()
        return q
    }

    function generate_query(query_id?: string): FindKyouQuery {
        const find_query = new FindKyouQuery()
        if (query_id) {
            find_query.query_id = query_id
        }

        // ↓ここで書くフィールドが generate_plaing_timeis_query のコピー対象
        if (keyword_query.value) {
            // 有効時は未パースプレースホルダの[]（パースは送信直前のcloneで行う）、無効時はnull
            const use_words = keyword_query.value.get_use_words()
            find_query.words = use_words ? [] : null
            find_query.not_words = use_words ? [] : null
            find_query.words_and = keyword_query.value.get_use_word_and_search()
            find_query.keywords = keyword_query.value.get_keywords().concat()
        }

        // 記録タイプはTimeIs固定。ユーザに選ばせないので生成時に立てる
        // （use-mi-find-query-editor-view の rep_types = ["mi"] と同じ書き方）
        find_query.rep_types = ["timeis"]

        // タグ絞り込み: トグルONなら子ツリーのチェック値（未取得なら0件=[]）、OFFならnull（未使用）
        find_query.tags = use_tag_filter.value ? (tag_query.value?.get_tags()?.concat() ?? []) : null
        if (tag_query.value) {
            find_query.tags_and = tag_query.value.get_is_and_search()
        }

        // plaing_time はここでは触らない（適用側が常に強制する）

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
        use_tag_filter.value = find_query.tags !== null
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

    function onSaveClicked(): void {
        emits('requested_apply', generate_query(props.gkill_api.generate_uuid()))
        emits('requested_close_dialog')
    }

    // ── Return ──
    return {
        // Template refs
        keyword_query,
        tag_query,

        // State
        default_query,
        query,
        use_tag_filter,
        is_mounted,
        loading,
        inited_keyword_query_for_query_sidebar,
        inited_tag_query_for_query_sidebar,

        // Computed
        loading_class,
        inited,

        // Exposed methods
        generate_query,
        get_default_query,

        // Template event handlers
        emits_current_query,
        emits_cleard_keyword_query,
        emits_cleard_tag_query,
        emits_default_query,
        onTagQueryRequestUpdateCheckedTags,
        onInitedKeyword,
        onInitedTag,
        onSaveClicked,
    }
}
