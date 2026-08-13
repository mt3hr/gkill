import type { FindKyouQuery } from "../api/find_query/find-kyou-query"
import type { Kyou } from "../datas/kyou"
import { DnoteTrendAggregator } from "./dnote-trend-aggregator"
import type DnoteTrendPoint from "./dnote-trend/dnote-trend-point"
import { correlation_statistics } from "./dnote-correlation/correlation-statistics"
import type {
    DnoteCorrelationCell,
    DnoteCorrelationGraphQuery,
    DnoteCorrelationMethod,
    DnoteCorrelationPairPoint,
    DnoteCorrelationResult,
} from "./dnote-correlation"

/**
 * 2つの指標の時系列を突き合わせて、行列の1セル分（散布図の点列＋統計量）を組み立てる。
 * 統計量そのものの計算は dnote-correlation/correlation-statistics.ts が持つ。
 */
export function build_correlation_cell(row_metric_id: string, column_metric_id: string, row_series: Array<DnoteTrendPoint>, column_series: Array<DnoteTrendPoint>, lag: number, method: DnoteCorrelationMethod): DnoteCorrelationCell {
    const points = new Array<DnoteCorrelationPairPoint>()
    // 正のlagは「行の指標が先、列の指標が後」を表す。
    // インデックスで対応付けることで日・週・月のどの粒度でも同じ意味になる。
    for (let row_index = 0; row_index < row_series.length; row_index++) {
        const column_index = row_index + lag
        if (column_index < 0 || column_index >= column_series.length) continue
        const row_point = row_series[row_index]
        const column_point = column_series[column_index]
        // 記録が存在しないバケットの0は観測値ではないため、相関へ混ぜない。
        if (row_point.match_kyous.length === 0 || column_point.match_kyous.length === 0) continue
        if (!Number.isFinite(row_point.value) || !Number.isFinite(column_point.value)) continue
        points.push({
            row_bucket_key: row_point.bucket_key,
            row_label: row_point.label,
            column_bucket_key: column_point.bucket_key,
            column_label: column_point.label,
            x: row_point.value,
            y: column_point.value,
            x_value_string: row_point.value_string,
            y_value_string: column_point.value_string,
            row_match_kyous: row_point.match_kyous,
            column_match_kyous: column_point.match_kyous,
        })
    }
    return {
        row_metric_id,
        column_metric_id,
        points,
        ...correlation_statistics(points.map(point => point.x), points.map(point => point.y), method),
    }
}

/**
 * 相関グラフの集計。指標ごとにトレンドグラフと同じ時系列集計を回し、
 * 全指標の総当たり（対称な正方行列）でセルを埋める。サーバーAPIは使用しない。
 */
export class DnoteCorrelationAggregator {
    constructor(private readonly query: DnoteCorrelationGraphQuery) { }

    public async aggregate(abort_controller: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<DnoteCorrelationResult> {
        // 指標ごとの系列は互いに独立なので並行に集計してよい
        const series = new Map<string, Array<DnoteTrendPoint>>()
        await Promise.all(this.query.metrics.map(async metric => {
            const aggregator = new DnoteTrendAggregator(metric.predicate, metric.aggregate_target, this.query.granularity)
            series.set(metric.id, await aggregator.aggregate_trend(abort_controller, kyous, find_kyou_query, kyou_is_loaded))
        }))

        const cells = this.query.metrics.map(row_metric => this.query.metrics.map(column_metric => {
            const row_series = series.get(row_metric.id) ?? []
            const column_series = series.get(column_metric.id) ?? []
            return build_correlation_cell(row_metric.id, column_metric.id, row_series, column_series, this.query.lag, this.query.method)
        }))
        return { query: this.query, series, cells }
    }
}
