/**
 * Tests for McpWriteServer from gkill-write-server.mjs.
 *
 * The server is tested with a mock GkillWriteClient so no real HTTP calls are made.
 */

import { describe, test, expect, vi, beforeEach } from "vitest";
import { McpWriteServer } from "../gkill-write-server.mjs";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function createMockClient(overrides = {}) {
  return {
    callWrite: vi.fn().mockResolvedValue({ errors: [], messages: [] }),
    login: vi.fn().mockResolvedValue("mock-session-id"),
    defaultLocale: "ja",
    userId: "testuser",
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------
describe("McpWriteServer constructor", () => {
  test("accepts a client", () => {
    const mock = createMockClient();
    const server = new McpWriteServer(mock);
    expect(server.client).toBe(mock);
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
    server = new McpWriteServer(mockClient);
  });

  test("dispatches gkill_add_kmemo to /api/add_kmemo", async () => {
    mockClient.callWrite.mockResolvedValue({
      added_kmemo: { id: "k1", content: "hello" },
      added_kyou: { id: "k1", data_type: "kmemo" },
      errors: [],
    });

    const result = await server.handleToolCall("gkill_add_kmemo", { content: "hello" });

    expect(mockClient.callWrite).toHaveBeenCalledTimes(1);
    const [pathname, body] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/add_kmemo");
    expect(body.kmemo.content).toBe("hello");
    expect(body.want_response_kyou).toBe(true);
    expect(result.added_kmemo.id).toBe("k1");
    expect(result.added_kyou.data_type).toBe("kmemo");
  });

  test("dispatches gkill_add_urlog to /api/add_urlog", async () => {
    mockClient.callWrite.mockResolvedValue({
      added_urlog: { id: "u1", url: "https://example.com" },
      added_kyou: { id: "u1" },
      errors: [],
    });

    const result = await server.handleToolCall("gkill_add_urlog", { url: "https://example.com" });

    const [pathname, body] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/add_urlog");
    expect(body.urlog.url).toBe("https://example.com");
    expect(result.added_urlog.id).toBe("u1");
  });

  test("dispatches gkill_add_nlog to /api/add_nlog", async () => {
    mockClient.callWrite.mockResolvedValue({
      added_nlog: { id: "n1", amount: 1500 },
      added_kyou: { id: "n1" },
      errors: [],
    });

    const result = await server.handleToolCall("gkill_add_nlog", { title: "lunch", amount: 1500 });

    const [pathname, body] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/add_nlog");
    expect(body.nlog.amount).toBe(1500);
    expect(body.nlog.title).toBe("lunch");
    expect(result.added_nlog.amount).toBe(1500);
  });

  test("dispatches gkill_add_lantana to /api/add_lantana", async () => {
    mockClient.callWrite.mockResolvedValue({
      added_lantana: { id: "l1", mood: 7 },
      added_kyou: { id: "l1" },
      errors: [],
    });

    const result = await server.handleToolCall("gkill_add_lantana", { mood: 7 });

    const [pathname, body] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/add_lantana");
    expect(body.lantana.mood).toBe(7);
    expect(result.added_lantana.mood).toBe(7);
  });

  test("dispatches gkill_add_timeis to /api/add_timeis", async () => {
    mockClient.callWrite.mockResolvedValue({
      added_timeis: { id: "t1", title: "coding" },
      added_kyou: { id: "t1" },
      errors: [],
    });

    const result = await server.handleToolCall("gkill_add_timeis", { title: "coding" });

    const [pathname, body] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/add_timeis");
    expect(body.timeis.title).toBe("coding");
    expect(result.added_timeis.title).toBe("coding");
  });

  test("dispatches gkill_add_mi to /api/add_mi", async () => {
    mockClient.callWrite.mockResolvedValue({
      added_mi: { id: "m1", title: "fix bug", board_name: "dev" },
      added_kyou: { id: "m1" },
      errors: [],
    });

    const result = await server.handleToolCall("gkill_add_mi", { title: "fix bug", board_name: "dev" });

    const [pathname, body] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/add_mi");
    expect(body.mi.title).toBe("fix bug");
    expect(body.mi.board_name).toBe("dev");
    expect(body.mi.is_checked).toBe(false);
    expect(result.added_mi.title).toBe("fix bug");
  });

  test("dispatches gkill_add_kc to /api/add_kc", async () => {
    mockClient.callWrite.mockResolvedValue({
      added_kc: { id: "c1", num_value: 42 },
      added_kyou: { id: "c1" },
      errors: [],
    });

    const result = await server.handleToolCall("gkill_add_kc", { title: "steps", num_value: 42 });

    const [pathname, body] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/add_kc");
    expect(body.kc.num_value).toBe(42);
    expect(result.added_kc.num_value).toBe(42);
  });

  test("dispatches gkill_add_tag to /api/add_tag", async () => {
    mockClient.callWrite.mockResolvedValue({
      added_tag: { id: "tg1", tag: "important", target_id: "k1" },
      added_kyou: { id: "tg1" },
      errors: [],
    });

    const result = await server.handleToolCall("gkill_add_tag", { tag: "important", target_id: "k1" });

    const [pathname, body] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/add_tag");
    expect(body.tag.tag).toBe("important");
    expect(body.tag.target_id).toBe("k1");
    expect(result.added_tag.tag).toBe("important");
  });

  test("dispatches gkill_add_text to /api/add_text", async () => {
    mockClient.callWrite.mockResolvedValue({
      added_text: { id: "tx1", text: "note", target_id: "k1" },
      added_kyou: { id: "tx1" },
      errors: [],
    });

    const result = await server.handleToolCall("gkill_add_text", { text: "note", target_id: "k1" });

    const [pathname, body] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/add_text");
    expect(body.text.text).toBe("note");
    expect(body.text.target_id).toBe("k1");
    expect(result.added_text.text).toBe("note");
  });

  test("dispatches gkill_submit_kftl to /api/submit_kftl_text", async () => {
    mockClient.callWrite.mockResolvedValue({
      messages: [{ message: "created 2 records" }],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_submit_kftl", { kftl_text: "/mi Buy milk" });

    const [pathname, body] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/submit_kftl_text");
    expect(body.kftl_text).toBe("/mi Buy milk");
    expect(result.messages).toHaveLength(1);
  });

  test("dispatches gkill_delete_kyou to correct update endpoint", async () => {
    // First call: GET to fetch current entity
    mockClient.callWrite.mockResolvedValueOnce({
      kmemo_histories: [{ id: "k1", content: "hello", is_deleted: false, rep_name: "rep1" }],
      errors: [],
    });
    // Second call: UPDATE with is_deleted=true
    mockClient.callWrite.mockResolvedValueOnce({
      updated_kmemo: { id: "k1", is_deleted: true },
      updated_kyou: { id: "k1" },
      errors: [],
    });

    const result = await server.handleToolCall("gkill_delete_kyou", { id: "k1", data_type: "kmemo" });

    const [getPathname] = mockClient.callWrite.mock.calls[0];
    expect(getPathname).toBe("/api/get_kmemo");
    const [updatePathname, body] = mockClient.callWrite.mock.calls[1];
    expect(updatePathname).toBe("/api/update_kmemo");
    expect(body.kmemo.id).toBe("k1");
    expect(body.kmemo.is_deleted).toBe(true);
    expect(body.kmemo.content).toBe("hello");
    expect(result.updated_kmemo.is_deleted).toBe(true);
  });

  test("gkill_delete_kyou works for mi data_type", async () => {
    // First call: GET to fetch current entity
    mockClient.callWrite.mockResolvedValueOnce({
      mi_histories: [{ id: "m1", title: "task1", is_deleted: false, rep_name: "rep1" }],
      errors: [],
    });
    // Second call: UPDATE with is_deleted=true
    mockClient.callWrite.mockResolvedValueOnce({
      updated_mi: { id: "m1", is_deleted: true },
      updated_kyou: { id: "m1" },
      errors: [],
    });

    const result = await server.handleToolCall("gkill_delete_kyou", { id: "m1", data_type: "mi" });

    const [getPathname] = mockClient.callWrite.mock.calls[0];
    expect(getPathname).toBe("/api/get_mi");
    const [updatePathname] = mockClient.callWrite.mock.calls[1];
    expect(updatePathname).toBe("/api/update_mi");
    expect(result.updated_mi.is_deleted).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// handleToolCall dispatch — Read convenience tools
// ---------------------------------------------------------------------------
describe("handleToolCall — read convenience tools", () => {
  let mockClient;
  let server;

  beforeEach(() => {
    mockClient = createMockClient();
    server = new McpWriteServer(mockClient);
  });

  test("dispatches gkill_get_all_rep_names to /api/get_all_rep_names", async () => {
    mockClient.callWrite.mockResolvedValue({
      rep_names: ["rep1", "rep2"],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_all_rep_names", {});

    const [pathname] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/get_all_rep_names");
    expect(result.rep_names).toEqual(["rep1", "rep2"]);
  });

  test("dispatches gkill_get_mi_board_list to /api/get_mi_board_list", async () => {
    mockClient.callWrite.mockResolvedValue({
      boards: ["board1", "board2"],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_mi_board_list", {});

    const [pathname] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/get_mi_board_list");
    expect(result.boards).toEqual(["board1", "board2"]);
  });

  test("dispatches gkill_get_all_tag_names to /api/get_all_tag_names", async () => {
    mockClient.callWrite.mockResolvedValue({
      tag_names: ["tag1", "tag2"],
      errors: [],
    });

    const result = await server.handleToolCall("gkill_get_all_tag_names", {});

    const [pathname] = mockClient.callWrite.mock.calls[0];
    expect(pathname).toBe("/api/get_all_tag_names");
    expect(result.tag_names).toEqual(["tag1", "tag2"]);
  });
});

// ---------------------------------------------------------------------------
// handleToolCall — unknown tool
// ---------------------------------------------------------------------------
describe("handleToolCall — error cases", () => {
  test("throws for unknown tool", async () => {
    const server = new McpWriteServer(createMockClient());
    await expect(server.handleToolCall("unknown_tool", {})).rejects.toThrow("Unknown tool");
  });
});

// ---------------------------------------------------------------------------
// handleToolCall — entity defaults
// ---------------------------------------------------------------------------
describe("handleToolCall — entity defaults", () => {
  test("sets common fields on kmemo entity", async () => {
    const mockClient = createMockClient();
    mockClient.callWrite.mockResolvedValue({
      added_kmemo: { id: "k1" },
      added_kyou: { id: "k1" },
      errors: [],
    });
    const server = new McpWriteServer(mockClient);

    await server.handleToolCall("gkill_add_kmemo", { content: "test" });

    const [, body] = mockClient.callWrite.mock.calls[0];
    const kmemo = body.kmemo;
    expect(kmemo.id).toBeTruthy();
    expect(kmemo.rep_name).toBe("");
    expect(kmemo.data_type).toBe("kmemo");
    expect(kmemo.create_app).toBe("gkill_mcp_write");
    expect(kmemo.create_device).toBe("mcp");
    expect(kmemo.create_user).toBe("testuser");
    expect(kmemo.is_deleted).toBe(false);
    expect(kmemo.create_time).toBeTruthy();
    expect(kmemo.update_time).toBeTruthy();
  });
});

// ---------------------------------------------------------------------------
// JSON-RPC protocol
// ---------------------------------------------------------------------------
describe("JSON-RPC protocol", () => {
  let server;

  beforeEach(() => {
    server = new McpWriteServer(createMockClient());
  });

  test("initialize returns server info", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: {},
    });
    expect(response.result.serverInfo.name).toBe("gkill-write-mcp");
    expect(response.result.protocolVersion).toBe("2024-11-05");
  });

  test("ping returns empty result", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0",
      id: 2,
      method: "ping",
    });
    expect(response.result).toEqual({});
  });

  test("tools/list returns 23 tools", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0",
      id: 3,
      method: "tools/list",
    });
    expect(response.result.tools).toHaveLength(23);
  });

  test("tools/list includes all expected tool names", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0",
      id: 4,
      method: "tools/list",
    });
    const names = response.result.tools.map((t) => t.name);
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
    // Read convenience tools
    expect(names).toContain("gkill_get_all_rep_names");
    expect(names).toContain("gkill_get_mi_board_list");
    expect(names).toContain("gkill_get_all_tag_names");
  });

  test("unknown method returns error", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0",
      id: 5,
      method: "unknown/method",
    });
    expect(response.error.code).toBe(-32601);
  });

  test("invalid request returns error", async () => {
    const response = await server.handleMessage({ id: 6 });
    expect(response.error.code).toBe(-32600);
  });

  test("notifications/initialized returns null", async () => {
    const response = await server.handleMessage({
      jsonrpc: "2.0",
      method: "notifications/initialized",
    });
    expect(response).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// buildToolResult
// ---------------------------------------------------------------------------
describe("buildToolResult", () => {
  test("includes structuredContent for success", () => {
    const server = new McpWriteServer(createMockClient());
    const result = server.buildToolResult("gkill_add_kmemo", { added_kmemo: { id: "k1" } }, false);
    expect(result.isError).toBe(false);
    expect(result.structuredContent).toBeDefined();
    expect(result.content[0].text).toContain("Created kmemo: k1");
  });

  test("marks errors correctly", () => {
    const server = new McpWriteServer(createMockClient());
    const result = server.buildToolResult("gkill_add_kmemo", { error: "test error" }, true);
    expect(result.isError).toBe(true);
    expect(result.content[0].text).toContain("gkill_add_kmemo failed");
  });
});

// ---------------------------------------------------------------------------
// handlePayload — batch
// ---------------------------------------------------------------------------
describe("handlePayload", () => {
  test("handles batch of messages", async () => {
    const server = new McpWriteServer(createMockClient());
    const result = await server.handlePayload([
      { jsonrpc: "2.0", id: 1, method: "ping" },
      { jsonrpc: "2.0", id: 2, method: "ping" },
    ]);
    expect(result).toHaveLength(2);
  });

  test("handles empty batch", async () => {
    const server = new McpWriteServer(createMockClient());
    const result = await server.handlePayload([]);
    expect(result.error.code).toBe(-32600);
  });

  test("handles single message", async () => {
    const server = new McpWriteServer(createMockClient());
    const result = await server.handlePayload({ jsonrpc: "2.0", id: 1, method: "ping" });
    expect(result.result).toEqual({});
  });
});
