package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type UpdateNlogResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	UpdatedNlog *reps.Nlog `json:"updated_nlog"`

	UpdatedNlogKyou *reps.Kyou `json:"updated_nlog_kyou"`
}
