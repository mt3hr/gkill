/**
 * Tests for McpServer from gkill-read-server.mjs.
 *
 * The server is tested with a mock GkillReadClient so no real HTTP calls are made.
 */

import { describe, test, expect, vi, beforeEach } from "vitest";
import { McpServer } from "../gkill-read-server.mjs";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function createMockClient(overrides = {}) {
  return {
    callRead: vi.fn().mockResolvedValue({ errors: [], messages: [] }),
    fetchFile: vi.fn().mockResolvedValue({ buffer: Buffer.from("test"), contentType: "application/octet-stream" }),
    login: vi.fn().mockResolvedValue("mock-session-id"),
    defaultLocale: "ja",
    ...overrides,
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
    mockClient.callRead.mockResolvedValue({
      kyous: [{ id: "1" }],
      total_count: 1,
      returned_count: 1,
      has_more: false,
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_kyous", { query: {} });

    expect(mockClient.callRead).toHaveBeenCalledTimes(1);
    const [pathname] = mockClient.callRead.mock.calls[0];
    expect(pathname).toBe("/api/get_kyous_mcp");
    expect(result.kyous).toHaveLength(1);
    expect(result.returned_count).toBe(1);
  });

  test("dispatches gkill_get_mi_board_list to /api/get_mi_board_list", async () => {
    mockClient.callRead.mockResolvedValue({
      boards: ["board1", "board2"],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_mi_board_list", {});

    const [pathname] = mockClient.callRead.mock.calls[0];
    expect(pathname).toBe("/api/get_mi_board_list");
    expect(result.boards).toEqual(["board1", "board2"]);
  });

  test("dispatches gkill_get_all_tag_names to /api/get_all_tag_names", async () => {
    mockClient.callRead.mockResolvedValue({
      tag_names: ["tag1", "tag2"],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_all_tag_names", {});

    const [pathname] = mockClient.callRead.mock.calls[0];
    expect(pathname).toBe("/api/get_all_tag_names");
    expect(result.tag_names).toEqual(["tag1", "tag2"]);
  });

  test("dispatches gkill_get_all_rep_names to /api/get_all_rep_names", async () => {
    mockClient.callRead.mockResolvedValue({
      rep_names: ["repo1"],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_all_rep_names", {});

    const [pathname] = mockClient.callRead.mock.calls[0];
    expect(pathname).toBe("/api/get_all_rep_names");
    expect(result.rep_names).toEqual(["repo1"]);
  });

  test("dispatches gkill_get_gps_log to /api/get_gps_log", async () => {
    mockClient.callRead.mockResolvedValue({
      gps_logs: [{ lat: 35.0, lng: 139.0 }],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_gps_log", {
      start_date: "2026-01-01",
      end_date: "2026-01-31",
    });

    const [pathname] = mockClient.callRead.mock.calls[0];
    expect(pathname).toBe("/api/get_gps_log");
    expect(result.gps_logs).toHaveLength(1);
  });

  test("dispatches gkill_get_application_config to /api/get_application_config", async () => {
    mockClient.callRead.mockResolvedValue({
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

    const [pathname] = mockClient.callRead.mock.calls[0];
    expect(pathname).toBe("/api/get_application_config");
    expect(result.tag_struct).toBeDefined();
    expect(result.mi_default_board).toBe("default");
  });

  test("throws for unknown tool name", async () => {
    await expect(server.handleToolCall("nonexistent_tool", {})).rejects.toThrow(
      "Unknown tool: nonexistent_tool",
    );
  });

  test("passes currentSessionId as sessionIdOverride to callRead", async () => {
    mockClient.callRead.mockResolvedValue({
      tag_names: ["t1"],
      errors: [],
    });
    server.currentSessionId = "oauth-session-xyz";

    await server.handleToolCall("gkill_get_all_tag_names", {});

    // 4th argument should be the session override
    expect(mockClient.callRead).toHaveBeenCalledWith(
      "/api/get_all_tag_names",
      expect.any(Object),
      true,
      "oauth-session-xyz",
    );
  });

  test("passes null sessionIdOverride when currentSessionId is not set", async () => {
    mockClient.callRead.mockResolvedValue({
      tag_names: ["t1"],
      errors: [],
    });

    await server.handleToolCall("gkill_get_all_tag_names", {});

    expect(mockClient.callRead).toHaveBeenCalledWith(
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
    expect(response.result.tools.length).toBe(7);

    const toolNames = response.result.tools.map((t) => t.name);
    expect(toolNames).toContain("gkill_get_kyous");
    expect(toolNames).toContain("gkill_get_all_tag_names");
    expect(toolNames).toContain("gkill_get_idf_file");
  });

  test("responds to tools/call with tool result", async () => {
    mockClient.callRead.mockResolvedValue({
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
});
