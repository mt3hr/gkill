package main

import (
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// ビルダはプロセス内に1本だけ常駐する goroutine。
// ハンドラは kick を投げて、即座に「今キャッシュにあるもの」を返す。
//
// gkillのハンドラ期限は「30秒以内なら良い」ではなく「数十ミリ秒で返れ」を意味する。
//   - IsAlive の期限は5秒で、超えるとプロセスが殺される
//   - 一覧は find_kyous 1回のあとに行数ぶんの get_content_html が
//     1本のスロットに並ぶので、3秒かかるハンドラでも4件目から順番待ちが破綻する
//
// 同期・単一トランザクションの構築を新規に書かないこと（ADR-0305）。
// デッドラインで kill されるとロールバックして進捗ゼロに戻り、
// 次の find_kyous でまた最初から、の無限ループになる。

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

// runOnce は走査→取り込みを1周する。
//
// os.Stdout には絶対に書かない。あれはプロトコルのチャネルで、
// 1行でも混ざるとJSONストリームが壊れる。
// ログは sdk.LogXxx（stderr + $GKILL_HOME/logs の gkill_log）に出す。
func (b *builder) runOnce(pluginDir string, config pluginConfig) {
	_ = buildOnce(pluginDir, config)
}

// buildCacheFn は buildOnce が呼ぶ本体。
// テストが差し替えて失敗経路（cache_meta への記録と ERROR 行）を見る。
var buildCacheFn = func(pluginDir string, config pluginConfig) error {
	return globalCache.build(pluginDir, config)
}

// buildOnce は走査→取り込みを1周し、失敗を cache_meta に残してから返す。
// 常駐ビルダ（runOnce）と単独モード（Handler.BuildCache。gkill_server generate_plugin_cache）の
// 両方がここを通るので、どちらで失敗しても設定画面の build_state は同じ見え方になる。
func buildOnce(pluginDir string, config pluginConfig) error {
	started := time.Now()
	if err := buildCacheFn(pluginDir, config); err != nil {
		globalCache.setMeta("build_state", "error")
		globalCache.setMeta("build_error", err.Error())
		sdk.LogError("%s: build error: %v", appName, err)
		return err
	}
	// 構築完了は節目。ログファイルにだけ残り、stderr の last_error は汚さない
	sdk.LogInfo("%s: build done in %s", appName, time.Since(started).Round(time.Millisecond))
	return nil
}

// startBuilder はハンドラの先頭から呼ぶ。起動していなければ起こし、
// 起動済みなら作り直しを促すだけで、待たずに戻る。
func startBuilder(pluginDir string, configOf func() pluginConfig) {
	globalBuilder.EnsureStarted(pluginDir, configOf)
	globalBuilder.Kick()
}
