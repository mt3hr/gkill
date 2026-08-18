/**
 * 画面間の変更通知バス。
 *
 * ポート(rudbeckia)は同じ画面や別の画面を並べて開けるので、片方で起きた
 * 追加・更新・削除をもう片方へ伝える必要がある。gkill には画面をまたぐ通知経路が
 * 無かったので新設した。ここで固定するのは以下:
 *
 * 1. **追記ログであること。** スカラー（最新の1件）だと同じ tick に複数件起きたとき
 *    最後の1件しか見えず、残りが黙って落ちる（KFTLの複数行保存が典型）。
 * 2. **自分が出した通知は受けないこと。** 受けると発生元が二重適用する。
 * 3. **`reload_list` は1回に畳むこと。** 畳まないと1回の保存で画面の枚数ぶん全件検索が走る。
 * 4. **後から開いたウィンドウが過去を再生しないこと。**
 */
import { describe, expect, test, vi } from 'vitest'

import {
    create_kyou_change_bus,
    KYOU_CHANGE_LOG_MAX_LENGTH,
    type KyouChangeNotice,
} from '@/classes/kyou-change-bus'
import { apply_notices, type KyouChangeSink } from '@/classes/use-kyou-change-subscriber'
import type { Kyou } from '@/classes/datas/kyou'

function make_kyou(id: string): Kyou {
    return { id: id } as unknown as Kyou
}

function make_sink(): KyouChangeSink & {
    registered: Array<string>
    reloaded: Array<string>
    deleted: Array<string>
    reload_list_count: number
} {
    const sink = {
        registered: new Array<string>(),
        reloaded: new Array<string>(),
        deleted: new Array<string>(),
        reload_list_count: 0,
        apply_registered: (kyou: Kyou) => { sink.registered.push(kyou.id) },
        apply_reload: (kyou: Kyou) => { sink.reloaded.push(kyou.id) },
        apply_deleted: (kyou: Kyou) => { sink.deleted.push(kyou.id) },
        apply_reload_list: () => { sink.reload_list_count++ },
    }
    return sink
}

describe('create_kyou_change_bus', () => {
    test('publish した順に seq が振られる', () => {
        const bus = create_kyou_change_bus()

        bus.publish('a', { kind: 'registered', kyou: make_kyou('k1') }, 1)
        bus.publish('a', { kind: 'reload', kyou: make_kyou('k2') }, 2)

        expect(bus.log.map((notice) => notice.seq)).toEqual([1, 2])
        expect(bus.last_seq()).toBe(2)
    })

    // スカラー通知だとここが1件しか届かない
    test('同じtickに複数件 publish しても全部残る', () => {
        const bus = create_kyou_change_bus()

        for (let i = 0; i < 5; i++) {
            bus.publish('a', { kind: 'registered', kyou: make_kyou(`k${i}`) }, i)
        }

        expect(bus.drain_from(0).length, '同じtickの通知が落ちている').toBe(5)
    })

    test('drain_from は自分のカーソルより後だけ返す', () => {
        const bus = create_kyou_change_bus()
        bus.publish('a', { kind: 'reload', kyou: make_kyou('k1') }, 1)
        bus.publish('a', { kind: 'reload', kyou: make_kyou('k2') }, 2)
        bus.publish('a', { kind: 'reload', kyou: make_kyou('k3') }, 3)

        const drained = bus.drain_from(1)

        expect(drained.map((notice) => notice.seq)).toEqual([2, 3])
    })

    test('上限を超えたら古いものから捨てる', () => {
        const bus = create_kyou_change_bus()

        for (let i = 0; i < KYOU_CHANGE_LOG_MAX_LENGTH + 10; i++) {
            bus.publish('a', { kind: 'reload', kyou: make_kyou(`k${i}`) }, i)
        }

        expect(bus.log.length).toBe(KYOU_CHANGE_LOG_MAX_LENGTH)
        expect(bus.log[0].seq, '古いものから捨てていない').toBe(11)
    })

    // requested_at を運ばないと kyou-reload.ts の合流が成立せず、
    // 同じ Kyou を画面の枚数ぶん取りに行く
    test('requested_at はそのまま運ばれる', () => {
        const bus = create_kyou_change_bus()
        bus.publish('a', { kind: 'reload', kyou: make_kyou('k1') }, 12345)

        expect(bus.log[0].requested_at).toBe(12345)
    })
})

describe('apply_notices', () => {
    function notice(seq: number, origin_id: string, change: KyouChangeNotice['change']): KyouChangeNotice {
        return { seq: seq, origin_id: origin_id, requested_at: seq, change: change }
    }

    test('種別ごとに対応する適用関数を呼ぶ', () => {
        const sink = make_sink()

        apply_notices('me', sink, [
            notice(1, 'other', { kind: 'registered', kyou: make_kyou('k1') }),
            notice(2, 'other', { kind: 'reload', kyou: make_kyou('k2') }),
            notice(3, 'other', { kind: 'deleted', kyou: make_kyou('k3') }),
        ])

        expect(sink.registered).toEqual(['k1'])
        expect(sink.reloaded).toEqual(['k2'])
        expect(sink.deleted).toEqual(['k3'])
    })

    // 受けると発生元が二重適用する。追加は insert_kyou_sorted の id 重複判定で
    // 救われるが、削除と引き直しは救われない
    test('自分が出した通知は受けない', () => {
        const sink = make_sink()

        apply_notices('me', sink, [
            notice(1, 'me', { kind: 'deleted', kyou: make_kyou('k1') }),
            notice(2, 'other', { kind: 'deleted', kyou: make_kyou('k2') }),
        ])

        expect(sink.deleted, '自分が出した削除まで適用している').toEqual(['k2'])
    })

    // 畳まないと、1回の KFTL 保存で開いている画面ぶんの全件検索が走る
    test('reload_list は1回に畳む', () => {
        const sink = make_sink()

        apply_notices('me', sink, [
            notice(1, 'other', { kind: 'reload_list' }),
            notice(2, 'other', { kind: 'reload_list' }),
            notice(3, 'other', { kind: 'reload_list' }),
        ])

        expect(sink.reload_list_count).toBe(1)
    })

    test('reload_list が混ざったら個別の適用はしない（どうせ全件取り直す）', () => {
        const sink = make_sink()

        apply_notices('me', sink, [
            notice(1, 'other', { kind: 'reload', kyou: make_kyou('k1') }),
            notice(2, 'other', { kind: 'reload_list' }),
        ])

        expect(sink.reload_list_count).toBe(1)
        expect(sink.reloaded, '全件取り直すのに個別の引き直しも投げている').toEqual([])
    })

    test('自分が出した reload_list では取り直さない', () => {
        const sink = make_sink()

        apply_notices('me', sink, [notice(1, 'me', { kind: 'reload_list' })])

        expect(sink.reload_list_count).toBe(0)
    })

    test('requested_at をそのまま適用関数へ渡す', () => {
        const apply_reload = vi.fn()
        const sink: KyouChangeSink = {
            apply_registered: vi.fn(),
            apply_reload: apply_reload,
            apply_deleted: vi.fn(),
            apply_reload_list: vi.fn(),
        }

        apply_notices('me', sink, [notice(7, 'other', { kind: 'reload', kyou: make_kyou('k1') })])

        expect(apply_reload).toHaveBeenCalledWith(expect.objectContaining({ id: 'k1' }), 7)
    })
})
