package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type AddNotificationRequest struct {
	SessionID string `json:"session_id"`

	Notification *reps.Notification `json:"notification"`
}
