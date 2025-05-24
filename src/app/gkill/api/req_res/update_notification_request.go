package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type UpdateNotificationRequest struct {
	SessionID string `json:"session_id"`

	Notification *reps.Notification `json:"notification"`

	TXID *string `json:"tx_id"`
}
