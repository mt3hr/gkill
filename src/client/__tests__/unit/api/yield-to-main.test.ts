import { describe, test, expect, vi, afterEach } from 'vitest'
import { yield_to_main } from '@/classes/api/yield-to-main'

interface SchedulerLike {
    yield?: () => Promise<void>
}

function set_scheduler(scheduler: SchedulerLike | undefined): void {
    (globalThis as { scheduler?: SchedulerLike }).scheduler = scheduler
}

describe('yield_to_main', () => {
    afterEach(() => {
        set_scheduler(undefined)
    })

    test('解決する', async () => {
        await expect(yield_to_main()).resolves.toBeUndefined()
    })

    test('scheduler.yield があればそれを使う', async () => {
        const yield_spy = vi.fn().mockResolvedValue(undefined)
        set_scheduler({ yield: yield_spy })

        await yield_to_main()

        expect(yield_spy).toHaveBeenCalledTimes(1)
    })

    test('scheduler が無い環境(Electron 22 / jsdom)では setTimeout に落ちる', async () => {
        const set_timeout_spy = vi.spyOn(globalThis, 'setTimeout')
        try {
            await yield_to_main()
            expect(set_timeout_spy).toHaveBeenCalled()
        } finally {
            set_timeout_spy.mockRestore()
        }
    })
})
