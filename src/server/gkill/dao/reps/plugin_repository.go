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

	// GetContentHTML は指定KyouIDの詳細ビューHTMLをプラグインから取得する。
	GetContentHTML(ctx context.Context, kyouID string) (string, error)

	// GetConfigHTML はプラグイン設定画面のHTMLをプラグインから取得する。
	GetConfigHTML(ctx context.Context) (string, error)

	// PostConfig はフォームデータをプラグインに送り、設定を保存させる。
	PostConfig(ctx context.Context, formData map[string]string) error

	// IsAlive はプラグインプロセスが起動・応答中かを返す。
	IsAlive(ctx context.Context) bool

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
