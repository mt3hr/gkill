/**
 * D-note Aggregate Target tests.
 */
import {
  makeKyouWithNlog,
  makeKyouWithLantana,
  makeKyouWithKc,
  makeKyouWithKmemo,
  makeKyouWithGitCommitLog,
  makeKyouWithTimeis,
} from '../../helpers/factory'

import AggregateCountKyou from '@/classes/dnote/dnote-aggregate-target/aggregate-count-kyou'
import AggregateSumNlogAmount from '@/classes/dnote/dnote-aggregate-target/aggregate-sum-nlog-amount'
import AggregateSumLantanaMood from '@/classes/dnote/dnote-aggregate-target/aggregate-sum-lantana-mood'
import AggregateSumKCNumValue from '@/classes/dnote/dnote-aggregate-target/aggregate-sum-kc-num-value'
import AggregateAverageLantanaMood from '@/classes/dnote/dnote-aggregate-target/aggregate-average-lantana-mood'
import AggregateAverageNlogAmount from '@/classes/dnote/dnote-aggregate-target/aggregate-average-nlog-amount'
import AggregateSumGitCommitLogCodeCount from '@/classes/dnote/dnote-aggregate-target/aggregate-sum-git-commit-log-code-count'
import AggregateAverageTimeIsStartTime from '@/classes/dnote/dnote-aggregate-target/aggregate-average-timeis-start-time'
import AggregateAverageTimeIsEndTime from '@/classes/dnote/dnote-aggregate-target/aggregate-average-timeis-end-time'
import AggregateAverageKCNumValue from '@/classes/dnote/dnote-aggregate-target/aggregate-average-kc-num-value'
import AggregateMaxKCNumValue from '@/classes/dnote/dnote-aggregate-target/aggregate-max-kc-num-value'
import format_aggregated_number from '@/classes/dnote/dnote-aggregate-target/format-aggregated-number'

const asKyou = (obj: unknown) => obj
const emptyQuery = {} as never

// ========== Count ==========

describe('AggregateCountKyou', () => {
  const target = new AggregateCountKyou()

  test('counts from null to 1', async () => {
    const kyou = asKyou(makeKyouWithKmemo('test'))
    const result = await target.append_aggregate_element_value(null, kyou, emptyQuery)
    expect(result).toBe(1)
  })

  test('accumulates count across multiple kyous', async () => {
    const kyou = asKyou(makeKyouWithKmemo('test'))
    let val = await target.append_aggregate_element_value(null, kyou, emptyQuery)
    val = await target.append_aggregate_element_value(val, kyou, emptyQuery)
    val = await target.append_aggregate_element_value(val, kyou, emptyQuery)
    expect(val).toBe(3)
  })

  test('result_to_string returns count as string', async () => {
    expect(await target.result_to_string(5)).toBe('5')
    expect(await target.result_to_string(null)).toBe('0')
  })

  test('to_json returns correct type', () => {
    expect(target.to_json().type).toBe('AggregateCountKyou')
  })
})

// ========== Sum Nlog Amount ==========

describe('AggregateSumNlogAmount', () => {
  const target = new AggregateSumNlogAmount()

  test('sums nlog amounts from null', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', '品', 500))
    const result = await target.append_aggregate_element_value(null, kyou, emptyQuery)
    expect(result).toBe(500)
  })

  test('accumulates across multiple kyous', async () => {
    const k1 = asKyou(makeKyouWithNlog('店A', '品A', 300))
    const k2 = asKyou(makeKyouWithNlog('店B', '品B', 700))
    let val = await target.append_aggregate_element_value(null, k1, emptyQuery)
    val = await target.append_aggregate_element_value(val, k2, emptyQuery)
    expect(val).toBe(1000)
  })

  test('result_to_string formats correctly', async () => {
    expect(await target.result_to_string(1234)).toBe('1234')
    expect(await target.result_to_string(null)).toBe('0')
  })
})

// ========== Sum Lantana Mood ==========

describe('AggregateSumLantanaMood', () => {
  const target = new AggregateSumLantanaMood()

  test('sums mood values from null', async () => {
    const kyou = asKyou(makeKyouWithLantana(7))
    const result = await target.append_aggregate_element_value(null, kyou, emptyQuery)
    expect(result).toBe(7)
  })

  test('accumulates across kyous', async () => {
    const k1 = asKyou(makeKyouWithLantana(3))
    const k2 = asKyou(makeKyouWithLantana(8))
    let val = await target.append_aggregate_element_value(null, k1, emptyQuery)
    val = await target.append_aggregate_element_value(val, k2, emptyQuery)
    expect(val).toBe(11)
  })
})

// ========== Sum KC Num Value ==========

describe('AggregateSumKCNumValue', () => {
  const target = new AggregateSumKCNumValue()

  test('sums kc num_value from null', async () => {
    const kyou = asKyou(makeKyouWithKc('歩数', 5000))
    const result = await target.append_aggregate_element_value(null, kyou, emptyQuery)
    expect(result).toBe(5000)
  })

  test('accumulates across kyous', async () => {
    const k1 = asKyou(makeKyouWithKc('歩数', 3000))
    const k2 = asKyou(makeKyouWithKc('歩数', 7000))
    let val = await target.append_aggregate_element_value(null, k1, emptyQuery)
    val = await target.append_aggregate_element_value(val, k2, emptyQuery)
    expect(val).toBe(10000)
  })
})

// ========== Average Lantana Mood ==========

describe('AggregateAverageLantanaMood', () => {
  const target = new AggregateAverageLantanaMood()

  test('averages mood values across kyous', async () => {
    const k1 = asKyou(makeKyouWithLantana(4))
    const k2 = asKyou(makeKyouWithLantana(8))
    let val = await target.append_aggregate_element_value(null, k1, emptyQuery)
    val = await target.append_aggregate_element_value(val, k2, emptyQuery)
    const str = await target.result_to_string(val)
    expect(str).toBe('6') // (4+8)/2 = 6
  })

  test('handles single element', async () => {
    const kyou = asKyou(makeKyouWithLantana(7))
    const val = await target.append_aggregate_element_value(null, kyou, emptyQuery)
    const str = await target.result_to_string(val)
    expect(str).toBe('7')
  })

  test('result_to_string with null returns 0', async () => {
    const str = await target.result_to_string(null)
    expect(str).toBe('0')
  })
})

// ========== Average Nlog Amount ==========

describe('AggregateAverageNlogAmount', () => {
  const target = new AggregateAverageNlogAmount()

  test('averages nlog amounts', async () => {
    const k1 = asKyou(makeKyouWithNlog('店', '品', 200))
    const k2 = asKyou(makeKyouWithNlog('店', '品', 800))
    let val = await target.append_aggregate_element_value(null, k1, emptyQuery)
    val = await target.append_aggregate_element_value(val, k2, emptyQuery)
    const str = await target.result_to_string(val)
    expect(str).toBe('500') // (200+800)/2
  })

  test('handles single element', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', '品', 300))
    const val = await target.append_aggregate_element_value(null, kyou, emptyQuery)
    const str = await target.result_to_string(val)
    expect(str).toBe('300')
  })
})

// ========== Sum Git Commit Log Code Count ==========

describe('AggregateSumGitCommitLogCodeCount', () => {
  const target = new AggregateSumGitCommitLogCodeCount()

  test('sums code counts (addition - deletion)', async () => {
    const k1 = asKyou(makeKyouWithGitCommitLog('commit1', 10, 5))
    const k2 = asKyou(makeKyouWithGitCommitLog('commit2', 20, 3))
    let val = await target.append_aggregate_element_value(null, k1, emptyQuery)
    val = await target.append_aggregate_element_value(val, k2, emptyQuery)
    // (10-5) + (20-3) = 5 + 17 = 22
    expect(val).toBe(22)
  })
})

// ========== TimeIs の平均開始/終了時刻 ==========

describe('AggregateAverageTimeIsStartTime', () => {
  const target = new AggregateAverageTimeIsStartTime()

  const kyouStartingAt = (hour: number, minute = 0) =>
    asKyou(makeKyouWithTimeis('テスト', {
      typed_timeis: { start_time: new Date(2025, 2, 15, hour, minute, 0), end_time: null },
    }))

  test('averages start times of day', async () => {
    let val = await target.append_aggregate_element_value(null, kyouStartingAt(9), emptyQuery)
    val = await target.append_aggregate_element_value(val, kyouStartingAt(11), emptyQuery)
    expect(await target.result_to_string(val)).toBe('10:00')
  })

  test('averages across midnight instead of landing at noon', async () => {
    // 単純平均だと 12:00 になってしまうケース
    let val = await target.append_aggregate_element_value(null, kyouStartingAt(23), emptyQuery)
    val = await target.append_aggregate_element_value(val, kyouStartingAt(1), emptyQuery)
    expect(await target.result_to_string(val)).toBe('00:00')
  })

  test('a single record averages to itself', async () => {
    const val = await target.append_aggregate_element_value(null, kyouStartingAt(7, 30), emptyQuery)
    expect(await target.result_to_string(val)).toBe('07:30')
  })

  test('returns empty string when nothing matched', async () => {
    const kyou = asKyou(makeKyouWithKmemo('timeisではない'))
    const val = await target.append_aggregate_element_value(null, kyou, emptyQuery)
    expect(await target.result_to_string(val)).toBe('')
    expect(await target.result_to_string(null)).toBe('')
  })

  test('returns empty string when times cancel each other out', async () => {
    // 09:00 と 21:00 はちょうど真逆なので平均時刻が定まらない
    let val = await target.append_aggregate_element_value(null, kyouStartingAt(9), emptyQuery)
    val = await target.append_aggregate_element_value(val, kyouStartingAt(21), emptyQuery)
    expect(await target.result_to_string(val)).toBe('')
  })
})

describe('AggregateAverageTimeIsEndTime', () => {
  const target = new AggregateAverageTimeIsEndTime()

  const kyouEndingAt = (hour: number | null) =>
    asKyou(makeKyouWithTimeis('テスト', {
      typed_timeis: {
        start_time: new Date(2025, 2, 15, 8, 0, 0),
        end_time: hour === null ? null : new Date(2025, 2, 15, hour, 0, 0),
      },
    }))

  test('averages end times of day', async () => {
    let val = await target.append_aggregate_element_value(null, kyouEndingAt(18), emptyQuery)
    val = await target.append_aggregate_element_value(val, kyouEndingAt(20), emptyQuery)
    expect(await target.result_to_string(val)).toBe('19:00')
  })

  test('skips records that have not ended yet', async () => {
    let val = await target.append_aggregate_element_value(null, kyouEndingAt(null), emptyQuery)
    expect(await target.result_to_string(val)).toBe('')
    val = await target.append_aggregate_element_value(val, kyouEndingAt(18), emptyQuery)
    expect(await target.result_to_string(val)).toBe('18:00')
  })
})

// ========== 表示用の丸め ==========

describe('format_aggregated_number', () => {
  test('rounds to 2 decimals', () => {
    // 素の toString() だと 71.28604651162791 になっていた
    expect(format_aggregated_number(71.28604651162791)).toBe('71.29')
  })

  test('does not add a decimal point to integers', () => {
    expect(format_aggregated_number(8903)).toBe('8903')
  })

  test('drops a trailing zero', () => {
    expect(format_aggregated_number(71.2)).toBe('71.2')
  })

  test('keeps the sign of negative values', () => {
    expect(format_aggregated_number(-1234.5678)).toBe('-1234.57')
  })

  test('falls back to 0 for non finite values', () => {
    expect(format_aggregated_number(Number.NaN)).toBe('0')
    expect(format_aggregated_number(Number.POSITIVE_INFINITY)).toBe('0')
  })
})

describe('KC の集計結果は丸めて返す', () => {
  test('AggregateAverageKCNumValue rounds the average', async () => {
    const target = new AggregateAverageKCNumValue()
    let val = await target.append_aggregate_element_value(null, asKyou(makeKyouWithKc('安静時心拍数', 70)), emptyQuery)
    val = await target.append_aggregate_element_value(val, asKyou(makeKyouWithKc('安静時心拍数', 71)), emptyQuery)
    val = await target.append_aggregate_element_value(val, asKyou(makeKyouWithKc('安静時心拍数', 73)), emptyQuery)
    expect(await target.result_to_string(val)).toBe('71.33')
  })

  test('AggregateSumKCNumValue removes float noise', async () => {
    const target = new AggregateSumKCNumValue()
    // 0.1 + 0.2 は素の合計だと 0.30000000000000004 になる
    let val = await target.append_aggregate_element_value(null, asKyou(makeKyouWithKc('消費カロリー(日計)', 0.1)), emptyQuery)
    val = await target.append_aggregate_element_value(val, asKyou(makeKyouWithKc('消費カロリー(日計)', 0.2)), emptyQuery)
    expect(await target.result_to_string(val)).toBe('0.3')
  })

  test('AggregateMaxKCNumValue keeps the raw value', async () => {
    const target = new AggregateMaxKCNumValue()
    let val = await target.append_aggregate_element_value(null, asKyou(makeKyouWithKc('心拍数(最大)', 151)), emptyQuery)
    val = await target.append_aggregate_element_value(val, asKyou(makeKyouWithKc('心拍数(最大)', 199)), emptyQuery)
    expect(await target.result_to_string(val)).toBe('199')
  })
})

describe('気分と出費の集計結果も丸めて返す', () => {
  test('AggregateAverageLantanaMood rounds the average', async () => {
    const target = new AggregateAverageLantanaMood()
    let val = await target.append_aggregate_element_value(null, asKyou(makeKyouWithLantana(3)), emptyQuery)
    val = await target.append_aggregate_element_value(val, asKyou(makeKyouWithLantana(4)), emptyQuery)
    val = await target.append_aggregate_element_value(val, asKyou(makeKyouWithLantana(8)), emptyQuery)
    expect(await target.result_to_string(val)).toBe('5')
  })

  test('AggregateSumNlogAmount removes float noise', async () => {
    const target = new AggregateSumNlogAmount()
    // 金額に小数が入ると素の合計は 12345.700000000001 になる
    let val = await target.append_aggregate_element_value(null, asKyou(makeKyouWithNlog('店A', '品A', 12345.6)), emptyQuery)
    val = await target.append_aggregate_element_value(val, asKyou(makeKyouWithNlog('店B', '品B', 0.1)), emptyQuery)
    expect(await target.result_to_string(val)).toBe('12345.7')
  })
})
