package mcp

// 期限付きの file-link トークンをメモリに持つ保管庫（旧 file-link-store.mjs）。
//
// リモートの MCP クライアント（OAuth 越しのクラウド AI など）はローカルのファイルパスを
// 読めないので、代わりに不透明な URL `${publicBaseUrl}/files/${token}` を渡す。
// トークンはちょうど1ファイル（rep_name + file_name）に結びつき、鋳造したときに
// 認証されていた gkill セッションを運ぶので、公開の配信ルートは URL に資格情報を
// 出さずにバイト列を取りに行ける。
//
// トークンそのものが安全境界: 推測不能（crypto/rand）、1ファイル限定、期限付き。
// URL を知っている者はそのファイルを期限まで取れる —— それ以外は何もできない。

import (
	"os"
	"strconv"
	"sync"
	"time"
)

// FileLinkTTL はトークンの既定寿命。リモートクライアントは鋳造直後に取りに来るので
// 短くて安全。「あとで会話を見返す」用途のため GKILL_MCP_FILE_LINK_TTL_MS で延ばせる。
var FileLinkTTL = fileLinkTTLFromEnv()

const defaultFileLinkTTL = 60 * time.Minute

func fileLinkTTLFromEnv() time.Duration {
	return ParseFileLinkTTL(os.Getenv("GKILL_MCP_FILE_LINK_TTL_MS"))
}

// ParseFileLinkTTL はミリ秒の文字列を寿命にする。空・不正は既定、1秒未満は1秒。
func ParseFileLinkTTL(value string) time.Duration {
	ms, err := strconv.ParseFloat(value, 64)
	if err != nil || ms == 0 {
		return defaultFileLinkTTL
	}
	if ms < 1000 {
		return time.Second
	}
	return time.Duration(ms) * time.Millisecond
}

// FileLink はトークン1つが指すファイル。
type FileLink struct {
	GkillSessionID string
	RepName        string
	FileName       string
	IsImage        bool
}

type fileLinkEntry struct {
	value     FileLink
	expiresAt time.Time
}

// FileLinkStore は TTL 付きのメモリ保管庫。
type FileLinkStore struct {
	mu      sync.Mutex
	links   map[string]fileLinkEntry
	ttl     time.Duration
	stopCh  chan struct{}
	stopped bool
}

// NewFileLinkStore は保管庫を作る。ttl が 0 以下なら FileLinkTTL。
func NewFileLinkStore(ttl time.Duration) *FileLinkStore {
	if ttl <= 0 {
		ttl = FileLinkTTL
	}
	return &FileLinkStore{links: map[string]fileLinkEntry{}, ttl: ttl}
}

// TTL は既定寿命。
func (s *FileLinkStore) TTL() time.Duration { return s.ttl }

// StartCleanup は期限切れの掃除を定期的に回す（既定 5 分）。
func (s *FileLinkStore) StartCleanup(interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	s.StopCleanup()
	s.mu.Lock()
	stopCh := make(chan struct{})
	s.stopCh = stopCh
	s.mu.Unlock()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.Sweep()
			case <-stopCh:
				return
			}
		}
	}()
}

// StopCleanup は定期掃除を止める。
func (s *FileLinkStore) StopCleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
}

// Sweep は期限切れを全部消す。
func (s *FileLinkStore) Sweep() {
	now := timeNow()
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, entry := range s.links {
		if now.After(entry.expiresAt) {
			delete(s.links, token)
		}
	}
}

// Mint は1ファイルのトークンを既定寿命で鋳造する。
func (s *FileLinkStore) Mint(data FileLink) string {
	token, _ := s.MintLink(data, s.ttl)
	return token
}

// MintWithTTL は寿命を指定して鋳造する（負なら即期限切れ）。
func (s *FileLinkStore) MintWithTTL(data FileLink, ttl time.Duration) string {
	token, _ := s.MintLink(data, ttl)
	return token
}

// MintLink はトークンと期限を返す。応答に「いつまで有効か」を添えるため
// （人間へ渡したリンクは、1時間後に開いたとき切れていることを知らないと役に立たない）。
func (s *FileLinkStore) MintLink(data FileLink, ttl time.Duration) (string, time.Time) {
	token := GenerateToken()
	expiresAt := timeNow().Add(ttl)
	s.mu.Lock()
	s.links[token] = fileLinkEntry{value: data, expiresAt: expiresAt}
	s.mu.Unlock()
	return token, expiresAt
}

// Resolve はトークンをファイル情報にする。未知か期限切れなら false（期限切れはその場で消す）。
func (s *FileLinkStore) Resolve(token string) (FileLink, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.links[token]
	if !ok {
		return FileLink{}, false
	}
	if timeNow().After(entry.expiresAt) {
		delete(s.links, token)
		return FileLink{}, false
	}
	return entry.value, true
}

// Has はトークンが（期限に関わらず）保管されているか。
func (s *FileLinkStore) Has(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.links[token]
	return ok
}

// Len は保管中のトークン数。
func (s *FileLinkStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.links)
}
