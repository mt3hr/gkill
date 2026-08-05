/**
 * D-note Serialization Dictionary and round-trip tests.
 */
import register_dictionary, {
  build_dnote_predicate_from_json,
  build_dnote_aggregate_target_from_json,
} from '@/classes/dnote/serialize/register-dictionary'
import PredicateDictionary from '@/classes/dnote/serialize/dnote-predicate-dictionary'
import AggregateTargetDictionary from '@/classes/dnote/serialize/dnote-aggregate-target-dictionary'
import DnoteKeyGetterDictionary from '@/classes/dnote/serialize/dnote-key-getter-dictionary'
import DnoteKyouFilterDictionary from '@/classes/dnote/serialize/dnote-kyou-filter-dictionary'

// Ensure dictionaries are populated
beforeAll(() => {
  register_dictionary()
})

describe('dnote-predicate-dictionary', () => {
  test('all predicate type names resolve to constructors', () => {
    const expectedTypes = [
      'AndPredicate', 'OrPredicate', 'NotPredicate',
      'KmemoContentContainsPredicate', 'KmemoContentEqualPredicate',
      'NlogAmountGreaterThanPredicate', 'NlogAmountLessThanPredicate',
      'NlogShopContainsPredicate', 'NlogShopEqualPredicate',
      'NlogTitleContainsPredicate', 'NlogTitleEqualPredicate',
      'LantanaMoodEqualPredicate', 'LantanaMoodGreaterThanPredicate', 'LantanaMoodLessThanPredicate',
      'MiTitleContainsPredicate', 'MiTitleEqualPredicate',
      'TagEqualPredicate',
      'DataTypePrefixPredicate',
      'RelatedTimeBeforePredicate', 'RelatedTimeAfterPredicate',
      'TimeIsTitleContainsPredicate', 'TimeIsTitleEqualPredicate',
      'KCTitleContainsPredicate', 'KCTitleEqualPredicate',
    ]
    for (const type of expectedTypes) {
      expect(PredicateDictionary.has(type)).toBe(true)
    }
  })
})

describe('dnote-aggregate-target-dictionary', () => {
  test('all aggregate target type names resolve to constructors', () => {
    const expectedTypes = [
      'AggregateCountKyou',
      'AggregateSumNlogAmount', 'AggregateSumLantanaMood',
      'AggregateSumKCNumValue', 'AggregateSumTimeIsTime',
      'AggregateSumGitCommitLogCodeCount',
      'AggregateAverageLantanaMood', 'AggregateAverageNlogAmount',
    ]
    for (const type of expectedTypes) {
      expect(AggregateTargetDictionary.has(type)).toBe(true)
    }
  })
})

describe('dnote-key-getter-dictionary', () => {
  test('all key getter type names resolve to constructors', () => {
    const expectedTypes = [
      'DataTypeGetter', 'LantanaMoodGetter', 'NlogShopNameGetter',
      'RelatedMonthGetter', 'RelatedWeekDayGetter', 'RelatedWeekGetter',
      'RelatedDateGetter', 'TagGetter', 'TitleGetter',
    ]
    for (const type of expectedTypes) {
      expect(DnoteKeyGetterDictionary.has(type)).toBe(true)
    }
  })
})

describe('dnote-kyou-filter-dictionary', () => {
  test('all filter type names resolve to constructors', () => {
    expect(DnoteKyouFilterDictionary.has('FilterTopKyous')).toBe(true)
    expect(DnoteKyouFilterDictionary.has('FilterBottomKyous')).toBe(true)
  })
})

describe('predicate round-trip', () => {
  test('simple predicate survives JSON -> predicate -> JSON', () => {
    const json = { type: 'KmemoContentContainsPredicate', value: 'テスト' }
    const predicate = build_dnote_predicate_from_json(json)
    const output = predicate.predicate_struct_to_json()
    expect(output.type).toBe('KmemoContentContainsPredicate')
    expect(output.value).toBe('テスト')
  })

  test('logical AND predicate survives round-trip', () => {
    const json = {
      logic: 'AND',
      type: 'AndPredicate',
      predicates: [
        { type: 'KmemoContentContainsPredicate', value: 'A' },
        { type: 'DataTypePrefixPredicate', data_type_prefix: 'km' },
      ]
    }
    const predicate = build_dnote_predicate_from_json(json)
    const output = predicate.predicate_struct_to_json()
    expect(output.logic).toBe('AND')
    expect(output.predicates.length).toBe(2)
  })
})

describe('aggregate target backward compatibility (旧綴り Agregate*)', () => {
  // 保存済みの集計定義(user_config の DNOTE_JSON_DATA)には旧綴りの type 文字列が
  // 入っている。読み込みは新旧どちらも受け付け、書き出しは新綴りに寄せる。
  const legacy_types = [
    'AgregateAverageGitCommitLogAdditionCodeCount', 'AgregateAverageGitCommitLogCodeCount',
    'AgregateAverageGitCommitLogDeletionCodeCount', 'AgregateAverageLantanaMood',
    'AgregateAverageNlogAmount', 'AgregateAverageTimeIsEndTime', 'AgregateAverageTimeIsStartTime',
    'AgregateAverageTimeIsTime', 'AgregateCountKyou', 'AgregateSumGitCommitLogAdditionCodeCount',
    'AgregateSumGitCommitLogCodeCount', 'AgregateSumGitCommitLogDeletionCodeCount',
    'AgregateSumLantanaMood', 'AgregateSumNlogAmount', 'AgregateSumTimeIsTime',
    'AgregateAverageKCNumValue', 'AgregateMaxKCNumValue', 'AgregateMinKCNumValue',
    'AgregateSumKCNumValue',
  ]

  test('all legacy type names still resolve to constructors', () => {
    for (const type of legacy_types) {
      expect(AggregateTargetDictionary.has(type)).toBe(true)
    }
  })

  test('legacy json round-trips into the new spelling', () => {
    for (const type of legacy_types) {
      const target = build_dnote_aggregate_target_from_json({ type })
      const output = target.to_json()
      expect(output.type).toBe(type.replace('Agregate', 'Aggregate'))
    }
  })

  test('legacy and new names resolve to the same constructor', () => {
    for (const type of legacy_types) {
      const new_type = type.replace('Agregate', 'Aggregate')
      expect(AggregateTargetDictionary.get(type)).toBe(AggregateTargetDictionary.get(new_type))
    }
  })
})
