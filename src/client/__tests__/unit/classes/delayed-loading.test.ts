/**
 * useDelayedLoading のテスト。
 * 一覧では数十行が同時に読み込み中になるので、速く終わった読み込みで
 * インジケータが明滅しないことを担保する。
 */
import { describe, test, expect, vi, afterEach } from 'vitest'
import { ref } from 'vue'
import { LOADING_INDICATOR_DELAY_MS, useDelayedLoading } from '@/classes/use-delayed-loading'

afterEach(() => {
    vi.useRealTimers()
})

describe('useDelayedLoading', () => {
    test('しきい値に届かないうちは立たない', async () => {
        vi.useFakeTimers()
        const is_loading = ref(true)
        const is_shown = useDelayedLoading(is_loading)

        await vi.advanceTimersByTimeAsync(LOADING_INDICATOR_DELAY_MS - 1)

        expect(is_shown.value).toBe(false)
    })

    test('しきい値を超えたら立つ', async () => {
        vi.useFakeTimers()
        const is_loading = ref(true)
        const is_shown = useDelayedLoading(is_loading)

        await vi.advanceTimersByTimeAsync(LOADING_INDICATOR_DELAY_MS)

        expect(is_shown.value).toBe(true)
    })

    test('しきい値に届く前に読み込みが終われば立たない', async () => {
        vi.useFakeTimers()
        const is_loading = ref(true)
        const is_shown = useDelayedLoading(is_loading)

        await vi.advanceTimersByTimeAsync(LOADING_INDICATOR_DELAY_MS - 1)
        is_loading.value = false
        await vi.advanceTimersByTimeAsync(1000)

        expect(is_shown.value).toBe(false)
    })

    test('読み込みが終われば下りる', async () => {
        vi.useFakeTimers()
        const is_loading = ref(true)
        const is_shown = useDelayedLoading(is_loading)

        await vi.advanceTimersByTimeAsync(LOADING_INDICATOR_DELAY_MS)
        expect(is_shown.value).toBe(true)

        is_loading.value = false
        await vi.advanceTimersByTimeAsync(0)

        expect(is_shown.value).toBe(false)
    })

    test('最初から読み込み中でなければ立たない', async () => {
        vi.useFakeTimers()
        const is_loading = ref(false)
        const is_shown = useDelayedLoading(is_loading)

        await vi.advanceTimersByTimeAsync(1000)

        expect(is_shown.value).toBe(false)
    })

    test('待ち時間は指定できる', async () => {
        vi.useFakeTimers()
        const is_loading = ref(true)
        const is_shown = useDelayedLoading(is_loading, 1000)

        await vi.advanceTimersByTimeAsync(999)
        expect(is_shown.value).toBe(false)

        await vi.advanceTimersByTimeAsync(1)
        expect(is_shown.value).toBe(true)
    })
})
