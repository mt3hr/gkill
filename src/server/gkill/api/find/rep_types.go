package find

import "slices"

// KyouRepTypes は FindQuery.RepTypes が受理する正準値の一覧（Kyouを返すrep種別のみ）。
//
// この値集合の実装上の正本は selectMatchRepsFromQuery（find_filter.go）の switch で、
// そこと集合一致することをテスト（TestKyouRepTypesCoversRepsOfKyouRepType）が固定する。
// rep の生成側の語彙（gkill_dao_manager.go の switch）はこれに tag / text /
// notification / gpslog を加えたもので、そちらはソース走査テストが対応を固定する。
//
// 表示ラベル（ApplicationConfig の rep_type_struct）とは別物で、1:1対応しない。
// とくに画像・動画・音声・ZIPなどのファイル系repは "directory"（IDFディレクトリ
// インデックス）が正準値。この語彙が長らくどのAPIからも取得できず、MCPクライアントが
// 総当たりでしか発見できなかったため、/api/get_rep_infos_mcp が公開するようになった。
var KyouRepTypes = []string{
	"kmemo",
	"kc",
	"urlog",
	"timeis",
	"mi",
	"nlog",
	"lantana",
	"rekyou",
	"mirekyou",
	"directory",
	"git_commit_log",
}

// IsKyouRepType は repType が KyouRepTypes に含まれるかを返す。
func IsKyouRepType(repType string) bool {
	return slices.Contains(KyouRepTypes, repType)
}
