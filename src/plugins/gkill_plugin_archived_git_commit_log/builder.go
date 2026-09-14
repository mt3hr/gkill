package main

// 同期・単一tx構築を書かない理由（進捗ゼロループ）と却下案:
// documents/adr/0305-plugin-background-builder-wal.md

import (
	"context"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// ビルダはプロセス内に1本だけ常駐する goroutine。
// ハンドラは kick を投げて、即座に「今キャッシュにあるもの」を返す。
//
// gkillのハンドラ期限は「30秒以内なら良い」ではなく「数十ミリ秒で返れ」を意味する。
//   - IsAlive の期限は5秒で、超えるとプロセスが殺される
//   - 一覧は find_kyous 1回のあとに行数ぶんの呼び出しが1本のスロットに並ぶ
//
// 実データ（88 zip・3,447 コミット）の初回構築は go-git の行数集計を含めて約 60 秒かかり、
// 同期でやると必ず殺される。codex / fitbit と同じ常駐ビルダにする。

// builderIdleInterval は何も無くても様子を見に行く間隔。
// アーカイブは基本不変なので、2回目以降は指紋の比較だけで終わる。
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

// runOnce は走査→取り込みを1周する。
//
// os.Stdout には絶対に書かない。あれはプロトコルのチャネルで、
// 1行でも混ざるとJSONストリームが壊れる。ログはstderrに出す。
func (b *builder) runOnce(pluginDir string, config pluginConfig) {
	if err := globalCache.build(context.Background(), pluginDir, config); err != nil {
		globalCache.setMeta("build_state", "error")
		globalCache.setMeta("build_error", err.Error())
		sdk.LogError("%s: build error: %v", appName, err)
	}
}

// startBuilder はハンドラの先頭から呼ぶ。起動していなければ起こし、
// 起動済みなら作り直しを促すだけで、待たずに戻る。
func startBuilder(pluginDir string, configOf func() pluginConfig) {
	globalBuilder.EnsureStarted(pluginDir, configOf)
	globalBuilder.Kick()
}
