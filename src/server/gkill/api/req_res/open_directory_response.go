package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
)

type OpenDirectoryResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`
}
