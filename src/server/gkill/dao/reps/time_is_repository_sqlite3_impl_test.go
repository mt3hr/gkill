package reps

import (
	"context"
	"testing"
	"time"
)

func makeTimeIs(id, title string) TimeIs {
	now := testTime()
	return TimeIs{
		IsDeleted:    false,
		ID:           id,
		Title:        title,
		DataType:     "timeis",
		StartTime:    now,
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

func TestTimeIsAddAndGet(t *testing.T) {
	repo := newTempTimeIsRepo(t)
	ctx := context.Background()

	ti := makeTimeIs("timeis-001", "作業A")
	if err := repo.AddTimeIsInfo(ctx, ti); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	got, err := repo.GetTimeIs(ctx, "timeis-001", nil)
	if err != nil {
		t.Fatalf("GetTimeIs failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetTimeIs returned nil")
	}
	if got.Title != "作業A" {
		t.Errorf("Title = %q, want %q", got.Title, "作業A")
	}
	if !got.StartTime.Equal(ti.StartTime) {
		t.Errorf("StartTime = %v, want %v", got.StartTime, ti.StartTime)
	}
}

func TestTimeIsFindTimeIs(t *testing.T) {
	repo := newTempTimeIsRepo(t)
	ctx := context.Background()

	ti1 := makeTimeIs("timeis-f1", "作業1")
	ti1.UpdateTime = ti1.UpdateTime.Add(1 * time.Second)
	if err := repo.AddTimeIsInfo(ctx, ti1); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	ti2 := makeTimeIs("timeis-f2", "作業2")
	ti2.UpdateTime = ti2.UpdateTime.Add(2 * time.Second)
	if err := repo.AddTimeIsInfo(ctx, ti2); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	query := makeDefaultFindQuery()
	timeIss, err := repo.FindTimeIs(ctx, query)
	if err != nil {
		t.Fatalf("FindTimeIs failed: %v", err)
	}
	if len(timeIss) != 2 {
		t.Errorf("expected 2 TimeIs entries, got %d", len(timeIss))
	}
}

func TestTimeIsGetHistories(t *testing.T) {
	repo := newTempTimeIsRepo(t)
	ctx := context.Background()

	// Add first version
	ti1 := makeTimeIs("timeis-hist", "初版作業")
	if err := repo.AddTimeIsInfo(ctx, ti1); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	// Add second version with different UpdateTime
	ti2 := makeTimeIs("timeis-hist", "改訂版作業")
	ti2.UpdateTime = ti2.UpdateTime.Add(time.Hour)
	if err := repo.AddTimeIsInfo(ctx, ti2); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	histories, err := repo.GetTimeIsHistories(ctx, "timeis-hist")
	if err != nil {
		t.Fatalf("GetTimeIsHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}
}
