package mcp

// gkill MCP の OAuth 2.1 認可サーバ（旧 oauth-server.mjs）。
// MCP spec 2025-03-26 の「認可サーバ + リソースサーバ同居」モデル。
// 利用者の資格情報の検証は gkill 本体（/api/login）へ委ねる。

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// OAuthServerOptions は OAuthServer の構成。
type OAuthServerOptions struct {
	// Issuer は公開 URL（例 "http://localhost:8808"）。
	Issuer string
	// Scope はこのサーバの唯一の許可スコープ (gkill:read / gkill:write / gkill:readwrite)。
	// metadata・authorize 既定値・トークン発行・受理検証の全てをこの1値から生成する。
	// 以前は metadata だけが "gkill:read" 固定で、ReadWrite サーバの広告と矛盾し、
	// scope が権限境界として機能していなかった (2026-08-30 レビュー P0)。
	Scope string
	// AuthenticateUser は (userID, passwordSha256) → gkill セッション ID。空なら拒否。
	AuthenticateUser AuthenticateUserFunc
	// PersistPath はリフレッシュトークンと DCR クライアントの永続化先。空ならメモリだけ。
	PersistPath string
	// Log は保管庫の失敗を gkill_mcp_<kind>_error.log へ届けるためのロガー。
	Log *Logger
}

// OAuthServer は MCP HTTP トランスポート用の OAuth 2.1 サーバ。
type OAuthServer struct {
	Issuer           string
	Scope            string
	Store            *OAuthStore
	authenticateUser AuthenticateUserFunc
}

// OAuthResult はエンドポイントの応答。HTML か JSON のどちらかを持つ。
type OAuthResult struct {
	Status      int
	ContentType string
	HTML        string
	JSON        *jsonobj.Object
}

var trailingSlashes = regexp.MustCompile(`/+$`)

// NewOAuthServer は認可サーバを作り、状態ファイルを読んで定期掃除を始める。
func NewOAuthServer(opts OAuthServerOptions) (*OAuthServer, error) {
	if opts.Issuer == "" {
		return nil, errors.New("OAuthServer requires an issuer URL")
	}
	if opts.Scope == "" {
		return nil, errors.New("OAuthServer requires a scope")
	}
	if opts.AuthenticateUser == nil {
		return nil, errors.New("OAuthServer requires an authenticateUser function")
	}
	s := &OAuthServer{
		Issuer:           trailingSlashes.ReplaceAllString(opts.Issuer, ""),
		Scope:            opts.Scope,
		authenticateUser: opts.AuthenticateUser,
		Store:            NewOAuthStore(opts.PersistPath, opts.Log),
	}
	s.Store.Load()
	s.Store.StartCleanup(0)
	return s, nil
}

// Close は定期掃除を止める（テストと終了時）。
func (s *OAuthServer) Close() {
	if s.Store != nil {
		s.Store.StopCleanup()
	}
}

// ---------------------------------------------------------------------------
// Metadata endpoint: GET /.well-known/oauth-authorization-server
// ---------------------------------------------------------------------------

// GetMetadata は RFC 8414 のメタデータ。
func (s *OAuthServer) GetMetadata() *jsonobj.Object {
	return jsonobj.Obj(
		"issuer", s.Issuer,
		"authorization_endpoint", s.Issuer+"/oauth/authorize",
		"token_endpoint", s.Issuer+"/oauth/token",
		"registration_endpoint", s.Issuer+"/oauth/register",
		"response_types_supported", jsonobj.Strings("code"),
		"grant_types_supported", jsonobj.Strings("authorization_code", "refresh_token"),
		"token_endpoint_auth_methods_supported", jsonobj.Strings("none"),
		"code_challenge_methods_supported", jsonobj.Strings("S256"),
		"scopes_supported", jsonobj.Strings(s.Scope),
	)
}

// ---------------------------------------------------------------------------
// Authorization endpoint: GET /oauth/authorize — show login form
// ---------------------------------------------------------------------------

func (s *OAuthServer) clientName(clientID string) string {
	if client, ok := s.Store.GetClient(clientID); ok {
		if name, ok := client.String("client_name"); ok {
			return name
		}
	}
	return ""
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// HandleAuthorizeGet はログインフォームを返す。
func (s *OAuthServer) HandleAuthorizeGet(query map[string]string) OAuthResult {
	if message := s.validateAuthorizeParams(query); message != "" {
		return OAuthResult{Status: 400, ContentType: "text/html", HTML: s.errorHTML(message)}
	}
	html := RenderLoginPage(LoginPageParams{
		ClientID:            query["client_id"],
		ClientName:          s.clientName(query["client_id"]),
		RedirectURI:         query["redirect_uri"],
		State:               query["state"],
		CodeChallenge:       query["code_challenge"],
		CodeChallengeMethod: orDefault(query["code_challenge_method"], "S256"),
		// 要求 scope は検証済み (このサーバの唯一の許可 scope と一致するものだけ通る) なので、
		// 表示・再送信とも正準値 s.Scope に正規化する。省略時の既定もここ。
		Scope:    s.Scope,
		Issuer:   s.Issuer,
		Resource: query["resource"],
		Error:    "",
	})
	return OAuthResult{Status: 200, ContentType: "text/html", HTML: html}
}

// ---------------------------------------------------------------------------
// Authorization endpoint: POST /oauth/authorize — authenticate and issue code
// ---------------------------------------------------------------------------

// HandleAuthorizePost は資格情報を検証して認可コードを発行する。
func (s *OAuthServer) HandleAuthorizePost(ctx context.Context, form map[string]string) OAuthResult {
	if message := s.validateAuthorizeParams(form); message != "" {
		return OAuthResult{Status: 400, ContentType: "text/html", HTML: s.errorHTML(message)}
	}

	// Authenticate user via gkill backend
	sessionID, err := s.authenticateUser(ctx, form["user_id"], form["password_sha256"])
	if err != nil {
		sessionID = ""
	}
	if sessionID == "" {
		// Re-render login form with error
		html := RenderLoginPage(LoginPageParams{
			ClientID:            form["client_id"],
			ClientName:          s.clientName(form["client_id"]),
			RedirectURI:         form["redirect_uri"],
			State:               form["state"],
			CodeChallenge:       form["code_challenge"],
			CodeChallengeMethod: orDefault(form["code_challenge_method"], "S256"),
			Scope:               s.Scope,
			Issuer:              s.Issuer,
			Resource:            form["resource"],
			Error:               "ログインに失敗しました。ユーザーIDまたはパスワードを確認してください。",
		})
		return OAuthResult{Status: 200, ContentType: "text/html", HTML: html}
	}

	// Generate authorization code
	code := GenerateToken()
	s.Store.PutCode(code, jsonobj.Obj(
		"clientId", form["client_id"],
		"redirectUri", form["redirect_uri"],
		"codeChallenge", form["code_challenge"],
		"codeChallengeMethod", orDefault(form["code_challenge_method"], "S256"),
		// scope は検証済みなので正準値へ正規化して刻む。トークンにもこの値が乗る。
		"scope", s.Scope,
		"resource", form["resource"],
		"gkillSessionId", sessionID,
		"userId", form["user_id"],
	))

	// Build redirect URL with code
	// validateAuthorizeParams を通っているので解析できるはずだが、戻り値を捨てると
	// 解析に失敗したときに nil 参照で落ちる（プロセスごと落ちる panic になる）。
	redirectURL, err := url.Parse(form["redirect_uri"])
	if err != nil || redirectURL == nil {
		return OAuthResult{Status: 400, ContentType: "text/html", HTML: s.errorHTML("redirect_uri must be a valid URL")}
	}
	params := redirectURL.Query()
	params.Set("code", code)
	if form["state"] != "" {
		params.Set("state", form["state"])
	}
	redirectURL.RawQuery = params.Encode()

	// Show success page with auto-redirect (instead of immediate 302)
	html := RenderSuccessPage(redirectURL.String())
	return OAuthResult{Status: 200, ContentType: "text/html", HTML: html}
}

// ---------------------------------------------------------------------------
// Token endpoint: POST /oauth/token
// ---------------------------------------------------------------------------

// HandleTokenRequest はトークンエンドポイント。
func (s *OAuthServer) HandleTokenRequest(body map[string]string) OAuthResult {
	switch body["grant_type"] {
	case "authorization_code":
		return s.handleAuthorizationCodeGrant(body)
	case "refresh_token":
		return s.handleRefreshTokenGrant(body)
	}
	return s.tokenError("unsupported_grant_type", "Supported: authorization_code, refresh_token")
}

func (s *OAuthServer) handleAuthorizationCodeGrant(body map[string]string) OAuthResult {
	code := body["code"]
	codeVerifier := body["code_verifier"]
	clientID := body["client_id"]
	redirectURI := body["redirect_uri"]
	resource := body["resource"]

	if code == "" {
		return s.tokenError("invalid_request", "Missing code")
	}
	if codeVerifier == "" {
		return s.tokenError("invalid_request", "Missing code_verifier")
	}

	// Retrieve and consume the authorization code (one-time use)
	codeData, ok := s.Store.GetAndDeleteCode(code)
	if !ok || codeData == nil {
		return s.tokenError("invalid_grant", "Invalid or expired authorization code")
	}
	storedClientID, _ := codeData.String("clientId")
	storedRedirectURI, _ := codeData.String("redirectUri")
	storedResource, _ := codeData.String("resource")
	storedChallenge, _ := codeData.String("codeChallenge")
	storedMethod, _ := codeData.String("codeChallengeMethod")
	storedScope, _ := codeData.String("scope")

	// Validate client_id matches
	if clientID != "" && clientID != storedClientID {
		return s.tokenError("invalid_grant", "client_id mismatch")
	}

	// Validate redirect_uri matches
	if redirectURI != "" && redirectURI != storedRedirectURI {
		return s.tokenError("invalid_grant", "redirect_uri mismatch")
	}

	// Validate resource matches (RFC 8707)
	if resource != "" && storedResource != "" && resource != storedResource {
		return s.tokenError("invalid_grant", "resource mismatch")
	}

	// Validate PKCE
	if !VerifyCodeChallenge(codeVerifier, storedChallenge, storedMethod) {
		return s.tokenError("invalid_grant", "PKCE verification failed")
	}

	// scope 不整合の code はトークンへ昇格させない (認可時に正規化済みなので、
	// ここに来るのは scope 修正前のプロセスが発行した残骸だけ)。
	if storedScope != s.Scope {
		return s.tokenError("invalid_scope", `This server issues only "`+s.Scope+`". Re-authorize.`)
	}

	// Issue tokens
	accessToken := GenerateToken()
	refreshToken := GenerateToken()
	tokenData := jsonobj.Obj(
		"clientId", codeData.Value("clientId"),
		"scope", codeData.Value("scope"),
		"gkillSessionId", codeData.Value("gkillSessionId"),
		"userId", codeData.Value("userId"),
	)

	s.Store.PutAccessToken(accessToken, tokenData)
	s.Store.PutRefreshToken(refreshToken, tokenData.Clone())

	return OAuthResult{Status: 200, JSON: jsonobj.Obj(
		"access_token", accessToken,
		"token_type", "Bearer",
		"expires_in", int(TTLAccessToken.Seconds()),
		"refresh_token", refreshToken,
		"scope", codeData.Value("scope"),
	)}
}

func (s *OAuthServer) handleRefreshTokenGrant(body map[string]string) OAuthResult {
	refreshToken := body["refresh_token"]
	clientID := body["client_id"]

	if refreshToken == "" {
		return s.tokenError("invalid_request", "Missing refresh_token")
	}

	tokenData, ok := s.Store.GetRefreshToken(refreshToken)
	if !ok || tokenData == nil {
		return s.tokenError("invalid_grant", "Invalid or expired refresh token")
	}
	storedClientID, _ := tokenData.String("clientId")
	storedScope, _ := tokenData.String("scope")

	// Validate client_id if provided
	if clientID != "" && clientID != storedClientID {
		return s.tokenError("invalid_grant", "client_id mismatch")
	}

	// scope 修正前に永続化された refresh token (例: ReadWrite サーバ上の "gkill:read") は
	// ここで打ち切って再認可させる。回転して生き延びさせても Bearer 受理で 403 になるだけ。
	if storedScope != s.Scope {
		s.Store.DeleteRefreshToken(refreshToken)
		return s.tokenError("invalid_scope", `This server issues only "`+s.Scope+`". Re-authorize.`)
	}

	// Rotate refresh token (delete old, issue new)
	s.Store.DeleteRefreshToken(refreshToken)
	newAccessToken := GenerateToken()
	newRefreshToken := GenerateToken()

	s.Store.PutAccessToken(newAccessToken, tokenData.Clone())
	s.Store.PutRefreshToken(newRefreshToken, tokenData.Clone())

	return OAuthResult{Status: 200, JSON: jsonobj.Obj(
		"access_token", newAccessToken,
		"token_type", "Bearer",
		"expires_in", int(TTLAccessToken.Seconds()),
		"refresh_token", newRefreshToken,
		"scope", tokenData.Value("scope"),
	)}
}

// ---------------------------------------------------------------------------
// Dynamic Client Registration: POST /oauth/register
// ---------------------------------------------------------------------------

// scriptCapableSchemes は「移動しただけで、移動元のページの生成元でスクリプトが走りうる」scheme。
var scriptCapableSchemes = map[string]bool{
	"javascript": true,
	"data":       true,
	"vbscript":   true,
	"blob":       true,
	"about":      true,
	"filesystem": true,
}

// isSafeRedirectURI は redirect_uri として受け付けてよい絶対 URL かを見る。
//
// **絶対 URL であることだけで通さないこと。** 認可成功ページは redirect_uri を
// `window.location.href` に入れて自動遷移する（oauth_html.go の RenderSuccessPage）。
// 動的登録（RFC 7591）は認証を要求しない設計なので、`javascript:` を redirect_uri に
// 登録できてしまうと、「登録する → 被害者に認可 URL を踏ませてログインさせる」の2手で
// gkill の生成元で任意のスクリプトが走り、認可コードごと持ち去られる。
//
// ネイティブアプリのカスタム scheme（`myapp://callback` など）は正当な使い方なので通す。
// 拒むのは上の「移動でスクリプトが走る」scheme だけ。
func isSafeRedirectURI(s string) bool {
	parsed, err := url.Parse(s)
	if err != nil {
		return false
	}
	if parsed.Scheme == "" || scriptCapableSchemes[strings.ToLower(parsed.Scheme)] {
		return false
	}
	return parsed.Host != "" || parsed.Opaque != "" || parsed.Path != ""
}

// HandleRegister は DCR（RFC 7591）。
func (s *OAuthServer) HandleRegister(body any) OAuthResult {
	metadataIn, _ := body.(*jsonobj.Object)
	if metadataIn == nil {
		metadataIn = jsonobj.New()
	}
	redirectURIs, isArray := metadataIn.Array("redirect_uris")
	if !isArray || len(redirectURIs) == 0 {
		return OAuthResult{Status: 400, JSON: jsonobj.Obj("error", "invalid_client_metadata", "error_description", "redirect_uris is required")}
	}

	// Validate each redirect_uri is a valid URL
	for _, uri := range redirectURIs {
		text, isString := uri.(string)
		if !isString || !isSafeRedirectURI(text) {
			return OAuthResult{Status: 400, JSON: jsonobj.Obj("error", "invalid_client_metadata", "error_description", "Invalid redirect_uri: "+jsString(uri))}
		}
	}

	clientID := GenerateToken()
	clientName := ""
	if name := metadataIn.Value("client_name"); jsTruthy(name) {
		clientName = jsString(name)
	}
	metadata := jsonobj.Obj(
		"client_id", clientID,
		"client_id_issued_at", timeNow().Unix(),
		"client_name", clientName,
		"redirect_uris", redirectURIs,
		"grant_types", jsonobj.Strings("authorization_code", "refresh_token"),
		"response_types", jsonobj.Strings("code"),
		"token_endpoint_auth_method", "none",
	)

	s.Store.PutClient(clientID, metadata)

	return OAuthResult{Status: 201, JSON: metadata}
}

// ---------------------------------------------------------------------------
// Token validation (for resource server — validating incoming Bearer tokens)
// ---------------------------------------------------------------------------

// ValidateAccessToken は Bearer アクセストークンを検証し、その中身 ({clientId, scope, gkillSessionId, userId}) を返す。
func (s *OAuthServer) ValidateAccessToken(token string) (*jsonobj.Object, bool) {
	if token == "" || s.Store == nil {
		return nil, false
	}
	return s.Store.GetAccessToken(token)
}

// ExtractBearerToken は Authorization ヘッダから Bearer トークンを取り出す。無ければ空文字列。
func ExtractBearerToken(authHeader string) string {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return authHeader[len("Bearer "):]
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

var whitespaceRun = regexp.MustCompile(`\s+`)

func (s *OAuthServer) validateAuthorizeParams(params map[string]string) string {
	if params["response_type"] == "" || params["response_type"] != "code" {
		return "response_type must be 'code'"
	}
	if params["client_id"] == "" {
		return "client_id is required"
	}
	if params["redirect_uri"] == "" {
		return "redirect_uri is required"
	}
	// 登録時にも同じ判定をするが、ここでも掛ける。古い登録が残っている場合と、
	// 登録を経ない経路が将来増えた場合に、成功ページの自動遷移を守る最後の砦になる。
	if !isSafeRedirectURI(params["redirect_uri"]) {
		return "redirect_uri must be a valid URL"
	}
	if params["code_challenge"] == "" {
		return "code_challenge is required (PKCE)"
	}
	if method := params["code_challenge_method"]; method != "" && !IsSupportedChallengeMethod(method) {
		return "Unsupported code_challenge_method: " + method + ". Use S256."
	}
	// scope はこのサーバの唯一の許可値と厳密一致のみ (省略は既定 = s.Scope として許す)。
	// 黙って別 scope のトークンを発行すると、利用者は「読み取りを許可した」つもりで
	// 書き込み可能なトークンを渡しかねない。空白区切りの複数要求も全要素一致のみ通す。
	if scope := params["scope"]; scope != "" {
		for _, requested := range whitespaceRun.Split(strings.TrimSpace(scope), -1) {
			if requested != s.Scope {
				return `unsupported scope "` + requested + `" — this server issues only "` + s.Scope + `"`
			}
		}
	}
	// DCR で登録済みのクライアントのみ認可する。未登録 client_id を許すと、
	// 攻撃者が任意の client_id + 自分の redirect_uri + 自分の PKCE challenge で認可 URL を
	// 組み立て、被害者にログインさせて code を自分の redirect_uri に受け取り、
	// 自分の verifier で交換してフルトークンを得られる（PKCE は攻撃者が challenge を
	// 選ぶので無力）。gkill には事前登録の仕組みが無く、メタデータで registration_endpoint を
	// 広告済みで正規クライアント（Claude.ai / ChatGPT コネクタ）は DCR を行うので壊れない。
	client, ok := s.Store.GetClient(params["client_id"])
	if !ok {
		return "unknown client_id — register via the registration endpoint first"
	}
	if registered, ok := client.Array("redirect_uris"); ok {
		matched := false
		for _, uri := range registered {
			if uri == params["redirect_uri"] {
				matched = true
				break
			}
		}
		if !matched {
			return "redirect_uri does not match registered client"
		}
	}
	return ""
}

func (s *OAuthServer) tokenError(errorCode, description string) OAuthResult {
	return OAuthResult{Status: 400, JSON: jsonobj.Obj("error", errorCode, "error_description", description)}
}

func (s *OAuthServer) errorHTML(message string) string {
	esc := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(message)
	return `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Error</title></head><body><h1>Error</h1><p>` + esc + `</p></body></html>`
}
