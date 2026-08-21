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
function makeOAuth(issuer = "http://127.0.0.1:0") {
  return new OAuthServer({
    issuer,
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
      const oauth = makeOAuth();
      const server = variant.make(createMockClient(), null);
      const transport = new HttpTransport(server, 0, oauth, { scope: variant.scope });
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
    oauth = makeOAuth();
    const server = SERVER_VARIANTS.find((v) => v.scope === scope).make(client, null);
    transport = new HttpTransport(server, 0, oauth, { scope });
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
    transport = new HttpTransport(new ReadServer(client, null), 0, oauth, { scope: "gkill:read" });
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
