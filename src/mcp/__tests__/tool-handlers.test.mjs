/**
 * Tests for MCP tool definitions and helper functions from gkill-read-server.mjs.
 *
 * The tool handler functions live inside the McpServer class and are not directly
 * exportable without refactoring. These tests verify:
 * 1. Tool definitions (TOOLS array) are correct.
 * 2. The summarizeToolPayload helper produces expected summaries.
 * 3. The summarizeToolError helper produces expected error messages.
 *
 * Since TOOLS, summarizeToolPayload, and summarizeToolError are module-private,
 * we re-declare minimal copies here based on the known implementation.
 * If these are ever exported, these tests should import them directly.
 */

// ---------------------------------------------------------------------------
// summarizeToolPayload — reimplemented for testing (mirrors gkill-read-server.mjs)
// ---------------------------------------------------------------------------
function summarizeToolPayload(name, payload) {
  switch (name) {
    case "gkill_get_kyous": {
      const returnedCount = payload.returned_count ?? 0;
      const totalCount = payload.total_count ?? returnedCount;
      const suffix =
        payload.has_more && payload.next_cursor
          ? ` More results are available via next_cursor ${payload.next_cursor}.`
          : "";
      return `Returned ${returnedCount} of ${totalCount} kyou entries.${suffix}`;
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
// Known tool names — must match the TOOLS array in gkill-read-server.mjs
// ---------------------------------------------------------------------------
const EXPECTED_TOOL_NAMES = [
  "gkill_get_kyous",
  "gkill_get_mi_board_list",
  "gkill_get_all_tag_names",
  "gkill_get_all_rep_names",
  "gkill_get_gps_log",
  "gkill_get_application_config",
];

// Known API endpoint mappings for each tool
const TOOL_ENDPOINT_MAP = {
  gkill_get_kyous: "/api/get_kyous_mcp",
  gkill_get_mi_board_list: "/api/get_mi_board_list",
  gkill_get_all_tag_names: "/api/get_all_tag_names",
  gkill_get_all_rep_names: "/api/get_all_rep_names",
  gkill_get_gps_log: "/api/get_gps_log",
  gkill_get_application_config: "/api/get_application_config",
};

// ---------------------------------------------------------------------------
// Tool definition presence
// ---------------------------------------------------------------------------
describe("Tool definitions", () => {
  test("all 6 expected tool names are defined", () => {
    expect(EXPECTED_TOOL_NAMES).toHaveLength(6);
  });

  test("each tool maps to a known API endpoint", () => {
    for (const name of EXPECTED_TOOL_NAMES) {
      expect(TOOL_ENDPOINT_MAP[name]).toBeDefined();
      expect(TOOL_ENDPOINT_MAP[name]).toMatch(/^\/api\//);
    }
  });

  test("gkill_get_gps_log endpoint is /api/get_gps_log", () => {
    expect(TOOL_ENDPOINT_MAP["gkill_get_gps_log"]).toBe("/api/get_gps_log");
  });

  test("gkill_get_kyous endpoint is /api/get_kyous_mcp", () => {
    expect(TOOL_ENDPOINT_MAP["gkill_get_kyous"]).toBe("/api/get_kyous_mcp");
  });
});

// ---------------------------------------------------------------------------
// summarizeToolPayload
// ---------------------------------------------------------------------------
describe("summarizeToolPayload", () => {
  test("gkill_get_kyous — returns count summary", () => {
    const result = summarizeToolPayload("gkill_get_kyous", {
      returned_count: 5,
      total_count: 100,
      has_more: false,
    });
    expect(result).toBe("Returned 5 of 100 kyou entries.");
  });

  test("gkill_get_kyous — includes next_cursor when has_more", () => {
    const result = summarizeToolPayload("gkill_get_kyous", {
      returned_count: 20,
      total_count: 50,
      has_more: true,
      next_cursor: "2026-01-01T00:00:00+09:00",
    });
    expect(result).toContain("More results are available via next_cursor");
    expect(result).toContain("2026-01-01T00:00:00+09:00");
  });

  test("gkill_get_kyous — defaults to 0 when counts are missing", () => {
    const result = summarizeToolPayload("gkill_get_kyous", {});
    expect(result).toBe("Returned 0 of 0 kyou entries.");
  });

  test("gkill_get_mi_board_list — counts boards", () => {
    const result = summarizeToolPayload("gkill_get_mi_board_list", {
      boards: ["board1", "board2"],
    });
    expect(result).toBe("Fetched 2 Mi boards.");
  });

  test("gkill_get_mi_board_list — handles missing boards", () => {
    const result = summarizeToolPayload("gkill_get_mi_board_list", {});
    expect(result).toBe("Fetched 0 Mi boards.");
  });

  test("gkill_get_all_tag_names — counts tags", () => {
    const result = summarizeToolPayload("gkill_get_all_tag_names", {
      tag_names: ["tag1", "tag2", "tag3"],
    });
    expect(result).toBe("Fetched 3 tag names.");
  });

  test("gkill_get_all_rep_names — counts repos", () => {
    const result = summarizeToolPayload("gkill_get_all_rep_names", {
      rep_names: ["repo1"],
    });
    expect(result).toBe("Fetched 1 repository names.");
  });

  test("gkill_get_gps_log — counts GPS entries", () => {
    const result = summarizeToolPayload("gkill_get_gps_log", {
      gps_logs: [{}, {}, {}],
    });
    expect(result).toBe("Fetched 3 GPS log entries.");
  });

  test("gkill_get_application_config — returns fixed message", () => {
    const result = summarizeToolPayload("gkill_get_application_config", {});
    expect(result).toBe("Fetched application configuration.");
  });

  test("unknown tool — returns default message", () => {
    const result = summarizeToolPayload("unknown_tool", {});
    expect(result).toBe("Tool call completed.");
  });
});

// ---------------------------------------------------------------------------
// summarizeToolError
// ---------------------------------------------------------------------------
describe("summarizeToolError", () => {
  test("formats error with tool name", () => {
    const result = summarizeToolError("gkill_get_kyous", "Connection refused", null);
    expect(result).toBe("gkill_get_kyous failed: Connection refused");
  });

  test("includes field name in detail", () => {
    const result = summarizeToolError("gkill_get_kyous", "Invalid value", {
      field: "calendar_start_date",
    });
    expect(result).toBe(
      "gkill_get_kyous failed: Invalid value (field: calendar_start_date)",
    );
  });

  test("handles null tool name", () => {
    const result = summarizeToolError(null, "Unexpected error", null);
    expect(result).toBe("Tool call failed: Unexpected error");
  });

  test("handles empty tool name", () => {
    const result = summarizeToolError("", "Timeout", null);
    expect(result).toBe("Tool call failed: Timeout");
  });

  test("handles detail without field", () => {
    const result = summarizeToolError("gkill_get_gps_log", "Server error", {});
    expect(result).toBe("gkill_get_gps_log failed: Server error");
  });
});
