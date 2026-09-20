package mcp

// oauth_server.go の検査。

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type oauthOverrides struct {
	issuer           string
	scope            string
	authenticateUser AuthenticateUserFunc
}

// createOAuthServer はモックの authenticateUser を持つ標準の OAuthServer。
func createOAuthServer(t *testing.T, ov oauthOverrides) *OAuthServer {
	t.Helper()
	authenticateUser := ov.authenticateUser
	if authenticateUser == nil {
		authenticateUser = func(_ context.Context, userID, passwordSha256 string) (string, error) {
			if userID == "admin" && passwordSha256 == "abc123" {
				return "gkill-session-001", nil
			}
			return "", nil
		}
	}
	issuer := ov.issuer
	if issuer == "" {
		issuer = "http://localhost:8808"
	}
	scope := ov.scope
	if scope == "" {
		scope = "gkill:read"
	}
	server, err := NewOAuthServer(OAuthServerOptions{Issuer: issuer, Scope: scope, AuthenticateUser: authenticateUser})
	expectNoError(t, err)
	// 未登録 client_id は認可を拒否するようになったので、テスト既定の "test-client" を
	// DCR 相当で登録しておく（authorizeParams / fullAuthCodeFlow が使う既定クライアント）。
	server.Store.PutClient("test-client", obj("redirect_uris", strs("http://localhost/callback")))
	t.Cleanup(server.Close)
	return server
}

type pkcePair struct{ verifier, challenge string }

// makeS256Pair は PKCE S256 の組を作る。
func newS256Pair() pkcePair {
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	verifier := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(verifier))
	return pkcePair{verifier: verifier, challenge: base64.RawURLEncoding.EncodeToString(sum[:])}
}

// extractRedirectURL は成功ページの HTML から window.location.href = "..." の URL を取り出す。
func extractRedirectURL(html string) string {
	match := regexp.MustCompile(`window\.location\.href\s*=\s*"([^"]+)"`).FindStringSubmatch(html)
	if match == nil {
		return ""
	}
	return match[1]
}

// extractCodeFromResult は認可 POST の成功結果から認可コードを取り出す。
func extractCodeFromResult(t *testing.T, result OAuthResult) string {
	t.Helper()
	expectEqual(t, result.Status, 200)
	expectEqual(t, result.ContentType, "text/html")
	redirectURL := extractRedirectURL(result.HTML)
	expectTrue(t, redirectURL != "", "no redirect URL in the success page")
	parsed, err := url.Parse(redirectURL)
	expectNoError(t, err)
	return parsed.Query().Get("code")
}

// authorizeParams は標準の認可パラメータ。
func authorizeParams(extra map[string]string) map[string]string {
	params := map[string]string{
		"response_type":         "code",
		"client_id":             "test-client",
		"redirect_uri":          "http://localhost/callback",
		"code_challenge":        newS256Pair().challenge,
		"code_challenge_method": "S256",
		"scope":                 "gkill:read",
		"state":                 "xyz",
	}
	for key, value := range extra {
		params[key] = value
	}
	return params
}

func withCredentials(params map[string]string, userID, passwordSha256 string) map[string]string {
	out := map[string]string{}
	for key, value := range params {
		out[key] = value
	}
	out["user_id"] = userID
	out["password_sha256"] = passwordSha256
	return out
}

// fullAuthCodeFlow は認可コードフローを最後まで回し、トークン応答を返す。
func fullAuthCodeFlow(t *testing.T, server *OAuthServer, pkce *pkcePair) *jsonobj.Object {
	t.Helper()
	pair := newS256Pair()
	if pkce != nil {
		pair = *pkce
	}
	params := authorizeParams(map[string]string{"code_challenge": pair.challenge})
	postResult := server.HandleAuthorizePost(context.Background(), withCredentials(params, "admin", "abc123"))
	code := extractCodeFromResult(t, postResult)

	tokenResult := server.HandleTokenRequest(map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"code_verifier": pair.verifier,
		"client_id":     "test-client",
		"redirect_uri":  "http://localhost/callback",
	})
	expectEqual(t, tokenResult.Status, 200)
	return tokenResult.JSON
}

func authorizeAndGetCode(t *testing.T, server *OAuthServer, pair pkcePair, extra map[string]string) string {
	t.Helper()
	params := authorizeParams(map[string]string{"code_challenge": pair.challenge})
	for key, value := range extra {
		params[key] = value
	}
	return extractCodeFromResult(t, server.HandleAuthorizePost(context.Background(), withCredentials(params, "admin", "abc123")))
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------
func TestOAuthServerConstructor(t *testing.T) {
	t.Run("creates with valid options", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		expectEqual(t, server.Issuer, "http://localhost:8808")
	})

	t.Run("strips trailing slash from issuer", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{issuer: "http://localhost:8808/"})
		expectEqual(t, server.Issuer, "http://localhost:8808")
	})

	t.Run("throws without issuer", func(t *testing.T) {
		_, err := NewOAuthServer(OAuthServerOptions{Issuer: "", Scope: "gkill:read", AuthenticateUser: func(context.Context, string, string) (string, error) { return "", nil }})
		expectErrorContains(t, err, "issuer")
	})

	t.Run("throws without authenticateUser", func(t *testing.T) {
		_, err := NewOAuthServer(OAuthServerOptions{Issuer: "http://x", Scope: "gkill:read", AuthenticateUser: nil})
		expectErrorContains(t, err, "authenticateUser")
	})

	// scope 無しの生成を許すと、既定 "gkill:read" へ静かに落ちて ReadWrite サーバの
	// metadata 矛盾 (2026-08-30 レビュー P0) が再発する。生成時点で落とす。
	t.Run("throws without scope", func(t *testing.T) {
		_, err := NewOAuthServer(OAuthServerOptions{Issuer: "http://x", AuthenticateUser: func(context.Context, string, string) (string, error) { return "", nil }})
		expectErrorContains(t, err, "scope")
	})
}

// ---------------------------------------------------------------------------
// Metadata
// ---------------------------------------------------------------------------
func TestOAuthServerGetMetadata(t *testing.T) {
	t.Run("returns correct metadata structure", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		meta := server.GetMetadata()
		expectEqual(t, meta.Value("issuer"), "http://localhost:8808")
		expectEqual(t, meta.Value("authorization_endpoint"), "http://localhost:8808/oauth/authorize")
		expectEqual(t, meta.Value("token_endpoint"), "http://localhost:8808/oauth/token")
		expectEqual(t, meta.Value("registration_endpoint"), "http://localhost:8808/oauth/register")
		expectEqual(t, meta.Value("response_types_supported"), strs("code"))
		expectTrue(t, containsValue(arrAt(t, meta, "grant_types_supported"), "authorization_code"), "authorization_code missing")
		expectTrue(t, containsValue(arrAt(t, meta, "grant_types_supported"), "refresh_token"), "refresh_token missing")
		expectTrue(t, containsValue(arrAt(t, meta, "code_challenge_methods_supported"), "S256"), "S256 missing")
		expectEqual(t, meta.Value("scopes_supported"), strs("gkill:read"))
	})

	// 以前は "gkill:read" がハードコードされ、ReadWrite サーバでも authorization-server
	// metadata だけが gkill:read を広告して protected-resource と矛盾していた。
	t.Run("scopes_supported follows the injected scope", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{scope: "gkill:readwrite"})
		expectEqual(t, server.GetMetadata().Value("scopes_supported"), strs("gkill:readwrite"))
	})
}

// ---------------------------------------------------------------------------
// Authorize GET
// ---------------------------------------------------------------------------
func TestOAuthServerHandleAuthorizeGet(t *testing.T) {
	t.Run("returns 200 with login form HTML", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleAuthorizeGet(authorizeParams(nil))
		expectEqual(t, result.Status, 200)
		expectEqual(t, result.ContentType, "text/html")
		mustContain(t, result.HTML, "gkill")
		mustContain(t, result.HTML, "loginForm")
	})

	t.Run("returns 400 for missing response_type", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		expectEqual(t, server.HandleAuthorizeGet(authorizeParams(map[string]string{"response_type": ""})).Status, 400)
	})

	t.Run("returns 400 for invalid response_type", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		expectEqual(t, server.HandleAuthorizeGet(authorizeParams(map[string]string{"response_type": "token"})).Status, 400)
	})

	t.Run("returns 400 for missing client_id", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		expectEqual(t, server.HandleAuthorizeGet(authorizeParams(map[string]string{"client_id": ""})).Status, 400)
	})

	t.Run("returns 400 for missing redirect_uri", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		expectEqual(t, server.HandleAuthorizeGet(authorizeParams(map[string]string{"redirect_uri": ""})).Status, 400)
	})

	t.Run("returns 400 for invalid redirect_uri", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		expectEqual(t, server.HandleAuthorizeGet(authorizeParams(map[string]string{"redirect_uri": "not-a-url"})).Status, 400)
	})

	t.Run("returns 400 for missing code_challenge", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		expectEqual(t, server.HandleAuthorizeGet(authorizeParams(map[string]string{"code_challenge": ""})).Status, 400)
	})

	t.Run("returns 400 for unsupported code_challenge_method", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		expectEqual(t, server.HandleAuthorizeGet(authorizeParams(map[string]string{"code_challenge_method": "S512"})).Status, 400)
	})

	t.Run("defaults code_challenge_method to S256", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		params := authorizeParams(nil)
		delete(params, "code_challenge_method")
		expectEqual(t, server.HandleAuthorizeGet(params).Status, 200)
	})
}

// ---------------------------------------------------------------------------
// Authorize POST — success
// ---------------------------------------------------------------------------
func TestOAuthServerHandleAuthorizePostSuccess(t *testing.T) {
	t.Run("returns success page with redirect URL on successful login", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		result := server.HandleAuthorizePost(context.Background(), withCredentials(authorizeParams(map[string]string{"code_challenge": pair.challenge}), "admin", "abc123"))
		expectEqual(t, result.Status, 200)
		redirectURL, err := url.Parse(extractRedirectURL(result.HTML))
		expectNoError(t, err)
		expectTrue(t, redirectURL.Query().Get("code") != "", "code missing")
		expectEqual(t, redirectURL.Query().Get("state"), "xyz")
		expectEqual(t, redirectURL.Scheme+"://"+redirectURL.Host+redirectURL.Path, "http://localhost/callback")
	})

	t.Run("omits state from redirect when not provided", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		result := server.HandleAuthorizePost(context.Background(), withCredentials(authorizeParams(map[string]string{"code_challenge": pair.challenge, "state": ""}), "admin", "abc123"))
		expectEqual(t, result.Status, 200)
		redirectURL, err := url.Parse(extractRedirectURL(result.HTML))
		expectNoError(t, err)
		expectTrue(t, !redirectURL.Query().Has("state"), "state present")
	})
}

// ---------------------------------------------------------------------------
// Authorize POST — failure
// ---------------------------------------------------------------------------
func TestOAuthServerHandleAuthorizePostFailure(t *testing.T) {
	t.Run("re-renders login form with error on bad credentials", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleAuthorizePost(context.Background(), withCredentials(authorizeParams(nil), "admin", "wrong"))
		expectEqual(t, result.Status, 200)
		mustContain(t, result.HTML, "ログインに失敗しました")
	})

	t.Run("re-renders login form when authenticateUser throws", func(t *testing.T) {
		server2 := createOAuthServer(t, oauthOverrides{authenticateUser: func(context.Context, string, string) (string, error) {
			return "", errors.New("network error")
		}})
		result := server2.HandleAuthorizePost(context.Background(), withCredentials(authorizeParams(nil), "admin", "abc123"))
		expectEqual(t, result.Status, 200)
		mustContain(t, result.HTML, "ログインに失敗しました")
	})

	t.Run("returns 400 for invalid params in POST", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleAuthorizePost(context.Background(), withCredentials(authorizeParams(map[string]string{"response_type": "token"}), "admin", "abc123"))
		expectEqual(t, result.Status, 400)
	})
}

// ---------------------------------------------------------------------------
// Token endpoint — authorization_code grant
// ---------------------------------------------------------------------------
func TestOAuthServerTokenAuthorizationCode(t *testing.T) {
	t.Run("exchanges valid code + verifier for tokens", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		tokens := fullAuthCodeFlow(t, server, &pair)
		expectTrue(t, jsTruthy(tokens.Value("access_token")), "access_token missing")
		expectTrue(t, jsTruthy(tokens.Value("refresh_token")), "refresh_token missing")
		expectEqual(t, tokens.Value("token_type"), "Bearer")
		expectEqual(t, tokens.Value("expires_in"), 3600)
		expectEqual(t, tokens.Value("scope"), "gkill:read")
	})

	t.Run("rejects code replay (one-time use)", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		code := authorizeAndGetCode(t, server, pair, nil)

		// First exchange succeeds
		result1 := server.HandleTokenRequest(map[string]string{"grant_type": "authorization_code", "code": code, "code_verifier": pair.verifier})
		expectEqual(t, result1.Status, 200)

		// Second exchange fails (code already consumed)
		result2 := server.HandleTokenRequest(map[string]string{"grant_type": "authorization_code", "code": code, "code_verifier": pair.verifier})
		expectEqual(t, result2.Status, 400)
		expectEqual(t, result2.JSON.Value("error"), "invalid_grant")
	})

	t.Run("rejects wrong code_verifier (PKCE failure)", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		code := authorizeAndGetCode(t, server, pair, nil)

		result := server.HandleTokenRequest(map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"code_verifier": "wrong-verifier-that-is-definitely-not-correct-at-all",
		})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_grant")
		mustContain(t, strAt(t, result.JSON, "error_description"), "PKCE")
	})

	t.Run("rejects expired authorization code", func(t *testing.T) {
		clock := useFakeTimers(t)
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		code := authorizeAndGetCode(t, server, pair, nil)

		// Advance time past code TTL (5 minutes)
		clock.advance(5*time.Minute + time.Millisecond)

		result := server.HandleTokenRequest(map[string]string{"grant_type": "authorization_code", "code": code, "code_verifier": pair.verifier})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_grant")
	})

	t.Run("rejects mismatched client_id", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		code := authorizeAndGetCode(t, server, pair, nil)

		result := server.HandleTokenRequest(map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"code_verifier": pair.verifier,
			"client_id":     "different-client",
		})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_grant")
	})

	t.Run("rejects mismatched redirect_uri", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		code := authorizeAndGetCode(t, server, pair, nil)

		result := server.HandleTokenRequest(map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"code_verifier": pair.verifier,
			"redirect_uri":  "http://evil.com/callback",
		})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_grant")
	})

	t.Run("rejects missing code", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleTokenRequest(map[string]string{"grant_type": "authorization_code", "code_verifier": "x"})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_request")
	})

	t.Run("rejects missing code_verifier", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleTokenRequest(map[string]string{"grant_type": "authorization_code", "code": "some-code"})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_request")
	})
}

// ---------------------------------------------------------------------------
// Token endpoint — refresh_token grant
// ---------------------------------------------------------------------------
func TestOAuthServerTokenRefreshToken(t *testing.T) {
	t.Run("issues new tokens from valid refresh_token", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		tokens := fullAuthCodeFlow(t, server, nil)
		result := server.HandleTokenRequest(map[string]string{"grant_type": "refresh_token", "refresh_token": strAt(t, tokens, "refresh_token")})
		expectEqual(t, result.Status, 200)
		expectTrue(t, jsTruthy(result.JSON.Value("access_token")), "access_token missing")
		expectTrue(t, jsTruthy(result.JSON.Value("refresh_token")), "refresh_token missing")
		// New tokens should differ
		expectTrue(t, result.JSON.Value("access_token") != tokens.Value("access_token"), "access_token unchanged")
		expectTrue(t, result.JSON.Value("refresh_token") != tokens.Value("refresh_token"), "refresh_token unchanged")
	})

	t.Run("rotates refresh_token (old one becomes invalid)", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		tokens := fullAuthCodeFlow(t, server, nil)
		oldRefresh := strAt(t, tokens, "refresh_token")

		// Use refresh token
		result1 := server.HandleTokenRequest(map[string]string{"grant_type": "refresh_token", "refresh_token": oldRefresh})
		expectEqual(t, result1.Status, 200)

		// Old refresh token should now be invalid
		result2 := server.HandleTokenRequest(map[string]string{"grant_type": "refresh_token", "refresh_token": oldRefresh})
		expectEqual(t, result2.Status, 400)
		expectEqual(t, result2.JSON.Value("error"), "invalid_grant")
	})

	t.Run("rejects invalid refresh_token", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleTokenRequest(map[string]string{"grant_type": "refresh_token", "refresh_token": "invalid-token"})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_grant")
	})

	t.Run("rejects missing refresh_token", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleTokenRequest(map[string]string{"grant_type": "refresh_token"})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_request")
	})

	t.Run("rejects mismatched client_id on refresh", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		tokens := fullAuthCodeFlow(t, server, nil)
		result := server.HandleTokenRequest(map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": strAt(t, tokens, "refresh_token"),
			"client_id":     "wrong-client",
		})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_grant")
	})
}

// ---------------------------------------------------------------------------
// Token endpoint — unsupported grant type
// ---------------------------------------------------------------------------
func TestOAuthServerTokenUnsupported(t *testing.T) {
	t.Run("rejects unsupported grant_type", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleTokenRequest(map[string]string{"grant_type": "client_credentials"})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "unsupported_grant_type")
	})
}

// ---------------------------------------------------------------------------
// Dynamic Client Registration
// ---------------------------------------------------------------------------
func TestOAuthServerHandleRegister(t *testing.T) {
	t.Run("registers a client with valid metadata", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleRegister(obj("redirect_uris", strs("http://localhost/callback"), "client_name", "My App"))
		expectEqual(t, result.Status, 201)
		expectTrue(t, jsTruthy(result.JSON.Value("client_id")), "client_id missing")
		expectEqual(t, result.JSON.Value("client_name"), "My App")
		expectEqual(t, result.JSON.Value("redirect_uris"), strs("http://localhost/callback"))
		expectTrue(t, containsValue(arrAt(t, result.JSON, "grant_types"), "authorization_code"), "grant_types lacks authorization_code")
	})

	t.Run("includes client_id_issued_at in response", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		before := time.Now().Unix()
		result := server.HandleRegister(obj("redirect_uris", strs("http://localhost/cb")))
		after := time.Now().Unix()
		issuedAt := int64(floatOf(t, result.JSON.Value("client_id_issued_at")))
		expectTrue(t, issuedAt >= before, "issued_at before now")
		expectTrue(t, issuedAt <= after, "issued_at after now")
	})

	t.Run("registered client can be retrieved from store", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleRegister(obj("redirect_uris", strs("http://localhost/cb")))
		stored, ok := server.Store.GetClient(strAt(t, result.JSON, "client_id"))
		expectTrue(t, ok, "client not stored")
		expectEqual(t, stored.Value("client_id"), result.JSON.Value("client_id"))
	})

	t.Run("rejects missing redirect_uris", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleRegister(obj())
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_client_metadata")
	})

	t.Run("rejects empty redirect_uris array", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		expectEqual(t, server.HandleRegister(obj("redirect_uris", arr())).Status, 400)
	})

	t.Run("rejects invalid redirect_uri URL", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleRegister(obj("redirect_uris", strs("not-a-url")))
		expectEqual(t, result.Status, 400)
		mustContain(t, strAt(t, result.JSON, "error_description"), "Invalid redirect_uri")
	})

	t.Run("defaults client_name to empty string", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleRegister(obj("redirect_uris", strs("http://localhost/cb")))
		expectEqual(t, result.JSON.Value("client_name"), "")
	})
}

// ---------------------------------------------------------------------------
// Token validation
// ---------------------------------------------------------------------------
func TestOAuthServerValidateAccessToken(t *testing.T) {
	t.Run("returns token data for valid access token", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		tokens := fullAuthCodeFlow(t, server, nil)
		data, ok := server.ValidateAccessToken(strAt(t, tokens, "access_token"))
		expectTrue(t, ok, "token invalid")
		expectEqual(t, data.Value("userId"), "admin")
		expectEqual(t, data.Value("gkillSessionId"), "gkill-session-001")
		expectEqual(t, data.Value("clientId"), "test-client")
	})

	t.Run("returns null for invalid token", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		_, ok := server.ValidateAccessToken("bad-token")
		expectTrue(t, !ok, "bad token validated")
	})

	t.Run("returns null for empty token", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		_, ok := server.ValidateAccessToken("")
		expectTrue(t, !ok, "empty token validated")
	})

	t.Run("returns null for expired access token", func(t *testing.T) {
		clock := useFakeTimers(t)
		server := createOAuthServer(t, oauthOverrides{})
		tokens := fullAuthCodeFlow(t, server, nil)
		clock.advance(60*time.Minute + time.Millisecond) // 1 hour + 1ms
		_, ok := server.ValidateAccessToken(strAt(t, tokens, "access_token"))
		expectTrue(t, !ok, "expired token validated")
	})
}

// ---------------------------------------------------------------------------
// extractBearerToken
// ---------------------------------------------------------------------------
func TestOAuthServerExtractBearerToken(t *testing.T) {
	t.Run("extracts token from valid Bearer header", func(t *testing.T) {
		expectEqual(t, ExtractBearerToken("Bearer abc123"), "abc123")
	})

	t.Run("returns empty for non-Bearer header", func(t *testing.T) {
		expectEqual(t, ExtractBearerToken("Basic abc123"), "")
	})

	t.Run("returns empty for empty string", func(t *testing.T) {
		expectEqual(t, ExtractBearerToken(""), "")
	})

	t.Run("returns empty for null/undefined", func(t *testing.T) {
		// Go では省略値は空文字列
		expectEqual(t, ExtractBearerToken(""), "")
	})
}

// ---------------------------------------------------------------------------
// Full end-to-end flow
// ---------------------------------------------------------------------------
func TestOAuthServerFullE2EFlow(t *testing.T) {
	t.Run("authorize → token → validate → refresh → validate", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()

		// 1. GET authorize (show form)
		getResult := server.HandleAuthorizeGet(authorizeParams(map[string]string{"code_challenge": pair.challenge}))
		expectEqual(t, getResult.Status, 200)

		// 2. POST authorize (login)
		code := authorizeAndGetCode(t, server, pair, nil)

		// 3. Token exchange
		tokenResult := server.HandleTokenRequest(map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"code_verifier": pair.verifier,
			"client_id":     "test-client",
			"redirect_uri":  "http://localhost/callback",
		})
		expectEqual(t, tokenResult.Status, 200)

		// 4. Validate access token
		tokenData, ok := server.ValidateAccessToken(strAt(t, tokenResult.JSON, "access_token"))
		expectTrue(t, ok, "token invalid")
		expectEqual(t, tokenData.Value("userId"), "admin")
		expectEqual(t, tokenData.Value("gkillSessionId"), "gkill-session-001")

		// 5. Refresh
		refreshResult := server.HandleTokenRequest(map[string]string{"grant_type": "refresh_token", "refresh_token": strAt(t, tokenResult.JSON, "refresh_token")})
		expectEqual(t, refreshResult.Status, 200)

		// 6. Validate new access token
		newData, ok := server.ValidateAccessToken(strAt(t, refreshResult.JSON, "access_token"))
		expectTrue(t, ok, "new token invalid")
		expectEqual(t, newData.Value("userId"), "admin")

		// 7. Old access token still valid (not rotated, only refresh token rotates)
		_, ok = server.ValidateAccessToken(strAt(t, tokenResult.JSON, "access_token"))
		expectTrue(t, ok, "old access token invalidated")
	})
}

// ---------------------------------------------------------------------------
// RFC 8707 resource parameter
// ---------------------------------------------------------------------------
func TestOAuthServerResourceParameter(t *testing.T) {
	t.Run("stores resource in authorization code data", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		params := authorizeParams(map[string]string{"code_challenge": pair.challenge, "resource": "http://localhost:8808/mcp"})
		postResult := server.HandleAuthorizePost(context.Background(), withCredentials(params, "admin", "abc123"))
		expectEqual(t, postResult.Status, 200)
		mustContain(t, postResult.HTML, "ログイン成功")
	})

	t.Run("token exchange succeeds with matching resource", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		resource := "http://localhost:8808/mcp"
		code := authorizeAndGetCode(t, server, pair, map[string]string{"resource": resource})

		result := server.HandleTokenRequest(map[string]string{"grant_type": "authorization_code", "code": code, "code_verifier": pair.verifier, "resource": resource})
		expectEqual(t, result.Status, 200)
		expectTrue(t, jsTruthy(result.JSON.Value("access_token")), "access_token missing")
	})

	t.Run("token exchange rejects mismatched resource", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		code := authorizeAndGetCode(t, server, pair, map[string]string{"resource": "http://localhost:8808/mcp"})

		result := server.HandleTokenRequest(map[string]string{"grant_type": "authorization_code", "code": code, "code_verifier": pair.verifier, "resource": "http://evil.com/mcp"})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_grant")
		mustContain(t, strAt(t, result.JSON, "error_description"), "resource")
	})

	t.Run("token exchange succeeds when resource omitted from token request", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		code := authorizeAndGetCode(t, server, pair, map[string]string{"resource": "http://localhost:8808/mcp"})

		result := server.HandleTokenRequest(map[string]string{"grant_type": "authorization_code", "code": code, "code_verifier": pair.verifier})
		expectEqual(t, result.Status, 200)
	})

	t.Run("login form preserves resource on auth failure", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		pair := newS256Pair()
		params := authorizeParams(map[string]string{"code_challenge": pair.challenge, "resource": "http://localhost:8808/mcp"})
		result := server.HandleAuthorizePost(context.Background(), withCredentials(params, "admin", "wrong"))
		expectEqual(t, result.Status, 200)
		mustContain(t, result.HTML, "http://localhost:8808/mcp")
	})
}

// ---------------------------------------------------------------------------
// DCR redirect_uri validation at authorize time
// ---------------------------------------------------------------------------
func TestOAuthServerRedirectURIValidationAgainstDCR(t *testing.T) {
	t.Run("rejects redirect_uri not matching DCR-registered client", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		regResult := server.HandleRegister(obj("redirect_uris", strs("http://localhost/callback"), "client_name", "Test"))
		clientID := strAt(t, regResult.JSON, "client_id")

		result := server.HandleAuthorizeGet(authorizeParams(map[string]string{"client_id": clientID, "redirect_uri": "http://evil.com/callback"}))
		expectEqual(t, result.Status, 400)
		mustContain(t, result.HTML, "redirect_uri does not match")
	})

	t.Run("accepts redirect_uri matching DCR-registered client", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		regResult := server.HandleRegister(obj("redirect_uris", strs("http://localhost/callback")))
		clientID := strAt(t, regResult.JSON, "client_id")

		result := server.HandleAuthorizeGet(authorizeParams(map[string]string{"client_id": clientID, "redirect_uri": "http://localhost/callback"}))
		expectEqual(t, result.Status, 200)
	})

	t.Run("rejects unregistered client_id (auth-code interception hardening)", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleAuthorizeGet(authorizeParams(map[string]string{"client_id": "never-registered-client", "redirect_uri": "http://any-url.com/callback"}))
		expectEqual(t, result.Status, 400)
		mustContain(t, result.HTML, "unknown client_id")
	})
}

// ---------------------------------------------------------------------------
// scope 境界 (2026-08-30 レビュー P0)
// ---------------------------------------------------------------------------
func TestOAuthServerScopeEnforcement(t *testing.T) {
	t.Run("authorize rejects a scope this server does not issue", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{scope: "gkill:readwrite"})
		result := server.HandleAuthorizeGet(authorizeParams(map[string]string{"scope": "gkill:read"}))
		expectEqual(t, result.Status, 400)
		mustContain(t, result.HTML, "unsupported scope")
	})

	t.Run("authorize rejects a multi-scope request containing a foreign scope", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleAuthorizeGet(authorizeParams(map[string]string{"scope": "gkill:read gkill:readwrite"}))
		expectEqual(t, result.Status, 400)
	})

	t.Run("authorize POST with a foreign scope is rejected before authentication and issues no code", func(t *testing.T) {
		// code を発行するのは POST 経路で、攻撃者はフォーム (hidden の scope は正準値固定)
		// を介さず POST を直接叩ける。GET 側だけの検証では code の発行そのものを守れない。
		var calls int64
		server := createOAuthServer(t, oauthOverrides{scope: "gkill:readwrite", authenticateUser: func(context.Context, string, string) (string, error) {
			atomic.AddInt64(&calls, 1)
			return "sess-should-never-exist", nil
		}})
		result := server.HandleAuthorizePost(context.Background(), withCredentials(authorizeParams(map[string]string{"scope": "gkill:read"}), "admin", "abc123"))
		expectEqual(t, result.Status, 400)
		mustContain(t, result.HTML, "unsupported scope")
		// 検証は認証より前 — 資格情報の処理も code の発行も一切走らない。
		expectEqual(t, atomic.LoadInt64(&calls), 0)
	})

	t.Run("omitted scope defaults to the server scope and the token carries it", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{scope: "gkill:readwrite"})
		pair := newS256Pair()
		params := authorizeParams(map[string]string{"code_challenge": pair.challenge})
		delete(params, "scope")
		code := extractCodeFromResult(t, server.HandleAuthorizePost(context.Background(), withCredentials(params, "admin", "abc123")))
		tokenResult := server.HandleTokenRequest(map[string]string{"grant_type": "authorization_code", "code": code, "code_verifier": pair.verifier})
		expectEqual(t, tokenResult.Status, 200)
		expectEqual(t, tokenResult.JSON.Value("scope"), "gkill:readwrite")
		data, _ := server.ValidateAccessToken(strAt(t, tokenResult.JSON, "access_token"))
		expectEqual(t, data.Value("scope"), "gkill:readwrite")
	})

	t.Run("refresh of a legacy wrong-scope token is rejected and the token is revoked", func(t *testing.T) {
		// scope 修正前の ReadWrite サーバが発行・永続化した "gkill:read" の refresh token を再現する。
		server := createOAuthServer(t, oauthOverrides{scope: "gkill:readwrite"})
		server.Store.PutRefreshToken("legacy-refresh", obj(
			"clientId", "test-client",
			"scope", "gkill:read",
			"gkillSessionId", "sess-legacy",
			"userId", "admin",
		))
		result := server.HandleTokenRequest(map[string]string{"grant_type": "refresh_token", "refresh_token": "legacy-refresh"})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_scope")
		// 打ち切った refresh token は残さない (再試行しても同じ失敗を繰り返すだけ)。
		_, ok := server.Store.GetRefreshToken("legacy-refresh")
		expectTrue(t, !ok, "legacy refresh token kept")
	})

	t.Run("a legacy wrong-scope authorization code is not exchanged for tokens", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{scope: "gkill:readwrite"})
		pair := newS256Pair()
		server.Store.PutCode("legacy-code", obj(
			"clientId", "test-client",
			"redirectUri", "http://localhost/callback",
			"codeChallenge", pair.challenge,
			"codeChallengeMethod", "S256",
			"scope", "gkill:read",
			"resource", "",
			"gkillSessionId", "sess-legacy",
			"userId", "admin",
		))
		result := server.HandleTokenRequest(map[string]string{"grant_type": "authorization_code", "code": "legacy-code", "code_verifier": pair.verifier})
		expectEqual(t, result.Status, 400)
		expectEqual(t, result.JSON.Value("error"), "invalid_scope")
	})
}

// ---------------------------------------------------------------------------
// 同意画面の権限表示 (2026-08-30 レビュー P0)
// ---------------------------------------------------------------------------
func TestOAuthServerConsentDisplayOnTheLoginPage(t *testing.T) {
	t.Run("readwrite login page shows the writable scope and its meaning", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{scope: "gkill:readwrite"})
		result := server.HandleAuthorizeGet(authorizeParams(map[string]string{"scope": "gkill:readwrite"}))
		expectEqual(t, result.Status, 200)
		mustContain(t, result.HTML, "gkill:readwrite")
		mustContain(t, result.HTML, "読み書き")
		mustContain(t, result.HTML, "削除")
		// class 属性そのものを見る (<style> 内の .consent-writable は常に居るため、
		// 文字列 "consent-writable" の存在検査では強調表示の欠落を検出できない)。
		mustContain(t, result.HTML, `class="consent-writable"`)
	})

	t.Run("write login page shows the write-only scope as writable", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{scope: "gkill:write"})
		result := server.HandleAuthorizeGet(authorizeParams(map[string]string{"scope": "gkill:write"}))
		expectEqual(t, result.Status, 200)
		mustContain(t, result.HTML, "gkill:write")
		mustContain(t, result.HTML, "書き込み")
		mustContain(t, result.HTML, `class="consent-writable"`)
	})

	t.Run("read login page says read-only and is not marked writable", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		result := server.HandleAuthorizeGet(authorizeParams(nil))
		expectEqual(t, result.Status, 200)
		mustContain(t, result.HTML, "読み取り専用")
		mustNotContain(t, result.HTML, `class="consent-writable"`)
	})

	t.Run("a scope without a description falls back to the raw value", func(t *testing.T) {
		// scopeDescriptions に無い scope でも空表示にしない (label = 生の scope 値)。
		server := createOAuthServer(t, oauthOverrides{scope: "custom:x"})
		result := server.HandleAuthorizeGet(authorizeParams(map[string]string{"scope": "custom:x"}))
		expectEqual(t, result.Status, 200)
		mustContain(t, result.HTML, "custom:x (custom:x)")
		mustNotContain(t, result.HTML, "読み取り専用")
	})

	t.Run("login page shows the DCR client_name and the issuer", func(t *testing.T) {
		server := createOAuthServer(t, oauthOverrides{})
		reg := server.HandleRegister(obj("redirect_uris", strs("http://localhost/callback"), "client_name", "Example Connector"))
		result := server.HandleAuthorizeGet(authorizeParams(map[string]string{"client_id": strAt(t, reg.JSON, "client_id")}))
		expectEqual(t, result.Status, 200)
		mustContain(t, result.HTML, "Example Connector")
		mustContain(t, result.HTML, "http://localhost:8808")
	})

	t.Run("escapes a hostile DCR client_name before rendering it into the login page", func(t *testing.T) {
		// client_name は DCR (HandleRegister) が無検証で保存する攻撃者制御値で、
		// 同意ブロックの導入で初めてログイン HTML へ描画されるようになった。
		server := createOAuthServer(t, oauthOverrides{})
		reg := server.HandleRegister(obj("redirect_uris", strs("http://localhost/callback"), "client_name", `<script>alert(1)</script>"`))
		result := server.HandleAuthorizeGet(authorizeParams(map[string]string{"client_id": strAt(t, reg.JSON, "client_id")}))
		expectEqual(t, result.Status, 200)
		mustNotContain(t, result.HTML, "<script>alert(1)</script>")
		mustContain(t, result.HTML, "&lt;script&gt;alert(1)&lt;/script&gt;&quot;")
	})
}
