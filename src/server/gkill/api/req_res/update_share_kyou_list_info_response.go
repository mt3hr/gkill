package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/share_kyou_info"
)

type UpdateShareKyouListInfoResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	ShareKyouListInfo *share_kyou_info.ShareKyouInfo `json:"share_kyou_list_info"`
}
