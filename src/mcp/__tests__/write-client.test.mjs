/**
 * Tests for GkillWriteClient from gkill-write-server.mjs.
 *
 * All network calls are mocked via globalThis.fetch so no real server is needed.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from "vitest";
import { GkillWriteClient } from "../gkill-write-server.mjs";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function mockFetchOk(body = {}) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(body),
  });
}

function mockFetchError(status, body = {}) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve(body),
  });
}

// Save and restore env + fetch between tests.
let savedEnv;
let savedFetch;

beforeEach(() => {
  savedEnv = { ...process.env };
  savedFetch = globalThis.fetch;
  delete process.env.GKILL_BASE_URL;
  delete process.env.GKILL_USER;
  delete process.env.GKILL_PASSWORD_SHA256;
  delete process.env.GKILL_PASSWORD;
  delete process.env.GKILL_LOCALE;
  delete process.env.GKILL_SESSION_ID;
  delete process.env.GKILL_INSECURE;
  delete process.env.GKILL_FETCH_TIMEOUT_MS;
});

afterEach(() => {
  process.env = savedEnv;
  globalThis.fetch = savedFetch;
});

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------
describe("GkillWriteClient constructor", () => {
  test("sets defaults when env vars are absent", () => {
    const client = new GkillWriteClient();
    expect(client.baseUrl).toBe("http://127.0.0.1:9999");
    expect(client.userId).toBe("");
    expect(client.passwordSha256).toBe("");
    expect(client.password).toBe("");
    expect(client.defaultLocale).toBe("ja");
    expect(client.sessionId).toBe("");
    expect(client.dispatcher).toBeNull();
  });

  test("reads env vars when set", () => {
    process.env.GKILL_BASE_URL = "https://example.com:1234";
    process.env.GKILL_USER = "testuser";
    process.env.GKILL_PASSWORD_SHA256 = "abc123";
    process.env.GKILL_LOCALE = "en";
    process.env.GKILL_SESSION_ID = "sess-42";

    const client = new GkillWriteClient();
    expect(client.baseUrl).toBe("https://example.com:1234");
    expect(client.userId).toBe("testuser");
    expect(client.passwordSha256).toBe("abc123");
    expect(client.defaultLocale).toBe("en");
    expect(client.sessionId).toBe("sess-42");
  });

  test("sets dispatcher when GKILL_INSECURE=true", () => {
    process.env.GKILL_INSECURE = "true";
    const client = new GkillWriteClient();
    expect(client.dispatcher).not.toBeNull();
  });
});

// ---------------------------------------------------------------------------
// resolvePasswordSha256
// ---------------------------------------------------------------------------
describe("resolvePasswordSha256", () => {
  test("returns passwordSha256 when set", () => {
    process.env.GKILL_PASSWORD_SHA256 = "abc123";
    const client = new GkillWriteClient();
    expect(client.resolvePasswordSha256()).toBe("abc123");
  });

  test("hashes password when only password is set", () => {
    process.env.GKILL_PASSWORD = "secret";
    const client = new GkillWriteClient();
    const hash = client.resolvePasswordSha256();
    expect(hash).toHaveLength(64);
    expect(hash).not.toBe("secret");
  });

  test("returns empty string when no credentials", () => {
    const client = new GkillWriteClient();
    expect(client.resolvePasswordSha256()).toBe("");
  });
});

// ---------------------------------------------------------------------------
// buildApiUrl
// ---------------------------------------------------------------------------
describe("buildApiUrl", () => {
  test("constructs correct URL", () => {
    const client = new GkillWriteClient();
    expect(client.buildApiUrl("/api/add_kmemo")).toBe("http://127.0.0.1:9999/api/add_kmemo");
  });
});

// ---------------------------------------------------------------------------
// hasErrors / hasAuthErrors / formatErrors
// ---------------------------------------------------------------------------
describe("error helpers", () => {
  test("hasErrors returns false for empty errors", () => {
    const client = new GkillWriteClient();
    expect(client.hasErrors({ errors: [] })).toBe(false);
  });

  test("hasErrors returns true for non-empty errors", () => {
    const client = new GkillWriteClient();
    expect(client.hasErrors({ errors: [{ error_code: "E1" }] })).toBe(true);
  });

  test("hasAuthErrors detects auth error codes", () => {
    const client = new GkillWriteClient();
    expect(client.hasAuthErrors({ errors: [{ error_code: "ERR000013" }] })).toBe(true);
    expect(client.hasAuthErrors({ errors: [{ error_code: "ERR999999" }] })).toBe(false);
  });

  test("formatErrors joins error messages", () => {
    const client = new GkillWriteClient();
    const result = client.formatErrors({
      errors: [
        { error_code: "E1", error_message: "msg1" },
        { error_code: "E2", error_message: "msg2" },
      ],
    });
    expect(result).toBe("E1: msg1; E2: msg2");
  });
});

// ---------------------------------------------------------------------------
// login
// ---------------------------------------------------------------------------
describe("login", () => {
  test("returns existing session_id without calling API", async () => {
    process.env.GKILL_SESSION_ID = "existing-session";
    const client = new GkillWriteClient();
    const result = await client.login();
    expect(result).toBe("existing-session");
  });

  test("calls /api/login and returns session_id", async () => {
    process.env.GKILL_USER = "admin";
    process.env.GKILL_PASSWORD_SHA256 = "hash";
    mockFetchOk({ session_id: "new-session", errors: [] });

    const client = new GkillWriteClient();
    const result = await client.login();
    expect(result).toBe("new-session");
    expect(client.sessionId).toBe("new-session");
  });

  test("throws when credentials are missing", async () => {
    const client = new GkillWriteClient();
    await expect(client.login()).rejects.toThrow("Missing login credentials");
  });

  test("throws when login returns errors", async () => {
    process.env.GKILL_USER = "admin";
    process.env.GKILL_PASSWORD_SHA256 = "hash";
    mockFetchOk({ errors: [{ error_code: "E1", error_message: "bad password" }] });

    const client = new GkillWriteClient();
    await expect(client.login()).rejects.toThrow("Login failed");
  });
});

// ---------------------------------------------------------------------------
// callWrite
// ---------------------------------------------------------------------------
describe("callWrite", () => {
  test("posts to the correct endpoint with auth", async () => {
    process.env.GKILL_USER = "admin";
    process.env.GKILL_PASSWORD_SHA256 = "hash";

    // First call = login, second = actual API call
    let callCount = 0;
    globalThis.fetch = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount === 1) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ session_id: "sess", errors: [] }),
        });
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ added_kmemo: { id: "123" }, errors: [] }),
      });
    });

    const client = new GkillWriteClient();
    const result = await client.callWrite("/api/add_kmemo", { kmemo: {} }, true);
    expect(result.added_kmemo.id).toBe("123");
  });

  test("retries on auth error", async () => {
    process.env.GKILL_USER = "admin";
    process.env.GKILL_PASSWORD_SHA256 = "hash";

    let callCount = 0;
    globalThis.fetch = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount === 1) {
        // Initial login
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ session_id: "old-sess", errors: [] }),
        });
      }
      if (callCount === 2) {
        // First attempt returns auth error
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ errors: [{ error_code: "ERR000013", error_message: "session expired" }] }),
        });
      }
      if (callCount === 3) {
        // Re-login
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ session_id: "new-sess", errors: [] }),
        });
      }
      // Retry succeeds
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ added_kmemo: { id: "456" }, errors: [] }),
      });
    });

    const client = new GkillWriteClient();
    const result = await client.callWrite("/api/add_kmemo", { kmemo: {} }, true);
    expect(result.added_kmemo.id).toBe("456");
    expect(callCount).toBe(4);
  });

  test("uses session override when provided", async () => {
    mockFetchOk({ added_kmemo: { id: "789" }, errors: [] });

    const client = new GkillWriteClient();
    client.sessionId = "default-sess";
    const result = await client.callWrite("/api/add_kmemo", { kmemo: {} }, true, "override-sess");

    const callBody = JSON.parse(globalThis.fetch.mock.calls[0][1].body);
    expect(callBody.session_id).toBe("override-sess");
  });
});
