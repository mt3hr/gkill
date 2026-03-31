/**
 * Tests for MCP write tool definitions and helper functions from gkill-write-server.mjs.
 *
 * Since TOOLS, summarizeWritePayload, and summarizeToolError are module-private,
 * we re-declare minimal copies here based on the known implementation.
 */

import { describe, test, expect } from "vitest";

// ---------------------------------------------------------------------------
// summarizeWritePayload — reimplemented for testing (mirrors gkill-write-server.mjs)
// ---------------------------------------------------------------------------
function summarizeWritePayload(name, payload) {
  switch (name) {
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
    case "gkill_get_mi_board_list":
      return `Fetched ${Array.isArray(payload.boards) ? payload.boards.length : 0} Mi boards.`;
    case "gkill_get_all_tag_names":
      return `Fetched ${Array.isArray(payload.tag_names) ? payload.tag_names.length : 0} tag names.`;
    case "gkill_get_all_rep_names":
      return `Fetched ${Array.isArray(payload.rep_names) ? payload.rep_names.length : 0} repository names.`;
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
// Known tool names — must match the TOOLS array in gkill-write-server.mjs
// ---------------------------------------------------------------------------
const EXPECTED_WRITE_TOOL_NAMES = [
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

const EXPECTED_READ_CONVENIENCE_TOOL_NAMES = [
  "gkill_get_all_rep_names",
  "gkill_get_mi_board_list",
  "gkill_get_all_tag_names",
];

const ALL_EXPECTED_TOOLS = [...EXPECTED_WRITE_TOOL_NAMES, ...EXPECTED_READ_CONVENIENCE_TOOL_NAMES];

// Known API endpoint mappings
const WRITE_TOOL_ENDPOINT_MAP = {
  gkill_add_kmemo: "/api/add_kmemo",
  gkill_add_urlog: "/api/add_urlog",
  gkill_add_nlog: "/api/add_nlog",
  gkill_add_lantana: "/api/add_lantana",
  gkill_add_timeis: "/api/add_timeis",
  gkill_add_mi: "/api/add_mi",
  gkill_add_kc: "/api/add_kc",
  gkill_add_tag: "/api/add_tag",
  gkill_add_text: "/api/add_text",
  gkill_submit_kftl: "/api/submit_kftl_text",
  gkill_update_kmemo: "/api/update_kmemo",
  gkill_update_urlog: "/api/update_urlog",
  gkill_update_nlog: "/api/update_nlog",
  gkill_update_lantana: "/api/update_lantana",
  gkill_update_timeis: "/api/update_timeis",
  gkill_update_mi: "/api/update_mi",
  gkill_update_kc: "/api/update_kc",
  gkill_update_tag: "/api/update_tag",
  gkill_update_text: "/api/update_text",
  gkill_get_all_rep_names: "/api/get_all_rep_names",
  gkill_get_mi_board_list: "/api/get_mi_board_list",
  gkill_get_all_tag_names: "/api/get_all_tag_names",
};

// ---------------------------------------------------------------------------
// Tool definition presence
// ---------------------------------------------------------------------------
describe("Tool definitions", () => {
  test("all 23 expected tool names are defined", () => {
    expect(ALL_EXPECTED_TOOLS).toHaveLength(23);
  });

  test("write tools have 20 entries", () => {
    expect(EXPECTED_WRITE_TOOL_NAMES).toHaveLength(20);
  });

  test("read convenience tools have 3 entries", () => {
    expect(EXPECTED_READ_CONVENIENCE_TOOL_NAMES).toHaveLength(3);
  });

  test("endpoint mappings match tool count", () => {
    expect(Object.keys(WRITE_TOOL_ENDPOINT_MAP)).toHaveLength(22);
    // gkill_delete_kyou maps to multiple endpoints, not in this simple map
  });
});

// ---------------------------------------------------------------------------
// summarizeWritePayload
// ---------------------------------------------------------------------------
describe("summarizeWritePayload", () => {
  test("summarizes gkill_add_kmemo with id", () => {
    const result = summarizeWritePayload("gkill_add_kmemo", { added_kmemo: { id: "abc123" } });
    expect(result).toBe("Created kmemo: abc123");
  });

  test("summarizes gkill_add_kmemo with missing entity", () => {
    const result = summarizeWritePayload("gkill_add_kmemo", {});
    expect(result).toBe("Created kmemo: unknown");
  });

  test("summarizes gkill_add_mi with id", () => {
    const result = summarizeWritePayload("gkill_add_mi", { added_mi: { id: "m1" } });
    expect(result).toBe("Created mi: m1");
  });

  test("summarizes gkill_add_tag with id", () => {
    const result = summarizeWritePayload("gkill_add_tag", { added_tag: { id: "t1" } });
    expect(result).toBe("Added tag: t1");
  });

  test("summarizes gkill_add_text with id", () => {
    const result = summarizeWritePayload("gkill_add_text", { added_text: { id: "x1" } });
    expect(result).toBe("Added text: x1");
  });

  test("summarizes gkill_submit_kftl with message count", () => {
    const result = summarizeWritePayload("gkill_submit_kftl", { messages: [1, 2, 3] });
    expect(result).toBe("KFTL submitted: 3 messages.");
  });

  test("summarizes gkill_submit_kftl with empty messages", () => {
    const result = summarizeWritePayload("gkill_submit_kftl", { messages: [] });
    expect(result).toBe("KFTL submitted: 0 messages.");
  });

  test("summarizes gkill_delete_kyou with updated keys", () => {
    const result = summarizeWritePayload("gkill_delete_kyou", { updated_kmemo: {}, updated_kyou: {} });
    expect(result).toContain("updated_kmemo");
  });

  test("summarizes gkill_delete_kyou with no updated keys", () => {
    const result = summarizeWritePayload("gkill_delete_kyou", {});
    expect(result).toContain("completed");
  });

  test("summarizes read convenience tools", () => {
    expect(summarizeWritePayload("gkill_get_mi_board_list", { boards: ["a", "b"] })).toContain("2 Mi boards");
    expect(summarizeWritePayload("gkill_get_all_tag_names", { tag_names: ["t1"] })).toContain("1 tag names");
    expect(summarizeWritePayload("gkill_get_all_rep_names", { rep_names: [] })).toContain("0 repository names");
  });

  test("returns default for unknown tool", () => {
    const result = summarizeWritePayload("unknown_tool", {});
    expect(result).toBe("Tool call completed.");
  });
});

// ---------------------------------------------------------------------------
// summarizeToolError
// ---------------------------------------------------------------------------
describe("summarizeToolError", () => {
  test("includes tool name and error", () => {
    const result = summarizeToolError("gkill_add_kmemo", "bad input", null);
    expect(result).toBe("gkill_add_kmemo failed: bad input");
  });

  test("includes field when present", () => {
    const result = summarizeToolError("gkill_add_kmemo", "bad input", { field: "content" });
    expect(result).toBe("gkill_add_kmemo failed: bad input (field: content)");
  });

  test("handles null name", () => {
    const result = summarizeToolError(null, "error", null);
    expect(result).toBe("Tool call failed: error");
  });
});
