package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type AddReKyouResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	AddedReKyou *reps.ReKyou `json:"added_re_kyou"`

	AddedKyou *reps.Kyou `json:"added_kyou"`
}
