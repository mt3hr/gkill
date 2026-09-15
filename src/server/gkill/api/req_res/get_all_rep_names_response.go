package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type GetAllRepNamesResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	RepNames []string `json:"rep_names"`
}
