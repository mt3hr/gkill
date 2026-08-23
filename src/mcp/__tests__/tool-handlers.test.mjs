/**
 * Tests for MCP read tool definitions and shared helpers.
 *
 * v2 から summarize / ディスパッチは lib/read-handlers.mjs に一本化され export されたので、
 * ここは再実装のミラーではなく**実物を直接 import して**検証する
 * （ミラーは実装が変わっても緑のまま古び、二重管理の温床だった）。
 */

import { describe, test, expect } from "vitest";

import { READ_TOOLS } from "../lib/read-tools.mjs";
import { PLUGIN_TOOLS } from "../lib/plugin-tools.mjs";
import { isReadToolName, summarizeReadToolPayload } from "../lib/read-handlers.mjs";
import { summarizeToolError } from "../lib/payload.mjs";

// ---------------------------------------------------------------------------
// Tool definition presence
// ---------------------------------------------------------------------------
describe("Tool definitions", () => {
  test("read server exposes 11 tools (10 read + 1 plugin)", () => {
    expect(READ_TOOLS).toHaveLength(10);
    expect(PLUGIN_TOOLS).toHaveLength(1);
  });

  test("read tool names are the v2 set", () => {
    expect(READ_TOOLS.map((tool) => tool.name)).toEqual([
      "gkill_get_kyous",
      "gkill_get_mi_board_list",
      "gkill_get_all_tag_names",
      "gkill_get_all_rep_names",
      "gkill_get_gps_log",
      "gkill_get_application_config",
      "gkill_get_rep_infos",
      "gkill_get_idf_file",
      "gkill_get_idf_file_path",
      "gkill_get_kyou_history",
    ]);
  });

  test("isReadToolName matches the definitions", () => {
    for (const tool of READ_TOOLS) {
      expect(isReadToolName(tool.name)).toBe(true);
    }
    expect(isReadToolName("gkill_get_plugin_list")).toBe(false);
    expect(isReadToolName("gkill_add_kmemo")).toBe(false);
  });

  test("every tool has an object inputSchema with additionalProperties: false", () => {
    for (const tool of READ_TOOLS) {
      expect(tool.inputSchema.type).toBe("object");
      expect(tool.inputSchema.additionalProperties).toBe(false);
    }
  });
});

// ---------------------------------------------------------------------------
// summarizeReadToolPayload (v2)
// ---------------------------------------------------------------------------
describe("summarizeReadToolPayload", () => {
  test("gkill_get_kyous — first page carries total and remaining", () => {
    const result = summarizeReadToolPayload("gkill_get_kyous", {
      kyous: [{}],
      returned_count: 20,
      total_count: 50,
      remaining_count: 30,
      has_more: true,
      next_cursor: "2026-01-01T00:00:00+09:00::abc",
    });
    expect(result).toContain("Returned 20 of 50");
    expect(result).toContain("30 remaining");
    expect(result).toContain("2026-01-01T00:00:00+09:00::abc");
  });

  test("gkill_get_kyous — cursor page has no total_count and must not claim completion", () => {
    // 旧実装は total_count ?? returned_count で「all results returned」と
    // 嘘の完了報告をしていた（v2ではtotal_countはcursor無し応答のみ）
    const result = summarizeReadToolPayload("gkill_get_kyous", {
      kyous: [{}],
      returned_count: 20,
      remaining_count: 5,
      has_more: true,
      next_cursor: "cursor",
    });
    expect(result).toContain("Returned 20 kyou entries (5 remaining)");
    expect(result).not.toContain("all results returned");
  });

  test("gkill_get_kyous — count_only", () => {
    const result = summarizeReadToolPayload("gkill_get_kyous", {
      kyous: [],
      returned_count: 0,
      remaining_count: 0,
      total_count: 124056,
      has_more: false,
    });
    expect(result).toBe("Counted 124056 entries.");
  });

  test("gkill_get_kyous — group_by buckets", () => {
    const result = summarizeReadToolPayload("gkill_get_kyous", {
      kyous: [],
      buckets: [{ key: "2026-01", count: 3 }, { key: "2026-02", count: 4 }],
      total_count: 7,
      returned_count: 0,
      remaining_count: 0,
      has_more: false,
    });
    expect(result).toBe("Aggregated 2 buckets (7 entries).");
  });

  test("gkill_get_mi_board_list — counts boards", () => {
    expect(summarizeReadToolPayload("gkill_get_mi_board_list", { boards: ["a", "b"] })).toBe("Fetched 2 Mi boards.");
  });

  test("gkill_get_all_tag_names — counts tags", () => {
    expect(summarizeReadToolPayload("gkill_get_all_tag_names", { tag_names: ["t1", "t2", "t3"] })).toBe("Fetched 3 tag names.");
  });

  test("gkill_get_all_rep_names — counts repos", () => {
    expect(summarizeReadToolPayload("gkill_get_all_rep_names", { rep_names: ["r"] })).toBe("Fetched 1 repository names.");
  });

  test("gkill_get_gps_log — paged points", () => {
    const result = summarizeReadToolPayload("gkill_get_gps_log", {
      gps_logs: [{}, {}, {}],
      returned_count: 3,
      remaining_count: 7,
      has_more: true,
      next_cursor: "abc",
    });
    expect(result).toContain("Returned 3 GPS points (7 remaining)");
  });

  test("gkill_get_gps_log — daily buckets", () => {
    const result = summarizeReadToolPayload("gkill_get_gps_log", {
      gps_logs: [],
      buckets: [{ key: "2026-08-01", count: 100 }],
      total_count: 100,
      has_more: false,
    });
    expect(result).toBe("Aggregated 1 daily buckets (100 GPS points).");
  });

  test("gkill_get_rep_infos — counts reps and types", () => {
    const result = summarizeReadToolPayload("gkill_get_rep_infos", {
      rep_infos: [{ rep_name: "Kmemo", rep_type: "kmemo" }],
      canonical_rep_types: ["kmemo", "kc"],
      plugins: [],
    });
    expect(result).toBe("Fetched 1 repositories (2 canonical rep types).");
  });

  test("unknown tool — returns null (server falls back)", () => {
    expect(summarizeReadToolPayload("unknown_tool", {})).toBeNull();
    expect(summarizeReadToolPayload("gkill_add_kmemo", {})).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// summarizeToolError (payload.mjs の実物)
// ---------------------------------------------------------------------------
describe("summarizeToolError", () => {
  test("formats error with tool name", () => {
    expect(summarizeToolError("gkill_get_kyous", "Connection refused", null)).toContain("gkill_get_kyous");
    expect(summarizeToolError("gkill_get_kyous", "Connection refused", null)).toContain("Connection refused");
  });

  test("handles empty tool name", () => {
    const result = summarizeToolError("", "Timeout", null);
    expect(result).toContain("Timeout");
  });
});
