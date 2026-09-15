import { ref, type Ref } from 'vue'
import { i18n } from '@/i18n'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { error_hint_message_id } from '@/classes/api/message/error-hints'

// 画面右上に出るエラー / メッセージの一覧（フィード）。
//
// 2026-09-15 まで、この一覧は use-*-page.ts の16本と *-page.vue の15本に同じ実装がコピーされていた。
// しかもサーバ由来のエラーは JSON に show_keep が無い → undefined → 「閉じられない・2.5秒で消える」で、
// クライアントの入力検証（show_keep=true で常駐）より**本物の障害のほうが早く消える**逆転が起きていた。
// エラーコードはホバーのツールチップだけで、スマホでは見えなかった。
//
// ここに1つだけ置く理由:
//   - エラーは閉じるまで残す・コピーできる・ヒント（何をすれば直るか）を出す、を1箇所で守る
//   - main.ts の unhandledrejection / errorHandler（コンポーネントの外）からも同じ表示へ流せる
// モジュール単位のシングルトンで、GkillAPI と同じ流儀（Pinia は入れない = ADR-0408）。
// 経緯と却下案は documents/adr/0411-error-feed-stays-until-closed.md。

export type GkillFeedLevel = 'info' | 'warning' | 'error'

export interface GkillFeedItem {
    id: string
    level: GkillFeedLevel
    // エラーコード（ERR...）またはメッセージコード（MSG...）
    code: string
    message: string
    // 次の一手。無ければ空
    hint: string
    // サーバの分類（何が起きたか）。無ければ空
    reason: string
    // サーバの分類（誰の問題か）。無ければ空
    kind: string
    closable: boolean
    // 同じ code+message が続けて届いた回数（オフライン中の連打で画面が埋まらないように1枚にまとめる）
    count: number
    received_at: Date
}

// info（成功・完了の知らせ）だけ自動で消す。エラーと warning は閉じるまで残す
export const info_auto_close_milli_seconds = 2500

const feed_items: Ref<Array<GkillFeedItem>> = ref([])
let next_id = 0

function generate_feed_id(): string {
    next_id++
    return `feed-${Date.now()}-${next_id}`
}

function translate(message_id: string): string {
    if (message_id === '') {
        return ''
    }
    return i18n.global.t(message_id)
}

// 同じ code+message+level のカードが既に出ていれば、それを返す（新しいカードを増やさない）
function find_same_item(level: GkillFeedLevel, code: string, message: string): GkillFeedItem | null {
    for (const item of feed_items.value) {
        if (item.level === level && item.code === code && item.message === message) {
            return item
        }
    }
    return null
}

function push_item(item: Omit<GkillFeedItem, 'id' | 'count' | 'received_at'>): GkillFeedItem {
    // info は数秒で消えるのでまとめない（まとめると「2件保存した」が1枚に見える）
    if (item.level !== 'info') {
        const same = find_same_item(item.level, item.code, item.message)
        if (same) {
            same.count++
            same.received_at = new Date()
            // ヒントや reason は新しいほうで上書き（同じコードでも reason が付いた版が後から来ることがある）
            if (item.hint !== '') {
                same.hint = item.hint
                same.reason = item.reason
                same.kind = item.kind
            }
            return same
        }
    }
    const pushed: GkillFeedItem = { ...item, id: generate_feed_id(), count: 1, received_at: new Date() }
    feed_items.value.push(pushed)
    if (item.level === 'info' && !item.closable) {
        setTimeout(() => close_feed_item(pushed.id), info_auto_close_milli_seconds)
    }
    return pushed
}

/** サーバ / クライアントの GkillError を表示する。null や空配列は何もしない */
export function push_errors(errors: Array<GkillError> | null | undefined): void {
    for (const error of errors ?? []) {
        if (!error || !error.error_message) {
            continue
        }
        // 呼び出し側の中断（reason=canceled）は利用者の操作の結果なので出さない
        if (error.reason === 'canceled') {
            continue
        }
        push_item({
            level: 'error',
            code: error.error_code ?? '',
            message: error.error_message,
            hint: translate(error_hint_message_id(error)),
            reason: error.reason ?? '',
            kind: error.error_kind ?? '',
            closable: true,
        })
    }
}

/** サーバ / クライアントの GkillMessage を表示する。level が warning なら閉じるまで残す */
export function push_messages(messages: Array<GkillMessage> | null | undefined): void {
    for (const message of messages ?? []) {
        if (!message || !message.message) {
            continue
        }
        const is_warning = message.level === 'warning'
        push_item({
            level: is_warning ? 'warning' : 'info',
            code: message.message_code ?? '',
            message: message.message,
            hint: '',
            reason: '',
            kind: '',
            closable: is_warning || Boolean(message.show_keep),
        })
    }
}

/**
 * 握られなかった例外（unhandledrejection / window.onerror / app.config.errorHandler）を表示する。
 * 以前はコンソールに出るだけで、利用者には「押しても何も起きない」にしか見えなかった。
 */
export function push_client_exception(err: unknown): void {
    const detail = err instanceof Error ? `${err.name}: ${err.message}` : String(err)
    push_item({
        level: 'error',
        code: GkillErrorCodes.unexpected_client_error,
        message: `${translate('UNEXPECTED_CLIENT_ERROR_MESSAGE')} (${detail})`,
        hint: translate('ERROR_HINT_UNEXPECTED_CLIENT_ERROR'),
        reason: '',
        kind: '',
        closable: true,
    })
}

export function close_feed_item(id: string): void {
    const index = feed_items.value.findIndex((item) => item.id === id)
    if (index !== -1) {
        feed_items.value.splice(index, 1)
    }
}

/** バグ報告用にカード1枚をテキストにする（コード・reason・本文・ヒント・時刻・画面パス） */
export function format_feed_item_for_copy(item: GkillFeedItem, pathname: string): string {
    const lines = new Array<string>()
    lines.push(item.reason !== '' ? `${item.code} ${item.reason}` : item.code)
    if (item.kind !== '') {
        lines.push(`kind: ${item.kind}`)
    }
    lines.push(item.message)
    if (item.hint !== '') {
        lines.push(item.hint)
    }
    if (item.count > 1) {
        lines.push(`x${item.count}`)
    }
    lines.push(item.received_at.toISOString())
    lines.push(pathname)
    return lines.join('\n')
}

/** テスト用。フィードを空にする */
export function reset_feed_items(): void {
    feed_items.value.splice(0, feed_items.value.length)
}

export function useGkillMessageFeed() {
    return {
        feed_items,
        push_errors,
        push_messages,
        push_client_exception,
        close_feed_item,
        format_feed_item_for_copy,
    }
}
