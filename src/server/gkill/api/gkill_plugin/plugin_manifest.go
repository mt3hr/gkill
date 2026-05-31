package gkill_plugin

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
}
