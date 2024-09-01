// ˅
package api

import (
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

// ˄

type FindKyouContext struct {
	// ˅

	// ˄

	RawQueryJSON string

	ParsedQuery map[string]string

	Repositories *reps.GkillRepositories

	MatchReps map[string]reps.Repository

	AllTags map[string]*reps.Tag

	MatchTags map[string]*reps.Tag

	MatchTexts map[string]*reps.Text

	MatchTimeIssAtFindTimeIs map[string]*reps.TimeIs

	MatchTimeIssAtFilterTags map[string]*reps.TimeIs

	MatchTimeIsTags map[string]*reps.Tag

	MatchTimeIsTexts map[string]*reps.Text

	MatchKyousCurrent map[string]*reps.Kyou

	MatchKyousAtFindKyou map[string]*reps.Kyou

	MatchKyousAtFilterTags map[string]*reps.Kyou

	MatchKyousAtFilterTimeIs map[string]*reps.Kyou

	MatchKyousAtFilterLocation map[string]*reps.Kyou

	ResultKyous []*reps.Kyou

	TagFilterMode *TagFilterMode

	TimeIsTagFilterMode *TagFilterMode

	// ˅

	// ˄
}

// ˅

// ˄
