package kftl

import (
	"context"
	"testing"
	"time"
)

// mockRequest is a minimal KFTLRequest implementation for testing.
type mockRequest struct {
	KFTLRequestBase
}

func (r *mockRequest) DoRequest(_ context.Context) error { return nil }

func newMockRequest(id string) *mockRequest {
	return &mockRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID: id,
			Ctx:       &KFTLStatementLineContext{},
		},
	}
}

func TestRequestMap_Empty(t *testing.T) {
	m := NewKFTLRequestMap()
	all := m.All()
	if len(all) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(all))
	}

	_, ok := m.Get("nonexistent")
	if ok {
		t.Fatal("expected Get on empty map to return false")
	}
}

func TestRequestMap_AddAndAll(t *testing.T) {
	m := NewKFTLRequestMap()

	r1 := newMockRequest("id-1")
	r2 := newMockRequest("id-2")

	if err := m.Set("id-1", r1); err != nil {
		t.Fatalf("Set id-1: %v", err)
	}
	if err := m.Set("id-2", r2); err != nil {
		t.Fatalf("Set id-2: %v", err)
	}

	// Get returns correct entries
	got, ok := m.Get("id-1")
	if !ok {
		t.Fatal("expected to find id-1")
	}
	if got.GetRequestID() != "id-1" {
		t.Fatalf("expected id-1, got %s", got.GetRequestID())
	}

	// All returns both
	all := m.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(all))
	}
}

func TestRequestMap_OrderPreserved(t *testing.T) {
	m := NewKFTLRequestMap()

	ids := []string{"c", "a", "b", "z", "m"}
	for _, id := range ids {
		if err := m.Set(id, newMockRequest(id)); err != nil {
			t.Fatalf("Set %s: %v", id, err)
		}
	}

	all := m.All()
	if len(all) != len(ids) {
		t.Fatalf("expected %d entries, got %d", len(ids), len(all))
	}
	for i, req := range all {
		if req.GetRequestID() != ids[i] {
			t.Errorf("position %d: expected %s, got %s", i, ids[i], req.GetRequestID())
		}
	}
}

func TestRequestMap_SetDuplicateNonPrototypeErrors(t *testing.T) {
	m := NewKFTLRequestMap()

	r1 := newMockRequest("dup")
	if err := m.Set("dup", r1); err != nil {
		t.Fatalf("first Set: %v", err)
	}

	r2 := newMockRequest("dup")
	err := m.Set("dup", r2)
	if err == nil {
		t.Fatal("expected error on duplicate non-prototype Set")
	}
}

func TestRequestMap_PrototypeInheritance(t *testing.T) {
	m := NewKFTLRequestMap()

	// Create a prototype request with tags and related time
	ctx := &KFTLStatementLineContext{
		BaseTime:  time.Now(),
		AddSecond: 0,
	}
	proto := newKFTLPrototypeRequest("inherit-id", ctx)
	proto.AddTag("tag1")
	proto.AddTag("tag2")
	rt := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	proto.SetRelatedTime(rt)

	if err := m.Set("inherit-id", proto); err != nil {
		t.Fatalf("Set prototype: %v", err)
	}

	// Now set a real request with the same ID — it should inherit
	real := newMockRequest("inherit-id")
	if err := m.Set("inherit-id", real); err != nil {
		t.Fatalf("Set real over prototype: %v", err)
	}

	got, ok := m.Get("inherit-id")
	if !ok {
		t.Fatal("expected to find inherit-id")
	}

	tags := got.GetTags()
	if len(tags) != 2 || tags[0] != "tag1" || tags[1] != "tag2" {
		t.Errorf("expected inherited tags [tag1, tag2], got %v", tags)
	}

	// Order should still have only one entry
	all := m.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 entry after prototype replacement, got %d", len(all))
	}
}
