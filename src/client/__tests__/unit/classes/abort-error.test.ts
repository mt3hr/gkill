/**
 * 中断（AbortController.abort()）の判定。
 *
 * この判定はもともと20箇所へ手書きで複製されていて、片方のブラウザの文言しか
 * 見ていない写しが混ざっていた。`abort-error.ts` へ集約し、
 * `convention-source-scan.test.ts` が「手書きが残っていないこと」を見張っている。
 * つまり **判定の中身が壊れたら20箇所へ同時に波及する**のに、
 * 集約先の実装そのものを実行するテストは無かった。ここで固定する。
 *
 * 誤判定の向きで症状が違う:
 *   - 中断を中断と見なせない → 画面を離れる/検索を差し替えるたびに
 *     利用者に無意味なエラーが出る（あるいは console がエラーで埋まる）
 *   - 中断でないものを中断と見なす → 本物の通信エラーが黙って消える
 */
import { describe, expect, test, vi, afterEach } from 'vitest'
import { is_abort_error, log_unless_aborted } from '@/classes/abort-error'

afterEach(() => {
    vi.restoreAllMocks()
})

describe('is_abort_error', () => {
    test('DOMException(AbortError) は中断', () => {
        expect(is_abort_error(new DOMException('aborted', 'AbortError'))).toBe(true)
    })

    // 判定はブラウザごとに文言が違うので**メッセージで見るしかない**。
    // 片方だけ見ている写しが実在したので、両方を固定する
    test('Chrome の文言は中断', () => {
        expect(is_abort_error(new Error('signal is aborted without reason'))).toBe(true)
    })

    test('Firefox の文言は中断', () => {
        expect(is_abort_error(new Error('user aborted a request'))).toBe(true)
    })

    test('文言が本文の途中にあっても中断', () => {
        expect(is_abort_error(new Error('failed to fetch: signal is aborted without reason'))).toBe(true)
    })

    test('AbortError 以外の DOMException は中断ではない', () => {
        expect(is_abort_error(new DOMException('boom', 'NotAllowedError'))).toBe(false)
    })

    test('ふつうの通信エラーは中断ではない', () => {
        expect(is_abort_error(new Error('Failed to fetch'))).toBe(false)
    })

    test.each([
        ['null', null],
        ['undefined', undefined],
        ['文字列', 'signal is aborted without reason'],
        ['オブジェクト', { message: 'signal is aborted without reason' }],
    ])('Error でないもの（%s）は中断ではない', (_label, value) => {
        // Error でないものを true にすると、本物の失敗が黙って消える
        expect(is_abort_error(value)).toBe(false)
    })
})

describe('log_unless_aborted', () => {
    test('中断は握りつぶす', () => {
        const spy = vi.spyOn(console, 'error').mockImplementation(() => { })
        log_unless_aborted(new DOMException('aborted', 'AbortError'))
        log_unless_aborted(new Error('user aborted a request'))
        expect(spy).not.toHaveBeenCalled()
    })

    test('中断でなければ console へ出す', () => {
        const spy = vi.spyOn(console, 'error').mockImplementation(() => { })
        const err = new Error('Failed to fetch')
        log_unless_aborted(err)
        expect(spy).toHaveBeenCalledTimes(1)
        expect(spy).toHaveBeenCalledWith(err)
    })
})
