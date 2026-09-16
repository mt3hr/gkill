import { describe, test, expect, vi, beforeEach } from 'vitest'

vi.mock('@/classes/use-gkill-message-feed', () => ({
    push_client_exception: vi.fn(),
}))

import { push_client_exception } from '@/classes/use-gkill-message-feed'
import {
    is_resize_observer_loop_notice,
    onUnhandledRejection,
    onVueError,
    onWindowError,
} from '@/classes/global-exception-feed'

// main.ts が window / Vue に登録する例外の配線。
// 出さないものを間違えると「検索を打ち切るたびにエラーが出る」「ResizeObserver の通知で
// エラーが積み上がる」になり、出すべきものを落とすと以前の「押しても何も起きない」に戻る。
// 中断の判定そのもの（ブラウザごとの文言）は abort-error.test.ts が守る。
const push_mock = vi.mocked(push_client_exception)

describe('global-exception-feed', () => {
    beforeEach(() => {
        push_mock.mockClear()
    })

    test('unhandledrejection: 中断は既定の出力ごと握りつぶし、それ以外はフィードへ', () => {
        const prevent = vi.fn()
        onUnhandledRejection({ reason: new DOMException('aborted', 'AbortError'), preventDefault: prevent })
        expect(prevent).toHaveBeenCalledTimes(1)
        expect(push_mock).not.toHaveBeenCalled()

        const failure = new TypeError('Failed to fetch')
        onUnhandledRejection({ reason: failure, preventDefault: prevent })
        expect(prevent).toHaveBeenCalledTimes(1)
        expect(push_mock).toHaveBeenCalledWith(failure)
    })

    test('window error: リソース読込失敗（error 無し）と ResizeObserver の通知は出さない', () => {
        expect(is_resize_observer_loop_notice('ResizeObserver loop completed with undelivered notifications.')).toBe(true)
        expect(is_resize_observer_loop_notice('Script error.')).toBe(false)
        expect(is_resize_observer_loop_notice(undefined)).toBe(false)

        onWindowError({ error: null, message: 'Script error.' })
        onWindowError({ error: new Error('loop'), message: 'ResizeObserver loop limit exceeded' })
        expect(push_mock).not.toHaveBeenCalled()

        const thrown = new Error('boom')
        onWindowError({ error: thrown, message: 'Uncaught Error: boom' })
        expect(push_mock).toHaveBeenCalledWith(thrown)
    })

    test('window error: 中断は出さない', () => {
        onWindowError({ error: new DOMException('aborted', 'AbortError'), message: 'Uncaught AbortError' })
        expect(push_mock).not.toHaveBeenCalled()
    })

    test('Vue の errorHandler は console に出し直してからフィードへ', () => {
        const console_error = vi.spyOn(console, 'error').mockImplementation(() => { })
        const thrown = new Error('render failed')
        onVueError(thrown, 'render function')
        expect(console_error).toHaveBeenCalledWith(thrown, 'render function')
        expect(push_mock).toHaveBeenCalledWith(thrown)
        console_error.mockRestore()
    })

    // KyouListView を速くスクロールすると v-virtual-scroll が KyouView を再利用し、props.kyou の
    // async watcher が飛行中の reload を自分で abort() する。その reject は unhandledrejection ではなく
    // Vue の errorHandler へ落ちる。ここで握らないと ERR900101 が右上に積み上がる（2026-09-16 に実際に出た）
    test('Vue の errorHandler: 中断は console にもフィードにも出さない', () => {
        const console_error = vi.spyOn(console, 'error').mockImplementation(() => { })
        onVueError(new DOMException('aborted', 'AbortError'), 'watcher callback')
        expect(console_error).not.toHaveBeenCalled()
        expect(push_mock).not.toHaveBeenCalled()
        console_error.mockRestore()
    })
})
