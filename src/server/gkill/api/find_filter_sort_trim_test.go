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
