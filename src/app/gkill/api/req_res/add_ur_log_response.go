package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type AddURLogResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	AddedURLog *reps.URLog `json:"added_urlog"`

	AddedURLogKyou *reps.Kyou `json:"added_urlog_kyou"`
}
