package reps

import (
	"context"

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
}
