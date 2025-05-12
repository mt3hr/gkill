import type DnoteAgregateTarget from "../dnote-agregate-target"
import AgregateAverageGitCommitLogAdditionCodeCount from "../dnote-agregate-target/agregate-average-git-commit-log-code-addition-count"
import AgregateAverageGitCommitLogCodeCount from "../dnote-agregate-target/agregate-average-git-commit-log-code-count"
import AgregateAverageGitCommitLogDeletionCodeCount from "../dnote-agregate-target/agregate-average-git-commit-log-code-deletion-count"
import AgregateAverageLantanaMood from "../dnote-agregate-target/agregate-average-lantana-mood"
import AgregateAverageNlogAmount from "../dnote-agregate-target/agregate-average-nlog-amount"
import AgregateAverageTimeIsEndTime from "../dnote-agregate-target/agregate-average-timeis-end-time"
import AgregateAverageTimeIsStartTime from "../dnote-agregate-target/agregate-average-timeis-start-time"
import AgregateAverageTimeisTime from "../dnote-agregate-target/agregate-average-timeis-time"
import AgregateCountKyou from "../dnote-agregate-target/agregate-count-kyou"
import AgregateSumGitCommitLogAdditionCodeCount from "../dnote-agregate-target/agregate-sum-git-commit-log-code-addition-count"
import AgregateSumGitCommitLogCodeCount from "../dnote-agregate-target/agregate-sum-git-commit-log-code-count"
import AgregateSumGitCommitLogDeletionCodeCount from "../dnote-agregate-target/agregate-sum-git-commit-log-code-deletion-count"
import AgregateSumLantanaMood from "../dnote-agregate-target/agregate-sum-lantana-mood"
import AgregateSumNlogAmount from "../dnote-agregate-target/agregate-sum-nlog-amount"
import AgregateSumTimeIsTime from "../dnote-agregate-target/agregate-sum-timeis-time"
import AgregateSumKCNumValue from "../dnote-agregate-target/agregate-sum-kc-num-value"
import type DnoteKeyGetter from "../dnote-key-getter"
import DataTypeGetter from "../dnote-key-getter/data-type-getter"
import LantanaMoodGetter from "../dnote-key-getter/lantana-mood-getter"
import NlogShopNameGetter from "../dnote-key-getter/nlog-shop-name-getter"
import RelatedMonthGetter from "../dnote-key-getter/related-month-getter"
import RelatedWeekDayGetter from "../dnote-key-getter/related-week-day-getter"
import RelatedWeekGetter from "../dnote-key-getter/related-week-getter"
import RelatedDateGetter from "../dnote-key-getter/rerated-date-getter"
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
import AgregateTargetDictionary from "./dnote-aggregate-target-dictionary"
import DnoteKeyGetterDictionary from "./dnote-key-getter-dictionary"
import PredicateDictonary from "./dnote-predicate-dictionary"

export default function regist_dictionary(): void {
    PredicateDictonary.set("AndPredicate", AndPredicate)
    PredicateDictonary.set("DataTypePrefixPredicate", DataTypePrefixPredicate)
    PredicateDictonary.set("GitCommitLogCodeAdditionGreaterThanPredicate", GitCommitLogCodeAdditionGreaterThanPredicate)
    PredicateDictonary.set("GitCommitLogCodeAdditionLessThanPredicate", GitCommitLogCodeAdditionLessThanPredicate)
    PredicateDictonary.set("GitCommitLogCodeDeletionGreaterThanPredicate", GitCommitLogCodeDeletionGreaterThanPredicate)
    PredicateDictonary.set("GitCommitLogCodeDeletionLessThanPredicate", GitCommitLogCodeDeletionLessThanPredicate)
    PredicateDictonary.set("GitCommitLogCodeGreaterThanPredicate", GitCommitLogCodeGreaterThanPredicate)
    PredicateDictonary.set("GitCommitLogCodeLessThanPredicate", GitCommitLogCodeLessThanPredicate)
    PredicateDictonary.set("KmemoContentContainsPredicate", KmemoContentContainsPredicate)
    PredicateDictonary.set("KmemoContentEqualPredicate", KmemoContentEqualPredicate)
    PredicateDictonary.set("TextContentContainsPredicate", KmemoContentContainsPredicate)
    PredicateDictonary.set("TextContentEqualPredicate", KmemoContentEqualPredicate)
    PredicateDictonary.set("LantanaMoodEqualPredicate", LantanaMoodEqualPredicate)
    PredicateDictonary.set("LantanaMoodGreaterThanPredicate", LantanaMoodGreaterThanPredicate)
    PredicateDictonary.set("LantanaMoodLessThanPredicate", LantanaMoodLessThanPredicate)
    PredicateDictonary.set("MiTitleContainsPredicate", MiTitleContainsPredicate)
    PredicateDictonary.set("MiTitleEqualPredicate", MiTitleEqualPredicate)
    PredicateDictonary.set("NlogAmountGreaterThanPredicate", NlogAmountGreaterThanPredicate)
    PredicateDictonary.set("NlogAmountLessThanPredicate", NlogAmountLessThanPredicate)
    PredicateDictonary.set("NlogShopContainsPredicate", NlogShopContainsPredicate)
    PredicateDictonary.set("NlogShopEqualPredicate", NlogShopEqualPredicate)
    PredicateDictonary.set("NlogTitleContainsPredicate", NlogTitleContainsPredicate)
    PredicateDictonary.set("NlogTitleEqualPredicate", NlogTitleEqualPredicate)
    PredicateDictonary.set("NotPredicate", NotPredicate)
    PredicateDictonary.set("OrPredicate", OrPredicate)
    PredicateDictonary.set("RelatedTimeWeekPredicate", RelatedTimeWeekPredicate)
    PredicateDictonary.set("TagEqualPredicate", TagEqualPredicate)
    PredicateDictonary.set("TimeIsTitleContainsPredicate", TimeIsTitleContainsPredicate)
    PredicateDictonary.set("TimeIsTitleEqualPredicate", TimeIsTitleEqualPredicate)
    DnoteKeyGetterDictionary.set("DataTypeGetter", DataTypeGetter)
    DnoteKeyGetterDictionary.set("LantanaMoodGetter", LantanaMoodGetter)
    DnoteKeyGetterDictionary.set("NlogShopNameGetter", NlogShopNameGetter)
    DnoteKeyGetterDictionary.set("RelatedMonthGetter", RelatedMonthGetter)
    DnoteKeyGetterDictionary.set("RelatedWeekDayGetter", RelatedWeekDayGetter)
    DnoteKeyGetterDictionary.set("RelatedWeekGetter", RelatedWeekGetter)
    DnoteKeyGetterDictionary.set("RelatedDateGetter", RelatedDateGetter)
    DnoteKeyGetterDictionary.set("TagGetter", TagGetter)
    DnoteKeyGetterDictionary.set("TitleGetter", TitleGetter)
    AgregateTargetDictionary.set("AgregateAverageGitCommitLogAdditionCodeCount", AgregateAverageGitCommitLogAdditionCodeCount)
    AgregateTargetDictionary.set("AgregateAverageGitCommitLogCodeCount", AgregateAverageGitCommitLogCodeCount)
    AgregateTargetDictionary.set("AgregateAverageGitCommitLogDeletionCodeCount", AgregateAverageGitCommitLogDeletionCodeCount)
    AgregateTargetDictionary.set("AgregateAverageLantanaMood", AgregateAverageLantanaMood)
    AgregateTargetDictionary.set("AgregateAverageNlogAmount", AgregateAverageNlogAmount)
    AgregateTargetDictionary.set("AgregateAverageTimeIsEndTime", AgregateAverageTimeIsEndTime)
    AgregateTargetDictionary.set("AgregateAverageTimeIsStartTime", AgregateAverageTimeIsStartTime)
    AgregateTargetDictionary.set("AgregateAverageTimeIsTime", AgregateAverageTimeisTime)
    AgregateTargetDictionary.set("AgregateCountKyou", AgregateCountKyou)
    AgregateTargetDictionary.set("AgregateSumGitCommitLogAdditionCodeCount", AgregateSumGitCommitLogAdditionCodeCount)
    AgregateTargetDictionary.set("AgregateSumGitCommitLogCodeCount", AgregateSumGitCommitLogCodeCount)
    AgregateTargetDictionary.set("AgregateSumGitCommitLogDeletionCodeCount", AgregateSumGitCommitLogDeletionCodeCount)
    AgregateTargetDictionary.set("AgregateSumLantanaMood", AgregateSumLantanaMood)
    AgregateTargetDictionary.set("AgregateSumNlogAmount", AgregateSumNlogAmount)
    AgregateTargetDictionary.set("AgregateSumTimeIsTime", AgregateSumTimeIsTime)
    AgregateTargetDictionary.set("AgregateAverageKCNumValue", AgregateSumKCNumValue)
    AgregateTargetDictionary.set("AgregateMaxKCNumValue", AgregateSumKCNumValue)
    AgregateTargetDictionary.set("AgregateMinKCNumValue", AgregateSumKCNumValue)
    AgregateTargetDictionary.set("AgregateSumKCNumValue", AgregateSumKCNumValue)
}

export function build_dnote_aggregate_target_from_json(json: any): DnoteAgregateTarget {
    const ctor = AgregateTargetDictionary.get(json.type)
    if (!ctor) throw new Error(`Unknown aggregate type: ${json.type}`)
    return new ctor(json.value)
}

export function build_dnote_key_getter_from_json(json: any): DnoteKeyGetter {
    const ctor = DnoteKeyGetterDictionary.get(json.type)
    if (!ctor) throw new Error(`Unknown getter type: ${json.type}`)
    return new ctor(json.value)
}

export function build_dnote_predicate_from_json(json: any): DnotePredicate {
    if ('logic' in json && Array.isArray(json.predicates)) {
        const children = json.predicates.map(build_dnote_predicate_from_json)
        if (json.logic === 'AND') return new AndPredicate(children)
        if (json.logic === 'OR') return new OrPredicate(children)
        if (json.logic === 'NOT') return new NotPredicate(children)
        throw new Error(`Unknown logic type: ${json.logic}`)
    }

    const ctor = PredicateDictonary.get(json.type)
    if (!ctor) throw new Error(`Unknown predicate type: ${json.type}`)
    return new ctor(json.value)
}