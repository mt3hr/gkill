package api

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
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
		return &FindKyouContext{
			DisableLatestDataRepositoryCache: disableCache,
			ParsedFindQuery:                  &find.FindQuery{},
			Repositories:                     &reps.GkillRepositories{LatestDataRepositoryAddresses: latestDataAddresses},
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
