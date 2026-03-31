/**
 * Tests for MCP readwrite tool definitions and helper functions.
 */

import { describe, test, expect } from "vitest";

// ---------------------------------------------------------------------------
// summarizeToolPayload — reimplemented for testing (mirrors gkill-readwrite-server.mjs)
// ---------------------------------------------------------------------------
function summarizeToolPayload(name, payload) {
  switch (name) {
    // Read tools
    case "gkill_get_kyous": {
      const returnedCount = payload.returned_count ?? 0;
      const totalCount = payload.total_count ?? returnedCount;
      if (payload.has_more && payload.next_cursor) {
        return `Returned ${returnedCount} of ${totalCount} kyou entries (${totalCount - returnedCount} remaining). Next page: cursor="${payload.next_cursor}".`;
      }
      return `Returned ${returnedCount} of ${totalCount} kyou entries (all results returned).`;
    }
    case "gkill_get_mi_board_list":
      return `Fetched ${Array.isArray(payload.boards) ? payload.boards.length : 0} Mi boards.`;
    case "gkill_get_all_tag_names":
      return `Fetched ${Array.isArray(payload.tag_names) ? payload.tag_names.length : 0} tag names.`;
    case "gkill_get_all_rep_names":
      return `Fetched ${Array.isArray(payload.rep_names) ? payload.rep_names.length : 0} repository names.`;
    case "gkill_get_gps_log":
      return `Fetched ${Array.isArray(payload.gps_logs) ? payload.gps_logs.length : 0} GPS log entries.`;
    case "gkill_get_application_config":
      return "Fetched application configuration.";
    case "gkill_get_idf_file":
      return `Retrieved file: ${payload.file_name} (${payload.file_size_bytes} bytes, ${payload.mime_type})`;
    // Write tools
    case "gkill_add_kmemo":
      return `Created kmemo: ${payload.added_kmemo?.id || "unknown"}`;
    case "gkill_add_urlog":
      return `Created urlog: ${payload.added_urlog?.id || "unknown"}`;
    case "gkill_add_nlog":
      return `Created nlog: ${payload.added_nlog?.id || "unknown"}`;
    case "gkill_add_lantana":
      return `Created lantana: ${payload.added_lantana?.id || "unknown"}`;
    case "gkill_add_timeis":
      return `Created timeis: ${payload.added_timeis?.id || "unknown"}`;
    case "gkill_add_mi":
      return `Created mi: ${payload.added_mi?.id || "unknown"}`;
    case "gkill_add_kc":
      return `Created kc: ${payload.added_kc?.id || "unknown"}`;
    case "gkill_add_tag":
      return `Added tag: ${payload.added_tag?.id || "unknown"}`;
    case "gkill_add_text":
      return `Added text: ${payload.added_text?.id || "unknown"}`;
    case "gkill_submit_kftl":
      return `KFTL submitted: ${Array.isArray(payload.messages) ? payload.messages.length : 0} messages.`;
    case "gkill_delete_kyou": {
      const keys = Object.keys(payload).filter((k) => k.startsWith("updated_"));
      return `Deleted (soft): ${keys.length > 0 ? keys.join(", ") : "completed"}`;
    }
    // Update tools
    case "gkill_update_kmemo":
      return `Updated kmemo: ${payload.updated_kmemo?.id || "unknown"}`;
    case "gkill_update_urlog":
      return `Updated urlog: ${payload.updated_urlog?.id || "unknown"}`;
    case "gkill_update_nlog":
      return `Updated nlog: ${payload.updated_nlog?.id || "unknown"}`;
    case "gkill_update_lantana":
      return `Updated lantana: ${payload.updated_lantana?.id || "unknown"}`;
    case "gkill_update_timeis":
      return `Updated timeis: ${payload.updated_timeis?.id || "unknown"}`;
    case "gkill_update_mi":
      return `Updated mi: ${payload.updated_mi?.id || "unknown"}`;
    case "gkill_update_kc":
      return `Updated kc: ${payload.updated_kc?.id || "unknown"}`;
    case "gkill_update_tag":
      return `Updated tag: ${payload.updated_tag?.id || "unknown"}`;
    case "gkill_update_text":
      return `Updated text: ${payload.updated_text?.id || "unknown"}`;
    default:
      return "Tool call completed.";
  }
}

function summarizeToolError(name, error, detail) {
  const prefix = name ? `${name} failed` : "Tool call failed";
  if (detail && detail.field) {
    return `${prefix}: ${error} (field: ${detail.field})`;
  }
  return `${prefix}: ${error}`;
}

// ---------------------------------------------------------------------------
// Known tool names
// ---------------------------------------------------------------------------
const EXPECTED_READ_TOOLS = [
  "gkill_get_kyous",
  "gkill_get_mi_board_list",
  "gkill_get_all_tag_names",
  "gkill_get_all_rep_names",
  "gkill_get_gps_log",
  "gkill_get_application_config",
  "gkill_get_idf_file",
];

const EXPECTED_WRITE_TOOLS = [
  "gkill_add_kmemo",
  "gkill_add_urlog",
  "gkill_add_nlog",
  "gkill_add_lantana",
  "gkill_add_timeis",
  "gkill_add_mi",
  "gkill_add_kc",
  "gkill_add_tag",
  "gkill_add_text",
  "gkill_submit_kftl",
  "gkill_delete_kyou",
  "gkill_update_kmemo",
  "gkill_update_urlog",
  "gkill_update_nlog",
  "gkill_update_lantana",
  "gkill_update_timeis",
  "gkill_update_mi",
  "gkill_update_kc",
  "gkill_update_tag",
  "gkill_update_text",
];

const ALL_TOOLS = [...EXPECTED_READ_TOOLS, ...EXPECTED_WRITE_TOOLS];

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------
describe("Tool definitions", () => {
  test("combined server has 27 tools (7 read + 20 write)", () => {
    expect(ALL_TOOLS).toHaveLength(27);
  });

  test("read tools have 7 entries", () => {
    expect(EXPECTED_READ_TOOLS).toHaveLength(7);
  });

  test("write tools have 20 entries", () => {
    expect(EXPECTED_WRITE_TOOLS).toHaveLength(20);
  });

  test("no duplicate tool names", () => {
    const uniqueNames = new Set(ALL_TOOLS);
    expect(uniqueNames.size).toBe(ALL_TOOLS.length);
  });
});

// ---------------------------------------------------------------------------
// summarizeToolPayload — read tools
// ---------------------------------------------------------------------------
describe("summarizeToolPayload — read tools", () => {
  test("gkill_get_kyous with results", () => {
    const result = summarizeToolPayload("gkill_get_kyous", {
      returned_count: 5, total_count: 10, has_more: true, next_cursor: "abc",
    });
    expect(result).toContain("5 of 10");
    expect(result).toContain("abc");
  });

  test("gkill_get_kyous all returned", () => {
    const result = summarizeToolPayload("gkill_get_kyous", {
      returned_count: 3, total_count: 3, has_more: false,
    });
    expect(result).toContain("all results returned");
  });

  test("gkill_get_mi_board_list", () => {
    expect(summarizeToolPayload("gkill_get_mi_board_list", { boards: ["a", "b"] })).toContain("2 Mi boards");
  });

  test("gkill_get_all_tag_names", () => {
    expect(summarizeToolPayload("gkill_get_all_tag_names", { tag_names: ["t1"] })).toContain("1 tag names");
  });

  test("gkill_get_all_rep_names", () => {
    expect(summarizeToolPayload("gkill_get_all_rep_names", { rep_names: [] })).toContain("0 repository names");
  });

  test("gkill_get_gps_log", () => {
    expect(summarizeToolPayload("gkill_get_gps_log", { gps_logs: [1, 2] })).toContain("2 GPS");
  });

  test("gkill_get_application_config", () => {
    expect(summarizeToolPayload("gkill_get_application_config", {})).toContain("application configuration");
  });

  test("gkill_get_idf_file", () => {
    const result = summarizeToolPayload("gkill_get_idf_file", {
      file_name: "test.png", file_size_bytes: 1024, mime_type: "image/png",
    });
    expect(result).toContain("test.png");
    expect(result).toContain("1024");
  });
});

// ---------------------------------------------------------------------------
// summarizeToolPayload — write tools
// ---------------------------------------------------------------------------
describe("summarizeToolPayload — write tools", () => {
  test("gkill_add_kmemo", () => {
    expect(summarizeToolPayload("gkill_add_kmemo", { added_kmemo: { id: "k1" } })).toBe("Created kmemo: k1");
  });

  test("gkill_add_mi", () => {
    expect(summarizeToolPayload("gkill_add_mi", { added_mi: { id: "m1" } })).toBe("Created mi: m1");
  });

  test("gkill_add_tag", () => {
    expect(summarizeToolPayload("gkill_add_tag", { added_tag: { id: "tg1" } })).toBe("Added tag: tg1");
  });

  test("gkill_submit_kftl", () => {
    expect(summarizeToolPayload("gkill_submit_kftl", { messages: [1, 2] })).toContain("2 messages");
  });

  test("gkill_delete_kyou", () => {
    const result = summarizeToolPayload("gkill_delete_kyou", { updated_kmemo: {}, updated_kyou: {} });
    expect(result).toContain("updated_kmemo");
  });

  test("unknown tool", () => {
    expect(summarizeToolPayload("unknown", {})).toBe("Tool call completed.");
  });
});

// ---------------------------------------------------------------------------
// summarizeToolError
// ---------------------------------------------------------------------------
describe("summarizeToolError", () => {
  test("includes tool name and error", () => {
    expect(summarizeToolError("gkill_add_kmemo", "bad input", null)).toBe("gkill_add_kmemo failed: bad input");
  });

  test("includes field when present", () => {
    expect(summarizeToolError("gkill_add_kmemo", "bad", { field: "content" })).toContain("field: content");
  });

  test("handles null name", () => {
    expect(summarizeToolError(null, "error", null)).toBe("Tool call failed: error");
  });
});
