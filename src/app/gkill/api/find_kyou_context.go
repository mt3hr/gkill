// ˅
package api

import "github.com/mt3hr/gkill/src/app/gkill/dbo/reps"

// ˄

type FindKyouContext struct {
	// ˅

	// ˄

	MatchReps []reps.Repository

	AllKyousWhenDateInRep []*reps.Kyou

	MatchKyousAtFilterWords []*reps.Kyou

	MatchKyousAtFilterTags []*reps.Kyou

	MatchKyousAtFilterTimeIs []*reps.Kyou

	MatchKyousAtFilterLocation []*reps.Kyou

	ResultKyous []*reps.Kyou

	Words *Words

	TagFilterMode *TagFilterMode

	TimeIsTagFilterMode *TagFilterMode

	// ˅

	// ˄
}

// ˅

// ˄
