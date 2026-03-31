/**
 * Tests for GkillClient from gkill-readwrite-server.mjs.
 *
 * All network calls are mocked via globalThis.fetch so no real server is needed.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from "vitest";
import { GkillClient } from "../gkill-readwrite-server.mjs";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function mockFetchOk(body = {}) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(body),
  });
}

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
describe("GkillClient constructor", () => {
  test("sets defaults when env vars are absent", () => {
    const client = new GkillClient();
    expect(client.baseUrl).toBe("http://127.0.0.1:9999");
    expect(client.userId).toBe("");
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

    const client = new GkillClient();
    expect(client.baseUrl).toBe("https://example.com:1234");
    expect(client.userId).toBe("testuser");
    expect(client.passwordSha256).toBe("abc123");
    expect(client.defaultLocale).toBe("en");
    expect(client.sessionId).toBe("sess-42");
  });

  test("sets dispatcher when GKILL_INSECURE=true", () => {
    process.env.GKILL_INSECURE = "true";
    const client = new GkillClient();
    expect(client.dispatcher).not.toBeNull();
  });
});

// ---------------------------------------------------------------------------
// resolvePasswordSha256
// ---------------------------------------------------------------------------
describe("resolvePasswordSha256", () => {
  test("returns passwordSha256 when set", () => {
    process.env.GKILL_PASSWORD_SHA256 = "abc123";
    const client = new GkillClient();
    expect(client.resolvePasswordSha256()).toBe("abc123");
  });

  test("hashes password when only password is set", () => {
    process.env.GKILL_PASSWORD = "secret";
    const client = new GkillClient();
    const hash = client.resolvePasswordSha256();
    expect(hash).toHaveLength(64);
  });

  test("returns empty string when no credentials", () => {
    const client = new GkillClient();
    expect(client.resolvePasswordSha256()).toBe("");
  });
});

// ---------------------------------------------------------------------------
// login
// ---------------------------------------------------------------------------
describe("login", () => {
  test("returns existing session_id without calling API", async () => {
    process.env.GKILL_SESSION_ID = "existing-session";
    const client = new GkillClient();
    const result = await client.login();
    expect(result).toBe("existing-session");
  });

  test("calls /api/login and returns session_id", async () => {
    process.env.GKILL_USER = "admin";
    process.env.GKILL_PASSWORD_SHA256 = "hash";
    mockFetchOk({ session_id: "new-session", errors: [] });

    const client = new GkillClient();
    const result = await client.login();
    expect(result).toBe("new-session");
  });

  test("throws when credentials are missing", async () => {
    const client = new GkillClient();
    await expect(client.login()).rejects.toThrow("Missing login credentials");
  });
});

// ---------------------------------------------------------------------------
// callApi
// ---------------------------------------------------------------------------
describe("callApi", () => {
  test("posts to the correct endpoint with auth", async () => {
    process.env.GKILL_USER = "admin";
    process.env.GKILL_PASSWORD_SHA256 = "hash";

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

    const client = new GkillClient();
    const result = await client.callApi("/api/add_kmemo", { kmemo: {} }, true);
    expect(result.added_kmemo.id).toBe("123");
  });

  test("retries on auth error", async () => {
    process.env.GKILL_USER = "admin";
    process.env.GKILL_PASSWORD_SHA256 = "hash";

    let callCount = 0;
    globalThis.fetch = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount === 1) return Promise.resolve({ ok: true, json: () => Promise.resolve({ session_id: "old", errors: [] }) });
      if (callCount === 2) return Promise.resolve({ ok: true, json: () => Promise.resolve({ errors: [{ error_code: "ERR000013" }] }) });
      if (callCount === 3) return Promise.resolve({ ok: true, json: () => Promise.resolve({ session_id: "new", errors: [] }) });
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ kyous: [], errors: [] }) });
    });

    const client = new GkillClient();
    const _result = await client.callApi("/api/get_kyous_mcp", {}, true);
    expect(callCount).toBe(4);
  });
});

// ---------------------------------------------------------------------------
// fetchFile
// ---------------------------------------------------------------------------
describe("fetchFile", () => {
  test("fetches file with session cookie", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => "image/png" },
      arrayBuffer: () => Promise.resolve(new ArrayBuffer(4)),
    });

    const client = new GkillClient();
    const result = await client.fetchFile("/files/rep/test.png", "sess-123");
    expect(result.contentType).toBe("image/png");
    expect(result.buffer).toBeInstanceOf(Buffer);

    const headers = globalThis.fetch.mock.calls[0][1].headers;
    expect(headers.Cookie).toContain("sess-123");
  });
});
