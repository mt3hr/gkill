'use strict'

// エラーの「次の一手」（ヒント文）を error_kind / reason から引く。
//
// サーバの error_message は操作単位（「メモ追加に失敗しました」）で、何をすれば直るかは書いていない。
// 代わりにサーバが機械語のトークンで「誰の問題か」(error_kind) と「何が起きたか」(reason) を付けて返すので、
// ここで i18n キーへ写す。reason のほうが具体的なので、あれば kind より優先する。
// 語彙の正本は Go 側（src/server/gkill/api/message/error_kind.go / error_reason.go）。
// ここに無いトークンが来たら kind の既定へ落ちる（未知の reason で何も出さないよりは、粗くても案内を出す）。

import { GkillErrorCodes } from './gkill_error'

// reason → i18n キー。Go の reasonTokens と1対1（増減したら両方を直す。verify_docs が件数を突き合わせる）
const reason_hint_message_ids: Readonly<Record<string, string>> = Object.freeze({
    write_rep_missing: 'ERROR_HINT_REASON_WRITE_REP_MISSING',
    storage_unavailable: 'ERROR_HINT_REASON_STORAGE_UNAVAILABLE',
    storage_corrupted: 'ERROR_HINT_REASON_STORAGE_CORRUPTED',
    storage_readonly: 'ERROR_HINT_REASON_STORAGE_READONLY',
    storage_full: 'ERROR_HINT_REASON_STORAGE_FULL',
    storage_io_error: 'ERROR_HINT_REASON_STORAGE_IO_ERROR',
    db_busy: 'ERROR_HINT_REASON_DB_BUSY',
    timeout: 'ERROR_HINT_REASON_TIMEOUT',
    canceled: 'ERROR_HINT_REASON_CANCELED',
    plugin_busy: 'ERROR_HINT_REASON_PLUGIN_BUSY',
    plugin_error: 'ERROR_HINT_REASON_PLUGIN_ERROR',
    external_fetch_failed: 'ERROR_HINT_REASON_EXTERNAL_FETCH_FAILED',
    local_only_access: 'ERROR_HINT_REASON_LOCAL_ONLY_ACCESS',
    tls_files_missing: 'ERROR_HINT_REASON_TLS_FILES_MISSING',
})

// error_kind → i18n キー。auth はヒントを出さない（check_auth がログイン画面へ飛ばすので読む間が無い）
const kind_hint_message_ids: Readonly<Record<string, string>> = Object.freeze({
    input: 'ERROR_HINT_KIND_INPUT',
    permission: 'ERROR_HINT_KIND_PERMISSION',
    not_found: 'ERROR_HINT_KIND_NOT_FOUND',
    conflict: 'ERROR_HINT_KIND_CONFLICT',
    too_large: 'ERROR_HINT_KIND_TOO_LARGE',
    rate_limit: 'ERROR_HINT_KIND_RATE_LIMIT',
    config: 'ERROR_HINT_KIND_CONFIG',
    server: 'ERROR_HINT_KIND_SERVER',
})

// クライアント生成（ERR9 帯）のうち、ヒントを持つもの。
// 入力検証のエラー（「タイトルが空です」等）は文言自体が次の一手なので載せない
const client_code_hint_message_ids: Readonly<Record<string, string>> = Object.freeze({
    [GkillErrorCodes.bad_response]: 'ERROR_HINT_BAD_RESPONSE',
    [GkillErrorCodes.unexpected_client_error]: 'ERROR_HINT_UNEXPECTED_CLIENT_ERROR',
})

export const reason_tokens: ReadonlyArray<string> = Object.freeze(Object.keys(reason_hint_message_ids))
export const error_kind_tokens: ReadonlyArray<string> = Object.freeze([...Object.keys(kind_hint_message_ids), 'auth'])

/**
 * ヒント文の i18n キーを返す。無ければ空文字。
 *
 * 優先順: reason → クライアント生成コード → error_kind。
 * error_kind が空（クライアントの入力検証など）ならヒントは無い。
 */
export function error_hint_message_id(error: { error_code?: string, error_kind?: string, reason?: string }): string {
    const reason = error.reason ?? ''
    if (reason !== '' && reason in reason_hint_message_ids) {
        return reason_hint_message_ids[reason]
    }
    const code = error.error_code ?? ''
    if (code !== '' && code in client_code_hint_message_ids) {
        return client_code_hint_message_ids[code]
    }
    const kind = error.error_kind ?? ''
    if (kind !== '' && kind in kind_hint_message_ids) {
        return kind_hint_message_ids[kind]
    }
    return ''
}
