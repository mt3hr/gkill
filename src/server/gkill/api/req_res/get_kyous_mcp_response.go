package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type GetKyousMCPResponse struct {
	Messages      []*message.GkillMessage `json:"messages"`
	Errors        []*message.GkillError   `json:"errors"`
	Kyous         []KyouMCPDTO            `json:"kyous"`
	TotalCount    int                     `json:"total_count"`
	ReturnedCount int                     `json:"returned_count"`
	HasMore       bool                    `json:"has_more"`
	NextCursor    string                  `json:"next_cursor,omitempty"`
}
