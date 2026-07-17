/**
 * DnoteTrendAggregator tests.
 *
 * DnoteTrendAggregator uses load_kyous() internally,
 * which calls kyou.clone(), kyou.load_typed_datas(), etc.
 * We mock load_kyous to return plain objects directly.
 */
import { vi } from 'vitest'
import {
  makeKyouWithKmemo,
  makeKyouWithNlog,
} from '../../helpers/factory'

// Mock load_kyous before importing aggregators
vi.mock('@/classes/dnote/kyou-loader', () => ({
  default: vi.fn(async (_abort: unknown, kyous: unknown[]) => kyous),
}))

import { DnoteTrendAggregator } from '@/classes/dnote/dnote-trend-aggregator'
import agregated_value_to_number from '@/classes/dnote/dnote-trend/agregated-value-to-number'
import AverageInfo from '@/classes/dnote/dnote-agregate-target/average-info'
import KmemoContentContainsPredicate from '@/classes/dnote/dnote-predicate/kmemo-content-contains-predicate'
import DataTypePrefixPredicate from '@/classes/dnote/dnote-predicate/data-type-prefix-predicate'
import AgregateCountKyou from '@/classes/dnote/dnote-agregate-target/agregate-count-kyou'
import AgregateSumNlogAmount from '@/classes/dnote/dnote-agregate-target/agregate-sum-nlog-amount'

const controller = new AbortController()
const emptyQuery = {} as never

function makeQuery(start: string, end: string) {
  return {
    use_calendar: true,
    calendar_start_date: new Date(start),
    calendar_end_date: new Date(end),
  } as never
}

// Kyous used in tests need clone() for the aggregator's match_kyous
function makeTestKyou(factory: (...args: never[]) => Record<string, unknown>, related_time: string, ...args: never[]) {
  const obj = factory(...args)
  obj.related_time = new Date(related_time)
  obj.clone = () => ({ ...obj })
  return obj as never
}

describe('DnoteTrendAggregator', () => {
  test('day granularity: zero-filled ascending buckets with correct counts', async () => {
    const predicate = new KmemoContentContainsPredicate('メモ')
    const target = new AgregateCountKyou()
    const aggregator = new DnoteTrendAggregator(predicate, target, 'day')

    const kyous = [
      makeTestKyou(makeKyouWithKmemo, '2026-07-01T10:00:00+09:00', 'メモA' as never),
      makeTestKyou(makeKyouWithKmemo, '2026-07-01T12:00:00+09:00', 'メモB' as never),
      makeTestKyou(makeKyouWithKmemo, '2026-07-03T09:00:00+09:00', 'メモC' as never),
      makeTestKyou(makeKyouWithKmemo, '2026-07-07T23:00:00+09:00', 'メモD' as never),
    ]

    const query = makeQuery('2026-07-01T00:00:00+09:00', '2026-07-07T23:59:59+09:00')
    const points = await aggregator.aggregate_trend(controller, kyous, query, true)

    expect(points.length).toBe(7)
    expect(points.map(p => p.value)).toEqual([2, 0, 1, 0, 0, 0, 1])
    // ascending bucket keys
    const keys = points.map(p => p.bucket_key)
    expect(keys).toEqual([...keys].sort())
    expect(points[0].match_kyous.length).toBe(2)
    expect(points[1].match_kyous.length).toBe(0)
  })

  test('week granularity: buckets straddling an ISO year boundary stay distinct and ordered', async () => {
    const predicate = new KmemoContentContainsPredicate('メモ')
    const target = new AgregateCountKyou()
    const aggregator = new DnoteTrendAggregator(predicate, target, 'week')

    const kyous = [
      makeTestKyou(makeKyouWithKmemo, '2025-12-30T10:00:00+09:00', 'メモA' as never),
      makeTestKyou(makeKyouWithKmemo, '2026-01-06T10:00:00+09:00', 'メモB' as never),
    ]

    // 2025-12-29 (Mon) and 2026-01-05 (Mon) are distinct ISO weeks
    const query = makeQuery('2025-12-29T00:00:00+09:00', '2026-01-11T23:59:59+09:00')
    const points = await aggregator.aggregate_trend(controller, kyous, query, true)

    expect(points.length).toBe(2)
    expect(points[0].value).toBe(1)
    expect(points[1].value).toBe(1)
    expect(points[0].bucket_key < points[1].bucket_key).toBe(true)
  })

  test('month granularity: 12 buckets summing nlog amounts per month', async () => {
    const predicate = new DataTypePrefixPredicate('nlog')
    const target = new AgregateSumNlogAmount()
    const aggregator = new DnoteTrendAggregator(predicate, target, 'month')

    const kyous = [
      makeTestKyou(makeKyouWithNlog, '2025-08-15T10:00:00+09:00', '店A' as never, '品' as never, 300 as never),
      makeTestKyou(makeKyouWithNlog, '2025-08-20T10:00:00+09:00', '店B' as never, '品' as never, 700 as never),
      makeTestKyou(makeKyouWithNlog, '2026-07-01T10:00:00+09:00', '店C' as never, '品' as never, 500 as never),
    ]

    const query = makeQuery('2025-08-01T00:00:00+09:00', '2026-07-31T23:59:59+09:00')
    const points = await aggregator.aggregate_trend(controller, kyous, query, true)

    expect(points.length).toBe(12)
    expect(points[0].value).toBe(1000)
    expect(points[0].value_string).toBe('1000')
    expect(points[11].value).toBe(500)
    expect(points.slice(1, 11).every(p => p.value === 0)).toBe(true)
  })

  test('predicate filtering excludes non-matching kyous from all buckets', async () => {
    const predicate = new KmemoContentContainsPredicate('対象')
    const target = new AgregateCountKyou()
    const aggregator = new DnoteTrendAggregator(predicate, target, 'day')

    const kyous = [
      makeTestKyou(makeKyouWithKmemo, '2026-07-01T10:00:00+09:00', '対象メモ' as never),
      makeTestKyou(makeKyouWithKmemo, '2026-07-01T11:00:00+09:00', '関係ない' as never),
    ]

    const query = makeQuery('2026-07-01T00:00:00+09:00', '2026-07-02T23:59:59+09:00')
    const points = await aggregator.aggregate_trend(controller, kyous, query, true)

    expect(points.length).toBe(2)
    expect(points[0].value).toBe(1)
    expect(points[1].value).toBe(0)
  })

  test('kyous outside the calendar window are ignored', async () => {
    const predicate = new KmemoContentContainsPredicate('メモ')
    const target = new AgregateCountKyou()
    const aggregator = new DnoteTrendAggregator(predicate, target, 'day')

    const kyous = [
      makeTestKyou(makeKyouWithKmemo, '2026-06-30T10:00:00+09:00', 'メモ枠外' as never),
      makeTestKyou(makeKyouWithKmemo, '2026-07-01T10:00:00+09:00', 'メモ枠内' as never),
    ]

    const query = makeQuery('2026-07-01T00:00:00+09:00', '2026-07-01T23:59:59+09:00')
    const points = await aggregator.aggregate_trend(controller, kyous, query, true)

    expect(points.length).toBe(1)
    expect(points[0].value).toBe(1)
  })

  test('no calendar range: window is derived from min/max related_time of kyous', async () => {
    const predicate = new KmemoContentContainsPredicate('メモ')
    const target = new AgregateCountKyou()
    const aggregator = new DnoteTrendAggregator(predicate, target, 'day')

    const kyous = [
      makeTestKyou(makeKyouWithKmemo, '2026-07-03T10:00:00+09:00', 'メモ中' as never),
      makeTestKyou(makeKyouWithKmemo, '2026-07-01T10:00:00+09:00', 'メモ古' as never),
      makeTestKyou(makeKyouWithKmemo, '2026-07-05T10:00:00+09:00', 'メモ新' as never),
    ]

    const points = await aggregator.aggregate_trend(controller, kyous, emptyQuery, true)

    expect(points.length).toBe(5) // 7/1 .. 7/5
    expect(points.map(p => p.value)).toEqual([1, 0, 1, 0, 1])
  })

  test('no calendar range and no kyous: single bucket for now', async () => {
    const predicate = new KmemoContentContainsPredicate('メモ')
    const target = new AgregateCountKyou()
    const aggregator = new DnoteTrendAggregator(predicate, target, 'day')

    const points = await aggregator.aggregate_trend(controller, [], emptyQuery, true)

    expect(points.length).toBe(1)
    expect(points[0].value).toBe(0)
  })

  test('bucket count cap keeps the newest side', async () => {
    const predicate = new KmemoContentContainsPredicate('メモ')
    const target = new AgregateCountKyou()
    const aggregator = new DnoteTrendAggregator(predicate, target, 'day')

    const kyous = [
      makeTestKyou(makeKyouWithKmemo, '2025-01-01T10:00:00+09:00', 'メモ古すぎ' as never),
      makeTestKyou(makeKyouWithKmemo, '2026-07-01T10:00:00+09:00', 'メモ新しい' as never),
    ]

    // 546日間 > 400バケット上限
    const query = makeQuery('2025-01-01T00:00:00+09:00', '2026-07-01T23:59:59+09:00')
    const points = await aggregator.aggregate_trend(controller, kyous, query, true)

    expect(points.length).toBe(400)
    // 新しい側が残る: 末尾は2026-07-01で件数1、先頭は2025-01-01より後
    expect(points[points.length - 1].bucket_key).toBe('2026-07-01')
    expect(points[points.length - 1].value).toBe(1)
    expect(points[0].bucket_key > '2025-01-01').toBe(true)
  })
})

describe('agregated_value_to_number', () => {
  test('null and undefined map to 0', () => {
    expect(agregated_value_to_number(null)).toBe(0)
    expect(agregated_value_to_number(undefined)).toBe(0)
  })

  test('numbers pass through', () => {
    expect(agregated_value_to_number(42)).toBe(42)
    expect(agregated_value_to_number(-3.5)).toBe(-3.5)
  })

  test('AverageInfo divides total_value by total_count', () => {
    const info = new AverageInfo()
    info.total_count = 4
    info.total_value = 10
    expect(agregated_value_to_number(info)).toBe(2.5)
  })

  test('AverageInfo with zero count maps to 0', () => {
    const info = new AverageInfo()
    expect(agregated_value_to_number(info)).toBe(0)
  })
})
