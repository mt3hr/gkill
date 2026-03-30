/**
 * D-note Serialization Dictionary and round-trip tests.
 */
import regist_dictionary, {
  build_dnote_predicate_from_json,
} from '@/classes/dnote/serialize/regist-dictionary'
import PredicateDictionary from '@/classes/dnote/serialize/dnote-predicate-dictionary'
import AgregateTargetDictionary from '@/classes/dnote/serialize/dnote-aggregate-target-dictionary'
import DnoteKeyGetterDictionary from '@/classes/dnote/serialize/dnote-key-getter-dictionary'
import DnoteKyouFilterDictionary from '@/classes/dnote/serialize/dnote-kyou-filter-dictionary'

// Ensure dictionaries are populated
beforeAll(() => {
  regist_dictionary()
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
      'AgregateCountKyou',
      'AgregateSumNlogAmount', 'AgregateSumLantanaMood',
      'AgregateSumKCNumValue', 'AgregateSumTimeIsTime',
      'AgregateSumGitCommitLogCodeCount',
      'AgregateAverageLantanaMood', 'AgregateAverageNlogAmount',
    ]
    for (const type of expectedTypes) {
      expect(AgregateTargetDictionary.has(type)).toBe(true)
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
