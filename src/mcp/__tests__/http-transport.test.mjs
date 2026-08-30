/**
 * HttpTransport の /mcp 経路の統合・回帰テスト。
 *
 * 既存テスト (file-link.test.mjs) は handleFileServe を直接呼ぶだけで、
 * handleRequest の Bearer 認証・OAuth 連携・並行性は一度もテスト経路に乗っていなかった。
 * そのため以下の3件が緑のCIをすり抜けていた:
 *   - C-01: http-transport.mjs が OAuthServer を import せず /mcp で ReferenceError
 *   - C-02: server.current* 共有フィールドで並行リクエストの user/session が混線
 *   - M-06: 公開ファイル配信に nosniff / CSP sandbox が無い
 *
 * このファイルは実ポート (listen 0 = OS採番) で HttpTransport を起動し、
 * global fetch で OAuth→Bearer→tools まで通す統合と、並行分離の回帰を守る。
 */

import crypto from "node:crypto";
import { EventEmitter } from "node:events";
import { describe, test, expect, vi, afterEach } from "vitest";
import { OAuthServer } from "../lib/oauth-server.mjs";
import { HttpTransport } from "../lib/http-transport.mjs";
import { McpServer as ReadServer } from "../gkill-read-server.mjs";
import { McpWriteServer } from "../gkill-write-server.mjs";
import { McpServer as ReadWriteServer } from "../gkill-readwrite-server.mjs";

// 3サーバの違いは scope とファイル配信ルートの有無だけ。共有 transport を各scopeで検査する。
const SERVER_VARIANTS = [
  { scope: "gkill:read", make: (client, log) => new ReadServer(client, log) },
  { scope: "gkill:write", make: (client, log) => new McpWriteServer(client, log) },
  { scope: "gkill:readwrite", make: (client, log) => new ReadWriteServer(client, log) },
];

function createMockClient(overrides = {}) {
  return {
    callApi: vi.fn().mockResolvedValue({ errors: [], messages: [] }),
    fetchFile: vi.fn().mockResolvedValue({ buffer: Buffer.from("x"), contentType: "application/octet-stream" }),
    login: vi.fn().mockResolvedValue("login-session"),
    userId: "client-user",
    defaultLocale: "ja",
    ...overrides,
  };
}

// authenticateUser は userId ごとに別セッションを返す (並行分離テストで token を見分けるため)。
// scope の正本は OAuthServer のこの1値。HttpTransport は oauthServer.scope を読むので、
// transport の options へ scope を渡さない (二重指定は「transport にも渡すのが普通」という
// 誤った作法を読み手へ伝えるだけ。不一致 throw の検査は専用テストが1本だけ持つ)。
function makeOAuth(scope = "gkill:read", issuer = "http://127.0.0.1:0") {
  return new OAuthServer({
    issuer,
    scope,
    authenticateUser: async (userId) => ({ sessionId: `sess-${userId}` }),
    persistPath: null,
  });
}

function makeS256Pair() {
  const verifier = crypto.randomBytes(32).toString("base64url");
  const challenge = crypto.createHash("sha256").update(verifier, "ascii").digest("base64url");
  return { verifier, challenge };
}

function extractRedirectUrl(html) {
  const m = html.match(/window\.location\.href\s*=\s*"([^"]+)"/);
  return m ? m[1] : null;
}

// DCR でクライアントを登録して client_id を得る。
// (S3-oauth で未登録 client_id が拒否されても壊れないよう、常に登録経由にする)
function registerClient(oauth, redirectUri = "http://localhost/callback") {
  const res = oauth.handleRegister({ redirect_uris: [redirectUri], client_name: "test" });
  expect(res.status).toBe(201);
  return res.body.client_id;
}

// 認可コードフローを OAuthServer 上で直接回して access token を1つ得る。
async function mintAccessToken(oauth, userId, scope, clientId, redirectUri = "http://localhost/callback") {
  const { verifier, challenge } = makeS256Pair();
  const params = {
    response_type: "code",
    client_id: clientId,
    redirect_uri: redirectUri,
    code_challenge: challenge,
    code_challenge_method: "S256",
    scope,
    state: "s",
  };
  const post = await oauth.handleAuthorizePost({ ...params, user_id: userId, password_sha256: "pw" });
  const code = new URL(extractRedirectUrl(post.body)).searchParams.get("code");
  const tok = oauth.handleTokenRequest({
    grant_type: "authorization_code",
    code,
    code_verifier: verifier,
    client_id: clientId,
    redirect_uri: redirectUri,
  });
  expect(tok.status).toBe(200);
  return tok.body.access_token;
}

function mockRes() {
  return {
    statusCode: null,
    headers: null,
    body: null,
    writeHead(code, headers) {
      this.statusCode = code;
      this.headers = headers || {};
    },
    end(body) {
      this.body = body;
    },
  };
}

// ---------------------------------------------------------------------------
// 層1: Bearer 無し POST /mcp が 401 (C-01 を単体で殺す — import 脱落なら ReferenceError で赤)
// ---------------------------------------------------------------------------
describe("handleRequest — Bearer auth (C-01 regression)", () => {
  for (const variant of SERVER_VARIANTS) {
    test(`POST /mcp without Bearer returns 401 for ${variant.scope}`, () => {
      const oauth = makeOAuth(variant.scope);
      const server = variant.make(createMockClient(), null);
      const transport = new HttpTransport(server, 0, oauth, {});
      const req = { method: "POST", url: "/mcp", headers: {}, socket: { remoteAddress: "127.0.0.1" } };
      const res = mockRes();

      // import 脱落があるとこの行 (OAuthServer.extractBearerToken) が ReferenceError を投げる。
      transport.handleRequest(req, res);

      expect(res.statusCode).toBe(401);
      expect(res.headers["WWW-Authenticate"]).toContain("resource_metadata=");
      oauth.close();
    });
  }
});

// ---------------------------------------------------------------------------
// 層2: 実ポート統合 (metadata / 認可後 tools/list / initialize が 200)
// ---------------------------------------------------------------------------
describe("HttpTransport over real HTTP", () => {
  let oauth;
  let transport;
  let port;

  async function startTransport(scope, client) {
    oauth = makeOAuth(scope);
    const server = SERVER_VARIANTS.find((v) => v.scope === scope).make(client, null);
    transport = new HttpTransport(server, 0, oauth, {});
    const httpServer = transport.start();
    await new Promise((resolve) => httpServer.once("listening", resolve));
    port = httpServer.address().port;
  }

  afterEach(async () => {
    if (transport) await transport.stop();
    if (oauth) oauth.close();
    transport = null;
    oauth = null;
  });

  test("protected-resource metadata is public and advertises the scope", async () => {
    await startTransport("gkill:read", createMockClient());
    const res = await fetch(`http://127.0.0.1:${port}/.well-known/oauth-protected-resource`);
    expect(res.status).toBe(200);
    const body = await res.json();
    expect(body.scopes_supported).toEqual(["gkill:read"]);
  });

  test("POST /mcp without Bearer returns 401 over HTTP", async () => {
    await startTransport("gkill:read", createMockClient());
    const res = await fetch(`http://127.0.0.1:${port}/mcp`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} }),
    });
    expect(res.status).toBe(401);
    expect(res.headers.get("www-authenticate")).toContain("resource_metadata=");
  });

  test("authorized initialize returns serverInfo", async () => {
    await startTransport("gkill:read", createMockClient());
    const clientId = registerClient(oauth);
    const token = await mintAccessToken(oauth, "admin", "gkill:read", clientId);
    const res = await fetch(`http://127.0.0.1:${port}/mcp`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} }),
    });
    expect(res.status).toBe(200);
    const body = await res.json();
    expect(body.result.serverInfo.name).toBeTruthy();
    expect(body.result.protocolVersion).toBe("2024-11-05");
  });
});

// ---------------------------------------------------------------------------
// 層2c: scope 境界 (2026-08-30 レビュー P0)
// 以前は protected-resource が variant の scope・authorization-server が "gkill:read"
// 固定で矛盾広告し、Bearer 受理はトークンの存在だけを見て scope を照合していなかった。
// そのため ReadWrite サーバ上で "gkill:read" のトークンでも書き込みツールが呼べた。
// ---------------------------------------------------------------------------
describe("OAuth scope boundary (P0)", () => {
  let oauth;
  let transport;

  afterEach(async () => {
    if (transport) await transport.stop();
    if (oauth) oauth.close();
    transport = null;
    oauth = null;
  });

  for (const variant of SERVER_VARIANTS) {
    test(`both metadata endpoints advertise exactly [${variant.scope}]`, async () => {
      oauth = makeOAuth(variant.scope);
      const server = variant.make(createMockClient(), null);
      transport = new HttpTransport(server, 0, oauth, {});
      const httpServer = transport.start();
      await new Promise((resolve) => httpServer.once("listening", resolve));
      const port = httpServer.address().port;
      const prm = await (await fetch(`http://127.0.0.1:${port}/.well-known/oauth-protected-resource`)).json();
      const asm = await (await fetch(`http://127.0.0.1:${port}/.well-known/oauth-authorization-server`)).json();
      // 元指摘: ReadWrite で protected-resource=gkill:readwrite / authorization-server=gkill:read。
      expect(prm.scopes_supported).toEqual([variant.scope]);
      expect(asm.scopes_supported).toEqual([variant.scope]);
    });
  }

  test("a legacy token with a foreign scope gets 403 insufficient_scope", () => {
    oauth = makeOAuth("gkill:readwrite");
    const server = new ReadWriteServer(createMockClient(), null);
    // 403 の監査イベントは運用者が scope 拒否を知る唯一の窓なので、応答と一緒に固定する。
    const warns = [];
    server.accessLog = {
      info() {},
      warn(msg, fields) { warns.push({ msg, fields }); },
      error() {},
    };
    transport = new HttpTransport(server, 0, oauth, {});
    // scope 修正前の ReadWrite サーバが発行した "gkill:read" のアクセストークンを再現する。
    oauth.store.putAccessToken("legacy-access", {
      clientId: "c",
      scope: "gkill:read",
      gkillSessionId: "sess-x",
      userId: "u",
    });
    const req = {
      method: "POST",
      url: "/mcp",
      headers: { authorization: "Bearer legacy-access" },
      socket: { remoteAddress: "127.0.0.1" },
    };
    const res = mockRes();
    transport.handleRequest(req, res);
    expect(res.statusCode).toBe(403);
    expect(JSON.parse(res.body).error).toBe("insufficient_scope");
    // 401 (トークン無効) と区別し、クライアントへ再認可を促すヘッダを返す。
    expect(res.headers["WWW-Authenticate"]).toContain('error="insufficient_scope"');
    expect(res.headers["WWW-Authenticate"]).toContain('scope="gkill:readwrite"');
    // どの scope のトークンが何を要求されて拒否されたかまでログに残る
    // (logRequest の http_request 行も 403 なので warn に並ぶ。ここでは監査イベント側だけを見る)。
    const scopeWarns = warns.filter((w) => w.msg === "token_scope_rejected");
    expect(scopeWarns).toEqual([
      expect.objectContaining({
        fields: expect.objectContaining({ token_scope: "gkill:read", required_scope: "gkill:readwrite" }),
      }),
    ]);
  });

  test("a matching-scope token still reaches the MCP handler", () => {
    oauth = makeOAuth("gkill:readwrite");
    const server = new ReadWriteServer(createMockClient(), null);
    transport = new HttpTransport(server, 0, oauth, {});
    oauth.store.putAccessToken("good-access", {
      clientId: "c",
      scope: "gkill:readwrite",
      gkillSessionId: "sess-x",
      userId: "u",
    });
    const req = {
      method: "GET",
      url: "/mcp",
      headers: { authorization: "Bearer good-access", accept: "application/json" },
      socket: { remoteAddress: "127.0.0.1" },
    };
    const res = mockRes();
    transport.handleRequest(req, res);
    // 認証は通過し、SSE の Accept ヘッダ検査 (406) まで到達している = 401/403 で弾かれていない。
    expect(res.statusCode).toBe(406);
  });

  test("mismatched transport/oauth scopes fail fast at construction", () => {
    oauth = makeOAuth("gkill:read");
    expect(
      () => new HttpTransport(new ReadWriteServer(createMockClient(), null), 0, oauth, { scope: "gkill:readwrite" }),
    ).toThrow("scope mismatch");
  });

  test("a transport with no scope source at all fails fast at construction", () => {
    // oauthServer が scope を持たないフェイクで、options.scope も無い組み合わせ。
    // ここで通すと Bearer 受理の scope 照合 (tokenData.scope !== this.scope) が
    // undefined 比較になり、全トークンを 403 で弾く事故になる。
    expect(
      () => new HttpTransport(new ReadWriteServer(createMockClient(), null), 0, { issuer: "https://mcp.example.test" }, {}),
    ).toThrow("requires a scope");
  });
});

// ---------------------------------------------------------------------------
// 層2b: 並行分離 (C-02 regression) — 2トークン同時 tools/call で session が混ざらない
// ---------------------------------------------------------------------------
describe("concurrent requests keep session context separate (C-02 regression)", () => {
  let oauth;
  let transport;
  let port;

  afterEach(async () => {
    if (transport) await transport.stop();
    if (oauth) oauth.close();
    transport = null;
    oauth = null;
  });

  test("two in-flight tool calls each see their own token's session", async () => {
    // callApi は両リクエストが揃うまで待つバリアで、意図的に処理を重ねる。
    // 共有フィールド方式に退行すると、この重なりの窓で sid が上書きされる。
    let inFlight = 0;
    let releaseBarrier;
    const barrier = new Promise((resolve) => {
      releaseBarrier = resolve;
    });
    const seenSids = [];
    const client = createMockClient({
      callApi: vi.fn(async (pathname, body, useSession, sid) => {
        seenSids.push(sid);
        inFlight += 1;
        if (inFlight === 2) releaseBarrier();
        await barrier;
        // sid をレスポンスに反映させ、リクエストと応答の対応も検証できるようにする。
        return { errors: [], messages: [], tag_names: [String(sid)] };
      }),
    });

    oauth = makeOAuth();
    transport = new HttpTransport(new ReadServer(client, null), 0, oauth, {});
    const httpServer = transport.start();
    await new Promise((resolve) => httpServer.once("listening", resolve));
    port = httpServer.address().port;

    const clientId = registerClient(oauth);
    const tokenA = await mintAccessToken(oauth, "alice", "gkill:read", clientId);
    const tokenB = await mintAccessToken(oauth, "bob", "gkill:read", clientId);

    const call = (token) =>
      fetch(`http://127.0.0.1:${port}/mcp`, {
        method: "POST",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 1,
          method: "tools/call",
          params: { name: "gkill_get_all_tag_names", arguments: {} },
        }),
      }).then((r) => r.json());

    const [resA, resB] = await Promise.all([call(tokenA), call(tokenB)]);

    // 各リクエストは自分のトークンのセッションで gkill を叩いている。
    expect(resA.result.structuredContent.tag_names).toEqual(["sess-alice"]);
    expect(resB.result.structuredContent.tag_names).toEqual(["sess-bob"]);
    // 両セッションが1回ずつ現れ、取り違えが無い。
    expect(seenSids.slice().sort()).toEqual(["sess-alice", "sess-bob"]);
  });
});

// ---------------------------------------------------------------------------
// M-06: 公開ファイル配信に nosniff / CSP sandbox が付く
// ---------------------------------------------------------------------------
describe("handleFileServe security headers (M-06)", () => {
  let transport;

  afterEach(() => {
    transport?.fileLinkStore?.stopCleanup();
  });

  function buildTransport(fetchFile) {
    const fakeServer = {
      client: { fetchFile },
      accessLog: { info() {}, warn() {}, error() {} },
    };
    return new HttpTransport(fakeServer, 0, { issuer: "https://mcp.example.test" }, {
      scope: "gkill:read",
      enableFileLinks: true,
    });
  }

  const req = { socket: { remoteAddress: "127.0.0.1" }, url: "/files/x", method: "GET", headers: {} };

  test("image/HTML files get nosniff and CSP sandbox", async () => {
    transport = buildTransport(vi.fn().mockResolvedValue({ buffer: Buffer.from([1]), contentType: "text/html" }));
    const token = transport.fileLinkStore.mint({
      gkillSessionId: "s", repName: "Files", fileName: "note.html", isImage: false,
    });
    const res = mockRes();
    await transport.handleFileServe(req, res, token, {});
    expect(res.statusCode).toBe(200);
    expect(res.headers["X-Content-Type-Options"]).toBe("nosniff");
    expect(res.headers["Content-Security-Policy"]).toBe("sandbox");
  });

  test("PDF gets nosniff but no sandbox (built-in viewer needs a real origin)", async () => {
    transport = buildTransport(vi.fn().mockResolvedValue({ buffer: Buffer.from([1]), contentType: "application/pdf" }));
    const token = transport.fileLinkStore.mint({
      gkillSessionId: "s", repName: "Files", fileName: "doc.pdf", isImage: false,
    });
    const res = mockRes();
    await transport.handleFileServe(req, res, token, {});
    expect(res.statusCode).toBe(200);
    expect(res.headers["X-Content-Type-Options"]).toBe("nosniff");
    expect(res.headers["Content-Security-Policy"]).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// 2026-08-30 監査 F-003 / F-004: ボディ上限・明示タイムアウト・ログのクエリ除去
// ---------------------------------------------------------------------------
describe("body caps, explicit timeouts, and log query redaction (2026-08-30 audit)", () => {
  let transport;
  let oauth;

  afterEach(async () => {
    await transport?.stop();
    oauth?.close();
    delete process.env.MCP_BIND_ADDR;
  });

  function makeCapturingLog(logs) {
    const push = (level) => (msg, fields) => logs.push({ level, msg, fields });
    return { info: push("info"), warn: push("warn"), error: push("error") };
  }

  function makeFakeReq(url = "/oauth/register") {
    const req = new EventEmitter();
    req.method = "POST";
    req.url = url;
    req.headers = {};
    req.socket = { remoteAddress: "127.0.0.1" };
    req.destroy = vi.fn();
    return req;
  }

  test("collectBody cuts an over-limit body with 413 and resolves null", async () => {
    oauth = makeOAuth();
    const server = new ReadServer(createMockClient(), null);
    server.accessLog = makeCapturingLog([]);
    transport = new HttpTransport(server, 0, oauth, {});
    const req = makeFakeReq();
    const res = mockRes();

    const pending = transport.collectBody(req, res, 8, "test_body_too_large");
    req.emit("data", Buffer.from("123456789")); // 9 bytes > limit 8
    const rawBody = await pending;

    expect(rawBody).toBeNull();
    expect(res.statusCode).toBe(413);
  });

  test("collectBody passes an at-limit body through unchanged", async () => {
    oauth = makeOAuth();
    const server = new ReadServer(createMockClient(), null);
    server.accessLog = makeCapturingLog([]);
    transport = new HttpTransport(server, 0, oauth, {});
    const req = makeFakeReq();
    const res = mockRes();

    const pending = transport.collectBody(req, res, 8, "test_body_too_large");
    req.emit("data", Buffer.from("12345678")); // exactly the limit
    req.emit("end");
    const rawBody = await pending;

    expect(rawBody?.toString("utf8")).toBe("12345678");
    expect(res.statusCode).toBeNull();
  });

  test("POST /oauth/register over the cap is rejected before handleRegister (route wiring)", async () => {
    oauth = makeOAuth();
    const logs = [];
    const server = new ReadServer(createMockClient(), null);
    server.accessLog = makeCapturingLog(logs);
    transport = new HttpTransport(server, 0, oauth, {});
    const registerSpy = vi.spyOn(oauth, "handleRegister");
    const httpServer = transport.start();
    const port = await new Promise((resolve) =>
      httpServer.on("listening", () => resolve(httpServer.address().port)));

    const net = await import("node:net");
    const sock = net.connect(port, "127.0.0.1");
    const body = Buffer.alloc(64 * 1024 + 16, 0x61);
    sock.write(`POST /oauth/register HTTP/1.1\r\nHost: t\r\nContent-Type: application/json\r\nContent-Length: ${body.length}\r\n\r\n`);
    sock.write(body);

    await vi.waitFor(() => {
      expect(
        logs.some((l) => l.fields?.reason === "oauth_register_body_too_large" && l.fields?.status === 413),
      ).toBe(true);
    });
    expect(registerSpy).not.toHaveBeenCalled();
    sock.destroy();
  });

  test("authorize query values (client_id/state/code_challenge) never reach the access log", () => {
    oauth = makeOAuth();
    const logs = [];
    const server = new ReadServer(createMockClient(), null);
    server.accessLog = makeCapturingLog(logs);
    transport = new HttpTransport(server, 0, oauth, {});

    const req = {
      method: "GET",
      url: "/oauth/authorize?client_id=SECRETCID&redirect_uri=https%3A%2F%2Fx%2FSECRETURI&state=SECRETSTATE&code_challenge=SECRETCC",
      headers: {},
      socket: { remoteAddress: "127.0.0.1" },
    };
    transport.logRequest(req, { statusCode: 200, reason: "oauth_authorize_get", redirect: "https://x/cb?code=SECRETCODE&state=SECRETSTATE" });

    expect(logs.length).toBe(1);
    expect(logs[0].fields.path).toBe("/oauth/authorize");
    expect(JSON.stringify(logs)).not.toContain("SECRET");
  });

  test("explicit timeouts and MCP_BIND_ADDR are applied at start()", async () => {
    process.env.MCP_BIND_ADDR = "127.0.0.1";
    oauth = makeOAuth();
    const server = new ReadServer(createMockClient(), null);
    transport = new HttpTransport(server, 0, oauth, {});
    const httpServer = transport.start();
    await new Promise((resolve) => httpServer.on("listening", resolve));

    expect(httpServer.headersTimeout).toBe(20 * 1000);
    expect(httpServer.requestTimeout).toBe(5 * 60 * 1000);
    expect(httpServer.address().address).toBe("127.0.0.1");
  });
});
