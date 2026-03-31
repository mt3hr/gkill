/**
 * Tests for McpServer from gkill-readwrite-server.mjs.
 *
 * The server is tested with a mock GkillClient so no real HTTP calls are made.
 */

import { describe, test, expect, vi, beforeEach } from "vitest";
import { McpServer } from "../gkill-readwrite-server.mjs";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function createMockClient(overrides = {}) {
  return {
    callApi: vi.fn().mockResolvedValue({ errors: [], messages: [] }),
    fetchFile: vi.fn().mockResolvedValue({ buffer: Buffer.from("test"), contentType: "application/octet-stream" }),
    login: vi.fn().mockResolvedValue("mock-session-id"),
    defaultLocale: "ja",
    userId: "testuser",
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
// handleToolCall dispatch — Read tools
// ---------------------------------------------------------------------------
describe("handleToolCall — read tools", () => {
  let mockClient;
  let server;

  beforeEach(() => {
    mockClient = createMockClient();
    server = new McpServer(mockClient);
  });

  test("dispatches gkill_get_kyous to /api/get_kyous_mcp", async () => {
    mockClient.callApi.mockResolvedValue({
      kyous: [{ id: "1" }], total_count: 1, returned_count: 1, has_more: false, errors: [],
    });

    const result = await server.handleToolCall("gkill_get_kyous", { query: {} });
    const [pathname] = mockClient.callApi.mock.calls[0];
    expect(pathname).toBe("/api/get_kyous_mcp");
    expect(result.kyous).toHaveLength(1);
  });

  test("dispatches gkill_get_mi_board_list", async () => {
    mockClient.callApi.mockResolvedValue({ boards: ["b1", "b2"], errors: [] });
    const result = await server.handleToolCall("gkill_get_mi_board_list", {});
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/get_mi_board_list");
    expect(result.boards).toEqual(["b1", "b2"]);
  });

  test("dispatches gkill_get_all_tag_names", async () => {
    mockClient.callApi.mockResolvedValue({ tag_names: ["t1"], errors: [] });
    const result = await server.handleToolCall("gkill_get_all_tag_names", {});
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/get_all_tag_names");
    expect(result.tag_names).toEqual(["t1"]);
  });

  test("dispatches gkill_get_all_rep_names", async () => {
    mockClient.callApi.mockResolvedValue({ rep_names: ["r1"], errors: [] });
    const result = await server.handleToolCall("gkill_get_all_rep_names", {});
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/get_all_rep_names");
    expect(result.rep_names).toEqual(["r1"]);
  });

  test("dispatches gkill_get_gps_log", async () => {
    mockClient.callApi.mockResolvedValue({ gps_logs: [{ lat: 35.6 }], errors: [] });
    const result = await server.handleToolCall("gkill_get_gps_log", {
      start_date: "2026-03-01", end_date: "2026-03-07",
    });
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/get_gps_log");
    expect(result.gps_logs).toHaveLength(1);
  });

  test("dispatches gkill_get_application_config", async () => {
    mockClient.callApi.mockResolvedValue({ application_config: { tag_struct: {} }, errors: [] });
    const result = await server.handleToolCall("gkill_get_application_config", {});
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/get_application_config");
    expect(result).toHaveProperty("tag_struct");
  });

  test("dispatches gkill_get_idf_file", async () => {
    mockClient.fetchFile.mockResolvedValue({
      buffer: Buffer.from("png-data"),
      contentType: "image/png",
    });
    const result = await server.handleToolCall("gkill_get_idf_file", {
      rep_name: "repo", file_name: "photo.png",
    });
    expect(result.file_name).toBe("photo.png");
    expect(result.mime_type).toBe("image/png");
    expect(result.is_image).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// handleToolCall dispatch — Write tools
// ---------------------------------------------------------------------------
describe("handleToolCall — write tools", () => {
  let mockClient;
  let server;

  beforeEach(() => {
    mockClient = createMockClient();
    server = new McpServer(mockClient);
  });

  test("dispatches gkill_add_kmemo", async () => {
    mockClient.callApi.mockResolvedValue({
      added_kmemo: { id: "k1", content: "hello" }, added_kyou: { id: "k1" }, errors: [],
    });
    const result = await server.handleToolCall("gkill_add_kmemo", { content: "hello" });
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/add_kmemo");
    expect(result.added_kmemo.id).toBe("k1");
  });

  test("dispatches gkill_add_urlog", async () => {
    mockClient.callApi.mockResolvedValue({
      added_urlog: { id: "u1" }, added_kyou: { id: "u1" }, errors: [],
    });
    const result = await server.handleToolCall("gkill_add_urlog", { url: "https://example.com" });
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/add_urlog");
    expect(result.added_urlog.id).toBe("u1");
  });

  test("dispatches gkill_add_nlog", async () => {
    mockClient.callApi.mockResolvedValue({
      added_nlog: { id: "n1", amount: 1500 }, added_kyou: { id: "n1" }, errors: [],
    });
    const result = await server.handleToolCall("gkill_add_nlog", { title: "lunch", amount: 1500 });
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/add_nlog");
    expect(result.added_nlog.amount).toBe(1500);
  });

  test("dispatches gkill_add_lantana", async () => {
    mockClient.callApi.mockResolvedValue({
      added_lantana: { id: "l1", mood: 7 }, added_kyou: { id: "l1" }, errors: [],
    });
    const result = await server.handleToolCall("gkill_add_lantana", { mood: 7 });
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/add_lantana");
    expect(result.added_lantana.mood).toBe(7);
  });

  test("dispatches gkill_add_timeis", async () => {
    mockClient.callApi.mockResolvedValue({
      added_timeis: { id: "t1", title: "coding" }, added_kyou: { id: "t1" }, errors: [],
    });
    const result = await server.handleToolCall("gkill_add_timeis", { title: "coding" });
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/add_timeis");
    expect(result.added_timeis.title).toBe("coding");
  });

  test("dispatches gkill_add_mi", async () => {
    mockClient.callApi.mockResolvedValue({
      added_mi: { id: "m1", title: "fix bug", board_name: "dev" }, added_kyou: { id: "m1" }, errors: [],
    });
    const result = await server.handleToolCall("gkill_add_mi", { title: "fix bug", board_name: "dev" });
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/add_mi");
    expect(result.added_mi.title).toBe("fix bug");
  });

  test("dispatches gkill_add_kc", async () => {
    mockClient.callApi.mockResolvedValue({
      added_kc: { id: "c1", num_value: 42 }, added_kyou: { id: "c1" }, errors: [],
    });
    const result = await server.handleToolCall("gkill_add_kc", { title: "steps", num_value: 42 });
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/add_kc");
    expect(result.added_kc.num_value).toBe(42);
  });

  test("dispatches gkill_add_tag", async () => {
    mockClient.callApi.mockResolvedValue({
      added_tag: { id: "tg1", tag: "important" }, added_kyou: { id: "tg1" }, errors: [],
    });
    const result = await server.handleToolCall("gkill_add_tag", { tag: "important", target_id: "k1" });
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/add_tag");
    expect(result.added_tag.tag).toBe("important");
  });

  test("dispatches gkill_add_text", async () => {
    mockClient.callApi.mockResolvedValue({
      added_text: { id: "tx1", text: "note" }, added_kyou: { id: "tx1" }, errors: [],
    });
    const result = await server.handleToolCall("gkill_add_text", { text: "note", target_id: "k1" });
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/add_text");
    expect(result.added_text.text).toBe("note");
  });

  test("dispatches gkill_submit_kftl", async () => {
    mockClient.callApi.mockResolvedValue({ messages: [{ message: "ok" }], errors: [] });
    const result = await server.handleToolCall("gkill_submit_kftl", { kftl_text: "/mi Buy milk" });
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/submit_kftl_text");
    expect(result.messages).toHaveLength(1);
  });

  test("dispatches gkill_delete_kyou", async () => {
    // First call: GET to fetch current entity
    mockClient.callApi.mockResolvedValueOnce({
      kmemo_histories: [{ id: "k1", content: "hello", is_deleted: false, rep_name: "rep1" }],
      errors: [],
    });
    // Second call: UPDATE with is_deleted=true
    mockClient.callApi.mockResolvedValueOnce({
      updated_kmemo: { id: "k1", is_deleted: true }, updated_kyou: { id: "k1" }, errors: [],
    });
    const result = await server.handleToolCall("gkill_delete_kyou", { id: "k1", data_type: "kmemo" });
    expect(mockClient.callApi.mock.calls[0][0]).toBe("/api/get_kmemo");
    expect(mockClient.callApi.mock.calls[1][0]).toBe("/api/update_kmemo");
    expect(result.updated_kmemo.is_deleted).toBe(true);
    expect(result.updated_kmemo.content).toBe("hello");
  });
});

// ---------------------------------------------------------------------------
// handleToolCall — entity defaults for write
// ---------------------------------------------------------------------------
describe("handleToolCall — entity defaults", () => {
  test("sets common fields on write entities", async () => {
    const mockClient = createMockClient();
    mockClient.callApi.mockResolvedValue({ added_kmemo: { id: "k1" }, added_kyou: { id: "k1" }, errors: [] });
    const server = new McpServer(mockClient);

    await server.handleToolCall("gkill_add_kmemo", { content: "test" });
    const [, body] = mockClient.callApi.mock.calls[0];
    const kmemo = body.kmemo;
    expect(kmemo.id).toBeTruthy();
    expect(kmemo.rep_name).toBe("");
    expect(kmemo.data_type).toBe("kmemo");
    expect(kmemo.create_app).toBe("gkill_mcp_readwrite");
    expect(kmemo.create_device).toBe("mcp");
    expect(kmemo.create_user).toBe("testuser");
    expect(kmemo.is_deleted).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// handleToolCall — unknown tool
// ---------------------------------------------------------------------------
describe("handleToolCall — error cases", () => {
  test("throws for unknown tool", async () => {
    const server = new McpServer(createMockClient());
    await expect(server.handleToolCall("unknown_tool", {})).rejects.toThrow("Unknown tool");
  });
});

// ---------------------------------------------------------------------------
// JSON-RPC protocol
// ---------------------------------------------------------------------------
describe("JSON-RPC protocol", () => {
  let server;

  beforeEach(() => {
    server = new McpServer(createMockClient());
  });

  test("initialize returns server info", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0", id: 1, method: "initialize", params: {},
    });
    expect(response.result.serverInfo.name).toBe("gkill-readwrite-mcp");
    expect(response.result.protocolVersion).toBe("2024-11-05");
  });

  test("ping returns empty result", async () => {
    const response = await server.handleMessage({ jsonrpc: "2.0", id: 2, method: "ping" });
    expect(response.result).toEqual({});
  });

  test("tools/list returns 27 tools", async () => {
    const response = await server.handleMessage({ jsonrpc: "2.0", id: 3, method: "tools/list" });
    expect(response.result.tools).toHaveLength(27);
  });

  test("tools/list includes all expected tool names", async () => {
    const response = await server.handleMessage({ jsonrpc: "2.0", id: 4, method: "tools/list" });
    const names = response.result.tools.map((t) => t.name);
    // Read tools
    expect(names).toContain("gkill_get_kyous");
    expect(names).toContain("gkill_get_mi_board_list");
    expect(names).toContain("gkill_get_all_tag_names");
    expect(names).toContain("gkill_get_all_rep_names");
    expect(names).toContain("gkill_get_gps_log");
    expect(names).toContain("gkill_get_application_config");
    expect(names).toContain("gkill_get_idf_file");
    // Write tools
    expect(names).toContain("gkill_add_kmemo");
    expect(names).toContain("gkill_add_urlog");
    expect(names).toContain("gkill_add_nlog");
    expect(names).toContain("gkill_add_lantana");
    expect(names).toContain("gkill_add_timeis");
    expect(names).toContain("gkill_add_mi");
    expect(names).toContain("gkill_add_kc");
    expect(names).toContain("gkill_add_tag");
    expect(names).toContain("gkill_add_text");
    expect(names).toContain("gkill_submit_kftl");
    expect(names).toContain("gkill_delete_kyou");
    // Update tools
    expect(names).toContain("gkill_update_kmemo");
    expect(names).toContain("gkill_update_urlog");
    expect(names).toContain("gkill_update_nlog");
    expect(names).toContain("gkill_update_lantana");
    expect(names).toContain("gkill_update_timeis");
    expect(names).toContain("gkill_update_mi");
    expect(names).toContain("gkill_update_kc");
    expect(names).toContain("gkill_update_tag");
    expect(names).toContain("gkill_update_text");
  });

  test("unknown method returns error", async () => {
    const response = await server.handleMessage({ jsonrpc: "2.0", id: 5, method: "unknown/method" });
    expect(response.error.code).toBe(-32601);
  });

  test("notifications/initialized returns null", async () => {
    const response = await server.handleMessage({ jsonrpc: "2.0", method: "notifications/initialized" });
    expect(response).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// buildToolResult — IDF image block
// ---------------------------------------------------------------------------
describe("buildToolResult", () => {
  test("includes image block for IDF images", () => {
    const server = new McpServer(createMockClient());
    const payload = {
      file_name: "photo.png",
      mime_type: "image/png",
      file_size_bytes: 1024,
      is_image: true,
      file_content_base64: "iVBORw0KGgo=",
    };
    const result = server.buildToolResult("gkill_get_idf_file", payload, false);
    expect(result.isError).toBe(false);
    const imageBlock = result.content.find((c) => c.type === "image");
    expect(imageBlock).toBeDefined();
    expect(imageBlock.data).toBe("iVBORw0KGgo=");
    expect(imageBlock.mimeType).toBe("image/png");
  });

  test("excludes file_content_base64 from text for IDF", () => {
    const server = new McpServer(createMockClient());
    const payload = {
      file_name: "doc.pdf",
      mime_type: "application/pdf",
      file_size_bytes: 2048,
      is_image: false,
      file_content_base64: "AAAA",
    };
    const result = server.buildToolResult("gkill_get_idf_file", payload, false);
    const textBlock = result.content.find((c) => c.type === "text");
    expect(textBlock.text).not.toContain("AAAA");
  });

  test("includes structuredContent for write tools", () => {
    const server = new McpServer(createMockClient());
    const result = server.buildToolResult("gkill_add_kmemo", { added_kmemo: { id: "k1" } }, false);
    expect(result.structuredContent).toBeDefined();
    expect(result.content[0].text).toContain("Created kmemo: k1");
  });
});

// ---------------------------------------------------------------------------
// handlePayload — batch
// ---------------------------------------------------------------------------
describe("handlePayload", () => {
  test("handles batch of messages", async () => {
    const server = new McpServer(createMockClient());
    const result = await server.handlePayload([
      { jsonrpc: "2.0", id: 1, method: "ping" },
      { jsonrpc: "2.0", id: 2, method: "ping" },
    ]);
    expect(result).toHaveLength(2);
  });

  test("handles single message", async () => {
    const server = new McpServer(createMockClient());
    const result = await server.handlePayload({ jsonrpc: "2.0", id: 1, method: "ping" });
    expect(result.result).toEqual({});
  });
});
