import type DnoteAggregateTarget from "../dnote-aggregate-target"
import AggregateAverageGitCommitLogAdditionCodeCount from "../dnote-aggregate-target/aggregate-average-git-commit-log-code-addition-count"
import AggregateAverageGitCommitLogCodeCount from "../dnote-aggregate-target/aggregate-average-git-commit-log-code-count"
import AggregateAverageGitCommitLogDeletionCodeCount from "../dnote-aggregate-target/aggregate-average-git-commit-log-code-deletion-count"
import AggregateAverageLantanaMood from "../dnote-aggregate-target/aggregate-average-lantana-mood"
import AggregateAverageNlogAmount from "../dnote-aggregate-target/aggregate-average-nlog-amount"
import AggregateAverageTimeIsEndTime from "../dnote-aggregate-target/aggregate-average-timeis-end-time"
import AggregateAverageTimeIsStartTime from "../dnote-aggregate-target/aggregate-average-timeis-start-time"
import AggregateAverageTimeIsTime from "../dnote-aggregate-target/aggregate-average-timeis-time"
import AggregateCountKyou from "../dnote-aggregate-target/aggregate-count-kyou"
import AggregateSumGitCommitLogAdditionCodeCount from "../dnote-aggregate-target/aggregate-sum-git-commit-log-code-addition-count"
import AggregateSumGitCommitLogCodeCount from "../dnote-aggregate-target/aggregate-sum-git-commit-log-code-count"
import AggregateSumGitCommitLogDeletionCodeCount from "../dnote-aggregate-target/aggregate-sum-git-commit-log-code-deletion-count"
import AggregateSumLantanaMood from "../dnote-aggregate-target/aggregate-sum-lantana-mood"
import AggregateSumNlogAmount from "../dnote-aggregate-target/aggregate-sum-nlog-amount"
import AggregateSumTimeIsTime from "../dnote-aggregate-target/aggregate-sum-timeis-time"
import AggregateSumKCNumValue from "../dnote-aggregate-target/aggregate-sum-kc-num-value"
import AggregateAverageKCNumValue from "../dnote-aggregate-target/aggregate-average-kc-num-value"
import AggregateMaxKCNumValue from "../dnote-aggregate-target/aggregate-max-kc-num-value"
import AggregateMinKCNumValue from "../dnote-aggregate-target/aggregate-min-kc-num-value"
import type DnoteKeyGetter from "../dnote-key-getter"
import DataTypeGetter from "../dnote-key-getter/data-type-getter"
import LantanaMoodGetter from "../dnote-key-getter/lantana-mood-getter"
import NlogShopNameGetter from "../dnote-key-getter/nlog-shop-name-getter"
import RelatedMonthGetter from "../dnote-key-getter/related-month-getter"
import RelatedWeekDayGetter from "../dnote-key-getter/related-week-day-getter"
import RelatedWeekGetter from "../dnote-key-getter/related-week-getter"
import RelatedDateGetter from "../dnote-key-getter/related-date-getter"
import TagGetter from "../dnote-key-getter/tag-getter"
import TitleGetter from "../dnote-key-getter/title-getter"
import type DnotePredicate from "../dnote-predicate"
import AndPredicate from "../dnote-predicate/and-predicate"
import DataTypePrefixPredicate from "../dnote-predicate/data-type-prefix-predicate"
import GitCommitLogCodeAdditionGreaterThanPredicate from "../dnote-predicate/git-commit-log-code-addition-greater-than-predicate"
import GitCommitLogCodeAdditionLessThanPredicate from "../dnote-predicate/git-commit-log-code-addition-less-than-predicate"
import GitCommitLogCodeDeletionGreaterThanPredicate from "../dnote-predicate/git-commit-log-code-deletion-greater-than-predicate"
import GitCommitLogCodeDeletionLessThanPredicate from "../dnote-predicate/git-commit-log-code-deletion-less-than-predicate"
import GitCommitLogCodeGreaterThanPredicate from "../dnote-predicate/git-commit-log-code-greater-than-predicate"
import GitCommitLogCodeLessThanPredicate from "../dnote-predicate/git-commit-log-code-less-than-predicate"
import KmemoContentContainsPredicate from "../dnote-predicate/kmemo-content-contains-predicate"
import KmemoContentEqualPredicate from "../dnote-predicate/kmemo-content-equal-predicate"
import LantanaMoodEqualPredicate from "../dnote-predicate/lantana-mood-equal-predicate"
import LantanaMoodGreaterThanPredicate from "../dnote-predicate/lantana-mood-greater-than-predicate"
import LantanaMoodLessThanPredicate from "../dnote-predicate/lantana-mood-less-than-predicate"
import MiTitleContainsPredicate from "../dnote-predicate/mi-title-contains-predicate"
import MiTitleEqualPredicate from "../dnote-predicate/mi-title-equal-predicate"
import NlogAmountGreaterThanPredicate from "../dnote-predicate/nlog-amount-greater-than-predicate"
import NlogAmountLessThanPredicate from "../dnote-predicate/nlog-amount-less-than-predicate"
import NlogShopContainsPredicate from "../dnote-predicate/nlog-shop-contains-predicate"
import NlogShopEqualPredicate from "../dnote-predicate/nlog-shop-equal-predicate"
import NlogTitleContainsPredicate from "../dnote-predicate/nlog-title-contains-predicate"
import NlogTitleEqualPredicate from "../dnote-predicate/nlog-title-equal-predicate"
import NotPredicate from "../dnote-predicate/not-predicate"
import OrPredicate from "../dnote-predicate/or-predicate"
import RelatedTimeWeekPredicate from "../dnote-predicate/related-time-week-predicate"
import TagEqualPredicate from "../dnote-predicate/tag-equal-predicate"
import TimeIsTitleContainsPredicate from "../dnote-predicate/timeis-title-contains-predicate"
import TimeIsTitleEqualPredicate from "../dnote-predicate/timeis-title-equal-predicate"
import AggregateTargetDictionary from "./dnote-aggregate-target-dictionary"
import DnoteKeyGetterDictionary from "./dnote-key-getter-dictionary"
import PredicateDictionary from "./dnote-predicate-dictionary"
import RelatedTimeAfterPredicate from "../dnote-predicate/related-time-after-predicate"
import RelatedTimeBeforePredicate from "../dnote-predicate/related-time-before-predicate"
import type DnoteKyouFilter from "../dnote-kyou-filter"
import DnoteKyouFilterDictionary from "./dnote-kyou-filter-dictionary"
import FilterBottomKyous from "../dnote-filter/filter-bottom-kyous"
import FilterTopKyous from "../dnote-filter/filter-top-kyous"
import KCTitleContainsPredicate from "../dnote-predicate/kc-title-contains-predicate"
import KCTitleEqualPredicate from "../dnote-predicate/kc-title-equal-predicate"
import EqualTitleTargetKyouPredicate from "../dnote-predicate/target-kyou-predicate/equal-title-target-kyou-predicate"
import EqualTagsAndTargetKyouPredicate from "../dnote-predicate/target-kyou-predicate/equal-tags-and-target-kyou-predicate"
import EqualTagsOrTargetKyouPredicate from "../dnote-predicate/target-kyou-predicate/equal-tags-or-target-kyou-predicate"
import EqualDataTypeTargetKyouPredicate from "../dnote-predicate/target-kyou-predicate/equal-rep-data-type-target-kyou-predicate"
import EqualIdTargetKyouPredicate from "../dnote-predicate/target-kyou-predicate/equal-id-target-kyou-predicate"

export default function register_dictionary(): void {
    PredicateDictionary.set("AndPredicate", AndPredicate)
    PredicateDictionary.set("DataTypePrefixPredicate", DataTypePrefixPredicate)
    PredicateDictionary.set("GitCommitLogCodeAdditionGreaterThanPredicate", GitCommitLogCodeAdditionGreaterThanPredicate)
    PredicateDictionary.set("GitCommitLogCodeAdditionLessThanPredicate", GitCommitLogCodeAdditionLessThanPredicate)
    PredicateDictionary.set("GitCommitLogCodeDeletionGreaterThanPredicate", GitCommitLogCodeDeletionGreaterThanPredicate)
    PredicateDictionary.set("GitCommitLogCodeDeletionLessThanPredicate", GitCommitLogCodeDeletionLessThanPredicate)
    PredicateDictionary.set("GitCommitLogCodeGreaterThanPredicate", GitCommitLogCodeGreaterThanPredicate)
    PredicateDictionary.set("GitCommitLogCodeLessThanPredicate", GitCommitLogCodeLessThanPredicate)
    PredicateDictionary.set("KmemoContentContainsPredicate", KmemoContentContainsPredicate)
    PredicateDictionary.set("KmemoContentEqualPredicate", KmemoContentEqualPredicate)
    PredicateDictionary.set("TextContentContainsPredicate", KmemoContentContainsPredicate)
    PredicateDictionary.set("TextContentEqualPredicate", KmemoContentEqualPredicate)
    PredicateDictionary.set("LantanaMoodEqualPredicate", LantanaMoodEqualPredicate)
    PredicateDictionary.set("LantanaMoodGreaterThanPredicate", LantanaMoodGreaterThanPredicate)
    PredicateDictionary.set("LantanaMoodLessThanPredicate", LantanaMoodLessThanPredicate)
    PredicateDictionary.set("MiTitleContainsPredicate", MiTitleContainsPredicate)
    PredicateDictionary.set("MiTitleEqualPredicate", MiTitleEqualPredicate)
    PredicateDictionary.set("NlogAmountGreaterThanPredicate", NlogAmountGreaterThanPredicate)
    PredicateDictionary.set("NlogAmountLessThanPredicate", NlogAmountLessThanPredicate)
    PredicateDictionary.set("NlogShopContainsPredicate", NlogShopContainsPredicate)
    PredicateDictionary.set("NlogShopEqualPredicate", NlogShopEqualPredicate)
    PredicateDictionary.set("NlogTitleContainsPredicate", NlogTitleContainsPredicate)
    PredicateDictionary.set("NlogTitleEqualPredicate", NlogTitleEqualPredicate)
    PredicateDictionary.set("NotPredicate", NotPredicate)
    PredicateDictionary.set("OrPredicate", OrPredicate)
    PredicateDictionary.set("RelatedTimeWeekPredicate", RelatedTimeWeekPredicate)
    PredicateDictionary.set("TagEqualPredicate", TagEqualPredicate)
    PredicateDictionary.set("TimeIsTitleContainsPredicate", TimeIsTitleContainsPredicate)
    PredicateDictionary.set("TimeIsTitleEqualPredicate", TimeIsTitleEqualPredicate)
    PredicateDictionary.set("KCTitleContainsPredicate", KCTitleContainsPredicate)
    PredicateDictionary.set("KCTitleEqualPredicate", KCTitleEqualPredicate)
    PredicateDictionary.set("RelatedTimeAfterPredicate", RelatedTimeAfterPredicate)
    PredicateDictionary.set("RelatedTimeBeforePredicate", RelatedTimeBeforePredicate)
    PredicateDictionary.set("EqualTitleTargetKyouPredicate", EqualTitleTargetKyouPredicate)
    PredicateDictionary.set("EqualTagsAndTargetKyouPredicate", EqualTagsAndTargetKyouPredicate)
    PredicateDictionary.set("EqualTagsOrTargetKyouPredicate", EqualTagsOrTargetKyouPredicate)
    PredicateDictionary.set("EqualDataTypeTargetKyouPredicate", EqualDataTypeTargetKyouPredicate)
    PredicateDictionary.set("EqualIdTargetKyouPredicate", EqualIdTargetKyouPredicate)
    DnoteKeyGetterDictionary.set("DataTypeGetter", DataTypeGetter)
    DnoteKeyGetterDictionary.set("LantanaMoodGetter", LantanaMoodGetter)
    DnoteKeyGetterDictionary.set("NlogShopNameGetter", NlogShopNameGetter)
    DnoteKeyGetterDictionary.set("RelatedMonthGetter", RelatedMonthGetter)
    DnoteKeyGetterDictionary.set("RelatedWeekDayGetter", RelatedWeekDayGetter)
    DnoteKeyGetterDictionary.set("RelatedWeekGetter", RelatedWeekGetter)
    DnoteKeyGetterDictionary.set("RelatedDateGetter", RelatedDateGetter)
    DnoteKeyGetterDictionary.set("TagGetter", TagGetter)
    DnoteKeyGetterDictionary.set("TitleGetter", TitleGetter)
    AggregateTargetDictionary.set("AggregateAverageGitCommitLogAdditionCodeCount", AggregateAverageGitCommitLogAdditionCodeCount)
    AggregateTargetDictionary.set("AggregateAverageGitCommitLogCodeCount", AggregateAverageGitCommitLogCodeCount)
    AggregateTargetDictionary.set("AggregateAverageGitCommitLogDeletionCodeCount", AggregateAverageGitCommitLogDeletionCodeCount)
    AggregateTargetDictionary.set("AggregateAverageLantanaMood", AggregateAverageLantanaMood)
    AggregateTargetDictionary.set("AggregateAverageNlogAmount", AggregateAverageNlogAmount)
    AggregateTargetDictionary.set("AggregateAverageTimeIsEndTime", AggregateAverageTimeIsEndTime)
    AggregateTargetDictionary.set("AggregateAverageTimeIsStartTime", AggregateAverageTimeIsStartTime)
    AggregateTargetDictionary.set("AggregateAverageTimeIsTime", AggregateAverageTimeIsTime)
    AggregateTargetDictionary.set("AggregateCountKyou", AggregateCountKyou)
    AggregateTargetDictionary.set("AggregateSumGitCommitLogAdditionCodeCount", AggregateSumGitCommitLogAdditionCodeCount)
    AggregateTargetDictionary.set("AggregateSumGitCommitLogCodeCount", AggregateSumGitCommitLogCodeCount)
    AggregateTargetDictionary.set("AggregateSumGitCommitLogDeletionCodeCount", AggregateSumGitCommitLogDeletionCodeCount)
    AggregateTargetDictionary.set("AggregateSumLantanaMood", AggregateSumLantanaMood)
    AggregateTargetDictionary.set("AggregateSumNlogAmount", AggregateSumNlogAmount)
    AggregateTargetDictionary.set("AggregateSumTimeIsTime", AggregateSumTimeIsTime)
    AggregateTargetDictionary.set("AggregateAverageKCNumValue", AggregateAverageKCNumValue)
    AggregateTargetDictionary.set("AggregateMaxKCNumValue", AggregateMaxKCNumValue)
    AggregateTargetDictionary.set("AggregateMinKCNumValue", AggregateMinKCNumValue)
    AggregateTargetDictionary.set("AggregateSumKCNumValue", AggregateSumKCNumValue)
    DnoteKyouFilterDictionary.set("FilterTopKyous", FilterTopKyous)
    DnoteKyouFilterDictionary.set("FilterBottomKyous", FilterBottomKyous)
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function build_dnote_aggregate_target_from_json(json: any): DnoteAggregateTarget {
    register_dictionary()
    const ctor = AggregateTargetDictionary.get(json.type)
    if (!ctor) throw new Error(`Unknown aggregate type: ${json.type}`)
    return ctor.from_json(json) as DnoteAggregateTarget
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function build_dnote_key_getter_from_json(json: any): DnoteKeyGetter {
    register_dictionary()
    const ctor = DnoteKeyGetterDictionary.get(json.type)
    if (!ctor) throw new Error(`Unknown getter type: ${json.type}`)
    return ctor.from_json(json) as DnoteKeyGetter
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function build_dnote_predicate_from_json(json: any): DnotePredicate {
    register_dictionary()
    if ('logic' in json && Array.isArray(json.predicates)) {
        const children = json.predicates.map(build_dnote_predicate_from_json)
        if (json.logic === 'AND') return new AndPredicate(children)
        if (json.logic === 'OR') return new OrPredicate(children)
        if (json.logic === 'NOT') return new NotPredicate(children)
        throw new Error(`Unknown logic type: ${json.logic}`)
    }

    const ctor = PredicateDictionary.get(json.type)
    if (!ctor) throw new Error(`Unknown predicate type: ${json.type}`)
    return ctor.from_json(json) as DnotePredicate
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function build_dnote_kyou_filter_from_json(json: any): DnoteKyouFilter {
    register_dictionary()
    const ctor = DnoteKyouFilterDictionary.get(json.type)
    if (!ctor) throw new Error(`Unknown getter type: ${json.type}`)
    return ctor.from_json(json) as DnoteKyouFilter
}