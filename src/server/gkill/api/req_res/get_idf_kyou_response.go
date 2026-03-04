package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type GetIDFKyouResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	IDFKyouHistories []reps.IDFKyou `json:"idf_kyou_histories"`
}
