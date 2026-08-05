import { describe, test, expect } from 'vitest'
import { hydrate, hydrate_all } from '@/classes/api/hydrate'

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
