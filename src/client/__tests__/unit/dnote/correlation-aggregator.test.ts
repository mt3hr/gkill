/**
 * 相関グラフの統計計算・ペア対応付け・設定のJSON往復を確かめる。
 *
 * 統計量は自前実装（外部ライブラリを使わない）なので、値が合っていることを
 * 手計算できる小さな標本で固定する。p値の期待値は t分布表と一致する桁まで見る。
 * ペア対応付けは「lagの向き」と「観測が無いバケットを混ぜないこと」が本質なので、
 * Kyou の中身には依存させず、集計済みの DnoteTrendPoint を直接組み立てて渡す。
 */
import { describe, expect, test } from "vitest"
import { build_correlation_cell } from "@/classes/dnote/dnote-correlation-aggregator"
import { correlation_statistics, pearson_correlation } from "@/classes/dnote/dnote-correlation/correlation-statistics"
import {
  DnoteCorrelationGraphQuery,
  DnoteCorrelationMetric,
  parse_dnote_correlation_graph,
  serialize_dnote_correlation_graph,
} from "@/classes/dnote/dnote-correlation"
import AggregateSumKCNumValue from "@/classes/dnote/dnote-aggregate-target/aggregate-sum-kc-num-value"
import KCTitleEqualPredicate from "@/classes/dnote/dnote-predicate/kc-title-equal-predicate"
import type DnoteTrendPoint from "@/classes/dnote/dnote-trend/dnote-trend-point"

function point(index: number, value: number, observed = true): DnoteTrendPoint {
  return {
    bucket_key: `2026-08-${String(index + 1).padStart(2, "0")}`,
    label: `8/${index + 1}`,
    value,
    value_string: value.toString(),
    match_kyous: observed ? [{} as never] : [],
  }
}

describe("correlation statistics", () => {
  test("Pearson detects perfect positive and negative correlation", () => {
    expect(pearson_correlation([1, 2, 3, 4], [2, 4, 6, 8])).toBeCloseTo(1)
    expect(pearson_correlation([1, 2, 3, 4], [8, 6, 4, 2])).toBeCloseTo(-1)
  })

  test("returns coefficient, p-value and Fisher 95% confidence interval", () => {
    const result = correlation_statistics([1, 2, 3, 4, 5], [2, 1, 4, 3, 5], "pearson")
    expect(result.coefficient).toBeCloseTo(0.8)
    expect(result.p_value).toBeCloseTo(0.10409, 4)
    expect(result.confidence_low).not.toBeNull()
    expect(result.confidence_high).not.toBeNull()
    expect(result.confidence_low!).toBeLessThan(result.coefficient!)
    expect(result.confidence_high!).toBeGreaterThan(result.coefficient!)
  })

  test("Spearman uses average ranks for ties", () => {
    const result = correlation_statistics([1, 1, 2, 3], [10, 10, 20, 30], "spearman")
    expect(result.coefficient).toBeCloseTo(1)
    expect(result.p_value).toBe(0)
  })

  test("insufficient or constant samples are unavailable", () => {
    expect(correlation_statistics([1, 2], [1, 2], "pearson").coefficient).toBeNull()
    expect(correlation_statistics([1, 1, 1], [1, 2, 3], "pearson").coefficient).toBeNull()
    expect(correlation_statistics([1, 2, 3], [1, 2, 3], "pearson").confidence_low).toBeNull()
  })
})

describe("correlation pair alignment", () => {
  test("positive lag compares an earlier row bucket with a later column bucket", () => {
    const row = [point(0, 1), point(1, 2), point(2, 3), point(3, 4)]
    const column = [point(0, 100), point(1, 10), point(2, 20), point(3, 30)]
    const cell = build_correlation_cell("row", "column", row, column, 1, "pearson")
    expect(cell.points.map(value => [value.x, value.y])).toEqual([[1, 10], [2, 20], [3, 30]])
    expect(cell.coefficient).toBeCloseTo(1)
  })

  test("missing and non-finite observations are excluded pairwise", () => {
    const row = [point(0, 1), point(1, 2, false), point(2, Number.NaN), point(3, 4), point(4, 5)]
    const column = [point(0, 10), point(1, 20), point(2, 30), point(3, 40), point(4, 50)]
    const cell = build_correlation_cell("row", "column", row, column, 0, "pearson")
    expect(cell.sample_size).toBe(3)
    expect(cell.points.map(value => value.x)).toEqual([1, 4, 5])
  })
})

test("correlation graph settings survive JSON round-trip", () => {
  const metric = new DnoteCorrelationMetric()
  metric.id = "metric-1"
  metric.title = "Weight"
  metric.predicate = new KCTitleEqualPredicate("weight")
  metric.aggregate_target = new AggregateSumKCNumValue()
  const query = new DnoteCorrelationGraphQuery()
  query.id = "graph-1"
  query.title = "Health"
  query.granularity = "week"
  query.method = "spearman"
  query.lag = 1
  query.metrics = [metric]

  const restored = parse_dnote_correlation_graph(serialize_dnote_correlation_graph(query))
  expect(restored.id).toBe(query.id)
  expect(restored.granularity).toBe("week")
  expect(restored.method).toBe("spearman")
  expect(restored.lag).toBe(1)
  expect(restored.metrics[0].predicate.predicate_struct_to_json()).toEqual(metric.predicate.predicate_struct_to_json())
  expect(restored.metrics[0].aggregate_target.to_json()).toEqual(metric.aggregate_target.to_json())
})
