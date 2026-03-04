package req_res

import "github.com/mt3hr/gkill/src/app/gkill/api/message"

type GetMiBoardResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	Boards []string `json:"boards"`
}
