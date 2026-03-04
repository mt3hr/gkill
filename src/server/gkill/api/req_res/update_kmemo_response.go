package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type UpdateKmemoResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	UpdatedKmemo *reps.Kmemo `json:"updated_kmemo"`

	UpdatedKyou *reps.Kyou `json:"updated_kyou"`
}
