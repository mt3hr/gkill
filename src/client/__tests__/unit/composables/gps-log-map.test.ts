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
