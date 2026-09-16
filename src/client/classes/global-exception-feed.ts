'use strict'

// 握られなかった例外を画面右上のフィード（use-gkill-message-feed.ts）へ流す配線。
// main.ts が window / Vue アプリに登録する。以前はコンソールにしか出ず、
// 利用者には「押しても何も起きない」にしか見えなかった。
//
// main.ts はアプリを mount する副作用を持つのでテストから import できない。
// 判定と配線をここへ切り出し、単体テスト（global-exception-feed.test.ts）で固定する。
// 中断（AbortController）の判定は classes/abort-error.ts の1実装だけを使う（手書きしない）。

import { is_abort_error } from '@/classes/abort-error'
import { push_client_exception } from '@/classes/use-gkill-message-feed'

// ResizeObserver の「loop completed with undelivered notifications」はブラウザが出す無害な通知
export function is_resize_observer_loop_notice(message: unknown): boolean {
    return typeof message === "string" && message.includes("ResizeObserver loop")
}

// unhandledrejection: 検索の打ち切り（abort）は利用者の操作なので既定の console 出力ごと握りつぶし、
// それ以外はフィードへ
export function onUnhandledRejection(event: { reason: unknown, preventDefault: () => void }): void {
    if (is_abort_error(event.reason)) {
        event.preventDefault()
        return
    }
    push_client_exception(event.reason)
}

// window の error: リソース読込失敗（error が無い）と ResizeObserver の通知は出さない
export function onWindowError(event: { error: unknown, message: unknown }): void {
    if (!event.error || is_resize_observer_loop_notice(event.message)) {
        return
    }
    push_client_exception(event.error)
}

// Vue の errorHandler: 置くと Vue 自身の console 出力が止まるので、ここで出し直してからフィードへ
export function onVueError(err: unknown, info: string): void {
    console.error(err, info)
    push_client_exception(err)
}
