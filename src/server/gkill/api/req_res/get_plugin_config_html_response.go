package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type GetPluginConfigHTMLResponse struct {
	Messages message.GkillMessages `json:"messages"`
	Errors   message.GkillErrors   `json:"errors"`
	HTML     string                `json:"html"`
}
