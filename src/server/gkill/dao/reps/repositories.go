package reps

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/threads"
)

type Repositories []Repository

// goForRep はrepの種類に応じてスレッドプールを使うかを決めます。
//
// プラグインはサブプロセスの応答待ちでCPUを使わず、かつ1プラグイン1並列に
// プラグイン側のミューテックスで直列化されています。
// ここでプールのスロットを占有させると、検索1本がプラグインのロック待ちで
// 最大30秒スロットを握り続け、他の全リクエストがプール枯渇で止まります。
// 実際にrykvがプラグインKyouの本文を大量に要求したときにこれが起きました。
func goForRep(ctx context.Context, wg *sync.WaitGroup, rep Repository, fn func()) error {
	if _, isPlugin := rep.(PluginRepository); isPlugin {
		if err := ctx.Err(); err != nil {
			return err
		}
		wg.Go(fn)
		return nil
	}
	return threads.Go(ctx, wg, fn)
}

func (r Repositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	return r.findKyous(ctx, query, true)
}

// maxIDsPerFindQuery は1回の検索でIDリストに載せる上限です。
//
// IDリストは各repのSQLへ ID IN (?, ?, ...) として展開されます。
// このときMiは5射影のUNIONで、**5本それぞれに同じリストを丸ごと展開する**ので
// (mi_repository_sqlite3_impl.go)、バインド変数は 5N+5 になり、
// SQLiteの上限(SQLITE_MAX_VARIABLE_NUMBER=32766)を **N=6553で超えます**。
// 超えるとPrepareが失敗し、検索が丸ごと落ちます。
//
// IDリストはORの羅列なので、分割して和を取れば結果は変わりません。
// 4000なら5倍展開でも20005で収まり、将来もっと展開する実装が現れても余裕があります。
//
// 2026-08-18に実データ(確認待ちの記録7,122件のIDを一度に渡した)で踏みました。
// そのときは失敗がGkillErrorにならず「成功・0件」に見えていました
// (message.EnsureNotEmptyのコメントを参照)。
//
// 却下案（分割せずエラーにする／UNIONを畳む／呼び出し側で分割する）と経緯:
// documents/adr/0005-chunk-find-query-ids.md
const maxIDsPerFindQuery = 4000

// findChunkedByIDs はIDリストをmaxIDsPerFindQueryずつに割って検索し、結果を連結します。
//
// IDリストはORの羅列なので、分割しても結果は変わりません
// (同じIDが2つの塊に入ることは無いので、重複も生まれません)。
// 分割が要らない件数のときはそのまま1回呼ぶだけなので、
// IDリストを渡しうる検索は常にこれを通して構いません。
func findChunkedByIDs[T any](ctx context.Context, query *find.FindQuery, findFunc func(context.Context, *find.FindQuery) ([]T, error)) ([]T, error) {
	if len(query.IDs) <= maxIDsPerFindQuery {
		return findFunc(ctx, query)
	}

	collected := []T{}
	for idsChunk := range slices.Chunk(query.IDs, maxIDsPerFindQuery) {
		clonedQuery := *query
		clonedQuery.IDs = idsChunk

		found, err := findFunc(ctx, &clonedQuery)
		if err != nil {
			return nil, err
		}
		collected = append(collected, found...)
	}
	return collected, nil
}

// findKyousByChunkedIDs はIDリストをmaxIDsPerFindQueryずつに割って検索し、和を返します。
//
// IDリストはORの羅列なので、分割しても結果は変わりません
// (同じIDが2つの塊に入ることは無いので、重複も生まれません)。
func (r Repositories) findKyousByChunkedIDs(ctx context.Context, query *find.FindQuery, parallel bool) (map[string][]Kyou, error) {
	// キャッシュ再構築は分割の前に1回だけ行う。
	// 塊ごとに走らせると同じ再構築を塊の数だけ繰り返すことになる。
	if query.UpdateCache {
		if err := r.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}

	matchKyous := map[string][]Kyou{}
	for idsChunk := range slices.Chunk(query.IDs, maxIDsPerFindQuery) {
		clonedQuery := *query
		clonedQuery.IDs = idsChunk
		clonedQuery.UpdateCache = false

		matchKyousInChunk, err := r.findKyous(ctx, &clonedQuery, parallel)
		if err != nil {
			return nil, err
		}
		for key, kyous := range matchKyousInChunk {
			matchKyous[key] = append(matchKyous[key], kyous...)
		}
	}
	return matchKyous, nil
}

// FindKyousSequential は各リポジトリを並列化せずに順に検索します。
//
// あるリポジトリのFindKyousの中からさらにRepositories.FindKyousを呼ぶ場面
// （ReKyou/MiReKyouがワード検索をターゲットへ委譲するとき）で使ってください。
// threads.Goはプールのスロットを呼び出し元で同期取得するため、
// スロットを保持したまま子のスロットを待つ入れ子を作るとプールが枯渇して止まります。
func (r Repositories) FindKyousSequential(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	return r.findKyous(ctx, query, false)
}

func (r Repositories) findKyous(ctx context.Context, query *find.FindQuery, parallel bool) (map[string][]Kyou, error) {
	// IDを渡されすぎているときは分割して検索する。理由はmaxIDsPerFindQueryを参照。
	if len(query.IDs) > maxIDsPerFindQuery {
		return r.findKyousByChunkedIDs(ctx, query, parallel)
	}

	// update_cache=trueの場合、並列dispatch前に逐次UpdateCacheする。
	// threads.Goのネスト（FindKyous並列→UpdateCache内の並列）でスレッドプールが枯渇しデッドロックするのを防止する。
	if query.UpdateCache {
		if err := r.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
		clonedQuery := *query
		clonedQuery.UpdateCache = false
		query = &clonedQuery
	}

	matchKyous := map[string][]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan map[string][]Kyou, len(r))
	errch := make(chan error, len(r))
	defer close(ch)
	defer close(errch)

	// 並列処理（入れ子から呼ばれたときは逐次）
	for _, rep := range r {
		findInRep := func() {
			matchKyousInRep, err := rep.FindKyous(ctx, query)
			if err != nil {
				repName, _ := rep.GetRepName(ctx)
				err = fmt.Errorf("error at %s: %w", repName, err)
				errch <- err
				return
			}
			ch <- matchKyousInRep
		}
		if !parallel {
			findInRep()
			continue
		}
		err := goForRep(ctx, wg, rep, findInRep)
		if err != nil {
			errch <- err
		}
	}
	wg.Wait()

	// エラー集約。複数repが同時に失敗しても全部残す
	// (以前は最後の1件で上書きされ、他のrepの故障が見えなかった)
errloop:
	for {
		select {
		case e := <-errch:
			err = errors.Join(err, fmt.Errorf("error at find kyous: %w", e))
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Kyou集約。キーごとに全repの全entryをappendする(最新版の選択は後段のfind_filterが行う)
loop:
	for {
		select {
		case matchKyousInRep := <-ch:
			if matchKyousInRep == nil {
				continue loop
			}

			for _, kyous := range matchKyousInRep {
				for _, kyou := range kyous {
					key := kyou.ID
					if !query.OnlyLatestData {
						key += strconv.FormatInt(kyou.UpdateTime.Unix(), 10)
					}
					// 空スライスの事前確保はしない。存在しないキーへの append は
					// nilスライスに対して働くので結果は同じで、
					// レコード1件につき1回の無駄な確保(実データで56万回)が消える。
					matchKyous[key] = append(matchKyous[key], kyou)
				}
			}
		default:
			break loop
		}
	}
	return matchKyous, nil
}

func (r Repositories) Close(ctx context.Context) error {
	reps, err := r.UnWrap()
	if err != nil {
		return err
	}

	existErr := false
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(reps))
	defer close(errch)

	// 並列処理
	for _, rep := range reps {
		err := goForRep(ctx, wg, rep, func() {
			// クロージャの外の err に書くと全goroutineが同じ変数を書き潰す (go test -race で落ちる)
			err := rep.Close(ctx)
			if err != nil {
				errch <- err
				return
			}
		})
		if err != nil {
			errch <- err
		}
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = errors.Join(err, fmt.Errorf("error at close: %w", e))
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return err
	}

	return nil
}

func (r Repositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var matchKyou *Kyou
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(r))
	errch := make(chan error, len(r))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r {
		err := goForRep(ctx, wg, rep, func() {
			matchKyouInRep, err := rep.GetKyou(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKyouInRep
		})
		if err != nil {
			errch <- err
		}
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = errors.Join(err, fmt.Errorf("error at get kyou: %w", e))
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Kyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKyouInRep := <-ch:
			if matchKyouInRep == nil {
				continue loop
			}
			if matchKyou != nil {
				if matchKyouInRep.UpdateTime.After(matchKyou.UpdateTime) {
					matchKyou = matchKyouInRep
				}
			} else {
				matchKyou = matchKyouInRep
			}
		default:
			break loop
		}
	}

	return matchKyou, nil
}

func (r Repositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(r))
	defer close(errch)

	// rep単位の所要時間。どのリポジトリが支配的かを実データで特定するために測る。
	// 遅かったものだけをログに出す。
	statCh := make(chan repUpdateCacheStat, len(r))
	defer close(statCh)

	// UpdateCache並列処理（threads.Goは内部でネストするためセマフォデッドロック回避のため素のgoroutineを使用）
	for _, rep := range r {
		wg.Go(func() {
			start := time.Now()
			if e := rep.UpdateCache(ctx); e != nil {
				errch <- e
			}
			repName, _ := rep.GetRepName(ctx)
			statCh <- repUpdateCacheStat{repName: repName, elapsed: time.Since(start)}
		})
	}
	wg.Wait()

	stats := make([]repUpdateCacheStat, 0, len(r))
statloop:
	for {
		select {
		case stat := <-statCh:
			stats = append(stats, stat)
		default:
			break statloop
		}
	}
	logSlowRepUpdateCaches(ctx, stats)

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = errors.Join(err, fmt.Errorf("error at update cache: %w", e))
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return err
	}
	return nil
}

func (r Repositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}

	ids := []string{id}
	for _, rep := range r {
		query := &find.FindQuery{
			IDs: ids,
		}
		kyous, err := rep.FindKyous(ctx, query)
		if len(kyous) == 0 || err != nil {
			continue
		}
		matchPathInRep, err := rep.GetPath(ctx, id)
		if err != nil {
			continue
		}
		matchPaths = append(matchPaths, matchPathInRep)
	}
	if len(matchPaths) == 0 {
		return "", fmt.Errorf("not found path for id: %s", id)
	}
	return matchPaths[0], nil
}

func (r Repositories) GetRepName(ctx context.Context) (string, error) {
	return "Reps", nil
}

func (r Repositories) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	kyouHistories := map[string]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Kyou, len(r))
	errch := make(chan error, len(r))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r {
		err := goForRep(ctx, wg, rep, func() {
			matchKyousInRep, err := rep.GetKyouHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKyousInRep
		})
		if err != nil {
			errch <- err
		}
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = errors.Join(err, fmt.Errorf("error at get kyou histories: %w", e))
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Kyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKyousInRep := <-ch:
			for _, kyou := range matchKyousInRep {
				if existKyou, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existKyou.UpdateTime) {
						kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)] = kyou
					}
				} else {
					kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)] = kyou
				}
			}
		default:
			break loop
		}
	}

	kyouHistoriesList := []Kyou{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	slices.SortFunc(kyouHistoriesList, func(a, b Kyou) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (r Repositories) GetKyouHistoriesByRepName(ctx context.Context, id string, repName *string) ([]Kyou, error) {
	repImpls, err := r.UnWrap()
	if err != nil {
		return nil, err
	}

	kyouHistories := map[string]Kyou{}
	existErr := false
	wg := &sync.WaitGroup{}
	ch := make(chan []Kyou, len(repImpls))
	errch := make(chan error, len(repImpls))
	defer close(ch)
	defer close(errch)

	// 並列処理。
	// repNameの絞り込みはdispatchの前に済ませる。goroutineの中でやると
	// 一致しないrepのぶんまでスレッドプールのスロットを取ってしまい、
	// rep数が多い利用者では1リクエストでrep数ぶんのセマフォ往復が発生する。
	for _, rep := range repImpls {
		if repName != nil {
			repNameInRep, err := rep.GetRepName(ctx)
			if err != nil {
				errch <- fmt.Errorf("error at get rep name: %w", err)
				continue
			}
			if repNameInRep != *repName {
				continue
			}
		}

		err := goForRep(ctx, wg, rep, func() {
			matchKyousInRep, err := rep.GetKyouHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKyousInRep
		})
		if err != nil {
			errch <- err
		}
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = errors.Join(err, fmt.Errorf("error at get kyou histories: %w", e))
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Kyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKyousInRep := <-ch:
			for _, kyou := range matchKyousInRep {
				if existKyou, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existKyou.UpdateTime) {
						kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)] = kyou
					}
				} else {
					kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)] = kyou
				}
			}
		default:
			break loop
		}
	}

	kyouHistoriesList := []Kyou{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	slices.SortFunc(kyouHistoriesList, func(a, b Kyou) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (r Repositories) UnWrap() ([]Repository, error) {
	repositories := []Repository{}
	for _, rep := range r {
		unwraped, err := rep.UnWrap()
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, unwraped...)
	}
	return repositories, nil
}

func (r Repositories) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	allAddrs := []gkill_cache.LatestDataRepositoryAddress{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []gkill_cache.LatestDataRepositoryAddress, len(r))
	errch := make(chan error, len(r))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r {
		err := goForRep(ctx, wg, rep, func() {
			addrs, err := rep.GetLatestDataRepositoryAddress(ctx, updateCache)
			if err != nil {
				errch <- err
				return
			}
			ch <- addrs
		})
		if err != nil {
			errch <- err
		}
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = errors.Join(err, fmt.Errorf("error at get latest data repository address: %w", e))
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// 集約
loop:
	for {
		select {
		case addrs := <-ch:
			if addrs == nil {
				continue loop
			}
			allAddrs = append(allAddrs, addrs...)
		default:
			break loop
		}
	}
	return allAddrs, nil
}
