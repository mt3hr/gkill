package main

// 同期・単一tx構築を書かない理由（進捗ゼロループ）と却下案:
// documents/adr/0024-plugin-background-builder-wal.md

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// ビルダはプロセス内に1本だけ常駐する goroutine。
// ハンドラは kick を投げて、即座に「今キャッシュにあるもの」を返す。
//
// gkillのハンドラ期限は「30秒以内なら良い」ではなく「数十ミリ秒で返れ」を意味する。
//   - IsAlive の期限は5秒で、超えるとプロセスが殺される
//   - 一覧は find_kyous 1回のあとに行数ぶんの get_content_html が
//     1本のスロットに並ぶので、3秒かかるハンドラでも4件目から順番待ちが破綻する
//
// 実データ(52ファイル・245MB)の初回構築は数十秒かかる。同期でやると必ず殺される。
// 参考にした gkill_plugin_claudecode は同期で走査しているが、あれは真似しないこと。

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
	b.runOnce(pluginDir, configOf())

	for {
		select {
		case <-b.kick:
		case <-ticker.C:
		}
		b.runOnce(pluginDir, configOf())
	}
}

// runOnce は走査→取り込み→畳み直しを1周する。
//
// os.Stdout には絶対に書かない。あれはプロトコルのチャネルで、
// 1行でも混ざるとJSONストリームが壊れる。ログはstderrに出す。
func (b *builder) runOnce(pluginDir string, config pluginConfig) {
	if err := globalCache.build(pluginDir, config); err != nil {
		globalCache.setMeta("build_state", "error")
		globalCache.setMeta("build_error", err.Error())
		fmt.Fprintf(os.Stderr, "%s: build error: %v\n", appName, err)
	}
}

// startBuilder はハンドラの先頭から呼ぶ。起動していなければ起こし、
// 起動済みなら作り直しを促すだけで、待たずに戻る。
func startBuilder(pluginDir string, configOf func() pluginConfig) {
	globalBuilder.EnsureStarted(pluginDir, configOf)
	globalBuilder.Kick()
}
