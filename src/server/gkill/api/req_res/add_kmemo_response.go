package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type AddKmemoResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	AddedKmemo *reps.Kmemo `json:"added_kmemo"`

	AddedKyou *reps.Kyou `json:"added_kyou"`
}
