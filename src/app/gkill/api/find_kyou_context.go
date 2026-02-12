package api

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type FindKyouContext struct {
	GkillDAOManager                  *dao.GkillDAOManager
	DisableLatestDataRepositoryCache bool
	UserID                           string
	Device                           string
	ParsedFindQuery                  *find.FindQuery            `json:"parsed_find_query"`
	Repositories                     *reps.GkillRepositories    `json:"repositories"`
	MatchReps                        map[string]reps.Repository `json:"match_reps"`
	AllTags                          map[string]reps.Tag        `json:"all_tags"`
	AllHideTagsWhenUnchecked         map[string]reps.Tag        `json:"all_hide_tags_when_unchecked"`
	MatchHideTagsWhenUncheckedKyou   map[string]reps.Tag
	MatchHideTagsWhenUncheckedTimeIs map[string]reps.Tag
	RelatedTagIDs                    map[string]interface{}
	MatchTags                        map[string]reps.Tag    `json:"match_tags"`
	MatchTexts                       map[string]reps.Text   `json:"match_texts"`
	MatchTimeIssAtFindTimeIs         map[string]reps.TimeIs `json:"match_time_iss_at_find_time_is"`
	MatchTimeIssAtFilterTags         map[string]reps.TimeIs `json:"match_time_iss_at_filter_tags"`
	MatchMisAtFilterMi               map[string]reps.Mi     `json:"match_mis_at_filter_mi"`
	MatchTimeIsTags                  map[string]reps.Tag    `json:"match_time_is_tags"`
	MatchTimeIsTexts                 map[string]reps.Text   `json:"match_time_is_texts"`
	MatchKyousCurrent                map[string][]reps.Kyou `json:"match_kyous_current"`
	MatchKyousAtFindKyou             map[string][]reps.Kyou `json:"match_kyous_at_find_kyou"`
	MatchKyousAtFilterMi             map[string][]reps.Kyou `json:"match_kyous_at_filter_mi"`
	MatchKyousAtFilterTags           map[string][]reps.Kyou `json:"match_kyous_at_filter_tags"`
	MatchKyousAtFilterTimeIs         map[string][]reps.Kyou `json:"match_kyous_at_filter_time_is"`
	MatchKyousAtFilterLocation       map[string][]reps.Kyou `json:"match_kyous_at_filter_location"`
	MatchKyousAtFilterImage          map[string][]reps.Kyou `json:"match_kyous_at_filter_image"`
	ResultKyous                      []reps.Kyou            `json:"result_kyous"`
}
