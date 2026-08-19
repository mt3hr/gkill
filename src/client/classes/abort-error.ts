'use strict'

/**
 * `AbortController.abort()` による中断かどうか。
 *
 * 中断は「利用者が画面を離れた」「後発の検索に差し替えられた」等の正常な流れで起きるので、
 * ログにもエラー表示にも出さない。判定はブラウザごとに文言が違うので**メッセージで見るしかない**
 * （Chrome: "signal is aborted without reason" / Firefox: "user aborted a request"）。
 *
 * この判定は 20 箇所へ手書きで複製されていて、片方の文言しか見ていない写しも混ざっていた。
 * 増やすときはここへ足すこと。
 */
export function is_abort_error(err: unknown): boolean {
    if (err instanceof DOMException && err.name === 'AbortError') {
        return true
    }
    if (!(err instanceof Error)) {
        return false
    }
    return err.message.includes('signal is aborted without reason')
        || err.message.includes('user aborted a request')
}

/** 中断でなければ console へ出す。中断は握りつぶす */
export function log_unless_aborted(err: unknown): void {
    if (is_abort_error(err)) {
        return
    }
    console.error(err)
}
