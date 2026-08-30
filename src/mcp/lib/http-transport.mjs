// Streamable HTTP トランスポート (MCP spec 2024-11-05)。3つのMCPサーバで共有する。
//
// POST /mcp (リクエスト) / GET /mcp (SSE) / DELETE /mcp (セッション終了) に加えて、
// OAuth 2.1 のエンドポイントと、リモートクライアント向けのファイル配信 /files/{token} を持つ。
//
// 以前は3本に逐語コピーされていた。サーバごとに違うのは
// OAuth のスコープ名と、ファイル配信ルートを載せるかどうかの2点だけなので、
// その2つだけをコンストラクタの options で受ける。

import http from "node:http";
import { OAuthServer } from "./oauth-server.mjs";
import { FileLinkStore } from "./file-link-store.mjs";
import { THUMB_QUERY_REGEX, normalizeMimeType } from "./payload.mjs";

// リクエストボディの経路別上限。上限なしの Buffer.concat は、無認証で到達できる
// OAuth 経路からプロセスのメモリを枯渇させられる (2026-08-30 監査 F-004)。
// MCP POST は Bearer 必須だが、有効な資格情報を持つクライアントにも上限は掛ける。
const MAX_MCP_BODY_BYTES = 10 * 1024 * 1024; // 10MB
const MAX_OAUTH_BODY_BYTES = 64 * 1024; // 64KB (フォーム/JSON の認可・トークン・登録)

// ログへ書くパスはクエリを落とす。/oauth/authorize は client_id / redirect_uri /
// state / code_challenge をクエリで受けるため、生の req.url を記録すると
// 認可フローの秘匿値がアクセスログへ残る (2026-08-30 監査 F-003)。
function pathWithoutQuery(url) {
  return String(url || "").split("?")[0];
}

// HttpTransport: Streamable HTTP transport (MCP spec 2024-11-05).
// Supports POST /mcp (requests), GET /mcp (SSE stream), DELETE /mcp (session end).
// OAuth 2.1 endpoints for ChatGPT and Claude.ai MCP connectors.
export class HttpTransport {
  /**
   * @param {object} server MCPサーバ本体
   * @param {number} port
   * @param {OAuthServer} oauthServer
   * @param {{scope?: string, enableFileLinks?: boolean}} options
   *   scope           … OAuth のスコープ名。正本は OAuthServer.scope で、通常ここでは渡さない。
   *                     oauthServer が scope を持たないフェイク (テスト) のときだけ使われる
   *   enableFileLinks … /files/{token} の配信ルートを載せるか。
   *                     ファイル系ツールを持たない書き込み専用サーバでは false
   */
  constructor(server, port, oauthServer, options) {
    this.server = server;
    this.port = port;
    this.oauthServer = oauthServer;
    // scope は OAuthServer と共有の1値。以前は transport と OAuthServer が別々の値を
    // 持てたため、protected-resource と authorization-server の広告が矛盾していた
    // (2026-08-30 レビュー P0)。二重指定の不一致は静かに広告し続けず起動時に落とす。
    if (options.scope !== undefined && oauthServer.scope !== undefined && options.scope !== oauthServer.scope) {
      throw new Error(`OAuth scope mismatch: transport=${options.scope} oauthServer=${oauthServer.scope}`);
    }
    this.scope = oauthServer.scope ?? options.scope;
    if (!this.scope) throw new Error("HttpTransport requires a scope (via OAuthServer or options)");
    this.enableFileLinks = Boolean(options.enableFileLinks);
    // HTTP越しのクライアントは別マシン (例: クラウド上のAI) でありうる。
    // このMCPサーバ自身がgkillと同居していても、絶対パスを渡してよい相手ではない。
    this.server.isLocalTransport = false;
    if (this.enableFileLinks) {
      // リモートクライアントには実パスの代わりに期限付きの公開ファイルURLを渡す。
      // issuer は MCP_OAUTH_ISSUER (公開URL) で、この配信ルート自身の基点になる。
      this.fileLinkStore = new FileLinkStore();
      this.fileLinkStore.startCleanup();
      this.server.fileLinkContext = {
        publicBaseUrl: this.oauthServer.issuer,
        store: this.fileLinkStore,
      };
    }
  }

  start() {
    // httpServer はテストから stop() で確実に閉じられるようフィールドに保持する。
    // listen のポートに 0 を渡すと OS が空きポートを採番する (テスト用。本番は this.port)。
    this.httpServer = http.createServer((req, res) => this.handleRequest(req, res));
    // ヘッダとリクエスト全体の期限を明示する (Node 既定は 60s / 300s だが、
    // 既定に暗黙依存すると Node の版で挙動が変わる)。スローなヘッダ送信と
    // 終わらないリクエストをここで打ち切る (2026-08-30 監査 F-004)。
    this.httpServer.headersTimeout = 20 * 1000;
    this.httpServer.requestTimeout = 5 * 60 * 1000;
    // bind 先は MCP_BIND_ADDR で絞れる (例: リバースプロキシ/トンネル併用時は 127.0.0.1)。
    // 既定は互換のため全インターフェース待受のまま。
    const bindAddr = process.env.MCP_BIND_ADDR || "0.0.0.0";
    this.httpServer.listen(this.port, bindAddr, () => {
      const boundPort = this.httpServer.address()?.port ?? this.port;
      // 手で起動したときに見えるよう stderr へも出すが、ログにも1行残す。
      this.server.accessLog?.info?.("http_listening", { port: boundPort, issuer: this.oauthServer.issuer });
      process.stderr.write(`MCP HTTP server listening on http://${bindAddr}:${boundPort}/mcp [OAuth issuer: ${this.oauthServer.issuer}]\n`);
    });
    return this.httpServer;
  }

  // ボディを上限付きで収集する。超過したら 413 を返してソケットを破棄し、null で解決する。
  // ボディを読む4経路 (MCP POST / OAuth authorize POST / token / register) は必ずここを通す。
  collectBody(req, res, limitBytes, logReason) {
    return new Promise((resolve) => {
      const chunks = [];
      let received = 0;
      let done = false;
      req.on("data", (chunk) => {
        if (done) return;
        received += chunk.length;
        if (received > limitBytes) {
          done = true;
          this.logRequest(req, { statusCode: 413, reason: logReason });
          res.writeHead(413, { "Content-Type": "application/json" });
          // 413 を書き切ってから接続を破棄する (クライアントは残りのボディを送り続けるため)。
          res.end(JSON.stringify({ error: "Payload Too Large" }), () => {
            req.destroy();
          });
          resolve(null);
          return;
        }
        chunks.push(chunk);
      });
      req.on("end", () => {
        if (done) return;
        done = true;
        resolve(Buffer.concat(chunks));
      });
      req.on("error", () => {
        if (done) return;
        done = true;
        resolve(null);
      });
    });
  }

  // テスト用。listen 中の httpServer を閉じ、file-link の掃除タイマーも止める。
  stop() {
    return new Promise((resolve) => {
      this.fileLinkStore?.stopCleanup();
      if (this.httpServer) {
        this.httpServer.close(() => resolve());
      } else {
        resolve();
      }
    });
  }

  parseRoute(req) {
    const url = new URL(req.url, "http://localhost");
    const pathname = url.pathname;
    const query = Object.fromEntries(url.searchParams);

    // Protected Resource Metadata (RFC 9728)
    if (pathname === "/.well-known/oauth-protected-resource" ||
        pathname === "/.well-known/oauth-protected-resource/mcp") {
      return { type: "oauth-protected-resource", pathname };
    }

    // OAuth Authorization Server Metadata (RFC 8414)
    if (pathname === "/.well-known/oauth-authorization-server") {
      return { type: "oauth-metadata", pathname, query };
    }

    // OAuth endpoints — /oauth/* canonical, /* fallback for Claude.ai (known bug: ignores metadata endpoints)
    if (pathname === "/oauth/authorize" || pathname === "/authorize") {
      return { type: "oauth-authorize", pathname, query };
    }
    if (pathname === "/oauth/token" || pathname === "/token") {
      return { type: "oauth-token", pathname };
    }
    if (pathname === "/oauth/register" || pathname === "/register") {
      return { type: "oauth-register", pathname };
    }

    // MCP endpoint
    if (pathname === "/mcp") {
      return { type: "mcp", pathname };
    }

    // Public file delivery — token in the path, no Bearer needed (image fetchers
    // cannot send auth headers). The token itself is the security boundary.
    if (this.enableFileLinks && pathname.startsWith("/files/")) {
      return { type: "file", pathname, token: decodeURIComponent(pathname.slice("/files/".length)), query };
    }

    return null;
  }

  // アクセスログへ1行書く。生の stderr へは書かない (MCP_LOG を素通りするので
  // `MCP_LOG=none` でも止まらず、「MCP_LOG で制御できる」が嘘になる)。
  // path はクエリを落とした形だけを記録する (pathWithoutQuery のコメント参照)。
  // extra.redirect 等、ここに列挙されていないフィールドはログへ載らない。
  logRequest(req, extra = {}) {
    const statusCode = extra.statusCode || 0;
    const level = statusCode >= 400 ? "warn" : "info";
    this.server.accessLog[level]("http_request", {
      remote_addr: req.socket?.remoteAddress || null,
      method: req.method,
      path: pathWithoutQuery(req.url),
      status: statusCode,
      ...(extra.methods ? { methods: extra.methods } : {}),
      ...(extra.reason ? { reason: extra.reason } : {}),
      ...(extra.responseBytes !== undefined ? { response_bytes: extra.responseBytes } : {}),
    });
  }

  sendJson(res, statusCode, payload, headers = {}) {
    const body = payload === undefined ? "" : JSON.stringify(payload);
    const baseHeaders = body
      ? { "Content-Type": "application/json" }
      : {};
    res.writeHead(statusCode, { ...baseHeaders, ...headers });
    res.end(body);
    return Buffer.byteLength(body, "utf8");
  }

  summarizeJsonRpcMethods(payload) {
    if (Array.isArray(payload)) {
      return payload
        .map((item) => (item && typeof item === "object" && "method" in item ? item.method : "invalid"))
        .join(",");
    }
    if (payload && typeof payload === "object" && "method" in payload) {
      return payload.method;
    }
    return "invalid";
  }

  // 公開ファイル配信。トークンからファイルを解決し、gkillからバイトを取って返す。
  // Bearer不要 (画像取得は認証ヘッダを付けられない) なので、トークンが唯一の防御線。
  async handleFileServe(req, res, token, query) {
    const store = this.server.fileLinkContext?.store;
    const link = store ? store.resolve(token) : null;
    if (!link) {
      this.logRequest(req, { statusCode: 404, reason: "file_token_invalid" });
      this.sendJson(res, 404, { error: "Not Found" });
      return;
    }

    let gkillPath =
      "/files/" +
      encodeURIComponent(link.repName) +
      "/" +
      link.fileName
        .split("/")
        .map((s) => encodeURIComponent(s))
        .join("/");
    // サムネ指定は画像のときだけ、かつ WxH 形式に限って gkill に転送する。
    if (link.isImage && typeof query?.thumb === "string" && THUMB_QUERY_REGEX.test(query.thumb)) {
      gkillPath += `?thumb=${query.thumb}`;
    }

    try {
      const { buffer, contentType } = await this.server.client.fetchFile(gkillPath, link.gkillSessionId);
      this.logRequest(req, { statusCode: 200, responseBytes: buffer.length });
      // Mirrors: gkill_server_api/utils.go withUserContentSecurityHeaders。
      // 拡張子許可リストの無い利用者ファイルを、OAuth ログインフォームと同一オリジンで
      // 無認証配信するので、.html/.svg がスクリプト実行経路になりうる。nosniff は常時、
      // CSP sandbox は .pdf 以外に付ける (.pdf は sandbox の opaque origin で
      // Chrome 内蔵PDFビューワが動かなくなるため除外)。sandbox は文書読み込み時のみ効き
      // <img>/<video> のサブリソース表示には影響しないので、画像取得の本来用途は壊れない。
      const isPdf = link.fileName.toLowerCase().endsWith(".pdf");
      const responseHeaders = {
        "Content-Type": normalizeMimeType(contentType) || "application/octet-stream",
        "Content-Length": buffer.length,
        "Cache-Control": "private, max-age=300",
        "Access-Control-Allow-Origin": "*",
        "X-Content-Type-Options": "nosniff",
      };
      if (!isPdf) {
        responseHeaders["Content-Security-Policy"] = "sandbox";
      }
      res.writeHead(200, responseHeaders);
      res.end(buffer);
    } catch (error) {
      this.logRequest(req, { statusCode: 502, reason: "file_fetch_failed" });
      this.server.accessLog.error("file_fetch_error", {
        remote_addr: req.socket?.remoteAddress || null,
        error: error instanceof Error ? error.message : String(error),
      });
      this.sendJson(res, 502, { error: "Bad Gateway" });
    }
  }

  handleRequest(req, res) {
    const route = this.parseRoute(req);
    if (!route) {
      this.logRequest(req, { statusCode: 404, reason: "route_not_found" });
      this.sendJson(res, 404, { error: "Not Found. Use POST /mcp" });
      return;
    }

    // OAuth discovery/auth endpoints — no Bearer auth required
    if (route.type === "oauth-protected-resource") {
      return this.handleProtectedResourceMetadata(req, res);
    }
    if (route.type === "oauth-metadata") {
      return this.handleOAuthMetadata(req, res);
    }
    if (route.type === "oauth-authorize") {
      return this.handleOAuthAuthorize(req, res, route.query);
    }
    if (route.type === "oauth-token") {
      return this.handleOAuthToken(req, res);
    }
    if (route.type === "oauth-register") {
      return this.handleOAuthRegister(req, res);
    }

    // Public file delivery — Bearer 不要。トークンが防御線。
    if (route.type === "file") {
      if (req.method === "OPTIONS") {
        this.logRequest(req, { statusCode: 204 });
        res.writeHead(204, {
          "Access-Control-Allow-Origin": "*",
          "Access-Control-Allow-Methods": "GET, OPTIONS",
        });
        res.end();
        return;
      }
      if (req.method !== "GET") {
        this.logRequest(req, { statusCode: 405, reason: "method_not_allowed" });
        this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "GET, OPTIONS" });
        return;
      }
      return this.handleFileServe(req, res, route.token, route.query);
    }

    // MCP endpoint — require OAuth Bearer token
    const bearerToken = OAuthServer.extractBearerToken(req.headers["authorization"] || "");
    const tokenData = bearerToken ? this.oauthServer.validateAccessToken(bearerToken) : null;
    const resourceMetadataUrl = `${this.oauthServer.issuer}/.well-known/oauth-protected-resource`;

    if (!tokenData) {
      this.logRequest(req, { statusCode: 401, reason: "unauthorized" });
      this.server.accessLog.warn("token_rejected", {
        remote_addr: req.socket?.remoteAddress || null,
        method: req.method, path: pathWithoutQuery(req.url),
      });
      this.sendJson(res, 401, {
        error: "Unauthorized",
        error_description: "Bearer token required",
      }, {
        "WWW-Authenticate": `Bearer resource_metadata="${resourceMetadataUrl}"`,
      });
      return;
    }

    // scope はこのサーバの唯一の許可値と厳密一致のみ受理する (RFC 6750 insufficient_scope)。
    // 以前はトークンの存在だけを見ていたため、scope 修正前の ReadWrite サーバが発行した
    // "gkill:read" トークンでも書き込みツールを呼べた (2026-08-30 レビュー P0)。
    // 401 ではなく 403 を返すのは「トークンは本物だが権限が足りない = 再認可が要る」を
    // クライアントに伝えるため。
    if (tokenData.scope !== this.scope) {
      this.logRequest(req, { statusCode: 403, reason: "insufficient_scope" });
      this.server.accessLog.warn("token_scope_rejected", {
        remote_addr: req.socket?.remoteAddress || null,
        method: req.method, path: pathWithoutQuery(req.url),
        token_scope: tokenData.scope ?? null, required_scope: this.scope,
      });
      this.sendJson(res, 403, {
        error: "insufficient_scope",
        error_description: `This server requires scope "${this.scope}". Re-authorize to obtain it.`,
      }, {
        "WWW-Authenticate": `Bearer error="insufficient_scope", scope="${this.scope}", resource_metadata="${resourceMetadataUrl}"`,
      });
      return;
    }

    // 1リクエスト分の認証文脈は不変オブジェクトに固めて handlePost へ渡す。
    // 以前は server.currentUserId 等の共有フィールドに書いて await をまたいで
    // 読んでいたため、並行リクエストで別要求の user/session が混線していた。
    const requestContext = Object.freeze({
      sessionId: tokenData.gkillSessionId || null,
      userId: tokenData.userId || null,
      remoteAddr: req.socket?.remoteAddress || null,
    });
    switch (req.method) {
      case "POST":
        return this.handlePost(req, res, requestContext);
      case "GET":
        return this.handleGet(req, res);
      case "DELETE":
        return this.handleDelete(req, res);
      default:
        this.logRequest(req, { statusCode: 405, reason: "method_not_allowed" });
        this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "GET, POST, DELETE" });
    }
  }

  async handlePost(req, res, requestContext = null) {
    const rawBody = await this.collectBody(req, res, MAX_MCP_BODY_BYTES, "mcp_body_too_large");
    if (rawBody === null) {
      return;
    }
    {
      let payload;
      try {
        payload = JSON.parse(rawBody.toString("utf8"));
      } catch {
        this.logRequest(req, { statusCode: 400, reason: "parse_error" });
        this.sendJson(res, 400, { jsonrpc: "2.0", id: null, error: { code: -32700, message: "Parse error" } });
        return;
      }

      try {
        // 認証文脈は不変引数で末端まで流す (共有フィールドに書かないので並行しても混線しない)。
        const response = await this.server.handlePayload(payload, requestContext);
        const methods = this.summarizeJsonRpcMethods(payload);

        if (response === null) {
          this.logRequest(req, { methods, statusCode: 202, responseBytes: 0 });
          res.writeHead(202);
          res.end();
          return;
        }
        const responseBytes = this.sendJson(res, 200, response);
        this.logRequest(req, { methods, statusCode: 200, responseBytes });
      } catch (error) {
        // ここは真の内部例外。accessLog を通さないと gkill_mcp_error.log に残らない。
        this.server.accessLog.error("http_handler_error", {
          method: req.method,
          path: pathWithoutQuery(req.url),
          error: String(error),
        });
        const id =
          payload && !Array.isArray(payload) && Object.prototype.hasOwnProperty.call(payload, "id") ? payload.id : null;
        const responseBytes = this.sendJson(res, 200, {
          jsonrpc: "2.0",
          id,
          error: { code: -32603, message: "Internal error" },
        });
        this.logRequest(req, {
          methods: this.summarizeJsonRpcMethods(payload),
          statusCode: 200,
          responseBytes,
          reason: "internal_error",
        });
      }
    }
  }

  handleGet(req, res) {
    // SSE endpoint for server-initiated notifications.
    // Currently gkill has no server-push notifications, so just hold the connection open.
    const accept = req.headers["accept"] || "";
    if (!accept.includes("text/event-stream")) {
      this.logRequest(req, { statusCode: 406, reason: "missing_sse_accept_header" });
      this.sendJson(res, 406, { error: "Not Acceptable. Use Accept: text/event-stream" });
      return;
    }

    res.writeHead(200, {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache",
      Connection: "keep-alive",
    });
    // Keep connection alive with periodic comments
    const keepAlive = setInterval(() => {
      res.write(": keepalive\n\n");
    }, 30000);
    this.logRequest(req, { statusCode: 200, reason: "sse_open" });
    req.on("close", () => {
      clearInterval(keepAlive);
      this.logRequest(req, { statusCode: 200, reason: "sse_closed" });
    });
  }

  handleDelete(req, res) {
    // Stateless mode: DELETE is accepted as a no-op for clients that still send session cleanup.
    const responseBytes = this.sendJson(res, 200, { ok: true });
    this.logRequest(req, { statusCode: 200, responseBytes, reason: "stateless_delete_noop" });
  }

  // --- OAuth endpoint handlers ---

  handleProtectedResourceMetadata(req, res) {
    if (req.method !== "GET") {
      this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "GET" });
      return;
    }
    const issuer = this.oauthServer.issuer;
    const body = {
      resource: `${issuer}/mcp`,
      authorization_servers: [issuer],
      scopes_supported: [this.scope],
      bearer_methods_supported: ["header"],
    };
    this.sendJson(res, 200, body);
    this.logRequest(req, { statusCode: 200, reason: "oauth_protected_resource" });
  }

  handleOAuthMetadata(req, res) {
    if (req.method !== "GET") {
      this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "GET" });
      return;
    }
    const meta = this.oauthServer.getMetadata();
    this.sendJson(res, 200, meta);
    this.logRequest(req, { statusCode: 200, reason: "oauth_metadata" });
  }

  handleOAuthAuthorize(req, res, query) {
    if (req.method === "GET") {
      const result = this.oauthServer.handleAuthorizeGet(query);
      this._sendOAuthResult(req, res, result, "oauth_authorize_get");
      return;
    }
    if (req.method === "POST") {
      this.collectBody(req, res, MAX_OAUTH_BODY_BYTES, "oauth_authorize_body_too_large").then(async (rawBody) => {
        if (rawBody === null) {
          return;
        }
        try {
          const formData = Object.fromEntries(new URLSearchParams(rawBody.toString("utf8")));
          const result = await this.oauthServer.handleAuthorizePost(formData);
          this._sendOAuthResult(req, res, result, "oauth_authorize_post");
        } catch (error) {
          this.server.accessLog.error("oauth_authorize_error", { error: String(error) });
          this.sendJson(res, 500, { error: "Internal Server Error" });
        }
      });
      return;
    }
    this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "GET, POST" });
  }

  handleOAuthToken(req, res) {
    if (req.method !== "POST") {
      this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "POST" });
      return;
    }
    this.collectBody(req, res, MAX_OAUTH_BODY_BYTES, "oauth_token_body_too_large").then((rawBody) => {
      if (rawBody === null) {
        return;
      }
      try {
        const bodyStr = rawBody.toString("utf8");
        // Token endpoint accepts both application/x-www-form-urlencoded and application/json
        let body;
        const contentType = req.headers["content-type"] || "";
        if (contentType.includes("application/json")) {
          body = JSON.parse(bodyStr);
        } else {
          body = Object.fromEntries(new URLSearchParams(bodyStr));
        }
        const result = this.oauthServer.handleTokenRequest(body);
        this.sendJson(res, result.status, result.body);
        this.logRequest(req, { statusCode: result.status, reason: "oauth_token" });
      } catch (error) {
        this.server.accessLog.error("oauth_token_error", { error: String(error) });
        this.sendJson(res, 500, { error: "server_error", error_description: "Internal Server Error" });
      }
    });
  }

  handleOAuthRegister(req, res) {
    if (req.method !== "POST") {
      this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "POST" });
      return;
    }
    this.collectBody(req, res, MAX_OAUTH_BODY_BYTES, "oauth_register_body_too_large").then((rawBody) => {
      if (rawBody === null) {
        return;
      }
      try {
        const body = JSON.parse(rawBody.toString("utf8"));
        const result = this.oauthServer.handleRegister(body);
        this.sendJson(res, result.status, result.body);
        this.logRequest(req, { statusCode: result.status, reason: "oauth_register" });
      } catch (error) {
        this.server.accessLog.warn("oauth_register_invalid_request", { error: String(error) });
        this.sendJson(res, 400, { error: "invalid_client_metadata", error_description: "Invalid JSON" });
      }
    });
  }

  /** Send an OAuth result (HTML, redirect, or JSON). */
  _sendOAuthResult(req, res, result, reason) {
    if (result.redirect) {
      res.writeHead(result.status, { Location: result.redirect });
      res.end();
      // redirect URL は code / state を含むためログへ渡さない (2026-08-30 監査 F-003)。
      this.logRequest(req, { statusCode: result.status, reason });
      return;
    }
    if (result.contentType === "text/html") {
      const body = result.body;
      res.writeHead(result.status, { "Content-Type": "text/html; charset=utf-8" });
      res.end(body);
      this.logRequest(req, { statusCode: result.status, reason });
      return;
    }
    this.sendJson(res, result.status, result.body);
    this.logRequest(req, { statusCode: result.status, reason });
  }
}
