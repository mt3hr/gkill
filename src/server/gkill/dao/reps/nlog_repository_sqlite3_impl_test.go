package reps

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func makeNlog(id, title, shop string, amount float64) Nlog {
	now := testTime()
	return Nlog{
		IsDeleted:    false,
		ID:           id,
		Title:        title,
		Shop:         shop,
		Amount:       json.Number(fmt.Sprintf("%g", amount)),
		DataType:     "nlog",
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

func TestNlogAddAndGet(t *testing.T) {
	repo := newTempNlogRepo(t)
	ctx := context.Background()

	nlog := makeNlog("nlog-001", "おにぎり", "コンビニ", 500)
	if err := repo.AddNlogInfo(ctx, nlog); err != nil {
		t.Fatalf("AddNlogInfo failed: %v", err)
	}

	got, err := repo.GetNlog(ctx, "nlog-001", nil)
	if err != nil {
		t.Fatalf("GetNlog failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetNlog returned nil")
	}
	if got.Shop != "コンビニ" {
		t.Errorf("Shop = %q, want %q", got.Shop, "コンビニ")
	}
	if got.Title != "おにぎり" {
		t.Errorf("Title = %q, want %q", got.Title, "おにぎり")
	}
	if got.Amount != json.Number("500") {
		t.Errorf("Amount = %v, want %v", got.Amount, json.Number("500"))
	}
}

func TestNlogFindKyous(t *testing.T) {
	repo := newTempNlogRepo(t)
	ctx := context.Background()

	n1 := makeNlog("nlog-a", "ランチ", "定食屋", 800)
	n2 := makeNlog("nlog-b", "コーヒー", "カフェ", 350)
	n2.UpdateTime = n2.UpdateTime.Add(time.Second)

	if err := repo.AddNlogInfo(ctx, n1); err != nil {
		t.Fatalf("AddNlogInfo failed: %v", err)
	}
	if err := repo.AddNlogInfo(ctx, n2); err != nil {
		t.Fatalf("AddNlogInfo failed: %v", err)
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

func TestNlogGetHistories(t *testing.T) {
	repo := newTempNlogRepo(t)
	ctx := context.Background()

	n1 := makeNlog("nlog-hist", "弁当", "スーパー", 400)
	if err := repo.AddNlogInfo(ctx, n1); err != nil {
		t.Fatalf("AddNlogInfo failed: %v", err)
	}

	n2 := makeNlog("nlog-hist", "弁当", "スーパー", 450)
	n2.UpdateTime = n2.UpdateTime.Add(time.Hour)
	if err := repo.AddNlogInfo(ctx, n2); err != nil {
		t.Fatalf("AddNlogInfo failed: %v", err)
	}

	histories, err := repo.GetNlogHistories(ctx, "nlog-hist")
	if err != nil {
		t.Fatalf("GetNlogHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}
}
