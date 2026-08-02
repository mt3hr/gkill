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

func TestFindFilterConstruction(t *testing.T) {
	f := &FindFilter{}
	if f == nil {
		t.Fatal("FindFilter construction returned nil")
	}
}

func TestNoTagsConstant(t *testing.T) {
	if NoTags != "no tags" {
		t.Errorf("NoTags = %q, want %q", NoTags, "no tags")
	}
}

// replaceLatestKyouInfos は、マッチした版がグローバル最新(LatestDataRepositoryAddress)で
// なければレコードを除外し、最新版が含まれていればその版のentryのみ残す。
// キャッシュ有無(DisableLatestDataRepositoryCache)の両ブランチで同じ挙動になることを確認する。
func TestReplaceLatestKyouInfos_ExcludeStaleKeepLatest(t *testing.T) {
	const targetID = "timeis-1"
	const latestRepName = "TimeIs_Pixel9a_20250919"
	staleUpdateTime := time.Date(2026, 6, 27, 19, 2, 0, 0, time.Local)  // 古い版(可視rep)
	latestUpdateTime := time.Date(2026, 6, 28, 22, 2, 31, 0, time.Local) // グローバル最新

	latestDataAddresses := map[string]gkill_cache.LatestDataRepositoryAddress{
		targetID: {TargetID: targetID, LatestDataRepositoryName: latestRepName, DataUpdateTime: latestUpdateTime},
	}

	timeIsKyou := func(dataType, repName string, updateTime time.Time) reps.Kyou {
		return reps.Kyou{ID: targetID, DataType: dataType, RepName: repName, UpdateTime: updateTime}
	}

	newContext := func(disableCache bool, matchKyous map[string][]reps.Kyou) *FindKyouContext {
		repositories := &reps.GkillRepositories{}
		repositories.SetLatestDataRepositoryAddresses(latestDataAddresses)
		return &FindKyouContext{
			DisableLatestDataRepositoryCache: disableCache,
			ParsedFindQuery:                  &find.FindQuery{},
			Repositories:                     repositories,
			MatchKyousCurrent:                matchKyous,
		}
	}

	for _, disableCache := range []bool{true, false} {
		f := &FindFilter{}

		// 可視repのマッチが古い版だけ → グローバル最新は別repなので除外される
		staleOnly := newContext(disableCache, map[string][]reps.Kyou{
			targetID: {
				timeIsKyou("timeis_start", "TimeIs", staleUpdateTime),
				timeIsKyou("timeis_end", "TimeIs", staleUpdateTime),
			},
		})
		if _, err := f.replaceLatestKyouInfos(context.Background(), staleOnly); err != nil {
			t.Fatalf("disableCache=%v: replaceLatestKyouInfos error: %v", disableCache, err)
		}
		if _, ok := staleOnly.MatchKyousCurrent[targetID]; ok {
			t.Errorf("disableCache=%v: 古い版だけのレコードは除外されるべき", disableCache)
		}

		// 古い版と最新版が混在 → 最新版のstart/end 2件のみ残る
		staleAndLatest := newContext(disableCache, map[string][]reps.Kyou{
			targetID: {
				timeIsKyou("timeis_start", "TimeIs", staleUpdateTime),
				timeIsKyou("timeis_start", latestRepName, latestUpdateTime),
				timeIsKyou("timeis_end", latestRepName, latestUpdateTime),
			},
		})
		if _, err := f.replaceLatestKyouInfos(context.Background(), staleAndLatest); err != nil {
			t.Fatalf("disableCache=%v: replaceLatestKyouInfos error: %v", disableCache, err)
		}
		kept := staleAndLatest.MatchKyousCurrent[targetID]
		if len(kept) != 2 {
			t.Fatalf("disableCache=%v: 最新版の2件が残るべき, got %d", disableCache, len(kept))
		}
		for _, kyou := range kept {
			if !kyou.UpdateTime.Equal(latestUpdateTime) {
				t.Errorf("disableCache=%v: 古い版が残っている: %#v", disableCache, kyou)
			}
		}
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
				ParsedFindQuery:                  &find.FindQuery{UseTags: true, Tags: []string{c.tagName}},
				Repositories:                     repositories,
				MatchTags:                        map[string]reps.Tag{},
			}

			f := &FindFilter{}
			if _, err := f.findTags(ctx, findCtx); err != nil {
				t.Fatalf("findTags failed: %v", err)
			}
			if len(findCtx.MatchTags) != c.wantCount {
				t.Errorf("disableCache=%v tag=%q: MatchTags = %d, want %d", disableCache, c.tagName, len(findCtx.MatchTags), c.wantCount)
			}
		}
	}
}
