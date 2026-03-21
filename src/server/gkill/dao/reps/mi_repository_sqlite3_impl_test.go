package reps

import (
	"context"
	"testing"
	"time"
)

func TestMiAddAndGet(t *testing.T) {
	repo := newTempMiRepo(t)
	ctx := context.Background()

	mi := makeMi("mi-001", "タスクA")
	mi.IsChecked = true
	mi.BoardName = "work"
	if err := repo.AddMiInfo(ctx, mi); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	got, err := repo.GetMi(ctx, "mi-001", nil)
	if err != nil {
		t.Fatalf("GetMi failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetMi returned nil")
	}
	if got.Title != "タスクA" {
		t.Errorf("Title = %q, want %q", got.Title, "タスクA")
	}
	if got.IsChecked != true {
		t.Errorf("IsChecked = %v, want true", got.IsChecked)
	}
	if got.BoardName != "work" {
		t.Errorf("BoardName = %q, want %q", got.BoardName, "work")
	}
}

func TestMiFindByBoard(t *testing.T) {
	repo := newTempMiRepo(t)
	ctx := context.Background()

	// Add 2 Mi with board "work"
	m1 := makeMi("mi-w1", "仕事タスク1")
	m1.BoardName = "work"
	m1.UpdateTime = m1.UpdateTime.Add(1 * time.Second)
	if err := repo.AddMiInfo(ctx, m1); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	m2 := makeMi("mi-w2", "仕事タスク2")
	m2.BoardName = "work"
	m2.UpdateTime = m2.UpdateTime.Add(2 * time.Second)
	if err := repo.AddMiInfo(ctx, m2); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	// Add 1 Mi with board "personal"
	m3 := makeMi("mi-p1", "個人タスク")
	m3.BoardName = "personal"
	m3.UpdateTime = m3.UpdateTime.Add(3 * time.Second)
	if err := repo.AddMiInfo(ctx, m3); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	query := makeDefaultFindQuery()
	query.UseMiBoardName = true
	query.MiBoardName = "work"

	mis, err := repo.FindMi(ctx, query)
	if err != nil {
		t.Fatalf("FindMi failed: %v", err)
	}
	// Count unique IDs with board "work"
	workIDs := map[string]bool{}
	for _, m := range mis {
		workIDs[m.ID] = true
	}
	if len(workIDs) < 2 {
		t.Errorf("expected at least 2 unique Mi IDs with board 'work', got %d", len(workIDs))
	}
}

func TestMiFindByCheckState(t *testing.T) {
	repo := newTempMiRepo(t)
	ctx := context.Background()

	// Add 1 checked Mi
	m1 := makeMi("mi-checked", "完了タスク")
	m1.IsChecked = true
	m1.UpdateTime = m1.UpdateTime.Add(1 * time.Second)
	if err := repo.AddMiInfo(ctx, m1); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	// Add 1 unchecked Mi
	m2 := makeMi("mi-unchecked", "未完了タスク")
	m2.IsChecked = false
	m2.UpdateTime = m2.UpdateTime.Add(2 * time.Second)
	if err := repo.AddMiInfo(ctx, m2); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	query := makeDefaultFindQuery()
	mis, err := repo.FindMi(ctx, query)
	if err != nil {
		t.Fatalf("FindMi failed: %v", err)
	}
	uniqueIDs := map[string]bool{}
	for _, m := range mis {
		uniqueIDs[m.ID] = true
	}
	if len(uniqueIDs) < 2 {
		t.Errorf("expected at least 2 unique Mi IDs, got %d", len(uniqueIDs))
	}
}

func TestMiGetBoardNames(t *testing.T) {
	repo := newTempMiRepo(t)
	ctx := context.Background()

	m1 := makeMi("mi-bn1", "仕事タスク")
	m1.BoardName = "work"
	m1.UpdateTime = m1.UpdateTime.Add(1 * time.Second)
	if err := repo.AddMiInfo(ctx, m1); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	m2 := makeMi("mi-bn2", "個人タスク")
	m2.BoardName = "personal"
	m2.UpdateTime = m2.UpdateTime.Add(2 * time.Second)
	if err := repo.AddMiInfo(ctx, m2); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	boards, err := repo.GetBoardNames(ctx)
	if err != nil {
		t.Fatalf("GetBoardNames failed: %v", err)
	}

	boardSet := make(map[string]bool)
	for _, b := range boards {
		boardSet[b] = true
	}
	if !boardSet["work"] {
		t.Errorf("expected board 'work' in results, got %v", boards)
	}
	if !boardSet["personal"] {
		t.Errorf("expected board 'personal' in results, got %v", boards)
	}
}

func TestMiGetHistories(t *testing.T) {
	repo := newTempMiRepo(t)
	ctx := context.Background()

	// Add first version
	m1 := makeMi("mi-hist", "初版タスク")
	if err := repo.AddMiInfo(ctx, m1); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	// Add second version with different UpdateTime
	m2 := makeMi("mi-hist", "改訂版タスク")
	m2.UpdateTime = m2.UpdateTime.Add(time.Hour)
	if err := repo.AddMiInfo(ctx, m2); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	histories, err := repo.GetMiHistories(ctx, "mi-hist")
	if err != nil {
		t.Fatalf("GetMiHistories failed: %v", err)
	}
	if len(histories) < 2 {
		t.Errorf("expected at least 2 history entries, got %d", len(histories))
	}
}
