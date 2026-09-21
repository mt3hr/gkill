package api

// sortAndTrimKyousMap の回帰テスト。
//
// 修正対象のバグ: 重複排除キーが RelatedTime.Unix() だったため、
// 同一IDの複数版（rep横断で回収されたもの）が同秒に衝突してスライス順で
// 最後の1件だけが残っていた。スライス順はチャネル回収順で非決定的なので、
// 旧版が残ると後段の replaceLatestKyouInfos がレコードごと除外し、
// 「検索のたびに出たり消えたりする」症状になっていた。
//
// 修正後のキーは (UpdateTime, DataType, RelatedTime) の複合キー。
// 「同一版の同一射影の rep 間重複」だけが潰れ、版違い・射影違いは残る。

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// 同一ID・同一RelatedTimeの新旧2版が両方残り、先頭が新版であること
func TestSortAndTrimKyousMap_KeepsNewestVersionAcrossReps(t *testing.T) {
	ctx := context.Background()

	relatedTime := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	oldVersion := reps.Kyou{
		ID:          "kyou-1",
		DataType:    "kmemo",
		RelatedTime: relatedTime,
		UpdateTime:  time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		RepName:     "rep-old",
	}
	newVersion := reps.Kyou{
		ID:          "kyou-1",
		DataType:    "kmemo",
		RelatedTime: relatedTime,
		UpdateTime:  time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC),
		RepName:     "rep-new",
	}

	// 新版を先に置く: 旧実装(RelatedTime.Unix()キー)だと後勝ちで旧版だけが残り、
	// このテストが決定的に落ちる
	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"kyou-1": {newVersion, oldVersion},
		},
	}

	f := &FindFilter{}
	if _, err := f.sortAndTrimKyousMap(ctx, findCtx); err != nil {
		t.Fatalf("sortAndTrimKyousMap failed: %v", err)
	}

	kyous := findCtx.MatchKyousCurrent["kyou-1"]
	if len(kyous) != 2 {
		t.Fatalf("新旧2版とも残るはず: got %d件", len(kyous))
	}
	if !kyous[0].UpdateTime.Equal(newVersion.UpdateTime) {
		t.Errorf("先頭は新版のはず: got UpdateTime=%v", kyous[0].UpdateTime)
	}
}

// 同一版の同一射影がrep間重複した場合は1件に潰れること
func TestSortAndTrimKyousMap_DedupsSameVersionAcrossReps(t *testing.T) {
	ctx := context.Background()

	kyou := reps.Kyou{
		ID:          "kyou-1",
		DataType:    "kmemo",
		RelatedTime: time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		UpdateTime:  time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
	}
	fromRepA := kyou
	fromRepA.RepName = "rep-a"
	fromRepB := kyou
	fromRepB.RepName = "rep-b"

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"kyou-1": {fromRepA, fromRepB},
		},
	}

	f := &FindFilter{}
	if _, err := f.sortAndTrimKyousMap(ctx, findCtx); err != nil {
		t.Fatalf("sortAndTrimKyousMap failed: %v", err)
	}

	if len(findCtx.MatchKyousCurrent["kyou-1"]) != 1 {
		t.Errorf("同一版のrep間重複は1件に潰れるはず: got %d件", len(findCtx.MatchKyousCurrent["kyou-1"]))
	}
}

// TimeIsのstart/end射影が同秒でも両方残ること
func TestSortAndTrimKyousMap_KeepsSameSecondProjections(t *testing.T) {
	ctx := context.Background()

	sameSecond := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	updateTime := time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC)
	startProjection := reps.Kyou{
		ID:          "timeis-1",
		DataType:    "timeis_start",
		RelatedTime: sameSecond,
		UpdateTime:  updateTime,
	}
	endProjection := reps.Kyou{
		ID:          "timeis-1",
		DataType:    "timeis_end",
		RelatedTime: sameSecond,
		UpdateTime:  updateTime,
	}

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"timeis-1": {startProjection, endProjection},
		},
	}

	f := &FindFilter{}
	if _, err := f.sortAndTrimKyousMap(ctx, findCtx); err != nil {
		t.Fatalf("sortAndTrimKyousMap failed: %v", err)
	}

	if len(findCtx.MatchKyousCurrent["timeis-1"]) != 2 {
		t.Errorf("開始と終了の射影は同秒でも両方残るはず: got %d件", len(findCtx.MatchKyousCurrent["timeis-1"]))
	}
}

func BenchmarkSortAndTrimKyousMap_PeriodOfTime(b *testing.B) {
	const kyouCount = 20_000
	startSecond := int64(9 * 60 * 60)
	endSecond := int64(18 * 60 * 60)
	query := &find.FindQuery{
		PeriodOfTimeStartTimeSecond: &startSecond,
		PeriodOfTimeEndTimeSecond:   &endSecond,
		PeriodOfTimeWeekOfDays:      []find.WeekOfDays{find.MonDay, find.TuesDay, find.WednesDay, find.ThursDay, find.FriDay},
	}
	base := time.Date(2026, 8, 3, 0, 0, 0, 0, time.Local)
	kyous := make(map[string][]reps.Kyou, kyouCount)
	for i := range kyouCount {
		id := string(rune(i + 1))
		relatedTime := base.Add(time.Duration(i) * time.Minute)
		kyous[id] = []reps.Kyou{{
			ID:          id,
			DataType:    "kmemo",
			RelatedTime: relatedTime,
			UpdateTime:  relatedTime,
		}}
	}

	filter := &FindFilter{}
	b.ReportAllocs()
	for b.Loop() {
		findCtx := &FindKyouContext{ParsedFindQuery: query, MatchKyousCurrent: kyous}
		if _, err := filter.sortAndTrimKyousMap(context.Background(), findCtx); err != nil {
			b.Fatal(err)
		}
	}
}

// 最終段の重複排除は (ID, DataType, RelatedTimeナノ秒) の複合キーで行う。
//
// **素のID重複排除にしてはいけない** ―― TimeIs は同じIDから timeis_start /
// timeis_end の2行が正当に出る（DataTypeが違う）。複合キーならこの2行は残り、
// 完全重複（同一コミットが2経路から合流した等）だけが畳まれる。
// 経路の合流で残る完全重複の最終防衛線（指摘 C2 の一部）。
func TestSortResultKyous_DedupKeepsTimeIsProjections(t *testing.T) {
	ctx := context.Background()
	commitTime := time.Date(2026, 8, 1, 12, 0, 0, 0, time.Local)
	startTime := time.Date(2026, 8, 1, 10, 0, 0, 0, time.Local)
	endTime := time.Date(2026, 8, 1, 11, 0, 0, 0, time.Local)

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{},
		ResultKyous: []reps.Kyou{
			// 完全重複（同一git commitが2つのrep経由で合流したケース）→ 1本になる
			{ID: "commit-1", DataType: "git_commit_log", RelatedTime: commitTime, UpdateTime: commitTime},
			{ID: "commit-1", DataType: "git_commit_log", RelatedTime: commitTime, UpdateTime: commitTime},
			// TimeIs の start/end 2射影（同一ID・別DataType）→ 両方残る
			{ID: "timeis-1", DataType: "timeis_start", RelatedTime: startTime, UpdateTime: startTime},
			{ID: "timeis-1", DataType: "timeis_end", RelatedTime: endTime, UpdateTime: startTime},
		},
	}

	f := &FindFilter{}
	if _, err := f.sortResultKyous(ctx, findCtx); err != nil {
		t.Fatalf("sortResultKyous failed: %v", err)
	}

	counts := map[string]int{}
	for _, kyou := range findCtx.ResultKyous {
		counts[kyou.ID+"/"+kyou.DataType]++
	}
	if counts["commit-1/git_commit_log"] != 1 {
		t.Errorf("完全重複のcommit-1は1本に畳まれるはず: got %d", counts["commit-1/git_commit_log"])
	}
	if counts["timeis-1/timeis_start"] != 1 || counts["timeis-1/timeis_end"] != 1 {
		t.Errorf("TimeIsのstart/end 2射影は両方残るはず: got %v", counts)
	}
	if len(findCtx.ResultKyous) != 3 {
		t.Errorf("結果は3本のはず: got %d", len(findCtx.ResultKyous))
	}
}

// 同一秒・同一ID・同一DataTypeでもナノ秒が違えば別entryとして残る
// (dedupの同一性はナノ秒精度。秒に丸めると別時刻の行を巻き込む)。
func TestSortResultKyous_DedupKeepsSubsecondDistinct(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.Local)

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{},
		ResultKyous: []reps.Kyou{
			{ID: "kyou-1", DataType: "kmemo", RelatedTime: base, UpdateTime: base},
			{ID: "kyou-1", DataType: "kmemo", RelatedTime: base.Add(500 * time.Millisecond), UpdateTime: base},
		},
	}

	f := &FindFilter{}
	if _, err := f.sortResultKyous(ctx, findCtx); err != nil {
		t.Fatalf("sortResultKyous failed: %v", err)
	}
	if len(findCtx.ResultKyous) != 2 {
		t.Errorf("ナノ秒違いの2本は両方残るはず: got %d", len(findCtx.ResultKyous))
	}
}
