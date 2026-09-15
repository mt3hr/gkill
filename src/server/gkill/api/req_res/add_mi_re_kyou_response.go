package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type AddMiReKyouResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	AddedMiReKyou *reps.MiReKyou `json:"added_mirekyou"`

	AddedKyou *reps.Kyou `json:"added_kyou"`
}
