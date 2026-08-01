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

// TestTextRepositoriesGetTextPicksLatestAcrossReps は、同一本文IDの新しい版が
// 別のrepにあるとき、rep跨ぎ集約が最新版を選ぶことを確認する。
// repの並び順に依らず最新版が選ばれること
func TestTextRepositoriesGetTextPicksLatestAcrossReps(t *testing.T) {
	ctx := context.Background()

	oldVersion := makeText("cross-rep-text", "cross-rep-target", "編集前の本文")
	newVersion := makeText("cross-rep-text", "cross-rep-target", "編集後の本文")
	newVersion.UpdateTime = oldVersion.UpdateTime.Add(1 * time.Minute)

	for _, order := range []string{"old_first", "new_first"} {
		t.Run(order, func(t *testing.T) {
			repWithOld := newTempTextRepo(t)
			repWithNew := newTempTextRepo(t)
			if err := repWithOld.AddTextInfo(ctx, oldVersion); err != nil {
				t.Fatalf("AddTextInfo failed: %v", err)
			}
			if err := repWithNew.AddTextInfo(ctx, newVersion); err != nil {
				t.Fatalf("AddTextInfo failed: %v", err)
			}

			textReps := TextRepositories{repWithOld, repWithNew}
			if order == "new_first" {
				textReps = TextRepositories{repWithNew, repWithOld}
			}

			got, err := textReps.GetText(ctx, "cross-rep-text", nil)
			if err != nil {
				t.Fatalf("GetText failed: %v", err)
			}
			if got == nil {
				t.Fatal("GetText returned nil")
			}
			if got.Text != "編集後の本文" {
				t.Errorf("Text = %q, want %q (latest version)", got.Text, "編集後の本文")
			}
		})
	}
}

// TestTextOnlyLatestVersionIsVisible は、TEXTテーブルがappend-onlyであることを踏まえ、
// IDごとにUPDATE_TIMEが最大の版だけが見えることを確認する。
// 編集前の本文で検索にヒットしてはいけない
func TestTextOnlyLatestVersionIsVisible(t *testing.T) {
	repo := newTempTextRepo(t)
	ctx := context.Background()

	old := makeText("text-edited", "target-edited", "編集前の本文")
	edited := makeText("text-edited", "target-edited", "編集後の本文")
	edited.UpdateTime = old.UpdateTime.Add(1 * time.Minute)

	for _, text := range []Text{old, edited} {
		if err := repo.AddTextInfo(ctx, text); err != nil {
			t.Fatalf("AddTextInfo failed: %v", err)
		}
	}

	t.Run("GetTextsByTargetID", func(t *testing.T) {
		texts, err := repo.GetTextsByTargetID(ctx, "target-edited")
		if err != nil {
			t.Fatalf("GetTextsByTargetID failed: %v", err)
		}
		if len(texts) != 1 {
			t.Fatalf("expected 1 text (latest version only), got %d: %v", len(texts), texts)
		}
		if texts[0].Text != "編集後の本文" {
			t.Errorf("Text = %q, want %q", texts[0].Text, "編集後の本文")
		}
	})

	t.Run("GetText", func(t *testing.T) {
		got, err := repo.GetText(ctx, "text-edited", nil)
		if err != nil {
			t.Fatalf("GetText failed: %v", err)
		}
		if got == nil {
			t.Fatal("GetText returned nil")
		}
		if got.Text != "編集後の本文" {
			t.Errorf("Text = %q, want %q (latest version)", got.Text, "編集後の本文")
		}
		if !got.UpdateTime.Equal(edited.UpdateTime) {
			t.Errorf("UpdateTime = %v, want %v (latest version)", got.UpdateTime, edited.UpdateTime)
		}

		// UpdateTimeを指定した場合は履歴表示用にその版が取れる
		oldUpdateTime := old.UpdateTime
		gotOld, err := repo.GetText(ctx, "text-edited", &oldUpdateTime)
		if err != nil {
			t.Fatalf("GetText with update time failed: %v", err)
		}
		if gotOld == nil {
			t.Fatal("GetText with update time returned nil")
		}
		if gotOld.Text != "編集前の本文" {
			t.Errorf("Text = %q, want %q (specified version)", gotOld.Text, "編集前の本文")
		}
	})

	t.Run("FindTexts", func(t *testing.T) {
		oldTexts, err := repo.FindTexts(ctx, makeWordFindQuery([]string{"編集前"}))
		if err != nil {
			t.Fatalf("FindTexts failed: %v", err)
		}
		if len(oldTexts) != 0 {
			t.Errorf("expected 0 texts for pre-edit word '編集前', got %d", len(oldTexts))
		}

		newTexts, err := repo.FindTexts(ctx, makeWordFindQuery([]string{"編集後"}))
		if err != nil {
			t.Fatalf("FindTexts failed: %v", err)
		}
		if len(newTexts) != 1 {
			t.Errorf("expected 1 text for '編集後', got %d", len(newTexts))
		}
	})
}
