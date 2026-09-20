package mcp

// OAuth の認可コード・アクセストークン・リフレッシュトークン・動的クライアント登録の
// メモリ保管庫（旧 oauth-store.mjs）。エントリは TTL で期限切れになる。
// リフレッシュトークンとクライアント登録は JSON ファイルへ永続化できる。
//
// ファイルの書式は Node 版と同じ（{"refreshTokens": {token: {value, expiresAt}}, "clients": {id: metadata}}、
// expiresAt はエポックミリ秒）。既存の状態ファイルをそのまま読めなければ、
// 稼働中の ChatGPT / claude.ai コネクタが全部再認可になる。

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// 既定の TTL。
const (
	TTLAuthorizationCode = 5 * time.Minute
	TTLAccessToken       = 60 * time.Minute
	TTLRefreshToken      = 30 * 24 * time.Hour
)

// randomBytes は乱数の読み口（テストとゴールデン再生で決定的な列に差し替える）。
var randomBytes = func(n int) []byte {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return buf
}

// GenerateToken は推測不能な不透明トークン（64 桁の hex）を作る。
func GenerateToken() string {
	return hex.EncodeToString(randomBytes(32))
}

// OAuthStats は保管庫の件数（デバッグ用）。
type OAuthStats struct {
	Codes         int
	AccessTokens  int
	RefreshTokens int
	Clients       int
}

// OAuthStore は TTL 付きのメモリ保管庫。各エントリは {value, expiresAt}。
//
// 4つの表は挿入順を保つため *jsonobj.Object で持つ（Node の Map と同じ順序で永続化される）。
type OAuthStore struct {
	mu            sync.Mutex
	codes         *jsonobj.Object
	accessTokens  *jsonobj.Object
	refreshTokens *jsonobj.Object
	clients       *jsonobj.Object
	persistPath   string
	log           *Logger
	stopCh        chan struct{}
}

// NewOAuthStore は保管庫を作る。persistPath が空ならメモリだけで動く。
func NewOAuthStore(persistPath string, log *Logger) *OAuthStore {
	return &OAuthStore{
		codes:         jsonobj.New(),
		accessTokens:  jsonobj.New(),
		refreshTokens: jsonobj.New(),
		clients:       jsonobj.New(),
		persistPath:   persistPath,
		log:           log,
	}
}

func epochMillis(t time.Time) int64 { return t.UnixMilli() }

func newEntry(value *jsonobj.Object, ttl time.Duration) *jsonobj.Object {
	return jsonobj.Obj("value", value, "expiresAt", epochMillis(timeNow().Add(ttl)))
}

func entryExpiresAt(entry *jsonobj.Object) int64 {
	f, ok := entry.Float("expiresAt")
	if !ok {
		return 0
	}
	return int64(f)
}

func entryValue(entry *jsonobj.Object) *jsonobj.Object {
	value, ok := entry.Object("value")
	if !ok {
		return nil
	}
	return value
}

// StartCleanup は期限切れの掃除を定期的に回す（既定 5 分）。
func (s *OAuthStore) StartCleanup(interval time.Duration) {
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
func (s *OAuthStore) StopCleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
}

// --- Authorization Codes ---

// PutCode は認可コードを既定 TTL で保存する。
func (s *OAuthStore) PutCode(code string, data *jsonobj.Object) {
	s.PutCodeWithTTL(code, data, TTLAuthorizationCode)
}

// PutCodeWithTTL は認可コードを TTL つきで保存する。
func (s *OAuthStore) PutCodeWithTTL(code string, data *jsonobj.Object, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes.Set(code, newEntry(data, ttl))
}

// GetAndDeleteCode は認可コードを取り出して消す（1回限り）。無いか期限切れなら false。
func (s *OAuthStore) GetAndDeleteCode(code string) (*jsonobj.Object, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.codes.Object(code)
	if !ok {
		return nil, false
	}
	s.codes.Delete(code)
	if epochMillis(timeNow()) > entryExpiresAt(entry) {
		return nil, false
	}
	return entryValue(entry), true
}

// CodesLen は保管中の認可コード数。
func (s *OAuthStore) CodesLen() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.codes.Len()
}

// --- Access Tokens ---

// PutAccessToken はアクセストークンを既定 TTL で保存する。
func (s *OAuthStore) PutAccessToken(token string, data *jsonobj.Object) {
	s.PutAccessTokenWithTTL(token, data, TTLAccessToken)
}

// PutAccessTokenWithTTL はアクセストークンを TTL つきで保存する。
func (s *OAuthStore) PutAccessTokenWithTTL(token string, data *jsonobj.Object, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accessTokens.Set(token, newEntry(data, ttl))
}

// GetAccessToken はアクセストークンの中身。無いか期限切れなら false（期限切れは消す）。
func (s *OAuthStore) GetAccessToken(token string) (*jsonobj.Object, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.accessTokens.Object(token)
	if !ok {
		return nil, false
	}
	if epochMillis(timeNow()) > entryExpiresAt(entry) {
		s.accessTokens.Delete(token)
		return nil, false
	}
	return entryValue(entry), true
}

// HasAccessToken はトークンが（期限に関わらず）保管されているか。
func (s *OAuthStore) HasAccessToken(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accessTokens.Has(token)
}

// DeleteAccessToken はアクセストークンを消す。
func (s *OAuthStore) DeleteAccessToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accessTokens.Delete(token)
}

// --- Refresh Tokens ---

// PutRefreshToken はリフレッシュトークンを既定 TTL で保存し、ファイルへ書く。
func (s *OAuthStore) PutRefreshToken(token string, data *jsonobj.Object) {
	s.PutRefreshTokenWithTTL(token, data, TTLRefreshToken)
}

// PutRefreshTokenWithTTL はリフレッシュトークンを TTL つきで保存し、ファイルへ書く。
func (s *OAuthStore) PutRefreshTokenWithTTL(token string, data *jsonobj.Object, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshTokens.Set(token, newEntry(data, ttl))
	s.saveLocked()
}

// GetRefreshToken はリフレッシュトークンの中身。無いか期限切れなら false（期限切れは消す）。
func (s *OAuthStore) GetRefreshToken(token string) (*jsonobj.Object, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.refreshTokens.Object(token)
	if !ok {
		return nil, false
	}
	if epochMillis(timeNow()) > entryExpiresAt(entry) {
		s.refreshTokens.Delete(token)
		return nil, false
	}
	return entryValue(entry), true
}

// DeleteRefreshToken はリフレッシュトークンを消し（回転時など）、ファイルへ書く。
func (s *OAuthStore) DeleteRefreshToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshTokens.Delete(token)
	s.saveLocked()
}

// --- Client Registrations ---

// PutClient は動的登録されたクライアントを保存し、ファイルへ書く。
func (s *OAuthStore) PutClient(clientID string, metadata *jsonobj.Object) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients.Set(clientID, metadata)
	s.saveLocked()
}

// GetClient は登録済みクライアントのメタデータ。無ければ false。
func (s *OAuthStore) GetClient(clientID string) (*jsonobj.Object, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	metadata, ok := s.clients.Object(clientID)
	if !ok || metadata == nil {
		return nil, false
	}
	return metadata, true
}

// --- Cleanup ---

// Sweep は codes / accessTokens / refreshTokens の期限切れを全部消す。
func (s *OAuthStore) Sweep() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := epochMillis(timeNow())
	for _, table := range []*jsonobj.Object{s.codes, s.accessTokens, s.refreshTokens} {
		for _, key := range table.Keys() {
			entry, ok := table.Object(key)
			if ok && now > entryExpiresAt(entry) {
				table.Delete(key)
			}
		}
	}
}

// Stats は件数を返す（デバッグ用）。
func (s *OAuthStore) Stats() OAuthStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return OAuthStats{
		Codes:         s.codes.Len(),
		AccessTokens:  s.accessTokens.Len(),
		RefreshTokens: s.refreshTokens.Len(),
		Clients:       s.clients.Len(),
	}
}

// --- Persistence ---

// Load はリフレッシュトークンとクライアント登録をファイルから読む。
// ファイルが無い・壊れているときは黙って空から始める。期限切れのリフレッシュトークンは飛ばす。
func (s *OAuthStore) Load() {
	if s.persistPath == "" {
		return
	}
	raw, err := os.ReadFile(s.persistPath)
	if err != nil {
		return
	}
	parsed, err := jsonobj.Unmarshal(raw)
	if err != nil {
		return
	}
	data, ok := parsed.(*jsonobj.Object)
	if !ok || data == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := epochMillis(timeNow())
	if refreshTokens, ok := data.Object("refreshTokens"); ok {
		for _, token := range refreshTokens.Keys() {
			entry, ok := refreshTokens.Object(token)
			if !ok || entry == nil {
				continue
			}
			if entryExpiresAt(entry) > now {
				s.refreshTokens.Set(token, jsonobj.Obj("value", entry.Value("value"), "expiresAt", entry.Value("expiresAt")))
			}
		}
	}
	if clients, ok := data.Object("clients"); ok {
		for _, clientID := range clients.Keys() {
			s.clients.Set(clientID, clients.Value(clientID))
		}
	}
}

// saveLocked はリフレッシュトークンとクライアント登録をファイルへ書く。persistPath が空なら何もしない。
func (s *OAuthStore) saveLocked() {
	if s.persistPath == "" {
		return
	}
	data := jsonobj.Obj(
		"refreshTokens", s.refreshTokens,
		"clients", s.clients,
	)
	if err := s.writeStateFile(data); err != nil {
		// ここが落ちると refresh token と DCR クライアントが揮発し、次回は全部再認可になる。
		// stderr だけだと誰も見ない（サービス起動）ので、ログにも残す。
		s.log.Error("oauth_state_save_error", "error", err.Error())
		fmt.Fprintf(os.Stderr, "OAuth state save error: %s\n", err.Error())
	}
}

func (s *OAuthStore) writeStateFile(data *jsonobj.Object) error {
	if err := os.MkdirAll(filepath.Dir(s.persistPath), 0o755); err != nil {
		return err
	}
	// 一時ファイルへ 0600 で書いてから rename で本体を差し替える（同一ディレクトリなので原子的）。
	// 素朴な書き込みは (1) 既定 0644 で30日有効の refresh token が他ローカルユーザから読める
	// (2) 書き込み途中でクラッシュすると JSON が壊れ、次回 load が全 refresh token / DCR クライアントを
	// 捨てて再認可になる、の2点があった。mode 0600 は Windows では実質 no-op（ACL は親から継承）。
	tmpPath := s.persistPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(jsonobj.MarshalIndentString(data, "  ")), 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.persistPath)
}
