package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// appName はstderrログの識別子。
const appName = "gkill_plugin_claudeai"

// ビルダはプロセス内に1本だけ常駐する goroutine。
// ハンドラは kick を投げて、即座に「今キャッシュにあるもの」を返す。
//
// gkillのハンドラ期限は「30秒以内なら良い」ではなく「数十ミリ秒で返れ」を意味する。
//   - IsAlive の期限は5秒で、超えるとプロセスが殺される
//   - 一覧は find_kyous 1回のあとに行数ぶんの get_content_html が
//     1本のスロットに並ぶので、3秒かかるハンドラでも4件目から順番待ちが破綻する
//
// 大きなエクスポートの取り込みは数十秒かかる。同期でやると必ず殺される。
// 以前の cache.go は GetMessages が単一ロック下で rebuild を同期実行しており、
// デッドラインで kill→ロールバック→進捗ゼロ→次の find_kyous でまた最初から、の
// 無限ループに陥っていた。同梱の gkill_plugin_codex / gkill_plugin_claudecode の
// 常駐ビルダ方式へ揃える。

// builderIdleInterval は何も無くても様子を見に行く間隔。
const builderIdleInterval = 5 * time.Minute

type builder struct {
	// kick はバッファ1。ノンブロッキング送信で「作り直して」と伝える。
	kick      chan struct{}
	startOnce sync.Once
}

var globalBuilder = &builder{kick: make(chan struct{}, 1)}

// EnsureStarted はビルダを起動する。何度呼んでも1本しか起きない。
func (b *builder) EnsureStarted(pluginDir string, sourceOf func() expandedSource) {
	b.startOnce.Do(func() {
		go b.loop(pluginDir, sourceOf)
	})
}

// Kick は作り直しを促す。待たない。
func (b *builder) Kick() {
	select {
	case b.kick <- struct{}{}:
	default:
	}
}

func (b *builder) loop(pluginDir string, sourceOf func() expandedSource) {
	ticker := time.NewTicker(builderIdleInterval)
	defer ticker.Stop()

	// 起動直後に1回走らせる
	b.runOnce(pluginDir, sourceOf())

	for {
		select {
		case <-b.kick:
		case <-ticker.C:
		}
		b.runOnce(pluginDir, sourceOf())
	}
}

// runOnce は走査→取り込み→掃除を1周する。
//
// os.Stdout には絶対に書かない。あれはプロトコルのチャネルで、
// 1行でも混ざるとJSONストリームが壊れる。ログはstderrに出す。
func (b *builder) runOnce(pluginDir string, src expandedSource) {
	if err := globalCache.build(pluginDir, src); err != nil {
		globalCache.setMeta("build_state", "error")
		globalCache.setMeta("build_error", err.Error())
		fmt.Fprintf(os.Stderr, "%s: build error: %v\n", appName, err)
	}
}

// startBuilder はハンドラの先頭から呼ぶ。起動していなければ起こし、
// 起動済みなら作り直しを促すだけで、待たずに戻る。
func startBuilder(pluginDir string, sourceOf func() expandedSource) {
	globalBuilder.EnsureStarted(pluginDir, sourceOf)
	globalBuilder.Kick()
}
