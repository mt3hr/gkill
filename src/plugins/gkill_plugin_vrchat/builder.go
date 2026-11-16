package main

// 同期・単一tx構築を書かない理由（進捗ゼロループ）と却下案:
// documents/adr/0305-plugin-background-builder-wal.md

import (
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// ビルダはプロセス内に1本だけ常駐する goroutine。gkill_plugin_codex と同じ形。
// ハンドラは kick を投げて、即座に「今キャッシュにあるもの」を返す。
//
// gkillのハンドラ期限は「数十ミリ秒で返れ」を意味する。
//   - IsAlive の期限は5秒で、超えるとプロセスが殺される
//   - 一覧は find_kyous のあとに行数ぶんの呼び出しが1本のスロットに並ぶ

// builderIdleInterval は何も無くても様子を見に行く間隔。
const builderIdleInterval = 5 * time.Minute

type builder struct {
	// kick はバッファ1。ノンブロッキング送信で「作り直して」と伝える。
	kick      chan struct{}
	startOnce sync.Once
}

var globalBuilder = &builder{kick: make(chan struct{}, 1)}

// EnsureStarted はビルダを起動する。何度呼んでも1本しか起きない。
func (b *builder) EnsureStarted(pluginDir string, configOf func() pluginConfig) {
	b.startOnce.Do(func() {
		go b.loop(pluginDir, configOf)
	})
}

// Kick は作り直しを促す。待たない。
func (b *builder) Kick() {
	select {
	case b.kick <- struct{}{}:
	default:
	}
}

func (b *builder) loop(pluginDir string, configOf func() pluginConfig) {
	ticker := time.NewTicker(builderIdleInterval)
	defer ticker.Stop()

	// 起動直後に1回走らせる
	_ = buildOnce(pluginDir, configOf())

	for {
		select {
		case <-b.kick:
		case <-ticker.C:
		}
		_ = buildOnce(pluginDir, configOf())
	}
}

// buildCacheFn は buildOnce が呼ぶ本体。テストが差し替えて失敗経路（cache_meta への記録と ERROR 行）を見る。
var buildCacheFn = func(pluginDir string, config pluginConfig) error { return globalCache.build(pluginDir, config) }

// buildOnce は走査→取り込み→記録の作り直しを1周し、失敗を cache_meta に残してから返す。
// 常駐ビルダと単独モード（Handler.BuildCache。gkill_server generate_plugin_cache）の
// 両方がここを通るので、どちらで失敗しても設定画面の build_state は同じ見え方になる。
//
// os.Stdout には絶対に書かない。あれはプロトコルのチャネルで、
// 1行でも混ざるとJSONストリームが壊れる。ログは sdk.LogXxx に出す。
func buildOnce(pluginDir string, config pluginConfig) error {
	started := time.Now()
	if err := buildCacheFn(pluginDir, config); err != nil {
		globalCache.setMeta("build_state", "error")
		globalCache.setMeta("build_error", err.Error())
		sdk.LogError("%s: build error: %v", appName, err)
		return err
	}
	sdk.LogInfo("%s: build done in %s", appName, time.Since(started).Round(time.Millisecond))
	return nil
}

// startBuilder はハンドラの先頭から呼ぶ。起動していなければ起こし、
// 起動済みなら作り直しを促すだけで、待たずに戻る。
func startBuilder(pluginDir string, configOf func() pluginConfig) {
	globalBuilder.EnsureStarted(pluginDir, configOf)
	globalBuilder.Kick()
}
