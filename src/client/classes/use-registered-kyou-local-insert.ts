'use strict'

import type { Ref } from 'vue'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Kyou } from '@/classes/datas/kyou'
import { hydrate } from '@/classes/api/hydrate'
import { new_reload_batch, refresh_kyou } from '@/classes/kyou-reload'
import { can_decide_query_locally, decide_local_insert, insert_kyou_sorted } from '@/classes/kyou-local-insert'

/**
 * 追加されたKyouを、列を再検索せずに差し込む。
 *
 * 以前は追加のたびに開いている全列が `/api/get_kyous` を投げ直していた。
 * 1件の追加で「書き込み1回 ＋ 列数ぶんのフル検索」になり、列はいったん空配列へ
 * 差し替わってから戻るので、数十万件の環境では毎回数秒の再取得とチラつきが起きていた。
 * 削除(splice)と更新(1件差し替え)は既にこの方式なので、追加だけが取り残されていた。
 *
 * rykv と mi はコピー由来の対称実装なので、手順はここに1つだけ置く。
 */

export interface RegisteredKyouLocalInsertOptions {
    querys: Ref<Array<FindKyouQuery>>
    match_kyous_list: Ref<Array<Array<Kyou>>>
    /** クライアントで判定しきれない条件の列を、従来どおり丸ごと引き直す */
    reload_list_by_query_id: (query_id: string) => Promise<void>
    /** 差し込みで列の中身が変わったあとの追随(件数カレンダー・Dnoteの再集計など) */
    onColumnMutated?: (query_id: string) => void
}

/** refresh_kyou の合流キーと同じ粒度。同じキーの列は1往復にまとめられる */
function build_refresh_group_key(query: FindKyouQuery): string {
    return query.for_mi ? `mi|${query.mi_sort_type}` : 'kyou'
}

/**
 * 引き直しに使うクエリ。
 *
 * `build_mi_reload_query` は使わない ―― あれは「既存行の data_type から
 * その列のソート種別を復元する」ための関数で、新規Kyouには列の mi_sort_type を
 * そのまま使うのが正しい。
 * 非mi列でクエリを渡すと `Kyou.reload` が `load_typed_mi()` を無条件に呼ぶため、
 * kmemoに対して無駄な `/api/get_mi` が飛ぶうえ、rykv列に載るmiの related_time を
 * その列にとって誤った値へ書き換えてしまう。
 */
function build_refresh_query(query: FindKyouQuery): FindKyouQuery | undefined {
    return query.for_mi ? query : undefined
}

export function useRegisteredKyouLocalInsert(options: RegisteredKyouLocalInsertOptions) {
    const { querys, match_kyous_list, reload_list_by_query_id } = options

    function apply_to_column(query_id: string, refreshed: Kyou): void {
        // await中に列の削除・並べ替え・再検索が起きうる。列はquery_idで引き直す
        // (index を await をまたいで持つと別の列へ差し込んでしまう)
        const current_index = querys.value.findIndex(query => query.query_id === query_id)
        if (current_index === -1) {
            return
        }
        const current_query = querys.value[current_index]
        const current_list = match_kyous_list.value[current_index]
        if (!current_query || !current_list) {
            return
        }

        // 同一インスタンスを複数の列に置くと、後段の load_typed_datas 等で副作用が出る。
        // decide_local_insert は for_mi 列で related_time / data_type を書き換えるので、
        // その意味でもクローンを渡す必要がある
        const decision = decide_local_insert(refreshed.clone(), current_query)
        if (decision.kind === 'undecidable') {
            void reload_list_by_query_id(query_id)
            return
        }
        if (decision.kind === 'skip') {
            return
        }
        let is_mutated = false
        for (const row of decision.rows) {
            is_mutated = insert_kyou_sorted(current_list, row, current_query) || is_mutated
        }
        if (is_mutated) {
            options.onColumnMutated?.(query_id)
        }
    }

    async function insert_registered_kyou(raw: unknown): Promise<void> {
        // add_* の応答は hydrate を通っていない生JSONで、related_time が文字列のまま
        // clone()/reload() も生えていない。実体化しないと refresh_kyou が落ちる
        const kyou = raw instanceof Kyou ? raw : hydrate(new Kyou(), raw)
        if (!kyou.id) {
            return
        }

        // 列・focused・開いているダイアログが同じ追加を受けて独立に引き直すので、
        // 同じ値を渡して1往復に合流させる
        const requested_at = new_reload_batch()

        // 列の同一性は query_id。ここでスナップショットを取り、以降 index は使わない
        const groups = new Map<string, { query: FindKyouQuery | undefined, query_ids: Array<string> }>()
        for (const query of querys.value) {
            if (!query) {
                continue
            }
            const gate = can_decide_query_locally(query)
            if (!gate.ok) {
                void reload_list_by_query_id(query.query_id)
                continue
            }
            const group_key = build_refresh_group_key(query)
            const group = groups.get(group_key)
            if (group) {
                group.query_ids.push(query.query_id)
                continue
            }
            groups.set(group_key, { query: build_refresh_query(query), query_ids: [query.query_id] })
        }
        if (groups.size === 0) {
            return
        }

        // グループ単位で並行に引く。列ごとに直列awaitすると列数ぶん待つことになる
        await Promise.all(Array.from(groups.values()).map(async (group) => {
            const refreshed = await refresh_kyou(kyou, group.query, requested_at)
            // 引けなかった列は従来どおり引き直す(半端な行を差し込むより安全)
            if (!refreshed || !refreshed.id) {
                for (const query_id of group.query_ids) {
                    void reload_list_by_query_id(query_id)
                }
                return
            }
            // 消えたものは差し込まない。再検索も要らない
            if (refreshed.is_deleted) {
                return
            }
            for (const query_id of group.query_ids) {
                apply_to_column(query_id, refreshed)
            }
        }))
    }

    function onRegisteredKyou(raw: unknown): void {
        void insert_registered_kyou(raw).catch((err: unknown) => {
            console.error(err)
        })
    }

    return {
        onRegisteredKyou,
        insert_registered_kyou,
    }
}
