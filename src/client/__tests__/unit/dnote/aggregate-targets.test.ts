/**
 * D-note Aggregate Target tests.
 */
import {
  makeKyouWithNlog,
  makeKyouWithLantana,
  makeKyouWithKc,
  makeKyouWithKmemo,
  makeKyouWithGitCommitLog,
} from '../../helpers/factory'

import AgregateCountKyou from '@/classes/dnote/dnote-agregate-target/agregate-count-kyou'
import AgregateSumNlogAmount from '@/classes/dnote/dnote-agregate-target/agregate-sum-nlog-amount'
import AgregateSumLantanaMood from '@/classes/dnote/dnote-agregate-target/agregate-sum-lantana-mood'
import AgregateSumKCNumValue from '@/classes/dnote/dnote-agregate-target/agregate-sum-kc-num-value'
import AgregateAverageLantanaMood from '@/classes/dnote/dnote-agregate-target/agregate-average-lantana-mood'
import AgregateAverageNlogAmount from '@/classes/dnote/dnote-agregate-target/agregate-average-nlog-amount'
import AgregateSumGitCommitLogCodeCount from '@/classes/dnote/dnote-agregate-target/agregate-sum-git-commit-log-code-count'

const asKyou = (obj: any) => obj
const emptyQuery = {} as any

// ========== Count ==========

describe('AgregateCountKyou', () => {
  const target = new AgregateCountKyou()

  test('counts from null to 1', async () => {
    const kyou = asKyou(makeKyouWithKmemo('test'))
    const result = await target.append_agregate_element_value(null, kyou, emptyQuery)
    expect(result).toBe(1)
  })

  test('accumulates count across multiple kyous', async () => {
    const kyou = asKyou(makeKyouWithKmemo('test'))
    let val = await target.append_agregate_element_value(null, kyou, emptyQuery)
    val = await target.append_agregate_element_value(val, kyou, emptyQuery)
    val = await target.append_agregate_element_value(val, kyou, emptyQuery)
    expect(val).toBe(3)
  })

  test('result_to_string returns count as string', async () => {
    expect(await target.result_to_string(5)).toBe('5')
    expect(await target.result_to_string(null)).toBe('0')
  })

  test('to_json returns correct type', () => {
    expect(target.to_json().type).toBe('AgregateCountKyou')
  })
})

// ========== Sum Nlog Amount ==========

describe('AgregateSumNlogAmount', () => {
  const target = new AgregateSumNlogAmount()

  test('sums nlog amounts from null', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', '品', 500))
    const result = await target.append_agregate_element_value(null, kyou, emptyQuery)
    expect(result).toBe(500)
  })

  test('accumulates across multiple kyous', async () => {
    const k1 = asKyou(makeKyouWithNlog('店A', '品A', 300))
    const k2 = asKyou(makeKyouWithNlog('店B', '品B', 700))
    let val = await target.append_agregate_element_value(null, k1, emptyQuery)
    val = await target.append_agregate_element_value(val, k2, emptyQuery)
    expect(val).toBe(1000)
  })

  test('result_to_string formats correctly', async () => {
    expect(await target.result_to_string(1234)).toBe('1234')
    expect(await target.result_to_string(null)).toBe('0')
  })
})

// ========== Sum Lantana Mood ==========

describe('AgregateSumLantanaMood', () => {
  const target = new AgregateSumLantanaMood()

  test('sums mood values from null', async () => {
    const kyou = asKyou(makeKyouWithLantana(7))
    const result = await target.append_agregate_element_value(null, kyou, emptyQuery)
    expect(result).toBe(7)
  })

  test('accumulates across kyous', async () => {
    const k1 = asKyou(makeKyouWithLantana(3))
    const k2 = asKyou(makeKyouWithLantana(8))
    let val = await target.append_agregate_element_value(null, k1, emptyQuery)
    val = await target.append_agregate_element_value(val, k2, emptyQuery)
    expect(val).toBe(11)
  })
})

// ========== Sum KC Num Value ==========

describe('AgregateSumKCNumValue', () => {
  const target = new AgregateSumKCNumValue()

  test('sums kc num_value from null', async () => {
    const kyou = asKyou(makeKyouWithKc('歩数', 5000))
    const result = await target.append_agregate_element_value(null, kyou, emptyQuery)
    expect(result).toBe(5000)
  })

  test('accumulates across kyous', async () => {
    const k1 = asKyou(makeKyouWithKc('歩数', 3000))
    const k2 = asKyou(makeKyouWithKc('歩数', 7000))
    let val = await target.append_agregate_element_value(null, k1, emptyQuery)
    val = await target.append_agregate_element_value(val, k2, emptyQuery)
    expect(val).toBe(10000)
  })
})

// ========== Average Lantana Mood ==========

describe('AgregateAverageLantanaMood', () => {
  const target = new AgregateAverageLantanaMood()

  test('averages mood values across kyous', async () => {
    const k1 = asKyou(makeKyouWithLantana(4))
    const k2 = asKyou(makeKyouWithLantana(8))
    let val = await target.append_agregate_element_value(null, k1, emptyQuery)
    val = await target.append_agregate_element_value(val, k2, emptyQuery)
    const str = await target.result_to_string(val)
    expect(str).toBe('6') // (4+8)/2 = 6
  })

  test('handles single element', async () => {
    const kyou = asKyou(makeKyouWithLantana(7))
    const val = await target.append_agregate_element_value(null, kyou, emptyQuery)
    const str = await target.result_to_string(val)
    expect(str).toBe('7')
  })

  test('result_to_string with null returns 0', async () => {
    const str = await target.result_to_string(null)
    expect(str).toBe('0')
  })
})

// ========== Average Nlog Amount ==========

describe('AgregateAverageNlogAmount', () => {
  const target = new AgregateAverageNlogAmount()

  test('averages nlog amounts', async () => {
    const k1 = asKyou(makeKyouWithNlog('店', '品', 200))
    const k2 = asKyou(makeKyouWithNlog('店', '品', 800))
    let val = await target.append_agregate_element_value(null, k1, emptyQuery)
    val = await target.append_agregate_element_value(val, k2, emptyQuery)
    const str = await target.result_to_string(val)
    expect(str).toBe('500') // (200+800)/2
  })

  test('handles single element', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', '品', 300))
    const val = await target.append_agregate_element_value(null, kyou, emptyQuery)
    const str = await target.result_to_string(val)
    expect(str).toBe('300')
  })
})

// ========== Sum Git Commit Log Code Count ==========

describe('AgregateSumGitCommitLogCodeCount', () => {
  const target = new AgregateSumGitCommitLogCodeCount()

  test('sums code counts (addition - deletion)', async () => {
    const k1 = asKyou(makeKyouWithGitCommitLog('commit1', 10, 5))
    const k2 = asKyou(makeKyouWithGitCommitLog('commit2', 20, 3))
    let val = await target.append_agregate_element_value(null, k1, emptyQuery)
    val = await target.append_agregate_element_value(val, k2, emptyQuery)
    // (10-5) + (20-3) = 5 + 17 = 22
    expect(val).toBe(22)
  })
})
