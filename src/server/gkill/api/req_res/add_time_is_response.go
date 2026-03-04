package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type AddTimeIsResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	AddedTimeis *reps.TimeIs `json:"added_timeis"`

	AddedKyou *reps.Kyou `json:"added_kyou"`
}
