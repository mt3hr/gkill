package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type UpdateReKyouResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	UpdatedReKyou *reps.ReKyou `json:"updated_rekyou"`

	UpdatedKyou *reps.Kyou `json:"updated_kyou"`
}
