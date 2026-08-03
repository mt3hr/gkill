package reps

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
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

// TestKmemoFindKyous_NotWordFilter は、除外ワードを指定した検索から
// そのワードを含むレコードが確かに落ちることを確認する。
//
// 否定条件の生成が
//   ( COL NOT LIKE ? OR ID NOT LIKE ? )
// になっていたころは、IDがUUIDで検索語を含むことは実質ないため右辺が常に真になり、
// 除外がまったく効いていなかった。ド・モルガンによりANDでなければならない。
func TestKmemoFindKyous_NotWordFilter(t *testing.T) {
	repo := newTempKmemoRepo(t)
	ctx := context.Background()

	for _, k := range []Kmemo{
		makeKmemo("kmemo-food", "今日のランチはカレーだった"),
		makeKmemo("kmemo-work", "会議の議事録"),
	} {
		if err := repo.AddKmemoInfo(ctx, k); err != nil {
			t.Fatalf("AddKmemoInfo failed: %v", err)
		}
	}

	query := &find.FindQuery{
		UseWords:       true,
		NotWords:       []string{"カレー"},
		OnlyLatestData: true,
	}
	kmemos, err := repo.FindKmemo(ctx, query)
	if err != nil {
		t.Fatalf("FindKmemo with not-word filter failed: %v", err)
	}

	for _, k := range kmemos {
		if k.ID == "kmemo-food" {
			t.Error("除外ワードを含むレコードが検索結果に残っている")
		}
	}
	if len(kmemos) != 1 || (len(kmemos) > 0 && kmemos[0].ID != "kmemo-work") {
		t.Errorf("除外ワードを含まないレコードだけが残るべき: %+v", kmemos)
	}
}

// TestKmemoRepositories_GetKmemoReturnsNewestAcrossReps は、同じIDが複数の
// リポジトリに存在するとき、UpdateTime が最新のものが返ることを確認する。
//
// 集約ループのコメントは「UpdateTimeが最新のものを収める」なのに、比較が
//   if 候補.UpdateTime.Before(現在の勝者.UpdateTime) { 現在の勝者 = 候補 }
// となっており、実際には最古のものが残っていた。
// 同種のループは全体で101箇所あり、うち24箇所がこの向き違いだった。
func TestKmemoRepositories_GetKmemoReturnsNewestAcrossReps(t *testing.T) {
	ctx := context.Background()
	oldRepo := newTempKmemoRepo(t)
	newRepo := newTempKmemoRepo(t)

	older := makeKmemo("dup-kmemo", "古い内容")
	newer := makeKmemo("dup-kmemo", "新しい内容")
	newer.UpdateTime = older.UpdateTime.Add(1 * time.Hour)

	if err := oldRepo.AddKmemoInfo(ctx, older); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}
	if err := newRepo.AddKmemoInfo(ctx, newer); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}

	// 並び順に依存しないよう、両方の順序で確認する
	for _, c := range []struct {
		name string
		reps KmemoRepositories
	}{
		{"古いrepが先", KmemoRepositories{oldRepo, newRepo}},
		{"新しいrepが先", KmemoRepositories{newRepo, oldRepo}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.reps.GetKmemo(ctx, "dup-kmemo", nil)
			if err != nil {
				t.Fatalf("GetKmemo failed: %v", err)
			}
			if got == nil {
				t.Fatal("GetKmemo returned nil")
			}
			if got.Content != "新しい内容" {
				t.Errorf("Content = %q, want %q（古い版が返っている）", got.Content, "新しい内容")
			}
		})
	}
}
