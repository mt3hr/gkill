'use strict'

import { i18n } from '@/i18n'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import type { GkillAPI } from '@/classes/api/gkill-api'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { Kyou } from '@/classes/datas/kyou'
import { DeleteKyouCascadeRequest } from '@/classes/api/req_res/delete-kyou-cascade-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'

export interface CascadeDeleteKyouOptions {
    kyou: Kyou
    gkill_api: GkillAPI
    application_config: ApplicationConfig
}

export interface CascadeDeleteKyouResult {
    // 画面から取り除くべきKyouのid。削除対象のKyou自身と、それを参照していたReKyou/MiReKyouのid
    deleted_ids: Array<string>
    errors: Array<GkillError>
}

// 画面から取り除く行になる種別（付随データのタグ・テキスト・通知は行ではない）
const deleted_row_data_types = new Set(["kmemo", "kc", "urlog", "nlog", "timeis", "mi", "mirekyou", "lantana", "idf", "rekyou"])

/**
 * Kyouと、それにくっついているTag/Text/Notification、それを参照しているReKyou/MiReKyouを
 * まとめて論理削除する。
 *
 * 探索と削除はサーバの1実装（/api/delete_kyou_cascade → usecase.DeleteKyouCascade）が行う。MCP の gkill_delete_kyou も
 * 同じ実装を使うので、画面と MCP で消える範囲が食い違わない（以前はこの TypeScript が探索と tx の組み立てを持っていて、
 * MCP は対象1件だけを消していた）。サーバは全件を1つのトランザクションで確定するので、**全部消えるか、何も消えないか**の
 * どちらかになる。
 * 経緯と却下案: documents/adr/0636-cascade-delete-on-the-server.md / documents/adr/0410-bundle-multi-write-operations-in-tx.md
 */
export async function cascade_delete_kyou(options: CascadeDeleteKyouOptions): Promise<CascadeDeleteKyouResult> {
    const { kyou, gkill_api, application_config } = options

    // 共有画面はdeviceもuser_idも空なので、削除自体を行わない
    if (application_config.for_share_kyou) {
        return { deleted_ids: [], errors: [] }
    }

    const req = new DeleteKyouCascadeRequest()
    req.target_id = kyou.id
    req.data_type = kyou.data_type
    req.update_app = "gkill"
    const res = await gkill_api.delete_kyou_cascade(req)
    // 成功時の errors が null で来る組み合わせ（古い SW キャッシュ・古いサーバ）でも展開しない
    const errors = res.errors ?? []
    if (errors.length !== 0) {
        // 1つのトランザクションなので、失敗したときは何も消えていない。行はすべて画面に残す
        return { deleted_ids: [], errors: errors }
    }

    const deleted = res.deleted ?? []
    // 消した全idのService Workerキャッシュを落とす（付随データの target_id 単位のキャッシュも含めて引き直させる）
    await Promise.all(deleted.map(record => delete_gkill_kyou_cache(record.id)))
    return {
        deleted_ids: deleted.filter(record => deleted_row_data_types.has(record.data_type)).map(record => record.id),
        errors: [],
    }
}

/**
 * 画面から取り除くためだけのKyou。消費側は id しか見ないので中身は詰めない。
 */
export function build_deleted_kyou_stub(id: string): Kyou {
    const kyou = new Kyou()
    kyou.id = id
    kyou.is_deleted = true
    return kyou
}

/**
 * 連鎖削除が想定外に失敗したときのエラー。呼び出し元の catch でも使う。
 * 削除は1つのトランザクションなので、失敗したときは何も消えていない。文言もそう言う。
 */
export function build_cascade_delete_failed_error(): GkillError {
    const error = new GkillError()
    error.error_code = GkillErrorCodes.cascade_delete_failed
    error.error_message = i18n.global.t('FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE')
    return error
}
