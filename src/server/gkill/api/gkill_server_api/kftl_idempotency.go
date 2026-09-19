package gkill_server_api

import (
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

// idempotencyEntry は成功した送信1回ぶんの控え。
//
// fingerprint は本文と create_app の指紋（kftlSubmissionFingerprint）。同じキーで**別の本文**が
// 届いたことを見分けるために持つ。2026-09-19 までキーだけを覚えていて、別の本文でも
// 「成功・created:[]」で返し、その本文は保存されていなかった（2026-09-18 の MCP 実利用報告）。
// created は元の応答の created[]。応答を受け取り損ねた呼び出し側が再送で ID を回収するために返す。
type idempotencyEntry struct {
	fingerprint string
	created     []*req_res.SubmitKFTLTextCreated
	doneAt      time.Time
}

// idempotencyStore は「成功済みの冪等キー」を TTL 付きで覚える小さなインメモリ台帳。
// KFTL 送信の再配送（Wear のワーカー再送・MCP の応答取りこぼしなど）を
// 1回の登録に畳み、元の結果を再生するために使う。成功したときだけ記録するので、失敗した送信の
// 再試行は通常どおり再実行される。TTL を過ぎたキーは掃除して単調肥大を防ぐ。
type idempotencyStore struct {
	mu   sync.Mutex
	done map[string]idempotencyEntry
	ttl  time.Duration
}

func newIdempotencyStore(ttl time.Duration) *idempotencyStore {
	return &idempotencyStore{done: map[string]idempotencyEntry{}, ttl: ttl}
}

// lookup は key が TTL 内に記録済みならその控えを返す。
func (s *idempotencyStore) lookup(key string) (idempotencyEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.done[key]
	if !ok || time.Since(entry.doneAt) > s.ttl {
		return idempotencyEntry{}, false
	}
	return entry, true
}

// markDone は key を「成功済み」として指紋と結果ごと記録し、ついでに期限切れを掃除する。
func (s *idempotencyStore) markDone(key, fingerprint string, created []*req_res.SubmitKFTLTextCreated) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, entry := range s.done {
		if now.Sub(entry.doneAt) > s.ttl {
			delete(s.done, k)
		}
	}
	s.done[key] = idempotencyEntry{
		fingerprint: fingerprint,
		created:     append([]*req_res.SubmitKFTLTextCreated(nil), created...),
		doneAt:      now,
	}
}

// kftlIdempotencyStore は KFTL 送信の冪等台帳。TTL はワーカー再送が収まる程度に取る。
var kftlIdempotencyStore = newIdempotencyStore(10 * time.Minute)
