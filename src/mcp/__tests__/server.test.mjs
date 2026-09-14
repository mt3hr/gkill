/**
 * Tests for McpServer from gkill-read-server.mjs.
 *
 * The server is tested with a mock GkillReadClient so no real HTTP calls are made.
 */

import { describe, test, expect, vi, beforeEach } from "vitest";
import { McpServer } from "../gkill-read-server.mjs";
import { MAX_IDF_FILE_BYTES } from "../lib/constants.mjs";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function createMockClient(overrides = {}) {
  return {
    callApi: vi.fn().mockResolvedValue({ errors: [], messages: [] }),
    fetchFile: vi.fn().mockResolvedValue({ buffer: Buffer.from("test"), contentType: "application/octet-stream" }),
    login: vi.fn().mockResolvedValue("mock-session-id"),
    defaultLocale: "ja",
    ...overrides,
  };
}

// get_kyous が返すプラグインKyou 1件分。
function pluginKyouResult() {
  return {
    data_type: "claude_code_message",
    related_time: "2026-08-05T10:00:00+09:00",
    payload: {
      kind: "plugin",
      data_type: "claude_code_message",
      rep_name: "Claude Code",
      kyou_id: "kyou-1",
    },
  };
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------
describe("McpServer constructor", () => {
  test("accepts a client", () => {
    const mock = createMockClient();
    const server = new McpServer(mock);
    expect(server.client).toBe(mock);
  });
});

// ---------------------------------------------------------------------------
// handleToolCall dispatch
// ---------------------------------------------------------------------------
describe("handleToolCall", () => {
  let mockClient;
  let server;

  beforeEach(() => {
    mockClient = createMockClient();
    server = new McpServer(mockClient);
  });

  test("dispatches gkill_get_kyous to /api/get_kyous_mcp", async () => {
    mockClient.callApi.mockResolvedValue({
      kyous: [{ id: "1" }],
      total_count: 1,
      returned_count: 1,
      has_more: false,
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_kyous", { query: {} });

    expect(mockClient.callApi).toHaveBeenCalledTimes(1);
    const [pathname] = mockClient.callApi.mock.calls[0];
    expect(pathname).toBe("/api/get_kyous_mcp");
    expect(result.kyous).toHaveLength(1);
    expect(result.returned_count).toBe(1);
    // M-05: 付随データの取得が全て成功なら partial は付かない
    expect(result.partial).toBeUndefined();
  });

  test("gkill_get_kyous surfaces partial/warnings when attached data fetch failed (M-05)", async () => {
    mockClient.callApi.mockResolvedValue({
      kyous: [{ id: "1" }],
      total_count: 1,
      returned_count: 1,
      has_more: false,
      partial: true,
      warnings: ["failed to fetch tags for 2 record(s); attached data is incomplete"],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_kyous", { query: {} });

    expect(result.partial).toBe(true);
    expect(result.warnings).toEqual([
      "failed to fetch tags for 2 record(s); attached data is incomplete",
    ]);
  });

  test("dispatches gkill_get_mi_board_list to /api/get_mi_board_list", async () => {
    mockClient.callApi.mockResolvedValue({
      boards: ["board1", "board2"],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_mi_board_list", {});

    const [pathname] = mockClient.callApi.mock.calls[0];
    expect(pathname).toBe("/api/get_mi_board_list");
    expect(result.boards).toEqual(["board1", "board2"]);
  });

  test("dispatches gkill_get_all_tag_names to /api/get_all_tag_names", async () => {
    mockClient.callApi.mockResolvedValue({
      tag_names: ["tag1", "tag2"],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_all_tag_names", {});

    const [pathname] = mockClient.callApi.mock.calls[0];
    expect(pathname).toBe("/api/get_all_tag_names");
    expect(result.tag_names).toEqual(["tag1", "tag2"]);
  });

  test("dispatches gkill_get_all_rep_names to /api/get_all_rep_names", async () => {
    mockClient.callApi.mockResolvedValue({
      rep_names: ["repo1"],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_all_rep_names", {});

    const [pathname] = mockClient.callApi.mock.calls[0];
    expect(pathname).toBe("/api/get_all_rep_names");
    expect(result.rep_names).toEqual(["repo1"]);
  });

  test("dispatches gkill_get_gps_log to /api/get_gps_log", async () => {
    mockClient.callApi.mockResolvedValue({
      gps_logs: [{ lat: 35.0, lng: 139.0 }],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_gps_log", {
      start_date: "2026-01-01",
      end_date: "2026-01-31",
    });

    const [pathname] = mockClient.callApi.mock.calls[0];
    expect(pathname).toBe("/api/get_gps_log");
    expect(result.gps_logs).toHaveLength(1);
  });

  test("dispatches gkill_get_application_config to /api/get_application_config", async () => {
    mockClient.callApi.mockResolvedValue({
      application_config: {
        tag_struct: { tags: [] },
        mi_board_struct: {},
        rep_struct: {},
        rep_type_struct: {},
        device_struct: {},
        kftl_template_struct: {},
        mi_default_board: "default",
        show_tags_in_list: true,
      },
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_application_config", {});

    const [pathname] = mockClient.callApi.mock.calls[0];
    expect(pathname).toBe("/api/get_application_config");
    expect(result.tag_struct).toBeDefined();
    expect(result.mi_default_board).toBe("default");
  });

  test("dispatches gkill_get_plugin_list to /api/get_plugin_list", async () => {
    mockClient.callApi.mockResolvedValue({
      plugins: [{ name: "gkill_plugin_claudecode", rep_name: "Claude Code", is_alive: true }],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_plugin_list", {});

    expect(mockClient.callApi).toHaveBeenCalledWith("/api/get_plugin_list", {}, true, null);
    expect(result.plugins).toHaveLength(1);
  });

  test("passes currentSessionId to plugin tools too", async () => {
    mockClient.callApi.mockResolvedValue({ plugins: [], errors: [] });
    server.currentSessionId = "oauth-session-xyz";

    await server.handleToolCall("gkill_get_plugin_list", {});

    expect(mockClient.callApi).toHaveBeenCalledWith(
      "/api/get_plugin_list",
      expect.any(Object),
      true,
      "oauth-session-xyz",
    );
  });

  test("gkill_get_kyous leaves plugin bodies alone by default", async () => {
    mockClient.callApi.mockResolvedValue({
      kyous: [pluginKyouResult()],
      total_count: 1,
      returned_count: 1,
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_kyous", {});

    expect(mockClient.callApi).toHaveBeenCalledTimes(1);
    expect(result.plugin_content).toBeUndefined();
    expect(result.kyous[0].payload.content_status).toBeUndefined();
  });

  test("gkill_get_kyous inlines plugin bodies when include_plugin_content is true", async () => {
    mockClient.callApi.mockImplementation((pathname) =>
      pathname === "/api/get_kyous_mcp"
        ? Promise.resolve({ kyous: [pluginKyouResult()], total_count: 1, returned_count: 1, errors: [] })
        : Promise.resolve({ html: "<html><head><style>p{color:red}</style></head><body><p>会話の本文</p></body></html>", errors: [] }),
    );

    const result = await server.handleToolCall("gkill_get_kyous", { include_plugin_content: true });

    expect(mockClient.callApi).toHaveBeenCalledWith(
      "/api/get_plugin_content_html",
      { rep_name: "Claude Code", kyou_id: "kyou-1" },
      true,
      null,
    );
    expect(result.kyous[0].payload.content_text).toBe("会話の本文");
    expect(result.kyous[0].payload.content_status).toBe("ok");
    expect(result.plugin_content.inlined).toBe(1);
  });

  test("gkill_get_kyous does not forward the inline args to the gkill endpoint", async () => {
    mockClient.callApi.mockImplementation((pathname) =>
      pathname === "/api/get_kyous_mcp"
        ? Promise.resolve({ kyous: [pluginKyouResult()], total_count: 1, returned_count: 1, errors: [] })
        : Promise.resolve({ html: "<p>x</p>", errors: [] }),
    );

    await server.handleToolCall("gkill_get_kyous", {
      include_plugin_content: true,
      plugin_content_max_text_length: 100,
      plugin_content_format: "text",
    });

    const body = mockClient.callApi.mock.calls[0][1];
    expect(body).not.toHaveProperty("include_plugin_content");
    expect(body).not.toHaveProperty("plugin_content_max_text_length");
    expect(body).not.toHaveProperty("plugin_content_format");
  });

  test("gkill_get_kyous still returns results when a plugin body fetch fails", async () => {
    mockClient.callApi.mockImplementation((pathname) =>
      pathname === "/api/get_kyous_mcp"
        ? Promise.resolve({ kyous: [pluginKyouResult()], total_count: 1, returned_count: 1, errors: [] })
        : Promise.reject(new Error("plugin is down")),
    );

    const result = await server.handleToolCall("gkill_get_kyous", { include_plugin_content: true });

    expect(result.kyous).toHaveLength(1);
    expect(result.kyous[0].payload.content_status).toBe("error");
    expect(result.plugin_content.errors).toBe(1);
  });

  test("throws for unknown tool name", async () => {
    await expect(server.handleToolCall("nonexistent_tool", {})).rejects.toThrow(
      "Unknown tool: nonexistent_tool",
    );
  });

  test("a removed tool name is dispatched to the replacement hint, not a dead end", async () => {
    // ツール一覧はクライアントのセッション寿命で固定されるので、削除済みの
    // gkill_get_idf_file_path は旧セッションから呼ばれ続ける。plugin でも read でも
    // ない名前がサーバディスパッチの行き止まりへ落ちたとき、REMOVED_TOOL_HINTS
    // (lib/constants.mjs) の案内文まで届くことを実経路で確かめる。
    // ツール名の echo 自体が gkill_get_idf_file を含むので、案内文にしか無い文言で見る
    await expect(server.handleToolCall("gkill_get_idf_file_path", {})).rejects.toThrow(
      /removed on 2026-08-24/,
    );
    await expect(server.handleToolCall("gkill_get_idf_file_path", {})).rejects.toThrow(
      /otherwise call gkill_get_idf_file/,
    );
    // 案内で完結し、gkill への API 呼び出しは発生しない
    expect(mockClient.callApi).not.toHaveBeenCalled();
    expect(mockClient.fetchFile).not.toHaveBeenCalled();
  });

  test("passes currentSessionId as sessionIdOverride to callApi", async () => {
    mockClient.callApi.mockResolvedValue({
      tag_names: ["t1"],
      errors: [],
    });
    server.currentSessionId = "oauth-session-xyz";

    await server.handleToolCall("gkill_get_all_tag_names", {});

    // 4th argument should be the session override
    expect(mockClient.callApi).toHaveBeenCalledWith(
      "/api/get_all_tag_names",
      expect.any(Object),
      true,
      "oauth-session-xyz",
    );
  });

  test("passes null sessionIdOverride when currentSessionId is not set", async () => {
    mockClient.callApi.mockResolvedValue({
      tag_names: ["t1"],
      errors: [],
    });

    await server.handleToolCall("gkill_get_all_tag_names", {});

    expect(mockClient.callApi).toHaveBeenCalledWith(
      "/api/get_all_tag_names",
      expect.any(Object),
      true,
      null,
    );
  });
});

// ---------------------------------------------------------------------------
// handleMessage — JSON-RPC level
// ---------------------------------------------------------------------------
describe("handleMessage", () => {
  let mockClient;
  let server;

  beforeEach(() => {
    mockClient = createMockClient();
    server = new McpServer(mockClient);
  });

  test("responds to initialize with server info", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0",
      method: "initialize",
      id: 1,
    });

    expect(response.jsonrpc).toBe("2.0");
    expect(response.id).toBe(1);
    expect(response.result.serverInfo.name).toBe("gkill-read-mcp");
    expect(response.result.capabilities.tools).toBeDefined();
  });

  // ツール一覧の世代（2026-09-14 レビュー P0）。initialize の version と
  // gkill_status の description 末尾と gkill_status の応答が、同じ値を指すこと。
  test("initialize version, the stamped gkill_status description and the gkill_status response agree on schema_revision", async () => {
    const initialize = await server.handleMessage({ jsonrpc: "2.0", method: "initialize", id: 1 });
    const match = /\+schema\.([0-9a-f]{12})$/.exec(initialize.result.serverInfo.version);
    expect(match).not.toBeNull();
    const revision = match[1];

    const list = await server.handleMessage({ jsonrpc: "2.0", method: "tools/list", id: 2 });
    const status = list.result.tools.find((tool) => tool.name === "gkill_status");
    expect(status.description.endsWith(` [schema_revision: ${revision}]`)).toBe(true);
    // 他のツールには焼き込まない
    for (const tool of list.result.tools) {
      if (tool.name !== "gkill_status") {
        expect(tool.description).not.toContain("[schema_revision:");
      }
    }

    mockClient.callApi.mockResolvedValue({
      application_config: { user_id: "testuser", device: "testdevice", version: "9.9.9" },
    });
    const call = await server.handleMessage({
      jsonrpc: "2.0",
      method: "tools/call",
      id: 3,
      params: { name: "gkill_status", arguments: {} },
    });
    expect(call.result.isError).toBe(false);
    expect(call.result.structuredContent.schema_revision).toBe(revision);
    expect(call.result.structuredContent.server_kind).toBe("read");
    expect(call.result.structuredContent.tool_count).toBe(list.result.tools.length);
    expect(call.result.structuredContent.transport).toBe("http"); // isLocalTransport は既定 false
    expect(call.result.content[0].text).toContain("Connected to testuser@testdevice via read server");
  });

  test("responds to ping", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0",
      method: "ping",
      id: 42,
    });

    expect(response).toEqual({ jsonrpc: "2.0", id: 42, result: {} });
  });

  test("responds to tools/list with tool definitions", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0",
      method: "tools/list",
      id: 2,
    });

    expect(response.jsonrpc).toBe("2.0");
    expect(response.id).toBe(2);
    expect(Array.isArray(response.result.tools)).toBe(true);
    expect(response.result.tools.length).toBe(11);

    const toolNames = response.result.tools.map((t) => t.name);
    expect(toolNames).toContain("gkill_status");
    expect(toolNames).toContain("gkill_get_kyous");
    expect(toolNames).toContain("gkill_get_all_tag_names");
    expect(toolNames).toContain("gkill_get_idf_file");
    expect(toolNames).toContain("gkill_get_plugin_list");
    expect(toolNames).not.toContain("gkill_get_plugin_content");
  });

  test("responds to tools/call with tool result", async () => {
    mockClient.callApi.mockResolvedValue({
      tag_names: ["alpha", "beta"],
      errors: [],
    });

    const response = await server.handleMessage({
      jsonrpc: "2.0",
      method: "tools/call",
      params: { name: "gkill_get_all_tag_names", arguments: {} },
      id: 3,
    });

    expect(response.jsonrpc).toBe("2.0");
    expect(response.id).toBe(3);
    expect(response.result.isError).toBe(false);
    expect(response.result.content[0].text).toContain("2 tag names");
    expect(response.result.structuredContent.tag_names).toEqual(["alpha", "beta"]);
  });

  test("returns error result for unknown tool via tools/call", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0",
      method: "tools/call",
      params: { name: "bad_tool", arguments: {} },
      id: 4,
    });

    expect(response.jsonrpc).toBe("2.0");
    expect(response.id).toBe(4);
    expect(response.result.isError).toBe(true);
    expect(response.result.content[0].text).toContain("bad_tool failed");
  });

  test("returns method-not-found for unknown methods", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0",
      method: "nonexistent/method",
      id: 5,
    });

    expect(response.error.code).toBe(-32601);
    expect(response.error.message).toContain("nonexistent/method");
  });

  test("returns null for notifications/initialized", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0",
      method: "notifications/initialized",
    });

    expect(response).toBeNull();
  });

  test("returns invalid-request for malformed messages", async () => {
    const response = await server.handleMessage({ not_jsonrpc: true });

    expect(response.error.code).toBe(-32600);
    expect(response.error.message).toBe("Invalid Request");
  });
});

// ---------------------------------------------------------------------------
// gkill_get_idf_file tool
// ---------------------------------------------------------------------------
describe("gkill_get_idf_file", () => {
  let mockClient;
  let server;

  beforeEach(() => {
    mockClient = createMockClient();
    server = new McpServer(mockClient);
  });

  test("dispatches to fetchFile with correct path", async () => {
    const fileContent = Buffer.from([0x89, 0x50, 0x4e, 0x47]);
    mockClient.fetchFile.mockResolvedValue({ buffer: fileContent, contentType: "image/png" });
    server.currentSessionId = "test-session";

    const result = await server.handleToolCall("gkill_get_idf_file", {
      rep_name: "my_repo",
      file_name: "photo.png",
    });

    expect(mockClient.fetchFile).toHaveBeenCalledWith("/files/my_repo/photo.png", "test-session");
    expect(result.file_name).toBe("photo.png");
    expect(result.mime_type).toBe("image/png");
    expect(result.file_size_bytes).toBe(4);
    expect(result.is_image).toBe(true);
    expect(result.file_content_base64).toBe(fileContent.toString("base64"));
  });

  test("returns is_image false for non-image files", async () => {
    mockClient.fetchFile.mockResolvedValue({ buffer: Buffer.from("data"), contentType: "application/pdf" });
    server.currentSessionId = "sess";

    const result = await server.handleToolCall("gkill_get_idf_file", {
      rep_name: "repo",
      file_name: "doc.pdf",
    });

    expect(result.is_image).toBe(false);
    expect(result.mime_type).toBe("application/pdf");
  });

  test("handles nested file paths", async () => {
    mockClient.fetchFile.mockResolvedValue({ buffer: Buffer.from("x"), contentType: "text/plain" });
    server.currentSessionId = "sess";

    await server.handleToolCall("gkill_get_idf_file", {
      rep_name: "repo",
      file_name: "sub/dir/file.txt",
    });

    expect(mockClient.fetchFile).toHaveBeenCalledWith("/files/repo/sub/dir/file.txt", "sess");
  });

  test("uses login when currentSessionId is not set", async () => {
    mockClient.fetchFile.mockResolvedValue({ buffer: Buffer.from("x"), contentType: "text/plain" });
    server.currentSessionId = null;

    await server.handleToolCall("gkill_get_idf_file", {
      rep_name: "repo",
      file_name: "file.txt",
    });

    expect(mockClient.login).toHaveBeenCalled();
    expect(mockClient.fetchFile).toHaveBeenCalledWith("/files/repo/file.txt", "mock-session-id");
  });

  test("buildToolResult includes image content block for images", () => {
    const payload = {
      file_name: "img.jpg",
      mime_type: "image/jpeg",
      file_size_bytes: 100,
      is_image: true,
      file_content_base64: "base64data",
    };

    const result = server.buildToolResult("gkill_get_idf_file", payload, false);

    expect(result.isError).toBe(false);
    expect(result.content).toHaveLength(2);
    expect(result.content[0].type).toBe("text");
    expect(result.content[1].type).toBe("image");
    expect(result.content[1].data).toBe("base64data");
    expect(result.content[1].mimeType).toBe("image/jpeg");
  });

  test("buildToolResult does not include image block for non-images", () => {
    const payload = {
      file_name: "doc.pdf",
      mime_type: "application/pdf",
      file_size_bytes: 200,
      is_image: false,
      file_content_base64: "pdfdata",
    };

    const result = server.buildToolResult("gkill_get_idf_file", payload, false);

    expect(result.content).toHaveLength(1);
    expect(result.content[0].type).toBe("text");
  });

  // 画像のバイト列はimageブロックで届く。structuredContentにも同じbase64が入ると
  // 1レスポンスに同じデータが2回乗り、クライアントのツール結果上限を超えて切り捨てられ、
  // 画像そのものが届かなくなる。
  test("buildToolResult omits base64 from text and structuredContent for images", () => {
    const payload = {
      file_name: "img.jpg",
      mime_type: "image/jpeg",
      file_size_bytes: 100,
      is_image: true,
      file_content_base64: "base64data",
    };

    const result = server.buildToolResult("gkill_get_idf_file", payload, false);

    expect(result.structuredContent.file_content_base64).toBeUndefined();
    expect(result.structuredContent.file_name).toBe("img.jpg");
    expect(result.content[0].text).not.toContain("base64data");
    // imageブロックからは従来どおりバイト列が届く
    expect(result.content[1].data).toBe("base64data");
  });

  // 非画像にはimageブロックが付かないため、structuredContentが唯一のバイト列の渡し口。
  test("buildToolResult keeps base64 in structuredContent for non-images", () => {
    const payload = {
      file_name: "doc.pdf",
      mime_type: "application/pdf",
      file_size_bytes: 200,
      is_image: false,
      file_content_base64: "pdfdata",
    };

    const result = server.buildToolResult("gkill_get_idf_file", payload, false);

    expect(result.structuredContent.file_content_base64).toBe("pdfdata");
    expect(result.content[0].text).not.toContain("pdfdata");
  });

  test("buildToolResult strips charset parameter from the image block mimeType", () => {
    const payload = {
      file_name: "img.jpg",
      mime_type: "image/jpeg; charset=binary",
      file_size_bytes: 100,
      is_image: true,
      file_content_base64: "base64data",
    };

    const result = server.buildToolResult("gkill_get_idf_file", payload, false);

    expect(result.content[1].mimeType).toBe("image/jpeg");
  });

  // structuredContent だけを見ると base64 が無く「画像が返っていない」と誤読された
  // (2026-08-30 レビュー 5.5)。imageブロックへ移した印を構造化側にも残す。
  test("buildToolResult marks image payloads with image_content_attached", () => {
    const payload = {
      file_name: "img.jpg",
      mime_type: "image/jpeg",
      file_size_bytes: 100,
      is_image: true,
      file_content_base64: "base64data",
    };

    const result = server.buildToolResult("gkill_get_idf_file", payload, false);

    expect(result.structuredContent.image_content_attached).toBe(true);
  });

  // 非画像は structuredContent が唯一のバイト列の渡し口 (base64 が残る) なので印は不要。
  test("buildToolResult does not mark non-image payloads", () => {
    const payload = {
      file_name: "doc.pdf",
      mime_type: "application/pdf",
      file_size_bytes: 200,
      is_image: false,
      file_content_base64: "pdfdata",
    };

    const result = server.buildToolResult("gkill_get_idf_file", payload, false);

    expect(result.structuredContent.image_content_attached).toBeUndefined();
  });

  test("rejects files larger than the size limit", async () => {
    const huge = Buffer.alloc(MAX_IDF_FILE_BYTES + 1);
    mockClient.fetchFile.mockResolvedValue({ buffer: huge, contentType: "video/mp4" });
    server.currentSessionId = "sess";

    await expect(
      server.handleToolCall("gkill_get_idf_file", { rep_name: "repo", file_name: "big.mp4" }),
    ).rejects.toThrow(/too large/);
  });

  test("points at thumb as the way out when the file is too large", async () => {
    const huge = Buffer.alloc(MAX_IDF_FILE_BYTES + 1);
    mockClient.fetchFile.mockResolvedValue({ buffer: huge, contentType: "image/png" });
    server.currentSessionId = "sess";

    await expect(
      server.handleToolCall("gkill_get_idf_file", { rep_name: "repo", file_name: "huge.png" }),
    ).rejects.toThrow(/retry with thumb/);
  });

  test("appends thumb to the file path in the same form as file_url", async () => {
    mockClient.fetchFile.mockResolvedValue({ buffer: Buffer.from("j"), contentType: "image/jpeg" });
    server.currentSessionId = "sess";

    const result = await server.handleToolCall("gkill_get_idf_file", {
      rep_name: "repo",
      file_name: "photo.png",
      thumb: "1024x1024",
    });

    expect(mockClient.fetchFile).toHaveBeenCalledWith("/files/repo/photo.png?thumb=1024x1024", "sess");
    // 縮小して取ったことが応答から分かる (原寸と取り違えない)
    expect(result.thumb).toBe("1024x1024");
    expect(result.is_image).toBe(true);
  });

  test("puts is_video before thumb, matching the client's build_media_url", async () => {
    mockClient.fetchFile.mockResolvedValue({ buffer: Buffer.from("j"), contentType: "image/jpeg" });
    server.currentSessionId = "sess";

    await server.handleToolCall("gkill_get_idf_file", {
      rep_name: "repo",
      file_name: "clip.mp4",
      thumb: "400x400",
      is_video: true,
    });

    expect(mockClient.fetchFile).toHaveBeenCalledWith(
      "/files/repo/clip.mp4?is_video=true&thumb=400x400",
      "sess",
    );
  });

  test("omits the query entirely when no thumb is asked for", async () => {
    mockClient.fetchFile.mockResolvedValue({ buffer: Buffer.from("x"), contentType: "image/png" });
    server.currentSessionId = "sess";

    const result = await server.handleToolCall("gkill_get_idf_file", {
      rep_name: "repo",
      file_name: "photo.png",
    });

    expect(mockClient.fetchFile).toHaveBeenCalledWith("/files/repo/photo.png", "sess");
    expect(result.thumb).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// file_path exposure (local stdio clients only)
// ---------------------------------------------------------------------------
describe("file_path exposure", () => {
  let mockClient;
  let server;

  beforeEach(() => {
    mockClient = createMockClient();
    server = new McpServer(mockClient);
  });

  test("defaults to non-local so a transport that forgets to opt in never leaks paths", () => {
    expect(server.isLocalTransport).toBe(false);
  });

  test("buildToolResult keeps file_path in kyou payloads for local clients", () => {
    server.isLocalTransport = true;
    const payload = {
      kyous: [{ data_type: "idf", payload: { kind: "idf", file_path: "$HOME/gkill/photo.png" } }],
    };

    const result = server.buildToolResult("gkill_get_kyous", payload, false);

    expect(result.structuredContent.kyous[0].payload.file_path).toBe("$HOME/gkill/photo.png");
  });

  test("buildToolResult strips file_path from kyou payloads for remote clients", () => {
    server.isLocalTransport = false;
    const payload = {
      kyous: [{ data_type: "idf", payload: { kind: "idf", file_path: "$HOME/gkill/photo.png" } }],
    };

    const result = server.buildToolResult("gkill_get_kyous", payload, false);

    expect(result.structuredContent.kyous[0].payload.file_path).toBeUndefined();
    expect(result.content[0].text).not.toContain("$HOME/gkill/photo.png");
  });
});

// ---------------------------------------------------------------------------
// warnings / partial の1行要約への昇格 (2026-08-30 レビュー P1)
//
// 以前の要約は件数と cursor だけで、未知タグで0件でも `No entries matched.` としか
// 出なかった。warnings は構造化結果に入っていても、要約だけを見る利用者・モデルは
// 「本当に0件」と誤読する。昇格は mcp-server-base の summarizeToolPayload 1箇所で
// 全ツールに掛かるので、ここでは buildToolResult 経由 (実配線) で検証する。
// ---------------------------------------------------------------------------
describe("warnings / partial elevation into the one-line summary", () => {
  let server;

  beforeEach(() => {
    server = new McpServer(createMockClient());
  });

  // content[0].text は `${summary}\n\n${JSON}` で、JSON 側には warnings が常に入る。
  // 要約行だけを検査するため1行目を切り出す。
  function summaryLine(result) {
    return result.content[0].text.split("\n")[0];
  }

  test("a zero-hit result with a warning does not read as a clean zero", () => {
    const payload = {
      kyous: [],
      total_count: 0,
      returned_count: 0,
      remaining_count: 0,
      has_more: false,
      warnings: ['unknown tag "no-such-tag" in query.tags'],
    };
    const line = summaryLine(server.buildToolResult("gkill_get_kyous", payload, false));
    expect(line).toContain("No entries matched.");
    expect(line).toContain('WARNING: unknown tag "no-such-tag"');
  });

  test("partial shows up even without warnings", () => {
    const payload = {
      kyous: [{ id: "k1" }],
      total_count: 1,
      returned_count: 1,
      remaining_count: 0,
      has_more: false,
      partial: true,
    };
    const line = summaryLine(server.buildToolResult("gkill_get_kyous", payload, false));
    expect(line).toContain("PARTIAL:");
  });

  test("a warning shows up even when partial is false (broken-rep count case)", () => {
    // 壊れた rep の warning があっても、ページング上の打ち切りが無ければ partial は false の
    // ままになりうる (ADR-0216)。partial だけを見て warnings の表示を省略してはいけない。
    const payload = {
      kyous: [],
      total_count: 12345,
      returned_count: 0,
      remaining_count: 0,
      has_more: false,
      warnings: ['repository "BrokenRep" could not be read; results may be incomplete'],
    };
    const line = summaryLine(server.buildToolResult("gkill_get_kyous", payload, false));
    expect(line).toContain("Counted 12345 entries.");
    expect(line).toContain("WARNING:");
    expect(line).not.toContain("PARTIAL:");
  });

  test("multiple warnings show the first plus a count", () => {
    const payload = {
      kyous: [],
      total_count: 0,
      returned_count: 0,
      remaining_count: 0,
      has_more: false,
      warnings: ["first warning", "second warning", "third warning"],
    };
    const line = summaryLine(server.buildToolResult("gkill_get_kyous", payload, false));
    expect(line).toContain("WARNING: first warning");
    expect(line).toContain("(+2 more)");
    expect(line).not.toContain("second warning");
  });

  test("an over-long warning is truncated in the summary (full text stays in warnings[])", () => {
    const longWarning = "w".repeat(250);
    const payload = {
      kyous: [],
      total_count: 0,
      returned_count: 0,
      remaining_count: 0,
      has_more: false,
      warnings: [longWarning],
    };
    const result = server.buildToolResult("gkill_get_kyous", payload, false);
    const line = summaryLine(result);
    expect(line).toContain("WARNING:");
    expect(line).toContain("…");
    expect(line.length).toBeLessThan(300);
    expect(result.structuredContent.warnings[0]).toBe(longWarning);
  });

  test("a stale-schema warning keeps its dedicated note and is not double-reported", () => {
    // 古スキーマ警告は appendStaleSchemaNoteToSummary が専用文言で扱う。
    // 汎用の WARNING: にも載せると同じ指摘が1行に2回並ぶ。
    const payload = {
      kyous: [],
      total_count: 0,
      returned_count: 0,
      remaining_count: 0,
      has_more: false,
      warnings: ["this client's tool schema snapshot looks stale: data_types arrived as a JSON string"],
    };
    const line = summaryLine(server.buildToolResult("gkill_get_kyous", payload, false));
    expect(line).toContain("tool schema looks stale");
    expect(line).not.toContain("WARNING:");
  });

  test("a clean result gets no WARNING / PARTIAL suffix", () => {
    const payload = {
      kyous: [{ id: "k1" }],
      total_count: 1,
      returned_count: 1,
      remaining_count: 0,
      has_more: false,
    };
    const line = summaryLine(server.buildToolResult("gkill_get_kyous", payload, false));
    expect(line).not.toContain("WARNING:");
    expect(line).not.toContain("PARTIAL:");
  });

  test("partial and a warning appear together, PARTIAL first", () => {
    // 実運用で最も起きやすい複合 (壊れた rep で欠けつつ、別 rep の付随データも落ちた等)。
    // 並び順まで固定するのは、順序が入れ替わると「PARTIAL の説明が WARNING の続き」に
    // 読めてしまうため。
    const payload = {
      kyous: [{ id: "k1" }],
      total_count: 1,
      returned_count: 1,
      remaining_count: 0,
      has_more: false,
      partial: true,
      warnings: ['repository "BrokenRep" could not be read; results may be incomplete'],
    };
    const line = summaryLine(server.buildToolResult("gkill_get_kyous", payload, false));
    const partialIndex = line.indexOf("PARTIAL:");
    const warningIndex = line.indexOf("WARNING:");
    expect(partialIndex).toBeGreaterThan(-1);
    expect(warningIndex).toBeGreaterThan(partialIndex);
  });

  test("a stale warning mixed with a real one is not counted into (+N more)", () => {
    // stale はフィルタ後に数えるので、実 warning 1件 + stale 1件で "(+1 more)" が
    // 付いてはいけない (残った実 warning は1件だけ)。数え方を取り違えると件数が嘘になる。
    const payload = {
      kyous: [],
      total_count: 0,
      returned_count: 0,
      remaining_count: 0,
      has_more: false,
      warnings: [
        "this client's tool schema snapshot looks stale: count_only arrived as a JSON string",
        'unknown tag "no-such-tag" in query.tags',
      ],
    };
    const line = summaryLine(server.buildToolResult("gkill_get_kyous", payload, false));
    expect(line).toContain('WARNING: unknown tag "no-such-tag"');
    expect(line).not.toContain("more)");
    expect(line).toContain("tool schema looks stale");
  });

  test("the summary warning is cut at exactly 200 characters, not before", () => {
    // 境界: 200文字ちょうどは切らない / 201文字で切って "…" を付ける。
    const exactly200 = "w".repeat(200);
    const over200 = "w".repeat(201);
    const base = { kyous: [], total_count: 0, returned_count: 0, remaining_count: 0, has_more: false };

    const lineAtLimit = summaryLine(
      server.buildToolResult("gkill_get_kyous", { ...base, warnings: [exactly200] }, false),
    );
    expect(lineAtLimit).toContain(exactly200);
    expect(lineAtLimit).not.toContain("…");

    const lineOverLimit = summaryLine(
      server.buildToolResult("gkill_get_kyous", { ...base, warnings: [over200] }, false),
    );
    expect(lineOverLimit).toContain(`${exactly200}…`);
    expect(lineOverLimit).not.toContain(over200);
  });
});
