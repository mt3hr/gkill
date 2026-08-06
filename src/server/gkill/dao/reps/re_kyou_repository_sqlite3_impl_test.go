package reps

import (
	"context"
	"testing"
	"time"
)

func makeReKyou(id, targetID string) ReKyou {
	now := testTime()
	return ReKyou{IsDeleted: false, ID: id, TargetID: targetID, DataType: "re_kyou", RelatedTime: now,
		CreateTime: now, CreateApp: "test_app", CreateDevice: "test_device", CreateUser: "test_user",
		UpdateTime: now, UpdateApp: "test_app", UpdateUser: "test_user", UpdateDevice: "test_device"}
}

func TestReKyouAddAndGet(t *testing.T) {
	repo := newTempReKyouRepo(t, nil)
	ctx := context.Background()

	rk := makeReKyou("rekyou-001", "target-001")
	if err := repo.AddReKyouInfo(ctx, rk); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}

	got, err := repo.GetReKyou(ctx, "rekyou-001", nil)
	if err != nil {
		t.Fatalf("GetReKyou failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetReKyou returned nil")
	}
	if got.ID != "rekyou-001" {
		t.Errorf("ID = %q, want %q", got.ID, "rekyou-001")
	}
	if got.TargetID != "target-001" {
		t.Errorf("TargetID = %q, want %q", got.TargetID, "target-001")
	}
}

// Note: FindKyous and FindReKyou require a non-nil GkillRepositories reference
// (they call GetRepositoriesWithoutReKyouRep internally).
// Find tests are covered in API integration tests.

func TestReKyouGetHistories(t *testing.T) {
	repo := newTempReKyouRepo(t, nil)
	ctx := context.Background()

	rk1 := makeReKyou("rekyou-hist", "target-hist")
	if err := repo.AddReKyouInfo(ctx, rk1); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}

	rk2 := makeReKyou("rekyou-hist", "target-hist")
	rk2.UpdateTime = rk2.UpdateTime.Add(time.Hour)
	if err := repo.AddReKyouInfo(ctx, rk2); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}

	histories, err := repo.GetReKyouHistories(ctx, "rekyou-hist")
	if err != nil {
		t.Fatalf("GetReKyouHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}
}

func TestReKyouGetReKyousByTargetID(t *testing.T) {
	ctx := context.Background()
	rep1 := newTempReKyouRepo(t, nil)
	rep2 := newTempReKyouRepo(t, nil)
	rekyouReps := ReKyouRepositories{ReKyouRepositories: []ReKyouRepository{rep1, rep2}}

	// tgt-Aを参照するもの2件と、tgt-Bを参照するもの1件
	if err := rep1.AddReKyouInfo(ctx, makeReKyou("rk-1", "tgt-A")); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}
	if err := rep2.AddReKyouInfo(ctx, makeReKyou("rk-2", "tgt-A")); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}
	if err := rep2.AddReKyouInfo(ctx, makeReKyou("rk-3", "tgt-B")); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}

	matchReKyous, err := rekyouReps.GetReKyousByTargetID(ctx, "tgt-A")
	if err != nil {
		t.Fatalf("GetReKyousByTargetID failed: %v", err)
	}
	if len(matchReKyous) != 2 {
		t.Fatalf("expected 2 rekyous for tgt-A, got %d", len(matchReKyous))
	}

	// 存在しないtarget_idでは空スライス（nilではない）
	noMatch, err := rekyouReps.GetReKyousByTargetID(ctx, "tgt-unknown")
	if err != nil {
		t.Fatalf("GetReKyousByTargetID failed: %v", err)
	}
	if noMatch == nil {
		t.Fatal("GetReKyousByTargetID returned nil, want empty slice")
	}
	if len(noMatch) != 0 {
		t.Errorf("expected 0 rekyous for unknown target, got %d", len(noMatch))
	}
}

func TestReKyouGetReKyousByTargetIDExcludesDeleted(t *testing.T) {
	ctx := context.Background()
	rep1 := newTempReKyouRepo(t, nil)
	rep2 := newTempReKyouRepo(t, nil)
	rekyouReps := ReKyouRepositories{ReKyouRepositories: []ReKyouRepository{rep1, rep2}}

	if err := rep1.AddReKyouInfo(ctx, makeReKyou("rk-1", "tgt-A")); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}
	if err := rep2.AddReKyouInfo(ctx, makeReKyou("rk-2", "tgt-A")); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}

	// rk-2を論理削除する（追記型なので新しいUpdateTimeで積む）
	deletedReKyou := makeReKyou("rk-2", "tgt-A")
	deletedReKyou.IsDeleted = true
	deletedReKyou.UpdateTime = deletedReKyou.UpdateTime.Add(time.Hour)
	if err := rep2.AddReKyouInfo(ctx, deletedReKyou); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}

	matchReKyous, err := rekyouReps.GetReKyousByTargetID(ctx, "tgt-A")
	if err != nil {
		t.Fatalf("GetReKyousByTargetID failed: %v", err)
	}
	if len(matchReKyous) != 1 {
		t.Fatalf("expected 1 rekyou after deleting rk-2, got %d", len(matchReKyous))
	}
	if matchReKyous[0].ID != "rk-1" {
		t.Errorf("ID = %q, want %q", matchReKyous[0].ID, "rk-1")
	}
}
