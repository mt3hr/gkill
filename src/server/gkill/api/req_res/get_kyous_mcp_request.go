package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/find"

// GetKyousMCPRequest は /api/get_kyous_mcp のリクエスト。
//
// v2 でトップレベルのフィルタ・集計パラメータを追加した（FindQuery は不変のまま）。
// 契約の全体は documents/adr/0604-mcp-composite-cursor-strict-limits.md を参照。
// 旧 include_id / include_rep_name は廃止（id / rep_name は常時付与。未知フィールドは
// encoding/json が無視するので旧クライアントの送信は無害）。
type GetKyousMCPRequest struct {
	SessionID       string          `json:"session_id"`
	Query           *find.FindQuery `json:"query"`
	LocaleName      string          `json:"locale_name"`
	Limit           int             `json:"limit"`             // default 50、1..1000。厳密な上限
	Cursor          string          `json:"cursor"`            // "{RFC3339Nano}::{ID}"。旧形式(RFC3339/日付のみ)も受理
	MaxSizeMB       float64         `json:"max_size_mb"`       // default 1.0。厳密な上限(例外はページ先頭1件のみ+警告)
	IsIncludeTimeIs *bool           `json:"is_include_timeis"` // default true

	// CountOnly はDTO構築・付随データ取得を全て飛ばして件数だけ返す。cursor とは併用不可。
	CountOnly bool `json:"count_only"`
	// GroupBy は件数をバケット集計して buckets で返す。
	// month|day|week_of_day|hour|data_type|rep_name|url_domain|file_extension。cursor とは併用不可。
	GroupBy string `json:"group_by"`
	// DataTypes は DTO の data_type 文字列(mi_create / claude_conversation 等)の許可リスト。
	// nil=未使用、非nil空=0件（FindQuery の null 意味論に揃える）。
	DataTypes []string `json:"data_types"`

	// CreateApps / UpdateApps は「どのアプリが書いたか / 最後に更新したか」の許可リスト。
	// gkill_kftl / gkill_wear / gkill_mcp_readwrite / gkill_mcp_write / urlog_bookmarklet / gkill / git など。
	// nil=未使用、非nil空=0件（FindQuery の null 意味論に揃える）。
	//
	// FindQuery に足していないのは意図的。ReKyou / MiReKyou のワード委譲が
	// 利用者のクエリをそのまま下位検索へ流すので、SQL まで降ろすと
	// 「MCPで作ったリポストだが参照先はブラウザ作成」が黙って消える。
	CreateApps []string `json:"create_apps"`
	UpdateApps []string `json:"update_apps"`
	// NumMin / NumMax は数値ペイロード(kc.num_value / nlog.amount / lantana.mood)の範囲絞り込み。
	// どちらか非nilで有効になり、有効時は数値を持たない種別の行は結果から外れる。
	NumMin *float64 `json:"num_min"`
	NumMax *float64 `json:"num_max"`
	// IDFKinds は idf(ファイル)の種別絞り込み。image|video|audio|zip|other。
	// nil=未使用、非nil空=0件。有効時は idf 以外の行は結果から外れる。
	IDFKinds []string `json:"idf_kinds"`
	// IncludeFileSize は idf ペイロードに file_size(バイト)を載せる(ページ内のみ os.Stat)。
	IncludeFileSize bool `json:"include_file_size"`
}

func (r *GetKyousMCPRequest) ShouldIncludeTimeIs() bool {
	if r.IsIncludeTimeIs == nil {
		return true
	}
	return *r.IsIncludeTimeIs
}
