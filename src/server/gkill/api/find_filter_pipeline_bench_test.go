package api

// 性能の判断基準（ns/op を使わない）と、実測で否決した最適化の一覧:
// documents/adr/0008-perf-judge-by-allocs-not-ns-op.md

// FindKyous の後段パイプライン（SQLの外側）の割り当てを固定するベンチマーク。
//
// 実測(2026-08-16, pprof)では検索CPUの約40%がGCで、Goヒープ増分よりも
// 「1回の検索で作って捨てるオブジェクトの量」が支配的だった。
// ここが動かない最適化は入れない、という判断のための土台。
//
// 入力は実データの形に合わせて **IDごとに1エントリ** にしてある。
// usecase.GetKyous が OnlyLatestData=true を固定し(usecase/kyou.go)、
// --cache_in_memory(既定true)では型ごとにrepが1つへ畳まれるため、
// 実データでも MatchKyousCurrent のバケツはほぼ常に1件になる。
//
// **見るべきは allocs/op と B/op**。ns/op は計測環境によっては同一コードで倍近くぶれるので、
// 速くなった/遅くなったの判断には使わないこと。
//
// 実測(20万ID, go test -run '^$' -bench . -benchmem -benchtime 5x ./gkill/api/, 2026-08-18)
//
//	ベンチ                                 変更前 B/op    変更前 allocs  変更後 B/op    変更後 allocs
//	-------------------------------------  ------------  -------------  ------------  -------------
//	FindKyousPostPipeline                   329,297,096        503,202   156,152,249          1,062
//	SortAndTrimKyousMap_SingleEntryBuckets  121,180,904        401,045    12,593,456            514
//	ReplaceLatestKyouInfos                   73,180,968        201,045    12,593,510            515
//	SortResultKyous                                   0              0             0              0
//
// 変更の内訳:
//   - sortAndTrimKyousMap に len==1 の高速路(IDごとの一時マップと slices.Collect を丸ごと省く)
//   - replaceLatestKyouInfos で1要素スライスを作り直さない
//   - filterTagsKyous のOR分岐を「3つのマップを作る」から「1周してdelete」へ
//   - 結果マップと ResultKyous の事前確保
//
// ReplaceLatestKyouInfosParallel は割り当てではなく**競合**を見るためのもの。
// 20x で 3,441,280 ns/op -> 1,839,830 ns/op (プロセス共有mutexを最内ループから外した効果)。

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// benchKyouCount はベンチの入力件数。
// 実データは56万件だが、CIで回せる範囲として20万件にしてある。
// 割り当ては件数に線形なので、比較目的にはこれで足りる。
const benchKyouCount = 200_000

// buildBenchKyous は MatchKyousCurrent 相当のマップを作る。
func buildBenchKyous(count int) map[string][]reps.Kyou {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	kyous := make(map[string][]reps.Kyou, count)
	for i := range count {
		id := fmt.Sprintf("kyou-%08d", i)
		relatedTime := base.Add(time.Duration(i) * time.Minute)
		kyous[id] = []reps.Kyou{{
			ID:          id,
			DataType:    "kmemo",
			RepName:     "rep-1",
			RelatedTime: relatedTime,
			CreateTime:  relatedTime,
			UpdateTime:  relatedTime,
		}}
	}
	return kyous
}

// buildBenchMultiEntryKyous は1IDに複数版・複数射影がぶら下がる低速路用の入力。
// 高速路を入れたあとも、こちらが劣化していないことを見るために要る。
func buildBenchMultiEntryKyous(count int) map[string][]reps.Kyou {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	kyous := make(map[string][]reps.Kyou, count)
	for i := range count {
		id := fmt.Sprintf("kyou-%08d", i)
		relatedTime := base.Add(time.Duration(i) * time.Minute)
		kyous[id] = []reps.Kyou{
			{ID: id, DataType: "timeis_start", RepName: "rep-1", RelatedTime: relatedTime, UpdateTime: relatedTime},
			{ID: id, DataType: "timeis_end", RepName: "rep-1", RelatedTime: relatedTime, UpdateTime: relatedTime},
			{ID: id, DataType: "timeis_start", RepName: "rep-2", RelatedTime: relatedTime, UpdateTime: relatedTime.Add(-time.Hour)},
		}
	}
	return kyous
}

// buildBenchTagContext は既定のrykvクエリ相当（タグフィルタあり）の文脈を作る。
// MatchTags は半分のKyouに付き、RelatedTagIDs も同じ半分。
func buildBenchTagContext(kyous map[string][]reps.Kyou) (*find.FindQuery, map[string]reps.Tag, map[string]struct{}) {
	query := &find.FindQuery{
		Tags:    []string{"tag-a"},
		TagsAnd: false,
	}
	matchTags := make(map[string]reps.Tag, len(kyous)/2)
	relatedTagIDs := make(map[string]struct{}, len(kyous)/2)
	i := 0
	for id := range kyous {
		if i%2 == 0 {
			tagID := "tag-" + id
			matchTags[tagID] = reps.Tag{ID: tagID, TargetID: id, Tag: "tag-a"}
			relatedTagIDs[id] = struct{}{}
		}
		i++
	}
	return query, matchTags, relatedTagIDs
}

// BenchmarkFindKyousPostPipeline は SQL の外側を通しで測る。
// sortAndTrimKyousMap -> filterTagsKyous -> replaceLatestKyouInfos -> flatten -> sortResultKyous。
func BenchmarkFindKyousPostPipeline(b *testing.B) {
	kyous := buildBenchKyous(benchKyouCount)
	query, matchTags, relatedTagIDs := buildBenchTagContext(kyous)
	repositories := &reps.GkillRepositories{}

	filter := &FindFilter{}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		findCtx := &FindKyouContext{
			ParsedFindQuery:                  query,
			Repositories:                     repositories,
			DisableLatestDataRepositoryCache: true,
			MatchKyousCurrent:                kyous,
			MatchTags:                        matchTags,
			RelatedTagIDs:                    relatedTagIDs,
			MatchHideTagsWhenUncheckedKyou:   map[string]reps.Tag{},
		}
		if _, err := filter.sortAndTrimKyousMap(ctx, findCtx); err != nil {
			b.Fatal(err)
		}
		if _, err := filter.filterTagsKyous(ctx, findCtx); err != nil {
			b.Fatal(err)
		}
		if _, err := filter.replaceLatestKyouInfos(ctx, findCtx); err != nil {
			b.Fatal(err)
		}
		for _, kyousInID := range findCtx.MatchKyousCurrent {
			findCtx.ResultKyous = append(findCtx.ResultKyous, kyousInID...)
		}
		if _, err := filter.sortResultKyous(ctx, findCtx); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSortAndTrimKyousMap_SingleEntryBuckets は実データの支配的な形。
// IDごとの一時マップ(1件あたり約2.4KB)が消えるかどうかがここに出る。
func BenchmarkSortAndTrimKyousMap_SingleEntryBuckets(b *testing.B) {
	kyous := buildBenchKyous(benchKyouCount)
	query := &find.FindQuery{}
	filter := &FindFilter{}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		findCtx := &FindKyouContext{ParsedFindQuery: query, MatchKyousCurrent: kyous}
		if _, err := filter.sortAndTrimKyousMap(ctx, findCtx); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSortAndTrimKyousMap_MultiEntryBuckets は低速路（重複排除と整列が本当に要る形）。
func BenchmarkSortAndTrimKyousMap_MultiEntryBuckets(b *testing.B) {
	kyous := buildBenchMultiEntryKyous(benchKyouCount / 4)
	query := &find.FindQuery{}
	filter := &FindFilter{}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		findCtx := &FindKyouContext{ParsedFindQuery: query, MatchKyousCurrent: kyous}
		if _, err := filter.sortAndTrimKyousMap(ctx, findCtx); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReplaceLatestKyouInfos はIDごとの1要素スライス確保を測る。
func BenchmarkReplaceLatestKyouInfos(b *testing.B) {
	kyous := buildBenchKyous(benchKyouCount)
	query := &find.FindQuery{}
	repositories := &reps.GkillRepositories{}
	filter := &FindFilter{}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		findCtx := &FindKyouContext{
			ParsedFindQuery:                  query,
			Repositories:                     repositories,
			DisableLatestDataRepositoryCache: true,
			MatchKyousCurrent:                kyous,
		}
		if _, err := filter.replaceLatestKyouInfos(ctx, findCtx); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReplaceLatestKyouInfosParallel は同時検索を模す。
//
// GetLatestDataRepositoryAddress は結果Kyou 1件ごとに呼ばれ、その中の syncState() が
// **プロセス共有の** sync.Mutex を取る。直列のベンチではほぼ見えないが、
// 並列にすると全リクエストが最内ループの頻度で互いを待つのが出る。
func BenchmarkReplaceLatestKyouInfosParallel(b *testing.B) {
	kyous := buildBenchKyous(benchKyouCount / 10)
	query := &find.FindQuery{}
	repositories := &reps.GkillRepositories{}
	filter := &FindFilter{}
	ctx := context.Background()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			findCtx := &FindKyouContext{
				ParsedFindQuery:                  query,
				Repositories:                     repositories,
				DisableLatestDataRepositoryCache: true,
				MatchKyousCurrent:                kyous,
			}
			if _, err := filter.replaceLatestKyouInfos(ctx, findCtx); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSortResultKyous は 232バイト構造体の全ソートを測る。
func BenchmarkSortResultKyous(b *testing.B) {
	kyous := buildBenchKyous(benchKyouCount)
	source := make([]reps.Kyou, 0, benchKyouCount)
	for _, kyousInID := range kyous {
		source = append(source, kyousInID...)
	}
	work := make([]reps.Kyou, len(source))

	query := &find.FindQuery{}
	filter := &FindFilter{}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		copy(work, source)
		findCtx := &FindKyouContext{ParsedFindQuery: query, ResultKyous: work}
		if _, err := filter.sortResultKyous(ctx, findCtx); err != nil {
			b.Fatal(err)
		}
	}
}
