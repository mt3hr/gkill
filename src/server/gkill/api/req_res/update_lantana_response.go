package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type UpdateLantanaResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	UpdatedLantana *reps.Lantana `json:"updated_lantana"`

	UpdatedKyou *reps.Kyou `json:"updated_kyou"`
}
