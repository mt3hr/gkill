package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type GetKmemoResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	KmemoHistories []reps.Kmemo `json:"kmemo_histories"`
}
