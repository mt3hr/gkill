package sdk

import "context"

// Handler はプラグインが実装するハンドラ定義。
// 不要なメソッドはnilのままで可（デフォルト実装が使われる）。
type Handler struct {
	// FindKyous は検索クエリに合致するKyouを返す。必須。
	FindKyous func(ctx context.Context, q Query, cfg Config) ([]Kyou, error)

	// GetKyou はIDでKyouを1件返す。nilの場合はFindKyousで代替する。
	GetKyou func(ctx context.Context, id string, cfg Config) (*Kyou, error)

	// GetContentHTML はKyouIDに対応する詳細ビューのHTMLを返す。
	// nilの場合はデフォルトのシンプルなHTMLが使われる。
	GetContentHTML func(ctx context.Context, kyouID string, cfg Config) (string, error)

	// GetConfigHTML はプラグイン設定画面のHTMLを返す。
	// nilの場合はデフォルトのHTMLが使われる。
	GetConfigHTML func(ctx context.Context, cfg Config) (string, error)

	// PostConfig はフォームデータを受けて設定を更新する。
	// nilの場合はデフォルトの保存処理（Config をJSON保存）が使われる。
	PostConfig func(ctx context.Context, form map[string]string, cfg Config) (Config, error)

	// GetGPSLogs は期間に含まれるGPSログを返す。
	// manifest.jsonのprovidesに"gpslog"を書いたプラグインでは必須。
	// nilのままだと get_gps_logs は「未実装」エラーを返す。
	//
	// q.Limit を必ず尊重すること。無視して全件返すと、
	// gkill側の bufio.Scanner（32MB）が token too long で読めなくなる。
	GetGPSLogs func(ctx context.Context, q GPSLogQuery, cfg Config) (GPSLogPage, error)

	// RepName はリポジトリ表示名（manifest.jsonのrep_nameと一致させること）。
	RepName string

	// DefaultConfig はconfig.jsonが無いときに書き出す既定設定。
	// nilなら生成しない。既存のconfig.jsonは上書きされない。
	DefaultConfig Config
}
