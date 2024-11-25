package req_res

import "github.com/mt3hr/gkill/src/app/gkill/api/message"

type GetGkillInfoResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	UserID string `json:"user_id"`

	Device string `json:"device"`

	UserIsAdmin bool `json:"user_is_admin"`
}
