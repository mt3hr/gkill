package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type UpdateMiReKyouResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	UpdatedMiReKyou *reps.MiReKyou `json:"updated_mirekyou"`

	UpdatedKyou *reps.Kyou `json:"updated_kyou"`
}
