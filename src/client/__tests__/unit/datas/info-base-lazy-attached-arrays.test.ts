/**
 * InfoBase / Kyou の attached_* 配列の遅延確保の検証。
 *
 * 検索応答(get_kyous)には attached_* が1つも含まれないので、
 * 30万件の実体化ではこれらの配列は一度も書かれない。
 * それでもコンストラクタで確保すると1件につき5本、30万件で150万個の使い捨てになる。
 *
 * `_abort_controller` と同じく underscore 公開フィールド + ゲッターにしてある。
 * TS private は Vue の UnwrapRef が落として `Ref<Array<Kyou>>` への代入が型エラーになり、
 * ES private(#) は reactive Proxy 越しの this で壊れるので、どちらも使えない。
 */
import { describe, test, expect } from 'vitest'
import { reactive } from 'vue'
import '@/classes/api/gkill-api'
import { Kyou } from '@/classes/datas/kyou'
import { Tag } from '@/classes/datas/tag'

const lazy_fields = [
    '_attached_tags',
    '_attached_texts',
    '_attached_notifications',
    '_attached_timeis_kyou',
    '_attached_histories',
] as const

describe('InfoBase / Kyou の attached_* 遅延確保', () => {
    test('生成直後はどれも確保されていない', () => {
        const kyou = new Kyou() as unknown as Record<string, unknown>
        for (const field of lazy_fields) {
            expect(kyou[field], field + ' がコンストラクタで確保されている').toBeNull()
        }
    })

    test('getterの初回アクセスで確保され、2回目以降は同一インスタンスを返す', () => {
        const kyou = new Kyou()
        const tags = kyou.attached_tags
        expect(Array.isArray(tags)).toBe(true)
        expect(kyou.attached_tags).toBe(tags)
        expect(kyou.attached_texts).toBe(kyou.attached_texts)
        expect(kyou.attached_notifications).toBe(kyou.attached_notifications)
        expect(kyou.attached_timeis_kyou).toBe(kyou.attached_timeis_kyou)
        expect(kyou.attached_histories).toBe(kyou.attached_histories)
    })

    test('setterで差し替えられる', () => {
        const kyou = new Kyou()
        const tag = new Tag()
        kyou.attached_tags = [tag]
        expect(kyou.attached_tags).toEqual([tag])
    })

    // KyouはVueのreactive経由で使われる
    test('Vueのreactive proxy越しでもgetter/setterが動く', () => {
        const kyou = reactive(new Kyou())
        expect(Array.isArray(kyou.attached_tags)).toBe(true)
        kyou.attached_tags = []
        expect(kyou.attached_tags).toEqual([])
    })

    // clone がゲッター越しに読むと、未確保のものまで確保してしまう
    test('cloneは未確保の配列を確保しない', () => {
        const kyou = new Kyou()
        const cloned = kyou.clone() as unknown as Record<string, unknown>
        for (const field of lazy_fields) {
            if (field === '_attached_histories') {
                // attached_histories は clone がコピーしないので、生成直後と同じくnull
                expect(cloned[field]).toBeNull()
                continue
            }
            expect(cloned[field], field + ' が clone で確保されている').toBeNull()
        }
    })

    test('cloneは確保済みの配列を別インスタンスとして写す', () => {
        const kyou = new Kyou()
        const tag = new Tag()
        kyou.attached_tags = [tag]
        const cloned = kyou.clone() as Kyou
        expect(cloned.attached_tags).toEqual([tag])
        expect(cloned.attached_tags).not.toBe(kyou.attached_tags)
    })
})
