package api

// TimeIs系フィルタ（filterTagsTimeIs / findTimeIs）の回帰テスト。
//
// 修正対象のバグ:
//   - TimeIsの非表示タグが TimeIsTags!=nil のとき適用されていなかった(計算条件が適用側と逆)。
//     タグ絞りなし(TimeIsTags==nil)では Kyou 側と対称に適用しない(旧コードの nil 分岐 delete は
//     集合が常に空で発火しない死にコードだった＝監査 M-5(d))
//   - use_timeis_tags=true + timeis_tags=nil でどの分岐にも入らず検索全体が0件になっていた
//   - AND分岐で存在しないタグ名が黙って無視され、タグ名照合が大小を区別していた
//   - findTimeIs のクエリに OnlyLatestData が無く、編集前タイトルの旧版がヒットしていた
//   - findTimeIs の本文ヒットID検索が leaf rep 直叩きで findChunkedByIDs を回避していた(M-5(a))

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

// タグ絞りなし(TimeIsTags=nil)では強制非表示タグを適用しないこと（Kyou 側と対称）。
// getMatchHideTagsWhenUnckedTimeIs は TimeIsTags!=nil のゲートでしか集合を埋めないので、
// nil のときは集合が常に空＝適用しないのが本番の実挙動。Kyou 側 getMatchHideTagsWhenUnckedKyou も
// Tags==nil で早期returnして適用しない。ここでは集合を仮に埋めても nil 分岐が触れないことを固定する。
func TestFilterTagsTimeIs_NoTagFilter_DoesNotApplyHideTags(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TimeIsWords: []string{},
			TimeIsTags:  nil,
		},
		MatchTimeIssAtFindTimeIs: timeIsMapOf("timeis-visible", "timeis-hidden"),
		MatchTimeIssAtFilterTags: map[string]reps.TimeIs{},
		// 本番では nil 分岐でこの集合は空だが、仮に埋まっていても適用されないことを示す
		MatchHideTagsWhenUncheckedTimeIs: tagMapOf(
			tagFor("timeis-hidden", "非表示タグ"),
		),
	}

	f := &FindFilter{}
	if _, err := f.filterTagsTimeIs(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsTimeIs failed: %v", err)
	}

	// タグ絞りなしなので、削除済みでない TimeIs は非表示タグに関係なく全て通る
	if _, exist := findCtx.MatchTimeIssAtFilterTags["timeis-hidden"]; !exist {
		t.Errorf("タグ絞りなしでは強制非表示タグを適用しない: timeis-hidden も残るはず")
	}
	if _, exist := findCtx.MatchTimeIssAtFilterTags["timeis-visible"]; !exist {
		t.Errorf("timeis-visible は残るはず")
	}
}

// M-5(b): findTimeIsTags は名前の個数で SQL経路(≤32)と Go照合経路(>32)を切り替える。
// **どちらを通っても MatchTimeIsTags が同じ**でなければ、タグの個数で検索結果が変わる
// 静かな壊れ方になる。以前は名前ごとに GetTagsByTagName を投げて O(行数×名前数) だった。
func TestFindTimeIsTags_BothPathsAgree(t *testing.T) {
	ctx := context.Background()

	tagRep, err := reps.NewTagRepositorySQLite3Impl(ctx, filepath.Join(t.TempDir(), "tag.db"), true)
	if err != nil {
		t.Fatalf("failed to create tag repository: %v", err)
	}
	t.Cleanup(func() { tagRep.Close(ctx) })

	baseTime := time.Date(2026, 6, 27, 19, 2, 0, 0, time.Local)
	targetTag := reps.Tag{
		ID: "tag-hit", TargetID: "timeis-1", Tag: "会議",
		RelatedTime: baseTime, CreateTime: baseTime, UpdateTime: baseTime,
		CreateApp: "test", CreateDevice: "test", CreateUser: "test",
		UpdateApp: "test", UpdateDevice: "test", UpdateUser: "test",
	}
	if err := tagRep.AddTagInfo(ctx, targetTag); err != nil {
		t.Fatalf("AddTagInfo failed: %v", err)
	}

	latestDataAddresses := map[string]gkill_cache.LatestDataRepositoryAddress{
		"tag-hit": {TargetID: "timeis-1", DataUpdateTime: baseTime},
	}

	// 33名前(>32) は Go照合経路、["会議"] 単独は SQL経路。ダミー名はどのタグにも一致しない。
	manyNames := []string{"会議"}
	for i := 0; i < 33; i++ {
		manyNames = append(manyNames, "dummy-"+string(rune('a'+i%26))+string(rune('0'+i/26)))
	}

	for _, disableCache := range []bool{true, false} {
		run := func(names []string) map[string]reps.Tag {
			repositories := &reps.GkillRepositories{TagReps: reps.TagRepositories{tagRep}}
			repositories.SetLatestDataRepositoryAddresses(latestDataAddresses)
			findCtx := &FindKyouContext{
				DisableLatestDataRepositoryCache: disableCache,
				ParsedFindQuery:                  &find.FindQuery{TimeIsTags: names},
				Repositories:                     repositories,
				MatchTimeIsTags:                  map[string]reps.Tag{},
			}
			f := &FindFilter{}
			if _, err := f.findTimeIsTags(ctx, findCtx); err != nil {
				t.Fatalf("findTimeIsTags failed: %v", err)
			}
			return findCtx.MatchTimeIsTags
		}

		sqlPath := run([]string{"会議"})    // 1名前 → SQL経路
		goPath := run(manyNames)          // 33名前 → Go照合経路

		if len(sqlPath) != 1 || len(goPath) != 1 {
			t.Fatalf("disableCache=%v: SQL経路=%d件 Go経路=%d件, どちらも1件のはず", disableCache, len(sqlPath), len(goPath))
		}
		for id := range sqlPath {
			if _, exist := goPath[id]; !exist {
				t.Errorf("disableCache=%v: SQL経路にしか無いタグ %q(2経路がずれている)", disableCache, id)
			}
		}
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

func TestFilterPlaingTimeIsKyous_UsesInclusiveMergedIntervals(t *testing.T) {
	end11 := intervalTestTime(11)
	end12 := intervalTestTime(12)
	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{TimeIsWords: []string{}},
		MatchTimeIssAtFilterTags: map[string]reps.TimeIs{
			"first":  {ID: "first", StartTime: intervalTestTime(10), EndTime: &end11},
			"second": {ID: "second", StartTime: intervalTestTime(11), EndTime: &end12},
			"open":   {ID: "open", StartTime: intervalTestTime(15)},
		},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"start":    {{ID: "start", RelatedTime: intervalTestTime(10)}},
			"shared":   {{ID: "shared", RelatedTime: intervalTestTime(11)}},
			"end":      {{ID: "end", RelatedTime: intervalTestTime(12)}},
			"gap":      {{ID: "gap", RelatedTime: intervalTestTime(13)}},
			"open-end": {{ID: "open-end", RelatedTime: intervalTestTime(23)}},
		},
	}

	filter := &FindFilter{}
	if _, err := filter.filterPlaingTimeIsKyous(context.Background(), findCtx); err != nil {
		t.Fatalf("filterPlaingTimeIsKyous failed: %v", err)
	}
	for _, id := range []string{"start", "shared", "end", "open-end"} {
		if _, exist := findCtx.MatchKyousCurrent[id]; !exist {
			t.Errorf("%s should match", id)
		}
	}
	if _, exist := findCtx.MatchKyousCurrent["gap"]; exist {
		t.Error("gap should not match")
	}
}
