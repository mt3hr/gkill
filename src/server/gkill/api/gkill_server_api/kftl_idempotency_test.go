package gkill_server_api

import (
	"testing"
	"time"
)

// 記録前は未達、markDone 後は TTL 内なら達成済み。
func TestIdempotencyStore_MarkAndCheck(t *testing.T) {
	s := newIdempotencyStore(time.Hour)
	if s.alreadyDone("k1") {
		t.Fatalf("未記録のキーが達成済みになっている")
	}
	s.markDone("k1")
	if !s.alreadyDone("k1") {
		t.Fatalf("記録したキーが達成済みにならない")
	}
	// 別キーは独立
	if s.alreadyDone("k2") {
		t.Fatalf("別キーが巻き込まれて達成済みになっている")
	}
}

// TTL を過ぎたキーは達成済みでなくなる（再実行されるべき）。
func TestIdempotencyStore_Expiry(t *testing.T) {
	s := newIdempotencyStore(1 * time.Millisecond)
	s.markDone("k1")
	time.Sleep(20 * time.Millisecond)
	if s.alreadyDone("k1") {
		t.Fatalf("TTL 超過後も達成済みのまま")
	}
}

// markDone は期限切れエントリを掃除して単調肥大しない。
func TestIdempotencyStore_GCOnMarkDone(t *testing.T) {
	s := newIdempotencyStore(1 * time.Millisecond)
	s.markDone("old")
	time.Sleep(20 * time.Millisecond)
	// 別キーを記録すると、その中で期限切れの "old" が掃除される
	s.markDone("new")
	s.mu.Lock()
	_, oldStillThere := s.done["old"]
	size := len(s.done)
	s.mu.Unlock()
	if oldStillThere {
		t.Fatalf("期限切れエントリが掃除されていない")
	}
	if size != 1 {
		t.Fatalf("掃除後のサイズが想定外: got %d want 1", size)
	}
}
