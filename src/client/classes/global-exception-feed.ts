'use strict'

// 握られなかった例外を画面右上のフィード（use-gkill-message-feed.ts）へ流す配線。
// main.ts が window / Vue アプリに登録する。以前はコンソールにしか出ず、
// 利用者には「押しても何も起きない」にしか見えなかった。
//
// main.ts はアプリを mount する副作用を持つのでテストから import できない。
// 判定と配線をここへ切り出し、単体テスト（global-exception-feed.test.ts）で固定する。
// 中断（AbortController）の判定は classes/abort-error.ts の1実装だけを使う（手書きしない）。
//
// 中断は3つの入口すべてで出さない。unhandledrejection だけで握っていたころ、KyouListView を
// 速くスクロールすると ERR900101（本文は AbortError）が右上に積み上がった。
// v-virtual-scroll が KyouView を再利用して props.kyou が差し替わるたびに、飛行中の reload を
// 自分で abort() している。その async watcher の reject は unhandledrejection ではなく
// Vue の errorHandler（callWithAsyncErrorHandling）へ落ちるので、そこで見ていないと画面に出る。
// 中断は「利用者が画面を離れた・後発に差し替えられた」の正常な流れなので、どの入口から来ても出さない。

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

// window の error: リソース読込失敗（error が無い）・ResizeObserver の通知・中断は出さない
export function onWindowError(event: { error: unknown, message: unknown }): void {
    if (!event.error || is_resize_observer_loop_notice(event.message) || is_abort_error(event.error)) {
        return
    }
    push_client_exception(event.error)
}

// Vue の errorHandler: 置くと Vue 自身の console 出力が止まるので、ここで出し直してからフィードへ。
// 中断（async な watcher / lifecycle hook の reject がここへ来る）は console にも出さない
export function onVueError(err: unknown, info: string): void {
    if (is_abort_error(err)) {
        return
    }
    console.error(err, info)
    push_client_exception(err)
}
