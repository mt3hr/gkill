import type { Kyou } from "../datas/kyou"
import type DnoteAggregateTarget from "./dnote-aggregate-target"
import AggregateCountKyou from "./dnote-aggregate-target/aggregate-count-kyou"
import type DnotePredicate from "./dnote-predicate"
import AndPredicate from "./dnote-predicate/and-predicate"
import type DnoteTrendPoint from "./dnote-trend/dnote-trend-point"
import type { DnoteTrendGranularity } from "./dnote-trend/dnote-trend-types"
import { build_dnote_aggregate_target_from_json, build_dnote_predicate_from_json } from "./serialize/register-dictionary"

export type DnoteCorrelationMethod = "pearson" | "spearman"

/** 相関グラフの1指標。トレンドグラフ1本ぶんと同じ「条件＋集計対象」を持つ */
export class DnoteCorrelationMetric {
    id = ""
    title = ""
    predicate: DnotePredicate = new AndPredicate([])
    aggregate_target: DnoteAggregateTarget = new AggregateCountKyou()
}

/** 相関グラフの設定。粒度と時間ずれ(lag)は全指標で共通にする（指標ごとだと行列の意味が定まらない） */
export class DnoteCorrelationGraphQuery {
    id = ""
    title = ""
    granularity: DnoteTrendGranularity = "day"
    method: DnoteCorrelationMethod = "pearson"
    lag = 0
    metrics = new Array<DnoteCorrelationMetric>()
}

/** 散布図の1点。対応付けたバケットの両側を保持し、点から元のKyouへ辿れるようにする */
export interface DnoteCorrelationPairPoint {
    row_bucket_key: string
    row_label: string
    column_bucket_key: string
    column_label: string
    x: number
    y: number
    x_value_string: string
    y_value_string: string
    row_match_kyous: Array<Kyou>
    column_match_kyous: Array<Kyou>
}

/** ヒートマップの1セル。算出できなかった場合は係数・p値・信頼区間がすべて null になる */
export interface DnoteCorrelationCell {
    row_metric_id: string
    column_metric_id: string
    coefficient: number | null
    p_value: number | null
    confidence_low: number | null
    confidence_high: number | null
    sample_size: number
    points: Array<DnoteCorrelationPairPoint>
}

export interface DnoteCorrelationResult {
    query: DnoteCorrelationGraphQuery
    series: Map<string, Array<DnoteTrendPoint>>
    cells: Array<Array<DnoteCorrelationCell>>
}

/**
 * ApplicationConfig に入っている JSON から設定を復元する。
 * 設定は手で書き換えられるうえ版も混在するので、型が合わない値はすべて既定値へ倒す。
 */
export function parse_dnote_correlation_graph(json: Record<string, unknown>): DnoteCorrelationGraphQuery {
    const query = new DnoteCorrelationGraphQuery()
    query.id = typeof json.id === "string" ? json.id : ""
    query.title = typeof json.title === "string" ? json.title : ""
    query.granularity = json.granularity === "week" || json.granularity === "month" ? json.granularity : "day"
    query.method = json.method === "spearman" ? "spearman" : "pearson"
    query.lag = typeof json.lag === "number" && Number.isInteger(json.lag) ? json.lag : 0
    query.metrics = (Array.isArray(json.metrics) ? json.metrics : []).map(value => {
        const metric_json = value as Record<string, unknown>
        const metric = new DnoteCorrelationMetric()
        metric.id = typeof metric_json.id === "string" ? metric_json.id : ""
        metric.title = typeof metric_json.title === "string" ? metric_json.title : ""
        if (metric_json.predicate && typeof metric_json.predicate === "object") {
            metric.predicate = build_dnote_predicate_from_json(metric_json.predicate as Record<string, unknown>)
        }
        if (metric_json.aggregate_target && typeof metric_json.aggregate_target === "object") {
            metric.aggregate_target = build_dnote_aggregate_target_from_json(metric_json.aggregate_target as Record<string, unknown>)
        }
        return metric
    })
    return query
}

export function serialize_dnote_correlation_graph(query: DnoteCorrelationGraphQuery): Record<string, unknown> {
    return {
        id: query.id,
        title: query.title,
        granularity: query.granularity,
        method: query.method,
        lag: query.lag,
        metrics: query.metrics.map(metric => ({
            id: metric.id,
            title: metric.title,
            predicate: metric.predicate.predicate_struct_to_json(),
            aggregate_target: metric.aggregate_target.to_json(),
        })),
    }
}

/**
 * JSON往復で複製する。条件・集計対象はクラスなので浅いコピーでは共有されてしまい、
 * 編集ダイアログでの変更が、キャンセルしても元の定義へ漏れる。
 */
export function clone_dnote_correlation_graph(query: DnoteCorrelationGraphQuery): DnoteCorrelationGraphQuery {
    return parse_dnote_correlation_graph(serialize_dnote_correlation_graph(query))
}
