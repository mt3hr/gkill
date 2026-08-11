// Package gkill_plugin はプラグインプロトコルの型定義(stdio改行区切りJSON)。
package gkill_plugin

import "slices"

// PluginProvidedKind はプラグインがKyouのメタ情報以外に提供できるデータ種別。
// manifest.jsonのprovidesに書く。値はFindQuery.RepTypesと同じ語彙にそろえてある。
type PluginProvidedKind string

const (
	// PluginProvidesKmemo はテキストメモを提供することを表す。
	PluginProvidesKmemo PluginProvidedKind = "kmemo"
	// PluginProvidesKC は数値記録を提供することを表す。
	PluginProvidesKC PluginProvidedKind = "kc"
	// PluginProvidesURLog はブックマークを提供することを表す。
	PluginProvidesURLog PluginProvidedKind = "urlog"
	// PluginProvidesNlog は支出記録を提供することを表す。
	PluginProvidesNlog PluginProvidedKind = "nlog"
	// PluginProvidesLantana は気分値を提供することを表す。
	PluginProvidesLantana PluginProvidedKind = "lantana"
	// PluginProvidesTimeIs は時間計測を提供することを表す。
	PluginProvidesTimeIs PluginProvidedKind = "timeis"
	// PluginProvidesMi はタスクを提供することを表す。
	PluginProvidesMi PluginProvidedKind = "mi"
	// PluginProvidesTag はタグを提供することを表す。
	PluginProvidesTag PluginProvidedKind = "tag"
	// PluginProvidesText はテキストを提供することを表す。
	PluginProvidesText PluginProvidedKind = "text"
	// PluginProvidesNotification は通知を提供することを表す。
	PluginProvidesNotification PluginProvidedKind = "notification"
	// PluginProvidesGPSLog はGPSログを提供することを表す。
	// GPSログはKyouではないので、これだけは型別データではなく専用コマンド
	// (get_gps_logs)で受け渡す。
	PluginProvidesGPSLog PluginProvidedKind = "gpslog"
)

// AllPluginProvidedKinds はprovidesに書ける値の全集合。
var AllPluginProvidedKinds = []PluginProvidedKind{
	PluginProvidesKmemo, PluginProvidesKC, PluginProvidesURLog, PluginProvidesNlog,
	PluginProvidesLantana, PluginProvidesTimeIs, PluginProvidesMi,
	PluginProvidesTag, PluginProvidesText, PluginProvidesNotification,
	PluginProvidesGPSLog,
}

// IsTyped は種別がKyouに載る型別データかを返す。
// Tag / Text / Notification（付随データ）とGPSLog（Kyouではない）はfalse。
func (k PluginProvidedKind) IsTyped() bool {
	switch k {
	case PluginProvidesTag, PluginProvidesText, PluginProvidesNotification, PluginProvidesGPSLog:
		return false
	}
	return true
}

// PluginManifest はプラグインのmanifest.jsonの内容を表す。
// $GKILL_HOME/plugins/{userID}/{pluginName}/manifest.json に配置する。
type PluginManifest struct {
	// ProtocolVersion はgkillプラグインプロトコルのバージョン。現在は "1"。
	ProtocolVersion string `json:"protocol_version"`

	// Name はプラグインの識別名（ディレクトリ名と一致させること）。
	Name string `json:"name"`

	// Version はプラグインのバージョン（例: "1.0.0"）。
	Version string `json:"version"`

	// Description はプラグインの説明文。
	Description string `json:"description"`

	// DataType はこのプラグインが生成するKyouのdata_type値（例: "twitter_kyou"）。
	// 既存のdata_type（kmemo, kc等）と衝突しない一意な名前にすること。
	DataType string `json:"data_type"`

	// RepName はタイムライン上でのリポジトリ表示名（例: "Twitter"）。
	RepName string `json:"rep_name"`

	// Executable は実行ファイル名（拡張子なし、OSにより .exe 等を自動付加）。
	Executable string `json:"executable"`

	// MinGkillVersion はこのプラグインが動作する最低gkillバージョン。
	MinGkillVersion string `json:"min_gkill_version"`

	// Provides はこのプラグインがKyouのメタ情報以外に提供するデータ種別。
	//
	// 未指定（省略・null・空配列）が既定で、このときgkillはアダプタを一切登録しない。
	// 既存プラグインはmanifest.jsonを書き換えなくても従来どおり動く。
	// 書ける値は AllPluginProvidedKinds を参照。未知の値は警告して読み飛ばす。
	//
	// 型別データ（kc等）を宣言する場合、DataTypeはその種別で始めること。
	// クライアントはdata_typeの接頭辞で型別ビューを出し分けるため、
	// 食い違うと型別データが永久に読み込まれない。
	Provides []PluginProvidedKind `json:"provides,omitempty"`

	// EmitsKyou はこのプラグインがKyouを返すかを表す。省略時（null）はtrue。
	//
	// GPSログだけを提供するプラグインのように、Kyouを1件も返さないものはfalseにする。
	// falseのプラグインはRepositoriesに登録されないので、
	// 「記録保管場所」の一覧に出ず、検索のたびに空振りの往復も発生しない。
	// PluginRepsとGPSLogRepsには従来どおり登録されるので、
	// 設定画面・死活確認・GPSログの受け渡しはそのまま効く。
	//
	// Providesから推測はしない。provides:["kc"]と書いた作者のKyouが黙って消えると
	// 原因の分からない不具合になるので、明示的に切ってもらう。
	EmitsKyou *bool `json:"emits_kyou,omitempty"`
}

// EmitsKyouOrDefault はKyouを返すプラグインかを返す。未指定ならtrue。
func (m PluginManifest) EmitsKyouOrDefault() bool {
	return m.EmitsKyou == nil || *m.EmitsKyou
}

// ProvidedKinds はProvidesを集合にして返す。未指定なら空の集合。
// 呼び出し側でスライスの線形探索を書かないためのもの。
func (m PluginManifest) ProvidedKinds() map[PluginProvidedKind]struct{} {
	kinds := make(map[PluginProvidedKind]struct{}, len(m.Provides))
	for _, kind := range m.Provides {
		kinds[kind] = struct{}{}
	}
	return kinds
}

// Provides はこのプラグインが指定の種別を提供するかを返す。
func (m PluginManifest) ProvidesKind(kind PluginProvidedKind) bool {
	return slices.Contains(m.Provides, kind)
}
