package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type GetAllTagNamesResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	TagNames []string `json:"tag_names"`
}
