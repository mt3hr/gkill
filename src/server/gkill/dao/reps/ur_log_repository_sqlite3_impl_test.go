package reps

import (
	"context"
	"testing"
	"time"
)

func makeURLog(id, url, title string) URLog {
	now := testTime()
	return URLog{
		IsDeleted:    false,
		ID:           id,
		URL:          url,
		Title:        title,
		DataType:     "urlog",
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

func TestURLogAddAndGet(t *testing.T) {
	repo := newTempURLogRepo(t)
	ctx := context.Background()

	urlog := makeURLog("urlog-001", "https://example.com", "Example Site")
	if err := repo.AddURLogInfo(ctx, urlog); err != nil {
		t.Fatalf("AddURLogInfo failed: %v", err)
	}

	got, err := repo.GetURLog(ctx, "urlog-001", nil)
	if err != nil {
		t.Fatalf("GetURLog failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetURLog returned nil")
	}
	if got.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", got.URL, "https://example.com")
	}
	if got.Title != "Example Site" {
		t.Errorf("Title = %q, want %q", got.Title, "Example Site")
	}
}

func TestURLogFindKyous(t *testing.T) {
	repo := newTempURLogRepo(t)
	ctx := context.Background()

	u1 := makeURLog("urlog-a", "https://a.example.com", "Site A")
	u2 := makeURLog("urlog-b", "https://b.example.com", "Site B")
	u2.UpdateTime = u2.UpdateTime.Add(time.Second)

	if err := repo.AddURLogInfo(ctx, u1); err != nil {
		t.Fatalf("AddURLogInfo failed: %v", err)
	}
	if err := repo.AddURLogInfo(ctx, u2); err != nil {
		t.Fatalf("AddURLogInfo failed: %v", err)
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

func TestURLogGetHistories(t *testing.T) {
	repo := newTempURLogRepo(t)
	ctx := context.Background()

	u1 := makeURLog("urlog-hist", "https://example.com", "v1")
	if err := repo.AddURLogInfo(ctx, u1); err != nil {
		t.Fatalf("AddURLogInfo failed: %v", err)
	}

	u2 := makeURLog("urlog-hist", "https://example.com/updated", "v2")
	u2.UpdateTime = u2.UpdateTime.Add(time.Hour)
	if err := repo.AddURLogInfo(ctx, u2); err != nil {
		t.Fatalf("AddURLogInfo failed: %v", err)
	}

	histories, err := repo.GetURLogHistories(ctx, "urlog-hist")
	if err != nil {
		t.Fatalf("GetURLogHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}
}
