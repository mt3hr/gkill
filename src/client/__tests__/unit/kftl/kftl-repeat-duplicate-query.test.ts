import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

// Mock @/i18n so all KFTL modules use our test i18n
vi.mock('@/i18n', () => ({ i18n }))

import { KFTLStatement } from '@/classes/kftl/kftl-statement'
import { KFTLMiRequest } from '@/classes/kftl/kftl_mi/kftl-mi-request'
import { KFTLTimeIsRequest } from '@/classes/kftl/kftl_timeis/kftl-time-is-request'
import { find_existing_anchors } from '@/classes/kftl/kftl_repeat/kftl-repeat-duplicate'
import { MiCheckState } from '@/classes/api/find_query/mi-check-state'
import type { GkillAPI } from '@/classes/api/gkill-api'
import type { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import type { Kyou } from '@/classes/datas/kyou'

// 既存判定（「？？」ブロックの3行目 no）が**実際に送る検索条件**を固定する。
//
// FindKyouQuery のコンストラクタは tags / reps を `[]` で初期化する。null が
// 「フィルタ未使用」で、非nullの空配列は「0件指定」なので、既定のまま送ると
// サーバは無条件に0件を返す（find_filter.go の filterTagsKyous と rep名フィルタ）。
// そうなると既存判定が丸ごと効かず、3行目が no でも全回が作られる
// ＝ 同じテキストを送るたびに記録が増える。
//
// kftl-repeat-statement.test.ts は find_existing_for_repeat ごとモックで差し替えるので、
// この失敗を素通しする。**送るクエリそのもの**はここでしか見ていない。
//
// Go 側の対: kftl_repeat_duplicate.go の newRepeatDuplicateQuery（Tags / Reps は nil のまま）

function ymdhm(y: number, m: number, d: number, hh: number, mm: number): Date {
    return new Date(y, m - 1, d, hh, mm, 0, 0)
}

interface CapturingAPI {
    api: GkillAPI
    sent: Array<GetKyousRequest>
}

/** 送られた GetKyousRequest を貯めるだけの API。 */
function capturing_api(kyous: Array<Kyou>): CapturingAPI {
    const sent: Array<GetKyousRequest> = []
    const api = {
        get_kyous: (request: GetKyousRequest) => {
            sent.push(request)
            // 成功時 messages / errors は null で返る（Goの構造体タグに omitempty が無い）。
            // 空配列で返すと null を踏むコードを素通しする
            return Promise.resolve({ kyous: kyous, messages: null, errors: null })
        },
    } as unknown as GkillAPI
    return { api: api, sent: sent }
}

/** 型別データを読み終えた状態のタスクの Kyou。anchor_of が見る欄だけ持たせる。 */
function mi_kyou(title: string, board_name: string, estimate_start_time: Date): Kyou {
    return {
        is_deleted: false,
        data_type: 'mi_create',
        load_typed_datas: () => Promise.resolve([]),
        typed_mi: {
            title: title,
            board_name: board_name,
            estimate_start_time: estimate_start_time,
            estimate_end_time: null,
            limit_time: null,
        },
    } as unknown as Kyou
}

describe('既存判定が送る検索条件', () => {
    const from = ymdhm(2026, 9, 3, 18, 0)
    const to = ymdhm(2026, 9, 19, 18, 0)

    // コンストラクタ既定の [] は「0件指定」。null（未使用）で送らないと必ず0件になる
    test('タグと記録保管場所では絞らない', async () => {
        const mock = capturing_api([])
        await find_existing_anchors(mock.api, from, to, false, () => true, () => null)
        expect(mock.sent.length).toBe(1)
        expect(mock.sent[0].query.tags, 'tags は null（フィルタ未使用）').toBeNull()
        expect(mock.sent[0].query.reps, 'reps は null（フィルタ未使用）').toBeNull()
    })

    test('期間は渡されたぶんをそのまま使う', async () => {
        const mock = capturing_api([])
        await find_existing_anchors(mock.api, from, to, false, () => true, () => null)
        expect(mock.sent[0].query.calendar_start_date!.getTime()).toBe(from.getTime())
        expect(mock.sent[0].query.calendar_end_date!.getTime()).toBe(to.getTime())
    })

    test('タスクは予定日時の射影を全部立て、チェック状態で絞らない', async () => {
        const mock = capturing_api([])
        await find_existing_anchors(mock.api, from, to, true, () => true, () => null)
        const query = mock.sent[0].query
        // 射影を1つでも落とすと、その日時軸を持つ記録を取りこぼして重複を作る
        expect(query.for_mi).toBe(true)
        expect(query.include_create_mi, 'include_create_mi').toBe(true)
        expect(query.include_check_mi, 'include_check_mi').toBe(true)
        expect(query.include_limit_mi, 'include_limit_mi').toBe(true)
        expect(query.include_start_mi, 'include_start_mi').toBe(true)
        expect(query.include_end_mi, 'include_end_mi').toBe(true)
        // 既定の uncheck のままだと完了済みのタスクを取りこぼし、再送で重複する
        expect(query.mi_check_state, 'mi_check_state').toBe(MiCheckState.all)
    })

    test('タスク以外はタスク用の条件を立てない', async () => {
        const mock = capturing_api([])
        await find_existing_anchors(mock.api, from, to, false, () => true, () => null)
        expect(mock.sent[0].query.for_mi).toBe(false)
    })

    // data_type で絞ってから load_typed_datas する。読まずに済むものへは往復しない
    test('data_type が一致しない記録は型別データを読みに行かない', async () => {
        const load_typed_datas = vi.fn(() => Promise.resolve([]))
        const kyou = { is_deleted: false, data_type: 'kmemo', load_typed_datas: load_typed_datas } as unknown as Kyou
        const mock = capturing_api([kyou])
        const anchors = await find_existing_anchors(mock.api, from, to, false,
            (data_type) => data_type === 'timeis', () => ymdhm(2026, 9, 11, 18, 0))
        expect(load_typed_datas).not.toHaveBeenCalled()
        expect(anchors.size).toBe(0)
    })
})

describe('メモ帳からタスクを繰り返したときの既存スキップ', () => {
    // 2026-09-02 は水曜。金曜は 9/4・9/11・9/18
    const text = 'ーみ\n週報\n仕事\n18:00\n\n\n？？\n金\n3\n？？'

    beforeEach(() => {
        vi.useFakeTimers()
        vi.setSystemTime(ymdhm(2026, 9, 2, 10, 0))
    })
    afterEach(() => {
        vi.useRealTimers()
    })

    function mis(requests: Array<unknown>): Array<KFTLMiRequest> {
        return requests.filter((request) => request instanceof KFTLMiRequest) as Array<KFTLMiRequest>
    }

    test('既にあるタスクの回だけ飛ばす', async () => {
        const mock = capturing_api([mi_kyou('週報', '仕事', ymdhm(2026, 9, 11, 18, 0))])
        const got = mis(await new KFTLStatement(text).generate_requests(mock.api, null))
        expect(got.length, '9/11 は既にあるので飛ばす').toBe(2)
        expect(got.map((request) => request.get_estimate_start_time()!.getTime())).toEqual([
            ymdhm(2026, 9, 4, 18, 0).getTime(), ymdhm(2026, 9, 18, 18, 0).getTime(),
        ])
    })

    // 送信経路の端から端まで通したときも既定値のまま送っていないこと
    test('送信経路でもタグ・記録保管場所で絞らない', async () => {
        const mock = capturing_api([])
        await new KFTLStatement(text).generate_requests(mock.api, null)
        expect(mock.sent.length).toBe(1)
        expect(mock.sent[0].query.tags).toBeNull()
        expect(mock.sent[0].query.reps).toBeNull()
        expect(mock.sent[0].query.mi_check_state).toBe(MiCheckState.all)
    })

    // タイトルが違えば別の記録。既存判定は型別データの中身で見る
    test('タイトルが違う記録は既存とみなさない', async () => {
        const mock = capturing_api([mi_kyou('日報', '仕事', ymdhm(2026, 9, 11, 18, 0))])
        const got = mis(await new KFTLStatement(text).generate_requests(mock.api, null))
        expect(got.length).toBe(3)
    })

    // 板名が違えば別の記録
    test('板名が違う記録は既存とみなさない', async () => {
        const mock = capturing_api([mi_kyou('週報', '私用', ymdhm(2026, 9, 11, 18, 0))])
        const got = mis(await new KFTLStatement(text).generate_requests(mock.api, null))
        expect(got.length).toBe(3)
    })
})

// ─── 状態を持つ型（打刻・リポストタスク）────────────────────────────────────
//
// タスクの mi_check_state と同じ穴が他の型にも無いかを実測で固定する。
// 打刻は「実行中 / 終了済み」という状態を持ち、1件が開始行（timeis_start）と
// 終了行（timeis_end）の2つの Kyou として返る。どちらも同じ TimeIs の版を指すので
// アンカー（開始時刻）は同じになり、二重に数えてはいけない。

/** 型別データを読み終えた状態の打刻の Kyou。開始行・終了行のどちらも同じ実体を指す。 */
function timeis_kyou(data_type: string, title: string, start_time: Date, end_time: Date | null): Kyou {
    return {
        is_deleted: false,
        data_type: data_type,
        load_typed_datas: () => Promise.resolve([]),
        typed_timeis: { title: title, start_time: start_time, end_time: end_time },
    } as unknown as Kyou
}

describe('メモ帳から打刻を繰り返したときの既存スキップ', () => {
    // 2026-09-02 は水曜。金曜は 9/4・9/11・9/18
    const text = 'ーち\n作業\n？2026-09-02 09:00:00\n？2026-09-02 10:00:00\n？？\n金\n3\n？？'

    beforeEach(() => {
        vi.useFakeTimers()
        vi.setSystemTime(ymdhm(2026, 9, 2, 8, 0))
    })
    afterEach(() => {
        vi.useRealTimers()
    })

    function timeiss(requests: Array<unknown>): Array<KFTLTimeIsRequest> {
        return requests.filter((request) => request instanceof KFTLTimeIsRequest) as Array<KFTLTimeIsRequest>
    }

    // 打刻の状態（実行中 / 終了済み）で絞る条件は既定でも立たない。
    // include_end_timeis は既定 true で「終了行も返す」側なので、取りこぼしは起きない
    test('終了済みの打刻でもその回を飛ばす', async () => {
        const mock = capturing_api([
            timeis_kyou('timeis_start', '作業', ymdhm(2026, 9, 11, 9, 0), ymdhm(2026, 9, 11, 10, 0)),
        ])
        const got = timeiss(await new KFTLStatement(text).generate_requests(mock.api, null))
        expect(got.length, '9/11 は既にあるので飛ばす').toBe(2)
        expect(mock.sent[0].query.tags).toBeNull()
        expect(mock.sent[0].query.reps).toBeNull()
        // 実行中だけに絞る条件（playing_time）も立てない
        expect(mock.sent[0].query.playing_time).toBeNull()
    })

    // 開始行と終了行は同じIDの同じ版なので、アンカーは1つにまとまる
    test('開始行と終了行が両方返っても1回ぶんしか飛ばさない', async () => {
        const start_time = ymdhm(2026, 9, 11, 9, 0)
        const end_time = ymdhm(2026, 9, 11, 10, 0)
        const mock = capturing_api([
            timeis_kyou('timeis_start', '作業', start_time, end_time),
            timeis_kyou('timeis_end', '作業', start_time, end_time),
        ])
        const got = timeiss(await new KFTLStatement(text).generate_requests(mock.api, null))
        expect(got.length).toBe(2)
    })

    // 実行中（終了時刻なし）の打刻も既存とみなす。開始時刻がアンカーなので状態に依らない
    test('実行中の打刻でもその回を飛ばす', async () => {
        const mock = capturing_api([
            timeis_kyou('timeis_start', '作業', ymdhm(2026, 9, 11, 9, 0), null),
        ])
        const got = timeiss(await new KFTLStatement(text).generate_requests(mock.api, null))
        expect(got.length).toBe(2)
    })

    test('タイトルが違う打刻は既存とみなさない', async () => {
        const mock = capturing_api([
            timeis_kyou('timeis_start', '休憩', ymdhm(2026, 9, 11, 9, 0), ymdhm(2026, 9, 11, 10, 0)),
        ])
        const got = timeiss(await new KFTLStatement(text).generate_requests(mock.api, null))
        expect(got.length).toBe(3)
    })
})
