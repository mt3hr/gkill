package gkill_server_api

import (
	"context"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// cursor 頁の Mi 再検証（ADR-0621）が失敗したとき、頁を落とさず warnings を1行足して batch をそのまま返すこと。
// 再検証は「返却済みの Mi が別の射影で再出現する」のを防ぐ二次的な処理なので、失敗で頁全体を
// 空にすると、正しい記録まで消える（利用者には「途中で件数が減った」と見える）。
func TestRevalidateMiEntriesAgainstOriginalWindow_KeepsBatchAndWarnsWhenTheSearchFails(t *testing.T) {
	_, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	// 取り消し済みの context で検索を失敗させる
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	batch := []reps.Kyou{
		{ID: "mi-1", DataType: "mi_create"},
		{ID: "mi-1", DataType: "mi_check"},
		{ID: "kmemo-1", DataType: "kmemo"},
	}
	got, warnings := gkillAPI.revalidateMiEntriesAgainstOriginalWindow(ctx, "admin", "test-device", &find.FindQuery{}, batch)
	if len(got) != len(batch) {
		t.Fatalf("失敗時に batch が変わった: %d 件, want %d 件", len(got), len(batch))
	}
	for i := range batch {
		if got[i].ID != batch[i].ID || got[i].DataType != batch[i].DataType {
			t.Errorf("batch[%d] = %+v, want %+v", i, got[i], batch[i])
		}
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want 1件", warnings)
	}
	// 同じ ID の射影2本は1タスクとして数える
	if !strings.Contains(warnings[0], "could not re-validate 1 task entries") {
		t.Errorf("warning = %q, want 「1 task entries を再検証できなかった」", warnings[0])
	}
}

// 窓に依存する種別（Mi / MiReKyou）が無い頁は検索せずそのまま返す（warnings も無し）。
func TestRevalidateMiEntriesAgainstOriginalWindow_SkipsPagesWithoutTasks(t *testing.T) {
	_, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 検索が走れば失敗する context。走らないことの証明に使う

	batch := []reps.Kyou{{ID: "kmemo-1", DataType: "kmemo"}, {ID: "timeis-1", DataType: "timeis_start"}}
	got, warnings := gkillAPI.revalidateMiEntriesAgainstOriginalWindow(ctx, "admin", "test-device", &find.FindQuery{}, batch)
	if len(got) != 2 || len(warnings) != 0 {
		t.Fatalf("got %d 件 / warnings %v, want 2 件 / なし", len(got), warnings)
	}
}
