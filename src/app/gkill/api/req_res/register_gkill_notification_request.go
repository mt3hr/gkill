package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/share_kyou_info"

type RegisterGkillNotificationRequest struct {
	SessionID string `json:"session_id"`

	Subscription share_kyou_info.JSONString `json:"subscription"`

	PublicKey string `json:"public_key"`
}
