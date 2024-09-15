package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type UpdateURLogResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	UpdatedURLog *reps.URLog `json:"updated_urlog"`

	UpdatedURLogKyou *reps.Kyou `json:"updated_urlog_kyou"`
}
