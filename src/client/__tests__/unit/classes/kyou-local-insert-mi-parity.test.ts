/**
 * mi板の絞り込み・射影・並び順が、サーバ側と同じ答えを出すことの対テスト。
 *
 * 対になるGoテスト: src/server/gkill/api/find_filter_mi_test.go
 * 片方だけ直すと「一覧はサーバ順、追加した行だけクライアント順」で並びが割れるので、
 * どちらかを触ったら必ず両方を直すこと。テスト名にGoの関数名を入れてあるので
 * どちら側からでも grep で相方を見つけられる。
 */
import { describe, expect, it } from 'vitest'

import { apply_mi_projection, compare_kyou_for_query, does_kyou_match_query } from '@/classes/kyou-local-insert'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { MiCheckState } from '@/classes/api/find_query/mi-check-state'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import type { Kyou } from '@/classes/datas/kyou'
import type { Mi } from '@/classes/datas/mi'

function make_mi_kyou(id: string, data_type: string, related_time: Date, typed_mi: Partial<Mi>): Kyou {
    return {
        id: id,
        data_type: data_type,
        rep_name: 'mi_rep',
        related_time: related_time,
        create_time: typed_mi.create_time ?? related_time,
        update_time: typed_mi.update_time ?? related_time,
        is_deleted: false,
        attached_tags: [],
        typed_mi: typed_mi,
        typed_mirekyou: null,
    } as unknown as Kyou
}

function make_mi_query(overrides?: Partial<FindKyouQuery>): FindKyouQuery {
    const query = new FindKyouQuery()
    query.tags = null
    query.reps = null
    query.for_mi = true
    Object.assign(query, overrides)
    return query
}

// Go: TestFilterMiForMi_EmptyCheckStateTreatsAsAll
describe('TestFilterMiForMi_EmptyCheckStateTreatsAsAll', () => {
    it('mi_check_state 未指定(空文字)はAll扱いで残る', () => {
        const base_time = new Date(2026, 7, 1, 12, 0, 0)
        const query = make_mi_query({
            mi_check_state: '' as MiCheckState,
            include_create_mi: true,
            include_check_mi: true,
            include_limit_mi: true,
            include_start_mi: true,
            include_end_mi: true,
        })
        const checked = make_mi_kyou('mi-1', 'mi_create', base_time, {
            board_name: 'default', is_checked: true, create_time: base_time, update_time: base_time,
            limit_time: null, estimate_start_time: null, estimate_end_time: null,
        })
        const unchecked = make_mi_kyou('mi-2', 'mi_create', base_time, {
            board_name: 'default', is_checked: false, create_time: base_time, update_time: base_time,
            limit_time: null, estimate_start_time: null, estimate_end_time: null,
        })
        expect(does_kyou_match_query(checked, query)).toBe(true)
        expect(does_kyou_match_query(unchecked, query)).toBe(true)
    })
})

// Go: TestOverrideKyous_FallbackUsesCreateTime
describe('TestOverrideKyous_FallbackUsesCreateTime', () => {
    it('ソート基準の時刻が未設定なら _create を名乗り作成日時を表示時刻にする', () => {
        const create_time = new Date(2026, 7, 1, 9, 0, 0)
        const update_time = new Date(2026, 7, 5, 21, 0, 0)
        const kyou = make_mi_kyou('mi-1', 'mi', update_time, {
            board_name: 'default', is_checked: false,
            create_time: create_time, update_time: update_time,
            limit_time: null, estimate_start_time: null, estimate_end_time: null,
        })

        apply_mi_projection(kyou, MiSortType.estimate_start_time)

        expect(kyou.data_type).toBe('mi_create')
        // update_time(8/5) ではなく create_time(8/1) が入ること
        expect(kyou.related_time).toEqual(create_time)
    })
})

// Go: TestSortResultKyous_MiCreateTimeTieBreaksByID
describe('TestSortResultKyous_MiCreateTimeTieBreaksByID', () => {
    it('CreateTime同着はID昇順で決定化される', () => {
        const same_time = new Date(2026, 7, 1, 12, 0, 0)
        const query = make_mi_query({ mi_sort_type: MiSortType.create_time })
        const typed_mi = {
            board_name: 'default', is_checked: false,
            create_time: same_time, update_time: same_time,
            limit_time: null, estimate_start_time: null, estimate_end_time: null,
        }
        const kyous = [
            make_mi_kyou('mi-c', 'mi_create', same_time, typed_mi),
            make_mi_kyou('mi-a', 'mi_create', same_time, typed_mi),
            make_mi_kyou('mi-b', 'mi_create', same_time, typed_mi),
        ]

        kyous.sort((a, b) => compare_kyou_for_query(a, b, query))

        expect(kyous.map(kyou => kyou.id)).toEqual(['mi-a', 'mi-b', 'mi-c'])
    })
})

// Go 側にケースは無いが sortResultKyous:1445-1456 が定める仕様。
// 「指定日時がないものは、末尾に作成日時でくっつける」
describe('sortResultKyous - 未設定は末尾', () => {
    it('ソート基準の時刻が無いMiは、作成日時が古くても末尾へ回る', () => {
        const query = make_mi_query({ mi_sort_type: MiSortType.estimate_start_time })
        const old_create_time = new Date(2020, 0, 1, 0, 0, 0)
        const start_time = new Date(2026, 7, 10, 10, 0, 0)
        const typed_undated = {
            board_name: 'default', is_checked: false,
            create_time: old_create_time, update_time: old_create_time,
            limit_time: null, estimate_start_time: null, estimate_end_time: null,
        }
        const typed_dated = {
            board_name: 'default', is_checked: false,
            create_time: new Date(2026, 7, 9, 0, 0, 0), update_time: new Date(2026, 7, 9, 0, 0, 0),
            limit_time: null, estimate_start_time: start_time, estimate_end_time: null,
        }
        const kyous = [
            make_mi_kyou('mi-undated', 'mi_create', old_create_time, typed_undated),
            make_mi_kyou('mi-dated', 'mi_start', start_time, typed_dated),
        ]

        kyous.sort((a, b) => compare_kyou_for_query(a, b, query))

        expect(kyous.map(kyou => kyou.id)).toEqual(['mi-dated', 'mi-undated'])
    })
})
