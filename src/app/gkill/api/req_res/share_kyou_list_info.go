package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/share_kyou_info"

type ShareKyouListInfo struct {
	UserID string `json:"user_id"`

	Device string `json:"device"`

	ShareTitle string `json:"share_title"`

	IsShareDetail bool `json:"is_share_detail"`

	ShareID string `json:"share_id"`

	FindQueryJSON share_kyou_info.JSONString `json:"find_query_json"`
}
