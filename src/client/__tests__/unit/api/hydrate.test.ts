import { describe, test, expect, vi } from 'vitest'
import { hydrate, hydrate_all, hydrate_all_chunked } from '@/classes/api/hydrate'

class Sample {
    id = ''
    related_time: Date | null = null
    create_time: Date | null = null
    count = 0
    greet(): string {
        return 'hi ' + this.id
    }
}

describe('hydrate', () => {
    test('コピー後もクラスのメソッドが使える', () => {
        const result = hydrate(new Sample(), { id: 'a' })
        expect(result.greet()).toBe('hi a')
    })

    test('末尾が time のキーだけ Date に変換する', () => {
        const result = hydrate(new Sample(), {
            id: 'a',
            related_time: '2026-08-05T10:00:00.000Z',
            count: 3,
        })
        expect(result.related_time).toBeInstanceOf(Date)
        expect((result.related_time as Date).toISOString()).toBe('2026-08-05T10:00:00.000Z')
        expect(result.count).toBe(3)
    })

    test('null や空文字の時刻は変換しない', () => {
        const result = hydrate(new Sample(), { related_time: null, create_time: '' })
        expect(result.related_time).toBeNull()
        expect(result.create_time).toBe('')
    })

    test('date_suffixes を空にすると変換しない', () => {
        const result = hydrate(new Sample(), { related_time: '2026-08-05T10:00:00.000Z' }, { date_suffixes: [] })
        expect(result.related_time).not.toBeInstanceOf(Date)
    })

    test('date_suffixes を増やすとその接尾辞も変換する', () => {
        const result = hydrate(new Sample(), { some_date: '2026-08-05T10:00:00.000Z' } as never,
            { date_suffixes: ['time', 'date'] })
        expect((result as unknown as Record<string, unknown>).some_date).toBeInstanceOf(Date)
    })

    test('型に無いキーも写す(サーバが増やしたフィールドを捨てない)', () => {
        const result = hydrate(new Sample(), { unknown_field: 'x' })
        expect((result as unknown as Record<string, unknown>).unknown_field).toBe('x')
    })

    test('source が null / 非オブジェクトなら target をそのまま返す', () => {
        const target = new Sample()
        expect(hydrate(target, null)).toBe(target)
        expect(hydrate(target, 'string')).toBe(target)
    })
})

describe('hydrate_all', () => {
    test('配列の各要素をインスタンス化する', () => {
        const result = hydrate_all([{ id: 'a' }, { id: 'b' }], () => new Sample())
        expect(result.length).toBe(2)
        expect(result[0].greet()).toBe('hi a')
        expect(result[1].greet()).toBe('hi b')
    })

    test('配列でなければ空配列', () => {
        expect(hydrate_all(null, () => new Sample())).toEqual([])
        expect(hydrate_all({ id: 'a' }, () => new Sample())).toEqual([])
    })
})

describe('hydrate_all_chunked', () => {
    test('全要素がfactoryインスタンスへin-placeで置換され、日付も変換される', async () => {
        const list: Array<unknown> = [
            { id: 'a', related_time: '2026-08-05T10:00:00.000Z' },
            { id: 'b', related_time: '2026-08-06T10:00:00.000Z' },
        ]
        await hydrate_all_chunked(list, () => new Sample())
        expect(list[0]).toBeInstanceOf(Sample)
        expect((list[0] as Sample).greet()).toBe('hi a')
        expect((list[0] as Sample).related_time).toBeInstanceOf(Date)
        expect((list[1] as Sample).greet()).toBe('hi b')
    })

    test('1チャンクで終わる配列にはyield(setTimeout)を挟まない', async () => {
        const set_timeout_spy = vi.spyOn(globalThis, 'setTimeout')
        try {
            const list: Array<unknown> = [{ id: 'a' }, { id: 'b' }]
            await hydrate_all_chunked(list, () => new Sample(), { chunk_size: 5 })
            expect(set_timeout_spy).not.toHaveBeenCalled()
        } finally {
            set_timeout_spy.mockRestore()
        }
    })

    test('複数チャンクではチャンク間ごとにyieldが入る', async () => {
        const set_timeout_spy = vi.spyOn(globalThis, 'setTimeout')
        try {
            const list: Array<unknown> = Array.from({ length: 5 }, (_, i) => ({ id: String(i) }))
            await hydrate_all_chunked(list, () => new Sample(), { chunk_size: 2 })
            // チャンクは[0,1][2,3][4]の3つ、間のyieldは2回
            expect(set_timeout_spy).toHaveBeenCalledTimes(2)
            expect(list.every((element) => element instanceof Sample)).toBe(true)
        } finally {
            set_timeout_spy.mockRestore()
        }
    })

    test('中断済みsignalではsignal.reason(AbortError)がthrowされ、何も実体化しない', async () => {
        const abort_controller = new AbortController()
        abort_controller.abort()
        const list: Array<unknown> = [{ id: 'a' }]
        await expect(hydrate_all_chunked(list, () => new Sample(), { signal: abort_controller.signal }))
            .rejects.toMatchObject({ name: 'AbortError' })
        expect(list[0]).not.toBeInstanceOf(Sample)
    })

    test('処理途中で中断されたら次のチャンク境界で止まる', async () => {
        const abort_controller = new AbortController()
        const list: Array<unknown> = Array.from({ length: 5 }, (_, i) => ({ id: String(i) }))
        const promise = hydrate_all_chunked(list, () => new Sample(), { chunk_size: 2, signal: abort_controller.signal })
        // 最初のチャンクは同期実行済み。次のチャンクへ進む前に中断する
        abort_controller.abort()
        await expect(promise).rejects.toMatchObject({ name: 'AbortError' })
        expect(list[0]).toBeInstanceOf(Sample)
        expect(list[1]).toBeInstanceOf(Sample)
        expect(list[2]).not.toBeInstanceOf(Sample)
    })

    test('date_suffixesが伝播する', async () => {
        const list: Array<unknown> = [{ some_date: '2026-08-05T10:00:00.000Z' }]
        await hydrate_all_chunked(list, () => new Sample(), { date_suffixes: ['date'] })
        expect((list[0] as unknown as Record<string, unknown>).some_date).toBeInstanceOf(Date)
    })
})
