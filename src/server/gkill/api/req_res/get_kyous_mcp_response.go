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
	// Partial は付随データ(タグ/テキスト/通知/TimeIs)の一部取得に失敗し、
	// 返した Kyou の付随データが不完全なことを表す。AIクライアントが「欠落のない完全な結果」と
	// 誤解しないための印。true のとき Warnings に失敗の内訳が入る。
	Partial  bool     `json:"partial,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}
