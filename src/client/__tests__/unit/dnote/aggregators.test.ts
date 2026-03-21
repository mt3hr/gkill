/**
 * D-note Aggregator and ListAggregator tests.
 *
 * DnoteAgregator and DnoteListAggregator use load_kyous() internally,
 * which calls kyou.clone(), kyou.load_typed_datas(), etc.
 * We mock load_kyous to return plain objects directly.
 */
import { vi } from 'vitest'
import {
  makeKyouWithKmemo,
  makeKyouWithNlog,
  makeKyouWithLantana,
  makeKyouWithTags,
} from '../../helpers/factory'

// Mock load_kyous before importing aggregators
vi.mock('@/classes/dnote/kyou-loader', () => ({
  default: vi.fn(async (_abort: any, kyous: any[]) => kyous),
}))

import { DnoteAgregator } from '@/classes/dnote/dnote-aggregator'
import { DnoteListAggregator } from '@/classes/dnote/dnote-list-aggregator'
import KmemoContentContainsPredicate from '@/classes/dnote/dnote-predicate/kmemo-content-contains-predicate'
import DataTypePrefixPredicate from '@/classes/dnote/dnote-predicate/data-type-prefix-predicate'
import AgregateCountKyou from '@/classes/dnote/dnote-agregate-target/agregate-count-kyou'
import AgregateSumNlogAmount from '@/classes/dnote/dnote-agregate-target/agregate-sum-nlog-amount'
import TagGetter from '@/classes/dnote/dnote-key-getter/tag-getter'
import DataTypeGetter from '@/classes/dnote/dnote-key-getter/data-type-getter'

const asKyou = (obj: any) => obj
const emptyQuery = {} as any
const controller = new AbortController()

// Kyous used in tests need clone() for the aggregator's cloned_match_kyous
function makeTestKyou(factory: (...args: any[]) => any, ...args: any[]) {
  const obj = factory(...args)
  obj.clone = () => ({ ...obj })
  return asKyou(obj)
}

describe('DnoteAgregator', () => {
  test('filters by predicate and aggregates matching kyous', async () => {
    const predicate = new KmemoContentContainsPredicate('テスト')
    const target = new AgregateCountKyou()
    const aggregator = new DnoteAgregator(predicate, target)

    const kyous = [
      makeTestKyou(makeKyouWithKmemo, 'テストメモ1'),
      makeTestKyou(makeKyouWithKmemo, '関係ない'),
      makeTestKyou(makeKyouWithKmemo, 'テストメモ2'),
    ]

    const result = await aggregator.agregate(controller, kyous, emptyQuery, true)
    expect(result.result_string).toBe('2')
    expect(result.match_kyous.length).toBe(2)
  })

  test('returns empty match_kyous when nothing matches', async () => {
    const predicate = new KmemoContentContainsPredicate('存在しない')
    const target = new AgregateCountKyou()
    const aggregator = new DnoteAgregator(predicate, target)

    const kyous = [
      makeTestKyou(makeKyouWithKmemo, 'メモA'),
      makeTestKyou(makeKyouWithKmemo, 'メモB'),
    ]

    const result = await aggregator.agregate(controller, kyous, emptyQuery, true)
    expect(result.result_string).toBe('0')
    expect(result.match_kyous.length).toBe(0)
  })

  test('match_kyous are clones (not reference-equal)', async () => {
    const predicate = new KmemoContentContainsPredicate('テスト')
    const target = new AgregateCountKyou()
    const aggregator = new DnoteAgregator(predicate, target)

    const original = makeTestKyou(makeKyouWithKmemo, 'テスト')
    const kyous = [original]

    const result = await aggregator.agregate(controller, kyous, emptyQuery, true)
    expect(result.match_kyous[0]).not.toBe(original)
  })
})

describe('DnoteListAggregator', () => {
  test('groups by key and aggregates per group', async () => {
    const predicate = new DataTypePrefixPredicate('nlog')
    const keyGetter = new DataTypeGetter()
    const target = new AgregateSumNlogAmount()
    const aggregator = new DnoteListAggregator(predicate, keyGetter, target)

    const kyous = [
      makeTestKyou(makeKyouWithNlog, '店A', '品', 300),
      makeTestKyou(makeKyouWithNlog, '店B', '品', 700),
    ]

    const result = await aggregator.aggregate_grouping_list(controller, kyous, emptyQuery, true)
    expect(result.length).toBe(1) // all same data_type 'nlog'
    expect(result[0].title).toBe('nlog')
    expect(result[0].value).toBe('1000')
  })

  test('returns empty array when no matches', async () => {
    const predicate = new DataTypePrefixPredicate('nonexistent')
    const keyGetter = new DataTypeGetter()
    const target = new AgregateCountKyou()
    const aggregator = new DnoteListAggregator(predicate, keyGetter, target)

    const kyous = [makeTestKyou(makeKyouWithKmemo, 'test')]
    const result = await aggregator.aggregate_grouping_list(controller, kyous, emptyQuery, true)
    expect(result.length).toBe(0)
  })

  test('handles multiple keys per kyou (e.g. multiple tags)', async () => {
    const predicate = new DataTypePrefixPredicate('kmemo')
    const keyGetter = new TagGetter()
    const target = new AgregateCountKyou()
    const aggregator = new DnoteListAggregator(predicate, keyGetter, target)

    const kyou = makeTestKyou(makeKyouWithKmemo, 'test', {
      attached_tags: [
        { tag: 'tagA', id: 'a', target_id: 't' },
        { tag: 'tagB', id: 'b', target_id: 't' },
      ],
    })
    const kyous = [kyou]

    const result = await aggregator.aggregate_grouping_list(controller, kyous, emptyQuery, true)
    expect(result.length).toBe(2) // one per tag
    const titles = result.map((r: any) => r.title).sort()
    expect(titles).toEqual(['tagA', 'tagB'])
  })

  test('converts aggregate values to string in final output', async () => {
    const predicate = new DataTypePrefixPredicate('nlog')
    const keyGetter = new DataTypeGetter()
    const target = new AgregateSumNlogAmount()
    const aggregator = new DnoteListAggregator(predicate, keyGetter, target)

    const kyous = [makeTestKyou(makeKyouWithNlog, '店', '品', 500)]
    const result = await aggregator.aggregate_grouping_list(controller, kyous, emptyQuery, true)
    expect(typeof result[0].value).toBe('string')
  })
})
