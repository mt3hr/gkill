package api

// TimeIs系フィルタ（filterTagsTimeIs / findTimeIs）の回帰テスト。
//
// 修正対象のバグ:
//   - TimeIsの非表示タグがどの組み合わせでも一度も適用されていなかった
//     (計算条件が適用側と逆 + タグ絞りなし分岐にdelete処理が無かった)
//   - use_timeis_tags=true + timeis_tags=nil でどの分岐にも入らず検索全体が0件になっていた
//   - AND分岐で存在しないタグ名が黙って無視され、タグ名照合が大小を区別していた
//   - findTimeIs のクエリに OnlyLatestData が無く、編集前タイトルの旧版がヒットしていた

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

func timeIsMapOf(ids ...string) map[string]reps.TimeIs {
	m := map[string]reps.TimeIs{}
	for _, id := range ids {
		m[id] = reps.TimeIs{ID: id}
	}
	return m
}

// タグ絞りなし(TimeIsTags=nil)でも非表示タグが適用されること
func TestFilterTagsTimeIs_NoTagFilter_AppliesHideTags(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TimeIsWords: []string{},
		},
		MatchTimeIssAtFindTimeIs: timeIsMapOf("timeis-visible", "timeis-hidden"),
		MatchTimeIssAtFilterTags: map[string]reps.TimeIs{},
		MatchHideTagsWhenUncheckedTimeIs: tagMapOf(
			tagFor("timeis-hidden", "非表示タグ"),
		),
	}

	f := &FindFilter{}
	if _, err := f.filterTagsTimeIs(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsTimeIs failed: %v", err)
	}

	if _, exist := findCtx.MatchTimeIssAtFilterTags["timeis-hidden"]; exist {
		t.Errorf("非表示タグの対象 timeis-hidden は消えるはず")
	}
	if _, exist := findCtx.MatchTimeIssAtFilterTags["timeis-visible"]; !exist {
		t.Errorf("timeis-visible は残るはず")
	}
}

// timeis_tags=nil(未指定)は「タグ絞りなし」として全件通ること
// (旧形式の use_timeis_tags=true + timeis_tags=nil では、以前どの分岐にも入らず0件になっていた)
func TestFilterTagsTimeIs_NilTimeIsTags_PassesAll(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TimeIsWords: []string{},
			TimeIsTags:  nil,
		},
		MatchTimeIssAtFindTimeIs:         timeIsMapOf("timeis-1", "timeis-2"),
		MatchTimeIssAtFilterTags:         map[string]reps.TimeIs{},
		MatchHideTagsWhenUncheckedTimeIs: map[string]reps.Tag{},
	}

	f := &FindFilter{}
	if _, err := f.filterTagsTimeIs(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsTimeIs failed: %v", err)
	}

	if len(findCtx.MatchTimeIssAtFilterTags) != 2 {
		t.Errorf("timeis_tags未指定(nil)は全件通るはず: got %d件", len(findCtx.MatchTimeIssAtFilterTags))
	}
}

// タグのチェックが0個(timeis_tags=[])なら0件になること(Kyou側と同じ)
func TestFilterTagsTimeIs_EmptyTimeIsTags_YieldsEmpty(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TimeIsWords: []string{},
			TimeIsTags:  []string{},
		},
		MatchTimeIssAtFindTimeIs: timeIsMapOf("timeis-1"),
		MatchTimeIssAtFilterTags: map[string]reps.TimeIs{},
	}

	f := &FindFilter{}
	if _, err := f.filterTagsTimeIs(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsTimeIs failed: %v", err)
	}

	if len(findCtx.MatchTimeIssAtFilterTags) != 0 {
		t.Errorf("timeis_tags空なら0件のはず: got %d件", len(findCtx.MatchTimeIssAtFilterTags))
	}
}

// OR分岐: クエリのタグにマッチしたTimeIsが残り、非表示タグの対象が消えること
func TestFilterTagsTimeIs_Or_TagMatchAndHide(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TimeIsWords:   []string{},
			TimeIsTagsAnd: false,
			TimeIsTags:    []string{"tagA"},
		},
		MatchTimeIssAtFindTimeIs: timeIsMapOf("timeis-a", "timeis-b", "timeis-other"),
		MatchTimeIssAtFilterTags: map[string]reps.TimeIs{},
		MatchTimeIsTags: tagMapOf(
			tagFor("timeis-a", "tagA"),
			tagFor("timeis-b", "tagA"),
		),
		RelatedTagIDs: map[string]struct{}{
			"timeis-a":     {},
			"timeis-b":     {},
			"timeis-other": {},
		},
		MatchHideTagsWhenUncheckedTimeIs: tagMapOf(
			tagFor("timeis-b", "非表示タグ"),
		),
	}

	f := &FindFilter{}
	if _, err := f.filterTagsTimeIs(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsTimeIs failed: %v", err)
	}

	if _, exist := findCtx.MatchTimeIssAtFilterTags["timeis-a"]; !exist {
		t.Errorf("tagA付きの timeis-a は残るはず")
	}
	if _, exist := findCtx.MatchTimeIssAtFilterTags["timeis-b"]; exist {
		t.Errorf("非表示タグの対象 timeis-b は消えるはず")
	}
	if _, exist := findCtx.MatchTimeIssAtFilterTags["timeis-other"]; exist {
		t.Errorf("クエリに無いタグの timeis-other は残らないはず")
	}
}

// AND分岐: クエリ中のタグ名が1件もヒットしなければ結果は空になること
func TestFilterTagsTimeIs_And_MissingTagNameYieldsEmpty(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TimeIsWords:   []string{},
			TimeIsTagsAnd: true,
			TimeIsTags:    []string{"tagA", "存在しないタグ"},
		},
		MatchTimeIssAtFindTimeIs: timeIsMapOf("timeis-a"),
		MatchTimeIssAtFilterTags: map[string]reps.TimeIs{},
		MatchTimeIsTags: tagMapOf(
			tagFor("timeis-a", "tagA"),
		),
		RelatedTagIDs: map[string]struct{}{
			"timeis-a": {},
		},
	}

	f := &FindFilter{}
	if _, err := f.filterTagsTimeIs(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsTimeIs failed: %v", err)
	}

	if len(findCtx.MatchTimeIssAtFilterTags) != 0 {
		t.Errorf("存在しないタグ名とのANDは空になるはず: got %d件", len(findCtx.MatchTimeIssAtFilterTags))
	}
}

// AND分岐: タグ名照合が大文字小文字を無視すること
func TestFilterTagsTimeIs_And_TagNameCaseInsensitive(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TimeIsWords:   []string{},
			TimeIsTagsAnd: true,
			TimeIsTags:    []string{"work"},
		},
		MatchTimeIssAtFindTimeIs: timeIsMapOf("timeis-a"),
		MatchTimeIssAtFilterTags: map[string]reps.TimeIs{},
		MatchTimeIsTags: tagMapOf(
			tagFor("timeis-a", "Work"),
		),
		RelatedTagIDs: map[string]struct{}{
			"timeis-a": {},
		},
	}

	f := &FindFilter{}
	if _, err := f.filterTagsTimeIs(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsTimeIs failed: %v", err)
	}

	if _, exist := findCtx.MatchTimeIssAtFilterTags["timeis-a"]; !exist {
		t.Errorf("タグ名照合は大文字小文字を無視するはず")
	}
}

// findTimeIs は編集前タイトルの旧版でヒットしないこと(OnlyLatestData必須)。
// findTags/findTextsGenericと同型のバグの回帰テスト。
func TestFindTimeIs_ExcludeRenamedAwayVersion(t *testing.T) {
	ctx := context.Background()

	timeIsRep, err := reps.NewTimeIsRepositorySQLite3Impl(ctx, filepath.Join(t.TempDir(), "timeis.db"), true)
	if err != nil {
		t.Fatalf("failed to create timeis repository: %v", err)
	}
	t.Cleanup(func() { timeIsRep.Close(ctx) })

	baseTime := time.Date(2026, 6, 27, 19, 2, 0, 0, time.Local)
	latestTime := baseTime.Add(1 * time.Minute)
	newTimeIs := func(title string, updateTime time.Time) reps.TimeIs {
		return reps.TimeIs{
			ID: "timeis-renamed", Title: title, StartTime: baseTime,
			CreateTime: baseTime, UpdateTime: updateTime,
			CreateApp: "test", CreateDevice: "test", CreateUser: "test",
			UpdateApp: "test", UpdateDevice: "test", UpdateUser: "test",
		}
	}
	for _, timeis := range []reps.TimeIs{newTimeIs("会議", baseTime), newTimeIs("打合せ", latestTime)} {
		if err := timeIsRep.AddTimeIsInfo(ctx, timeis); err != nil {
			t.Fatalf("AddTimeIsInfo failed: %v", err)
		}
	}

	latestDataAddresses := map[string]gkill_cache.LatestDataRepositoryAddress{
		"timeis-renamed": {TargetID: "timeis-renamed", DataUpdateTime: latestTime},
	}

	for _, disableCache := range []bool{true, false} {
		for _, c := range []struct {
			word      string
			wantCount int
		}{
			{"会議", 0},  // 編集前のタイトルではヒットしない
			{"打合せ", 1}, // 最新のタイトルではヒットする
		} {
			repositories := &reps.GkillRepositories{
				TimeIsReps: reps.TimeIsRepositories{timeIsRep},
			}
			repositories.SetLatestDataRepositoryAddresses(latestDataAddresses)
			findCtx := &FindKyouContext{
				DisableLatestDataRepositoryCache: disableCache,
				ParsedFindQuery: &find.FindQuery{
					TimeIsWords: []string{c.word},
				},
				Repositories:             repositories,
				MatchTimeIsTexts:         map[string]reps.Text{},
				MatchTimeIssAtFindTimeIs: map[string]reps.TimeIs{},
			}

			f := &FindFilter{}
			if _, err := f.findTimeIs(ctx, findCtx); err != nil {
				t.Fatalf("findTimeIs failed: %v", err)
			}
			if len(findCtx.MatchTimeIssAtFindTimeIs) != c.wantCount {
				t.Errorf("disableCache=%v word=%q: MatchTimeIss = %d, want %d", disableCache, c.word, len(findCtx.MatchTimeIssAtFindTimeIs), c.wantCount)
			}
		}
	}
}
