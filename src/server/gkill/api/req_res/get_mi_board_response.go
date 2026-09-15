package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type GetMiBoardResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	Boards []string `json:"boards"`
}
