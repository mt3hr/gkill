package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type UpdateKmemoResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	UpdatedKmemo *reps.Kmemo `json:"updated_kmemo"`

	UpdatedKyou *reps.Kyou `json:"updated_kyou"`
}
