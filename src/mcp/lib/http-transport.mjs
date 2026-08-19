// Streamable HTTP トランスポート (MCP spec 2024-11-05)。3つのMCPサーバで共有する。
//
// POST /mcp (リクエスト) / GET /mcp (SSE) / DELETE /mcp (セッション終了) に加えて、
// OAuth 2.1 のエンドポイントと、リモートクライアント向けのファイル配信 /files/{token} を持つ。
//
// 以前は3本に逐語コピーされていた。サーバごとに違うのは
// OAuth のスコープ名と、ファイル配信ルートを載せるかどうかの2点だけなので、
// その2つだけをコンストラクタの options で受ける。

import http from "node:http";
import { FileLinkStore } from "./file-link-store.mjs";
import { THUMB_QUERY_REGEX, normalizeMimeType } from "./payload.mjs";

// HttpTransport: Streamable HTTP transport (MCP spec 2024-11-05).
// Supports POST /mcp (requests), GET /mcp (SSE stream), DELETE /mcp (session end).
// OAuth 2.1 endpoints for ChatGPT and Claude.ai MCP connectors.
export class HttpTransport {
  /**
   * @param {object} server MCPサーバ本体
   * @param {number} port
   * @param {OAuthServer} oauthServer
   * @param {{scope: string, enableFileLinks?: boolean}} options
   *   scope           … OAuth のスコープ名 (gkill:read / gkill:write / gkill:readwrite)
   *   enableFileLinks … /files/{token} の配信ルートを載せるか。
   *                     ファイル系ツールを持たない書き込み専用サーバでは false
   */
  constructor(server, port, oauthServer, options) {
    this.server = server;
    this.port = port;
    this.oauthServer = oauthServer;
    this.scope = options.scope;
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
    const httpServer = http.createServer((req, res) => this.handleRequest(req, res));
    httpServer.listen(this.port, "0.0.0.0", () => {
      process.stderr.write(`MCP HTTP server listening on http://0.0.0.0:${this.port}/mcp [OAuth issuer: ${this.oauthServer.issuer}]\n`);
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

  logRequest(req, extra = {}) {
    const payload = {
      method: req.method,
      path: req.url,
      sessionId: req.headers["mcp-session-id"] || null,
      ...extra,
    };
    process.stderr.write(`[${new Date().toISOString()}] MCP HTTP ${JSON.stringify(payload)}\n`);

    // Also write to access log file
    const statusCode = extra.statusCode || 0;
    const level = statusCode >= 400 ? "warn" : "info";
    this.server.accessLog[level]("http_request", {
      remote_addr: req.socket?.remoteAddress || null,
      method: req.method,
      path: req.url,
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
      res.writeHead(200, {
        "Content-Type": normalizeMimeType(contentType) || "application/octet-stream",
        "Content-Length": buffer.length,
        "Cache-Control": "private, max-age=300",
        "Access-Control-Allow-Origin": "*",
      });
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

    if (!tokenData) {
      this.logRequest(req, { statusCode: 401, reason: "unauthorized" });
      this.server.accessLog.warn("token_rejected", {
        remote_addr: req.socket?.remoteAddress || null,
        method: req.method, path: req.url,
      });
      const resourceMetadataUrl = `${this.oauthServer.issuer}/.well-known/oauth-protected-resource`;
      this.sendJson(res, 401, {
        error: "Unauthorized",
        error_description: "Bearer token required",
      }, {
        "WWW-Authenticate": `Bearer resource_metadata="${resourceMetadataUrl}"`,
      });
      return;
    }

    // NOTE: _lastTokenUserId は handlePost 内で server.currentUserId に転記される。
    // HTTP/1.1 直列処理を前提としており、HTTP/2 並行リクエスト時はリクエスト間でリークする可能性がある。
    this._lastTokenUserId = tokenData.userId || null;
    switch (req.method) {
      case "POST":
        return this.handlePost(req, res, tokenData.gkillSessionId);
      case "GET":
        return this.handleGet(req, res);
      case "DELETE":
        return this.handleDelete(req, res);
      default:
        this.logRequest(req, { statusCode: 405, reason: "method_not_allowed" });
        this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "GET, POST, DELETE" });
    }
  }

  handlePost(req, res, oauthSessionId = null) {
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", async () => {
      let payload;
      try {
        payload = JSON.parse(Buffer.concat(chunks).toString("utf8"));
      } catch {
        this.logRequest(req, { statusCode: 400, reason: "parse_error" });
        this.sendJson(res, 400, { jsonrpc: "2.0", id: null, error: { code: -32700, message: "Parse error" } });
        return;
      }

      try {
        // Set session override for OAuth-authenticated requests
        this.server.currentSessionId = oauthSessionId;
        this.server.currentUserId = this._lastTokenUserId || null;
        this.server.currentRemoteAddr = req.socket?.remoteAddress || null;
        const response = await this.server.handlePayload(payload);
        this.server.currentSessionId = null;
        this.server.currentUserId = null;
        this.server.currentRemoteAddr = null;
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
        this.server.currentSessionId = null;
        this.server.currentUserId = null;
        this.server.currentRemoteAddr = null;
        process.stderr.write(`HTTP handler error: ${String(error)}\n`);
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
    });
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
      const chunks = [];
      req.on("data", (chunk) => chunks.push(chunk));
      req.on("end", async () => {
        try {
          const bodyStr = Buffer.concat(chunks).toString("utf8");
          const formData = Object.fromEntries(new URLSearchParams(bodyStr));
          const result = await this.oauthServer.handleAuthorizePost(formData);
          this._sendOAuthResult(req, res, result, "oauth_authorize_post");
        } catch (error) {
          process.stderr.write(`OAuth authorize error: ${String(error)}\n`);
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
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", () => {
      try {
        const bodyStr = Buffer.concat(chunks).toString("utf8");
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
        process.stderr.write(`OAuth token error: ${String(error)}\n`);
        this.sendJson(res, 500, { error: "server_error", error_description: "Internal Server Error" });
      }
    });
  }

  handleOAuthRegister(req, res) {
    if (req.method !== "POST") {
      this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "POST" });
      return;
    }
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", () => {
      try {
        const body = JSON.parse(Buffer.concat(chunks).toString("utf8"));
        const result = this.oauthServer.handleRegister(body);
        this.sendJson(res, result.status, result.body);
        this.logRequest(req, { statusCode: result.status, reason: "oauth_register" });
      } catch (error) {
        process.stderr.write(`OAuth register error: ${String(error)}\n`);
        this.sendJson(res, 400, { error: "invalid_client_metadata", error_description: "Invalid JSON" });
      }
    });
  }

  /** Send an OAuth result (HTML, redirect, or JSON). */
  _sendOAuthResult(req, res, result, reason) {
    if (result.redirect) {
      res.writeHead(result.status, { Location: result.redirect });
      res.end();
      this.logRequest(req, { statusCode: result.status, reason, redirect: result.redirect });
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
