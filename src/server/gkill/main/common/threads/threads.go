// Package threads はNumCPU有界セマフォによるゴルーチン数制限。
package threads

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

var (
	sem  chan struct{}
	once sync.Once
)

// acquireWaitBeforeInline はプールに空きが無いときに空きを待つ上限です。
// これを超えたら呼び出し元goroutineでそのまま実行します(inlineフォールバック)。
//
// これは「混んでいる」と「入れ子で永久に空かない」を見分けるための閾値です。
// 短くすると前者でもフォールバックしてしまい、並列度を自分から捨てることになります。
// 実データ(リポジトリ940個・外付けHDD)では1repの処理が秒単位かかるため、
// 100ms にしていたときは通常のキャッシュ更新中に49秒で486回フォールバックしました。
// 入れ子のデッドロックは待っても永久に解けないので、閾値は大きくても困りません。
//
// テストから短縮できるよう定数ではなく変数にしています。
var acquireWaitBeforeInline = 30 * time.Second

const (
	// saturatedWindow は枯渇を検知したあと、空きを待たずにinlineへ倒す猶予期間です。
	// fan-outループの要素ごとに毎回 acquireWaitBeforeInline を払うと
	// 「rep数 × 閾値」のレイテンシを自分で作ってしまうため、
	// 一度枯渇を見たらしばらくは待たずに逐次実行へ倒します。
	saturatedWindow = 500 * time.Millisecond

	// inlineFallbackLogInterval はinlineフォールバックの警告ログを出す間隔です。
	// 枯渇時は大量に発生するので、累計はカウンタで見てログ自体は間引きます。
	inlineFallbackLogInterval = 10 * time.Second
)

var (
	// saturatedUntilUnixNano は「この時刻まではプールが枯渇しているとみなす」時刻です。
	saturatedUntilUnixNano atomic.Int64

	// inlineFallbackCount はinlineフォールバックした累計回数です。
	inlineFallbackCount atomic.Uint64

	// lastInlineFallbackLogUnixNano は最後に警告ログを出した時刻です。
	lastInlineFallbackLogUnixNano atomic.Int64
)

// mainから明示的に呼び出してください
func Init() {
	once.Do(func() {
		n := gkill_options.GoroutinePool
		if n <= 0 {
			n = 1
		}
		sem = make(chan struct{}, n)
	})
}

func Acquire(ctx context.Context) (release func(), err error) {
	if sem == nil {
		Init()
	}
	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	case <-ctx.Done():
		return func() {}, ctx.Err()
	}
}

// Go はfnをスレッドプールのスロット付きで非同期実行します。
//
// スロットは呼び出し元goroutine上で同期取得します。したがって
// 「スロットを保持した親が、子のスロットが空くのを待つ」入れ子を作ると
// プールが枯渇します。入れ子になる経路では集約リポジトリの逐次版
// (FindKyousSequential など) を使ってください。
//
// 枯渇したときは恒久ハングを避けるため、空きを待たずに
// fnを呼び出し元goroutineでそのまま実行します(inlineフォールバック)。
// このとき並列度は落ちますが、処理は必ず前に進みます。
// あくまで保険なので、これがあるからといって入れ子を作ってよいわけではありません。
//
// inline実行が安全であることは「fnは要素数ぶんのバッファを持つチャネルへ
// 結果を1件送るだけで、それ以外ではブロックしない」という前提に依存します。
// 集約リポジトリのfan-outは全てこの形です。新しい呼び出し箇所を足すときも
// この前提を守ってください。
func Go(ctx context.Context, wg *sync.WaitGroup, fn func()) error {
	if sem == nil {
		Init()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// 空きがあれば即座にスロットを取る
	select {
	case sem <- struct{}{}:
		goWithSlot(wg, fn)
		return nil
	default:
	}

	// 直前に枯渇を見ているなら、待たずに逐次実行へ倒す
	if isSaturated() {
		runInline(fn)
		return nil
	}

	timer := time.NewTimer(acquireWaitBeforeInline)
	defer timer.Stop()
	select {
	case sem <- struct{}{}:
		goWithSlot(wg, fn)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		markSaturated()
		runInline(fn)
		return nil
	}
}

// goWithSlot はスロットを取得済みの状態でfnを非同期実行します。
func goWithSlot(wg *sync.WaitGroup, fn func()) {
	wg.Go(func() {
		defer func() { <-sem }()
		fn()
	})
}

// runInline はスロットを取らずに呼び出し元goroutineでfnを実行します。
func runInline(fn func()) {
	inlineFallbackCount.Add(1)
	logInlineFallback()
	fn()
}

// isSaturated は直近で枯渇を検知したかを返します。
func isSaturated() bool {
	return time.Now().UnixNano() < saturatedUntilUnixNano.Load()
}

// markSaturated は枯渇を検知したことを記録します。
func markSaturated() {
	saturatedUntilUnixNano.Store(time.Now().Add(saturatedWindow).UnixNano())
}

// logInlineFallback は警告ログを間引いて出します。
// 枯渇は入れ子という構造上の問題を示すので、気づけるようにしておきます。
func logInlineFallback() {
	now := time.Now().UnixNano()
	last := lastInlineFallbackLogUnixNano.Load()
	if now-last < int64(inlineFallbackLogInterval) {
		return
	}
	if !lastInlineFallbackLogUnixNano.CompareAndSwap(last, now) {
		return
	}
	slog.Log(context.Background(), gkill_log.Warn, "goroutine pool exhausted, running inline",
		"pool_size", cap(sem),
		"inline_fallback_count", inlineFallbackCount.Load())
}

// InlineFallbackCount はinlineフォールバックした累計回数を返します。
// 枯渇が実際に起きているかの確認用です。
func InlineFallbackCount() uint64 {
	return inlineFallbackCount.Load()
}
