package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
)

// PluginRepository はプラグインプロセスとstdio JSONで通信するRepositoryインターフェース。
// Repository インターフェースを満たすほか、GetContentHTML / GetConfigHTML / PostConfig も提供する。
type PluginRepository interface {
	Repository

	// GetManifest はプラグインのmanifest情報を返す。
	GetManifest() gkill_plugin.PluginManifest

	// GetPluginDir はプラグインの置き場所（$GKILL_HOME/plugins/{userID}/{pluginName}/）を返す。
	// generate_plugin_cache がプラグインを単独モードで起動するときの --gkill-plugin-dir と
	// 実行ファイル（PluginExecutablePath）の解決に使う。
	GetPluginDir() string

	// GetRepNames はこのプラグインの記録が名乗る rep 名の全集合を返す（RepNamesProvider）。
	// get_rep_name の応答 rep_names を短時間キャッシュして返し、応答に rep_names が無い
	// （古いプラグイン・未実装）なら manifest の rep_name 1つを返す。
	// プラグインが応答しないときは前回の値か manifest の名前にフォールバックし、エラーにしない
	// （名前の列挙で検索全体を落とさないため）。
	GetRepNames(ctx context.Context) ([]string, error)

	// GetContentHTML は指定KyouIDの詳細ビューHTMLをプラグインから取得する。
	GetContentHTML(ctx context.Context, kyouID string) (string, error)

	// GetConfigHTML はプラグイン設定画面のHTMLをプラグインから取得する。
	GetConfigHTML(ctx context.Context) (string, error)

	// PostConfig はフォームデータをプラグインに送り、設定を保存させる。
	PostConfig(ctx context.Context, formData map[string]string) error

	// IsAlive はプラグインプロセスが起動・応答中かを返す。
	// 判定はping（必要ならプロセスを起動する）なので副作用がある。
	// 起動を伴わない読み取りには ProcessRunning を使うこと。
	IsAlive(ctx context.Context) bool

	// LastStderr はプラグインプロセスの stderr 末尾（直近約4KB）を返す。
	// ビルドエラー等の診断用で、何も出ていなければ空文字。
	// 「is_alive=true なのに0件」の理由をAPIから読めるようにする（外部監査 D2）。
	LastStderr() string

	// ProcessRunning はプロセスが起動済みかを副作用なしで返す。
	// IsAlive と違い、呼んでもプロセスは起動しない。
	ProcessRunning() bool

	// TypedIndex はプラグインが返した型別データ・付随データのインメモリ索引を返す。
	// manifest.jsonのprovidesが空のプラグインではnilを返す。
	// 型別リポジトリのアダプタはここからだけ読み、1件ずつプラグインへ往復しない。
	TypedIndex() *PluginTypedIndex

	// GetPluginGPSLogs は期間に含まれるGPSログをプラグインから取得する。
	// 期間の契約は GPSLogRepository.GetGPSLogs と同じ（両端を含む・nil可・逆順なら入れ替え）。
	// get_gps_logs を Offset を進めながら複数回呼んで繋ぐ。
	// providesに"gpslog"が無いプラグインではエラー。0件はエラーにしない。
	GetPluginGPSLogs(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]GPSLog, error)
}
