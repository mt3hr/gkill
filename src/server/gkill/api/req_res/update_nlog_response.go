package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type UpdateNlogResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	UpdatedNlog *reps.Nlog `json:"updated_nlog"`

	UpdatedKyou *reps.Kyou `json:"updated_kyou"`
}
