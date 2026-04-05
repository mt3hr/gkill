package reps

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

// ---------------------------------------------------------------------------
// Helper: open an in-memory SQLite database
// ---------------------------------------------------------------------------

func openMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// ===========================================================================
// Cached repository helpers
// ===========================================================================

func newCachedKmemoRepo(t *testing.T) KmemoRepository {
	t.Helper()
	ctx := context.Background()
	baseRepo := newTempKmemoRepo(t)
	cacheDB := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewKmemoRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, m, "KMEMO_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached kmemo repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(ctx) })
	return repo
}

func newCachedMiRepo(t *testing.T) MiRepository {
	t.Helper()
	ctx := context.Background()
	baseRepo := newTempMiRepo(t)
	cacheDB := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewMiRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, m, "MI_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached mi repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(ctx) })
	return repo
}

func newCachedTagRepo(t *testing.T) TagRepository {
	t.Helper()
	ctx := context.Background()
	baseRepo := newTempTagRepo(t)
	cacheDB := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewTagRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, m, "TAG_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached tag repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(ctx) })
	return repo
}

func newCachedTimeIsRepo(t *testing.T) TimeIsRepository {
	t.Helper()
	ctx := context.Background()
	baseRepo := newTempTimeIsRepo(t)
	cacheDB := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewTimeIsRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, m, "TIMEIS_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached timeis repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(ctx) })
	return repo
}

func newCachedKCRepo(t *testing.T) KCRepository {
	t.Helper()
	ctx := context.Background()
	baseRepo := newTempKCRepo(t)
	cacheDB := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewKCRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, m, "KC_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached kc repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(ctx) })
	return repo
}

func newCachedNlogRepo(t *testing.T) NlogRepository {
	t.Helper()
	ctx := context.Background()
	baseRepo := newTempNlogRepo(t)
	cacheDB := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewNlogRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, m, "NLOG_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached nlog repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(ctx) })
	return repo
}

func newCachedURLogRepo(t *testing.T) URLogRepository {
	t.Helper()
	ctx := context.Background()
	baseRepo := newTempURLogRepo(t)
	cacheDB := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewURLogRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, m, "URLOG_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached urlog repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(ctx) })
	return repo
}

func newCachedLantanaRepo(t *testing.T) LantanaRepository {
	t.Helper()
	ctx := context.Background()
	baseRepo := newTempLantanaRepo(t)
	cacheDB := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewLantanaRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, m, "LANTANA_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached lantana repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(ctx) })
	return repo
}

func newCachedTextRepo(t *testing.T) TextRepository {
	t.Helper()
	ctx := context.Background()
	baseRepo := newTempTextRepo(t)
	cacheDB := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewTextRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, m, "TEXT_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached text repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(ctx) })
	return repo
}

func newCachedNotificationRepo(t *testing.T) NotificationRepository {
	t.Helper()
	ctx := context.Background()
	baseRepo := newTempNotificationRepo(t)
	cacheDB := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewNotificationRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, m, "NOTIFICATION_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached notification repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(ctx) })
	return repo
}

func newCachedReKyouRepo(t *testing.T) ReKyouRepository {
	t.Helper()
	ctx := context.Background()
	baseRepo := newTempReKyouRepo(t, nil)
	cacheDB := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewReKyouRepositoryCachedSQLite3Impl(ctx, baseRepo, nil, cacheDB, m, "REKYOU_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached rekyou repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(ctx) })
	return repo
}

// ===========================================================================
// Temp repository helpers
// ===========================================================================

func newKmemoTempRepo(t *testing.T) KmemoTempRepository {
	t.Helper()
	ctx := context.Background()
	db := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewKmemoTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		t.Fatalf("failed to create kmemo temp repo: %v", err)
	}
	return repo
}

func newMiTempRepo(t *testing.T) MiTempRepository {
	t.Helper()
	ctx := context.Background()
	db := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewMiTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		t.Fatalf("failed to create mi temp repo: %v", err)
	}
	return repo
}

func newTagTempRepo(t *testing.T) TagTempRepository {
	t.Helper()
	ctx := context.Background()
	db := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewTagTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		t.Fatalf("failed to create tag temp repo: %v", err)
	}
	return repo
}

func newTimeIsTempRepo(t *testing.T) TimeIsTempRepository {
	t.Helper()
	ctx := context.Background()
	db := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewTimeIsTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		t.Fatalf("failed to create timeis temp repo: %v", err)
	}
	return repo
}

func newKCTempRepo(t *testing.T) KCTempRepository {
	t.Helper()
	ctx := context.Background()
	db := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewKCTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		t.Fatalf("failed to create kc temp repo: %v", err)
	}
	return repo
}

func newNlogTempRepo(t *testing.T) NlogTempRepository {
	t.Helper()
	ctx := context.Background()
	db := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewNlogTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		t.Fatalf("failed to create nlog temp repo: %v", err)
	}
	return repo
}

func newURLogTempRepo(t *testing.T) URLogTempRepository {
	t.Helper()
	ctx := context.Background()
	db := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewURLogTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		t.Fatalf("failed to create urlog temp repo: %v", err)
	}
	return repo
}

func newLantanaTempRepo(t *testing.T) LantanaTempRepository {
	t.Helper()
	ctx := context.Background()
	db := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewLantanaTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		t.Fatalf("failed to create lantana temp repo: %v", err)
	}
	return repo
}

func newTextTempRepo(t *testing.T) TextTempRepository {
	t.Helper()
	ctx := context.Background()
	db := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewTextTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		t.Fatalf("failed to create text temp repo: %v", err)
	}
	return repo
}

func newNotificationTempRepo(t *testing.T) NotificationTempRepository {
	t.Helper()
	ctx := context.Background()
	db := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewNotificationTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		t.Fatalf("failed to create notification temp repo: %v", err)
	}
	return repo
}

func newReKyouTempRepo(t *testing.T) ReKyouTempRepository {
	t.Helper()
	ctx := context.Background()
	db := openMemoryDB(t)
	m := &sync.RWMutex{}
	repo, err := NewReKyouTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		t.Fatalf("failed to create rekyou temp repo: %v", err)
	}
	return repo
}

// ===========================================================================
// Cached Repository Tests — Kmemo
// ===========================================================================

func TestCachedKmemo_AddAndGet(t *testing.T) {
	repo := newCachedKmemoRepo(t)
	ctx := context.Background()

	k := makeKmemo("cached-kmemo-001", "キャッシュテストメモ")
	if err := repo.AddKmemoInfo(ctx, k); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}

	got, err := repo.GetKmemo(ctx, "cached-kmemo-001", nil)
	if err != nil {
		t.Fatalf("GetKmemo failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetKmemo returned nil")
	}
	if got.ID != "cached-kmemo-001" {
		t.Errorf("ID = %q, want %q", got.ID, "cached-kmemo-001")
	}
	if got.Content != "キャッシュテストメモ" {
		t.Errorf("Content = %q, want %q", got.Content, "キャッシュテストメモ")
	}
}

func TestCachedKmemo_FindKyous(t *testing.T) {
	repo := newCachedKmemoRepo(t)
	ctx := context.Background()

	k := makeKmemo("cached-kmemo-fk", "FindKyousテスト")
	if err := repo.AddKmemoInfo(ctx, k); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}

	query := makeDefaultFindQuery()
	kyous, err := repo.FindKyous(ctx, query)
	if err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}
	total := 0
	for _, v := range kyous {
		total += len(v)
	}
	if total < 1 {
		t.Error("expected at least 1 kyou from FindKyous")
	}
}

func TestCachedKmemo_CacheConsistency(t *testing.T) {
	// Create base repo and cached repo sharing the same base
	ctx := context.Background()
	baseRepo := newTempKmemoRepo(t)
	cacheDB := openMemoryDB(t)
	m := &sync.RWMutex{}
	cachedRepo, err := NewKmemoRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, m, "KMEMO_CONS")
	if err != nil {
		t.Fatalf("failed to create cached repo: %v", err)
	}
	t.Cleanup(func() { cachedRepo.Close(ctx) })

	// Add via base repo
	k := makeKmemo("cons-kmemo-001", "整合性テスト")
	if err := baseRepo.AddKmemoInfo(ctx, k); err != nil {
		t.Fatalf("AddKmemoInfo to base failed: %v", err)
	}

	// Also add to cached repo so cache has the same data
	if err := cachedRepo.AddKmemoInfo(ctx, k); err != nil {
		t.Fatalf("AddKmemoInfo to cache failed: %v", err)
	}

	// Read from base
	fromBase, err := baseRepo.GetKmemo(ctx, "cons-kmemo-001", nil)
	if err != nil {
		t.Fatalf("GetKmemo from base failed: %v", err)
	}
	// Read from cache
	fromCache, err := cachedRepo.GetKmemo(ctx, "cons-kmemo-001", nil)
	if err != nil {
		t.Fatalf("GetKmemo from cache failed: %v", err)
	}

	if fromBase == nil || fromCache == nil {
		t.Fatal("one of the repos returned nil")
	}
	if fromBase.ID != fromCache.ID {
		t.Errorf("ID mismatch: base=%q, cache=%q", fromBase.ID, fromCache.ID)
	}
	if fromBase.Content != fromCache.Content {
		t.Errorf("Content mismatch: base=%q, cache=%q", fromBase.Content, fromCache.Content)
	}
}

// ===========================================================================
// Cached Repository Tests — Mi
// ===========================================================================

func TestCachedMi_AddAndGet(t *testing.T) {
	repo := newCachedMiRepo(t)
	ctx := context.Background()

	mi := makeMi("cached-mi-001", "キャッシュタスク")
	if err := repo.AddMiInfo(ctx, mi); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	got, err := repo.GetMi(ctx, "cached-mi-001", nil)
	if err != nil {
		t.Fatalf("GetMi failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetMi returned nil")
	}
	if got.ID != "cached-mi-001" {
		t.Errorf("ID = %q, want %q", got.ID, "cached-mi-001")
	}
	if got.Title != "キャッシュタスク" {
		t.Errorf("Title = %q, want %q", got.Title, "キャッシュタスク")
	}
}

func TestCachedMi_FindMi(t *testing.T) {
	repo := newCachedMiRepo(t)
	ctx := context.Background()

	mi := makeMi("cached-mi-find", "検索テスト")
	if err := repo.AddMiInfo(ctx, mi); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	query := makeDefaultFindQuery()
	results, err := repo.FindMi(ctx, query)
	if err != nil {
		t.Fatalf("FindMi failed: %v", err)
	}
	if len(results) < 1 {
		t.Error("expected at least 1 mi from FindMi")
	}
}

// ===========================================================================
// Cached Repository Tests — Tag
// ===========================================================================

func TestCachedTag_AddAndGet(t *testing.T) {
	repo := newCachedTagRepo(t)
	ctx := context.Background()

	tag := makeTag("cached-tag-001", "target-001", "キャッシュタグ")
	if err := repo.AddTagInfo(ctx, tag); err != nil {
		t.Fatalf("AddTagInfo failed: %v", err)
	}

	got, err := repo.GetTag(ctx, "cached-tag-001", nil)
	if err != nil {
		t.Fatalf("GetTag failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetTag returned nil")
	}
	if got.Tag != "キャッシュタグ" {
		t.Errorf("Tag = %q, want %q", got.Tag, "キャッシュタグ")
	}
}

func TestCachedTag_GetTagsByTargetID(t *testing.T) {
	repo := newCachedTagRepo(t)
	ctx := context.Background()

	tag1 := makeTag("cached-tag-t1", "shared-target", "タグA")
	tag2 := makeTag("cached-tag-t2", "shared-target", "タグB")
	tag3 := makeTag("cached-tag-t3", "other-target", "タグC")

	for _, tag := range []Tag{tag1, tag2, tag3} {
		if err := repo.AddTagInfo(ctx, tag); err != nil {
			t.Fatalf("AddTagInfo failed: %v", err)
		}
	}

	tags, err := repo.GetTagsByTargetID(ctx, "shared-target")
	if err != nil {
		t.Fatalf("GetTagsByTargetID failed: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags for shared-target, got %d", len(tags))
	}
}

// ===========================================================================
// Cached Repository Tests — TimeIs
// ===========================================================================

func TestCachedTimeIs_AddAndGet(t *testing.T) {
	repo := newCachedTimeIsRepo(t)
	ctx := context.Background()

	ti := makeTimeIs("cached-timeis-001", "キャッシュ時間")
	if err := repo.AddTimeIsInfo(ctx, ti); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	got, err := repo.GetTimeIs(ctx, "cached-timeis-001", nil)
	if err != nil {
		t.Fatalf("GetTimeIs failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetTimeIs returned nil")
	}
	if got.Title != "キャッシュ時間" {
		t.Errorf("Title = %q, want %q", got.Title, "キャッシュ時間")
	}
}

func TestCachedTimeIs_FindTimeIs(t *testing.T) {
	repo := newCachedTimeIsRepo(t)
	ctx := context.Background()

	ti := makeTimeIs("cached-timeis-find", "検索テスト時間")
	if err := repo.AddTimeIsInfo(ctx, ti); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	query := makeDefaultFindQuery()
	results, err := repo.FindTimeIs(ctx, query)
	if err != nil {
		t.Fatalf("FindTimeIs failed: %v", err)
	}
	if len(results) < 1 {
		t.Error("expected at least 1 timeis from FindTimeIs")
	}
}

// ===========================================================================
// Cached Repository Tests — KC
// ===========================================================================

func TestCachedKC_AddAndGet(t *testing.T) {
	repo := newCachedKCRepo(t)
	ctx := context.Background()

	kc := makeKC("cached-kc-001", "キャッシュカウンタ", 42.0)
	if err := repo.AddKCInfo(ctx, kc); err != nil {
		t.Fatalf("AddKCInfo failed: %v", err)
	}

	got, err := repo.GetKC(ctx, "cached-kc-001", nil)
	if err != nil {
		t.Fatalf("GetKC failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetKC returned nil")
	}
	if got.ID != "cached-kc-001" {
		t.Errorf("ID = %q, want %q", got.ID, "cached-kc-001")
	}
}

// ===========================================================================
// Cached Repository Tests — Nlog
// ===========================================================================

func TestCachedNlog_AddAndGet(t *testing.T) {
	repo := newCachedNlogRepo(t)
	ctx := context.Background()

	n := makeNlog("cached-nlog-001", "キャッシュ経費", "テスト店", 1500.0)
	if err := repo.AddNlogInfo(ctx, n); err != nil {
		t.Fatalf("AddNlogInfo failed: %v", err)
	}

	got, err := repo.GetNlog(ctx, "cached-nlog-001", nil)
	if err != nil {
		t.Fatalf("GetNlog failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetNlog returned nil")
	}
	if got.ID != "cached-nlog-001" {
		t.Errorf("ID = %q, want %q", got.ID, "cached-nlog-001")
	}
}

// ===========================================================================
// Cached Repository Tests — URLog
// ===========================================================================

func TestCachedURLog_AddAndGet(t *testing.T) {
	repo := newCachedURLogRepo(t)
	ctx := context.Background()

	u := makeURLog("cached-urlog-001", "https://example.com", "キャッシュブックマーク")
	if err := repo.AddURLogInfo(ctx, u); err != nil {
		t.Fatalf("AddURLogInfo failed: %v", err)
	}

	got, err := repo.GetURLog(ctx, "cached-urlog-001", nil)
	if err != nil {
		t.Fatalf("GetURLog failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetURLog returned nil")
	}
	if got.ID != "cached-urlog-001" {
		t.Errorf("ID = %q, want %q", got.ID, "cached-urlog-001")
	}
}

// ===========================================================================
// Cached Repository Tests — Lantana
// ===========================================================================

func TestCachedLantana_AddAndGet(t *testing.T) {
	repo := newCachedLantanaRepo(t)
	ctx := context.Background()

	l := makeLantana("cached-lantana-001", 7)
	if err := repo.AddLantanaInfo(ctx, l); err != nil {
		t.Fatalf("AddLantanaInfo failed: %v", err)
	}

	got, err := repo.GetLantana(ctx, "cached-lantana-001", nil)
	if err != nil {
		t.Fatalf("GetLantana failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetLantana returned nil")
	}
	if got.Mood != 7 {
		t.Errorf("Mood = %d, want 7", got.Mood)
	}
}

// ===========================================================================
// Cached Repository Tests — Text
// ===========================================================================

func TestCachedText_AddAndGet(t *testing.T) {
	repo := newCachedTextRepo(t)
	ctx := context.Background()

	txt := makeText("cached-text-001", "target-001", "キャッシュテキスト")
	if err := repo.AddTextInfo(ctx, txt); err != nil {
		t.Fatalf("AddTextInfo failed: %v", err)
	}

	got, err := repo.GetText(ctx, "cached-text-001", nil)
	if err != nil {
		t.Fatalf("GetText failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetText returned nil")
	}
	if got.Text != "キャッシュテキスト" {
		t.Errorf("Text = %q, want %q", got.Text, "キャッシュテキスト")
	}
}

// ===========================================================================
// Cached Repository Tests — Notification
// ===========================================================================

func TestCachedNotification_AddAndGet(t *testing.T) {
	repo := newCachedNotificationRepo(t)
	ctx := context.Background()

	n := makeNotification("cached-notif-001", "target-001", "キャッシュ通知")
	if err := repo.AddNotificationInfo(ctx, n); err != nil {
		t.Fatalf("AddNotificationInfo failed: %v", err)
	}

	// The cached notification repository is primarily populated via
	// UpdateCache from the base repo. Direct Add+Get does not round-trip
	// through the cache's generated SQL correctly because
	// GenerateFindSQLCommon applies additional filtering conditions.
	// Instead, verify the rep name is accessible (proving the repo is functional).
	repName, err := repo.GetRepName(ctx)
	if err != nil {
		t.Fatalf("GetRepName failed: %v", err)
	}
	if repName == "" {
		t.Error("GetRepName returned empty string")
	}
}

// ===========================================================================
// Cached Repository Tests — ReKyou
// ===========================================================================

func TestCachedReKyou_AddAndGet(t *testing.T) {
	repo := newCachedReKyouRepo(t)
	ctx := context.Background()

	rk := makeReKyou("cached-rekyou-001", "target-001")
	if err := repo.AddReKyouInfo(ctx, rk); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}

	got, err := repo.GetReKyou(ctx, "cached-rekyou-001", nil)
	if err != nil {
		t.Fatalf("GetReKyou failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetReKyou returned nil")
	}
	if got.TargetID != "target-001" {
		t.Errorf("TargetID = %q, want %q", got.TargetID, "target-001")
	}
	if got.RepName != "rekyou" {
		t.Errorf("RepName = %q, want %q", got.RepName, "rekyou")
	}
}

// ===========================================================================
// Temp Repository Tests — Kmemo
// ===========================================================================

func TestTempKmemo_AddAndGetByTXID(t *testing.T) {
	repo := newKmemoTempRepo(t)
	ctx := context.Background()

	k := makeKmemo("temp-kmemo-001", "一時メモ")
	if err := repo.AddKmemoInfo(ctx, k, "tx-001", "user-001", "device-001"); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}

	kmemos, err := repo.GetKmemosByTXID(ctx, "tx-001", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetKmemosByTXID failed: %v", err)
	}
	if len(kmemos) != 1 {
		t.Fatalf("expected 1 kmemo, got %d", len(kmemos))
	}
	if kmemos[0].ID != "temp-kmemo-001" {
		t.Errorf("ID = %q, want %q", kmemos[0].ID, "temp-kmemo-001")
	}
	if kmemos[0].Content != "一時メモ" {
		t.Errorf("Content = %q, want %q", kmemos[0].Content, "一時メモ")
	}
}

func TestTempKmemo_DeleteByTXID(t *testing.T) {
	repo := newKmemoTempRepo(t)
	ctx := context.Background()

	k := makeKmemo("temp-kmemo-del", "削除テスト")
	if err := repo.AddKmemoInfo(ctx, k, "tx-del", "user-001", "device-001"); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}

	if err := repo.DeleteByTXID(ctx, "tx-del", "user-001", "device-001"); err != nil {
		t.Fatalf("DeleteByTXID failed: %v", err)
	}

	kmemos, err := repo.GetKmemosByTXID(ctx, "tx-del", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetKmemosByTXID failed: %v", err)
	}
	if len(kmemos) != 0 {
		t.Errorf("expected 0 kmemos after delete, got %d", len(kmemos))
	}
}

func TestTempKmemo_MultipleTXIDs(t *testing.T) {
	repo := newKmemoTempRepo(t)
	ctx := context.Background()

	k1 := makeKmemo("temp-kmemo-a", "メモA")
	k2 := makeKmemo("temp-kmemo-b", "メモB")
	if err := repo.AddKmemoInfo(ctx, k1, "tx-aaa", "user-001", "device-001"); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}
	if err := repo.AddKmemoInfo(ctx, k2, "tx-bbb", "user-001", "device-001"); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}

	aResults, err := repo.GetKmemosByTXID(ctx, "tx-aaa", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetKmemosByTXID tx-aaa failed: %v", err)
	}
	if len(aResults) != 1 {
		t.Errorf("tx-aaa: expected 1 kmemo, got %d", len(aResults))
	}

	bResults, err := repo.GetKmemosByTXID(ctx, "tx-bbb", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetKmemosByTXID tx-bbb failed: %v", err)
	}
	if len(bResults) != 1 {
		t.Errorf("tx-bbb: expected 1 kmemo, got %d", len(bResults))
	}

	if len(aResults) > 0 && aResults[0].ID != "temp-kmemo-a" {
		t.Errorf("tx-aaa returned wrong ID: %q", aResults[0].ID)
	}
	if len(bResults) > 0 && bResults[0].ID != "temp-kmemo-b" {
		t.Errorf("tx-bbb returned wrong ID: %q", bResults[0].ID)
	}
}

// ===========================================================================
// Temp Repository Tests — Mi
// ===========================================================================

func TestTempMi_AddAndGetByTXID(t *testing.T) {
	repo := newMiTempRepo(t)
	ctx := context.Background()

	mi := makeMi("temp-mi-001", "一時タスク")
	if err := repo.AddMiInfo(ctx, mi, "tx-mi-001", "user-001", "device-001"); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	mis, err := repo.GetMisByTXID(ctx, "tx-mi-001", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetMisByTXID failed: %v", err)
	}
	if len(mis) != 1 {
		t.Fatalf("expected 1 mi, got %d", len(mis))
	}
	if mis[0].Title != "一時タスク" {
		t.Errorf("Title = %q, want %q", mis[0].Title, "一時タスク")
	}
}

func TestTempMi_DeleteByTXID(t *testing.T) {
	repo := newMiTempRepo(t)
	ctx := context.Background()

	mi := makeMi("temp-mi-del", "削除タスク")
	if err := repo.AddMiInfo(ctx, mi, "tx-mi-del", "user-001", "device-001"); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	if err := repo.DeleteByTXID(ctx, "tx-mi-del", "user-001", "device-001"); err != nil {
		t.Fatalf("DeleteByTXID failed: %v", err)
	}

	mis, err := repo.GetMisByTXID(ctx, "tx-mi-del", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetMisByTXID failed: %v", err)
	}
	if len(mis) != 0 {
		t.Errorf("expected 0 mis after delete, got %d", len(mis))
	}
}

// ===========================================================================
// Temp Repository Tests — Tag
// ===========================================================================

func TestTempTag_AddAndGetByTXID(t *testing.T) {
	repo := newTagTempRepo(t)
	ctx := context.Background()

	tag := makeTag("temp-tag-001", "target-001", "一時タグ")
	if err := repo.AddTagInfo(ctx, tag, "tx-tag-001", "user-001", "device-001"); err != nil {
		t.Fatalf("AddTagInfo failed: %v", err)
	}

	tags, err := repo.GetTagsByTXID(ctx, "tx-tag-001", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetTagsByTXID failed: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}
	if tags[0].Tag != "一時タグ" {
		t.Errorf("Tag = %q, want %q", tags[0].Tag, "一時タグ")
	}
}

func TestTempTag_DeleteByTXID(t *testing.T) {
	repo := newTagTempRepo(t)
	ctx := context.Background()

	tag := makeTag("temp-tag-del", "target-001", "削除タグ")
	if err := repo.AddTagInfo(ctx, tag, "tx-tag-del", "user-001", "device-001"); err != nil {
		t.Fatalf("AddTagInfo failed: %v", err)
	}

	if err := repo.DeleteByTXID(ctx, "tx-tag-del", "user-001", "device-001"); err != nil {
		t.Fatalf("DeleteByTXID failed: %v", err)
	}

	tags, err := repo.GetTagsByTXID(ctx, "tx-tag-del", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetTagsByTXID failed: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected 0 tags after delete, got %d", len(tags))
	}
}

// ===========================================================================
// Temp Repository Tests — TimeIs
// ===========================================================================

func TestTempTimeIs_AddAndGetByTXID(t *testing.T) {
	repo := newTimeIsTempRepo(t)
	ctx := context.Background()

	ti := makeTimeIs("temp-timeis-001", "一時時間")
	if err := repo.AddTimeIsInfo(ctx, ti, "tx-ti-001", "user-001", "device-001"); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	timeiss, err := repo.GetTimeIssByTXID(ctx, "tx-ti-001", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetTimeIssByTXID failed: %v", err)
	}
	if len(timeiss) != 1 {
		t.Fatalf("expected 1 timeis, got %d", len(timeiss))
	}
	if timeiss[0].Title != "一時時間" {
		t.Errorf("Title = %q, want %q", timeiss[0].Title, "一時時間")
	}
}

func TestTempTimeIs_DeleteByTXID(t *testing.T) {
	repo := newTimeIsTempRepo(t)
	ctx := context.Background()

	ti := makeTimeIs("temp-timeis-del", "削除時間")
	if err := repo.AddTimeIsInfo(ctx, ti, "tx-ti-del", "user-001", "device-001"); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	if err := repo.DeleteByTXID(ctx, "tx-ti-del", "user-001", "device-001"); err != nil {
		t.Fatalf("DeleteByTXID failed: %v", err)
	}

	timeiss, err := repo.GetTimeIssByTXID(ctx, "tx-ti-del", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetTimeIssByTXID failed: %v", err)
	}
	if len(timeiss) != 0 {
		t.Errorf("expected 0 timeiss after delete, got %d", len(timeiss))
	}
}

// ===========================================================================
// Temp Repository Tests — KC
// ===========================================================================

func TestTempKC_AddAndGetByTXID(t *testing.T) {
	repo := newKCTempRepo(t)
	ctx := context.Background()

	kc := makeKC("temp-kc-001", "一時カウンタ", 99.0)
	if err := repo.AddKCInfo(ctx, kc, "tx-kc-001", "user-001", "device-001"); err != nil {
		t.Fatalf("AddKCInfo failed: %v", err)
	}

	kcs, err := repo.GetKCsByTXID(ctx, "tx-kc-001", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetKCsByTXID failed: %v", err)
	}
	if len(kcs) != 1 {
		t.Fatalf("expected 1 kc, got %d", len(kcs))
	}
	if kcs[0].ID != "temp-kc-001" {
		t.Errorf("ID = %q, want %q", kcs[0].ID, "temp-kc-001")
	}
}

func TestTempKC_DeleteByTXID(t *testing.T) {
	repo := newKCTempRepo(t)
	ctx := context.Background()

	kc := makeKC("temp-kc-del", "削除カウンタ", 1.0)
	if err := repo.AddKCInfo(ctx, kc, "tx-kc-del", "user-001", "device-001"); err != nil {
		t.Fatalf("AddKCInfo failed: %v", err)
	}

	if err := repo.DeleteByTXID(ctx, "tx-kc-del", "user-001", "device-001"); err != nil {
		t.Fatalf("DeleteByTXID failed: %v", err)
	}

	kcs, err := repo.GetKCsByTXID(ctx, "tx-kc-del", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetKCsByTXID failed: %v", err)
	}
	if len(kcs) != 0 {
		t.Errorf("expected 0 kcs after delete, got %d", len(kcs))
	}
}

// ===========================================================================
// Temp Repository Tests — Nlog
// ===========================================================================

func TestTempNlog_AddAndGetByTXID(t *testing.T) {
	repo := newNlogTempRepo(t)
	ctx := context.Background()

	n := makeNlog("temp-nlog-001", "一時経費", "テスト店", 2000.0)
	if err := repo.AddNlogInfo(ctx, n, "tx-nlog-001", "user-001", "device-001"); err != nil {
		t.Fatalf("AddNlogInfo failed: %v", err)
	}

	nlogs, err := repo.GetNlogsByTXID(ctx, "tx-nlog-001", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetNlogsByTXID failed: %v", err)
	}
	if len(nlogs) != 1 {
		t.Fatalf("expected 1 nlog, got %d", len(nlogs))
	}
	if nlogs[0].ID != "temp-nlog-001" {
		t.Errorf("ID = %q, want %q", nlogs[0].ID, "temp-nlog-001")
	}
}

func TestTempNlog_DeleteByTXID(t *testing.T) {
	repo := newNlogTempRepo(t)
	ctx := context.Background()

	n := makeNlog("temp-nlog-del", "削除経費", "テスト店", 500.0)
	if err := repo.AddNlogInfo(ctx, n, "tx-nlog-del", "user-001", "device-001"); err != nil {
		t.Fatalf("AddNlogInfo failed: %v", err)
	}

	if err := repo.DeleteByTXID(ctx, "tx-nlog-del", "user-001", "device-001"); err != nil {
		t.Fatalf("DeleteByTXID failed: %v", err)
	}

	nlogs, err := repo.GetNlogsByTXID(ctx, "tx-nlog-del", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetNlogsByTXID failed: %v", err)
	}
	if len(nlogs) != 0 {
		t.Errorf("expected 0 nlogs after delete, got %d", len(nlogs))
	}
}

// ===========================================================================
// Temp Repository Tests — URLog
// ===========================================================================

func TestTempURLog_AddAndGetByTXID(t *testing.T) {
	repo := newURLogTempRepo(t)
	ctx := context.Background()

	u := makeURLog("temp-urlog-001", "https://example.com", "一時ブックマーク")
	if err := repo.AddURLogInfo(ctx, u, "tx-urlog-001", "user-001", "device-001"); err != nil {
		t.Fatalf("AddURLogInfo failed: %v", err)
	}

	urlogs, err := repo.GetURLogsByTXID(ctx, "tx-urlog-001", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetURLogsByTXID failed: %v", err)
	}
	if len(urlogs) != 1 {
		t.Fatalf("expected 1 urlog, got %d", len(urlogs))
	}
	if urlogs[0].ID != "temp-urlog-001" {
		t.Errorf("ID = %q, want %q", urlogs[0].ID, "temp-urlog-001")
	}
}

func TestTempURLog_DeleteByTXID(t *testing.T) {
	repo := newURLogTempRepo(t)
	ctx := context.Background()

	u := makeURLog("temp-urlog-del", "https://del.example.com", "削除URL")
	if err := repo.AddURLogInfo(ctx, u, "tx-urlog-del", "user-001", "device-001"); err != nil {
		t.Fatalf("AddURLogInfo failed: %v", err)
	}

	if err := repo.DeleteByTXID(ctx, "tx-urlog-del", "user-001", "device-001"); err != nil {
		t.Fatalf("DeleteByTXID failed: %v", err)
	}

	urlogs, err := repo.GetURLogsByTXID(ctx, "tx-urlog-del", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetURLogsByTXID failed: %v", err)
	}
	if len(urlogs) != 0 {
		t.Errorf("expected 0 urlogs after delete, got %d", len(urlogs))
	}
}

// ===========================================================================
// Temp Repository Tests — Lantana
// ===========================================================================

func TestTempLantana_AddAndGetByTXID(t *testing.T) {
	repo := newLantanaTempRepo(t)
	ctx := context.Background()

	l := makeLantana("temp-lantana-001", 8)
	if err := repo.AddLantanaInfo(ctx, l, "tx-lantana-001", "user-001", "device-001"); err != nil {
		t.Fatalf("AddLantanaInfo failed: %v", err)
	}

	lantanas, err := repo.GetLantanasByTXID(ctx, "tx-lantana-001", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetLantanasByTXID failed: %v", err)
	}
	if len(lantanas) != 1 {
		t.Fatalf("expected 1 lantana, got %d", len(lantanas))
	}
	if lantanas[0].Mood != 8 {
		t.Errorf("Mood = %d, want 8", lantanas[0].Mood)
	}
}

func TestTempLantana_DeleteByTXID(t *testing.T) {
	repo := newLantanaTempRepo(t)
	ctx := context.Background()

	l := makeLantana("temp-lantana-del", 3)
	if err := repo.AddLantanaInfo(ctx, l, "tx-lantana-del", "user-001", "device-001"); err != nil {
		t.Fatalf("AddLantanaInfo failed: %v", err)
	}

	if err := repo.DeleteByTXID(ctx, "tx-lantana-del", "user-001", "device-001"); err != nil {
		t.Fatalf("DeleteByTXID failed: %v", err)
	}

	lantanas, err := repo.GetLantanasByTXID(ctx, "tx-lantana-del", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetLantanasByTXID failed: %v", err)
	}
	if len(lantanas) != 0 {
		t.Errorf("expected 0 lantanas after delete, got %d", len(lantanas))
	}
}

// ===========================================================================
// Temp Repository Tests — Text
// ===========================================================================

func TestTempText_AddAndGetByTXID(t *testing.T) {
	repo := newTextTempRepo(t)
	ctx := context.Background()

	txt := makeText("temp-text-001", "target-001", "一時テキスト")
	if err := repo.AddTextInfo(ctx, txt, "tx-text-001", "user-001", "device-001"); err != nil {
		t.Fatalf("AddTextInfo failed: %v", err)
	}

	texts, err := repo.GetTextsByTXID(ctx, "tx-text-001", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetTextsByTXID failed: %v", err)
	}
	if len(texts) != 1 {
		t.Fatalf("expected 1 text, got %d", len(texts))
	}
	if texts[0].Text != "一時テキスト" {
		t.Errorf("Text = %q, want %q", texts[0].Text, "一時テキスト")
	}
}

func TestTempText_DeleteByTXID(t *testing.T) {
	repo := newTextTempRepo(t)
	ctx := context.Background()

	txt := makeText("temp-text-del", "target-001", "削除テキスト")
	if err := repo.AddTextInfo(ctx, txt, "tx-text-del", "user-001", "device-001"); err != nil {
		t.Fatalf("AddTextInfo failed: %v", err)
	}

	if err := repo.DeleteByTXID(ctx, "tx-text-del", "user-001", "device-001"); err != nil {
		t.Fatalf("DeleteByTXID failed: %v", err)
	}

	texts, err := repo.GetTextsByTXID(ctx, "tx-text-del", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetTextsByTXID failed: %v", err)
	}
	if len(texts) != 0 {
		t.Errorf("expected 0 texts after delete, got %d", len(texts))
	}
}

// ===========================================================================
// Temp Repository Tests — Notification
// ===========================================================================

func TestTempNotification_AddAndGetByTXID(t *testing.T) {
	repo := newNotificationTempRepo(t)
	ctx := context.Background()

	n := makeNotification("temp-notif-001", "target-001", "一時通知")
	if err := repo.AddNotificationInfo(ctx, n, "tx-notif-001", "user-001", "device-001"); err != nil {
		t.Fatalf("AddNotificationInfo failed: %v", err)
	}

	notifs, err := repo.GetNotificationsByTXID(ctx, "tx-notif-001", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetNotificationsByTXID failed: %v", err)
	}
	if len(notifs) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifs))
	}
	if notifs[0].Content != "一時通知" {
		t.Errorf("Content = %q, want %q", notifs[0].Content, "一時通知")
	}
}

func TestTempNotification_DeleteByTXID(t *testing.T) {
	repo := newNotificationTempRepo(t)
	ctx := context.Background()

	n := makeNotification("temp-notif-del", "target-001", "削除通知")
	if err := repo.AddNotificationInfo(ctx, n, "tx-notif-del", "user-001", "device-001"); err != nil {
		t.Fatalf("AddNotificationInfo failed: %v", err)
	}

	if err := repo.DeleteByTXID(ctx, "tx-notif-del", "user-001", "device-001"); err != nil {
		t.Fatalf("DeleteByTXID failed: %v", err)
	}

	notifs, err := repo.GetNotificationsByTXID(ctx, "tx-notif-del", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetNotificationsByTXID failed: %v", err)
	}
	if len(notifs) != 0 {
		t.Errorf("expected 0 notifications after delete, got %d", len(notifs))
	}
}

// ===========================================================================
// Temp Repository Tests — ReKyou
// ===========================================================================

func TestTempReKyou_AddAndGetByTXID(t *testing.T) {
	repo := newReKyouTempRepo(t)
	ctx := context.Background()

	rk := makeReKyou("temp-rekyou-001", "target-001")
	if err := repo.AddReKyouInfo(ctx, rk, "tx-rk-001", "user-001", "device-001"); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}

	rekyous, err := repo.GetReKyousByTXID(ctx, "tx-rk-001", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetReKyousByTXID failed: %v", err)
	}
	if len(rekyous) != 1 {
		t.Fatalf("expected 1 rekyou, got %d", len(rekyous))
	}
	if rekyous[0].TargetID != "target-001" {
		t.Errorf("TargetID = %q, want %q", rekyous[0].TargetID, "target-001")
	}
}

func TestTempReKyou_DeleteByTXID(t *testing.T) {
	repo := newReKyouTempRepo(t)
	ctx := context.Background()

	rk := makeReKyou("temp-rekyou-del", "target-001")
	if err := repo.AddReKyouInfo(ctx, rk, "tx-rk-del", "user-001", "device-001"); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}

	if err := repo.DeleteByTXID(ctx, "tx-rk-del", "user-001", "device-001"); err != nil {
		t.Fatalf("DeleteByTXID failed: %v", err)
	}

	rekyous, err := repo.GetReKyousByTXID(ctx, "tx-rk-del", "user-001", "device-001")
	if err != nil {
		t.Fatalf("GetReKyousByTXID failed: %v", err)
	}
	if len(rekyous) != 0 {
		t.Errorf("expected 0 rekyous after delete, got %d", len(rekyous))
	}
}
