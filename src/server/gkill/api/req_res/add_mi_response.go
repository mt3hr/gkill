package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type AddMiResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	AddedMi *reps.Mi `json:"added_mi"`

	AddedKyou *reps.Kyou `json:"added_kyou"`
}
