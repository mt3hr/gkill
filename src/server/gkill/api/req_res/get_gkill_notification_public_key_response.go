package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type GetGkillNotificationPublicKeyResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	GkillNotificationPublicKey string `json:"gkill_notification_public_key"`
}
