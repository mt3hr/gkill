package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type AddMiResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	AddedMi *reps.Mi `json:"added_mi"`

	AddedMiKyou *reps.Kyou `json:"added_mi_kyou"`
}
