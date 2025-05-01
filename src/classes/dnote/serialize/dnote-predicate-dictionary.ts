import AndPredicate from "../dnote-predicate/and-predicate"
import DataTypePrefixPredicate from "../dnote-predicate/data-type-prefix-predicate"
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

const PredicateDictonary = new Map<String, any>()
PredicateDictonary.set("AndPredicate", AndPredicate)
PredicateDictonary.set("OrPredicate", OrPredicate)
PredicateDictonary.set("NotPredicate", NotPredicate)
PredicateDictonary.set("TagPredicate", TagEqualPredicate)
PredicateDictonary.set("DataTypePrefixPredicate", DataTypePrefixPredicate)
PredicateDictonary.set("KmemoContentContainsPredicate", KmemoContentContainsPredicate)
PredicateDictonary.set("TimeIsTitleContainsPredicate", TimeIsTitleContainsPredicate)
PredicateDictonary.set("KmemoContentEqualPredicate", KmemoContentEqualPredicate)
PredicateDictonary.set("TimeIsTitleEqualPredicate", TimeIsTitleEqualPredicate)
PredicateDictonary.set("NlogTitleContainsPredicate", NlogTitleContainsPredicate)
PredicateDictonary.set("NlogTitleEqualPredicate", NlogTitleEqualPredicate)
PredicateDictonary.set("NlogShopContainsPredicate", NlogShopContainsPredicate)
PredicateDictonary.set("NlogShopEqualPredicate", NlogShopEqualPredicate)
PredicateDictonary.set("NlogAmountGreaterThanPredicate", NlogAmountGreaterThanPredicate)
PredicateDictonary.set("NlogAmountLessThanPredicate", NlogAmountLessThanPredicate)
PredicateDictonary.set("LantanaMoodEqualPredicate", LantanaMoodEqualPredicate)
PredicateDictonary.set("LantanaMoodLessThanPredicate", LantanaMoodLessThanPredicate)
PredicateDictonary.set("LantanaMoodGreaterThanPredicate", LantanaMoodGreaterThanPredicate)
PredicateDictonary.set("GitCommitLogCodeLessThanPredicate", GitCommitLogCodeLessThanPredicate)
PredicateDictonary.set("GitCommitLogCodeGreaterThanPredicate", GitCommitLogCodeGreaterThanPredicate)
PredicateDictonary.set("MiTitleContainsPredicate", MiTitleContainsPredicate)
PredicateDictonary.set("MiTitleEqualPredicate", MiTitleEqualPredicate)
PredicateDictonary.set("RelatedTimeWeekPredicate", RelatedTimeWeekPredicate)

export default PredicateDictonary