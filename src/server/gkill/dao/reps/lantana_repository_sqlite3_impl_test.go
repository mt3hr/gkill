package reps

import (
	"context"
	"testing"
	"time"
)

func makeLantana(id string, mood int) Lantana {
	now := testTime()
	return Lantana{
		IsDeleted:    false,
		ID:           id,
		Mood:         mood,
		DataType:     "lantana",
		RelatedTime:  now,
		CreateTime:   now,
		CreateApp:    "test_app",
		CreateDevice: "test_device",
		CreateUser:   "test_user",
		UpdateTime:   now,
		UpdateApp:    "test_app",
		UpdateUser:   "test_user",
		UpdateDevice: "test_device",
	}
}

func TestLantanaAddAndGet(t *testing.T) {
	repo := newTempLantanaRepo(t)
	ctx := context.Background()

	lantana := makeLantana("lantana-001", 7)
	if err := repo.AddLantanaInfo(ctx, lantana); err != nil {
		t.Fatalf("AddLantanaInfo failed: %v", err)
	}

	got, err := repo.GetLantana(ctx, "lantana-001", nil)
	if err != nil {
		t.Fatalf("GetLantana failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetLantana returned nil")
	}
	if got.Mood != 7 {
		t.Errorf("Mood = %d, want %d", got.Mood, 7)
	}
}

func TestLantanaFindKyous(t *testing.T) {
	repo := newTempLantanaRepo(t)
	ctx := context.Background()

	l1 := makeLantana("lantana-a", 5)
	l2 := makeLantana("lantana-b", 8)
	l2.UpdateTime = l2.UpdateTime.Add(time.Second)

	if err := repo.AddLantanaInfo(ctx, l1); err != nil {
		t.Fatalf("AddLantanaInfo failed: %v", err)
	}
	if err := repo.AddLantanaInfo(ctx, l2); err != nil {
		t.Fatalf("AddLantanaInfo failed: %v", err)
	}

	query := makeDefaultFindQuery()
	kyouMap, err := repo.FindKyous(ctx, query)
	if err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}
	total := 0
	for _, v := range kyouMap {
		total += len(v)
	}
	if total != 2 {
		t.Errorf("expected 2 entries, got %d", total)
	}
}

func TestLantanaFindLantana(t *testing.T) {
	repo := newTempLantanaRepo(t)
	ctx := context.Background()

	l1 := makeLantana("lantana-c", 3)
	l2 := makeLantana("lantana-d", 9)
	l2.UpdateTime = l2.UpdateTime.Add(time.Second)

	if err := repo.AddLantanaInfo(ctx, l1); err != nil {
		t.Fatalf("AddLantanaInfo failed: %v", err)
	}
	if err := repo.AddLantanaInfo(ctx, l2); err != nil {
		t.Fatalf("AddLantanaInfo failed: %v", err)
	}

	query := makeDefaultFindQuery()
	lantanas, err := repo.FindLantana(ctx, query)
	if err != nil {
		t.Fatalf("FindLantana failed: %v", err)
	}
	if len(lantanas) != 2 {
		t.Errorf("expected 2 entries, got %d", len(lantanas))
	}
}

func TestLantanaGetHistories(t *testing.T) {
	repo := newTempLantanaRepo(t)
	ctx := context.Background()

	l1 := makeLantana("lantana-hist", 4)
	if err := repo.AddLantanaInfo(ctx, l1); err != nil {
		t.Fatalf("AddLantanaInfo failed: %v", err)
	}

	l2 := makeLantana("lantana-hist", 6)
	l2.UpdateTime = l2.UpdateTime.Add(time.Hour)
	if err := repo.AddLantanaInfo(ctx, l2); err != nil {
		t.Fatalf("AddLantanaInfo failed: %v", err)
	}

	histories, err := repo.GetLantanaHistories(ctx, "lantana-hist")
	if err != nil {
		t.Fatalf("GetLantanaHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}
}
