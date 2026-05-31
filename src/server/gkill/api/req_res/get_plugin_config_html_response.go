package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type GetPluginConfigHTMLResponse struct {
	Messages []*message.GkillMessage `json:"messages"`
	Errors   []*message.GkillError   `json:"errors"`
	HTML     string                  `json:"html"`
}
