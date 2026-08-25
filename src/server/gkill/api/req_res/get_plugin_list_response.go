package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
)

// PluginInfo はAPIが返すプラグインの情報。
type PluginInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	DataType    string `json:"data_type"`
	RepName     string `json:"rep_name"`
	// EmitsKyou はこのプラグインがKyouを返すか（manifestのemits_kyou。未指定はtrue）。
	//
	// **falseのときDataTypeとRepNameは検索に使える値ではない。** そのプラグインは
	// Kyouを1件も出さないのでRepsに載らず、query.repsにもdata_typesにも一致しない。
	// manifestの必須項目なので値は入っているが、参照先のKyouが存在しない。
	// データはProvidesに対応する経路（gpslogならGPSログAPI）から読む。
	EmitsKyou bool `json:"emits_kyou"`
	// Provides はKyouのメタ情報以外に提供するデータ種別（manifestのprovides）。
	// 宣言していなければ省略。値の全集合は gkill_plugin.AllPluginProvidedKinds。
	Provides []string `json:"provides,omitempty"`
	// IsAlive は「プロセスが起動できてpingに応答したか」。判定は必要ならプロセスを起動する。
	// データが取り込めているかは表さない（それは TypedIndex / LastError を見る）。
	IsAlive bool `json:"is_alive"`
	// ProcessRunning はプロセスが現に起動済みか（副作用なしの受動読み）。
	ProcessRunning bool `json:"process_running"`
	// LastError はプラグインstderrの末尾（直近約4KB）。何も出ていなければ省略。
	// **索引構築の失敗はここには出ない**（タイムアウトやJSON不正はgkill側で起きるため）。
	// そちらは typed_index.last_build_error を見ること。
	// 「is_alive=true なのに0件」の理由（ビルドエラー等）をAPIから診断できるようにする
	// （外部監査 D2。以前はサーバのコンソールにしか出なかった）。
	LastError string `json:"last_error,omitempty"`
	// TypedIndex は型別/付随データを提供するプラグインの索引統計（鮮度・件数・時刻範囲）。
	// **これはKyouの索引**で、record_countはプラグインが返したKyouのユニーク数。
	// provides の無いプラグインと、provides が gpslog だけのプラグインでは省略
	// （後者は索引の材料が1件も無く、出すと never_built / 0 に固定されて
	// 「索引が壊れている」と誤読される。外部監査 D1 と 2026-08-24 の実利用報告）。
	TypedIndex *PluginTypedIndexStatsMCPDTO `json:"typed_index,omitempty"`
	// GPSIndex は provides に gpslog を含むプラグインのGPSスナップショット統計。
	// TypedIndex とは別物（GPSログはKyouではない）。一度も取得していなければ省略。
	GPSIndex *PluginGPSIndexStatsMCPDTO `json:"gps_index,omitempty"`
}

// PluginGPSIndexStatsMCPDTO はプラグインのGPSログスナップショットの統計DTO。
//
// GPSログはKyouではないので型別索引には載らない。ここが無いと
// 「ロケーション履歴がどこまで取り込まれたか」を知る手段が
// GPSログAPIを実際に引く以外に無かった。
type PluginGPSIndexStatsMCPDTO struct {
	// PointCount はスナップショットの点数。
	PointCount int `json:"point_count"`
	// Oldest / Newest は点の時刻範囲（RFC3339）。0件なら省略。
	Oldest string `json:"oldest,omitempty"`
	Newest string `json:"newest,omitempty"`
	// FetchedAt はスナップショットの取得時刻（＝この統計の鮮度）。
	FetchedAt string `json:"fetched_at"`
}

// PluginTypedIndexStatsMCPDTO はプラグイン型別索引の統計DTO。
type PluginTypedIndexStatsMCPDTO struct {
	// OK は索引が構築済みか。falseなら他フィールドは未確定（未構築と0件を区別するため）。
	OK bool `json:"ok"`
	// State は never_built / failed / ok。ok=false の内訳を分ける
	// （以前は「一度も構築していない」と「構築に失敗した」が潰れていた）。
	State string `json:"state"`
	// LastBuildError は直近の索引構築の失敗理由。成功していれば省略。
	// last_error（プラグインプロセスのstderr）とは別物で、
	// 索引構築の失敗要因はstderrには出ない。
	LastBuildError string `json:"last_build_error,omitempty"`
	// LastAttemptAt は直近に構築を試みた時刻（RFC3339）。一度も試していなければ省略。
	// 再構築はバックオフ中だとエラーすら発生しないので、これが無いと
	// 「なぜ何も起きていないのか」が分からない。
	LastAttemptAt string `json:"last_attempt_at,omitempty"`
	RecordCount   int    `json:"record_count"`
	// Oldest / Newest はレコードの related_time の範囲（RFC3339）。件数0なら省略。
	Oldest string `json:"oldest,omitempty"`
	Newest string `json:"newest,omitempty"`
	// Truncated は索引の上限による切り捨てが起きたか（trueならrecord_countは実数より小さい）。
	Truncated bool `json:"truncated"`
	// BuiltAt は索引の構築時刻（＝この統計の鮮度）。
	BuiltAt string `json:"built_at"`
}

type GetPluginListResponse struct {
	Messages []*message.GkillMessage `json:"messages"`
	Errors   []*message.GkillError   `json:"errors"`
	Plugins  []PluginInfo            `json:"plugins"`
}
