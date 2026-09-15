package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type UpdateIDFKyouResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	UpdatedIDFKyou *reps.IDFKyou `json:"updated_idf_kyou"`

	UpdatedKyou *reps.Kyou `json:"updated_kyou"`
}
