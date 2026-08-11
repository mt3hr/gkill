package reps

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	"golang.org/x/sync/singleflight"
)

const (
	// gpsLogPluginSnapshotTTL はスナップショットを持ち続ける時間。
	// プラグイン側のデータはユーザがエクスポートを置き直したときにしか変わらないので
	// 短くする意味は薄いが、置き直しても反映されないのは困るので5分にする。
	gpsLogPluginSnapshotTTL = 5 * time.Minute

	// gpsLogPluginFetchTimeout は全件取得に許す時間。
	// ページングで何往復もするので、1回あたりの pluginCallTimeout(30秒)より長くとる。
	gpsLogPluginFetchTimeout = 2 * time.Minute
)

// gpsLogRepositoryPluginImpl は「GPSログを提供するプラグイン」を
// GPSLogRepository として見せるアダプタです。
//
// GPSログは地図の描画1回ごと・地図フィルタ(HasMapFilter)の評価1回ごとに引かれます。
// 一方プラグインへの呼び出しは容量1のスロットで直列化されるので、
// 素通しすると列の数だけ行列が伸び、10秒待って ErrPluginBusy になります。
// そこで、期間を指定せず一度だけ取った全点をメモリに持ち、以降はそこから切り出します。
//
// 返すスライスは共有のスナップショットです。呼び出し側は要素を書き換えないこと。
// 追記で踏まないよう容量を切って返しています。
// gkill内の呼び出し元(GPSLogRepositories / find_filter.collectFromRepos)は
// どちらも append で自分のスライスへ写してから並べ替えるので問題ありません。
type gpsLogRepositoryPluginImpl struct {
	plugin PluginRepository

	// mu は snapshot と fetchedAt を守る。
	mu sync.RWMutex
	// snapshot は RelatedTime の昇順。公開後は書き換えず差し替えのみ。
	snapshot  []GPSLog
	fetchedAt time.Time

	// sf は同時に来た取得を1回にまとめる。
	// 地図の列が同時に描かれると同じ取得が数本同時に来るため。
	sf singleflight.Group
}

var _ GPSLogRepository = (*gpsLogRepositoryPluginImpl)(nil)
var _ ReadOnlyGPSLogRepository = (*gpsLogRepositoryPluginImpl)(nil)

// NewGPSLogPluginRepIfProvided は provides に "gpslog" があるプラグインだけを
// GPSLogRepository アダプタにして返します。無ければ (nil, false) を返します。
//
// 判定をここに置いているのは、呼び出し側(gkill_dao_manager)に
// gkill_plugin パッケージを持ち込ませないためです。
func NewGPSLogPluginRepIfProvided(plugin PluginRepository) (GPSLogRepository, bool) {
	if !plugin.GetManifest().ProvidesKind(gkill_plugin.PluginProvidesGPSLog) {
		return nil, false
	}
	return &gpsLogRepositoryPluginImpl{plugin: plugin}, true
}

func (g *gpsLogRepositoryPluginImpl) GetAllGPSLogs(ctx context.Context) ([]GPSLog, error) {
	snapshot := g.snapshotOf(ctx)
	return snapshot[0:len(snapshot):len(snapshot)], nil
}

func (g *gpsLogRepositoryPluginImpl) GetGPSLogs(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]GPSLog, error) {
	snapshot := g.snapshotOf(ctx)
	start, end := NormalizeGPSLogPeriod(startTime, endTime)
	if start == nil && end == nil {
		return snapshot[0:len(snapshot):len(snapshot)], nil
	}

	// 昇順に並べてあるので二分探索で切り出す。期間は他の時刻フィルタと同じく両端を含む。
	low := 0
	if start != nil {
		low = sort.Search(len(snapshot), func(i int) bool { return !snapshot[i].RelatedTime.Before(*start) })
	}
	high := len(snapshot)
	if end != nil {
		high = sort.Search(len(snapshot), func(i int) bool { return snapshot[i].RelatedTime.After(*end) })
	}
	if low >= high {
		return []GPSLog{}, nil
	}
	return snapshot[low:high:high], nil
}

func (g *gpsLogRepositoryPluginImpl) GetPath(ctx context.Context, _ string) (string, error) {
	return g.plugin.GetPath(ctx, "")
}

func (g *gpsLogRepositoryPluginImpl) GetRepName(ctx context.Context) (string, error) {
	return g.plugin.GetRepName(ctx)
}

// UpdateCache はメモリ上のスナップショットを捨て、次の取得で引き直させます。
// GkillRepositories.UpdateCache は GPSLogReps を回さないので現状この経路からは
// 呼ばれず、無効化は実質TTLが担っています。
func (g *gpsLogRepositoryPluginImpl) UpdateCache(_ context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.snapshot, g.fetchedAt = nil, time.Time{}
	return nil
}

func (g *gpsLogRepositoryPluginImpl) UnWrapTyped() ([]GPSLogRepository, error) {
	return []GPSLogRepository{g}, nil
}

// IsReadOnlyGPSLogRepository はアップロード先に選べないことを表す目印です。
func (g *gpsLogRepositoryPluginImpl) IsReadOnlyGPSLogRepository() {}

// snapshotOf はスナップショットを返します。取得に失敗しても空を返し、エラーにはしません。
//
// GPSLogRepositories.GetGPSLogs も find_filter の collectFromRepos も
// 「1つでもrepがエラーを返したら全体を nil, err にする」作りなので、
// ここでエラーを返すと、プラグインが混んでいるだけで地図も検索も丸ごと落ちます。
// プラグイン障害で検索全体を落とさないのは FindKyous と同じ方針で、
// 静かな欠落にならないよう AppendPluginFindWarning に記録します。
func (g *gpsLogRepositoryPluginImpl) snapshotOf(ctx context.Context) []GPSLog {
	g.mu.RLock()
	if g.snapshot != nil && time.Since(g.fetchedAt) < gpsLogPluginSnapshotTTL {
		snapshot := g.snapshot
		g.mu.RUnlock()
		return snapshot
	}
	g.mu.RUnlock()

	fetched, err, _ := g.sf.Do("snapshot", func() (any, error) {
		// 先頭の呼び出し元がキャンセルされても、相乗りした全員を巻き添えにしない。
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), gpsLogPluginFetchTimeout)
		defer cancel()

		gpsLogs, err := g.plugin.GetPluginGPSLogs(fetchCtx, nil, nil)
		if err != nil {
			return nil, err
		}
		slices.SortFunc(gpsLogs, func(a, b GPSLog) int { return a.RelatedTime.Compare(b.RelatedTime) })

		g.mu.Lock()
		g.snapshot, g.fetchedAt = gpsLogs, time.Now()
		g.mu.Unlock()
		return gpsLogs, nil
	})
	if err != nil {
		name := g.plugin.GetManifest().Name
		slog.Warn(fmt.Sprintf("plugin gps log fetch error %q: %q", name, err))
		AppendPluginFindWarning(ctx, name)
		return []GPSLog{}
	}
	return fetched.([]GPSLog)
}
