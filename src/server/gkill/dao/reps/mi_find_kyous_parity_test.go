package reps

// mi rep の FindKyous が cached / 非cached で同じ結果を返すことのパリティテスト。
//
// 修正対象のバグ:
//   - cached の create 分岐だけ ignoreCase=false で、大小違いのワード検索の結果が
//     cache有無で変わっていた
//   - 非cached の check/limit/start/end 分岐だけ onlyLatestData=true 固定で、
//     OnlyLatestData=false の検索の行数が cache有無で変わっていた

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

func TestMiFindKyousParityCachedVsNonCached(t *testing.T) {
	ctx := context.Background()

	baseRepo, err := NewMiRepositorySQLite3Impl(ctx, filepath.Join(t.TempDir(), "mi.db"), true)
	if err != nil {
		t.Fatalf("failed to create mi repository: %v", err)
	}
	t.Cleanup(func() { baseRepo.Close(ctx) })

	cacheDB := openMemoryDB(t)
	cachedRepo, err := NewMiRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, &sync.RWMutex{}, "MI_PARITY_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached mi repository: %v", err)
	}
	t.Cleanup(func() { cachedRepo.Close(ctx) })

	// cached実装のAddはキャッシュ側にしか書かないため、
	// 実体(base)へ書いてからUpdateCacheでキャッシュへ同期し、同一データを両実装に持たせる
	baseTime := time.Date(2026, 8, 1, 12, 0, 0, 0, time.Local)
	limitTime := baseTime.Add(24 * time.Hour)
	mi := makeMi("mi-parity-001", "Buy Milk")
	mi.LimitTime = &limitTime
	if err := baseRepo.AddMiInfo(ctx, mi); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}
	if err := cachedRepo.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}

	queries := map[string]*find.FindQuery{
		// 大小違いのワード検索(以前はcachedのmi_create行だけヒットしなかった)
		"lowercase word": func() *find.FindQuery {
			q := makeDefaultFindQuery()
			q.IncludeCreateMi = true
			q.Words = []string{"buy"}
			return q
		}(),
		// 全版検索(以前は非cachedのcheck/limit等の分岐だけ最新版に絞られていた)
		"all versions": func() *find.FindQuery {
			q := makeDefaultFindQuery()
			q.OnlyLatestData = false
			return q
		}(),
	}

	for name, query := range queries {
		cachedResult, err := cachedRepo.FindKyous(ctx, query)
		if err != nil {
			t.Fatalf("%s: cached FindKyous failed: %v", name, err)
		}
		baseResult, err := baseRepo.FindKyous(ctx, query)
		if err != nil {
			t.Fatalf("%s: base FindKyous failed: %v", name, err)
		}

		cachedKeys := flattenKyouKeys(cachedResult)
		baseKeys := flattenKyouKeys(baseResult)
		if !slices.Equal(cachedKeys, baseKeys) {
			t.Errorf("%s: cachedと非cachedの結果が食い違う:\n  cached=%v\n  base=%v", name, cachedKeys, baseKeys)
		}
		if name == "lowercase word" && len(cachedKeys) == 0 {
			t.Errorf("%s: 大小違いのワード検索がヒットしていない", name)
		}
	}
}

func flattenKyouKeys(kyousMap map[string][]Kyou) []string {
	keys := []string{}
	for _, kyous := range kyousMap {
		for _, kyou := range kyous {
			keys = append(keys, fmt.Sprintf("%s/%s/%d", kyou.ID, kyou.DataType, kyou.UpdateTime.Unix()))
		}
	}
	slices.Sort(keys)
	return keys
}
