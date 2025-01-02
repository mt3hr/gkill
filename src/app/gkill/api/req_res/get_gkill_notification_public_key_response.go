package req_res

import "github.com/mt3hr/gkill/src/app/gkill/api/message"

type GetGkillNotificationPublicKeyResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	GkillNotificationPublicKey string `json:"gkill_notification_public_key"`
}
