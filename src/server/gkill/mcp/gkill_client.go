package mcp

// gkill 本体を叩く HTTP クライアント。3つの MCP サーバで共有する（旧 gkill-client.mjs）。
//
// fetchFile は IDF ファイルの実体を取りに行く経路。
// 書き込み専用サーバはファイル系ツールを持たないので呼ばないが、
// 実装を分けるほどの差ではないので同じ型に置いてある。

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// GkillAPI はハンドラが gkill 本体へ話しかける口。テストはこれを差し替える。
type GkillAPI interface {
	// CallApi は JSON を POST し、応答本文（gkill の envelope）を返す。
	// requiresAuth のときは session_id を本文へ足し、セッション切れなら1回だけログインし直す。
	// sessionIDOverride が空でなければそのセッションを使う（HTTP モードの利用者セッション）。
	CallApi(ctx context.Context, pathname string, body *jsonobj.Object, requiresAuth bool, sessionIDOverride string) (*jsonobj.Object, error)
	// FetchFile は IDF ファイルの実体を GET する。
	FetchFile(ctx context.Context, filePath string, sessionID string) (*FileResponse, error)
	// Login はログインして新しいセッション ID を返す（保持しているセッションも更新する）。
	Login(ctx context.Context) (string, error)
	// Post は認証を伴わない素の POST（ログインとその周辺だけが使う）。
	Post(ctx context.Context, pathname string, body *jsonobj.Object) (*jsonobj.Object, error)
	// HasErrors は envelope の errors が空でないか。
	HasErrors(response *jsonobj.Object) bool
	DefaultLocale() string
	UserID() string
}

// FileResponse は FetchFile の結果。
type FileResponse struct {
	Buffer      []byte
	ContentType string
}

// isGkillEnvelope は gkill の応答形（成功でも失敗でも errors / messages を持つ）か。
// ステータスが 2xx でなくても、この形なら業務エラーとして呼び出し側へ渡す。
func isGkillEnvelope(body any) bool {
	o, ok := body.(*jsonobj.Object)
	if !ok || o == nil {
		return false
	}
	return o.Has("errors") || o.Has("messages")
}

// AuthErrorCodes は gkill 側が「セッションが無効」を表すときに返すエラーコード。
// これを見たら1回だけログインし直して再試行する。
var AuthErrorCodes = NewStringSet(
	"ERR000002", // AccountNotFoundError
	"ERR000013", // AccountSessionNotFoundError
	"ERR000238", // AccountDisabledError
	"ERR000373", // AccountSessionExpiredError
)

// DefaultGkillBaseURL は GKILL_BASE_URL の既定。
const DefaultGkillBaseURL = "http://127.0.0.1:9999"

// DefaultFetchTimeout は GKILL_FETCH_TIMEOUT_MS の既定。
const DefaultFetchTimeout = 120000 * time.Millisecond

// ClientConfig は GkillClient の構成（環境変数 / 設定ファイル / フラグから組む）。
type ClientConfig struct {
	BaseURL        string
	UserID         string
	PasswordSha256 string
	Password       string
	Locale         string
	SessionID      string
	Insecure       bool
	FetchTimeout   time.Duration
}

// ClientConfigFromEnv は Node 実装と同じ環境変数から構成を組む（無ければ既定値）。
func ClientConfigFromEnv() ClientConfig {
	insecure := os.Getenv("GKILL_INSECURE")
	return ClientConfig{
		BaseURL:        envOr("GKILL_BASE_URL", DefaultGkillBaseURL),
		UserID:         os.Getenv("GKILL_USER"),
		PasswordSha256: os.Getenv("GKILL_PASSWORD_SHA256"),
		Password:       os.Getenv("GKILL_PASSWORD"),
		Locale:         envOr("GKILL_LOCALE", "ja"),
		SessionID:      os.Getenv("GKILL_SESSION_ID"),
		Insecure:       insecure == "true" || insecure == "1",
		FetchTimeout:   ParseFetchTimeout(os.Getenv("GKILL_FETCH_TIMEOUT_MS")),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ParseFetchTimeout はミリ秒の文字列を時間にする。空・不正は既定。
func ParseFetchTimeout(value string) time.Duration {
	ms, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || ms <= 0 {
		return DefaultFetchTimeout
	}
	return time.Duration(ms) * time.Millisecond
}

// GkillClient は gkill 本体への HTTP クライアント。
type GkillClient struct {
	BaseURL        string
	PasswordSha256 string
	Password       string
	Insecure       bool
	FetchTimeout   time.Duration
	HTTPClient     *http.Client

	userID        string
	defaultLocale string

	mu        sync.Mutex
	sessionID string
}

var _ GkillAPI = (*GkillClient)(nil)

// NewGkillClient は構成からクライアントを作る。
func NewGkillClient(cfg ClientConfig) *GkillClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultGkillBaseURL
	}
	if cfg.Locale == "" {
		cfg.Locale = "ja"
	}
	if cfg.FetchTimeout <= 0 {
		cfg.FetchTimeout = DefaultFetchTimeout
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // GKILL_INSECURE は自己署名の gkill へ繋ぐための明示的な選択
	}
	return &GkillClient{
		BaseURL:        cfg.BaseURL,
		PasswordSha256: cfg.PasswordSha256,
		Password:       cfg.Password,
		Insecure:       cfg.Insecure,
		FetchTimeout:   cfg.FetchTimeout,
		HTTPClient:     &http.Client{Transport: transport},
		userID:         cfg.UserID,
		defaultLocale:  cfg.Locale,
		sessionID:      cfg.SessionID,
	}
}

// NewGkillClientFromEnv は環境変数だけからクライアントを作る（Node のコンストラクタと同じ）。
func NewGkillClientFromEnv() *GkillClient {
	return NewGkillClient(ClientConfigFromEnv())
}

// UserID は接続アカウント。
func (c *GkillClient) UserID() string { return c.userID }

// DefaultLocale は locale_name の既定。
func (c *GkillClient) DefaultLocale() string { return c.defaultLocale }

// SessionID は保持しているセッション ID。
func (c *GkillClient) SessionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionID
}

// SetSessionID はセッション ID を差し替える。
func (c *GkillClient) SetSessionID(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = sessionID
}

// ResolvePasswordSha256 はハッシュがあればそれを、平文しか無ければ SHA-256 にして返す。
func (c *GkillClient) ResolvePasswordSha256() string {
	if c.PasswordSha256 != "" {
		return c.PasswordSha256
	}
	if c.Password != "" {
		sum := sha256.Sum256([]byte(c.Password))
		return hex.EncodeToString(sum[:])
	}
	return ""
}

// BuildApiURL は new URL(pathname, baseUrl) と同じ解決。
func (c *GkillClient) BuildApiURL(pathname string) string {
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return c.BaseURL + pathname
	}
	ref, err := url.Parse(pathname)
	if err != nil {
		return c.BaseURL + pathname
	}
	return base.ResolveReference(ref).String()
}

// HasErrors は envelope に業務エラーが載っているか。
func (c *GkillClient) HasErrors(response *jsonobj.Object) bool {
	return responseHasErrors(response)
}

// HasAuthErrors はセッション無効を表すエラーが載っているか。
func (c *GkillClient) HasAuthErrors(response *jsonobj.Object) bool {
	if !c.HasErrors(response) {
		return false
	}
	errs, _ := response.Array("errors")
	for _, item := range errs {
		if o, ok := item.(*jsonobj.Object); ok {
			if code, ok := o.String("error_code"); ok && AuthErrorCodes.Has(code) {
				return true
			}
		}
	}
	return false
}

// FormatErrors は "CODE: message [kind=..., reason=...]" を "; " で繋ぐ。
// error_kind（誰の問題か）と reason（何が起きたか。分類できたときだけ）はサーバが付ける
// 機械語（ADR-0710）で、利用者のロケールに依らず次の手を示す
// （reason=write_rep_missing なら「書き込み先の設定」、db_busy なら「少し待って再試行」）。
func (c *GkillClient) FormatErrors(response *jsonobj.Object) string {
	if !c.HasErrors(response) {
		return ""
	}
	errs, _ := response.Array("errors")
	parts := []string{}
	for _, item := range errs {
		o, _ := item.(*jsonobj.Object)
		tokens := []string{}
		if kind := o.Value("error_kind"); jsTruthy(kind) {
			tokens = append(tokens, "kind="+jsString(kind))
		}
		if reason := o.Value("reason"); jsTruthy(reason) {
			tokens = append(tokens, "reason="+jsString(reason))
		}
		suffix := ""
		if len(tokens) > 0 {
			suffix = " [" + strings.Join(tokens, ", ") + "]"
		}
		code := "UNKNOWN"
		if v := o.Value("error_code"); v != nil && !jsonobj.IsUndefined(v) {
			code = jsString(v)
		}
		message := "unknown error"
		if v := o.Value("error_message"); v != nil && !jsonobj.IsUndefined(v) {
			message = jsString(v)
		}
		parts = append(parts, code+": "+message+suffix)
	}
	return strings.Join(parts, "; ")
}

// responseHasErrors は gkill の envelope に業務エラーが載っているか。
func responseHasErrors(response *jsonobj.Object) bool {
	if response == nil {
		return false
	}
	errs, ok := response.Array("errors")
	return ok && len(errs) > 0
}

// Post は JSON を POST して本文を返す。
func (c *GkillClient) Post(ctx context.Context, pathname string, body *jsonobj.Object) (*jsonobj.Object, error) {
	target := c.BuildApiURL(pathname)
	if body == nil {
		body = jsonobj.New()
	}
	encoded, err := jsonobj.Marshal(body)
	if err != nil {
		return nil, NewGkillApiError("Failed to encode request body for "+pathname+".", jsonobj.Obj("cause", err.Error()))
	}
	reqCtx, cancel := context.WithTimeout(ctx, c.FetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, target, bytes.NewReader(encoded))
	if err != nil {
		return nil, NewGkillApiError("Network error at "+pathname+".", jsonobj.Obj("url", target, "message", err.Error(), "cause", nil))
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, NewGkillApiError("Network error at "+pathname+".", jsonobj.Obj("url", target, "message", err.Error(), "cause", nil))
	}
	defer response.Body.Close()
	raw, readErr := io.ReadAll(response.Body)
	ok := response.StatusCode >= 200 && response.StatusCode <= 299
	var parsed any
	parseErr := readErr
	if parseErr == nil {
		parsed, parseErr = jsonobj.Unmarshal(raw)
	}
	if parseErr != nil {
		// 本文が JSON でない非2xx (本文なしの413/500、リバースプロキシの HTML 502/504 等) を
		// 「Failed to parse JSON response」で握り潰すと、HTTP ステータスが呼び出し側へ
		// 一切届かない。非2xxならステータスを含むエラーとして投げる。
		if !ok {
			return nil, NewGkillApiError("HTTP "+itoa(response.StatusCode)+" from "+pathname+".", jsonobj.Obj("status", response.StatusCode, "cause", parseErr.Error()))
		}
		return nil, NewGkillApiError("Failed to parse JSON response from "+pathname+".", jsonobj.Obj("cause", parseErr.Error()))
	}

	// gkill は 2026-08 から、異常時に 400/401/403/404/409/429/500 を返す。
	// ここでステータスだけを見て throw すると、本文の errors 配列が呼び出し側へ届かない。
	// 特に CallApi の「セッション切れを見つけたら1回だけログインし直す」経路
	// (HasAuthErrors) に到達しなくなり、MCP は長寿命プロセスなので
	// **セッション期限が来た時点で全ツールが復旧不能になる**。
	// なので gkill 形式の本文(errors 配列を持つ)なら、ステータスに関わらずそのまま返す。
	// 呼び出し側は今までどおり errors を見て判断する。
	if !ok && !isGkillEnvelope(parsed) {
		return nil, NewGkillApiError("HTTP "+itoa(response.StatusCode)+" from "+pathname+".", jsonobj.Obj("status", response.StatusCode, "body", parsed))
	}
	object, isObject := parsed.(*jsonobj.Object)
	if !isObject {
		return nil, NewGkillApiError("Failed to parse JSON response from "+pathname+".", jsonobj.Obj("cause", "response is not a JSON object"))
	}
	return object, nil
}

// Login はセッションが無ければログインしてセッション ID を返す。
func (c *GkillClient) Login(ctx context.Context) (string, error) {
	if sid := c.SessionID(); sid != "" {
		return sid, nil
	}
	passwordSha256 := c.ResolvePasswordSha256()
	if c.userID == "" || passwordSha256 == "" {
		return "", NewGkillApiError("Missing login credentials. Set GKILL_USER and GKILL_PASSWORD_SHA256 (or GKILL_PASSWORD).", nil)
	}
	response, err := c.Post(ctx, "/api/login", jsonobj.Obj(
		"user_id", c.userID,
		"password_sha256", passwordSha256,
		"locale_name", c.defaultLocale,
	))
	if err != nil {
		return "", err
	}
	if c.HasErrors(response) {
		return "", NewGkillApiError("Login failed: "+c.FormatErrors(response), response)
	}
	sessionID, ok := response.String("session_id")
	if !ok || sessionID == "" {
		return "", NewGkillApiError("Login succeeded but session_id is missing.", response)
	}
	c.SetSessionID(sessionID)
	return sessionID, nil
}

// CallApi は locale_name と session_id を補って POST し、セッション切れなら1回だけログインし直す。
func (c *GkillClient) CallApi(ctx context.Context, pathname string, requestBody *jsonobj.Object, requiresAuth bool, sessionIDOverride string) (*jsonobj.Object, error) {
	body := jsonobj.New()
	if requestBody != nil {
		body = requestBody.Clone()
	}
	localeName := c.defaultLocale
	if v, ok := body.String("locale_name"); ok && v != "" {
		localeName = v
	}
	body.Set("locale_name", localeName)

	if requiresAuth {
		sessionID := sessionIDOverride
		if sessionID == "" {
			if v, ok := body.String("session_id"); ok && v != "" {
				sessionID = v
			}
		}
		if sessionID == "" {
			sid, err := c.Login(ctx)
			if err != nil {
				return nil, err
			}
			sessionID = sid
		}
		body.Set("session_id", sessionID)
	}

	response, err := c.Post(ctx, pathname, body)
	if err != nil {
		return nil, err
	}
	if requiresAuth && c.HasAuthErrors(response) {
		c.SetSessionID("")
		sid, err := c.Login(ctx)
		if err != nil {
			return nil, err
		}
		body.Set("session_id", sid)
		response, err = c.Post(ctx, pathname, body)
		if err != nil {
			return nil, err
		}
	}

	if c.HasErrors(response) {
		return nil, NewGkillApiError("API error at "+pathname+": "+c.FormatErrors(response), response)
	}
	return response, nil
}

// FetchFile は IDF ファイルの実体を取る。
func (c *GkillClient) FetchFile(ctx context.Context, filePath string, sessionID string) (*FileResponse, error) {
	response, err := c.fetchFileOnce(ctx, filePath, sessionID)
	if err != nil {
		return nil, err
	}
	// /files/ はエンベロープ(errors配列)を返さないので、セッション切れは HTTP 401 でしか
	// 分からない。CallApi の再ログイン (HasAuthErrors→Login→再試行) をミラーし、
	// 401 なら1回だけログインし直して再試行する。2回目も失敗なら従来どおりエラー。
	if response.StatusCode == http.StatusUnauthorized {
		_ = response.Body.Close()
		c.SetSessionID("")
		refreshedSessionID, err := c.Login(ctx)
		if err != nil {
			return nil, err
		}
		response, err = c.fetchFileOnce(ctx, filePath, refreshedSessionID)
		if err != nil {
			return nil, err
		}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, NewGkillApiError("HTTP "+itoa(response.StatusCode)+" fetching file "+filePath+".", jsonobj.Obj("status", response.StatusCode))
	}
	contentType := response.Header.Get("content-type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	buffer, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, NewGkillApiError("Network error fetching file "+filePath+".", jsonobj.Obj("url", c.BuildApiURL(filePath), "message", err.Error()))
	}
	return &FileResponse{Buffer: buffer, ContentType: contentType}, nil
}

// fetchFileOnce は FetchFile の1回分の GET リクエスト。再ログイン後の再試行のために分離してある。
func (c *GkillClient) fetchFileOnce(ctx context.Context, filePath string, sessionID string) (*http.Response, error) {
	target := c.BuildApiURL(filePath)
	reqCtx, cancel := context.WithTimeout(ctx, c.FetchTimeout)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, target, nil)
	if err != nil {
		cancel()
		return nil, NewGkillApiError("Network error fetching file "+filePath+".", jsonobj.Obj("url", target, "message", err.Error()))
	}
	req.Header.Set("Cookie", "gkill_session_id="+sessionID)
	response, err := c.HTTPClient.Do(req)
	if err != nil {
		cancel()
		return nil, NewGkillApiError("Network error fetching file "+filePath+".", jsonobj.Obj("url", target, "message", err.Error()))
	}
	// 本文を読み切るまでタイムアウトを生かす（Body を閉じたときに cancel する）
	response.Body = &cancelOnClose{ReadCloser: response.Body, cancel: cancel}
	return response, nil
}

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}
