package mcp

// 編集前に読む: .claude/skills/gkill-mcp/SKILL.md
//
// Streamable HTTP トランスポート (MCP spec 2024-11-05)。3つの MCP サーバで共有する（旧 http-transport.mjs）。
//
// POST /mcp (リクエスト) / GET /mcp (SSE) / DELETE /mcp (セッション終了) に加えて、
// OAuth 2.1 のエンドポイントと、リモートクライアント向けのファイル配信 /files/{token} を持つ。
//
// サーバごとに違うのは OAuth のスコープ名と、ファイル配信ルートを載せるかどうかの2点だけなので、
// その2つだけをコンストラクタの options で受ける。

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// リクエストボディの経路別上限。上限なしの読み込みは、無認証で到達できる
// OAuth 経路からプロセスのメモリを枯渇させられる (指摘 F-004)。
// MCP POST は Bearer 必須だが、有効な資格情報を持つクライアントにも上限は掛ける。
const maxMcpBodyBytes = 10 * 1024 * 1024 // 10MB
const maxOAuthBodyBytes = 64 * 1024      // 64KB (フォーム/JSON の認可・トークン・登録)

// ログへ書くパスはクエリを落とす。/oauth/authorize は client_id / redirect_uri /
// state / code_challenge をクエリで受けるため、生の URL を記録すると
// 認可フローの秘匿値がアクセスログへ残る (指摘 F-003)。
func pathWithoutQuery(rawURL string) string {
	return strings.SplitN(rawURL, "?", 2)[0]
}

// HttpTransportOptions はスコープとファイル配信ルートの有無。
type HttpTransportOptions struct {
	// Scope は OAuth のスコープ名。正本は OAuthServer.Scope で、通常ここでは渡さない。
	// oauthServer が scope を持たないフェイク (テスト) のときだけ使われる。
	Scope string
	// EnableFileLinks は /files/{token} の配信ルートを載せるか。
	// ファイル系ツールを持たない書き込み専用サーバでは false。
	EnableFileLinks bool
	// BindAddr は待ち受けアドレス。空なら MCP_BIND_ADDR、それも無ければ 0.0.0.0。
	BindAddr string
}

// HttpTransport は /mcp・/oauth/*・/files/* を受ける HTTP サーバ。
type HttpTransport struct {
	Server          *Server
	Port            int
	OAuth           *OAuthServer
	Scope           string
	EnableFileLinks bool
	FileLinkStore   *FileLinkStore
	BindAddr        string

	httpServer *http.Server
	listener   net.Listener
}

// NewHttpTransport はトランスポートを作る（ポートはまだ開かない）。
func NewHttpTransport(server *Server, port int, oauth *OAuthServer, opts HttpTransportOptions) (*HttpTransport, error) {
	if oauth == nil {
		return nil, errors.New("HttpTransport requires an OAuthServer")
	}
	// scope は OAuthServer と共有の1値。以前は transport と OAuthServer が別々の値を
	// 持てたため、protected-resource と authorization-server の広告が矛盾していた
	// (2026-08-30 レビュー P0)。二重指定の不一致は静かに広告し続けず起動時に落とす。
	if opts.Scope != "" && oauth.Scope != "" && opts.Scope != oauth.Scope {
		return nil, fmt.Errorf("OAuth scope mismatch: transport=%s oauthServer=%s", opts.Scope, oauth.Scope)
	}
	scope := oauth.Scope
	if scope == "" {
		scope = opts.Scope
	}
	if scope == "" {
		return nil, errors.New("HttpTransport requires a scope (via OAuthServer or options)")
	}
	t := &HttpTransport{
		Server:          server,
		Port:            port,
		OAuth:           oauth,
		Scope:           scope,
		EnableFileLinks: opts.EnableFileLinks,
		BindAddr:        opts.BindAddr,
	}
	// HTTP 越しのクライアントは別マシン (例: クラウド上の AI) でありうる。
	// この MCP サーバ自身が gkill と同居していても、絶対パスを渡してよい相手ではない。
	server.IsLocalTransport = false
	if t.EnableFileLinks {
		// リモートクライアントには実パスの代わりに期限付きの公開ファイル URL を渡す。
		// issuer は MCP_OAUTH_ISSUER (公開 URL) で、この配信ルート自身の基点になる。
		t.FileLinkStore = NewFileLinkStore(0)
		t.FileLinkStore.StartCleanup(0)
		server.FileLinkContext = &FileLinkContext{PublicBaseURL: oauth.Issuer, Store: t.FileLinkStore}
	}
	return t, nil
}

// HTTPServer は待ち受け中の http.Server（テストがタイムアウト設定を確かめる）。
func (t *HttpTransport) HTTPServer() *http.Server { return t.httpServer }

// Addr は待ち受けアドレス（Start 後）。
func (t *HttpTransport) Addr() net.Addr {
	if t.listener == nil {
		return nil
	}
	return t.listener.Addr()
}

// BoundPort は実際に開いたポート（Port が 0 なら OS が採番したもの）。
func (t *HttpTransport) BoundPort() int {
	if addr, ok := t.Addr().(*net.TCPAddr); ok {
		return addr.Port
	}
	return t.Port
}

// Start はポートを開いて待ち受けを始める（非同期）。
func (t *HttpTransport) Start() error {
	bindAddr := t.BindAddr
	if bindAddr == "" {
		// bind 先は MCP_BIND_ADDR で絞れる (例: リバースプロキシ/トンネル併用時は 127.0.0.1)。
		// 既定は互換のため全インターフェース待受のまま。
		bindAddr = os.Getenv("MCP_BIND_ADDR")
	}
	if bindAddr == "" {
		bindAddr = "0.0.0.0"
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(bindAddr, strconv.Itoa(t.Port)))
	if err != nil {
		return err
	}
	t.listener = listener
	// ヘッダとリクエスト全体の期限を明示する。スローなヘッダ送信と
	// 終わらないリクエストをここで打ち切る (指摘 F-004)。
	t.httpServer = &http.Server{
		Handler:           t,
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       5 * time.Minute,
	}
	go func() {
		_ = t.httpServer.Serve(listener)
	}()
	boundPort := t.BoundPort()
	// 手で起動したときに見えるよう stderr へも出すが、ログにも1行残す。
	t.Server.Log.Info("http_listening", "port", boundPort, "issuer", t.OAuth.Issuer)
	fmt.Fprintf(os.Stderr, "MCP HTTP server listening on http://%s:%d/mcp [OAuth issuer: %s]\n", bindAddr, boundPort, t.OAuth.Issuer)
	return nil
}

// Stop は待ち受けを閉じ、file-link の掃除も止める。
func (t *HttpTransport) Stop(ctx context.Context) error {
	if t.FileLinkStore != nil {
		t.FileLinkStore.StopCleanup()
	}
	if t.httpServer != nil {
		return t.httpServer.Shutdown(ctx)
	}
	return nil
}

// collectBody はボディを上限付きで読む。超過したら 413 を返して false。
// ボディを読む4経路 (MCP POST / OAuth authorize POST / token / register) は必ずここを通す。
func (t *HttpTransport) collectBody(w http.ResponseWriter, r *http.Request, limitBytes int64, logReason string) ([]byte, bool) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limitBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			t.logRequest(r, logFields{StatusCode: 413, Reason: logReason})
			t.sendJSON(w, 413, jsonobj.Obj("error", "Payload Too Large"), nil)
		}
		return nil, false
	}
	return body, true
}

type route struct {
	kind     string
	pathname string
	query    map[string]string
	token    string
}

// queryMap は Object.fromEntries(url.searchParams)（同名キーは最後の値が勝つ）。
func queryMap(values url.Values) map[string]string {
	out := map[string]string{}
	for key, list := range values {
		if len(list) > 0 {
			out[key] = list[len(list)-1]
		}
	}
	return out
}

func (t *HttpTransport) parseRoute(r *http.Request) *route {
	pathname := r.URL.Path
	query := queryMap(r.URL.Query())

	// Protected Resource Metadata (RFC 9728)
	if pathname == "/.well-known/oauth-protected-resource" || pathname == "/.well-known/oauth-protected-resource/mcp" {
		return &route{kind: "oauth-protected-resource", pathname: pathname}
	}
	// OAuth Authorization Server Metadata (RFC 8414)
	if pathname == "/.well-known/oauth-authorization-server" {
		return &route{kind: "oauth-metadata", pathname: pathname, query: query}
	}
	// OAuth endpoints — /oauth/* canonical, /* fallback for Claude.ai (known bug: ignores metadata endpoints)
	if pathname == "/oauth/authorize" || pathname == "/authorize" {
		return &route{kind: "oauth-authorize", pathname: pathname, query: query}
	}
	if pathname == "/oauth/token" || pathname == "/token" {
		return &route{kind: "oauth-token", pathname: pathname}
	}
	if pathname == "/oauth/register" || pathname == "/register" {
		return &route{kind: "oauth-register", pathname: pathname}
	}
	// MCP endpoint
	if pathname == "/mcp" {
		return &route{kind: "mcp", pathname: pathname}
	}
	// Public file delivery — token in the path, no Bearer needed (image fetchers
	// cannot send auth headers). The token itself is the security boundary.
	if t.EnableFileLinks && strings.HasPrefix(pathname, "/files/") {
		token, err := url.PathUnescape(pathname[len("/files/"):])
		if err != nil {
			token = pathname[len("/files/"):]
		}
		return &route{kind: "file", pathname: pathname, token: token, query: query}
	}
	return nil
}

// remoteAddrOf は req.socket.remoteAddress 相当（ポートを除いた IP）。
func remoteAddrOf(r *http.Request) any {
	if r.RemoteAddr == "" {
		return nil
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// logFields は logRequest の付加欄。ここに列挙されていないもの（redirect URL 等）はログへ載らない。
type logFields struct {
	StatusCode    int
	Reason        string
	Methods       string
	ResponseBytes *int
}

// logRequest はアクセスログへ1行書く。path はクエリを落とした形だけを記録する。
func (t *HttpTransport) logRequest(r *http.Request, fields logFields) {
	kv := []any{
		"remote_addr", remoteAddrOf(r),
		"method", r.Method,
		"path", pathWithoutQuery(r.URL.RequestURI()),
		"status", fields.StatusCode,
	}
	if fields.Methods != "" {
		kv = append(kv, "methods", fields.Methods)
	}
	if fields.Reason != "" {
		kv = append(kv, "reason", fields.Reason)
	}
	if fields.ResponseBytes != nil {
		kv = append(kv, "response_bytes", *fields.ResponseBytes)
	}
	if fields.StatusCode >= 400 {
		t.Server.Log.Warn("http_request", kv...)
	} else {
		t.Server.Log.Access("http_request", kv...)
	}
}

func intPtr(n int) *int { return &n }

// sendJSON は JSON を書き、本文のバイト数を返す。payload が nil なら本文なし。
func (t *HttpTransport) sendJSON(w http.ResponseWriter, statusCode int, payload any, headers map[string]string) int {
	body := ""
	if payload != nil {
		body = jsonobj.MarshalString(payload)
	}
	if body != "" {
		w.Header().Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		w.Header().Set(key, value)
	}
	w.WriteHeader(statusCode)
	_, _ = io.WriteString(w, body)
	return len(body)
}

func summarizeJsonRpcMethods(payload any) string {
	methodOf := func(item any) string {
		if o, ok := item.(*jsonobj.Object); ok && o != nil && o.Has("method") {
			return jsString(o.Value("method"))
		}
		return "invalid"
	}
	if items, ok := jsonobj.AsArray(payload); ok {
		parts := make([]string, len(items))
		for i, item := range items {
			parts[i] = methodOf(item)
		}
		return strings.Join(parts, ",")
	}
	return methodOf(payload)
}

// HandleFileServe は公開ファイル配信。トークンからファイルを解決し、gkill からバイトを取って返す。
// Bearer 不要 (画像取得は認証ヘッダを付けられない) なので、トークンが唯一の防御線。
func (t *HttpTransport) HandleFileServe(w http.ResponseWriter, r *http.Request, token string, query url.Values) {
	var link FileLink
	ok := false
	if t.Server.FileLinkContext != nil && t.Server.FileLinkContext.Store != nil {
		link, ok = t.Server.FileLinkContext.Store.Resolve(token)
	}
	if !ok {
		t.logRequest(r, logFields{StatusCode: 404, Reason: "file_token_invalid"})
		t.sendJSON(w, 404, jsonobj.Obj("error", "Not Found"), nil)
		return
	}

	segments := strings.Split(link.FileName, "/")
	for i, segment := range segments {
		segments[i] = encodeURIComponent(segment)
	}
	gkillPath := "/files/" + encodeURIComponent(link.RepName) + "/" + strings.Join(segments, "/")
	// サムネ指定は画像のときだけ、かつ WxH 形式に限って gkill に転送する。
	if thumb := query.Get("thumb"); link.IsImage && thumb != "" && ThumbQueryRegex.MatchString(thumb) {
		gkillPath += "?thumb=" + thumb
	}

	file, err := t.Server.Client.FetchFile(r.Context(), gkillPath, link.GkillSessionID)
	if err != nil {
		t.logRequest(r, logFields{StatusCode: 502, Reason: "file_fetch_failed"})
		t.Server.Log.Error("file_fetch_error", "remote_addr", remoteAddrOf(r), "error", err.Error())
		t.sendJSON(w, 502, jsonobj.Obj("error", "Bad Gateway"), nil)
		return
	}
	t.logRequest(r, logFields{StatusCode: 200, ResponseBytes: intPtr(len(file.Buffer))})
	// Mirrors: gkill_server_api/utils.go withUserContentSecurityHeaders。
	// 拡張子許可リストの無い利用者ファイルを、OAuth ログインフォームと同一オリジンで
	// 無認証配信するので、.html/.svg がスクリプト実行経路になりうる。nosniff は常時、
	// CSP sandbox は .pdf 以外に付ける (.pdf は sandbox の opaque origin で
	// Chrome 内蔵 PDF ビューワが動かなくなるため除外)。sandbox は文書読み込み時のみ効き
	// <img>/<video> のサブリソース表示には影響しないので、画像取得の本来用途は壊れない。
	isPdf := strings.HasSuffix(strings.ToLower(link.FileName), ".pdf")
	contentType := NormalizeMimeType(file.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(file.Buffer)))
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !isPdf {
		w.Header().Set("Content-Security-Policy", "sandbox")
	}
	w.WriteHeader(200)
	_, _ = w.Write(file.Buffer)
}

// ServeHTTP はルート表に従って各ハンドラへ振り分ける（旧 handleRequest）。
func (t *HttpTransport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt := t.parseRoute(r)
	if rt == nil {
		t.logRequest(r, logFields{StatusCode: 404, Reason: "route_not_found"})
		t.sendJSON(w, 404, jsonobj.Obj("error", "Not Found. Use POST /mcp"), nil)
		return
	}

	// OAuth discovery/auth endpoints — no Bearer auth required
	switch rt.kind {
	case "oauth-protected-resource":
		t.handleProtectedResourceMetadata(w, r)
		return
	case "oauth-metadata":
		t.handleOAuthMetadata(w, r)
		return
	case "oauth-authorize":
		t.handleOAuthAuthorize(w, r, rt.query)
		return
	case "oauth-token":
		t.handleOAuthToken(w, r)
		return
	case "oauth-register":
		t.handleOAuthRegister(w, r)
		return
	case "file":
		// Public file delivery — Bearer 不要。トークンが防御線。
		if r.Method == http.MethodOptions {
			t.logRequest(r, logFields{StatusCode: 204})
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.WriteHeader(204)
			return
		}
		if r.Method != http.MethodGet {
			t.logRequest(r, logFields{StatusCode: 405, Reason: "method_not_allowed"})
			t.sendJSON(w, 405, jsonobj.Obj("error", "Method Not Allowed"), map[string]string{"Allow": "GET, OPTIONS"})
			return
		}
		t.HandleFileServe(w, r, rt.token, r.URL.Query())
		return
	}

	// MCP endpoint — require OAuth Bearer token
	bearerToken := ExtractBearerToken(r.Header.Get("Authorization"))
	var tokenData *jsonobj.Object
	if bearerToken != "" {
		tokenData, _ = t.OAuth.ValidateAccessToken(bearerToken)
	}
	resourceMetadataURL := t.OAuth.Issuer + "/.well-known/oauth-protected-resource"

	if tokenData == nil {
		t.logRequest(r, logFields{StatusCode: 401, Reason: "unauthorized"})
		t.Server.Log.Warn("token_rejected",
			"remote_addr", remoteAddrOf(r),
			"method", r.Method, "path", pathWithoutQuery(r.URL.RequestURI()),
		)
		t.sendJSON(w, 401, jsonobj.Obj(
			"error", "Unauthorized",
			"error_description", "Bearer token required",
		), map[string]string{"WWW-Authenticate": `Bearer resource_metadata="` + resourceMetadataURL + `"`})
		return
	}

	// scope はこのサーバの唯一の許可値と厳密一致のみ受理する (RFC 6750 insufficient_scope)。
	// 以前はトークンの存在だけを見ていたため、scope 修正前の ReadWrite サーバが発行した
	// "gkill:read" トークンでも書き込みツールを呼べた (2026-08-30 レビュー P0)。
	// 401 ではなく 403 を返すのは「トークンは本物だが権限が足りない = 再認可が要る」を
	// クライアントに伝えるため。
	tokenScope, hasScope := tokenData.String("scope")
	if !hasScope || tokenScope != t.Scope {
		var tokenScopeValue any
		if hasScope {
			tokenScopeValue = tokenScope
		}
		t.logRequest(r, logFields{StatusCode: 403, Reason: "insufficient_scope"})
		t.Server.Log.Warn("token_scope_rejected",
			"remote_addr", remoteAddrOf(r),
			"method", r.Method, "path", pathWithoutQuery(r.URL.RequestURI()),
			"token_scope", tokenScopeValue, "required_scope", t.Scope,
		)
		t.sendJSON(w, 403, jsonobj.Obj(
			"error", "insufficient_scope",
			"error_description", `This server requires scope "`+t.Scope+`". Re-authorize to obtain it.`,
		), map[string]string{"WWW-Authenticate": `Bearer error="insufficient_scope", scope="` + t.Scope + `", resource_metadata="` + resourceMetadataURL + `"`})
		return
	}

	// 1リクエスト分の認証文脈は不変の値に固めて handlePost へ渡す。
	// 以前は server.currentUserId 等の共有フィールドに書いて await をまたいで
	// 読んでいたため、並行リクエストで別要求の user/session が混線していた。
	sessionID, _ := tokenData.String("gkillSessionId")
	userID, _ := tokenData.String("userId")
	requestContext := RequestContext{SessionID: sessionID, UserID: userID, RemoteAddr: jsString(remoteAddrOf(r))}
	if remoteAddrOf(r) == nil {
		requestContext.RemoteAddr = ""
	}
	switch r.Method {
	case http.MethodPost:
		t.handlePost(w, r, requestContext)
	case http.MethodGet:
		t.handleGet(w, r)
	case http.MethodDelete:
		t.handleDelete(w, r)
	default:
		t.logRequest(r, logFields{StatusCode: 405, Reason: "method_not_allowed"})
		t.sendJSON(w, 405, jsonobj.Obj("error", "Method Not Allowed"), map[string]string{"Allow": "GET, POST, DELETE"})
	}
}

func (t *HttpTransport) handlePost(w http.ResponseWriter, r *http.Request, requestContext RequestContext) {
	rawBody, ok := t.collectBody(w, r, maxMcpBodyBytes, "mcp_body_too_large")
	if !ok {
		return
	}
	payload, err := jsonobj.Unmarshal(rawBody)
	if err != nil {
		t.logRequest(r, logFields{StatusCode: 400, Reason: "parse_error"})
		t.sendJSON(w, 400, jsonobj.Obj("jsonrpc", "2.0", "id", nil, "error", jsonobj.Obj("code", -32700, "message", "Parse error")), nil)
		return
	}

	response, handlerErr := t.handlePayloadSafely(r.Context(), payload, requestContext)
	methods := summarizeJsonRpcMethods(payload)
	if handlerErr != nil {
		// ここは真の内部例外。ログを通さないと gkill_mcp_<kind>_error.log に残らない。
		t.Server.Log.Error("http_handler_error",
			"method", r.Method,
			"path", pathWithoutQuery(r.URL.RequestURI()),
			"error", handlerErr.Error(),
		)
		var id any
		if o, isObject := payload.(*jsonobj.Object); isObject && o != nil && o.Has("id") {
			id = o.Value("id")
		}
		responseBytes := t.sendJSON(w, 200, jsonobj.Obj("jsonrpc", "2.0", "id", id, "error", jsonobj.Obj("code", -32603, "message", "Internal error")), nil)
		t.logRequest(r, logFields{Methods: methods, StatusCode: 200, ResponseBytes: intPtr(responseBytes), Reason: "internal_error"})
		return
	}
	if response == nil {
		t.logRequest(r, logFields{Methods: methods, StatusCode: 202, ResponseBytes: intPtr(0)})
		w.WriteHeader(202)
		return
	}
	responseBytes := t.sendJSON(w, 200, response, nil)
	t.logRequest(r, logFields{Methods: methods, StatusCode: 200, ResponseBytes: intPtr(responseBytes)})
}

// handlePayloadSafely はハンドラの panic を内部エラーとして拾う（Node の try/catch 相当）。
func (t *HttpTransport) handlePayloadSafely(ctx context.Context, payload any, requestContext RequestContext) (response any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			response = nil
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return t.Server.HandlePayload(ctx, payload, &requestContext), nil
}

func (t *HttpTransport) handleGet(w http.ResponseWriter, r *http.Request) {
	// SSE endpoint for server-initiated notifications.
	// Currently gkill has no server-push notifications, so just hold the connection open.
	if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		t.logRequest(r, logFields{StatusCode: 406, Reason: "missing_sse_accept_header"})
		t.sendJSON(w, 406, jsonobj.Obj("error", "Not Acceptable. Use Accept: text/event-stream"), nil)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(200)
	flusher, _ := w.(http.Flusher)
	if flusher != nil {
		flusher.Flush()
	}
	t.logRequest(r, logFields{StatusCode: 200, Reason: "sse_open"})
	// Keep connection alive with periodic comments
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			t.logRequest(r, logFields{StatusCode: 200, Reason: "sse_closed"})
			return
		case <-ticker.C:
			if _, err := io.WriteString(w, ": keepalive\n\n"); err != nil {
				t.logRequest(r, logFields{StatusCode: 200, Reason: "sse_closed"})
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

func (t *HttpTransport) handleDelete(w http.ResponseWriter, r *http.Request) {
	// Stateless mode: DELETE is accepted as a no-op for clients that still send session cleanup.
	responseBytes := t.sendJSON(w, 200, jsonobj.Obj("ok", true), nil)
	t.logRequest(r, logFields{StatusCode: 200, ResponseBytes: intPtr(responseBytes), Reason: "stateless_delete_noop"})
}

// --- OAuth endpoint handlers ---

func (t *HttpTransport) handleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		t.sendJSON(w, 405, jsonobj.Obj("error", "Method Not Allowed"), map[string]string{"Allow": "GET"})
		return
	}
	issuer := t.OAuth.Issuer
	body := jsonobj.Obj(
		"resource", issuer+"/mcp",
		"authorization_servers", jsonobj.Strings(issuer),
		"scopes_supported", jsonobj.Strings(t.Scope),
		"bearer_methods_supported", jsonobj.Strings("header"),
	)
	t.sendJSON(w, 200, body, nil)
	t.logRequest(r, logFields{StatusCode: 200, Reason: "oauth_protected_resource"})
}

func (t *HttpTransport) handleOAuthMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		t.sendJSON(w, 405, jsonobj.Obj("error", "Method Not Allowed"), map[string]string{"Allow": "GET"})
		return
	}
	t.sendJSON(w, 200, t.OAuth.GetMetadata(), nil)
	t.logRequest(r, logFields{StatusCode: 200, Reason: "oauth_metadata"})
}

// formMap は Object.fromEntries(new URLSearchParams(body))（同名キーは最後の値が勝つ）。
func formMap(raw string) (map[string]string, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, err
	}
	return queryMap(values), nil
}

func (t *HttpTransport) handleOAuthAuthorize(w http.ResponseWriter, r *http.Request, query map[string]string) {
	switch r.Method {
	case http.MethodGet:
		result := t.OAuth.HandleAuthorizeGet(query)
		t.sendOAuthResult(w, r, result, "oauth_authorize_get")
		return
	case http.MethodPost:
		rawBody, ok := t.collectBody(w, r, maxOAuthBodyBytes, "oauth_authorize_body_too_large")
		if !ok {
			return
		}
		formData, err := formMap(string(rawBody))
		if err != nil {
			t.Server.Log.Error("oauth_authorize_error", "error", err.Error())
			t.sendJSON(w, 500, jsonobj.Obj("error", "Internal Server Error"), nil)
			return
		}
		result := t.OAuth.HandleAuthorizePost(r.Context(), formData)
		t.sendOAuthResult(w, r, result, "oauth_authorize_post")
		return
	}
	t.sendJSON(w, 405, jsonobj.Obj("error", "Method Not Allowed"), map[string]string{"Allow": "GET, POST"})
}

// tokenBodyMap は JSON 本文を文字列の表にする（null / undefined は無いものとして扱う）。
func tokenBodyMap(parsed any) map[string]string {
	out := map[string]string{}
	if o, ok := parsed.(*jsonobj.Object); ok && o != nil {
		for _, key := range o.Keys() {
			value := o.Value(key)
			if value == nil || jsonobj.IsUndefined(value) {
				continue
			}
			out[key] = jsString(value)
		}
	}
	return out
}

func (t *HttpTransport) handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		t.sendJSON(w, 405, jsonobj.Obj("error", "Method Not Allowed"), map[string]string{"Allow": "POST"})
		return
	}
	rawBody, ok := t.collectBody(w, r, maxOAuthBodyBytes, "oauth_token_body_too_large")
	if !ok {
		return
	}
	// Token endpoint accepts both application/x-www-form-urlencoded and application/json
	var body map[string]string
	var err error
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var parsed any
		parsed, err = jsonobj.Unmarshal(rawBody)
		if err == nil {
			body = tokenBodyMap(parsed)
		}
	} else {
		body, err = formMap(string(rawBody))
	}
	if err != nil {
		t.Server.Log.Error("oauth_token_error", "error", err.Error())
		t.sendJSON(w, 500, jsonobj.Obj("error", "server_error", "error_description", "Internal Server Error"), nil)
		return
	}
	result := t.OAuth.HandleTokenRequest(body)
	t.sendJSON(w, result.Status, result.JSON, nil)
	t.logRequest(r, logFields{StatusCode: result.Status, Reason: "oauth_token"})
}

func (t *HttpTransport) handleOAuthRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		t.sendJSON(w, 405, jsonobj.Obj("error", "Method Not Allowed"), map[string]string{"Allow": "POST"})
		return
	}
	rawBody, ok := t.collectBody(w, r, maxOAuthBodyBytes, "oauth_register_body_too_large")
	if !ok {
		return
	}
	body, err := jsonobj.Unmarshal(rawBody)
	if err != nil {
		t.Server.Log.Warn("oauth_register_invalid_request", "error", err.Error())
		t.sendJSON(w, 400, jsonobj.Obj("error", "invalid_client_metadata", "error_description", "Invalid JSON"), nil)
		return
	}
	result := t.OAuth.HandleRegister(body)
	t.sendJSON(w, result.Status, result.JSON, nil)
	t.logRequest(r, logFields{StatusCode: result.Status, Reason: "oauth_register"})
}

// sendOAuthResult は OAuth の結果 (HTML か JSON) を送る。
func (t *HttpTransport) sendOAuthResult(w http.ResponseWriter, r *http.Request, result OAuthResult, reason string) {
	if result.ContentType == "text/html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(result.Status)
		_, _ = io.WriteString(w, result.HTML)
		t.logRequest(r, logFields{StatusCode: result.Status, Reason: reason})
		return
	}
	t.sendJSON(w, result.Status, result.JSON, nil)
	t.logRequest(r, logFields{StatusCode: result.Status, Reason: reason})
}

// encodeURIComponent は JS の encodeURIComponent（A-Z a-z 0-9 - _ . ! ~ * ' ( ) 以外を %XX に）。
func encodeURIComponent(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || strings.IndexByte("-_.!~*'()", c) >= 0 {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}
