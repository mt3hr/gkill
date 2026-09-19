package gkill_server_api

import (
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

// 記録前は未達、markDone 後は TTL 内なら指紋と結果ごと引ける。
func TestIdempotencyStore_MarkAndLookup(t *testing.T) {
	s := newIdempotencyStore(time.Hour)
	if _, ok := s.lookup("k1"); ok {
		t.Fatalf("未記録のキーが達成済みになっている")
	}
	created := []*req_res.SubmitKFTLTextCreated{{ID: "id-1", DataType: "kmemo"}}
	s.markDone("k1", "fp-1", created)
	entry, ok := s.lookup("k1")
	if !ok {
		t.Fatalf("記録したキーが達成済みにならない")
	}
	if entry.fingerprint != "fp-1" {
		t.Errorf("fingerprint = %q, want fp-1", entry.fingerprint)
	}
	if len(entry.created) != 1 || entry.created[0].ID != "id-1" {
		t.Errorf("created = %+v, want 元の1件", entry.created)
	}
	// 控えは呼び出し側のスライスと共有しない（後から書き換えても再生結果が変わらない）
	created[0] = &req_res.SubmitKFTLTextCreated{ID: "tampered"}
	if entry.created[0].ID != "id-1" {
		t.Errorf("控えが呼び出し側のスライスと共有されている: %q", entry.created[0].ID)
	}
	// 別キーは独立
	if _, ok := s.lookup("k2"); ok {
		t.Fatalf("別キーが巻き込まれて達成済みになっている")
	}
}

// TTL を過ぎたキーは達成済みでなくなる（再実行されるべき）。
func TestIdempotencyStore_Expiry(t *testing.T) {
	s := newIdempotencyStore(1 * time.Millisecond)
	s.markDone("k1", "fp", nil)
	time.Sleep(20 * time.Millisecond)
	if _, ok := s.lookup("k1"); ok {
		t.Fatalf("TTL 超過後も達成済みのまま")
	}
}

// markDone は期限切れエントリを掃除して単調肥大しない。
func TestIdempotencyStore_GCOnMarkDone(t *testing.T) {
	s := newIdempotencyStore(1 * time.Millisecond)
	s.markDone("old", "fp", nil)
	time.Sleep(20 * time.Millisecond)
	// 別キーを記録すると、その中で期限切れの "old" が掃除される
	s.markDone("new", "fp", nil)
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

// 指紋は本文と create_app の両方を見る（同じ本文でも create_app が違えば別の送信）。
func TestKFTLSubmissionFingerprint(t *testing.T) {
	a := kftlSubmissionFingerprint("memo", "gkill_kftl")
	if a != kftlSubmissionFingerprint("memo", "gkill_kftl") {
		t.Fatal("同じ入力で指紋が変わる")
	}
	if a == kftlSubmissionFingerprint("memo2", "gkill_kftl") {
		t.Error("本文が違うのに指紋が同じ")
	}
	if a == kftlSubmissionFingerprint("memo", "gkill_wear") {
		t.Error("create_app が違うのに指紋が同じ")
	}
}
