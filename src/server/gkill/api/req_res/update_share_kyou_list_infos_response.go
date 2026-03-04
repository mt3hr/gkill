package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/share_kyou_info"
)

type UpdateShareKyouListInfosResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	ShareKyouListInfos *share_kyou_info.ShareKyouInfo `json:"share_kyou_list_infos"`
}
