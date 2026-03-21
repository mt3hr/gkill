package reps

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
)

func TestKmemoAddAndGet(t *testing.T) {
	repo := newTempKmemoRepo(t)
	ctx := context.Background()

	kmemo := makeKmemo("kmemo-001", "テストメモ内容")
	if err := repo.AddKmemoInfo(ctx, kmemo); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}

	got, err := repo.GetKmemo(ctx, "kmemo-001", nil)
	if err != nil {
		t.Fatalf("GetKmemo failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetKmemo returned nil")
	}
	if got.ID != "kmemo-001" {
		t.Errorf("ID = %q, want %q", got.ID, "kmemo-001")
	}
	if got.Content != "テストメモ内容" {
		t.Errorf("Content = %q, want %q", got.Content, "テストメモ内容")
	}
	if got.IsDeleted != false {
		t.Errorf("IsDeleted = %v, want false", got.IsDeleted)
	}
}

func TestKmemoFindKyous_EmptyDB(t *testing.T) {
	repo := newTempKmemoRepo(t)
	ctx := context.Background()

	query := makeDefaultFindQuery()
	kyous, err := repo.FindKyous(ctx, query)
	if err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}
	if len(kyous) != 0 {
		t.Errorf("expected empty result, got %d entries", len(kyous))
	}
}

func TestKmemoFindKyous_WithData(t *testing.T) {
	repo := newTempKmemoRepo(t)
	ctx := context.Background()

	for i, content := range []string{"メモ1", "メモ2", "メモ3"} {
		k := makeKmemo("kmemo-"+string(rune('a'+i)), content)
		k.UpdateTime = k.UpdateTime.Add(time.Duration(i) * time.Second)
		if err := repo.AddKmemoInfo(ctx, k); err != nil {
			t.Fatalf("AddKmemoInfo failed: %v", err)
		}
	}

	query := makeDefaultFindQuery()
	kyous, err := repo.FindKyous(ctx, query)
	if err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}
	if len(kyous) != 3 {
		t.Errorf("expected 3 entries, got %d", len(kyous))
	}
}

func TestKmemoFindKyous_CalendarFilter(t *testing.T) {
	repo := newTempKmemoRepo(t)
	ctx := context.Background()

	t1 := testTime()
	t2 := testTime2()

	k1 := makeKmemo("kmemo-jan", "1月のメモ")
	k1.RelatedTime = t1
	k2 := makeKmemo("kmemo-feb", "2月のメモ")
	k2.RelatedTime = t2

	if err := repo.AddKmemoInfo(ctx, k1); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}
	if err := repo.AddKmemoInfo(ctx, k2); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}

	// Filter for January only
	start, _ := time.Parse(sqlite3impl.TimeLayout, "2025-01-01T00:00:00+09:00")
	end, _ := time.Parse(sqlite3impl.TimeLayout, "2025-01-31T23:59:59+09:00")
	query := makeCalendarFindQuery(start, end)

	kyous, err := repo.FindKyous(ctx, query)
	if err != nil {
		t.Fatalf("FindKyous with calendar filter failed: %v", err)
	}
	if len(kyous) != 1 {
		t.Errorf("expected 1 entry for January, got %d", len(kyous))
	}
}

func TestKmemoFindKyous_WordFilter(t *testing.T) {
	repo := newTempKmemoRepo(t)
	ctx := context.Background()

	k1 := makeKmemo("kmemo-food", "今日のランチはカレーだった")
	k2 := makeKmemo("kmemo-work", "会議の議事録")

	if err := repo.AddKmemoInfo(ctx, k1); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}
	if err := repo.AddKmemoInfo(ctx, k2); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}

	query := makeWordFindQuery([]string{"カレー"})
	kmemos, err := repo.FindKmemo(ctx, query)
	if err != nil {
		t.Fatalf("FindKmemo with word filter failed: %v", err)
	}
	if len(kmemos) != 1 {
		t.Errorf("expected 1 entry matching 'カレー', got %d", len(kmemos))
	}
	if len(kmemos) > 0 && kmemos[0].ID != "kmemo-food" {
		t.Errorf("expected kmemo-food, got %s", kmemos[0].ID)
	}
}

func TestKmemoGetHistories(t *testing.T) {
	repo := newTempKmemoRepo(t)
	ctx := context.Background()

	// Add two versions of the same ID with different update times
	k1 := makeKmemo("kmemo-hist", "初版")
	if err := repo.AddKmemoInfo(ctx, k1); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}

	k2 := makeKmemo("kmemo-hist", "改訂版")
	k2.UpdateTime = k2.UpdateTime.Add(time.Hour)
	if err := repo.AddKmemoInfo(ctx, k2); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}

	histories, err := repo.GetKmemoHistories(ctx, "kmemo-hist")
	if err != nil {
		t.Fatalf("GetKmemoHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}
}

func TestKmemoGetRepName(t *testing.T) {
	repo := newTempKmemoRepo(t)
	ctx := context.Background()

	repName, err := repo.GetRepName(ctx)
	if err != nil {
		t.Fatalf("GetRepName failed: %v", err)
	}
	if repName == "" {
		t.Error("GetRepName returned empty string")
	}
}

func TestKmemoGetPath(t *testing.T) {
	repo := newTempKmemoRepo(t)
	ctx := context.Background()

	kmemo := makeKmemo("kmemo-path", "パステスト")
	if err := repo.AddKmemoInfo(ctx, kmemo); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}

	path, err := repo.GetPath(ctx, "kmemo-path")
	if err != nil {
		t.Fatalf("GetPath failed: %v", err)
	}
	if path == "" {
		t.Error("GetPath returned empty string")
	}
}
