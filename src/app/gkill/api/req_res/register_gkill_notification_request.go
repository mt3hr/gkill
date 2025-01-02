package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/mi_share_info"

type RegisterGkillNotificationRequest struct {
	SessionID string `json:"session_id"`

	Subscription mi_share_info.JSONString `json:"subscription"`

	PublicKey string `json:"public_key"`
}
