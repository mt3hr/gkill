package reps

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// updateCacheSlowRepThreshold はrep単位の所要時間をログに出す下限です。
// リポジトリ定義がglobで数百に展開される利用者がいるので、
// 全rep分を出すとログが流れてしまいます。遅いものだけ残します。
const updateCacheSlowRepThreshold = 500 * time.Millisecond

// updateCachePhaseTimer は UpdateCache のフェーズ別所要時間を測ります。
//
// キャッシュ再構築が遅いという申告は「どのフェーズが遅いか」が分からないと
// 手の付けようがありません。実データ（リポジトリ数百・数十万行・外付けHDD）でしか
// 再現しないので、合成データのベンチではなく本番のログで切り分けられるようにしてあります。
type updateCachePhaseTimer struct {
	start  time.Time
	last   time.Time
	phases []updateCachePhase
}

// updateCachePhase はフェーズ1つぶんの所要時間です。
type updateCachePhase struct {
	name    string
	elapsed time.Duration
}

func newUpdateCachePhaseTimer() *updateCachePhaseTimer {
	now := time.Now()
	return &updateCachePhaseTimer{start: now, last: now}
}

// mark は直前のmarkからここまでを1フェーズとして記録します。
func (t *updateCachePhaseTimer) mark(name string) {
	now := time.Now()
	t.phases = append(t.phases, updateCachePhase{name: name, elapsed: now.Sub(t.last)})
	t.last = now
}

// logArgs はslogへ渡す key/value の並びを作ります。
func (t *updateCachePhaseTimer) logArgs() []any {
	args := make([]any, 0, len(t.phases)*2+2)
	args = append(args, "elapsed", time.Since(t.start).String())
	for _, phase := range t.phases {
		args = append(args, phase.name, phase.elapsed.String())
	}
	return args
}

// logUpdateCacheDone は UpdateCache 1回ぶんの結果を1行で出します。
// 1回につき1行なので常時Infoで出しても流れません。
func (t *updateCachePhaseTimer) logUpdateCacheDone(ctx context.Context) {
	slog.Log(ctx, gkill_log.Info, "update_cache done", t.logArgs()...)
}

// repUpdateCacheStat はrep単位の所要時間です。
type repUpdateCacheStat struct {
	repName string
	elapsed time.Duration
}

// logSlowRepUpdateCaches は遅かったrepだけをDebugで出します。
// 「どの種別のどのリポジトリが支配的か」を実データで特定するためのものです。
func logSlowRepUpdateCaches(ctx context.Context, stats []repUpdateCacheStat) {
	slow := make([]repUpdateCacheStat, 0, len(stats))
	for _, stat := range stats {
		if stat.elapsed >= updateCacheSlowRepThreshold {
			slow = append(slow, stat)
		}
	}
	if len(slow) == 0 {
		return
	}
	slices.SortFunc(slow, func(a, b repUpdateCacheStat) int {
		return cmp.Compare(b.elapsed, a.elapsed)
	})
	for _, stat := range slow {
		slog.Log(ctx, gkill_log.Debug, "update_cache slow rep",
			"rep_name", stat.repName,
			"elapsed", stat.elapsed.String())
	}
}
