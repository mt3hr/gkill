/**
 * D-note Key Getter tests.
 *
 * Key getters extract grouping keys from Kyou objects.
 * related_time must be a Date object for date-based getters.
 */
import {
  makeKyouWithKmemo,
  makeKyouWithNlog,
  makeKyouWithLantana,
  makeKyouWithMi,
  makeKyouWithTags,
  makeKyou,
} from '../../helpers/factory'

import TagGetter from '@/classes/dnote/dnote-key-getter/tag-getter'
import DataTypeGetter from '@/classes/dnote/dnote-key-getter/data-type-getter'
import RelatedMonthGetter from '@/classes/dnote/dnote-key-getter/related-month-getter'
import RelatedWeekDayGetter from '@/classes/dnote/dnote-key-getter/related-week-day-getter'
import RelatedDateGetter from '@/classes/dnote/dnote-key-getter/rerated-date-getter'
import NlogShopNameGetter from '@/classes/dnote/dnote-key-getter/nlog-shop-name-getter'
import LantanaMoodGetter from '@/classes/dnote/dnote-key-getter/lantana-mood-getter'
import TitleGetter from '@/classes/dnote/dnote-key-getter/title-getter'

const asKyou = (obj: any) => obj

describe('TagGetter', () => {
  const getter = new TagGetter()

  test('returns tag strings from attached_tags', () => {
    const kyou = asKyou(makeKyouWithTags(['重要', 'メモ']))
    const keys = getter.get_keys(kyou)
    expect(keys).toEqual(['重要', 'メモ'])
  })

  test('returns empty array when no tags', () => {
    const kyou = asKyou(makeKyouWithTags([]))
    const keys = getter.get_keys(kyou)
    expect(keys).toEqual([])
  })

  test('to_json returns correct type', () => {
    expect(getter.to_json().type).toBe('TagGetter')
  })
})

describe('DataTypeGetter', () => {
  const getter = new DataTypeGetter()

  test('returns data_type as key', () => {
    const kyou = asKyou(makeKyouWithKmemo('test'))
    const keys = getter.get_keys(kyou)
    expect(keys).toEqual(['kmemo'])
  })
})

describe('RelatedMonthGetter', () => {
  const getter = new RelatedMonthGetter()

  test('returns YYYY/MM formatted month', () => {
    const kyou = asKyou(makeKyouWithKmemo('test', { related_time: new Date(2025, 2, 15) })) // March (0-indexed month 2)
    const keys = getter.get_keys(kyou)
    expect(keys).toEqual(['2025/03'])
  })
})

describe('RelatedWeekDayGetter', () => {
  const getter = new RelatedWeekDayGetter()

  test('returns day-of-week', () => {
    // 2025-03-15 is Saturday
    const kyou = asKyou(makeKyouWithKmemo('test', { related_time: new Date(2025, 2, 15) }))
    const keys = getter.get_keys(kyou)
    expect(keys.length).toBe(1)
    expect(typeof keys[0]).toBe('string')
  })
})

describe('RelatedDateGetter', () => {
  const getter = new RelatedDateGetter()

  test('returns YYYY/MM/DD formatted date', () => {
    const kyou = asKyou(makeKyouWithKmemo('test', { related_time: new Date(2025, 2, 15) }))
    const keys = getter.get_keys(kyou)
    expect(keys).toEqual(['2025/03/15'])
  })
})

describe('NlogShopNameGetter', () => {
  const getter = new NlogShopNameGetter()

  test('returns shop name from typed_nlog', () => {
    const kyou = asKyou(makeKyouWithNlog('コンビニ', '弁当', 500))
    const keys = getter.get_keys(kyou)
    expect(keys).toEqual(['コンビニ'])
  })

  test('returns empty array when no nlog', () => {
    const kyou = asKyou(makeKyouWithKmemo('test'))
    const keys = getter.get_keys(kyou)
    expect(keys).toEqual([])
  })
})

describe('LantanaMoodGetter', () => {
  const getter = new LantanaMoodGetter()

  test('returns mood value as string key', () => {
    const kyou = asKyou(makeKyouWithLantana(7))
    const keys = getter.get_keys(kyou)
    expect(keys).toEqual(['7'])
  })
})

describe('TitleGetter', () => {
  const getter = new TitleGetter()

  test('returns title for mi', () => {
    const kyou = asKyou(makeKyouWithMi('重要タスク'))
    const keys = getter.get_keys(kyou)
    expect(keys).toEqual(['重要タスク'])
  })

  test('returns content for kmemo', () => {
    const kyou = asKyou(makeKyouWithKmemo('メモ内容'))
    const keys = getter.get_keys(kyou)
    expect(keys).toEqual(['メモ内容'])
  })
})
