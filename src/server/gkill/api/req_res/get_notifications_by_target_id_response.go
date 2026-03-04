package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type GetNotificationsByTargetIDResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	Notifications []reps.Notification `json:"notifications"`
}
