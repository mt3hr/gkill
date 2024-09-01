// ˅
package api

import (
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

// ˄

type FindKyouContext struct {
	// ˅

	RawQueryJSON string //TODO モデル反映

	ParsedQuery map[string]string //TODO モデル反映

	// ˄
	Repositories *reps.GkillRepositories //TODO モデル反映

	MatchReps map[string]reps.Repository

	// TODO モデル反映 AllKyousWhenDateInRep []*reps.Kyou

	// TODO モデル反映 MatchKyousAtFilterWords []*reps.Kyou

	AllTags map[string]*reps.Tag //TODO モデル反映

	MatchTags map[string]*reps.Tag //TODO モデル反映

	MatchTexts map[string]*reps.Text //TODO モデル反映

	MatchTimeIssAtFindTimeIs map[string]*reps.TimeIs //TODO モデル反映

	MatchTimeIssAtFilterTags map[string]*reps.TimeIs //TODO モデル反映

	MatchTimeIsTags map[string]*reps.Tag //TODO モデル反映

	MatchTimeIsTexts map[string]*reps.Text //TODO モデル反映

	MatchKyousCurrent map[string]*reps.Kyou //TODO モデル反映

	MatchKyousAtFindKyou map[string]*reps.Kyou //TODO モデル反映

	MatchKyousAtFilterTags map[string]*reps.Kyou

	MatchKyousAtFilterTimeIs map[string]*reps.Kyou

	MatchKyousAtFilterLocation map[string]*reps.Kyou

	ResultKyous []*reps.Kyou

	// TODO モデル反映 Words *Words

	TagFilterMode *TagFilterMode

	TimeIsTagFilterMode *TagFilterMode

	// ˅

	// ˄
}

// ˅

// ˄
