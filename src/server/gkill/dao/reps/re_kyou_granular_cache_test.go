package reps

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

func sanitizeTestUserID(name string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_", " ", "_")
	return replacer.Replace(name)
}

func newGranularReKyouFixture(t *testing.T) (*GkillRepositories, string, string, string) {
	t.Helper()
	ctx := context.Background()

	repositories, err := NewGkillRepositories(sanitizeTestUserID(t.Name()))
	if err != nil {
		t.Fatalf("failed to create repositories: %v", err)
	}
	repositories.ReKyouReps.GkillRepositories = repositories

	dir := t.TempDir()
	t.Cleanup(func() { _ = repositories.Close(context.Background()) })

	kmemoRep, err := NewKmemoRepositorySQLite3Impl(ctx, filepath.Join(dir, "kmemo.db"), true)
	if err != nil {
		t.Fatalf("failed to create kmemo repo: %v", err)
	}
	repositories.KmemoReps = KmemoRepositories{kmemoRep}
	repositories.WriteKmemoRep = kmemoRep

	targetID := "shared-target"
	if err := kmemoRep.AddKmemoInfo(ctx, makeKmemo(targetID, "target content")); err != nil {
		t.Fatalf("failed to add kmemo: %v", err)
	}

	pixelRep, err := NewReKyouRepositorySQLite3Impl(ctx, filepath.Join(dir, "ReKyou_Pixel9a_20250919.db"), true, repositories)
	if err != nil {
		t.Fatalf("failed to create pixel rekyou repo: %v", err)
	}
	pixelRepName, err := pixelRep.GetRepName(ctx)
	if err != nil {
		t.Fatalf("failed to get pixel rep name: %v", err)
	}

	s11Rep, err := NewReKyouRepositorySQLite3Impl(ctx, filepath.Join(dir, "ReKyou_S11_20260226.db"), true, repositories)
	if err != nil {
		t.Fatalf("failed to create s11 rekyou repo: %v", err)
	}
	s11RepName, err := s11Rep.GetRepName(ctx)
	if err != nil {
		t.Fatalf("failed to get s11 rep name: %v", err)
	}

	sharedID := "rekyou-shared"
	rekyouBase := makeReKyou(sharedID, targetID)
	rekyouBase.RepName = pixelRepName
	if err := pixelRep.AddReKyouInfo(ctx, rekyouBase); err != nil {
		t.Fatalf("failed to add pixel rekyou: %v", err)
	}

	rekyouLatest := rekyouBase
	rekyouLatest.RepName = s11RepName
	rekyouLatest.RelatedTime = rekyouLatest.RelatedTime.Add(time.Hour)
	rekyouLatest.CreateTime = rekyouLatest.CreateTime.Add(time.Hour)
	rekyouLatest.UpdateTime = rekyouLatest.UpdateTime.Add(time.Hour)
	if err := s11Rep.AddReKyouInfo(ctx, rekyouLatest); err != nil {
		t.Fatalf("failed to add s11 rekyou: %v", err)
	}

	originalReKyouReps := ReKyouRepositories{
		ReKyouRepositories: []ReKyouRepository{pixelRep, s11Rep},
		GkillRepositories:  repositories,
	}
	cachedReKyouRep, err := NewReKyouRepositoryCachedSQLite3Impl(ctx, &originalReKyouReps, repositories, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, sanitizeTestUserID(t.Name())+"_REKYOU_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached rekyou repo: %v", err)
	}

	repositories.ReKyouReps = ReKyouRepositories{
		ReKyouRepositories: []ReKyouRepository{cachedReKyouRep},
		GkillRepositories:  repositories,
	}
	repositories.Reps = Repositories{kmemoRep, cachedReKyouRep}
	repositories.WriteReKyouRep = pixelRep

	return repositories, pixelRepName, s11RepName, sharedID
}

func TestGkillRepositories_UpdateCacheBuildsReKyouOnFirstPass(t *testing.T) {
	ctx := context.Background()
	old := gkill_options.CacheReKyouReps
	enable := true
	gkill_options.CacheReKyouReps = &enable
	t.Cleanup(func() { gkill_options.CacheReKyouReps = old })

	repositories, pixelRepName, s11RepName, sharedID := newGranularReKyouFixture(t)

	if err := repositories.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache() error: %v", err)
	}

	matchKyous, err := repositories.ReKyouReps.FindKyous(ctx, makeDefaultFindQuery())
	if err != nil {
		t.Fatalf("FindKyous() error: %v", err)
	}
	if len(matchKyous) == 0 {
		t.Fatalf("FindKyous() returned no results on first cache build")
	}

	matchReKyous, err := repositories.ReKyouReps.FindReKyou(ctx, makeDefaultFindQuery())
	if err != nil {
		t.Fatalf("FindReKyou() error: %v", err)
	}
	if len(matchReKyous) == 0 {
		t.Fatalf("FindReKyou() returned no results on first cache build")
	}

	allRepNames, err := repositories.GetAllRepNames(ctx)
	if err != nil {
		t.Fatalf("GetAllRepNames() error: %v", err)
	}
	if !slices.Contains(allRepNames, pixelRepName) {
		t.Fatalf("GetAllRepNames() missing %s: %#v", pixelRepName, allRepNames)
	}
	if !slices.Contains(allRepNames, s11RepName) {
		t.Fatalf("GetAllRepNames() missing %s: %#v", s11RepName, allRepNames)
	}
	if slices.Contains(allRepNames, "ReKyou") {
		t.Fatalf("GetAllRepNames() should not include aggregate ReKyou: %#v", allRepNames)
	}

	addr, err := repositories.LatestDataRepositoryAddressDAO.GetLatestDataRepositoryAddress(ctx, sharedID)
	if err != nil {
		t.Fatalf("GetLatestDataRepositoryAddress() error: %v", err)
	}
	if addr.LatestDataRepositoryName != s11RepName {
		t.Fatalf("LatestDataRepositoryName = %s, want %s", addr.LatestDataRepositoryName, s11RepName)
	}

	got, err := repositories.GetKyou(ctx, sharedID, nil)
	if err != nil {
		t.Fatalf("GetKyou() error: %v", err)
	}
	if got == nil {
		t.Fatalf("GetKyou() returned nil")
	}
	if got.RepName != s11RepName {
		t.Fatalf("GetKyou().RepName = %s, want %s", got.RepName, s11RepName)
	}
}

func TestReKyouHistoriesByGranularRepNameWithCachedRepo(t *testing.T) {
	ctx := context.Background()
	old := gkill_options.CacheReKyouReps
	enable := true
	gkill_options.CacheReKyouReps = &enable
	t.Cleanup(func() { gkill_options.CacheReKyouReps = old })

	repositories, pixelRepName, _, sharedID := newGranularReKyouFixture(t)

	if err := repositories.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache() error: %v", err)
	}

	kyouHistories, err := repositories.Reps.GetKyouHistoriesByRepName(ctx, sharedID, &pixelRepName)
	if err != nil {
		t.Fatalf("GetKyouHistoriesByRepName() error: %v", err)
	}
	if len(kyouHistories) != 1 {
		t.Fatalf("GetKyouHistoriesByRepName() len = %d, want 1", len(kyouHistories))
	}
	if kyouHistories[0].RepName != pixelRepName {
		t.Fatalf("GetKyouHistoriesByRepName() rep = %s, want %s", kyouHistories[0].RepName, pixelRepName)
	}

	reKyouHistories, err := repositories.ReKyouReps.GetReKyouHistoriesByRepName(ctx, sharedID, &pixelRepName)
	if err != nil {
		t.Fatalf("GetReKyouHistoriesByRepName() error: %v", err)
	}
	if len(reKyouHistories) != 1 {
		t.Fatalf("GetReKyouHistoriesByRepName() len = %d, want 1", len(reKyouHistories))
	}
	if reKyouHistories[0].RepName != pixelRepName {
		t.Fatalf("GetReKyouHistoriesByRepName() rep = %s, want %s", reKyouHistories[0].RepName, pixelRepName)
	}
}

// TestReKyouGetReKyousByTargetIDReturnsEvenIfTargetDeleted は、参照先Kyouを論理削除したあとでも
// 逆引きがReKyouを返すことを押さえます。
//
// FindKyous は参照先の LatestDataRepositoryAddress.IsDeleted を見てReKyouを結果から外しますが、
// GetReKyousByTargetID はそのフィルタを持ちません。Kyou削除の連鎖処理が削除の前後どちらからでも
// 残骸を辿れるようにするための、意図的な差です。ここを揃えてしまうと連鎖が途切れます。
func TestReKyouGetReKyousByTargetIDReturnsEvenIfTargetDeleted(t *testing.T) {
	ctx := context.Background()
	old := gkill_options.CacheReKyouReps
	enable := true
	gkill_options.CacheReKyouReps = &enable
	t.Cleanup(func() { gkill_options.CacheReKyouReps = old })

	repositories, _, _, sharedID := newGranularReKyouFixture(t)
	// newGranularReKyouFixture が作るKmemoのID
	targetID := "shared-target"

	if err := repositories.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache() error: %v", err)
	}

	// 参照先が生きているうちはFindKyousにも出る
	matchKyous, err := repositories.ReKyouReps.FindKyous(ctx, makeDefaultFindQuery())
	if err != nil {
		t.Fatalf("FindKyous() error: %v", err)
	}
	if len(matchKyous) == 0 {
		t.Fatalf("FindKyous() returned no results before deleting the target")
	}

	// 参照先Kmemoを論理削除する。
	// アドレス表の更新まで含めて usecase.UpdateKmemo と同じことをする（repへの追記だけでは
	// LatestDataRepositoryAddress.IsDeleted が立たず、参照先削除フィルタが効かない）
	deletedKmemo := makeKmemo(targetID, "target content")
	deletedKmemo.IsDeleted = true
	deletedKmemo.UpdateTime = deletedKmemo.UpdateTime.Add(time.Hour)
	if err := repositories.WriteKmemoRep.AddKmemoInfo(ctx, deletedKmemo); err != nil {
		t.Fatalf("AddKmemoInfo() error: %v", err)
	}
	kmemoRepName, err := repositories.WriteKmemoRep.GetRepName(ctx)
	if err != nil {
		t.Fatalf("GetRepName() error: %v", err)
	}
	deletedAddress := gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              true,
		TargetID:                               targetID,
		DataUpdateTime:                         deletedKmemo.UpdateTime,
		LatestDataRepositoryName:               kmemoRepName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	repositories.SetLatestDataRepositoryAddress(targetID, deletedAddress)
	if _, err := repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, deletedAddress); err != nil {
		t.Fatalf("AddOrUpdateLatestDataRepositoryAddress() error: %v", err)
	}

	// FindKyousは参照先削除フィルタで0件になる
	matchKyous, err = repositories.ReKyouReps.FindKyous(ctx, makeDefaultFindQuery())
	if err != nil {
		t.Fatalf("FindKyous() error: %v", err)
	}
	if len(matchKyous) != 0 {
		t.Fatalf("FindKyous() len = %d, want 0 after deleting the target", len(matchKyous))
	}

	// 逆引きはフィルタを持たないので返り続ける
	matchReKyous, err := repositories.GetReKyousByTargetID(ctx, targetID)
	if err != nil {
		t.Fatalf("GetReKyousByTargetID() error: %v", err)
	}
	if len(matchReKyous) != 1 {
		t.Fatalf("GetReKyousByTargetID() len = %d, want 1 even after deleting the target", len(matchReKyous))
	}
	if matchReKyous[0].ID != sharedID {
		t.Fatalf("GetReKyousByTargetID() ID = %s, want %s", matchReKyous[0].ID, sharedID)
	}
}
