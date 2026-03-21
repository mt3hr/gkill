package reps

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func makeKC(id, title string, value float64) KC {
	now := testTime()
	return KC{
		IsDeleted:    false,
		ID:           id,
		Title:        title,
		NumValue:     json.Number(fmt.Sprintf("%g", value)),
		DataType:     "kc",
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

func TestKCAddAndGet(t *testing.T) {
	repo := newTempKCRepo(t)
	ctx := context.Background()

	kc := makeKC("kc-001", "体重", 42.5)
	if err := repo.AddKCInfo(ctx, kc); err != nil {
		t.Fatalf("AddKCInfo failed: %v", err)
	}

	got, err := repo.GetKC(ctx, "kc-001", nil)
	if err != nil {
		t.Fatalf("GetKC failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetKC returned nil")
	}
	if got.Title != "体重" {
		t.Errorf("Title = %q, want %q", got.Title, "体重")
	}
	if got.NumValue != json.Number("42.5") {
		t.Errorf("NumValue = %v, want %v", got.NumValue, json.Number("42.5"))
	}
}

func TestKCFindKyous(t *testing.T) {
	repo := newTempKCRepo(t)
	ctx := context.Background()

	kc1 := makeKC("kc-a", "歩数", 10000)
	kc2 := makeKC("kc-b", "体温", 36.5)
	kc2.UpdateTime = kc2.UpdateTime.Add(time.Second)

	if err := repo.AddKCInfo(ctx, kc1); err != nil {
		t.Fatalf("AddKCInfo failed: %v", err)
	}
	if err := repo.AddKCInfo(ctx, kc2); err != nil {
		t.Fatalf("AddKCInfo failed: %v", err)
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

func TestKCGetHistories(t *testing.T) {
	repo := newTempKCRepo(t)
	ctx := context.Background()

	kc1 := makeKC("kc-hist", "血圧", 120)
	if err := repo.AddKCInfo(ctx, kc1); err != nil {
		t.Fatalf("AddKCInfo failed: %v", err)
	}

	kc2 := makeKC("kc-hist", "血圧", 115)
	kc2.UpdateTime = kc2.UpdateTime.Add(time.Hour)
	if err := repo.AddKCInfo(ctx, kc2); err != nil {
		t.Fatalf("AddKCInfo failed: %v", err)
	}

	histories, err := repo.GetKCHistories(ctx, "kc-hist")
	if err != nil {
		t.Fatalf("GetKCHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}
}
