package reps

// キャッシュrepの行スキャン(検索の実処理)の割り当てを固定するベンチマーク。
//
// --cache_in_memory(既定true)では型ごとに cached rep が1つへ畳まれるので、
// 実データの検索が触るのはこの経路。find_filter の後段(api/find_filter_pipeline_bench_test.go)を
// 削ったあと、1行あたりのコストはここに残る。
//
// 実測(2万行, go test -run '^$' -bench 'BenchmarkCached' -benchmem ./gkill/dao/reps/, 2026-08-18)
//
//	ベンチ                           B/op        allocs/op   備考
//	-------------------------------  ----------  ----------  --------------------------------
//	CachedKmemoFindKyous             18,830,218     500,209  1行あたり25確保。ほぼ全部が行スキャン
//	CachedKmemoFindKyousCalendar        982,437      25,087  返る1000行ぶん。索引は効いている
//	CachedTimeIsFindKyous             2,046,781      50,144  UNION ALL 化で一時Btreeが消えた
//	CachedTimeIsFindKyousPlaing          15,906         152  返るのは数行
//
// **ns/op はこのマシンでは同一コードで10〜17msまでぶれる**ので判断に使わないこと。
// 索引を足す/外すの判断は dao/sqlite3impl/timeis_range_index_test.go のクエリプランで行う。
//
// 1行25確保の内訳は database/sql の列変換(13列)とスキャン先の文字列。
// ここを削るには射影から定数列(? AS DATA_TYPE)を落とすか、
// 低カーディナリティ列をインターンするしかない(どちらも未着手)。

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	_ "modernc.org/sqlite"
)

// benchRowCount はベンチの行数。実データは56万件だがCIで回せる範囲にしてある。
// 1行あたりの割り当ては件数に線形なので、比較目的にはこれで足りる。
const benchRowCount = 20_000

func openBenchMemoryDB(tb testing.TB) *sql.DB {
	tb.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		tb.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	tb.Cleanup(func() { db.Close() })
	return db
}

// analyzeBenchDB は本番の UpdateCache 末尾と同じ統計収集を流す。
// これをやらないとSQLiteに統計が無く、選択率の低い述語にまで索引を選んでしまうため、
// 索引を足したベンチが本番と逆の結論を出す。
func analyzeBenchDB(tb testing.TB, db *sql.DB) {
	tb.Helper()
	for _, stmt := range []string{"PRAGMA analysis_limit=400;", "ANALYZE;", "PRAGMA optimize;"} {
		if _, err := db.ExecContext(context.Background(), stmt); err != nil {
			tb.Fatalf("failed to run %q: %v", stmt, err)
		}
	}
}

func newBenchCachedKmemoRepo(tb testing.TB, count int) KmemoRepository {
	tb.Helper()
	ctx := context.Background()
	dir := tb.TempDir()
	// 下層repは空のまま。キャッシュ表へ直接書き込むので、
	// ファイルDBへの1行ずつのINSERT(fsync付き)をベンチのセットアップで払わずに済む。
	// 測りたいのはキャッシュ表に対する FindKyous なので、これで足りる。
	baseRepo, err := NewKmemoRepositorySQLite3Impl(ctx, filepath.Join(dir, "kmemo.db"), true)
	if err != nil {
		tb.Fatalf("failed to create kmemo repo: %v", err)
	}
	tb.Cleanup(func() { baseRepo.Close(ctx) })

	cacheDB := openBenchMemoryDB(tb)
	repo, err := NewKmemoRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, &sync.RWMutex{}, "KMEMO_CACHE")
	if err != nil {
		tb.Fatalf("failed to create cached kmemo repo: %v", err)
	}
	tb.Cleanup(func() { repo.Close(ctx) })

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	for i := range count {
		kmemo := makeKmemo(fmt.Sprintf("kmemo-%08d", i), fmt.Sprintf("content %d", i))
		kmemo.RepName = "bench_rep"
		kmemo.RelatedTime = base.Add(time.Duration(i) * time.Minute)
		kmemo.CreateTime = kmemo.RelatedTime
		kmemo.UpdateTime = kmemo.RelatedTime
		if err := repo.AddKmemoInfo(ctx, kmemo); err != nil {
			tb.Fatalf("failed to seed kmemo: %v", err)
		}
	}
	analyzeBenchDB(tb, cacheDB)
	return repo
}

func newBenchCachedTimeIsRepo(tb testing.TB, count int) TimeIsRepository {
	tb.Helper()
	ctx := context.Background()
	dir := tb.TempDir()
	baseRepo, err := NewTimeIsRepositorySQLite3Impl(ctx, filepath.Join(dir, "timeis.db"), true)
	if err != nil {
		tb.Fatalf("failed to create timeis repo: %v", err)
	}
	tb.Cleanup(func() { baseRepo.Close(ctx) })

	cacheDB := openBenchMemoryDB(tb)
	repo, err := NewTimeIsRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, &sync.RWMutex{}, "TIMEIS_CACHE")
	if err != nil {
		tb.Fatalf("failed to create cached timeis repo: %v", err)
	}
	tb.Cleanup(func() { repo.Close(ctx) })

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	for i := range count {
		timeIs := makeTimeIs(fmt.Sprintf("timeis-%08d", i), fmt.Sprintf("title %d", i))
		start := base.Add(time.Duration(i) * time.Minute)
		end := start.Add(30 * time.Second)
		timeIs.RepName = "bench_rep"
		timeIs.StartTime = start
		timeIs.EndTime = &end
		timeIs.CreateTime = start
		timeIs.UpdateTime = start
		if err := repo.AddTimeIsInfo(ctx, timeIs); err != nil {
			tb.Fatalf("failed to seed timeis: %v", err)
		}
	}
	analyzeBenchDB(tb, cacheDB)
	return repo
}

// BenchmarkCachedKmemoFindKyous は素の全件検索(既定のrykv相当の形)。
func BenchmarkCachedKmemoFindKyous(b *testing.B) {
	repo := newBenchCachedKmemoRepo(b, benchRowCount)
	query := &find.FindQuery{OnlyLatestData: true}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		kyous, err := repo.FindKyous(ctx, query)
		if err != nil {
			b.Fatal(err)
		}
		if len(kyous) != benchRowCount {
			b.Fatalf("unexpected count: %d", len(kyous))
		}
	}
}

// BenchmarkCachedKmemoFindKyousCalendar は期間で絞る形。索引が効くかがここに出る。
func BenchmarkCachedKmemoFindKyousCalendar(b *testing.B) {
	repo := newBenchCachedKmemoRepo(b, benchRowCount)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	start := base.Add(time.Duration(benchRowCount-1000) * time.Minute)
	end := base.Add(time.Duration(benchRowCount) * time.Minute)
	query := &find.FindQuery{OnlyLatestData: true, CalendarStartDate: &start, CalendarEndDate: &end}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := repo.FindKyous(ctx, query); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCachedTimeIsFindKyous は2射影のUNIONと、START/END_TIME_UNIX への絞り込み。
func BenchmarkCachedTimeIsFindKyous(b *testing.B) {
	repo := newBenchCachedTimeIsRepo(b, benchRowCount)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	start := base.Add(time.Duration(benchRowCount-1000) * time.Minute)
	end := base.Add(time.Duration(benchRowCount) * time.Minute)
	query := &find.FindQuery{OnlyLatestData: true, CalendarStartDate: &start, CalendarEndDate: &end, IncludeEndTimeIs: true}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := repo.FindKyous(ctx, query); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCachedTimeIsFindKyousPlaing はKyou 1件ごとの「実行中」判定と同じ形。
// 一覧の行数ぶん飛ぶ経路なので、1回のコストがそのまま体感になる。
func BenchmarkCachedTimeIsFindKyousPlaing(b *testing.B) {
	repo := newBenchCachedTimeIsRepo(b, benchRowCount)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	plaingTime := base.Add(time.Duration(benchRowCount/2) * time.Minute).Add(10 * time.Second)
	query := &find.FindQuery{OnlyLatestData: true, PlaingTime: &plaingTime, IncludeEndTimeIs: true}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := repo.FindKyous(ctx, query); err != nil {
			b.Fatal(err)
		}
	}
}
