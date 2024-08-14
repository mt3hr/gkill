// ˅
package find_filter

// ˄

type FindKyouContext struct {
	// ˅

	// ˄

	MatchReps []*Repository

	AllKyousWhenDateInRep []*Kyou

	MatchKyousAtFilterWords []*Kyou

	MatchKyousAtFilterTags []*Kyou

	MatchKyousAtFilterTimeIs []*Kyou

	MatchKyousAtFilterLocation []*Kyou

	ResultKyous []*Kyou

	Words *Words

	TagFilterMode *TagFilterMode

	TimeIsTagFilterMode *TagFilterMode

	// ˅

	// ˄
}

// ˅

// ˄
