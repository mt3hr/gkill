package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

// RepInfoMCPDTO はKyouを供給するrep1つの構造化情報。
// RepType は query.rep_types にそのまま渡せる正準値（表示ラベルではない）。
// RepName は query.reps にそのまま渡せる実rep名。
type RepInfoMCPDTO struct {
	RepName string `json:"rep_name"`
	RepType string `json:"rep_type"`
}

// AttachedDataRepInfoMCPDTO はタグ・テキスト・通知・GPSログの格納先rep。
//
// **これは query.reps へ渡す値ではない。** RepInfos と混ぜてはいけない ――
// 混ぜると呼び出し側が query.reps へ渡し、Kyou の RepName と一致しないので
// 静かに0件になる。「add_tag / add_text がどこへ書かれるのか」を
// **書く前に**知るための一覧で、絞り込みには使えない。
//
// これらの rep は Kyou を1件も生まないので Reps（＝利用者が選べる記録保管場所）
// には入っておらず、GetAllRepNames にも RepInfos にも出てこなかった。
// 書き込み応答の rep_name で事後に分かるだけだった（2026-08-24 の再監査）。
type AttachedDataRepInfoMCPDTO struct {
	RepName string `json:"rep_name"`
	// DataKind は tag / text / notification / gpslog のいずれか。
	DataKind string `json:"data_kind"`
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
	// AttachedDataReps はタグ・テキスト・通知・GPSログの格納先rep。
	// RepInfos とは用途が違う（query.reps へ渡してはいけない）。
	AttachedDataReps []AttachedDataRepInfoMCPDTO `json:"attached_data_reps"`
}
