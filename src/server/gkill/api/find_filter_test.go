package api

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

func TestNoTagsConstant(t *testing.T) {
	if NoTags != "no tags" {
		t.Errorf("NoTags = %q, want %q", NoTags, "no tags")
	}
}

// replaceLatestKyouInfos は、マッチした版がグローバル最新(LatestDataRepositoryAddress)で
// なければレコードを除外し、最新版が含まれていればその版のentryのみ残す。
//
// DisableLatestDataRepositoryCache では分岐しない、が不変条件。
// 以前はキャッシュ有無で2ブランチに分かれており、Plaing判定の粒度・アドレス未登録時の扱い・
// 保持件数が食い違っていた。統合済みなので両方の値で回す意味は無くなり、1回だけ実行する
// (キャッシュ設定を読む分岐を復活させたらここは意味を持たなくなるので、そのときは再度2値で回すこと)。
func TestReplaceLatestKyouInfos_ExcludeStaleKeepLatest(t *testing.T) {
	const targetID = "timeis-1"
	const latestRepName = "TimeIs_TestPhoneA_20250919"
	staleUpdateTime := time.Date(2026, 6, 27, 19, 2, 0, 0, time.Local)   // 古い版(可視rep)
	latestUpdateTime := time.Date(2026, 6, 28, 22, 2, 31, 0, time.Local) // グローバル最新

	latestDataAddresses := map[string]gkill_cache.LatestDataRepositoryAddress{
		targetID: {TargetID: targetID, LatestDataRepositoryName: latestRepName, DataUpdateTime: latestUpdateTime},
	}

	timeIsKyou := func(dataType, repName string, updateTime time.Time) reps.Kyou {
		return reps.Kyou{ID: targetID, DataType: dataType, RepName: repName, UpdateTime: updateTime}
	}

	newContext := func(matchKyous map[string][]reps.Kyou) *FindKyouContext {
		repositories := &reps.GkillRepositories{}
		repositories.SetLatestDataRepositoryAddresses(latestDataAddresses)
		return &FindKyouContext{
			ParsedFindQuery:   &find.FindQuery{},
			Repositories:      repositories,
			MatchKyousCurrent: matchKyous,
		}
	}

	f := &FindFilter{}

	// 可視repのマッチが古い版だけ → グローバル最新は別repなので除外される
	staleOnly := newContext(map[string][]reps.Kyou{
		targetID: {
			timeIsKyou("timeis_start", "TimeIs", staleUpdateTime),
			timeIsKyou("timeis_end", "TimeIs", staleUpdateTime),
		},
	})
	if _, err := f.replaceLatestKyouInfos(context.Background(), staleOnly); err != nil {
		t.Fatalf("replaceLatestKyouInfos error: %v", err)
	}
	if _, ok := staleOnly.MatchKyousCurrent[targetID]; ok {
		t.Errorf("古い版だけのレコードは除外されるべき")
	}

	// 古い版と最新版が混在 → 最新版のstart/end 2件のみ残る
	staleAndLatest := newContext(map[string][]reps.Kyou{
		targetID: {
			timeIsKyou("timeis_start", "TimeIs", staleUpdateTime),
			timeIsKyou("timeis_start", latestRepName, latestUpdateTime),
			timeIsKyou("timeis_end", latestRepName, latestUpdateTime),
		},
	})
	if _, err := f.replaceLatestKyouInfos(context.Background(), staleAndLatest); err != nil {
		t.Fatalf("replaceLatestKyouInfos error: %v", err)
	}
	kept := staleAndLatest.MatchKyousCurrent[targetID]
	if len(kept) != 2 {
		t.Fatalf("最新版の2件が残るべき, got %d", len(kept))
	}
	for _, kyou := range kept {
		if !kyou.UpdateTime.Equal(latestUpdateTime) {
			t.Errorf("古い版が残っている: %#v", kyou)
		}
	}

	// 最新版アドレス表に載っていないID(plugin/git/gps由来) は hasLatestData=false で素通しする。
	// 以前はここで無条件に除外され、プラグイン由来のKyouが検索結果から全滅していた。
	// 素通しした上で、射影を持たない型なので最新版1件に絞られる。
	const pluginID = "plugin-kyou-1"
	pluginKyou := func(updateTime time.Time) reps.Kyou {
		return reps.Kyou{ID: pluginID, DataType: "claude_conversation", RepName: "gkill_plugin_claudeai", UpdateTime: updateTime}
	}
	notInAddressTable := newContext(map[string][]reps.Kyou{
		pluginID: {
			pluginKyou(staleUpdateTime),
			pluginKyou(latestUpdateTime),
		},
	})
	if _, err := f.replaceLatestKyouInfos(context.Background(), notInAddressTable); err != nil {
		t.Fatalf("replaceLatestKyouInfos error: %v", err)
	}
	keptPlugin, ok := notInAddressTable.MatchKyousCurrent[pluginID]
	if !ok {
		t.Fatalf("最新版アドレス表に載っていないIDは素通しされるべき: %v", notInAddressTable.MatchKyousCurrent)
	}
	if len(keptPlugin) != 1 {
		t.Fatalf("射影を持たない型は最新版1件に絞られるべき, got %d", len(keptPlugin))
	}
	if !keptPlugin[0].UpdateTime.Equal(latestUpdateTime) {
		t.Errorf("残ったのが最新版ではない: %#v", keptPlugin[0])
	}
}

// drainFindErrors は FindKyous が並行実行するタグ/テキスト取得のエラーを回収する。
// 以前は待ち合わせより前に吸い出していたため、起動直後の空チャネルを見て即座に抜け、
// 6経路のエラーが常に捨てられて検索が「成功」を返していた。
// goroutineの完了が遅れてもエラーを取りこぼさないことを固定する。
func TestDrainFindErrors_CollectsLateGoroutineErrors(t *testing.T) {
	wg := &sync.WaitGroup{}
	errch := make(chan error, 23)
	gkillErrch := make(chan []*message.GkillError, 6)

	const goroutineCount = 3
	for i := range goroutineCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 回収側が先に走る状況（＝不具合時に取りこぼしていた状況）を作る
			time.Sleep(30 * time.Millisecond)
			errch <- fmt.Errorf("取得失敗 %d", i)
			gkillErrch <- []*message.GkillError{{ErrorCode: fmt.Sprintf("ERRTEST%d", i)}}
		}()
	}

	gkillErrors, err := drainFindErrors(wg, errch, gkillErrch)

	if err == nil {
		t.Fatal("goroutineが積んだエラーが回収されていない")
	}
	for i := range goroutineCount {
		if want := fmt.Sprintf("取得失敗 %d", i); !strings.Contains(err.Error(), want) {
			t.Errorf("エラー %q が回収結果に含まれていない: %v", want, err)
		}
	}
	if len(gkillErrors) != goroutineCount {
		t.Errorf("GkillError = %d件, want %d件", len(gkillErrors), goroutineCount)
	}
}

// エラーが1件も無いときは nil を返し、検索が成功として扱われること。
func TestDrainFindErrors_NoErrorReturnsNil(t *testing.T) {
	wg := &sync.WaitGroup{}
	errch := make(chan error, 23)
	gkillErrch := make(chan []*message.GkillError, 6)

	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errch <- nil
			gkillErrch <- nil
		}()
	}

	gkillErrors, err := drainFindErrors(wg, errch, gkillErrch)
	if err != nil {
		t.Errorf("エラーが無いのに err が返っている: %v", err)
	}
	if len(gkillErrors) != 0 {
		t.Errorf("GkillError = %d件, want 0件", len(gkillErrors))
	}
}

// findTags はタグ名でタグを引く。TAGテーブルはappend-onlyなので、
// リネームされたタグの旧版が残っており、旧名で検索するとヒットしてしまっていた。
// isLatestData は --cache_in_memory=true (既定) だと常にtrueを返すno-opなので、
// DisableLatestDataRepositoryCache の両ブランチで旧名がヒットしないことを確認する。
func TestFindTags_ExcludeRenamedAwayVersion(t *testing.T) {
	ctx := context.Background()

	tagRep, err := reps.NewTagRepositorySQLite3Impl(ctx, filepath.Join(t.TempDir(), "tag.db"), true)
	if err != nil {
		t.Fatalf("failed to create tag repository: %v", err)
	}
	t.Cleanup(func() { tagRep.Close(ctx) })

	baseTime := time.Date(2026, 6, 27, 19, 2, 0, 0, time.Local)
	latestTime := baseTime.Add(1 * time.Minute)
	newTag := func(tagName string, updateTime time.Time) reps.Tag {
		return reps.Tag{
			ID: "tag-renamed", TargetID: "target-1", Tag: tagName,
			RelatedTime: baseTime, CreateTime: baseTime, UpdateTime: updateTime,
			CreateApp: "test", CreateDevice: "test", CreateUser: "test",
			UpdateApp: "test", UpdateDevice: "test", UpdateUser: "test",
		}
	}
	for _, tag := range []reps.Tag{newTag("お避け", baseTime), newTag("お酒", latestTime)} {
		if err := tagRep.AddTagInfo(ctx, tag); err != nil {
			t.Fatalf("AddTagInfo failed: %v", err)
		}
	}

	latestDataAddresses := map[string]gkill_cache.LatestDataRepositoryAddress{
		"tag-renamed": {TargetID: "tag-renamed", DataUpdateTime: latestTime},
	}

	for _, disableCache := range []bool{true, false} {
		for _, c := range []struct {
			tagName   string
			wantCount int
		}{
			{"お避け", 0}, // 編集前のタグ名ではヒットしない
			{"お酒", 1},  // 最新のタグ名ではヒットする
		} {
			repositories := &reps.GkillRepositories{
				TagReps: reps.TagRepositories{tagRep},
			}
			repositories.SetLatestDataRepositoryAddresses(latestDataAddresses)
			findCtx := &FindKyouContext{
				DisableLatestDataRepositoryCache: disableCache,
				ParsedFindQuery:                  &find.FindQuery{Tags: []string{c.tagName}},
				Repositories:                     repositories,
				MatchTags:                        map[string]reps.Tag{},
			}

			f := &FindFilter{}
			if _, err := f.collectTagsForFilter(ctx, findCtx, true, false, false); err != nil {
				t.Fatalf("collectTagsForFilter failed: %v", err)
			}
			if len(findCtx.MatchTags) != c.wantCount {
				t.Errorf("disableCache=%v tag=%q: MatchTags = %d, want %d", disableCache, c.tagName, len(findCtx.MatchTags), c.wantCount)
			}

			// collectTagsForFilter は「SQLで名前を絞る」と「全部取ってGoで照合する」の
			// 2経路を名前の個数で切り替える(maxTagNamesForSQLFilter)。
			// **どちらを通っても結果が同じ**でなければ、タグの個数によって
			// 検索結果が変わるという静かな壊れ方になる。
			// needRelatedTagIDs=true はGo側で照合する経路
			goPathCtx := &FindKyouContext{
				DisableLatestDataRepositoryCache: disableCache,
				ParsedFindQuery:                  &find.FindQuery{Tags: []string{c.tagName}},
				Repositories:                     repositories,
				MatchTags:                        map[string]reps.Tag{},
				RelatedTagIDs:                    map[string]struct{}{},
			}
			if _, err := f.collectTagsForFilter(ctx, goPathCtx, true, true, false); err != nil {
				t.Fatalf("collectTagsForFilter (Go側で照合する経路) failed: %v", err)
			}
			if len(goPathCtx.MatchTags) != c.wantCount {
				t.Errorf("disableCache=%v tag=%q: Go側で照合する経路の MatchTags = %d, want %d", disableCache, c.tagName, len(goPathCtx.MatchTags), c.wantCount)
			}
			for id := range findCtx.MatchTags {
				if _, exist := goPathCtx.MatchTags[id]; !exist {
					t.Errorf("disableCache=%v tag=%q: SQL経路にしか無いタグ %q", disableCache, c.tagName, id)
				}
			}
		}
	}
}

// sortAndTrimKyousMap の曜日フィルタ(PeriodOfTimeWeekOfDays)の意味論を固定する。
//
//	nil     = 曜日で絞らない（全件残る）
//	非nil空 = 0件指定（全部消える）
//	全7曜日 = 曜日で絞らない（全件残る）
//	部分指定 = 該当曜日のKyouだけ残る
//
// nil を len==0 や len!=7 の分岐へ落とすと「時間帯だけ指定した検索で全件消える」ので、
// nil の先行ガードが要る。
// 時間帯(start/end)は1日全体を覆う値にして曜日フィルタだけが効く状態にする。
// start/endのどちらかが非nilでないと HasPeriodOfTimeFilter() が偽になって
// 曜日フィルタのブロックへ入らず、nilケースの検査にならないため。
func TestSortAndTrimKyousMap_PeriodOfTimeWeekOfDays(t *testing.T) {
	ctx := context.Background()

	// ローカル 00:00:00 〜 23:59:59 = 1日全体
	dayStartSecond := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local).Unix()
	dayEndSecond := time.Date(2026, 1, 1, 23, 59, 59, 0, time.Local).Unix()

	// 連続する7日ぶんのKyou。曜日はRelatedTimeから採るのでカレンダーの書き間違いが起きない
	baseDay := time.Date(2026, 8, 2, 12, 0, 0, 0, time.Local)
	kyouIDOfIndex := func(i int) string { return fmt.Sprintf("kyou-%d", i) }
	weekOfDayOfKyou := map[string]find.WeekOfDays{}
	for i := range 7 {
		weekOfDayOfKyou[kyouIDOfIndex(i)] = find.WeekOfDays(int(baseDay.AddDate(0, 0, i).Weekday()))
	}
	newMatchKyous := func() map[string][]reps.Kyou {
		matchKyous := map[string][]reps.Kyou{}
		for i := range 7 {
			id := kyouIDOfIndex(i)
			relatedTime := baseDay.AddDate(0, 0, i)
			matchKyous[id] = []reps.Kyou{{ID: id, DataType: "kmemo", RelatedTime: relatedTime, UpdateTime: relatedTime}}
		}
		return matchKyous
	}

	allWeekOfDays := []find.WeekOfDays{find.SunDay, find.MonDay, find.TuesDay, find.WednesDay, find.ThursDay, find.FriDay, find.SaturDay}

	for _, c := range []struct {
		name       string
		weekOfDays []find.WeekOfDays
		wantDays   []find.WeekOfDays
	}{
		{name: "nilは曜日で絞らない", weekOfDays: nil, wantDays: allWeekOfDays},
		{name: "非nil空は0件指定", weekOfDays: []find.WeekOfDays{}, wantDays: nil},
		{name: "全7曜日は曜日で絞らない", weekOfDays: allWeekOfDays, wantDays: allWeekOfDays},
		{name: "部分指定は該当曜日だけ残る", weekOfDays: []find.WeekOfDays{find.MonDay, find.FriDay}, wantDays: []find.WeekOfDays{find.MonDay, find.FriDay}},
	} {
		t.Run(c.name, func(t *testing.T) {
			findCtx := &FindKyouContext{
				ParsedFindQuery: &find.FindQuery{
					PeriodOfTimeStartTimeSecond: &dayStartSecond,
					PeriodOfTimeEndTimeSecond:   &dayEndSecond,
					PeriodOfTimeWeekOfDays:      c.weekOfDays,
				},
				MatchKyousCurrent: newMatchKyous(),
			}

			f := &FindFilter{}
			if _, err := f.sortAndTrimKyousMap(ctx, findCtx); err != nil {
				t.Fatalf("sortAndTrimKyousMap failed: %v", err)
			}

			wantDaySet := map[find.WeekOfDays]struct{}{}
			for _, day := range c.wantDays {
				wantDaySet[day] = struct{}{}
			}
			if len(findCtx.MatchKyousCurrent) != len(wantDaySet) {
				t.Fatalf("残った件数 = %d, want %d (%v)", len(findCtx.MatchKyousCurrent), len(wantDaySet), keysOfKyouMap(findCtx.MatchKyousCurrent))
			}
			for id := range findCtx.MatchKyousCurrent {
				if _, want := wantDaySet[weekOfDayOfKyou[id]]; !want {
					t.Errorf("指定外の曜日(%d)のKyou %q が残っている", int(weekOfDayOfKyou[id]), id)
				}
			}
		})
	}
}

// 時間帯フィルタの狭い窓（09:00〜10:00）の境界を、秒オブデイ表現とepoch表現の両方で固定する。
//
// 既存の時間帯テストは「1日全体を覆う窓」しか使っておらず、秒の解釈（epoch/秒オブデイ）が
// 変わっても赤くならなかった。この検査が無かったせいで、MCP経路の時間帯検索は
// 1年間ずっと+9時間ずれた窓で動いていた（外部監査で発覚）。
// 解釈の正本は find.NormalizeSecondOfDay: documents/adr/0108-period-of-time-second-of-day.md
func TestSortAndTrimKyousMap_PeriodOfTimeNarrowWindow(t *testing.T) {
	ctx := context.Background()

	// 境界の内外を1秒差で並べる（両端含む）
	day := time.Date(2026, 8, 19, 0, 0, 0, 0, time.Local)
	rows := map[string]time.Time{
		"before":    day.Add(8*time.Hour + 59*time.Minute + 59*time.Second), // 08:59:59 → 落ちる
		"at-start":  day.Add(9 * time.Hour),                                 // 09:00:00 → 残る
		"inside":    day.Add(9*time.Hour + 30*time.Minute),                  // 09:30:00 → 残る
		"at-end":    day.Add(10 * time.Hour),                                // 10:00:00 → 残る
		"after-end": day.Add(10*time.Hour + 1*time.Second),                  // 10:00:01 → 落ちる
	}
	newMatchKyous := func() map[string][]reps.Kyou {
		matchKyous := map[string][]reps.Kyou{}
		for id, relatedTime := range rows {
			matchKyous[id] = []reps.Kyou{{ID: id, DataType: "kmemo", RelatedTime: relatedTime, UpdateTime: relatedTime}}
		}
		return matchKyous
	}
	wantIDs := map[string]bool{"at-start": true, "inside": true, "at-end": true}

	secOfDayStart := int64(9 * 3600)
	secOfDayEnd := int64(10 * 3600)
	epochStart := time.Date(2026, 1, 1, 9, 0, 0, 0, time.Local).Unix()
	epochEnd := time.Date(2026, 1, 1, 10, 0, 0, 0, time.Local).Unix()

	for _, c := range []struct {
		name       string
		start, end int64
	}{
		{name: "秒オブデイ表現(MCP契約)", start: secOfDayStart, end: secOfDayEnd},
		{name: "epoch表現(Web契約)", start: epochStart, end: epochEnd},
	} {
		t.Run(c.name, func(t *testing.T) {
			findCtx := &FindKyouContext{
				ParsedFindQuery: &find.FindQuery{
					PeriodOfTimeStartTimeSecond: &c.start,
					PeriodOfTimeEndTimeSecond:   &c.end,
				},
				MatchKyousCurrent: newMatchKyous(),
			}
			f := &FindFilter{}
			if _, err := f.sortAndTrimKyousMap(ctx, findCtx); err != nil {
				t.Fatalf("sortAndTrimKyousMap failed: %v", err)
			}
			if len(findCtx.MatchKyousCurrent) != len(wantIDs) {
				t.Fatalf("残った件数 = %d, want %d (%v)", len(findCtx.MatchKyousCurrent), len(wantIDs), keysOfKyouMap(findCtx.MatchKyousCurrent))
			}
			for id := range wantIDs {
				if _, exist := findCtx.MatchKyousCurrent[id]; !exist {
					t.Errorf("%q が窓の内側なのに消えている", id)
				}
			}
		})
	}
}

// 夜跨ぎの窓（23:00〜01:00）も両表現で固定する。start > end のとき OR 判定に切り替わる分岐。
func TestSortAndTrimKyousMap_PeriodOfTimeOvernightWindow(t *testing.T) {
	ctx := context.Background()

	day := time.Date(2026, 8, 19, 0, 0, 0, 0, time.Local)
	rows := map[string]time.Time{
		"evening-out": day.Add(22*time.Hour + 59*time.Minute + 59*time.Second), // 22:59:59 → 落ちる
		"at-start":    day.Add(23 * time.Hour),                                 // 23:00:00 → 残る
		"midnight":    day.Add(24 * time.Hour),                                 // 翌00:00:00 → 残る
		"early":       day.Add(24*time.Hour + 30*time.Minute),                  // 翌00:30:00 → 残る
		"at-end":      day.Add(25 * time.Hour),                                 // 翌01:00:00 → 残る
		"after-end":   day.Add(25*time.Hour + 1*time.Second),                   // 翌01:00:01 → 落ちる
		"midday-out":  day.Add(12 * time.Hour),                                 // 12:00:00 → 落ちる
	}
	newMatchKyous := func() map[string][]reps.Kyou {
		matchKyous := map[string][]reps.Kyou{}
		for id, relatedTime := range rows {
			matchKyous[id] = []reps.Kyou{{ID: id, DataType: "kmemo", RelatedTime: relatedTime, UpdateTime: relatedTime}}
		}
		return matchKyous
	}
	wantIDs := map[string]bool{"at-start": true, "midnight": true, "early": true, "at-end": true}

	secOfDayStart := int64(23 * 3600)
	secOfDayEnd := int64(1 * 3600)
	epochStart := time.Date(2026, 1, 1, 23, 0, 0, 0, time.Local).Unix()
	epochEnd := time.Date(2026, 1, 1, 1, 0, 0, 0, time.Local).Unix()

	for _, c := range []struct {
		name       string
		start, end int64
	}{
		{name: "秒オブデイ表現(MCP契約)", start: secOfDayStart, end: secOfDayEnd},
		{name: "epoch表現(Web契約)", start: epochStart, end: epochEnd},
	} {
		t.Run(c.name, func(t *testing.T) {
			findCtx := &FindKyouContext{
				ParsedFindQuery: &find.FindQuery{
					PeriodOfTimeStartTimeSecond: &c.start,
					PeriodOfTimeEndTimeSecond:   &c.end,
				},
				MatchKyousCurrent: newMatchKyous(),
			}
			f := &FindFilter{}
			if _, err := f.sortAndTrimKyousMap(ctx, findCtx); err != nil {
				t.Fatalf("sortAndTrimKyousMap failed: %v", err)
			}
			if len(findCtx.MatchKyousCurrent) != len(wantIDs) {
				t.Fatalf("残った件数 = %d, want %d (%v)", len(findCtx.MatchKyousCurrent), len(wantIDs), keysOfKyouMap(findCtx.MatchKyousCurrent))
			}
			for id := range wantIDs {
				if _, exist := findCtx.MatchKyousCurrent[id]; !exist {
					t.Errorf("%q が夜跨ぎ窓の内側なのに消えている", id)
				}
			}
		})
	}
}
