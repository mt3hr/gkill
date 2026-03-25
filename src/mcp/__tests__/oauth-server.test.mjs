import crypto from "node:crypto";
import { OAuthServer } from "../lib/oauth-server.mjs";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Create a standard OAuthServer with a mock authenticateUser. */
function createServer(overrides = {}) {
  const authenticateUser = overrides.authenticateUser || (async (userId, passwordSha256) => {
    if (userId === "admin" && passwordSha256 === "abc123") {
      return { sessionId: "gkill-session-001" };
    }
    return null;
  });
  const server = new OAuthServer({
    issuer: overrides.issuer || "http://localhost:8808",
    authenticateUser,
  });
  return server;
}

/** Generate a PKCE S256 pair. */
function makeS256Pair() {
  const verifier = crypto.randomBytes(32).toString("base64url");
  const challenge = crypto.createHash("sha256").update(verifier, "ascii").digest("base64url");
  return { verifier, challenge };
}

/** Extract redirect URL from success page HTML (window.location.href = "..."). */
function extractRedirectUrl(html) {
  const match = html.match(/window\.location\.href\s*=\s*"([^"]+)"/);
  return match ? match[1] : null;
}

/** Extract authorization code from a successful authorize POST result. */
function extractCodeFromResult(result) {
  expect(result.status).toBe(200);
  expect(result.contentType).toBe("text/html");
  const redirectUrl = extractRedirectUrl(result.body);
  expect(redirectUrl).toBeTruthy();
  return new URL(redirectUrl).searchParams.get("code");
}

/** Standard authorize params. */
function authorizeParams(extra = {}) {
  const { challenge } = makeS256Pair();
  return {
    response_type: "code",
    client_id: "test-client",
    redirect_uri: "http://localhost/callback",
    code_challenge: challenge,
    code_challenge_method: "S256",
    scope: "gkill:read",
    state: "xyz",
    ...extra,
  };
}

/** Run full authorization code flow and return { accessToken, refreshToken }. */
async function fullAuthCodeFlow(server, pkce) {
  const { verifier, challenge } = pkce || makeS256Pair();
  const params = authorizeParams({ code_challenge: challenge });
  const postResult = await server.handleAuthorizePost({
    ...params,
    user_id: "admin",
    password_sha256: "abc123",
  });
  const code = extractCodeFromResult(postResult);

  const tokenResult = server.handleTokenRequest({
    grant_type: "authorization_code",
    code,
    code_verifier: verifier,
    client_id: "test-client",
    redirect_uri: "http://localhost/callback",
  });
  expect(tokenResult.status).toBe(200);
  return tokenResult.body;
}

afterEach(() => {
  vi.restoreAllMocks();
});

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------
describe("OAuthServer — constructor", () => {
  test("creates with valid options", () => {
    const server = createServer();
    expect(server.issuer).toBe("http://localhost:8808");
    server.close();
  });

  test("strips trailing slash from issuer", () => {
    const server = createServer({ issuer: "http://localhost:8808/" });
    expect(server.issuer).toBe("http://localhost:8808");
    server.close();
  });

  test("throws without issuer", () => {
    expect(() => new OAuthServer({ issuer: "", authenticateUser: async () => null })).toThrow("issuer");
  });

  test("throws without authenticateUser", () => {
    expect(() => new OAuthServer({ issuer: "http://x", authenticateUser: null })).toThrow("authenticateUser");
  });
});

// ---------------------------------------------------------------------------
// Metadata
// ---------------------------------------------------------------------------
describe("OAuthServer — getMetadata", () => {
  test("returns correct metadata structure", () => {
    const server = createServer();
    const meta = server.getMetadata();
    expect(meta.issuer).toBe("http://localhost:8808");
    expect(meta.authorization_endpoint).toBe("http://localhost:8808/oauth/authorize");
    expect(meta.token_endpoint).toBe("http://localhost:8808/oauth/token");
    expect(meta.registration_endpoint).toBe("http://localhost:8808/oauth/register");
    expect(meta.response_types_supported).toEqual(["code"]);
    expect(meta.grant_types_supported).toContain("authorization_code");
    expect(meta.grant_types_supported).toContain("refresh_token");
    expect(meta.code_challenge_methods_supported).toContain("S256");
    server.close();
  });
});

// ---------------------------------------------------------------------------
// Authorize GET
// ---------------------------------------------------------------------------
describe("OAuthServer — handleAuthorizeGet", () => {
  let server;
  beforeEach(() => { server = createServer(); });
  afterEach(() => { server.close(); });

  test("returns 200 with login form HTML", () => {
    const result = server.handleAuthorizeGet(authorizeParams());
    expect(result.status).toBe(200);
    expect(result.contentType).toBe("text/html");
    expect(result.body).toContain("gkill");
    expect(result.body).toContain("loginForm");
  });

  test("returns 400 for missing response_type", () => {
    const result = server.handleAuthorizeGet(authorizeParams({ response_type: "" }));
    expect(result.status).toBe(400);
  });

  test("returns 400 for invalid response_type", () => {
    const result = server.handleAuthorizeGet(authorizeParams({ response_type: "token" }));
    expect(result.status).toBe(400);
  });

  test("returns 400 for missing client_id", () => {
    const result = server.handleAuthorizeGet(authorizeParams({ client_id: "" }));
    expect(result.status).toBe(400);
  });

  test("returns 400 for missing redirect_uri", () => {
    const result = server.handleAuthorizeGet(authorizeParams({ redirect_uri: "" }));
    expect(result.status).toBe(400);
  });

  test("returns 400 for invalid redirect_uri", () => {
    const result = server.handleAuthorizeGet(authorizeParams({ redirect_uri: "not-a-url" }));
    expect(result.status).toBe(400);
  });

  test("returns 400 for missing code_challenge", () => {
    const result = server.handleAuthorizeGet(authorizeParams({ code_challenge: "" }));
    expect(result.status).toBe(400);
  });

  test("returns 400 for unsupported code_challenge_method", () => {
    const result = server.handleAuthorizeGet(authorizeParams({ code_challenge_method: "S512" }));
    expect(result.status).toBe(400);
  });

  test("defaults code_challenge_method to S256", () => {
    const params = authorizeParams();
    delete params.code_challenge_method;
    const result = server.handleAuthorizeGet(params);
    expect(result.status).toBe(200);
  });
});

// ---------------------------------------------------------------------------
// Authorize POST — success
// ---------------------------------------------------------------------------
describe("OAuthServer — handleAuthorizePost (success)", () => {
  let server;
  beforeEach(() => { server = createServer(); });
  afterEach(() => { server.close(); });

  test("returns success page with redirect URL on successful login", async () => {
    const { challenge } = makeS256Pair();
    const result = await server.handleAuthorizePost({
      ...authorizeParams({ code_challenge: challenge }),
      user_id: "admin",
      password_sha256: "abc123",
    });
    expect(result.status).toBe(200);
    const redirectUrl = new URL(extractRedirectUrl(result.body));
    expect(redirectUrl.searchParams.get("code")).toBeTruthy();
    expect(redirectUrl.searchParams.get("state")).toBe("xyz");
    expect(redirectUrl.origin + redirectUrl.pathname).toBe("http://localhost/callback");
  });

  test("omits state from redirect when not provided", async () => {
    const { challenge } = makeS256Pair();
    const result = await server.handleAuthorizePost({
      ...authorizeParams({ code_challenge: challenge, state: "" }),
      user_id: "admin",
      password_sha256: "abc123",
    });
    expect(result.status).toBe(200);
    const redirectUrl = new URL(extractRedirectUrl(result.body));
    expect(redirectUrl.searchParams.has("state")).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// Authorize POST — failure
// ---------------------------------------------------------------------------
describe("OAuthServer — handleAuthorizePost (failure)", () => {
  let server;
  beforeEach(() => { server = createServer(); });
  afterEach(() => { server.close(); });

  test("re-renders login form with error on bad credentials", async () => {
    const result = await server.handleAuthorizePost({
      ...authorizeParams(),
      user_id: "admin",
      password_sha256: "wrong",
    });
    expect(result.status).toBe(200);
    expect(result.body).toContain("ログインに失敗しました");
  });

  test("re-renders login form when authenticateUser throws", async () => {
    const server2 = createServer({
      authenticateUser: async () => { throw new Error("network error"); },
    });
    const result = await server2.handleAuthorizePost({
      ...authorizeParams(),
      user_id: "admin",
      password_sha256: "abc123",
    });
    expect(result.status).toBe(200);
    expect(result.body).toContain("ログインに失敗しました");
    server2.close();
  });

  test("returns 400 for invalid params in POST", async () => {
    const result = await server.handleAuthorizePost({
      ...authorizeParams({ response_type: "token" }),
      user_id: "admin",
      password_sha256: "abc123",
    });
    expect(result.status).toBe(400);
  });
});

// ---------------------------------------------------------------------------
// Token endpoint — authorization_code grant
// ---------------------------------------------------------------------------
describe("OAuthServer — token (authorization_code)", () => {
  let server;
  beforeEach(() => { server = createServer(); });
  afterEach(() => { server.close(); });

  test("exchanges valid code + verifier for tokens", async () => {
    const pkce = makeS256Pair();
    const tokens = await fullAuthCodeFlow(server, pkce);
    expect(tokens.access_token).toBeTruthy();
    expect(tokens.refresh_token).toBeTruthy();
    expect(tokens.token_type).toBe("Bearer");
    expect(tokens.expires_in).toBe(3600);
    expect(tokens.scope).toBe("gkill:read");
  });

  test("rejects code replay (one-time use)", async () => {
    const pkce = makeS256Pair();
    const params = authorizeParams({ code_challenge: pkce.challenge });
    const postResult = await server.handleAuthorizePost({
      ...params,
      user_id: "admin",
      password_sha256: "abc123",
    });
    const code = extractCodeFromResult(postResult);

    // First exchange succeeds
    const result1 = server.handleTokenRequest({
      grant_type: "authorization_code",
      code,
      code_verifier: pkce.verifier,
    });
    expect(result1.status).toBe(200);

    // Second exchange fails (code already consumed)
    const result2 = server.handleTokenRequest({
      grant_type: "authorization_code",
      code,
      code_verifier: pkce.verifier,
    });
    expect(result2.status).toBe(400);
    expect(result2.body.error).toBe("invalid_grant");
  });

  test("rejects wrong code_verifier (PKCE failure)", async () => {
    const pkce = makeS256Pair();
    const params = authorizeParams({ code_challenge: pkce.challenge });
    const postResult = await server.handleAuthorizePost({
      ...params,
      user_id: "admin",
      password_sha256: "abc123",
    });
    const code = extractCodeFromResult(postResult);

    const result = server.handleTokenRequest({
      grant_type: "authorization_code",
      code,
      code_verifier: "wrong-verifier-that-is-definitely-not-correct-at-all",
    });
    expect(result.status).toBe(400);
    expect(result.body.error).toBe("invalid_grant");
    expect(result.body.error_description).toContain("PKCE");
  });

  test("rejects expired authorization code", async () => {
    vi.useFakeTimers();
    try {
      const pkce = makeS256Pair();
      const params = authorizeParams({ code_challenge: pkce.challenge });
      const postResult = await server.handleAuthorizePost({
        ...params,
        user_id: "admin",
        password_sha256: "abc123",
      });
      const code = extractCodeFromResult(postResult);

      // Advance time past code TTL (5 minutes)
      vi.advanceTimersByTime(5 * 60 * 1000 + 1);

      const result = server.handleTokenRequest({
        grant_type: "authorization_code",
        code,
        code_verifier: pkce.verifier,
      });
      expect(result.status).toBe(400);
      expect(result.body.error).toBe("invalid_grant");
    } finally {
      vi.useRealTimers();
    }
  });

  test("rejects mismatched client_id", async () => {
    const pkce = makeS256Pair();
    const params = authorizeParams({ code_challenge: pkce.challenge });
    const postResult = await server.handleAuthorizePost({
      ...params,
      user_id: "admin",
      password_sha256: "abc123",
    });
    const code = extractCodeFromResult(postResult);

    const result = server.handleTokenRequest({
      grant_type: "authorization_code",
      code,
      code_verifier: pkce.verifier,
      client_id: "different-client",
    });
    expect(result.status).toBe(400);
    expect(result.body.error).toBe("invalid_grant");
  });

  test("rejects mismatched redirect_uri", async () => {
    const pkce = makeS256Pair();
    const params = authorizeParams({ code_challenge: pkce.challenge });
    const postResult = await server.handleAuthorizePost({
      ...params,
      user_id: "admin",
      password_sha256: "abc123",
    });
    const code = extractCodeFromResult(postResult);

    const result = server.handleTokenRequest({
      grant_type: "authorization_code",
      code,
      code_verifier: pkce.verifier,
      redirect_uri: "http://evil.com/callback",
    });
    expect(result.status).toBe(400);
    expect(result.body.error).toBe("invalid_grant");
  });

  test("rejects missing code", () => {
    const result = server.handleTokenRequest({
      grant_type: "authorization_code",
      code_verifier: "x",
    });
    expect(result.status).toBe(400);
    expect(result.body.error).toBe("invalid_request");
  });

  test("rejects missing code_verifier", () => {
    const result = server.handleTokenRequest({
      grant_type: "authorization_code",
      code: "some-code",
    });
    expect(result.status).toBe(400);
    expect(result.body.error).toBe("invalid_request");
  });
});

// ---------------------------------------------------------------------------
// Token endpoint — refresh_token grant
// ---------------------------------------------------------------------------
describe("OAuthServer — token (refresh_token)", () => {
  let server;
  beforeEach(() => { server = createServer(); });
  afterEach(() => { server.close(); });

  test("issues new tokens from valid refresh_token", async () => {
    const tokens = await fullAuthCodeFlow(server);
    const result = server.handleTokenRequest({
      grant_type: "refresh_token",
      refresh_token: tokens.refresh_token,
    });
    expect(result.status).toBe(200);
    expect(result.body.access_token).toBeTruthy();
    expect(result.body.refresh_token).toBeTruthy();
    // New tokens should differ
    expect(result.body.access_token).not.toBe(tokens.access_token);
    expect(result.body.refresh_token).not.toBe(tokens.refresh_token);
  });

  test("rotates refresh_token (old one becomes invalid)", async () => {
    const tokens = await fullAuthCodeFlow(server);
    const oldRefresh = tokens.refresh_token;

    // Use refresh token
    const result1 = server.handleTokenRequest({
      grant_type: "refresh_token",
      refresh_token: oldRefresh,
    });
    expect(result1.status).toBe(200);

    // Old refresh token should now be invalid
    const result2 = server.handleTokenRequest({
      grant_type: "refresh_token",
      refresh_token: oldRefresh,
    });
    expect(result2.status).toBe(400);
    expect(result2.body.error).toBe("invalid_grant");
  });

  test("rejects invalid refresh_token", () => {
    const result = server.handleTokenRequest({
      grant_type: "refresh_token",
      refresh_token: "invalid-token",
    });
    expect(result.status).toBe(400);
    expect(result.body.error).toBe("invalid_grant");
  });

  test("rejects missing refresh_token", () => {
    const result = server.handleTokenRequest({
      grant_type: "refresh_token",
    });
    expect(result.status).toBe(400);
    expect(result.body.error).toBe("invalid_request");
  });

  test("rejects mismatched client_id on refresh", async () => {
    const tokens = await fullAuthCodeFlow(server);
    const result = server.handleTokenRequest({
      grant_type: "refresh_token",
      refresh_token: tokens.refresh_token,
      client_id: "wrong-client",
    });
    expect(result.status).toBe(400);
    expect(result.body.error).toBe("invalid_grant");
  });
});

// ---------------------------------------------------------------------------
// Token endpoint — unsupported grant type
// ---------------------------------------------------------------------------
describe("OAuthServer — token (unsupported)", () => {
  test("rejects unsupported grant_type", () => {
    const server = createServer();
    const result = server.handleTokenRequest({
      grant_type: "client_credentials",
    });
    expect(result.status).toBe(400);
    expect(result.body.error).toBe("unsupported_grant_type");
    server.close();
  });
});

// ---------------------------------------------------------------------------
// Dynamic Client Registration
// ---------------------------------------------------------------------------
describe("OAuthServer — handleRegister", () => {
  let server;
  beforeEach(() => { server = createServer(); });
  afterEach(() => { server.close(); });

  test("registers a client with valid metadata", () => {
    const result = server.handleRegister({
      redirect_uris: ["http://localhost/callback"],
      client_name: "My App",
    });
    expect(result.status).toBe(201);
    expect(result.body.client_id).toBeTruthy();
    expect(result.body.client_name).toBe("My App");
    expect(result.body.redirect_uris).toEqual(["http://localhost/callback"]);
    expect(result.body.grant_types).toContain("authorization_code");
  });

  test("includes client_id_issued_at in response", () => {
    const before = Math.floor(Date.now() / 1000);
    const result = server.handleRegister({
      redirect_uris: ["http://localhost/cb"],
    });
    const after = Math.floor(Date.now() / 1000);
    expect(result.body.client_id_issued_at).toBeGreaterThanOrEqual(before);
    expect(result.body.client_id_issued_at).toBeLessThanOrEqual(after);
  });

  test("registered client can be retrieved from store", () => {
    const result = server.handleRegister({
      redirect_uris: ["http://localhost/cb"],
    });
    const stored = server.store.getClient(result.body.client_id);
    expect(stored).toBeTruthy();
    expect(stored.client_id).toBe(result.body.client_id);
  });

  test("rejects missing redirect_uris", () => {
    const result = server.handleRegister({});
    expect(result.status).toBe(400);
    expect(result.body.error).toBe("invalid_client_metadata");
  });

  test("rejects empty redirect_uris array", () => {
    const result = server.handleRegister({ redirect_uris: [] });
    expect(result.status).toBe(400);
  });

  test("rejects invalid redirect_uri URL", () => {
    const result = server.handleRegister({ redirect_uris: ["not-a-url"] });
    expect(result.status).toBe(400);
    expect(result.body.error_description).toContain("Invalid redirect_uri");
  });

  test("defaults client_name to empty string", () => {
    const result = server.handleRegister({
      redirect_uris: ["http://localhost/cb"],
    });
    expect(result.body.client_name).toBe("");
  });
});

// ---------------------------------------------------------------------------
// Token validation
// ---------------------------------------------------------------------------
describe("OAuthServer — validateAccessToken", () => {
  let server;
  beforeEach(() => { server = createServer(); });
  afterEach(() => { server.close(); });

  test("returns token data for valid access token", async () => {
    const tokens = await fullAuthCodeFlow(server);
    const data = server.validateAccessToken(tokens.access_token);
    expect(data).toBeTruthy();
    expect(data.userId).toBe("admin");
    expect(data.gkillSessionId).toBe("gkill-session-001");
    expect(data.clientId).toBe("test-client");
  });

  test("returns null for invalid token", () => {
    expect(server.validateAccessToken("bad-token")).toBeNull();
  });

  test("returns null for empty token", () => {
    expect(server.validateAccessToken("")).toBeNull();
  });

  test("returns null for expired access token", async () => {
    vi.useFakeTimers();
    try {
      const tokens = await fullAuthCodeFlow(server);
      vi.advanceTimersByTime(60 * 60 * 1000 + 1); // 1 hour + 1ms
      expect(server.validateAccessToken(tokens.access_token)).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });
});

// ---------------------------------------------------------------------------
// extractBearerToken
// ---------------------------------------------------------------------------
describe("OAuthServer.extractBearerToken", () => {
  test("extracts token from valid Bearer header", () => {
    expect(OAuthServer.extractBearerToken("Bearer abc123")).toBe("abc123");
  });

  test("returns empty for non-Bearer header", () => {
    expect(OAuthServer.extractBearerToken("Basic abc123")).toBe("");
  });

  test("returns empty for empty string", () => {
    expect(OAuthServer.extractBearerToken("")).toBe("");
  });

  test("returns empty for null/undefined", () => {
    expect(OAuthServer.extractBearerToken(null)).toBe("");
    expect(OAuthServer.extractBearerToken(undefined)).toBe("");
  });
});

// ---------------------------------------------------------------------------
// Full end-to-end flow
// ---------------------------------------------------------------------------
describe("OAuthServer — full E2E flow", () => {
  test("authorize → token → validate → refresh → validate", async () => {
    const server = createServer();
    const pkce = makeS256Pair();

    // 1. GET authorize (show form)
    const getResult = server.handleAuthorizeGet(authorizeParams({ code_challenge: pkce.challenge }));
    expect(getResult.status).toBe(200);

    // 2. POST authorize (login)
    const postResult = await server.handleAuthorizePost({
      ...authorizeParams({ code_challenge: pkce.challenge }),
      user_id: "admin",
      password_sha256: "abc123",
    });
    const code = extractCodeFromResult(postResult);

    // 3. Token exchange
    const tokenResult = server.handleTokenRequest({
      grant_type: "authorization_code",
      code,
      code_verifier: pkce.verifier,
      client_id: "test-client",
      redirect_uri: "http://localhost/callback",
    });
    expect(tokenResult.status).toBe(200);

    // 4. Validate access token
    const tokenData = server.validateAccessToken(tokenResult.body.access_token);
    expect(tokenData.userId).toBe("admin");
    expect(tokenData.gkillSessionId).toBe("gkill-session-001");

    // 5. Refresh
    const refreshResult = server.handleTokenRequest({
      grant_type: "refresh_token",
      refresh_token: tokenResult.body.refresh_token,
    });
    expect(refreshResult.status).toBe(200);

    // 6. Validate new access token
    const newData = server.validateAccessToken(refreshResult.body.access_token);
    expect(newData.userId).toBe("admin");

    // 7. Old access token still valid (not rotated, only refresh token rotates)
    // Actually access tokens are independent, so the old one is still valid until expiry
    const oldData = server.validateAccessToken(tokenResult.body.access_token);
    expect(oldData).toBeTruthy();

    server.close();
  });
});

// ---------------------------------------------------------------------------
// RFC 8707 resource parameter
// ---------------------------------------------------------------------------
describe("OAuthServer — resource parameter (RFC 8707)", () => {
  let server;
  beforeEach(() => { server = createServer(); });
  afterEach(() => { server.close(); });

  test("stores resource in authorization code data", async () => {
    const pkce = makeS256Pair();
    const params = authorizeParams({ code_challenge: pkce.challenge, resource: "http://localhost:8808/mcp" });
    const postResult = await server.handleAuthorizePost({
      ...params,
      user_id: "admin",
      password_sha256: "abc123",
    });
    expect(postResult.status).toBe(200);
    expect(postResult.body).toContain("ログイン成功");
  });

  test("token exchange succeeds with matching resource", async () => {
    const pkce = makeS256Pair();
    const resource = "http://localhost:8808/mcp";
    const params = authorizeParams({ code_challenge: pkce.challenge, resource });
    const postResult = await server.handleAuthorizePost({
      ...params,
      user_id: "admin",
      password_sha256: "abc123",
    });
    const code = extractCodeFromResult(postResult);

    const result = server.handleTokenRequest({
      grant_type: "authorization_code",
      code,
      code_verifier: pkce.verifier,
      resource,
    });
    expect(result.status).toBe(200);
    expect(result.body.access_token).toBeTruthy();
  });

  test("token exchange rejects mismatched resource", async () => {
    const pkce = makeS256Pair();
    const params = authorizeParams({ code_challenge: pkce.challenge, resource: "http://localhost:8808/mcp" });
    const postResult = await server.handleAuthorizePost({
      ...params,
      user_id: "admin",
      password_sha256: "abc123",
    });
    const code = extractCodeFromResult(postResult);

    const result = server.handleTokenRequest({
      grant_type: "authorization_code",
      code,
      code_verifier: pkce.verifier,
      resource: "http://evil.com/mcp",
    });
    expect(result.status).toBe(400);
    expect(result.body.error).toBe("invalid_grant");
    expect(result.body.error_description).toContain("resource");
  });

  test("token exchange succeeds when resource omitted from token request", async () => {
    const pkce = makeS256Pair();
    const params = authorizeParams({ code_challenge: pkce.challenge, resource: "http://localhost:8808/mcp" });
    const postResult = await server.handleAuthorizePost({
      ...params,
      user_id: "admin",
      password_sha256: "abc123",
    });
    const code = extractCodeFromResult(postResult);

    const result = server.handleTokenRequest({
      grant_type: "authorization_code",
      code,
      code_verifier: pkce.verifier,
    });
    expect(result.status).toBe(200);
  });

  test("login form preserves resource on auth failure", async () => {
    const pkce = makeS256Pair();
    const params = authorizeParams({ code_challenge: pkce.challenge, resource: "http://localhost:8808/mcp" });
    const result = await server.handleAuthorizePost({
      ...params,
      user_id: "admin",
      password_sha256: "wrong",
    });
    expect(result.status).toBe(200);
    expect(result.body).toContain("http://localhost:8808/mcp");
  });
});

// ---------------------------------------------------------------------------
// DCR redirect_uri validation at authorize time
// ---------------------------------------------------------------------------
describe("OAuthServer — redirect_uri validation against DCR", () => {
  let server;
  beforeEach(() => { server = createServer(); });
  afterEach(() => { server.close(); });

  test("rejects redirect_uri not matching DCR-registered client", () => {
    const regResult = server.handleRegister({
      redirect_uris: ["http://localhost/callback"],
      client_name: "Test",
    });
    const clientId = regResult.body.client_id;

    const result = server.handleAuthorizeGet(authorizeParams({
      client_id: clientId,
      redirect_uri: "http://evil.com/callback",
    }));
    expect(result.status).toBe(400);
    expect(result.body).toContain("redirect_uri does not match");
  });

  test("accepts redirect_uri matching DCR-registered client", () => {
    const regResult = server.handleRegister({
      redirect_uris: ["http://localhost/callback"],
    });
    const clientId = regResult.body.client_id;

    const result = server.handleAuthorizeGet(authorizeParams({
      client_id: clientId,
      redirect_uri: "http://localhost/callback",
    }));
    expect(result.status).toBe(200);
  });

  test("allows any redirect_uri for non-DCR clients", () => {
    const result = server.handleAuthorizeGet(authorizeParams({
      client_id: "pre-registered-client",
      redirect_uri: "http://any-url.com/callback",
    }));
    expect(result.status).toBe(200);
  });
});
