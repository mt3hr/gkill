package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type LogoutResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`
}
