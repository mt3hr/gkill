package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type UpdateTagResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	UpdatedTag *reps.Tag `json:"updated_tag"`
}
