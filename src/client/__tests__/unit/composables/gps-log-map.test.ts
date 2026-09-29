/**
 * useGpsLogMap の期間スライダー計算の回帰テスト。
 *
 * 以前は開始日から1日ずつ進めて終了日と文字列一致するまで回しており、
 * 「行き過ぎたら止まる」ガードが無かったため、終了日が開始日より前だったり
 * 日付がパース不能だったりすると同期ループが永久に回ってタブが固まった。
 *
 * このテストは「無限ループしないこと」を確認するもので、
 * 退行すると通常のアサーション失敗ではなくタイムアウトで落ちる。
 */
import { describe, it, expect, vi } from 'vitest'
import '@/classes/api/gkill-api'
import { useGpsLogMap } from '@/classes/use-gps-log-map'
import type { GPSLogMapProps } from '@/pages/views/gps-log-map-props'
import type { GPSLogMapEmits } from '@/pages/views/gps-log-map-emits'

function setup(start_date: Date, end_date: Date) {
    const props = {
        start_date,
        end_date,
        marker_time: start_date,
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
    it('開始日と終了日が同じなら1日ぶん', () => {
        const d = new Date('2026-08-03T00:00:00Z')
        const { time_slider_max } = setup(d, d)
        expect(time_slider_max.value).toBe(DAY - 1)
    })

    it('3日間なら3日ぶん', () => {
        const { time_slider_max } = setup(
            new Date('2026-08-03T00:00:00Z'),
            new Date('2026-08-05T00:00:00Z'),
        )
        expect(time_slider_max.value).toBe(DAY * 3 - 1)
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
            new Date('2026-01-01T00:00:00Z'),
            new Date('2026-12-31T00:00:00Z'),
        )
        expect(time_slider_max.value).toBe(DAY * 365 - 1)
        expect(Date.now() - started).toBeLessThan(1000)
    })
})

// 日付表示をクリックして開く日付ピッカー（v-model は date_picker_model）。
// 地図に何日を映すかは親の props（start_date）なので、選んだ日を requested_change_map_date で
// 親へ返すだけ。rykv は地図だけを切り替え、ダッシュボードは表示日ごと移す。
// 一覧を動かす requested_focus_time とは別のイベントで、こちらを出してはいけない
describe('useGpsLogMap の日付ピッカー', () => {
    function setup_with_emits(start_date: Date) {
        const emitted: Array<{ event: string, args: unknown[] }> = []
        const props = {
            start_date,
            end_date: start_date,
            marker_time: start_date,
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

    it('ピッカーの表示値は親が映している日（start_date）', () => {
        const start_date = new Date(2026, 7, 3, 9, 30, 0)
        const { api } = setup_with_emits(start_date)
        expect(api.date_picker_model.value.getTime()).toBe(start_date.getTime())
    })

    it('日を選ぶとピッカーが閉じ、requested_change_map_date をその日の 0 時で1回だけ出す', () => {
        const { emitted, api } = setup_with_emits(new Date(2026, 7, 3))
        api.is_show_date_picker.value = true

        api.date_picker_model.value = new Date(2016, 4, 15, 13, 45, 30)

        expect(api.is_show_date_picker.value, '選んだあともピッカーが開いたまま').toBe(false)
        expect(emitted.map((e) => e.event), '親へ返す requested_change_map_date が1回ちょうどではない').toEqual(['requested_change_map_date'])
        // 時刻付きで選んでも 0 時に丸める。丸めないと、rykv が同じ値をマーカー時刻（marker_time）にも
        // 入れるので、マーカー（スライダー）が選んだ日の 0 時ではなく選んだときの時刻の位置に立つ
        // （線は start_date_str の日付単位で引くので変わらない）
        const date = emitted[0].args[0] as Date
        expect(date.getTime()).toBe(new Date(2016, 4, 15, 0, 0, 0).getTime())
    })

    it('一覧を動かす requested_focus_time は出さない', () => {
        const { emitted, api } = setup_with_emits(new Date(2026, 7, 3))

        api.date_picker_model.value = new Date(2016, 4, 15)

        expect(emitted.some((e) => e.event === 'requested_focus_time'), '地図の日付ピッカーが一覧まで動かしている').toBe(false)
    })
})
