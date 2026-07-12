/**
 * Tests for the remote file-URL delivery path (Part 3):
 *  - FileLinkStore (mint/resolve/expiry)
 *  - buildToolResult injecting file_url for remote clients
 *  - HttpTransport.handleFileServe delivering bytes by token
 */

import { describe, test, expect, vi, beforeEach, afterEach } from "vitest";
import { McpServer, HttpTransport } from "../gkill-read-server.mjs";
import { FileLinkStore } from "../lib/file-link-store.mjs";

// ---------------------------------------------------------------------------
// FileLinkStore
// ---------------------------------------------------------------------------
describe("FileLinkStore", () => {
  test("mint then resolve returns the original file info", () => {
    const store = new FileLinkStore();
    const token = store.mint({
      gkillSessionId: "sess-1",
      repName: "Files",
      fileName: "a/b.png",
      isImage: true,
    });
    expect(typeof token).toBe("string");
    expect(token.length).toBeGreaterThan(16);

    const link = store.resolve(token);
    expect(link).toEqual({
      gkillSessionId: "sess-1",
      repName: "Files",
      fileName: "a/b.png",
      isImage: true,
    });
  });

  test("unknown token resolves to null", () => {
    const store = new FileLinkStore();
    expect(store.resolve("does-not-exist")).toBeNull();
  });

  test("expired token resolves to null and is dropped", () => {
    const store = new FileLinkStore();
    const token = store.mint({ gkillSessionId: "s", repName: "r", fileName: "f", isImage: false }, -1);
    expect(store.resolve(token)).toBeNull();
    // resolve dropped the expired entry
    expect(store.links.has(token)).toBe(false);
  });

  test("distinct tokens do not collide", () => {
    const store = new FileLinkStore();
    const t1 = store.mint({ gkillSessionId: "s1", repName: "r1", fileName: "f1", isImage: false });
    const t2 = store.mint({ gkillSessionId: "s2", repName: "r2", fileName: "f2", isImage: true });
    expect(t1).not.toBe(t2);
    expect(store.resolve(t1).repName).toBe("r1");
    expect(store.resolve(t2).repName).toBe("r2");
  });
});

// ---------------------------------------------------------------------------
// buildToolResult — file_url injection for remote clients
// ---------------------------------------------------------------------------
describe("buildToolResult file_url injection", () => {
  let server;
  let store;

  beforeEach(() => {
    server = new McpServer({ defaultLocale: "ja" });
    store = new FileLinkStore();
    server.isLocalTransport = false;
    server.fileLinkContext = { publicBaseUrl: "https://mcp.example.test", store };
    server.currentSessionId = "sess-xyz";
  });

  function idfResult(extra = {}) {
    return {
      kyous: [
        {
          data_type: "idf",
          payload: {
            kind: "idf",
            rep_name: "Files",
            file_name: "photo.png",
            is_image: true,
            file_path: "C:\\Users\\me\\gkill\\photo.png",
            ...extra,
          },
        },
      ],
    };
  }

  test("image payload gets a thumbnail file_url plus a full-size file_url_full, and file_path is removed", () => {
    const result = server.buildToolResult("gkill_get_kyous", idfResult(), false);
    const p = result.structuredContent.kyous[0].payload;

    expect(p.file_path).toBeUndefined();
    expect(p.file_url).toMatch(/^https:\/\/mcp\.example\.test\/files\/[0-9a-f]+\?thumb=1024x1024$/);
    expect(p.file_url_full).toMatch(/^https:\/\/mcp\.example\.test\/files\/[0-9a-f]+$/);

    // the minted token resolves back to the file, carrying the OAuth session
    const token = new URL(p.file_url_full).pathname.split("/").pop();
    expect(store.resolve(token)).toMatchObject({
      gkillSessionId: "sess-xyz",
      repName: "Files",
      fileName: "photo.png",
      isImage: true,
    });
  });

  test("non-image payload gets a single original file_url and no thumbnail", () => {
    const result = server.buildToolResult(
      "gkill_get_kyous",
      idfResult({ is_image: false, file_name: "doc.pdf" }),
      false,
    );
    const p = result.structuredContent.kyous[0].payload;

    expect(p.file_path).toBeUndefined();
    expect(p.file_url).toMatch(/^https:\/\/mcp\.example\.test\/files\/[0-9a-f]+$/);
    expect(p.file_url_full).toBeUndefined();
  });

  test("local (stdio) client keeps file_path and gets no file_url", () => {
    server.isLocalTransport = true;
    const result = server.buildToolResult("gkill_get_kyous", idfResult(), false);
    const p = result.structuredContent.kyous[0].payload;

    expect(p.file_path).toBe("C:\\Users\\me\\gkill\\photo.png");
    expect(p.file_url).toBeUndefined();
  });

  test("remote client without a fileLinkContext falls back to stripping file_path", () => {
    server.fileLinkContext = null;
    const result = server.buildToolResult("gkill_get_kyous", idfResult(), false);
    const p = result.structuredContent.kyous[0].payload;

    expect(p.file_path).toBeUndefined();
    expect(p.file_url).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// HttpTransport.handleFileServe — public byte delivery
// ---------------------------------------------------------------------------
describe("HttpTransport.handleFileServe", () => {
  let transport;
  let fetchFile;

  function mockRes() {
    return {
      statusCode: null,
      headers: null,
      body: null,
      writeHead(code, headers) {
        this.statusCode = code;
        this.headers = headers;
      },
      end(body) {
        this.body = body;
      },
    };
  }

  const req = { socket: { remoteAddress: "127.0.0.1" }, url: "/files/x", method: "GET", headers: {} };

  beforeEach(() => {
    fetchFile = vi.fn();
    const fakeServer = {
      client: { fetchFile },
      accessLog: { info() {}, warn() {}, error() {} },
    };
    transport = new HttpTransport(fakeServer, 0, { issuer: "https://mcp.example.test" });
  });

  afterEach(() => {
    transport.fileLinkStore.stopCleanup();
  });

  test("valid token delivers the bytes gkill returns", async () => {
    fetchFile.mockResolvedValue({ buffer: Buffer.from([1, 2, 3, 4]), contentType: "image/png" });
    const token = transport.fileLinkStore.mint({
      gkillSessionId: "sess-1",
      repName: "Files",
      fileName: "photo.png",
      isImage: true,
    });
    const res = mockRes();

    await transport.handleFileServe(req, res, token, {});

    expect(res.statusCode).toBe(200);
    expect(res.headers["Content-Type"]).toBe("image/png");
    expect(res.headers["Content-Length"]).toBe(4);
    expect(res.body).toEqual(Buffer.from([1, 2, 3, 4]));
    // fetched from gkill with the session bound to the token
    expect(fetchFile).toHaveBeenCalledWith("/files/Files/photo.png", "sess-1");
  });

  test("thumb query is forwarded to gkill only for images and only in WxH form", async () => {
    fetchFile.mockResolvedValue({ buffer: Buffer.from([0]), contentType: "image/jpeg" });
    const token = transport.fileLinkStore.mint({
      gkillSessionId: "s", repName: "Files", fileName: "photo.png", isImage: true,
    });

    await transport.handleFileServe(req, mockRes(), token, { thumb: "1024x1024" });
    expect(fetchFile).toHaveBeenLastCalledWith("/files/Files/photo.png?thumb=1024x1024", "s");

    // malformed thumb is ignored (not forwarded)
    await transport.handleFileServe(req, mockRes(), token, { thumb: "1024; rm -rf" });
    expect(fetchFile).toHaveBeenLastCalledWith("/files/Files/photo.png", "s");
  });

  test("thumb query is ignored for non-image files", async () => {
    fetchFile.mockResolvedValue({ buffer: Buffer.from([0]), contentType: "application/pdf" });
    const token = transport.fileLinkStore.mint({
      gkillSessionId: "s", repName: "Files", fileName: "doc.pdf", isImage: false,
    });

    await transport.handleFileServe(req, mockRes(), token, { thumb: "1024x1024" });
    expect(fetchFile).toHaveBeenLastCalledWith("/files/Files/doc.pdf", "s");
  });

  test("unknown or expired token returns 404 without touching gkill", async () => {
    const res = mockRes();
    await transport.handleFileServe(req, res, "bogus-token", {});
    expect(res.statusCode).toBe(404);
    expect(fetchFile).not.toHaveBeenCalled();
  });

  test("gkill fetch failure surfaces as 502", async () => {
    fetchFile.mockRejectedValue(new Error("backend down"));
    const token = transport.fileLinkStore.mint({
      gkillSessionId: "s", repName: "Files", fileName: "photo.png", isImage: true,
    });
    const res = mockRes();

    await transport.handleFileServe(req, res, token, {});
    expect(res.statusCode).toBe(502);
  });
});
