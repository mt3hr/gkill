/**
 * D-note Predicate tests.
 *
 * Predicates only access plain properties on Kyou objects (typed_kmemo?.content, etc.)
 * so we can use plain factory objects without needing real class instances.
 */
import {
  makeKyouWithKmemo,
  makeKyouWithNlog,
  makeKyouWithLantana,
  makeKyouWithKc,
  makeKyouWithMi,
  makeKyouWithTimeis,
  makeKyouWithTags,
  makeKyouWithGitCommitLog,
  makeKyou,
} from '../../helpers/factory'

import KmemoContentContainsPredicate from '@/classes/dnote/dnote-predicate/kmemo-content-contains-predicate'
import KmemoContentEqualPredicate from '@/classes/dnote/dnote-predicate/kmemo-content-equal-predicate'
import NlogAmountGreaterThanPredicate from '@/classes/dnote/dnote-predicate/nlog-amount-greater-than-predicate'
import NlogAmountLessThanPredicate from '@/classes/dnote/dnote-predicate/nlog-amount-less-than-predicate'
import NlogShopContainsPredicate from '@/classes/dnote/dnote-predicate/nlog-shop-contains-predicate'
import NlogShopEqualPredicate from '@/classes/dnote/dnote-predicate/nlog-shop-equal-predicate'
import NlogTitleContainsPredicate from '@/classes/dnote/dnote-predicate/nlog-title-contains-predicate'
import NlogTitleEqualPredicate from '@/classes/dnote/dnote-predicate/nlog-title-equal-predicate'
import LantanaMoodEqualPredicate from '@/classes/dnote/dnote-predicate/lantana-mood-equal-predicate'
import LantanaMoodGreaterThanPredicate from '@/classes/dnote/dnote-predicate/lantana-mood-greater-than-predicate'
import LantanaMoodLessThanPredicate from '@/classes/dnote/dnote-predicate/lantana-mood-less-than-predicate'
import MiTitleContainsPredicate from '@/classes/dnote/dnote-predicate/mi-title-contains-predicate'
import MiTitleEqualPredicate from '@/classes/dnote/dnote-predicate/mi-title-equal-predicate'
import TimeIsTitleContainsPredicate from '@/classes/dnote/dnote-predicate/timeis-title-contains-predicate'
import TimeIsTitleEqualPredicate from '@/classes/dnote/dnote-predicate/timeis-title-equal-predicate'
import KCTitleContainsPredicate from '@/classes/dnote/dnote-predicate/kc-title-contains-predicate'
import KCTitleEqualPredicate from '@/classes/dnote/dnote-predicate/kc-title-equal-predicate'
import TagEqualPredicate from '@/classes/dnote/dnote-predicate/tag-equal-predicate'
import DataTypePrefixPredicate from '@/classes/dnote/dnote-predicate/data-type-prefix-predicate'
import RelatedTimeBeforePredicate from '@/classes/dnote/dnote-predicate/related-time-before-predicate'
import RelatedTimeAfterPredicate from '@/classes/dnote/dnote-predicate/related-time-after-predicate'
import AndPredicate from '@/classes/dnote/dnote-predicate/and-predicate'
import OrPredicate from '@/classes/dnote/dnote-predicate/or-predicate'
import NotPredicate from '@/classes/dnote/dnote-predicate/not-predicate'

// Helper: cast plain objects as any for predicate calls
const asKyou = (obj: any) => obj

// ========== Kmemo Predicates ==========

describe('KmemoContentContainsPredicate', () => {
  const predicate = new KmemoContentContainsPredicate('テスト')

  test('matches when content contains target string', async () => {
    const kyou = asKyou(makeKyouWithKmemo('これはテストです'))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when content does not contain target', async () => {
    const kyou = asKyou(makeKyouWithKmemo('関係ないメモ'))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })

  test('does not match when typed_kmemo is null', async () => {
    const kyou = asKyou(makeKyou({ related_time: new Date() }))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })

  test('from_json / predicate_struct_to_json round-trip', () => {
    const json = predicate.predicate_struct_to_json()
    expect(json.type).toBe('KmemoContentContainsPredicate')
    expect(json.value).toBe('テスト')
    const restored = KmemoContentContainsPredicate.from_json(json)
    expect(restored).toBeInstanceOf(KmemoContentContainsPredicate)
  })
})

describe('KmemoContentEqualPredicate', () => {
  const predicate = new KmemoContentEqualPredicate('完全一致')

  test('matches on exact content', async () => {
    const kyou = asKyou(makeKyouWithKmemo('完全一致'))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match on partial content', async () => {
    const kyou = asKyou(makeKyouWithKmemo('完全一致ではない'))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

// ========== Nlog Predicates ==========

describe('NlogAmountGreaterThanPredicate', () => {
  const predicate = new NlogAmountGreaterThanPredicate(500)

  test('matches when amount exceeds threshold', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', '品物', 1000))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when amount below threshold', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', '品物', 100))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

describe('NlogAmountLessThanPredicate', () => {
  const predicate = new NlogAmountLessThanPredicate(500)

  test('matches when amount below threshold', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', '品物', 100))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when amount above threshold', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', '品物', 1000))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

describe('NlogShopContainsPredicate', () => {
  const predicate = new NlogShopContainsPredicate('コンビニ')

  test('matches on partial shop name', async () => {
    const kyou = asKyou(makeKyouWithNlog('近所のコンビニ', '弁当', 500))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when shop name differs', async () => {
    const kyou = asKyou(makeKyouWithNlog('スーパー', '弁当', 500))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

describe('NlogShopEqualPredicate', () => {
  const predicate = new NlogShopEqualPredicate('コンビニ')

  test('matches on exact shop name', async () => {
    const kyou = asKyou(makeKyouWithNlog('コンビニ', '弁当', 500))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match on partial match', async () => {
    const kyou = asKyou(makeKyouWithNlog('近所のコンビニ', '弁当', 500))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

describe('NlogTitleContainsPredicate', () => {
  const predicate = new NlogTitleContainsPredicate('弁当')

  test('matches on partial title', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', 'のり弁当', 500))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when title differs', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', 'おにぎり', 500))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

describe('NlogTitleEqualPredicate', () => {
  const predicate = new NlogTitleEqualPredicate('弁当')

  test('matches on exact title', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', '弁当', 500))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match on partial match', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', 'のり弁当', 500))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

// ========== Lantana Predicates ==========

describe('LantanaMoodEqualPredicate', () => {
  const predicate = new LantanaMoodEqualPredicate(7)

  test('matches on exact mood value', async () => {
    const kyou = asKyou(makeKyouWithLantana(7))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match on different mood', async () => {
    const kyou = asKyou(makeKyouWithLantana(5))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

describe('LantanaMoodGreaterThanPredicate', () => {
  // Note: implementation checks mood <= target (inverted logic from name)
  const predicate = new LantanaMoodGreaterThanPredicate(5)

  test('matches when mood is at or below target', async () => {
    const kyou = asKyou(makeKyouWithLantana(3))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when mood above target', async () => {
    const kyou = asKyou(makeKyouWithLantana(8))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

describe('LantanaMoodLessThanPredicate', () => {
  const predicate = new LantanaMoodLessThanPredicate(5)

  test('matches when mood below threshold', async () => {
    const kyou = asKyou(makeKyouWithLantana(3))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when mood above threshold', async () => {
    const kyou = asKyou(makeKyouWithLantana(8))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

// ========== Mi Predicates ==========

describe('MiTitleContainsPredicate', () => {
  const predicate = new MiTitleContainsPredicate('タスク')

  test('matches on partial title', async () => {
    const kyou = asKyou(makeKyouWithMi('重要なタスク'))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when title differs', async () => {
    const kyou = asKyou(makeKyouWithMi('メモ'))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

describe('MiTitleEqualPredicate', () => {
  const predicate = new MiTitleEqualPredicate('タスク')

  test('matches on exact title', async () => {
    const kyou = asKyou(makeKyouWithMi('タスク'))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match on partial match', async () => {
    const kyou = asKyou(makeKyouWithMi('重要なタスク'))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

// ========== TimeIs Predicates ==========

describe('TimeIsTitleContainsPredicate', () => {
  const predicate = new TimeIsTitleContainsPredicate('会議')

  test('matches on partial title', async () => {
    const kyou = asKyou(makeKyouWithTimeis('朝会議'))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when title differs', async () => {
    const kyou = asKyou(makeKyouWithTimeis('作業'))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

describe('TimeIsTitleEqualPredicate', () => {
  const predicate = new TimeIsTitleEqualPredicate('会議')

  test('matches on exact title', async () => {
    const kyou = asKyou(makeKyouWithTimeis('会議'))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match on partial match', async () => {
    const kyou = asKyou(makeKyouWithTimeis('朝会議'))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

// ========== KC Predicates ==========

describe('KCTitleContainsPredicate', () => {
  const predicate = new KCTitleContainsPredicate('歩数')

  test('matches on partial title', async () => {
    const kyou = asKyou(makeKyouWithKc('今日の歩数'))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when title differs', async () => {
    const kyou = asKyou(makeKyouWithKc('体重'))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

describe('KCTitleEqualPredicate', () => {
  const predicate = new KCTitleEqualPredicate('歩数')

  test('matches on exact title', async () => {
    const kyou = asKyou(makeKyouWithKc('歩数'))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match on partial match', async () => {
    const kyou = asKyou(makeKyouWithKc('今日の歩数'))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

// ========== Tag Predicate ==========

describe('TagEqualPredicate', () => {
  const predicate = new TagEqualPredicate('重要')

  test('matches when kyou has matching tag', async () => {
    const kyou = asKyou(makeKyouWithTags(['重要', 'メモ']))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when tag absent', async () => {
    const kyou = asKyou(makeKyouWithTags(['メモ', '日常']))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

// ========== DataType Predicate ==========

describe('DataTypePrefixPredicate', () => {
  const predicate = new DataTypePrefixPredicate('km')

  test('matches when data_type starts with prefix', async () => {
    const kyou = asKyou(makeKyouWithKmemo('test'))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when prefix differs', async () => {
    const kyou = asKyou(makeKyouWithNlog('店', '品', 100))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

// ========== RelatedTime Predicates ==========

describe('RelatedTimeBeforePredicate', () => {
  const cutoff = new Date('2025-06-01T00:00:00Z')
  const predicate = new RelatedTimeBeforePredicate(cutoff)

  test('matches when related_time before cutoff', async () => {
    const kyou = asKyou(makeKyouWithKmemo('test', { related_time: new Date('2025-03-01T00:00:00Z') }))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when related_time after cutoff', async () => {
    const kyou = asKyou(makeKyouWithKmemo('test', { related_time: new Date('2025-09-01T00:00:00Z') }))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

describe('RelatedTimeAfterPredicate', () => {
  const cutoff = new Date('2025-06-01T00:00:00Z')
  const predicate = new RelatedTimeAfterPredicate(cutoff)

  test('matches when related_time after cutoff', async () => {
    const kyou = asKyou(makeKyouWithKmemo('test', { related_time: new Date('2025-09-01T00:00:00Z') }))
    expect(await predicate.is_match(kyou, null)).toBe(true)
  })

  test('does not match when related_time before cutoff', async () => {
    const kyou = asKyou(makeKyouWithKmemo('test', { related_time: new Date('2025-03-01T00:00:00Z') }))
    expect(await predicate.is_match(kyou, null)).toBe(false)
  })
})

// ========== Logical Predicates ==========

describe('AndPredicate', () => {
  test('true only when both children match', async () => {
    const p1 = new KmemoContentContainsPredicate('テスト')
    const p2 = new DataTypePrefixPredicate('km')
    const and = new AndPredicate([p1, p2])
    const kyou = asKyou(makeKyouWithKmemo('テストメモ'))
    expect(await and.is_match(kyou, null)).toBe(true)
  })

  test('false when one child fails', async () => {
    const p1 = new KmemoContentContainsPredicate('テスト')
    const p2 = new DataTypePrefixPredicate('nl') // nlog prefix, won't match kmemo
    const and = new AndPredicate([p1, p2])
    const kyou = asKyou(makeKyouWithKmemo('テストメモ'))
    expect(await and.is_match(kyou, null)).toBe(false)
  })

  test('true when predicates array is empty', async () => {
    const and = new AndPredicate([])
    const kyou = asKyou(makeKyouWithKmemo('test'))
    expect(await and.is_match(kyou, null)).toBe(true)
  })
})

describe('OrPredicate', () => {
  test('true when either child matches', async () => {
    const p1 = new KmemoContentContainsPredicate('テスト')
    const p2 = new DataTypePrefixPredicate('nl') // won't match
    const or = new OrPredicate([p1, p2])
    const kyou = asKyou(makeKyouWithKmemo('テストメモ'))
    expect(await or.is_match(kyou, null)).toBe(true)
  })

  test('false when both fail', async () => {
    const p1 = new KmemoContentContainsPredicate('存在しない')
    const p2 = new DataTypePrefixPredicate('nl')
    const or = new OrPredicate([p1, p2])
    const kyou = asKyou(makeKyouWithKmemo('テストメモ'))
    expect(await or.is_match(kyou, null)).toBe(false)
  })

  test('true when predicates array is empty', async () => {
    const or = new OrPredicate([])
    const kyou = asKyou(makeKyouWithKmemo('test'))
    expect(await or.is_match(kyou, null)).toBe(true)
  })
})

describe('NotPredicate', () => {
  // Note: current NotPredicate implementation uses OR logic internally
  // (returns true if ANY child matches). Testing as-is.
  test('returns true when any child matches', async () => {
    const p1 = new KmemoContentContainsPredicate('テスト')
    const not = new NotPredicate([p1])
    const kyou = asKyou(makeKyouWithKmemo('テストメモ'))
    expect(await not.is_match(kyou, null)).toBe(true)
  })

  test('returns false when no child matches', async () => {
    const p1 = new KmemoContentContainsPredicate('存在しない')
    const not = new NotPredicate([p1])
    const kyou = asKyou(makeKyouWithKmemo('テストメモ'))
    expect(await not.is_match(kyou, null)).toBe(false)
  })
})
