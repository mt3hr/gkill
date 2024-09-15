package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type UpdateLantanaResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	UpdatedLantana *reps.Lantana `json:"updated_lantana"`

	UpdatedLantanaKyou *reps.Kyou `json:"updated_lantana_kyou"`
}
