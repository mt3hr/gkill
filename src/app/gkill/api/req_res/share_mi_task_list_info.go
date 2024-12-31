package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/mi_share_info"

type ShareMiTaskListInfo struct {
	UserID string `json:"user_id"`

	Device string `json:"device"`

	ShareTitle string `json:"share_title"`

	IsShareDetail bool `json:"is_share_detail"`

	ShareID string `json:"share_id"`

	FindQueryJSON mi_share_info.JSONString `json:"find_query_json"`
}
