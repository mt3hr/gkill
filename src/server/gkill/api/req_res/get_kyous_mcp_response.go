package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

// GetKyousMCPResponse は /api/get_kyous_mcp のレスポンス。
// 契約の全体は documents/adr/0053-mcp-composite-cursor-strict-limits.md を参照。
type GetKyousMCPResponse struct {
	Messages []*message.GkillMessage `json:"messages"`
	Errors   []*message.GkillError   `json:"errors"`
	Kyous    []KyouMCPDTO            `json:"kyous,omitempty"`
	// TotalCount は「条件に合う全件数」。**cursor 無しの応答(1ページ目・count_only・group_by)にのみ**入る。
	// カーソルは検索の期間上限へ押し下げられるため、2ページ目以降のハンドラは全件数を知らない
	// (知ろうとすると毎ページ全期間を検索し直すことになる。ADR-0052 の事故の教訓)。
	TotalCount *int `json:"total_count,omitempty"`
	// ReturnedCount はこの応答に載せた件数。
	ReturnedCount int `json:"returned_count"`
	// RemainingCount はこの応答の続き（カーソル以降）に残っている件数。全データ応答に入る。
	// 旧 v1 は TotalCount がカーソルの有無で「全件数/残件数」と意味を変えており、
	// 進捗率の分母に使うと静かに壊れていた（外部監査 S3）。
	RemainingCount int    `json:"remaining_count"`
	HasMore        bool   `json:"has_more"`
	NextCursor     string `json:"next_cursor,omitempty"`
	// Buckets は group_by 指定時のバケット集計結果。
	Buckets []KyouCountBucketMCPDTO `json:"buckets,omitempty"`
	// Partial は付随データ(タグ/テキスト/通知/TimeIs)の一部取得に失敗し、
	// 返した Kyou の付随データが不完全なことを表す。AIクライアントが「欠落のない完全な結果」と
	// 誤解しないための印。true のとき Warnings に失敗の内訳が入る。
	Partial bool `json:"partial,omitempty"`
	// Warnings は英語ベタ書きの警告列。付随データの欠落(Partial)に加え、
	// 未知のフィルタ値(rep_types/tags/reps/data_types の綴り違い等)の指摘にも使う
	// （黙って0件を返すと「該当なし」と区別が付かない。外部監査 S7）。
	Warnings []string `json:"warnings,omitempty"`
}

// KyouCountBucketMCPDTO は group_by 集計の1バケット。
type KyouCountBucketMCPDTO struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}
