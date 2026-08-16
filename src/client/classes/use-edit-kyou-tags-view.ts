import { computed, type Ref, ref, watch } from 'vue'
import type { EditKyouTagsViewProps } from '@/pages/views/edit-kyou-tags-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { Tag } from '@/classes/datas/tag'
import { GetTagsByTargetIDRequest } from '@/classes/api/req_res/get-tags-by-target-id-request'
import { parse_tag_names, tag_name_separator } from '@/classes/kyou-tags'

/** タグ絞り込みの「タグ無し」を表す番兵。実在のタグではないので履歴チップに出さない */
const no_tags_sentinel = 'no tags'

/**
 * Kyouの追加/編集画面に埋め込むタグ欄。
 *
 * 自分ではAPIを叩かない（既存タグの読み込みを除く）。集めた値は `defineExpose` で親へ渡し、
 * 実際の登録は親の save() が Kyou 本体を書いたあとに `classes/kyou-tags.ts` 経由で行う。
 * `add-notification-for-add-mi-view.vue` と同じ形。
 */
export function useEditKyouTagsView(options: {
    props: EditKyouTagsViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_loading = ref(false)
    /** 新しく付けるタグ。「、」区切りの1行テキスト（add-tag-view.vue と同じ流儀） */
    const tag_names_text: Ref<string> = ref("")
    /** 対象Kyouに既に付いているタグ。追加画面では常に空 */
    const existing_tags: Ref<Array<Tag>> = ref(new Array<Tag>())
    /**
     * 「⊗」を押して削除マークが付いた既存タグのid。
     *
     * 押した時点ではサーバへ行かない。保存を押して初めて論理削除が飛ぶので、
     * 押し間違えても「↺」で戻せるしダイアログを閉じれば無かったことになる。
     */
    const removed_tag_ids: Ref<Array<string>> = ref(new Array<string>())

    const tag_history: Ref<Array<string>> = ref(
        props.gkill_api.get_saved_tag_history().filter(tag_value => tag_value !== no_tags_sentinel)
    )

    /** 削除マークが付いていない既存タグの名前。入力欄の重複除去に使う */
    const kept_tag_names = computed(() => existing_tags.value
        .filter(tag => !removed_tag_ids.value.includes(tag.id))
        .map(tag => tag.tag.toLowerCase()))

    // ── Watchers ──
    // 編集ダイアログは同じインスタンスのまま別のKyouへ差し替わりうる
    watch(() => props.kyou?.id, () => {
        tag_names_text.value = ""
        removed_tag_ids.value = []
        load()
    })

    // ── Business logic ──
    async function load(): Promise<void> {
        if (!props.kyou) {
            existing_tags.value = new Array<Tag>()
            return
        }
        try {
            is_loading.value = true
            // props.kyou.attached_tags は当てにできない。編集ビューの load() が呼ぶのは
            // load_typed_datas() だけで、添付タグは読まれていない
            const req = new GetTagsByTargetIDRequest()
            req.target_id = props.kyou.id
            const res = await props.gkill_api.get_tags_by_target_id(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            existing_tags.value = res.tags ?? new Array<Tag>()
        } finally {
            is_loading.value = false
        }
    }

    function is_removed(tag: Tag): boolean {
        return removed_tag_ids.value.includes(tag.id)
    }

    /** 「⊗」と「↺」の両方がこれを呼ぶ。押すたびに削除マークが入れ替わる */
    function toggle_remove(tag: Tag): void {
        if (props.is_readonly) {
            return
        }
        const index = removed_tag_ids.value.indexOf(tag.id)
        if (index === -1) {
            removed_tag_ids.value.push(tag.id)
            return
        }
        removed_tag_ids.value.splice(index, 1)
    }

    /** 履歴チップを押したら入力欄の末尾へ足す。既に書かれていれば何もしない */
    function append_history_tag(tag_value: string): void {
        if (props.is_readonly) {
            return
        }
        const current = parse_tag_names(tag_names_text.value)
        const appended = parse_tag_names(tag_value)
            .filter(tag_name => !current.some(exist => exist.toLowerCase() === tag_name.toLowerCase()))
        if (appended.length === 0) {
            return
        }
        tag_names_text.value = current.concat(appended).join(tag_name_separator)
    }

    // ── Exposed to parent ──
    /**
     * 新しく付けるタグ名。
     *
     * 入力欄の中の重複に加えて、**削除マークの付いていない既存タグと同名のものも落とす**。
     * サーバの重複チェックはタグIDだけを見るので、落とさないと同じ名前が2件付く。
     */
    function get_tag_names(): Array<string> {
        return parse_tag_names(tag_names_text.value)
            .filter(tag_name => !kept_tag_names.value.includes(tag_name.toLowerCase()))
    }

    /** 「⊗」が付いた既存タグ。追加画面では常に空 */
    function get_removed_tags(): Array<Tag> {
        return existing_tags.value.filter(tag => removed_tag_ids.value.includes(tag.id))
    }

    /** 親の「更新がなかったらエラー」ガードを緩めるかの判定に使う */
    function has_pending_changes(): boolean {
        return get_tag_names().length !== 0 || get_removed_tags().length !== 0
    }

    /** 親の「リセット」から呼ぶ。入力欄と削除マークの両方を戻す */
    function reset(): void {
        tag_names_text.value = ""
        removed_tag_ids.value = []
    }

    // ── Init calls ──
    load()

    // ── Return ──
    return {
        // State
        is_loading,
        tag_names_text,
        existing_tags,
        tag_history,

        // Business logic / template handlers
        is_removed,
        toggle_remove,
        append_history_tag,

        // Exposed to parent
        get_tag_names,
        get_removed_tags,
        has_pending_changes,
        reset,
    }
}
