/**
 * useGpsLogMap の期間スライダー計算の回帰テスト。
 *
 * 以前は開始日から1日ずつ進めて終了日と文字列一致するまで回しており、
 * 「行き過ぎたら止まる」ガードが無かったため、終了日が開始日より前だったり
 * 日付がパース不能だったりすると同期ループが永久に回ってタブが固まった。
 *
 * このテストは「無限ループしないこと」を確認するもので、
 * 退行すると通常のアサーション失敗ではなくタイムアウトで落ちる。
 *
 * 期間に時刻を指定できるようになってからは、刻みを「期間の秒数」で出す。
 * 日数差から (日数差 + 1) * 86400 - 1 としていたころは、09:00〜18:00 のような指定でも
 * スライダーが1日ぶんを刻み、右半分が無効域になって「動かしても最後の点に貼り付く」
 * 形で静かに壊れていた。サイドバーが送るのは「初日の 0 時」〜「最終日の 23:59:59.999」なので、
 * 日単位の指定に対する刻みは従来と同じ値になる。
 */
import { describe, it, expect, vi } from 'vitest'
import { nextTick, reactive } from 'vue'
import '@/classes/api/gkill-api'
import { useGpsLogMap } from '@/classes/use-gps-log-map'
import type { GPSLogMapProps } from '@/pages/views/gps-log-map-props'
import type { GPSLogMapEmits } from '@/pages/views/gps-log-map-emits'

function setup(start_time: Date, end_time: Date) {
    const props = {
        start_time,
        end_time,
        marker_time: start_time,
        app_content_height: 600,
        gkill_api: {
            get_google_map_api_key: () => '',
            get_gps_log: vi.fn().mockResolvedValue({ errors: [], gps_logs: [] }),
        },
    } as unknown as GPSLogMapProps
    const emits = (() => { }) as unknown as GPSLogMapEmits
    return useGpsLogMap({ props, emits })
}

const DAY = 86400

describe('useGpsLogMap の time_slider_max', () => {
    // サイドバーが送るのは「初日の 0 時」〜「最終日の 23:59:59.999」。
    // 日単位の指定に対する刻みは、時刻対応の前後で同じ値でなければならない
    it('1日ぶん（0時〜23:59:59.999）なら1日ぶん', () => {
        const { time_slider_max } = setup(
            new Date(2026, 7, 3, 0, 0, 0, 0),
            new Date(2026, 7, 3, 23, 59, 59, 999),
        )
        expect(time_slider_max.value).toBe(DAY - 1)
    })

    it('3日間なら3日ぶん', () => {
        const { time_slider_max } = setup(
            new Date(2026, 7, 3, 0, 0, 0, 0),
            new Date(2026, 7, 5, 23, 59, 59, 999),
        )
        expect(time_slider_max.value).toBe(DAY * 3 - 1)
    })

    it('同じ日の中の時刻範囲なら、その秒数だけ刻む', () => {
        const { time_slider_max } = setup(
            new Date(2026, 7, 3, 9, 0, 0, 0),
            new Date(2026, 7, 3, 18, 0, 0, 0),
        )
        expect(time_slider_max.value, '時刻範囲なのに1日ぶん刻むと右半分が無効域になる').toBe(9 * 3600)
    })

    // ここから下は、修正前は無限ループしていたケース
    it('終了日が開始日より前でも有限時間で返る', () => {
        const { time_slider_max } = setup(
            new Date('2026-08-05T00:00:00Z'),
            new Date('2026-08-03T00:00:00Z'),
        )
        expect(time_slider_max.value).toBe(DAY - 1)
    })

    it('終了日がInvalid Dateでも有限時間で返る', () => {
        const { time_slider_max } = setup(
            new Date('2026-08-03T00:00:00Z'),
            new Date(NaN),
        )
        expect(time_slider_max.value).toBe(DAY - 1)
    })

    it('開始日がInvalid Dateでも有限時間で返る', () => {
        const { time_slider_max } = setup(
            new Date(NaN),
            new Date('2026-08-03T00:00:00Z'),
        )
        expect(time_slider_max.value).toBe(DAY - 1)
    })

    // 年単位でも一瞬で返ること（以前は日数ぶん moment() を生成していた）
    it('1年間でも即座に返る', () => {
        const started = Date.now()
        const { time_slider_max } = setup(
            new Date(2026, 0, 1, 0, 0, 0, 0),
            new Date(2026, 11, 31, 23, 59, 59, 999),
        )
        expect(time_slider_max.value).toBe(DAY * 365 - 1)
        expect(Date.now() - started).toBeLessThan(1000)
    })
})

// 日付表示をクリックして開く日付ピッカー（v-model は date_picker_model）。
// 地図に何日を映すかは親の props（start_time）なので、選んだ日を requested_change_map_date で
// 親へ返すだけ。rykv は地図だけを切り替え、ダッシュボードは表示日ごと移す。
// 一覧を動かす requested_focus_time とは別のイベントで、こちらを出してはいけない
describe('useGpsLogMap の日付ピッカー', () => {
    function setup_with_emits(start_time: Date) {
        const emitted: Array<{ event: string, args: unknown[] }> = []
        const props = {
            start_time,
            end_time: start_time,
            marker_time: start_time,
            app_content_height: 600,
            gkill_api: {
                get_google_map_api_key: () => '',
                get_gps_log: vi.fn().mockResolvedValue({ errors: [], gps_logs: [] }),
            },
        } as unknown as GPSLogMapProps
        const emits = ((event: string, ...args: unknown[]) => {
            emitted.push({ event, args })
        }) as unknown as GPSLogMapEmits
        return { emitted, api: useGpsLogMap({ props, emits }), props }
    }

    it('ピッカーの表示値は親が映している日（start_time）', () => {
        const start_time = new Date(2026, 7, 3, 9, 30, 0)
        const { api } = setup_with_emits(start_time)
        expect(api.date_picker_model.value.getTime()).toBe(start_time.getTime())
    })

    it('日を選ぶとピッカーが閉じ、requested_change_map_date をその日の 0 時で1回だけ出す', () => {
        const { emitted, api } = setup_with_emits(new Date(2026, 7, 3))
        api.is_show_date_picker.value = true

        api.date_picker_model.value = new Date(2016, 4, 15, 13, 45, 30)

        expect(api.is_show_date_picker.value, '選んだあともピッカーが開いたまま').toBe(false)
        expect(emitted.map((e) => e.event), '親へ返す requested_change_map_date が1回ちょうどではない').toEqual(['requested_change_map_date'])
        // 時刻付きで選んでも 0 時に丸める。丸めないと、rykv が同じ値をマーカー時刻（marker_time）にも
        // 入れるので、マーカー（スライダー）が選んだ日の 0 時ではなく選んだときの時刻の位置に立つ
        // （線は props の時刻範囲で引く）
        const date = emitted[0].args[0] as Date
        expect(date.getTime()).toBe(new Date(2016, 4, 15, 0, 0, 0).getTime())
    })

    it('一覧を動かす requested_focus_time は出さない', () => {
        const { emitted, api } = setup_with_emits(new Date(2026, 7, 3))

        api.date_picker_model.value = new Date(2016, 4, 15)

        expect(emitted.some((e) => e.event === 'requested_focus_time'), '地図の日付ピッカーが一覧まで動かしている').toBe(false)
    })
})

// 期間に時刻を指定できるようになったので、同じ日の中で時刻だけ動かしても GPS を引き直す。
// 以前は再取得の watch が日付文字列（YYYY-MM-DD）に張られており、時刻だけの変更では
// 何も起きなかった。記録一覧は時刻で絞られているのに地図だけ丸一日ぶん出る形で食い違う。
describe('useGpsLogMap の再取得', () => {
    function setup_reactive(start_time: Date, end_time: Date) {
        const get_gps_log = vi.fn().mockResolvedValue({ errors: [], gps_logs: [] })
        const props = reactive({
            start_time,
            end_time,
            marker_time: start_time,
            app_content_height: 600,
            gkill_api: {
                get_google_map_api_key: () => '',
                get_gps_log,
            },
        })
        const emits = (() => { }) as unknown as GPSLogMapEmits
        const api = useGpsLogMap({ props: props as unknown as GPSLogMapProps, emits })
        return { props, api, get_gps_log }
    }

    async function flush(times = 4): Promise<void> {
        for (let i = 0; i < times; i++) {
            await nextTick()
        }
    }

    it('同じ日の中で開始時刻だけ動かしても引き直す', async () => {
        const { props, get_gps_log } = setup_reactive(
            new Date(2026, 7, 3, 0, 0, 0, 0),
            new Date(2026, 7, 3, 23, 59, 59, 999),
        )
        await flush()
        get_gps_log.mockClear()

        props.start_time = new Date(2026, 7, 3, 9, 0, 0, 0)
        await flush()

        expect(get_gps_log, '同じ日の中で時刻だけ動かしても引き直していない').toHaveBeenCalledTimes(1)
        const req = get_gps_log.mock.calls[0][0]
        expect(req.start_date.getTime(), '日単位へ丸めている').toBe(new Date(2026, 7, 3, 9, 0, 0, 0).getTime())
        expect(req.end_date.getTime()).toBe(new Date(2026, 7, 3, 23, 59, 59, 999).getTime())
    })

    it('同じ日の中で終了時刻だけ動かしても引き直す', async () => {
        const { props, get_gps_log } = setup_reactive(
            new Date(2026, 7, 3, 0, 0, 0, 0),
            new Date(2026, 7, 3, 23, 59, 59, 999),
        )
        await flush()
        get_gps_log.mockClear()

        props.end_time = new Date(2026, 7, 3, 18, 0, 59, 999)
        await flush()

        expect(get_gps_log).toHaveBeenCalledTimes(1)
        expect(get_gps_log.mock.calls[0][0].end_date.getTime())
            .toBe(new Date(2026, 7, 3, 18, 0, 59, 999).getTime())
    })

    it('時刻を動かすとスライダーの刻みも追従する', async () => {
        const { props, api } = setup_reactive(
            new Date(2026, 7, 3, 0, 0, 0, 0),
            new Date(2026, 7, 3, 23, 59, 59, 999),
        )
        await flush()
        expect(api.time_slider_max.value).toBe(86399)

        props.start_time = new Date(2026, 7, 3, 9, 0, 0, 0)
        props.end_time = new Date(2026, 7, 3, 18, 0, 0, 0)
        await flush()

        expect(api.time_slider_max.value).toBe(9 * 3600)
    })
})
