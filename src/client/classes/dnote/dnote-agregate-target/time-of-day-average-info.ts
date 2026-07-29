export const MILLI_SECONDS_PER_DAY = 24 * 60 * 60 * 1000

/**
 * 時刻（時分秒）の平均を出すための累積値。
 *
 * 時刻は24時間で一周するため単純平均だと破綻する。
 * 23:00 と 01:00 の平均は 00:00 であってほしいが、
 * ミリ秒をそのまま足して割ると 12:00 になってしまう。
 * そこで各時刻を単位円上の角度に直し、ベクトルの平均から角度を戻す（円周平均）。
 *
 * なお 09:00 と 21:00 のように真逆へ二極化している場合は、
 * ベクトルが打ち消し合って平均時刻が定まらない。その場合は null を返す。
 */
export default class TimeOfDayAverageInfo {
    public total_count: number = 0
    public sin_total: number = 0
    public cos_total: number = 0

    /** 0時からの経過ミリ秒を1件足し込む */
    append(milli_second_of_day: number): void {
        const angle = (milli_second_of_day / MILLI_SECONDS_PER_DAY) * 2 * Math.PI
        this.sin_total += Math.sin(angle)
        this.cos_total += Math.cos(angle)
        this.total_count++
    }

    /** 平均時刻を0時からの経過ミリ秒で返す。求まらない場合は null */
    average_milli_second_of_day(): number | null {
        return average_milli_second_of_day(this.sin_total, this.cos_total, this.total_count)
    }

    clone(): TimeOfDayAverageInfo {
        const clone = new TimeOfDayAverageInfo()
        clone.total_count = this.total_count
        clone.sin_total = this.sin_total
        clone.cos_total = this.cos_total
        return clone
    }
}

/** Date の時分秒を0時からの経過ミリ秒に直す。日付とタイムゾーンは見ない */
export function milli_second_of_day(time: Date): number {
    return ((time.getHours() * 60 + time.getMinutes()) * 60 + time.getSeconds()) * 1000
}

/**
 * 累積した単位ベクトルから平均時刻を0時からの経過ミリ秒で求める。
 * 求まらない場合は null。
 */
export function average_milli_second_of_day(sin_total: number, cos_total: number, total_count: number): number | null {
    if (total_count === 0) {
        return null
    }
    const sin_average = sin_total / total_count
    const cos_average = cos_total / total_count

    // ベクトルの長さがほぼ0＝時刻が真逆に散っていて平均に意味がない
    if (Math.sqrt(sin_average * sin_average + cos_average * cos_average) < 1e-9) {
        return null
    }

    let angle = Math.atan2(sin_average, cos_average)
    if (angle < 0) {
        angle += 2 * Math.PI
    }
    return (angle / (2 * Math.PI)) * MILLI_SECONDS_PER_DAY
}
