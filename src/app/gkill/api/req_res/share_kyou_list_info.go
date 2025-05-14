package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/share_kyou_info"

type ShareKyouListInfo struct {
	ShareID string `json:"share_id"`

	UserID string `json:"user_id"`

	Device string `json:"device"`

	ShareTitle string `json:"share_title"`

	FindQueryJSON share_kyou_info.JSONString `json:"find_query_json"`

	ViewType string `json:"view_type"`

	IsShareTimeOnly bool `json:"is_share_time_only"`

	IsShareWithTags bool `json:"is_share_with_tags"`

	IsShareWithTexts bool `json:"is_share_with_texts"`

	IsShareWithTimeIss bool `json:"is_share_with_timeiss"`

	IsShareWithLocations bool `json:"is_share_with_locations"`
}
