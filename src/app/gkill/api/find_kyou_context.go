package api

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type FindKyouContext struct {
	ParsedFindQuery *find.FindQuery `json:"parsed_find_query"`

	Repositories *reps.GkillRepositories `json:"repositories"`

	MatchReps map[string]reps.Repository `json:"match_reps"`

	AllTags map[string]*reps.Tag `json:"all_tags"`

	MatchTags map[string]*reps.Tag `json:"match_tags"`

	MatchTexts map[string]*reps.Text `json:"match_texts"`

	MatchTimeIssAtFindTimeIs map[string]*reps.TimeIs `json:"match_time_iss_at_find_time_is"`

	MatchTimeIssAtFilterTags map[string]*reps.TimeIs `json:"match_time_iss_at_filter_tags"`

	MatchTimeIsTags map[string]*reps.Tag `json:"match_time_is_tags"`

	MatchTimeIsTexts map[string]*reps.Text `json:"match_time_is_texts"`

	MatchKyousCurrent map[string]*reps.Kyou `json:"match_kyous_current"`

	MatchKyousAtFindKyou map[string]*reps.Kyou `json:"match_kyous_at_find_kyou"`

	MatchKyousAtFilterTags map[string]*reps.Kyou `json:"match_kyous_at_filter_tags"`

	MatchKyousAtFilterTimeIs map[string]*reps.Kyou `json:"match_kyous_at_filter_time_is"`

	MatchKyousAtFilterLocation map[string]*reps.Kyou `json:"match_kyous_at_filter_location"`

	ResultKyous []*reps.Kyou `json:"result_kyous"`

	TagFilterMode *find.TagFilterMode `json:"tag_filter_mode"`

	TimeIsTagFilterMode *find.TagFilterMode `json:"time_is_tag_filter_mode"`
}
