package reps

import (
	"context"
	"testing"
	"time"
)

func TestTextAddAndGet(t *testing.T) {
	repo := newTempTextRepo(t)
	ctx := context.Background()

	text := makeText("text-001", "target-001", "テスト本文")
	if err := repo.AddTextInfo(ctx, text); err != nil {
		t.Fatalf("AddTextInfo failed: %v", err)
	}

	got, err := repo.GetText(ctx, "text-001", nil)
	if err != nil {
		t.Fatalf("GetText failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetText returned nil")
	}
	if got.ID != "text-001" {
		t.Errorf("ID = %q, want %q", got.ID, "text-001")
	}
	if got.TargetID != "target-001" {
		t.Errorf("TargetID = %q, want %q", got.TargetID, "target-001")
	}
	if got.Text != "テスト本文" {
		t.Errorf("Text = %q, want %q", got.Text, "テスト本文")
	}
}

func TestTextGetByTargetID(t *testing.T) {
	repo := newTempTextRepo(t)
	ctx := context.Background()

	// Add 2 texts with the same target_id
	t1 := makeText("text-a", "target-shared", "テキストA")
	t1.UpdateTime = t1.UpdateTime.Add(1 * time.Second)
	if err := repo.AddTextInfo(ctx, t1); err != nil {
		t.Fatalf("AddTextInfo failed: %v", err)
	}

	t2 := makeText("text-b", "target-shared", "テキストB")
	t2.UpdateTime = t2.UpdateTime.Add(2 * time.Second)
	if err := repo.AddTextInfo(ctx, t2); err != nil {
		t.Fatalf("AddTextInfo failed: %v", err)
	}

	// Add 1 text with a different target_id
	t3 := makeText("text-c", "target-other", "テキストC")
	t3.UpdateTime = t3.UpdateTime.Add(3 * time.Second)
	if err := repo.AddTextInfo(ctx, t3); err != nil {
		t.Fatalf("AddTextInfo failed: %v", err)
	}

	texts, err := repo.GetTextsByTargetID(ctx, "target-shared")
	if err != nil {
		t.Fatalf("GetTextsByTargetID failed: %v", err)
	}
	if len(texts) != 2 {
		t.Errorf("expected 2 texts for target-shared, got %d", len(texts))
	}
}

func TestTextFindTexts(t *testing.T) {
	repo := newTempTextRepo(t)
	ctx := context.Background()

	text := makeText("text-find-001", "target-find", "検索テスト")
	if err := repo.AddTextInfo(ctx, text); err != nil {
		t.Fatalf("AddTextInfo failed: %v", err)
	}

	query := makeDefaultFindQuery()
	texts, err := repo.FindTexts(ctx, query)
	if err != nil {
		t.Fatalf("FindTexts failed: %v", err)
	}
	if len(texts) != 1 {
		t.Errorf("expected 1 text, got %d", len(texts))
	}
}

func TestTextGetHistories(t *testing.T) {
	repo := newTempTextRepo(t)
	ctx := context.Background()

	// Add first version
	t1 := makeText("text-hist", "target-hist", "初版")
	if err := repo.AddTextInfo(ctx, t1); err != nil {
		t.Fatalf("AddTextInfo failed: %v", err)
	}

	// Add second version with different UpdateTime
	t2 := makeText("text-hist", "target-hist", "改訂版")
	t2.UpdateTime = t2.UpdateTime.Add(time.Hour)
	if err := repo.AddTextInfo(ctx, t2); err != nil {
		t.Fatalf("AddTextInfo failed: %v", err)
	}

	histories, err := repo.GetTextHistories(ctx, "text-hist")
	if err != nil {
		t.Fatalf("GetTextHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}
}
