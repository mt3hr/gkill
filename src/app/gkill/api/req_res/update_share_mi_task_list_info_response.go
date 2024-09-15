package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/mi_share_info"
)

type UpdateShareMiTaskListInfoResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	ShareMiTaskListInfos *mi_share_info.MiShareInfo `json:"share_mi_task_list_infos"`
}
