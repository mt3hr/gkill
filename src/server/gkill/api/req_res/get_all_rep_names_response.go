package req_res

import "github.com/mt3hr/gkill/src/app/gkill/api/message"

type GetAllRepNamesResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	RepNames []string `json:"rep_names"`
}
