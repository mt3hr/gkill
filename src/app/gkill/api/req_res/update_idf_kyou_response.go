package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type UpdateIDFKyouResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	UpdatedIDFKyou *reps.IDFKyou `json:"updated_idf_kyou"`

	UpdatedIDFKyouKyou *reps.Kyou `json:"updated_idf_kyou_kyou"`
}
