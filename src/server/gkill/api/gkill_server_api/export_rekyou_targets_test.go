package gkill_server_api

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// TestCollectExportReKyouTargets_ResolvesChainsAndTerminatesOnCycle は、
// ReKyou / MiReKyou の参照先を多段（最大 exportReKyouTargetMaxRounds 段）まで焼き込み、
// 循環参照や自己参照でも無限ループせずに終了することを確認する。
func TestCollectExportReKyouTargets_ResolvesChainsAndTerminatesOnCycle(t *testing.T) {
	_, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	ctx := context.Background()
	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	repositories, err := gkillAPI.GkillDAOManager.GetRepositories("admin", device)
	if err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}

	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// 1) 多段チェーン: miReKyou -> reKyou1 -> kmemo1
	if err := repositories.WriteKmemoRep.AddKmemoInfo(ctx, reps.Kmemo{
		ID:           "target-kmemo-1",
		Content:      "参照先のメモ",
		RelatedTime:  baseTime,
		CreateTime:   baseTime,
		CreateApp:    "test",
		CreateDevice: device,
		CreateUser:   "admin",
		UpdateTime:   baseTime,
		UpdateApp:    "test",
		UpdateDevice: device,
		UpdateUser:   "admin",
	}); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}
	if err := repositories.WriteReKyouRep.AddReKyouInfo(ctx, reps.ReKyou{
		ID:           "rekyou-mid",
		TargetID:     "target-kmemo-1",
		RelatedTime:  baseTime.Add(time.Minute),
		CreateTime:   baseTime.Add(time.Minute),
		CreateApp:    "test",
		CreateDevice: device,
		CreateUser:   "admin",
		UpdateTime:   baseTime.Add(time.Minute),
		UpdateApp:    "test",
		UpdateDevice: device,
		UpdateUser:   "admin",
	}); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}
	if err := repositories.WriteMiReKyouRep.AddMiReKyouInfo(ctx, reps.MiReKyou{
		ID:           "mirekyou-top",
		TargetID:     "rekyou-mid",
		BoardName:    "Inbox",
		CreateTime:   baseTime.Add(2 * time.Minute),
		CreateApp:    "test",
		CreateDevice: device,
		CreateUser:   "admin",
		UpdateTime:   baseTime.Add(2 * time.Minute),
		UpdateApp:    "test",
		UpdateDevice: device,
		UpdateUser:   "admin",
	}); err != nil {
		t.Fatalf("AddMiReKyouInfo failed: %v", err)
	}

	// 2) 循環参照: cycle-a <-> cycle-b
	if err := repositories.WriteReKyouRep.AddReKyouInfo(ctx, reps.ReKyou{
		ID:           "cycle-a",
		TargetID:     "cycle-b",
		RelatedTime:  baseTime.Add(3 * time.Minute),
		CreateTime:   baseTime.Add(3 * time.Minute),
		CreateApp:    "test",
		CreateDevice: device,
		CreateUser:   "admin",
		UpdateTime:   baseTime.Add(3 * time.Minute),
		UpdateApp:    "test",
		UpdateDevice: device,
		UpdateUser:   "admin",
	}); err != nil {
		t.Fatalf("AddReKyouInfo cycle-a failed: %v", err)
	}
	if err := repositories.WriteReKyouRep.AddReKyouInfo(ctx, reps.ReKyou{
		ID:           "cycle-b",
		TargetID:     "cycle-a",
		RelatedTime:  baseTime.Add(4 * time.Minute),
		CreateTime:   baseTime.Add(4 * time.Minute),
		CreateApp:    "test",
		CreateDevice: device,
		CreateUser:   "admin",
		UpdateTime:   baseTime.Add(4 * time.Minute),
		UpdateApp:    "test",
		UpdateDevice: device,
		UpdateUser:   "admin",
	}); err != nil {
		t.Fatalf("AddReKyouInfo cycle-b failed: %v", err)
	}

	if err := repositories.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}

	opts := &kyouBundleOptions{
		UserID:     "admin",
		Device:     device,
		LocaleName: "en",
		FindQuery:  &find.FindQuery{IDs: []string{"mirekyou-top", "cycle-a"}},
	}
	initialBundle, gkillErr := gkillAPI.collectKyouBundle(ctx, opts)
	if gkillErr != nil {
		t.Fatalf("collectKyouBundle failed: %+v", gkillErr)
	}

	merger := newExportMerger()
	merger.add(initialBundle)
	payload := minimalExportPayload("リポスト参照先テスト")
	merger.writeTo(payload)

	if gkillErr := gkillAPI.collectExportReKyouTargets(ctx, opts, merger, payload); gkillErr != nil {
		t.Fatalf("collectExportReKyouTargets failed: %+v", gkillErr)
	}

	foundIDs := map[string]bool{}
	for _, kyou := range payload.Kyous {
		foundIDs[kyou.ID] = true
	}
	for _, wantID := range []string{"mirekyou-top", "rekyou-mid", "target-kmemo-1", "cycle-a", "cycle-b"} {
		if !foundIDs[wantID] {
			t.Errorf("参照先 %q が焼き込みに含まれていない (got: %v)", wantID, foundIDs)
		}
	}
	foundKmemo := false
	for _, kmemo := range payload.Kmemos {
		if kmemo.ID == "target-kmemo-1" && kmemo.Content == "参照先のメモ" {
			foundKmemo = true
		}
	}
	if !foundKmemo {
		t.Error("2段先の Kmemo 実体が焼き込みに含まれていない")
	}
}

// TestCollectExportReKyouTargets_StopsAfterMaxRounds は、リポストを辿るのが
// exportReKyouTargetMaxRounds 段で止まることを確認する（循環は known で止まるので、上限そのものは
// 輪にならない長い連鎖でしか試せない）。
//
// 連鎖: deep-0（列に出る）→ deep-1 → deep-2 → deep-3 → deep-kmemo。3段で deep-3 までが焼かれ、
// 4段先の deep-kmemo は焼かれない（出力先ではその行が参照先なしで出る）。
func TestCollectExportReKyouTargets_StopsAfterMaxRounds(t *testing.T) {
	_, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	ctx := context.Background()
	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	repositories, err := gkillAPI.GkillDAOManager.GetRepositories("admin", device)
	if err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}

	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	if err := repositories.WriteKmemoRep.AddKmemoInfo(ctx, reps.Kmemo{
		ID: "deep-kmemo", Content: "4段先のメモ", RelatedTime: baseTime,
		CreateTime: baseTime, CreateApp: "test", CreateDevice: device, CreateUser: "admin",
		UpdateTime: baseTime, UpdateApp: "test", UpdateDevice: device, UpdateUser: "admin",
	}); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}
	chain := []struct{ id, target string }{
		{"deep-3", "deep-kmemo"}, {"deep-2", "deep-3"}, {"deep-1", "deep-2"}, {"deep-0", "deep-1"},
	}
	for i, link := range chain {
		at := baseTime.Add(time.Duration(i+1) * time.Minute)
		if err := repositories.WriteReKyouRep.AddReKyouInfo(ctx, reps.ReKyou{
			ID: link.id, TargetID: link.target, RelatedTime: at,
			CreateTime: at, CreateApp: "test", CreateDevice: device, CreateUser: "admin",
			UpdateTime: at, UpdateApp: "test", UpdateDevice: device, UpdateUser: "admin",
		}); err != nil {
			t.Fatalf("AddReKyouInfo %s failed: %v", link.id, err)
		}
	}
	if err := repositories.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}

	opts := &kyouBundleOptions{
		UserID:     "admin",
		Device:     device,
		LocaleName: "en",
		FindQuery:  &find.FindQuery{IDs: []string{"deep-0"}},
	}
	initialBundle, gkillErr := gkillAPI.collectKyouBundle(ctx, opts)
	if gkillErr != nil {
		t.Fatalf("collectKyouBundle failed: %+v", gkillErr)
	}
	// 型別データは列に出た記録のぶんだけ（FindReKyou は全件返すので、収集側で絞っている）
	if len(initialBundle.ReKyous) != 1 || initialBundle.ReKyous[0].ID != "deep-0" {
		t.Fatalf("列に出ていないリポストまで型別データに入っている: %+v", initialBundle.ReKyous)
	}
	merger := newExportMerger()
	merger.add(initialBundle)
	payload := minimalExportPayload("リポストの段数の上限")
	merger.writeTo(payload)

	if gkillErr := gkillAPI.collectExportReKyouTargets(ctx, opts, merger, payload); gkillErr != nil {
		t.Fatalf("collectExportReKyouTargets failed: %+v", gkillErr)
	}

	foundIDs := map[string]bool{}
	for _, kyou := range payload.Kyous {
		foundIDs[kyou.ID] = true
	}
	for _, wantID := range []string{"deep-0", "deep-1", "deep-2", "deep-3"} {
		if !foundIDs[wantID] {
			t.Errorf("%d 段以内の %q が焼かれていない (got: %v)", exportReKyouTargetMaxRounds, wantID, foundIDs)
		}
	}
	if foundIDs["deep-kmemo"] {
		t.Errorf("%d 段を超えた先まで辿っている（上限が効いていない）", exportReKyouTargetMaxRounds)
	}
}
