import AverageInfo from "../dnote-aggregate-target/average-info"
import TimeOfDayAverageInfo, { average_milli_second_of_day } from "../dnote-aggregate-target/time-of-day-average-info"

// 集計対象の累積値（number / AverageInfo / TimeOfDayAverageInfo）を
// グラフ描画用の数値に変換する
export default function aggregated_value_to_number(value: unknown): number {
    if (value === null || value === undefined) {
        return 0
    }
    if (typeof value === "number") {
        return Number.isFinite(value) ? value : 0
    }
    // 時刻の平均は円周平均なので、単純な total_value / total_count では出せない。
    // AverageInfo より先に判定する
    if (value instanceof TimeOfDayAverageInfo || (typeof value === "object" && "sin_total" in value && "cos_total" in value)) {
        const info = value as TimeOfDayAverageInfo
        const average = average_milli_second_of_day(info.sin_total, info.cos_total, info.total_count)
        return average === null ? 0 : average
    }
    if (value instanceof AverageInfo || (typeof value === "object" && "total_count" in value && "total_value" in value)) {
        const average_info = value as AverageInfo
        if (average_info.total_count === 0 || average_info.total_value === null) {
            return 0
        }
        return average_info.total_value / average_info.total_count
    }
    return 0
}
