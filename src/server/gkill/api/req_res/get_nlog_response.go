package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type GetNlogResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	NlogHistories []reps.Nlog `json:"nlog_histories"`
}
