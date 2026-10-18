package reps

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

func makeReKyou(id, targetID string) ReKyou {
	now := testTime()
	return ReKyou{IsDeleted: false, ID: id, TargetID: targetID, DataType: "re_kyou", RelatedTime: now,
		CreateTime: now, CreateApp: "test_app", CreateDevice: "test_device", CreateUser: "test_user",
		UpdateTime: now, UpdateApp: "test_app", UpdateUser: "test_user", UpdateDevice: "test_device"}
}

// makeReKyouWithDistinctAuditFields は作成・更新の欄6つを全部違う値にしたリポストを作る。
// makeReKyou は作成と更新で同じ値を使うので、作成と更新の取り違えまでは検出できない。
func makeReKyouWithDistinctAuditFields(id, targetID string) ReKyou {
	rk := makeReKyou(id, targetID)
	rk.CreateApp, rk.CreateDevice, rk.CreateUser = "create_app", "create_device", "create_user"
	rk.UpdateApp, rk.UpdateDevice, rk.UpdateUser = "update_app", "update_device", "update_user"
	return rk
}

// assertReKyouAuditFieldsRoundTripOnAllReads はリポストを1件足し、読み取り口7つすべてで
// 作成・更新の欄6つが入れ替わらずに戻ることを確かめる（assertAuditFieldsRoundTrip のリポスト版）。
//
// ReKyou の実 rep とキャッシュ版は GetReKyou / GetReKyouHistories / GetReKyousAllLatest の SELECT が
// CREATE_USER, CREATE_DEVICE の順なのに rows.Scan が (CreateDevice, CreateUser) の順で、読むたびに作成者と
// 作成端末が入れ替わっていた（FindReKyou・FindKyous は GetReKyousAllLatest を通るので同じく入れ替わる。
// GetKyou / GetKyouHistories は揃っていた）。読んだ値をそのまま新しい版として書き戻す経路
// （画面のリポストの編集・連鎖削除・MCP の更新/削除/復元）では入れ替わりが保存される。
func assertReKyouAuditFieldsRoundTripOnAllReads(t *testing.T, repo ReKyouRepository) {
	t.Helper()
	ctx := context.Background()

	rk := makeReKyouWithDistinctAuditFields("rekyou-audit", "target-audit")
	if err := repo.AddReKyouInfo(ctx, rk); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}
	want := auditFieldsOf(rk)

	assertOne := func(t *testing.T, entities []any, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if len(entities) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entities))
		}
		assertAuditFieldsRoundTrip(t, want, auditFieldsOf(entities[0]))
	}
	reKyousAsAny := func(list []ReKyou) []any {
		out := []any{}
		for _, x := range list {
			out = append(out, x)
		}
		return out
	}
	kyousAsAny := func(list []Kyou) []any {
		out := []any{}
		for _, x := range list {
			out = append(out, x)
		}
		return out
	}

	t.Run("GetReKyou", func(t *testing.T) {
		got, err := repo.GetReKyou(ctx, rk.ID, nil)
		if err != nil {
			t.Fatalf("GetReKyou failed: %v", err)
		}
		if got == nil {
			t.Fatal("GetReKyou returned nil")
		}
		assertAuditFieldsRoundTrip(t, want, auditFieldsOf(got))
	})
	t.Run("GetReKyouHistories", func(t *testing.T) {
		got, err := repo.GetReKyouHistories(ctx, rk.ID)
		assertOne(t, reKyousAsAny(got), err)
	})
	t.Run("GetReKyousAllLatest", func(t *testing.T) {
		got, err := repo.GetReKyousAllLatest(ctx)
		assertOne(t, reKyousAsAny(got), err)
	})
	t.Run("FindReKyou", func(t *testing.T) {
		got, err := repo.FindReKyou(ctx, &find.FindQuery{})
		assertOne(t, reKyousAsAny(got), err)
	})
	t.Run("FindKyous", func(t *testing.T) {
		got, err := repo.FindKyous(ctx, &find.FindQuery{})
		assertOne(t, kyousAsAny(got[rk.ID]), err)
	})
	t.Run("GetKyou", func(t *testing.T) {
		got, err := repo.GetKyou(ctx, rk.ID, nil)
		if err != nil {
			t.Fatalf("GetKyou failed: %v", err)
		}
		if got == nil {
			t.Fatal("GetKyou returned nil")
		}
		assertAuditFieldsRoundTrip(t, want, auditFieldsOf(got))
	})
	t.Run("GetKyouHistories", func(t *testing.T) {
		got, err := repo.GetKyouHistories(ctx, rk.ID)
		assertOne(t, kyousAsAny(got), err)
	})
}

func TestReKyouAuditFieldsRoundTrip(t *testing.T) {
	assertReKyouAuditFieldsRoundTripOnAllReads(t, newTempReKyouRepo(t, nil))
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

// Note: with a nil GkillRepositories reference, FindKyous and FindReKyou skip target resolution
// and pass every ReKyou through (assertReKyouAuditFieldsRoundTripOnAllReads relies on this).
// Find tests with target resolution are covered in API integration tests.

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
