import { OAuthStore, TTL, generateToken } from "../lib/oauth-store.mjs";

// ---------------------------------------------------------------------------
// generateToken
// ---------------------------------------------------------------------------
describe("generateToken", () => {
  test("returns a 64-character hex string", () => {
    const token = generateToken();
    expect(token).toMatch(/^[0-9a-f]{64}$/);
  });

  test("generates unique tokens", () => {
    const tokens = new Set(Array.from({ length: 50 }, () => generateToken()));
    expect(tokens.size).toBe(50);
  });
});

// ---------------------------------------------------------------------------
// TTL constants
// ---------------------------------------------------------------------------
describe("TTL constants", () => {
  test("AUTHORIZATION_CODE is 5 minutes", () => {
    expect(TTL.AUTHORIZATION_CODE).toBe(5 * 60 * 1000);
  });

  test("ACCESS_TOKEN is 1 hour", () => {
    expect(TTL.ACCESS_TOKEN).toBe(60 * 60 * 1000);
  });

  test("REFRESH_TOKEN is 30 days", () => {
    expect(TTL.REFRESH_TOKEN).toBe(30 * 24 * 60 * 60 * 1000);
  });

  test("TTL object is frozen", () => {
    expect(Object.isFrozen(TTL)).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// Authorization Codes
// ---------------------------------------------------------------------------
describe("OAuthStore — authorization codes", () => {
  let store;
  beforeEach(() => { store = new OAuthStore(); });

  test("putCode + getAndDeleteCode returns the stored data", () => {
    const data = { clientId: "c1", redirectUri: "http://localhost/cb" };
    store.putCode("code1", data);
    expect(store.getAndDeleteCode("code1")).toEqual(data);
  });

  test("getAndDeleteCode returns null for unknown code", () => {
    expect(store.getAndDeleteCode("nonexistent")).toBeNull();
  });

  test("getAndDeleteCode deletes the code (one-time use)", () => {
    store.putCode("code1", { clientId: "c1" });
    store.getAndDeleteCode("code1");
    expect(store.getAndDeleteCode("code1")).toBeNull();
  });

  test("getAndDeleteCode returns null for expired code", () => {
    vi.useFakeTimers();
    try {
      store.putCode("code1", { clientId: "c1" }, 1000); // 1 second TTL
      vi.advanceTimersByTime(1001);
      expect(store.getAndDeleteCode("code1")).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });

  test("getAndDeleteCode returns data before TTL expires", () => {
    vi.useFakeTimers();
    try {
      store.putCode("code1", { clientId: "c1" }, 5000);
      vi.advanceTimersByTime(4999);
      expect(store.getAndDeleteCode("code1")).toEqual({ clientId: "c1" });
    } finally {
      vi.useRealTimers();
    }
  });
});

// ---------------------------------------------------------------------------
// Access Tokens
// ---------------------------------------------------------------------------
describe("OAuthStore — access tokens", () => {
  let store;
  beforeEach(() => { store = new OAuthStore(); });

  test("putAccessToken + getAccessToken returns data", () => {
    const data = { clientId: "c1", userId: "admin" };
    store.putAccessToken("tok1", data);
    expect(store.getAccessToken("tok1")).toEqual(data);
  });

  test("getAccessToken returns null for unknown token", () => {
    expect(store.getAccessToken("unknown")).toBeNull();
  });

  test("getAccessToken returns null for expired token and cleans up", () => {
    vi.useFakeTimers();
    try {
      store.putAccessToken("tok1", { clientId: "c1" }, 1000);
      vi.advanceTimersByTime(1001);
      expect(store.getAccessToken("tok1")).toBeNull();
      // Entry should be deleted
      expect(store.accessTokens.has("tok1")).toBe(false);
    } finally {
      vi.useRealTimers();
    }
  });

  test("getAccessToken returns data before TTL", () => {
    vi.useFakeTimers();
    try {
      store.putAccessToken("tok1", { clientId: "c1" }, 5000);
      vi.advanceTimersByTime(4999);
      expect(store.getAccessToken("tok1")).toEqual({ clientId: "c1" });
    } finally {
      vi.useRealTimers();
    }
  });

  test("deleteAccessToken removes the token", () => {
    store.putAccessToken("tok1", { clientId: "c1" });
    store.deleteAccessToken("tok1");
    expect(store.getAccessToken("tok1")).toBeNull();
  });

  test("can store multiple tokens", () => {
    store.putAccessToken("tok1", { userId: "a" });
    store.putAccessToken("tok2", { userId: "b" });
    expect(store.getAccessToken("tok1")).toEqual({ userId: "a" });
    expect(store.getAccessToken("tok2")).toEqual({ userId: "b" });
  });
});

// ---------------------------------------------------------------------------
// Refresh Tokens
// ---------------------------------------------------------------------------
describe("OAuthStore — refresh tokens", () => {
  let store;
  beforeEach(() => { store = new OAuthStore(); });

  test("putRefreshToken + getRefreshToken returns data", () => {
    const data = { clientId: "c1", userId: "admin" };
    store.putRefreshToken("rt1", data);
    expect(store.getRefreshToken("rt1")).toEqual(data);
  });

  test("getRefreshToken returns null for unknown token", () => {
    expect(store.getRefreshToken("unknown")).toBeNull();
  });

  test("getRefreshToken returns null for expired token", () => {
    vi.useFakeTimers();
    try {
      store.putRefreshToken("rt1", { clientId: "c1" }, 1000);
      vi.advanceTimersByTime(1001);
      expect(store.getRefreshToken("rt1")).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });

  test("deleteRefreshToken removes the token", () => {
    store.putRefreshToken("rt1", { clientId: "c1" });
    store.deleteRefreshToken("rt1");
    expect(store.getRefreshToken("rt1")).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// Client Registrations
// ---------------------------------------------------------------------------
describe("OAuthStore — clients", () => {
  let store;
  beforeEach(() => { store = new OAuthStore(); });

  test("putClient + getClient returns metadata", () => {
    const meta = { client_name: "Test App", redirect_uris: ["http://localhost/cb"] };
    store.putClient("client1", meta);
    expect(store.getClient("client1")).toEqual(meta);
  });

  test("getClient returns null for unknown client", () => {
    expect(store.getClient("unknown")).toBeNull();
  });

  test("putClient overwrites existing registration", () => {
    store.putClient("c1", { client_name: "Old" });
    store.putClient("c1", { client_name: "New" });
    expect(store.getClient("c1")).toEqual({ client_name: "New" });
  });
});

// ---------------------------------------------------------------------------
// sweep
// ---------------------------------------------------------------------------
describe("OAuthStore — sweep", () => {
  test("removes expired entries from all maps", () => {
    vi.useFakeTimers();
    try {
      const store = new OAuthStore();
      store.putCode("c1", { x: 1 }, 1000);
      store.putAccessToken("t1", { x: 2 }, 2000);
      store.putRefreshToken("r1", { x: 3 }, 3000);

      // Before any expiry
      store.sweep();
      expect(store.stats()).toEqual({ codes: 1, accessTokens: 1, refreshTokens: 1, clients: 0 });

      // After code expires
      vi.advanceTimersByTime(1001);
      store.sweep();
      expect(store.stats()).toEqual({ codes: 0, accessTokens: 1, refreshTokens: 1, clients: 0 });

      // After access token expires
      vi.advanceTimersByTime(1000);
      store.sweep();
      expect(store.stats()).toEqual({ codes: 0, accessTokens: 0, refreshTokens: 1, clients: 0 });

      // After refresh token expires
      vi.advanceTimersByTime(1000);
      store.sweep();
      expect(store.stats()).toEqual({ codes: 0, accessTokens: 0, refreshTokens: 0, clients: 0 });
    } finally {
      vi.useRealTimers();
    }
  });

  test("does not remove unexpired entries", () => {
    vi.useFakeTimers();
    try {
      const store = new OAuthStore();
      store.putCode("c1", { x: 1 }, 10000);
      store.putAccessToken("t1", { x: 2 }, 10000);

      vi.advanceTimersByTime(5000);
      store.sweep();
      expect(store.stats().codes).toBe(1);
      expect(store.stats().accessTokens).toBe(1);
    } finally {
      vi.useRealTimers();
    }
  });

  test("does not remove client registrations (no TTL)", () => {
    const store = new OAuthStore();
    store.putClient("c1", { client_name: "App" });
    store.sweep();
    expect(store.getClient("c1")).toEqual({ client_name: "App" });
  });
});

// ---------------------------------------------------------------------------
// stats
// ---------------------------------------------------------------------------
describe("OAuthStore — stats", () => {
  test("returns zero counts for empty store", () => {
    const store = new OAuthStore();
    expect(store.stats()).toEqual({ codes: 0, accessTokens: 0, refreshTokens: 0, clients: 0 });
  });

  test("returns correct counts", () => {
    const store = new OAuthStore();
    store.putCode("c1", {});
    store.putCode("c2", {});
    store.putAccessToken("t1", {});
    store.putClient("cl1", {});
    expect(store.stats()).toEqual({ codes: 2, accessTokens: 1, refreshTokens: 0, clients: 1 });
  });
});

// ---------------------------------------------------------------------------
// startCleanup / stopCleanup
// ---------------------------------------------------------------------------
describe("OAuthStore — cleanup interval", () => {
  test("startCleanup triggers periodic sweep", () => {
    vi.useFakeTimers();
    try {
      const store = new OAuthStore();
      store.putCode("c1", {}, 100);
      store.startCleanup(500);

      vi.advanceTimersByTime(500);
      // Code should have expired (TTL 100ms) and been swept
      expect(store.codes.size).toBe(0);

      store.stopCleanup();
    } finally {
      vi.useRealTimers();
    }
  });

  test("stopCleanup stops the interval", () => {
    vi.useFakeTimers();
    try {
      const store = new OAuthStore();
      store.startCleanup(100);
      store.stopCleanup();
      store.putCode("c1", {}, 50);

      vi.advanceTimersByTime(200);
      // Sweep should NOT have run since cleanup was stopped
      expect(store.codes.size).toBe(1);
    } finally {
      vi.useRealTimers();
    }
  });

  test("startCleanup replaces previous interval", () => {
    vi.useFakeTimers();
    try {
      const store = new OAuthStore();
      store.startCleanup(100);
      store.startCleanup(200); // should replace
      store.stopCleanup();
    } finally {
      vi.useRealTimers();
    }
  });
});

// ---------------------------------------------------------------------------
// Persistence (JSON file)
// ---------------------------------------------------------------------------
import { existsSync, unlinkSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";

describe("OAuthStore — persistence", () => {
  let tmpDir;
  let persistPath;

  beforeEach(() => {
    tmpDir = join(tmpdir(), `oauth-store-test-${Date.now()}-${Math.random().toString(36).slice(2)}`);
    mkdirSync(tmpDir, { recursive: true });
    persistPath = join(tmpDir, "mcp_oauth_state.json");
  });

  afterEach(() => {
    try { unlinkSync(persistPath); } catch {}
    try { unlinkSync(tmpDir); } catch {}
  });

  test("save() writes refresh tokens and clients to JSON file", () => {
    const store = new OAuthStore(persistPath);
    store.putRefreshToken("rt1", { userId: "admin" }, 60000);
    store.putClient("c1", { client_name: "Test" });

    expect(existsSync(persistPath)).toBe(true);
    const data = JSON.parse(readFileSync(persistPath, "utf8"));
    expect(data.refreshTokens.rt1.value.userId).toBe("admin");
    expect(data.clients.c1.client_name).toBe("Test");
  });

  test("load() restores refresh tokens and clients from file", () => {
    // Write with store1
    const store1 = new OAuthStore(persistPath);
    store1.putRefreshToken("rt1", { userId: "admin" }, 60000);
    store1.putClient("c1", { client_name: "App" });

    // Load with store2
    const store2 = new OAuthStore(persistPath);
    store2.load();
    expect(store2.getRefreshToken("rt1")).toEqual({ userId: "admin" });
    expect(store2.getClient("c1")).toEqual({ client_name: "App" });
  });

  test("load() skips expired refresh tokens", () => {
    vi.useFakeTimers();
    try {
      const store1 = new OAuthStore(persistPath);
      store1.putRefreshToken("rt-expired", { userId: "x" }, 100);
      store1.putRefreshToken("rt-valid", { userId: "y" }, 60000);

      vi.advanceTimersByTime(200); // rt-expired is now past its expiresAt

      const store2 = new OAuthStore(persistPath);
      store2.load();
      expect(store2.getRefreshToken("rt-expired")).toBeNull();
      expect(store2.getRefreshToken("rt-valid")).toEqual({ userId: "y" });
    } finally {
      vi.useRealTimers();
    }
  });

  test("load() does not error when file does not exist", () => {
    const store = new OAuthStore(join(tmpDir, "nonexistent.json"));
    expect(() => store.load()).not.toThrow();
    expect(store.stats().refreshTokens).toBe(0);
  });

  test("load() does not error on invalid JSON", () => {
    const badPath = join(tmpDir, "bad.json");
    writeFileSync(badPath, "not json!", "utf8");
    const store = new OAuthStore(badPath);
    expect(() => store.load()).not.toThrow();
    expect(store.stats().refreshTokens).toBe(0);
  });

  test("putRefreshToken() auto-saves", () => {
    const store = new OAuthStore(persistPath);
    store.putRefreshToken("rt1", { x: 1 }, 60000);
    expect(existsSync(persistPath)).toBe(true);
  });

  test("putClient() auto-saves", () => {
    const store = new OAuthStore(persistPath);
    store.putClient("c1", { y: 2 });
    expect(existsSync(persistPath)).toBe(true);
  });

  test("deleteRefreshToken() auto-saves", () => {
    const store = new OAuthStore(persistPath);
    store.putRefreshToken("rt1", { x: 1 }, 60000);
    store.deleteRefreshToken("rt1");

    const data = JSON.parse(readFileSync(persistPath, "utf8"));
    expect(data.refreshTokens.rt1).toBeUndefined();
  });

  test("no-op when persistPath is null", () => {
    const store = new OAuthStore(null);
    store.putRefreshToken("rt1", { x: 1 });
    store.putClient("c1", { y: 2 });
    expect(() => store.load()).not.toThrow();
    // No file should be created anywhere
  });
});
