package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type GetSharedKyousResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	MiKyous []*reps.Kyou `json:"mi_kyous"`

	Title string `json:"title"`

	Mis []*reps.Mi `json:"mis"`

	Tags []*reps.Tag `json:"tags"`

	Texts []*reps.Text `json:"texts"`

	TimeIss []*reps.TimeIs `json:"timeiss"`
}
