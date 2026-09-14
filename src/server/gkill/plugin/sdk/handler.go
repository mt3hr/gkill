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

	// RepNames はこのプラグインの記録が名乗る rep 名の全集合を返す。
	// 1本のプラグインが複数のリポジトリを代表し、Kyou.RepName ごとに別の名前を出すときに実装する。
	// nil のままなら gkill は manifest の rep_name 1つだけとみなす（従来どおり）。
	//
	// 実装するなら、まだ1件も取り込んでいない間は nil ではなく**空スライス**を返すこと。
	// gkill は「実装していない(null)」と「いまは0個([])」を区別する。
	// 返した名前は get_all_rep_names に載り、query.reps の絞り込みと本文取得の引き当てに使われるので、
	// FindKyous が返す Kyou.RepName はこの集合のどれかに一致させること。
	RepNames func(ctx context.Context, cfg Config) ([]string, error)

	// DefaultConfig はconfig.jsonが無いときに書き出す既定設定。
	// nilなら生成しない。既存のconfig.jsonは上書きされない。
	DefaultConfig Config
}
