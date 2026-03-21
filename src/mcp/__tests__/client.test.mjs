/**
 * Tests for GkillReadClient from gkill-read-server.mjs.
 *
 * All network calls are mocked via globalThis.fetch so no real server is needed.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from "vitest";
import { GkillReadClient } from "../gkill-read-server.mjs";

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
  // Clear gkill-related env vars so defaults are used.
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
describe("GkillReadClient constructor", () => {
  test("sets defaults when env vars are absent", () => {
    const client = new GkillReadClient();
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

    const client = new GkillReadClient();
    expect(client.baseUrl).toBe("https://example.com:1234");
    expect(client.userId).toBe("testuser");
    expect(client.passwordSha256).toBe("abc123");
    expect(client.defaultLocale).toBe("en");
    expect(client.sessionId).toBe("sess-42");
  });
});

// ---------------------------------------------------------------------------
// buildApiUrl
// ---------------------------------------------------------------------------
describe("buildApiUrl", () => {
  test("constructs correct URL", () => {
    const client = new GkillReadClient();
    expect(client.buildApiUrl("/api/login")).toBe("http://127.0.0.1:9999/api/login");
  });

  test("constructs URL with custom base", () => {
    process.env.GKILL_BASE_URL = "https://gkill.example.com";
    const client = new GkillReadClient();
    expect(client.buildApiUrl("/api/get_kyous_mcp")).toBe(
      "https://gkill.example.com/api/get_kyous_mcp",
    );
  });
});

// ---------------------------------------------------------------------------
// resolvePasswordSha256
// ---------------------------------------------------------------------------
describe("resolvePasswordSha256", () => {
  test("returns GKILL_PASSWORD_SHA256 directly when set", () => {
    process.env.GKILL_PASSWORD_SHA256 = "deadbeef";
    const client = new GkillReadClient();
    expect(client.resolvePasswordSha256()).toBe("deadbeef");
  });

  test("computes SHA-256 from GKILL_PASSWORD when hash is not set", async () => {
    process.env.GKILL_PASSWORD = "secret";
    const client = new GkillReadClient();
    const hash = client.resolvePasswordSha256();
    // SHA-256 of "secret"
    const { createHash } = await import("node:crypto");
    const expected = createHash("sha256").update("secret").digest("hex");
    expect(hash).toBe(expected);
  });

  test("returns empty string when neither env var is set", () => {
    const client = new GkillReadClient();
    expect(client.resolvePasswordSha256()).toBe("");
  });
});

// ---------------------------------------------------------------------------
// hasErrors
// ---------------------------------------------------------------------------
describe("hasErrors", () => {
  test("returns true when errors array is non-empty", () => {
    const client = new GkillReadClient();
    expect(client.hasErrors({ errors: [{ error_code: "ERR" }] })).toBe(true);
  });

  test("returns false when errors array is empty", () => {
    const client = new GkillReadClient();
    expect(client.hasErrors({ errors: [] })).toBe(false);
  });

  test("returns false when errors field is missing", () => {
    const client = new GkillReadClient();
    expect(client.hasErrors({})).toBe(false);
  });

  test("returns false for null/undefined", () => {
    const client = new GkillReadClient();
    expect(client.hasErrors(null)).toBe(false);
    expect(client.hasErrors(undefined)).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// login
// ---------------------------------------------------------------------------
describe("login", () => {
  test("calls fetch with correct URL and body", async () => {
    process.env.GKILL_USER = "admin";
    process.env.GKILL_PASSWORD_SHA256 = "hash123";
    mockFetchOk({ session_id: "test-session", errors: [] });

    const client = new GkillReadClient();
    const sessionId = await client.login();

    expect(sessionId).toBe("test-session");
    expect(globalThis.fetch).toHaveBeenCalledTimes(1);

    const [url, opts] = globalThis.fetch.mock.calls[0];
    expect(url).toBe("http://127.0.0.1:9999/api/login");
    expect(opts.method).toBe("POST");
    expect(JSON.parse(opts.body)).toEqual({
      user_id: "admin",
      password_sha256: "hash123",
      locale_name: "ja",
    });
  });

  test("returns existing sessionId without calling fetch", async () => {
    process.env.GKILL_SESSION_ID = "existing-session";
    mockFetchOk();

    const client = new GkillReadClient();
    const sessionId = await client.login();

    expect(sessionId).toBe("existing-session");
    expect(globalThis.fetch).not.toHaveBeenCalled();
  });

  test("throws when credentials are missing", async () => {
    const client = new GkillReadClient();
    await expect(client.login()).rejects.toThrow("Missing login credentials");
  });
});

// ---------------------------------------------------------------------------
// post
// ---------------------------------------------------------------------------
describe("post", () => {
  test("sends correct request and returns JSON body", async () => {
    const responseBody = { data: "result", errors: [] };
    mockFetchOk(responseBody);

    const client = new GkillReadClient();
    const result = await client.post("/api/test", { key: "value" });

    expect(result).toEqual(responseBody);
    expect(globalThis.fetch).toHaveBeenCalledTimes(1);

    const [url, opts] = globalThis.fetch.mock.calls[0];
    expect(url).toBe("http://127.0.0.1:9999/api/test");
    expect(opts.method).toBe("POST");
    expect(opts.headers["Content-Type"]).toBe("application/json");
    expect(JSON.parse(opts.body)).toEqual({ key: "value" });
  });

  test("throws GkillApiError on non-ok response", async () => {
    mockFetchError(500, { errors: [{ error_code: "ERR500" }] });

    const client = new GkillReadClient();
    await expect(client.post("/api/fail", {})).rejects.toThrow("HTTP 500");
  });

  test("throws GkillApiError on network failure", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error("ECONNREFUSED"));

    const client = new GkillReadClient();
    await expect(client.post("/api/down", {})).rejects.toThrow("Network error");
  });
});
