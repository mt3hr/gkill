import { MILLI_SECONDS_PER_DAY } from "../dnote-aggregate-target/time-of-day-average-info"

/** 時刻平均の累積値。TimeOfDayAverageInfo と同じ形（clone 後の素のオブジェクトも受ける） */
export interface TimeOfDayTotals {
    sin_total: number
    cos_total: number
    total_count: number
}

export function is_time_of_day_totals(value: unknown): value is TimeOfDayTotals {
    return typeof value === "object" && value !== null && "sin_total" in value && "cos_total" in value && "total_count" in value
}

/**
 * バケットごとの時刻平均を、系列全体の平均時刻からのずれ（ミリ秒、−12h〜+12h）に直す。
 *
 * 時刻を「0時からの経過ミリ秒」のまま数値にすると 23:30 と 00:30 が両端に割れ、
 * 就寝時刻のように 0 時をまたいで散る系列は折れ線が崖になり相関も壊れる。
 * どこで切っても崖はできるので、切る位置を固定せず「その系列の平均時刻の真裏」で切る。
 * 就寝時刻なら平均 1 時の真裏（13 時）で切れて 17 時〜翌 11 時が一続きになる。
 *
 * 平均が定まらない系列（時刻が真逆に散っている）は 0 時起点へ倒す。
 * 値が無いバケットは null（呼び出し側が 0 に倒す）。
 */
export function time_of_day_deviations(totals: Array<TimeOfDayTotals | null>): Array<number | null> {
    let sin_sum = 0
    let cos_sum = 0
    let count = 0
    for (const total of totals) {
        if (!total || total.total_count === 0) continue
        sin_sum += total.sin_total
        cos_sum += total.cos_total
        count += total.total_count
    }
    const reference_angle = count > 0 && Math.hypot(sin_sum / count, cos_sum / count) >= 1e-9
        ? Math.atan2(sin_sum, cos_sum)
        : 0

    return totals.map(total => {
        if (!total || total.total_count === 0) return null
        const sin_average = total.sin_total / total.total_count
        const cos_average = total.cos_total / total.total_count
        if (Math.hypot(sin_average, cos_average) < 1e-9) return null
        let deviation = Math.atan2(sin_average, cos_average) - reference_angle
        // (−π, π] に折り返す
        while (deviation > Math.PI) deviation -= 2 * Math.PI
        while (deviation <= -Math.PI) deviation += 2 * Math.PI
        return (deviation / (2 * Math.PI)) * MILLI_SECONDS_PER_DAY
    })
}
