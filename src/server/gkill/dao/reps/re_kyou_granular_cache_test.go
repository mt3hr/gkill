package reps

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

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
