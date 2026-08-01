package reps

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// newTempMiReKyouRepo creates a MiReKyouRepository backed by a temp SQLite3 file.
// GkillRepositoriesはnilを渡す。ターゲット解決はリポジトリ横断の処理なので、
// 単体テストではスキップされる（すべて通る）。
func newTempMiReKyouRepo(t *testing.T) MiReKyouRepository {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewMiReKyouRepositorySQLite3Impl(context.Background(), filepath.Join(dir, "mirekyou.db"), true, nil)
	if err != nil {
		t.Fatalf("failed to create mirekyou repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

func makeMiReKyou(id, targetID string) MiReKyou {
	now := testTime()
	return MiReKyou{
		IsDeleted:    false,
		ID:           id,
		TargetID:     targetID,
		DataType:     "mirekyou_create",
		IsChecked:    false,
		BoardName:    "default",
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

func TestMiReKyouAddAndGet(t *testing.T) {
	repo := newTempMiReKyouRepo(t)
	ctx := context.Background()

	mirekyou := makeMiReKyou("mirekyou-001", "target-001")
	mirekyou.IsChecked = true
	mirekyou.BoardName = "work"
	if err := repo.AddMiReKyouInfo(ctx, mirekyou); err != nil {
		t.Fatalf("AddMiReKyouInfo failed: %v", err)
	}

	got, err := repo.GetMiReKyou(ctx, "mirekyou-001", nil)
	if err != nil {
		t.Fatalf("GetMiReKyou failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetMiReKyou returned nil")
	}
	if got.TargetID != "target-001" {
		t.Errorf("TargetID = %q, want %q", got.TargetID, "target-001")
	}
	if got.IsChecked != true {
		t.Errorf("IsChecked = %v, want true", got.IsChecked)
	}
	if got.BoardName != "work" {
		t.Errorf("BoardName = %q, want %q", got.BoardName, "work")
	}
}

func TestMiReKyouAddRejectsEmptyTargetID(t *testing.T) {
	repo := newTempMiReKyouRepo(t)
	ctx := context.Background()

	mirekyou := makeMiReKyou("mirekyou-notarget", "")
	if err := repo.AddMiReKyouInfo(ctx, mirekyou); err == nil {
		t.Error("AddMiReKyouInfo with empty target id should fail")
	}
}

func TestMiReKyouFindByBoard(t *testing.T) {
	repo := newTempMiReKyouRepo(t)
	ctx := context.Background()

	m1 := makeMiReKyou("mirekyou-w1", "target-w1")
	m1.BoardName = "work"
	m1.UpdateTime = m1.UpdateTime.Add(1 * time.Second)
	if err := repo.AddMiReKyouInfo(ctx, m1); err != nil {
		t.Fatalf("AddMiReKyouInfo failed: %v", err)
	}

	m2 := makeMiReKyou("mirekyou-w2", "target-w2")
	m2.BoardName = "work"
	m2.UpdateTime = m2.UpdateTime.Add(2 * time.Second)
	if err := repo.AddMiReKyouInfo(ctx, m2); err != nil {
		t.Fatalf("AddMiReKyouInfo failed: %v", err)
	}

	m3 := makeMiReKyou("mirekyou-p1", "target-p1")
	m3.BoardName = "personal"
	m3.UpdateTime = m3.UpdateTime.Add(3 * time.Second)
	if err := repo.AddMiReKyouInfo(ctx, m3); err != nil {
		t.Fatalf("AddMiReKyouInfo failed: %v", err)
	}

	query := makeDefaultFindQuery()
	query.UseMiBoardName = true
	query.MiBoardName = "work"

	mirekyous, err := repo.FindMiReKyou(ctx, query)
	if err != nil {
		t.Fatalf("FindMiReKyou failed: %v", err)
	}
	workIDs := map[string]bool{}
	for _, m := range mirekyous {
		workIDs[m.ID] = true
	}
	if len(workIDs) != 2 {
		t.Errorf("expected 2 unique MiReKyou IDs with board 'work', got %d", len(workIDs))
	}
	if workIDs["mirekyou-p1"] {
		t.Error("board 'personal' の MiReKyou が board 'work' の検索に混ざっている")
	}
}

// TestMiReKyouFindKyousProjections はMiと同じ5射影のKyouが返ることを確認する。
func TestMiReKyouFindKyousProjections(t *testing.T) {
	repo := newTempMiReKyouRepo(t)
	ctx := context.Background()

	now := testTime()
	limitTime := now.Add(24 * time.Hour)
	startTime := now.Add(1 * time.Hour)
	endTime := now.Add(2 * time.Hour)

	mirekyou := makeMiReKyou("mirekyou-proj", "target-proj")
	mirekyou.LimitTime = &limitTime
	mirekyou.EstimateStartTime = &startTime
	mirekyou.EstimateEndTime = &endTime
	if err := repo.AddMiReKyouInfo(ctx, mirekyou); err != nil {
		t.Fatalf("AddMiReKyouInfo failed: %v", err)
	}

	kyous, err := repo.FindKyous(ctx, makeDefaultFindQuery())
	if err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}

	dataTypes := map[string]bool{}
	for _, kyousInID := range kyous {
		for _, kyou := range kyousInID {
			dataTypes[kyou.DataType] = true
		}
	}
	for _, want := range []string{"mirekyou_create", "mirekyou_check", "mirekyou_limit", "mirekyou_start", "mirekyou_end"} {
		if !dataTypes[want] {
			t.Errorf("expected data type %q in FindKyous result, got %v", want, dataTypes)
		}
	}
}

// TestMiReKyouFindKyousWithoutOptionalTimes は日時未設定の射影が出ないことを確認する。
func TestMiReKyouFindKyousWithoutOptionalTimes(t *testing.T) {
	repo := newTempMiReKyouRepo(t)
	ctx := context.Background()

	mirekyou := makeMiReKyou("mirekyou-notime", "target-notime")
	if err := repo.AddMiReKyouInfo(ctx, mirekyou); err != nil {
		t.Fatalf("AddMiReKyouInfo failed: %v", err)
	}

	kyous, err := repo.FindKyous(ctx, makeDefaultFindQuery())
	if err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}

	dataTypes := map[string]bool{}
	for _, kyousInID := range kyous {
		for _, kyou := range kyousInID {
			dataTypes[kyou.DataType] = true
		}
	}
	if !dataTypes["mirekyou_create"] {
		t.Errorf("expected mirekyou_create, got %v", dataTypes)
	}
	for _, notWant := range []string{"mirekyou_limit", "mirekyou_start", "mirekyou_end"} {
		if dataTypes[notWant] {
			t.Errorf("日時未設定なのに %q が返っている", notWant)
		}
	}
}

func TestMiReKyouGetBoardNames(t *testing.T) {
	repo := newTempMiReKyouRepo(t)
	ctx := context.Background()

	m1 := makeMiReKyou("mirekyou-bn1", "target-bn1")
	m1.BoardName = "work"
	if err := repo.AddMiReKyouInfo(ctx, m1); err != nil {
		t.Fatalf("AddMiReKyouInfo failed: %v", err)
	}

	m2 := makeMiReKyou("mirekyou-bn2", "target-bn2")
	m2.BoardName = "personal"
	m2.UpdateTime = m2.UpdateTime.Add(1 * time.Second)
	if err := repo.AddMiReKyouInfo(ctx, m2); err != nil {
		t.Fatalf("AddMiReKyouInfo failed: %v", err)
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

func TestMiReKyouGetHistories(t *testing.T) {
	repo := newTempMiReKyouRepo(t)
	ctx := context.Background()

	m1 := makeMiReKyou("mirekyou-hist", "target-hist")
	if err := repo.AddMiReKyouInfo(ctx, m1); err != nil {
		t.Fatalf("AddMiReKyouInfo failed: %v", err)
	}

	m2 := makeMiReKyou("mirekyou-hist", "target-hist")
	m2.BoardName = "updated"
	m2.UpdateTime = m2.UpdateTime.Add(time.Hour)
	if err := repo.AddMiReKyouInfo(ctx, m2); err != nil {
		t.Fatalf("AddMiReKyouInfo failed: %v", err)
	}

	histories, err := repo.GetMiReKyouHistories(ctx, "mirekyou-hist")
	if err != nil {
		t.Fatalf("GetMiReKyouHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}

	// 最新版のみを取得すると1件になる
	latest, err := repo.GetMiReKyou(ctx, "mirekyou-hist", nil)
	if err != nil {
		t.Fatalf("GetMiReKyou failed: %v", err)
	}
	if latest == nil || latest.BoardName != "updated" {
		t.Errorf("GetMiReKyou should return the latest version, got %+v", latest)
	}
}

// TestMiReKyouGetLatestDataRepositoryAddress はTargetIDInDataにリポスト対象が入ることを確認する。
func TestMiReKyouGetLatestDataRepositoryAddress(t *testing.T) {
	repo := newTempMiReKyouRepo(t)
	ctx := context.Background()

	mirekyou := makeMiReKyou("mirekyou-addr", "target-addr")
	if err := repo.AddMiReKyouInfo(ctx, mirekyou); err != nil {
		t.Fatalf("AddMiReKyouInfo failed: %v", err)
	}

	addrs, err := repo.GetLatestDataRepositoryAddress(ctx, false)
	if err != nil {
		t.Fatalf("GetLatestDataRepositoryAddress failed: %v", err)
	}
	if len(addrs) != 1 {
		t.Fatalf("expected 1 address, got %d", len(addrs))
	}
	if addrs[0].TargetID != "mirekyou-addr" {
		t.Errorf("TargetID = %q, want %q", addrs[0].TargetID, "mirekyou-addr")
	}
	if addrs[0].TargetIDInData == nil || *addrs[0].TargetIDInData != "target-addr" {
		t.Errorf("TargetIDInData = %v, want %q", addrs[0].TargetIDInData, "target-addr")
	}
}

// TestMiReKyouToMi はMi検索パイプラインへ渡すための変換を確認する。
func TestMiReKyouToMi(t *testing.T) {
	now := testTime()
	limitTime := now.Add(24 * time.Hour)

	mirekyou := makeMiReKyou("mirekyou-tomi", "target-tomi")
	mirekyou.BoardName = "work"
	mirekyou.IsChecked = true
	mirekyou.LimitTime = &limitTime

	mi := mirekyou.ToMi()
	if mi.ID != mirekyou.ID {
		t.Errorf("ID = %q, want %q", mi.ID, mirekyou.ID)
	}
	if mi.Title != "" {
		t.Errorf("Title = %q, want empty (MiReKyouはタイトルを持たない)", mi.Title)
	}
	if mi.BoardName != "work" {
		t.Errorf("BoardName = %q, want %q", mi.BoardName, "work")
	}
	if !mi.IsChecked {
		t.Error("IsChecked = false, want true")
	}
	if mi.LimitTime == nil || !mi.LimitTime.Equal(limitTime) {
		t.Errorf("LimitTime = %v, want %v", mi.LimitTime, limitTime)
	}
}
