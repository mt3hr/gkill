package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

// RepInfoMCPDTO はKyouを供給するrep1つの構造化情報。
// RepType は query.rep_types にそのまま渡せる正準値（表示ラベルではない）。
// RepName は query.reps にそのまま渡せる実rep名。
type RepInfoMCPDTO struct {
	RepName string `json:"rep_name"`
	RepType string `json:"rep_type"`
}

// PluginRepInfoMCPDTO はプラグインrepの対応表。
// プラグインは rep_types では絞れない（正準値の集合に居ない）ため、
// RepName を query.reps へ、DataType を data_types へ渡して絞る。
type PluginRepInfoMCPDTO struct {
	RepName    string `json:"rep_name"`
	DataType   string `json:"data_type"`
	PluginName string `json:"plugin_name"`
}

// GetRepInfosMCPResponse は /api/get_rep_infos_mcp のレスポンス。
//
// rep_types の正準語彙（find.KyouRepTypes）は長らくどのAPIからも取得できず、
// MCPクライアントは総当たりでしか発見できなかった（外部監査 A1。「画像repの正準値が
// 表示ラベルと無関係な directory である」ことは総当たりでしか判明しなかった）。
// このエンドポイントが語彙とrep一覧の突き合わせ先になる。
type GetRepInfosMCPResponse struct {
	Messages []*message.GkillMessage `json:"messages"`
	Errors   []*message.GkillError   `json:"errors"`
	// RepInfos はKyouを供給するrepの一覧（(rep_name, rep_type) で重複排除済み）。
	// ファイルパスは含めない。
	RepInfos []RepInfoMCPDTO `json:"rep_infos"`
	// CanonicalRepTypes は query.rep_types が受理する正準値の全一覧（find.KyouRepTypes）。
	CanonicalRepTypes []string `json:"canonical_rep_types"`
	// Plugins はプラグインrepの対応表。
	Plugins []PluginRepInfoMCPDTO `json:"plugins"`
}
