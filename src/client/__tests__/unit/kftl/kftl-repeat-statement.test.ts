import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

// Mock @/i18n so all KFTL modules use our test i18n
vi.mock('@/i18n', () => ({ i18n }))

import { KFTLStatement } from '@/classes/kftl/kftl-statement'
import { KFTLRequest } from '@/classes/kftl/kftl-request'
import { KFTLKmemoRequest } from '@/classes/kftl/kftl_kmemo/kftl-kmemo-request'
import { KFTLMiRequest } from '@/classes/kftl/kftl_mi/kftl-mi-request'
import { KFTLMiReKyouRequest } from '@/classes/kftl/kftl_mirekyou/kftl-mi-re-kyou-request'
import { KFTLNlogRequest } from '@/classes/kftl/kftl_nlog/kftl-nlog-request'
import { KFTLTimeIsRequest } from '@/classes/kftl/kftl_timeis/kftl-time-is-request'
import { KFTLStatementLineContext } from '@/classes/kftl/kftl-statement-line-context'
import { expand_repeats } from '@/classes/kftl/kftl_repeat/kftl-repeat-expand'
import { new_repeat_spec, parse_repeat_condition } from '@/classes/kftl/kftl_repeat/kftl-repeat-spec'
import type { GkillAPI } from '@/classes/api/gkill-api'

// Go の src/server/gkill/api/kftl/kftl_repeat_test.go の結合テストと対。
// 2026-09-02 は水曜。時刻はアンカーから取り、起点ちょうどは含めない。

function ymdhm(y: number, m: number, d: number, hh: number, mm: number): Date {
    return new Date(y, m - 1, d, hh, mm, 0, 0)
}

function pick<T>(requests: Array<KFTLRequest>, ctor: new (...args: never[]) => T): Array<T> {
    return requests.filter((request) => request instanceof (ctor as never)) as Array<T>
}

describe('繰り返しブロックの行と展開', () => {
    beforeEach(() => {
        vi.useFakeTimers()
        vi.setSystemTime(ymdhm(2026, 9, 2, 10, 0))
    })
    afterEach(() => {
        vi.useRealTimers()
    })

    test('全日時欄が同じ日数だけずれ、欄どうしの相対差は保たれる', async () => {
        const text = 'ーみ\n週報\n仕事\n18:00\n19:00\n20:00\n？？\n金\n2\n？？'
        const mis = pick(await new KFTLStatement(text).generate_requests(), KFTLMiRequest)
        expect(mis.length).toBe(2)
        const wants = [
            { start: ymdhm(2026, 9, 4, 18, 0), end: ymdhm(2026, 9, 4, 19, 0), limit: ymdhm(2026, 9, 4, 20, 0) },
            { start: ymdhm(2026, 9, 11, 18, 0), end: ymdhm(2026, 9, 11, 19, 0), limit: ymdhm(2026, 9, 11, 20, 0) },
        ]
        for (let i = 0; i < wants.length; i++) {
            expect(mis[i].get_estimate_start_time()!.getTime(), `[${i}] 見積開始`).toBe(wants[i].start.getTime())
            expect(mis[i].get_estimate_end_time()!.getTime(), `[${i}] 見積終了`).toBe(wants[i].end.getTime())
            expect(mis[i].get_limit_time()!.getTime(), `[${i}] 期限`).toBe(wants[i].limit.getTime())
        }
        expect(mis[0].get_request_id()).not.toBe(mis[1].get_request_id())
    })

    // タグ・テキストと同じく項目の位置を消費しない。閉じたあとは同じ項目位置へ戻る
    test('繰り返しブロックは項目の位置を消費しない', async () => {
        const text = 'ーみ\n週報\n仕事\n？？\n金\n2\n？？\n18:00'
        const mis = pick(await new KFTLStatement(text).generate_requests(), KFTLMiRequest)
        expect(mis.length).toBe(2)
        expect(mis[0].get_estimate_start_time()!.getTime()).toBe(ymdhm(2026, 9, 4, 18, 0).getTime())
    })

    // 支出ブロックは全支払いが1グループ。店名・関連時刻と同じブロック共有の扱い
    test('支出は支払い2件 × 3回 = 6レコード', async () => {
        const text = 'ーん\nスーパー\n牛乳\n200\nパン\n300\n？？\n金\n3\n？？'
        const nlogs = pick(await new KFTLStatement(text).generate_requests(), KFTLNlogRequest)
        expect(nlogs.length).toBe(6)
        const want_days = [
            ymdhm(2026, 9, 4, 10, 0), ymdhm(2026, 9, 4, 10, 0),
            ymdhm(2026, 9, 11, 10, 0), ymdhm(2026, 9, 11, 10, 0),
            ymdhm(2026, 9, 18, 10, 0), ymdhm(2026, 9, 18, 10, 0),
        ]
        const want_titles = ['牛乳', 'パン', '牛乳', 'パン', '牛乳', 'パン']
        for (let i = 0; i < nlogs.length; i++) {
            expect(nlogs[i].get_related_time()!.getTime(), `[${i}] 関連時刻`).toBe(want_days[i].getTime())
            expect(nlogs[i].title, `[${i}] 品名`).toBe(want_titles[i])
            expect(nlogs[i].shop_name).toBe('スーパー')
        }
    })

    // 打刻の開始時刻行は related_time にしか入らず、KFTLTimeIsRequest.start_time は do_request が
    // related_time から書くまで空。start_time をアンカーにすると 1970 からの日数ぶんずらされ、
    // 2026-09-10 に送った打刻が 2083-05-17 で登録された。年を明示的に見るのはその再発シグネチャ
    test('打刻は開始時刻を基準にし、開始と終了を同じ日数だけずらす', async () => {
        const text = 'ーち\n仕事\n08:30\n17:30\n？？\n毎日\n5\n\n2026-09-07\n？？'
        const timeiss = pick(await new KFTLStatement(text).generate_requests(), KFTLTimeIsRequest)
        expect(timeiss.length).toBe(5)
        for (let i = 0; i < timeiss.length; i++) {
            const day = 7 + i
            const start = timeiss[i].get_related_time()!
            const end = timeiss[i].get_end_time()!
            expect(start.getFullYear(), `[${i}] 開始の年`).toBe(2026)
            expect(start.getTime(), `[${i}] 開始`).toBe(ymdhm(2026, 9, day, 8, 30).getTime())
            expect(end.getTime(), `[${i}] 終了`).toBe(ymdhm(2026, 9, day, 17, 30).getTime())
            expect(timeiss[i].get_title(), `[${i}] タイトル`).toBe('仕事')
        }
        expect(new Set(timeiss.map((request) => request.get_request_id())).size, 'IDは回ごとに別').toBe(5)
    })

    test('メモは関連時刻を基準にする', async () => {
        const text = '今日の日記\n？？\n毎日\n3\n？？'
        const kmemos = pick(await new KFTLStatement(text).generate_requests(), KFTLKmemoRequest)
        expect(kmemos.length).toBe(3)
        const want = [ymdhm(2026, 9, 3, 10, 0), ymdhm(2026, 9, 4, 10, 0), ymdhm(2026, 9, 5, 10, 0)]
        for (let i = 0; i < kmemos.length; i++) {
            expect(kmemos[i].get_related_time()!.getTime(), `[${i}]`).toBe(want[i].getTime())
        }
    })

    // テキストIDを使い回すと append-only なので最後の1件以外が消える
    test('タグとテキストも複製され、テキストIDは回ごとに別', async () => {
        const text = '今日の日記\n。日記\nーー\n本文\nーー\n？？\n毎日\n2\n？？'
        const kmemos = pick(await new KFTLStatement(text).generate_requests(), KFTLKmemoRequest)
        expect(kmemos.length).toBe(2)
        for (const kmemo of kmemos) {
            expect(kmemo.get_tags()).toEqual(['日記'])
            expect(kmemo.get_texts()).toEqual(['本文'])
        }
    })

    // 4行を書き終えたあとの位置は受け皿（行ラベルは「**********」）。空行は見逃してブロックの
    // 中に留まるので、空行を挟んでから閉じても件数は変わらない
    test('4行のあとに空行を挟んでも「？？」で閉じられる', async () => {
        const text = '今日の日記\n？？\n毎日\n3\nno\n2026-09-10\n\n？？'
        const kmemos = pick(await new KFTLStatement(text).generate_requests(), KFTLKmemoRequest)
        expect(kmemos.length).toBe(3)
    })

    // 「～～」に揃える。閉じ忘れても必須2行が揃っていれば有効
    test('閉じ忘れても展開される', async () => {
        const kmemos = pick(await new KFTLStatement('今日の日記\n？？\n毎日\n3').generate_requests(), KFTLKmemoRequest)
        expect(kmemos.length).toBe(3)
    })

    test('リポストタスクは対象を同じままで繰り返し、元の記録は増えない', async () => {
        const text = '牛乳を買う\n～～\n仕事\n18:00\n？？\n金\n2\n？？\n～～'
        const requests = await new KFTLStatement(text).generate_requests()
        const mirekyous = pick(requests, KFTLMiReKyouRequest)
        expect(mirekyous.length).toBe(2)
        expect(pick(requests, KFTLKmemoRequest).length, '元の記録は1件のまま').toBe(1)
        const want = [ymdhm(2026, 9, 4, 18, 0), ymdhm(2026, 9, 11, 18, 0)]
        for (let i = 0; i < mirekyous.length; i++) {
            expect(mirekyous[i].get_estimate_start_time()!.getTime(), `[${i}]`).toBe(want[i].getTime())
        }
    })
})

describe('繰り返しブロックの不正行', () => {
    beforeEach(() => {
        vi.useFakeTimers()
        vi.setSystemTime(ymdhm(2026, 9, 2, 10, 0))
    })
    afterEach(() => {
        vi.useRealTimers()
    })

    const cases: Array<{ name: string; text: string }> = [
        { name: '打刻開始のみは繰り返せない', text: 'ーた\n仕事\n？？\n金\n3\n？？' },
        { name: '打刻終了は繰り返せない', text: 'ーえ\n仕事\n？？\n金\n3\n？？' },
        { name: 'タグ指定の打刻終了は繰り返せない', text: 'ーたえ\n仕事\n？？\n金\n3\n？？' },
        { name: '対象の記録が無い', text: '？？\n金\n3\n？？' },
        { name: '予定日時が1つも無いタスク', text: 'ーみ\n週報\n仕事\n？？\n金\n3\n？？' },
        { name: '4行を書き終えたあとの余分な行', text: '今日の日記\n？？\n毎日\n3\nno\n2026-09-10\nゴミ\n？？' },
        { name: '引数を同じ行に書いた', text: '今日の日記\n？？ 金 3' },
        { name: '条件が読めない', text: '今日の日記\n？？\nきのう\n3\n？？' },
        { name: '回数が範囲外', text: '今日の日記\n？？\n毎日\n1001\n？？' },
    ]
    for (const c of cases) {
        test(c.name, async () => {
            const invalids = await new KFTLStatement(c.text).get_invalid_line_indexs()
            expect(invalids.length, `不正行として拾われること: ${JSON.stringify(c.text)}`).toBeGreaterThan(0)
        })
    }

    // 「？？」のカーブアウトは「完全一致」と「直後が空白（引数つき）」の2つだけ。
    // 「??なんだこれ」のように直後が空白でない行は、従来どおり「？」の前方一致へ流れて
    // 日時として解釈され、日時でなければ不正行になる（Go 側と揃えてある）
    test('？？ で始まっても直後が空白でなければ関連時刻として扱う', async () => {
        const invalids = await new KFTLStatement('今日の日記\n??なんだこれ').get_invalid_line_indexs()
        expect(invalids).toEqual([1])
    })

    // 打っている途中の状態。4行のあとの空行でピンクの不正行にしない
    // （行ラベルは「**********」を出しているので、エラーにすると表示と食い違う）
    test('4行のあとの空行は不正行にならない', async () => {
        expect(await new KFTLStatement('今日の日記\n？？\n毎日\n3\nno\n2026-09-10\n\n？？').get_invalid_line_indexs()).toEqual([])
    })

    test('閉じる前の末尾の空行も不正行にならない', async () => {
        expect(await new KFTLStatement('今日の日記\n？？\n毎日\n3\nno\n2026-09-10\n').get_invalid_line_indexs()).toEqual([])
    })

    test('正しく書けば不正行にならない', async () => {
        const text = 'ーみ\n週報\n仕事\n18:00\n？？\n金\n3\nno\n2026-09-12\n？？'
        expect(await new KFTLStatement(text).get_invalid_line_indexs()).toEqual([])
    })
})

// ─── 既存スキップ（3行目 no）─────────────────────────────────────────────────

/** 既存判定の結果を差し替えられるリクエスト。実APIを立てずに「飛ばす」ところだけを固定する。 */
class RepeatMockRequest extends KFTLRequest {
    existing: Set<number>

    constructor(request_id: string, context: KFTLStatementLineContext, existing: Set<number>) {
        super(request_id, context)
        this.existing = existing
    }

    override clone_for_repeat(new_request_id: string, day_shift: number): KFTLRequest {
        const cloned = new RepeatMockRequest(new_request_id, this.get_context(), this.existing)
        this.copy_base_state_for_repeat(cloned, day_shift)
        return cloned
    }

    override async find_existing_for_repeat(): Promise<Set<number>> {
        return this.existing
    }
}

describe('既存スキップ', () => {
    const base = ymdhm(2026, 9, 2, 10, 0)
    const anchor = ymdhm(2026, 9, 2, 18, 0)
    // 既存判定は gkill_api が null だと丸ごと飛ぶ。モックは API を使わないのでダミーで足りる
    const dummy_api = {} as unknown as GkillAPI

    function make(existing: Set<number>, count: number, add_if_exists: boolean): Array<KFTLRequest> {
        const context = new KFTLStatementLineContext('tx', '', 'mock-1', '', [], true)
        const request = new RepeatMockRequest('mock-1', context, existing)
        request.set_related_time(anchor)
        const spec = new_repeat_spec(0)
        spec.cond = parse_repeat_condition('金')
        spec.count = count
        spec.origin = base
        spec.add_if_exists = add_if_exists
        request.set_repeat_spec(spec)
        return [request]
    }

    // 既定（3行目 no）は既にある回を飛ばす。同じテキストを何度送っても増えない
    test('既にある回を飛ばす', async () => {
        const existing = new Set<number>([ymdhm(2026, 9, 11, 18, 0).getTime()])
        const got = await expand_repeats(make(existing, 3, false), base, dummy_api, null)
        expect(got.length).toBe(2)
        expect(got.map((r) => r.get_related_time()!.getTime())).toEqual([
            ymdhm(2026, 9, 4, 18, 0).getTime(), ymdhm(2026, 9, 18, 18, 0).getTime(),
        ])
    })

    // 元のIDを引き継ぐのは「実際に作る最初の1件」。先頭の回が飛んでも引き継がれる
    test('先頭の回が飛ばされても作られた1件目が元のIDを持つ', async () => {
        const existing = new Set<number>([ymdhm(2026, 9, 4, 18, 0).getTime()])
        const got = await expand_repeats(make(existing, 2, false), base, dummy_api, null)
        expect(got.length).toBe(1)
        expect(got[0].get_request_id()).toBe('mock-1')
        expect(got[0].get_related_time()!.getTime()).toBe(ymdhm(2026, 9, 11, 18, 0).getTime())
    })

    test('yes は既存を見ずに全部作る', async () => {
        const existing = new Set<number>([ymdhm(2026, 9, 11, 18, 0).getTime()])
        const got = await expand_repeats(make(existing, 3, true), base, dummy_api, null)
        expect(got.length).toBe(3)
    })

    // 全部の回が既にあれば0件。再送しても増えないという冪等性の下限
    test('全部の回が既にあれば0件', async () => {
        const existing = new Set<number>([
            ymdhm(2026, 9, 4, 18, 0).getTime(),
            ymdhm(2026, 9, 11, 18, 0).getTime(),
        ])
        const got = await expand_repeats(make(existing, 2, false), base, dummy_api, null)
        expect(got.length).toBe(0)
    })
})
