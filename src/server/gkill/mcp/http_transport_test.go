package mcp

// HttpTransport の /mcp 経路の統合・回帰テスト。
//
// 既存テスト (file_link_test.go) は HandleFileServe を直接呼ぶだけで、
// ServeHTTP の Bearer 認証・OAuth 連携・並行性は一度もテスト経路に乗っていなかった。
// そのため以下の3件が緑の CI をすり抜けていた:
//   - C-01: http-transport が OAuthServer を import せず /mcp で ReferenceError
//   - C-02: server.current* 共有フィールドで並行リクエストの user/session が混線
//   - M-06: 公開ファイル配信に nosniff / CSP sandbox が無い
//
// このファイルは実ポート (0 = OS 採番) で HttpTransport を起動し、
// http.Client で OAuth→Bearer→tools まで通す統合と、並行分離の回帰を守る。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// 3サーバの違いは scope とファイル配信ルートの有無だけ。共有 transport を各 scope で検査する。
type serverVariant struct {
	scope string
	make  func(client GkillAPI, log *Logger) *Server
}

var serverVariants = []serverVariant{
	{"gkill:read", NewReadServer},
	{"gkill:write", NewWriteServer},
	{"gkill:readwrite", NewReadWriteServer},
}

func createTransportMockClient() *mockClient {
	return &mockClient{
		callApi: func(_ string, _ *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) {
			return obj("errors", arr(), "messages", arr()), nil
		},
		fetchFile: func(_ string, _ string) (*FileResponse, error) {
			return &FileResponse{Buffer: []byte("x"), ContentType: "application/octet-stream"}, nil
		},
		login:         func() (string, error) { return "login-session", nil },
		userID:        "client-user",
		defaultLocale: "ja",
	}
}

// makeOAuth は userId ごとに別セッションを返す OAuthServer (並行分離テストで token を見分けるため)。
// scope の正本は OAuthServer のこの1値。HttpTransport は oauth.Scope を読むので、
// transport の options へ scope を渡さない。
func makeOAuth(t *testing.T, scope string) *OAuthServer {
	t.Helper()
	oauth, err := NewOAuthServer(OAuthServerOptions{
		Issuer: "http://127.0.0.1:0",
		Scope:  scope,
		AuthenticateUser: func(_ context.Context, userID, _ string) (string, error) {
			return "sess-" + userID, nil
		},
	})
	expectNoError(t, err)
	t.Cleanup(oauth.Close)
	return oauth
}

// registerClient は DCR でクライアントを登録して client_id を得る。
func registerClient(t *testing.T, oauth *OAuthServer) string {
	t.Helper()
	res := oauth.HandleRegister(obj("redirect_uris", strs("http://localhost/callback"), "client_name", "test"))
	expectEqual(t, res.Status, 201)
	return strAt(t, res.JSON, "client_id")
}

// mintAccessToken は認可コードフローを OAuthServer 上で直接回して access token を1つ得る。
func mintAccessToken(t *testing.T, oauth *OAuthServer, userID, scope, clientID string) string {
	t.Helper()
	pair := newS256Pair()
	params := map[string]string{
		"response_type":         "code",
		"client_id":             clientID,
		"redirect_uri":          "http://localhost/callback",
		"code_challenge":        pair.challenge,
		"code_challenge_method": "S256",
		"scope":                 scope,
		"state":                 "s",
		"user_id":               userID,
		"password_sha256":       "pw",
	}
	post := oauth.HandleAuthorizePost(context.Background(), params)
	redirectURL, err := url.Parse(extractRedirectURL(post.HTML))
	expectNoError(t, err)
	tok := oauth.HandleTokenRequest(map[string]string{
		"grant_type":    "authorization_code",
		"code":          redirectURL.Query().Get("code"),
		"code_verifier": pair.verifier,
		"client_id":     clientID,
		"redirect_uri":  "http://localhost/callback",
	})
	expectEqual(t, tok.Status, 200)
	return strAt(t, tok.JSON, "access_token")
}

func mustTransport(t *testing.T, server *Server, port int, oauth *OAuthServer, opts HttpTransportOptions) *HttpTransport {
	t.Helper()
	transport, err := NewHttpTransport(server, port, oauth, opts)
	expectNoError(t, err)
	t.Cleanup(func() {
		if transport.FileLinkStore != nil {
			transport.FileLinkStore.StopCleanup()
		}
	})
	return transport
}

// startTransport は実ポートで待ち受け、ポート番号を返す。
func startTransport(t *testing.T, transport *HttpTransport) int {
	t.Helper()
	expectNoError(t, transport.Start())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = transport.Stop(ctx)
	})
	return transport.BoundPort()
}

func httpJSONBody(t *testing.T, res *http.Response) *jsonobj.Object {
	t.Helper()
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	expectNoError(t, err)
	return parseObj(t, string(raw))
}

func postJSON(t *testing.T, target string, body string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, target, strings.NewReader(body))
	expectNoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	res, err := http.DefaultClient.Do(req)
	expectNoError(t, err)
	return res
}

// ---------------------------------------------------------------------------
// 層1: Bearer 無し POST /mcp が 401 (C-01 を単体で殺す)
// ---------------------------------------------------------------------------
func TestHandleRequestBearerAuth(t *testing.T) {
	for _, variant := range serverVariants {
		t.Run("POST /mcp without Bearer returns 401 for "+variant.scope, func(t *testing.T) {
			oauth := makeOAuth(t, variant.scope)
			server := variant.make(createTransportMockClient(), nil)
			transport := mustTransport(t, server, 0, oauth, HttpTransportOptions{})
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			req.RemoteAddr = "127.0.0.1:1"
			res := httptest.NewRecorder()

			transport.ServeHTTP(res, req)

			expectEqual(t, res.Code, 401)
			mustContain(t, res.Header().Get("WWW-Authenticate"), "resource_metadata=")
		})
	}
}

// ---------------------------------------------------------------------------
// 層2: 実ポート統合 (metadata / 認可後 tools/list / initialize が 200)
// ---------------------------------------------------------------------------
func TestHttpTransportOverRealHTTP(t *testing.T) {
	start := func(t *testing.T, scope string, client GkillAPI) (*OAuthServer, int) {
		t.Helper()
		oauth := makeOAuth(t, scope)
		var server *Server
		for _, v := range serverVariants {
			if v.scope == scope {
				server = v.make(client, nil)
			}
		}
		transport := mustTransport(t, server, 0, oauth, HttpTransportOptions{})
		return oauth, startTransport(t, transport)
	}

	t.Run("protected-resource metadata is public and advertises the scope", func(t *testing.T) {
		_, port := start(t, "gkill:read", createTransportMockClient())
		res, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/.well-known/oauth-protected-resource", port))
		expectNoError(t, err)
		expectEqual(t, res.StatusCode, 200)
		body := httpJSONBody(t, res)
		expectEqual(t, body.Value("scopes_supported"), strs("gkill:read"))
	})

	t.Run("POST /mcp without Bearer returns 401 over HTTP", func(t *testing.T) {
		_, port := start(t, "gkill:read", createTransportMockClient())
		res := postJSON(t, fmt.Sprintf("http://127.0.0.1:%d/mcp", port), `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, nil)
		defer res.Body.Close()
		expectEqual(t, res.StatusCode, 401)
		mustContain(t, res.Header.Get("Www-Authenticate"), "resource_metadata=")
	})

	t.Run("authorized initialize returns serverInfo", func(t *testing.T) {
		oauth, port := start(t, "gkill:read", createTransportMockClient())
		clientID := registerClient(t, oauth)
		token := mintAccessToken(t, oauth, "admin", "gkill:read", clientID)
		res := postJSON(t, fmt.Sprintf("http://127.0.0.1:%d/mcp", port), `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, map[string]string{"Authorization": "Bearer " + token})
		expectEqual(t, res.StatusCode, 200)
		body := httpJSONBody(t, res)
		expectTrue(t, jsTruthy(objAt(t, body, "result", "serverInfo").Value("name")), "serverInfo.name missing")
		expectEqual(t, objAt(t, body, "result").Value("protocolVersion"), "2024-11-05")
	})
}

// ---------------------------------------------------------------------------
// 層2c: scope 境界 (2026-08-30 レビュー P0)
// ---------------------------------------------------------------------------
func TestOAuthScopeBoundary(t *testing.T) {
	for _, variant := range serverVariants {
		t.Run("both metadata endpoints advertise exactly ["+variant.scope+"]", func(t *testing.T) {
			oauth := makeOAuth(t, variant.scope)
			server := variant.make(createTransportMockClient(), nil)
			transport := mustTransport(t, server, 0, oauth, HttpTransportOptions{})
			port := startTransport(t, transport)
			prmRes, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/.well-known/oauth-protected-resource", port))
			expectNoError(t, err)
			prm := httpJSONBody(t, prmRes)
			asmRes, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/.well-known/oauth-authorization-server", port))
			expectNoError(t, err)
			asm := httpJSONBody(t, asmRes)
			// 元指摘: ReadWrite で protected-resource=gkill:readwrite / authorization-server=gkill:read。
			expectEqual(t, prm.Value("scopes_supported"), strs(variant.scope))
			expectEqual(t, asm.Value("scopes_supported"), strs(variant.scope))
		})
	}

	t.Run("a legacy token with a foreign scope gets 403 insufficient_scope", func(t *testing.T) {
		oauth := makeOAuth(t, "gkill:readwrite")
		server := NewReadWriteServer(createTransportMockClient(), nil)
		// 403 の監査イベントは運用者が scope 拒否を知る唯一の窓なので、応答と一緒に固定する。
		logger, recorder := newRecordingLogger()
		server.Log = logger
		transport := mustTransport(t, server, 0, oauth, HttpTransportOptions{})
		// scope 修正前の ReadWrite サーバが発行した "gkill:read" のアクセストークンを再現する。
		oauth.Store.PutAccessToken("legacy-access", obj(
			"clientId", "c",
			"scope", "gkill:read",
			"gkillSessionId", "sess-x",
			"userId", "u",
		))
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set("Authorization", "Bearer legacy-access")
		req.RemoteAddr = "127.0.0.1:1"
		res := httptest.NewRecorder()
		transport.ServeHTTP(res, req)
		expectEqual(t, res.Code, 403)
		expectEqual(t, parseObj(t, res.Body.String()).Value("error"), "insufficient_scope")
		// 401 (トークン無効) と区別し、クライアントへ再認可を促すヘッダを返す。
		mustContain(t, res.Header().Get("WWW-Authenticate"), `error="insufficient_scope"`)
		mustContain(t, res.Header().Get("WWW-Authenticate"), `scope="gkill:readwrite"`)
		// どの scope のトークンが何を要求されて拒否されたかまでログに残る。
		record, ok := recorder.find("token_scope_rejected")
		expectTrue(t, ok, "token_scope_rejected not logged")
		expectEqual(t, recorder.count("token_scope_rejected"), 1)
		expectEqual(t, record.Attrs.Value("token_scope"), "gkill:read")
		expectEqual(t, record.Attrs.Value("required_scope"), "gkill:readwrite")
	})

	t.Run("a matching-scope token still reaches the MCP handler", func(t *testing.T) {
		oauth := makeOAuth(t, "gkill:readwrite")
		server := NewReadWriteServer(createTransportMockClient(), nil)
		transport := mustTransport(t, server, 0, oauth, HttpTransportOptions{})
		oauth.Store.PutAccessToken("good-access", obj(
			"clientId", "c",
			"scope", "gkill:readwrite",
			"gkillSessionId", "sess-x",
			"userId", "u",
		))
		req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		req.Header.Set("Authorization", "Bearer good-access")
		req.Header.Set("Accept", "application/json")
		req.RemoteAddr = "127.0.0.1:1"
		res := httptest.NewRecorder()
		transport.ServeHTTP(res, req)
		// 認証は通過し、SSE の Accept ヘッダ検査 (406) まで到達している = 401/403 で弾かれていない。
		expectEqual(t, res.Code, 406)
	})

	t.Run("mismatched transport/oauth scopes fail fast at construction", func(t *testing.T) {
		oauth := makeOAuth(t, "gkill:read")
		_, err := NewHttpTransport(NewReadWriteServer(createTransportMockClient(), nil), 0, oauth, HttpTransportOptions{Scope: "gkill:readwrite"})
		expectErrorContains(t, err, "scope mismatch")
	})

	t.Run("a transport with no scope source at all fails fast at construction", func(t *testing.T) {
		// oauthServer が scope を持たないフェイクで、options.Scope も無い組み合わせ。
		// ここで通すと Bearer 受理の scope 照合が空文字列比較になり、全トークンを 403 で弾く事故になる。
		_, err := NewHttpTransport(NewReadWriteServer(createTransportMockClient(), nil), 0, &OAuthServer{Issuer: "https://mcp.example.test"}, HttpTransportOptions{})
		expectErrorContains(t, err, "requires a scope")
	})
}

// ---------------------------------------------------------------------------
// 層2b: 並行分離 (C-02 regression) — 2トークン同時 tools/call で session が混ざらない
// ---------------------------------------------------------------------------
func TestConcurrentRequestsKeepSessionContextSeparate(t *testing.T) {
	t.Run("two in-flight tool calls each see their own token's session", func(t *testing.T) {
		// callApi は両リクエストが揃うまで待つバリアで、意図的に処理を重ねる。
		// 共有フィールド方式に退行すると、この重なりの窓で sid が上書きされる。
		var mu sync.Mutex
		inFlight := 0
		barrier := make(chan struct{})
		seenSids := []string{}
		client := createTransportMockClient()
		client.callApi = func(_ string, _ *jsonobj.Object, _ bool, sid string) (*jsonobj.Object, error) {
			mu.Lock()
			seenSids = append(seenSids, sid)
			inFlight++
			if inFlight == 2 {
				close(barrier)
			}
			mu.Unlock()
			<-barrier
			// sid をレスポンスに反映させ、リクエストと応答の対応も検証できるようにする。
			return obj("errors", arr(), "messages", arr(), "tag_names", strs(sid)), nil
		}

		oauth := makeOAuth(t, "gkill:read")
		transport := mustTransport(t, NewReadServer(client, nil), 0, oauth, HttpTransportOptions{})
		port := startTransport(t, transport)

		clientID := registerClient(t, oauth)
		tokenA := mintAccessToken(t, oauth, "alice", "gkill:read", clientID)
		tokenB := mintAccessToken(t, oauth, "bob", "gkill:read", clientID)

		call := func(token string) *jsonobj.Object {
			res := postJSON(t, fmt.Sprintf("http://127.0.0.1:%d/mcp", port),
				`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"gkill_get_all_tag_names","arguments":{}}}`,
				map[string]string{"Authorization": "Bearer " + token})
			return httpJSONBody(t, res)
		}

		var resA, resB *jsonobj.Object
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); resA = call(tokenA) }()
		go func() { defer wg.Done(); resB = call(tokenB) }()
		wg.Wait()

		// 各リクエストは自分のトークンのセッションで gkill を叩いている。
		expectEqual(t, objAt(t, resA, "result", "structuredContent").Value("tag_names"), strs("sess-alice"))
		expectEqual(t, objAt(t, resB, "result", "structuredContent").Value("tag_names"), strs("sess-bob"))
		// 両セッションが1回ずつ現れ、取り違えが無い。
		expectEqual(t, sortedCopy(seenSids), []string{"sess-alice", "sess-bob"})
	})
}

// ---------------------------------------------------------------------------
// M-06: 公開ファイル配信に nosniff / CSP sandbox が付く
// ---------------------------------------------------------------------------
func TestHandleFileServeSecurityHeaders(t *testing.T) {
	buildTransport := func(t *testing.T, fetchFile func(string, string) (*FileResponse, error)) *HttpTransport {
		t.Helper()
		fakeServer := &Server{Client: &mockClient{fetchFile: fetchFile}}
		return mustTransport(t, fakeServer, 0, &OAuthServer{Issuer: "https://mcp.example.test"}, HttpTransportOptions{Scope: "gkill:read", EnableFileLinks: true})
	}
	newReq := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/files/x", nil)
		req.RemoteAddr = "127.0.0.1:1"
		return req
	}

	t.Run("image/HTML files get nosniff and CSP sandbox", func(t *testing.T) {
		transport := buildTransport(t, func(_ string, _ string) (*FileResponse, error) {
			return &FileResponse{Buffer: []byte{1}, ContentType: "text/html"}, nil
		})
		token := transport.FileLinkStore.Mint(FileLink{GkillSessionID: "s", RepName: "Files", FileName: "note.html", IsImage: false})
		res := httptest.NewRecorder()
		transport.HandleFileServe(res, newReq(), token, url.Values{})
		expectEqual(t, res.Code, 200)
		expectEqual(t, res.Header().Get("X-Content-Type-Options"), "nosniff")
		expectEqual(t, res.Header().Get("Content-Security-Policy"), "sandbox")
	})

	t.Run("PDF gets nosniff but no sandbox (built-in viewer needs a real origin)", func(t *testing.T) {
		transport := buildTransport(t, func(_ string, _ string) (*FileResponse, error) {
			return &FileResponse{Buffer: []byte{1}, ContentType: "application/pdf"}, nil
		})
		token := transport.FileLinkStore.Mint(FileLink{GkillSessionID: "s", RepName: "Files", FileName: "doc.pdf", IsImage: false})
		res := httptest.NewRecorder()
		transport.HandleFileServe(res, newReq(), token, url.Values{})
		expectEqual(t, res.Code, 200)
		expectEqual(t, res.Header().Get("X-Content-Type-Options"), "nosniff")
		expectEqual(t, res.Header().Get("Content-Security-Policy"), "")
	})
}

// ---------------------------------------------------------------------------
// 2026-08-30 監査 F-003 / F-004: ボディ上限・明示タイムアウト・ログのクエリ除去
// ---------------------------------------------------------------------------
func TestBodyCapsExplicitTimeoutsAndLogQueryRedaction(t *testing.T) {
	t.Run("collectBody cuts an over-limit body with 413 and resolves null", func(t *testing.T) {
		oauth := makeOAuth(t, "gkill:read")
		server := NewReadServer(createTransportMockClient(), nil)
		logger, _ := newRecordingLogger()
		server.Log = logger
		transport := mustTransport(t, server, 0, oauth, HttpTransportOptions{})
		req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader("123456789")) // 9 bytes > limit 8
		req.RemoteAddr = "127.0.0.1:1"
		res := httptest.NewRecorder()

		rawBody, ok := transport.collectBody(res, req, 8, "test_body_too_large")

		expectTrue(t, rawBody == nil && !ok, "over-limit body was accepted")
		expectEqual(t, res.Code, 413)
	})

	t.Run("collectBody passes an at-limit body through unchanged", func(t *testing.T) {
		oauth := makeOAuth(t, "gkill:read")
		server := NewReadServer(createTransportMockClient(), nil)
		logger, _ := newRecordingLogger()
		server.Log = logger
		transport := mustTransport(t, server, 0, oauth, HttpTransportOptions{})
		req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader("12345678")) // exactly the limit
		req.RemoteAddr = "127.0.0.1:1"
		res := httptest.NewRecorder()

		rawBody, ok := transport.collectBody(res, req, 8, "test_body_too_large")

		expectTrue(t, ok, "at-limit body was rejected")
		expectEqual(t, string(rawBody), "12345678")
		expectTrue(t, res.Body.Len() == 0, "a response was written")
	})

	t.Run("POST /oauth/register over the cap is rejected before handleRegister (route wiring)", func(t *testing.T) {
		oauth := makeOAuth(t, "gkill:read")
		server := NewReadServer(createTransportMockClient(), nil)
		logger, recorder := newRecordingLogger()
		server.Log = logger
		transport := mustTransport(t, server, 0, oauth, HttpTransportOptions{})
		port := startTransport(t, transport)

		body := bytes.Repeat([]byte{0x61}, 64*1024+16)
		res, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/oauth/register", port), "application/json", bytes.NewReader(body))
		if err == nil {
			res.Body.Close()
		}

		waitUntil(t, func() bool {
			r, ok := recorder.find("http_request")
			if !ok {
				return false
			}
			return jsonobj.Equal(r.Attrs.Value("reason"), "oauth_register_body_too_large") && jsonobj.Equal(r.Attrs.Value("status"), 413)
		})
		expectEqual(t, oauth.Store.Stats().Clients, 0)
	})

	t.Run("authorize query values (client_id/state/code_challenge) never reach the access log", func(t *testing.T) {
		oauth := makeOAuth(t, "gkill:read")
		server := NewReadServer(createTransportMockClient(), nil)
		logger, recorder := newRecordingLogger()
		server.Log = logger
		transport := mustTransport(t, server, 0, oauth, HttpTransportOptions{})

		req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=SECRETCID&redirect_uri=https%3A%2F%2Fx%2FSECRETURI&state=SECRETSTATE&code_challenge=SECRETCC", nil)
		req.RemoteAddr = "127.0.0.1:1"
		transport.logRequest(req, logFields{StatusCode: 200, Reason: "oauth_authorize_get"})

		recorder.mu.Lock()
		records := append([]logRecord{}, recorder.records...)
		recorder.mu.Unlock()
		expectEqual(t, len(records), 1)
		expectEqual(t, records[0].Attrs.Value("path"), "/oauth/authorize")
		encoded, err := json.Marshal(records[0].Attrs.Keys())
		expectNoError(t, err)
		mustNotContain(t, string(encoded)+jsonobj.MarshalString(records[0].Attrs), "SECRET")
	})

	t.Run("explicit timeouts and MCP_BIND_ADDR are applied at start()", func(t *testing.T) {
		t.Setenv("MCP_BIND_ADDR", "127.0.0.1")
		oauth := makeOAuth(t, "gkill:read")
		server := NewReadServer(createTransportMockClient(), nil)
		transport := mustTransport(t, server, 0, oauth, HttpTransportOptions{})
		startTransport(t, transport)

		expectEqual(t, int64(transport.HTTPServer().ReadHeaderTimeout), int64(20*time.Second))
		expectEqual(t, int64(transport.HTTPServer().ReadTimeout), int64(5*time.Minute))
		mustContain(t, transport.Addr().String(), "127.0.0.1:")
	})
}
