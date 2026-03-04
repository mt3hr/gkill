package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type UpdateTimeisResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	UpdatedTimeis *reps.TimeIs `json:"updated_timeis"`

	UpdatedKyou *reps.Kyou `json:"updated_kyou"`
}
