// 編集前に読む: .claude/skills/gkill-client-tags/SKILL.md（この領域の不変条件の正本）
'use strict'

// 複数書き込みの操作を tx_id で束ねて commit_tx で確定する理由と却下案:
// documents/adr/0410-bundle-multi-write-operations-in-tx.md

import type { GkillAPI } from '@/classes/api/gkill-api'
import type { GkillError } from '@/classes/api/gkill-error'
import type { Kyou } from '@/classes/datas/kyou'
import { CommitTXRequest } from '@/classes/api/req_res/commit-tx-request'
import { DiscardTXRequest } from '@/classes/api/req_res/discard-tx-request'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'

export interface RunInTxResult {
    // commit_tx まで通ったか。false のときは discard_tx 済みで、サーバには何も書かれていない
    committed: boolean
    errors: Array<GkillError>
}

/**
 * 複数の書き込みを1つの tx_id で束ねて確定する。
 *
 * `work` は渡された tx_id を各 add_* / update_* の `req.tx_id` に入れて temp rep へ積み、
 * 失敗したエラーを返す（空なら成功）。1本でも失敗すれば discard_tx して何も残さない。
 * 全部通れば commit_tx する。サーバの commit_tx は1つの SQLite トランザクションなので、
 * ここが失敗しても何も書かれていない（ERR000419）。その場合も temp を discard する。
 *
 * `work` の中で throw した場合も discard してから投げ直す（temp に積んだ行を残さない）。
 */
export async function run_in_tx(
    gkill_api: GkillAPI,
    work: (tx_id: string) => Promise<Array<GkillError>>,
): Promise<RunInTxResult> {
    const tx_id = gkill_api.generate_uuid()

    let work_errors: Array<GkillError>
    try {
        work_errors = await work(tx_id)
    } catch (err: unknown) {
        await discard_tx(gkill_api, tx_id)
        throw err
    }
    if (work_errors.length !== 0) {
        const discard_errors = await discard_tx(gkill_api, tx_id)
        return { committed: false, errors: work_errors.concat(discard_errors) }
    }

    const commit_req = new CommitTXRequest()
    commit_req.tx_id = tx_id
    const commit_res = await gkill_api.commit_tx(commit_req)
    const commit_errors = commit_res.errors ?? []
    if (commit_errors.length !== 0) {
        const discard_errors = await discard_tx(gkill_api, tx_id)
        return { committed: false, errors: commit_errors.concat(discard_errors) }
    }
    return { committed: true, errors: [] }
}

/**
 * temp rep に積んだ行を捨てる。失敗はエラーとして返すだけで投げない
 * （本命のエラーが別にあり、これは後始末）。
 */
export async function discard_tx(gkill_api: GkillAPI, tx_id: string): Promise<Array<GkillError>> {
    const req = new DiscardTXRequest()
    req.tx_id = tx_id
    try {
        const res = await gkill_api.discard_tx(req)
        return res.errors ?? []
    } catch (_err: unknown) {
        // 通信断などで捨てられなくても、temp はインメモリなので次回の起動で消える
        return []
    }
}

/**
 * commit 後の Kyou を引き直す。
 *
 * tx 中の add_* / update_* は応答に Kyou を載せられない（一時リポジトリにしか無い）ので、
 * commit を終えてから初めて実体が手に入る。commit より前にこの id を引いた応答を
 * ServiceWorker の POST キャッシュが掴んでいることがあるため、引く前にキャッシュを捨てる。
 * 引けなければ null（呼び出し元は従来どおり一覧全体の引き直しへ落とす）。
 */
export async function fetch_committed_kyou(gkill_api: GkillAPI, id: string): Promise<Kyou | null> {
    try {
        await delete_gkill_kyou_cache(id)
    } catch (_err: unknown) {
        // Cache API が使えない環境ではスキップ
    }
    const req = new GetKyouRequest()
    req.id = id
    const res = await gkill_api.get_kyou(req)
    if (res.errors && res.errors.length !== 0) {
        return null
    }
    return res.kyou_histories[0] ?? null
}
