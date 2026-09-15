package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type UpdateURLogResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	UpdatedURLog *reps.URLog `json:"updated_urlog"`

	UpdatedKyou *reps.Kyou `json:"updated_kyou"`
}
