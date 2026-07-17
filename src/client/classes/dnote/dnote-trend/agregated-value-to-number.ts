import AverageInfo from "../dnote-agregate-target/average-info"

// 集計対象の累積値（number または AverageInfo）をグラフ描画用の数値に変換する
export default function agregated_value_to_number(value: unknown): number {
    if (value === null || value === undefined) {
        return 0
    }
    if (typeof value === "number") {
        return Number.isFinite(value) ? value : 0
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
