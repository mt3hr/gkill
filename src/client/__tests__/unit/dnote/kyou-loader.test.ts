import { describe, expect, test } from 'vitest'
import load_kyous from '@/classes/dnote/kyou-loader'
import type { Kyou } from '@/classes/datas/kyou'
import type DnotePredicate from '@/classes/dnote/dnote-predicate'

/**
 * load_kyous の並列度と早期打ち切りの回帰テスト。
 *
 * 1件ずつ await していたころは 1,000件 × RTT20ms で約20秒かかっていた。
 * 並列化しても結果の中身は同じなので、同時実行数を数えないと
 * 直列に戻ったことに気づけない。
 *
 * 一方、limit付き(ryuuの前後1件探し)は「先頭から順に見て条件に合う件数が
 * limitに達したら止める」意味なので、先回りして読んではいけない。
 */

type Tracker = { inflight: number, max_inflight: number, prepared: Array<string> }

function makeKyou(id: string, tracker: Tracker): Kyou {
    const load = async () => {
        tracker.inflight++
        tracker.max_inflight = Math.max(tracker.max_inflight, tracker.inflight)
        await new Promise(resolve => setTimeout(resolve, 1))
        tracker.inflight--
        return []
    }
    const kyou = {
        id: id,
        abort_controller: null as unknown as AbortController,
        clone() {
            tracker.prepared.push(id)
            return kyou
        },
        reload: async () => [],
        load_typed_datas: load,
        load_attached_tags: load,
        load_attached_texts: load,
    }
    return kyou as unknown as Kyou
}

describe('load_kyous', () => {
    test('limit指定なしなら複数件を並列で読む', async () => {
        const tracker: Tracker = { inflight: 0, max_inflight: 0, prepared: [] }
        const kyous = Array.from({ length: 24 }, (_, i) => makeKyou(`kyou-${i}`, tracker))

        const result = await load_kyous(new AbortController(), kyous, false, true)

        expect(result).toHaveLength(24)
        if (tracker.max_inflight <= 3) {
            throw new Error(`直列に戻っている (同時実行の最大 = ${tracker.max_inflight})。件数×RTTかかる`)
        }
    })

    test('並列でも順序は保たれる', async () => {
        const tracker: Tracker = { inflight: 0, max_inflight: 0, prepared: [] }
        const kyous = Array.from({ length: 20 }, (_, i) => makeKyou(`kyou-${i}`, tracker))

        const result = await load_kyous(new AbortController(), kyous, false, true)

        expect(result.map(k => k.id)).toEqual(kyous.map(k => k.id))
    })

    // limit付きは先回りして読まない。読むと余計な取得が発生する。
    test('limit指定ありなら条件に合った時点で止まり、先の件は読まない', async () => {
        const tracker: Tracker = { inflight: 0, max_inflight: 0, prepared: [] }
        const kyous = Array.from({ length: 20 }, (_, i) => makeKyou(`kyou-${i}`, tracker))
        const target = makeKyou('target', tracker)
        // 先頭で必ずマッチする述語
        const predicate = { is_match: async () => true } as unknown as DnotePredicate

        const result = await load_kyous(new AbortController(), kyous, false, true, predicate, target, 1)

        expect(result).toHaveLength(1)
        if (tracker.prepared.length !== 1) {
            throw new Error(`打ち切り後の件まで読んでいる (読んだ件数 = ${tracker.prepared.length})`)
        }
    })
})
