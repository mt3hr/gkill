'use strict'

// 既知タグと tags_and の列を触らない理由、および直らない範囲:
// documents/adr/0033-add-unknown-tag-to-column-filter.md

import { nextTick, type Ref } from 'vue'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Tag } from '@/classes/datas/tag'

/**
 * 利用者がその場で作った新しいタグを、開いている列の検索条件へ足す。
 *
 * 直している不具合: **タグを付けて追加した記録が、追加した直後に一覧から消える。**
 *
 * 既定クエリは「絞らない」を `tags = null` ではなく
 * 「そのときの check_when_inited タグ名の列挙」として物質化する
 * (`find-kyou-query.ts` の generate_default_query_for_rykv)。
 * それが localStorage の列状態へ丸ごと落ちる一方、タグ宇宙(tag_struct)は
 * 毎回サーバから引き直して育つので、**列の条件だけが保存時点で凍る**。
 * タグが1つも無い時期に作られた列は `tags = ["no tags"]` の1件だけになり、
 * `tags` は非nullなのでフィルタは有効 → タグの付いた記録は1件も通らない。
 * 落ちるのはサーバ検索(find_filter.go)と局所挿入(kyou-local-insert.ts の matches_tags)の両方で、
 * **エラーも警告も出ない**。
 *
 * ここで使ってよい情報は「そのタグがタグツリーに無かった」という**決定可能な事実だけ**。
 * 未知だった＝利用者がついさっき作った＝「意図的にチェックを外した」ことは原理的にありえない。
 * 逆に既知のタグは、外れているなら利用者が外したのかもしれないので**触らない**
 * (「保存後に増えたタグ」と「利用者が外したタグ」は現状の保存データでは区別できない)。
 *
 * rykv と mi はコピー由来の対称実装なので、手順はここに1つだけ置く。
 */

// ── Types ──

export type NewTagColumnPatchDecision =
    | { kind: 'patch', tags: Array<string> }
    | { kind: 'skip', reason: 'tags_unused' | 'tags_and' | 'already_included' }

export interface RegisteredTagColumnFilterOptions {
    querys: Ref<Array<FindKyouQuery>>
    querys_backup: Ref<Array<FindKyouQuery>>
    /** そのタグ名が既にタグツリーにあるか。ビューは props.application_config.tag_struct を包んで渡す */
    is_known_tag_name: (tag_name: string) => boolean
    /** 条件を書き換えた列の引き直し。ビューの同名関数をそのまま渡す */
    reload_list_by_query_id: (query_id: string) => Promise<void>
    /** focused_query に触れうる書き込みを1tick抑止で包む。ビューの同名関数を渡す */
    run_with_sidebar_search_suppressed: (fn: () => void) => void
}

// ── Pure ──

/**
 * 列1つぶんの判定。
 *
 * **`tags_and === true` の列には足してはいけない。** AND は
 * `query.tags.every(...)`（kyou-local-insert.ts の matches_tags、find_filter.go と一致）なので、
 * `["no tags", "新タグ"]` の積は
 *   - 新タグ付きの記録 … has_no_tags === false で落ちる
 *   - タグ無しの記録   … has_tag_name(新タグ) で落ちる
 * となり**必ず空**になる。AND 列では足しても目当ての記録は救えず、
 * ほかの記録を巻き込んで列を丸ごと消すだけ。
 */
export function decide_new_tag_column_patch(
    query: FindKyouQuery,
    new_tag_names: ReadonlyArray<string>,
): NewTagColumnPatchDecision {
    // null = タグで絞っていない。足す必要が無い（記録は元から通る）
    if (query.tags === null) {
        return { kind: 'skip', reason: 'tags_unused' }
    }
    if (query.tags_and) {
        return { kind: 'skip', reason: 'tags_and' }
    }
    const current = query.tags
    const missing = new_tag_names.filter(tag_name => tag_name !== '' && !current.includes(tag_name))
    if (missing.length === 0) {
        // 差分ゼロで引き直すと、意味の無い検索が列の数だけ飛ぶ
        return { kind: 'skip', reason: 'already_included' }
    }
    return { kind: 'patch', tags: current.concat(missing) }
}

// ── Composable ──

export function useRegisteredTagColumnFilter(options: RegisteredTagColumnFilterOptions) {
    const {
        querys,
        querys_backup,
        is_known_tag_name,
        reload_list_by_query_id,
        run_with_sidebar_search_suppressed,
    } = options

    // 1tick ぶんを溜めてから流す。add_tags_to_target はタグを1件ずつ登録して
    // 1件ずつ emit する(kyou-tags.ts)ので、まとめないと新タグ3つで
    // 列あたり3本 search() が走り、2本が abort されるだけになる
    const pending_new_tag_names = new Set<string>()
    let flush_scheduled = false

    function flush(): void {
        flush_scheduled = false
        const new_tag_names = Array.from(pending_new_tag_names)
        pending_new_tag_names.clear()
        if (new_tag_names.length === 0) {
            return
        }

        // await を挟まないので index は安定している。
        // 書き換えた列の query_id だけ控えて、そのあとで引き直す
        const patched_query_ids = new Array<string>()
        run_with_sidebar_search_suppressed(() => {
            for (let i = 0; i < querys.value.length; i++) {
                const query = querys.value[i]
                if (!query) {
                    continue
                }
                const decision = decide_new_tag_column_patch(query, new_tag_names)
                if (decision.kind === 'skip') {
                    continue
                }
                const next = query.clone()
                next.tags = decision.tags
                // querys と querys_backup は同じ tick で揃える。
                // backup がずれると、サイドバーの機械的な残響が search() の
                // deep_equals 早期returnで落ちなくなる
                querys.value.splice(i, 1, next)
                // 並びがずれている控えは触らない。埋めに行くと splice が末尾へ足して
                // 並びを壊し、以後どの列でも早期returnが効かなくなる
                const backup = querys_backup.value[i]
                if (backup && backup.query_id === next.query_id) {
                    querys_backup.value.splice(i, 1, next.clone())
                }
                patched_query_ids.push(next.query_id)
            }
        })

        // localStorage へは自分で書かない。search() が必ず set_saved_* を通るので、
        // 引き直しを通せば揃う。自前で書くと「条件だけ変わって引き直さない」経路が生まれ、
        // 次回起動時だけ列が変わるという最悪の非対称になる
        for (const query_id of patched_query_ids) {
            void reload_list_by_query_id(query_id)
        }
    }

    function note_new_tag_name(tag_name: string): void {
        pending_new_tag_names.add(tag_name)
        if (flush_scheduled) {
            return
        }
        flush_scheduled = true
        void nextTick(flush)
    }

    /**
     * `registered_tag` を親へ emit する**前**に、同期で呼ぶこと。
     *
     * emit 先(use-rykv-page.ts / use-mi-page.ts)の check_tag_update が
     * タグツリーへ足したあとでは、「利用者がついさっき作った」ことを二度と知れない。
     *
     * @returns 未知タグとして受け付けたら true（＝発生元が変更通知を publish してよい合図）
     */
    function onRegisteredTag(tag: Tag): boolean {
        const tag_name = tag.tag
        if (!tag_name || is_known_tag_name(tag_name)) {
            return false
        }
        note_new_tag_name(tag_name)
        return true
    }

    /**
     * 他画面からの通知用。**既知判定をやり直さない入口。**
     *
     * 受け手が判定し直すと必ず取りこぼす ―― 窓A/Bは同じ application_config を共有していて、
     * 通知が届く頃には発生元の check_tag_update がツリーへ足し終えている可能性が高いため。
     */
    function apply_new_tag_names(tag_names: ReadonlyArray<string>): void {
        for (const tag_name of tag_names) {
            if (tag_name) {
                note_new_tag_name(tag_name)
            }
        }
    }

    return {
        onRegisteredTag,
        apply_new_tag_names,
    }
}
