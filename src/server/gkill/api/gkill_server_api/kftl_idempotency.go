package gkill_server_api

import (
	"sync"
	"time"
)

// idempotencyStore は「成功済みの冪等キー」を TTL 付きで覚える小さなインメモリ台帳。
// KFTL 送信の再配送（Wear のワーカー再送などで結果だけ届かなかった場合）を
// 1回の登録に畳むために使う。成功したときだけ記録するので、失敗した送信の再試行は
// 通常どおり再実行される。TTL を過ぎたキーは掃除して単調肥大を防ぐ。
type idempotencyStore struct {
	mu   sync.Mutex
	done map[string]time.Time
	ttl  time.Duration
}

func newIdempotencyStore(ttl time.Duration) *idempotencyStore {
	return &idempotencyStore{done: map[string]time.Time{}, ttl: ttl}
}

// alreadyDone は key が TTL 内に記録済みかを返す。
func (s *idempotencyStore) alreadyDone(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.done[key]
	return ok && time.Since(t) <= s.ttl
}

// markDone は key を「成功済み」として記録し、ついでに期限切れを掃除する。
func (s *idempotencyStore) markDone(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, t := range s.done {
		if now.Sub(t) > s.ttl {
			delete(s.done, k)
		}
	}
	s.done[key] = now
}

// kftlIdempotencyStore は KFTL 送信の冪等台帳。TTL はワーカー再送が収まる程度に取る。
var kftlIdempotencyStore = newIdempotencyStore(10 * time.Minute)
