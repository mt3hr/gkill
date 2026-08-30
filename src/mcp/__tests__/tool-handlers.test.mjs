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
import { summarizeToolError, entityNotFoundMessage } from "../lib/payload.mjs";
import { WRITE_TOOLS } from "../lib/write-tools.mjs";
import { WRITE_SERVER_READ_TOOL_NAMES } from "../gkill-write-server.mjs";
import { CROSS_SERVER_TOOL_MENTIONS } from "../lib/constants.mjs";

// ---------------------------------------------------------------------------
// Tool definition presence
// ---------------------------------------------------------------------------
describe("Tool definitions", () => {
  test("read server exposes 10 tools (9 read + 1 plugin)", () => {
    expect(READ_TOOLS).toHaveLength(9);
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

  test("gkill_get_kyous tells AI to inspect repository warnings independently of partial", () => {
    const description = READ_TOOLS.find((tool) => tool.name === "gkill_get_kyous")?.description || "";
    expect(description).toContain("even when partial is false");
    expect(description).toContain("failed to load");
    expect(description).toContain("do not put a repository named by that warning back into query.reps");
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
    expect(result).toBe("Fetched 1 repositories, 2 canonical rep types.");
  });

  // fields で rep_infos を外した呼び出しに「Fetched 0 repositories」と言うと、
  // 自分で外しただけなのに「リポジトリが0件」と読める（2026-08-25 の実利用レビュー）。
  test("gkill_get_rep_infos — says a field was omitted rather than reporting zero", () => {
    const result = summarizeReadToolPayload("gkill_get_rep_infos", {
      canonical_rep_types: ["kmemo", "kc"],
      plugins: [],
      attached_data_reps: [],
    });
    expect(result).toContain("omitted by fields");
    expect(result).not.toContain("0 repositories");
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

// ---------------------------------------------------------------------------
// 説明文が名指しするツールは、そのサーバに実在すること
// ---------------------------------------------------------------------------
//
// 読み取り専用サーバの説明が gkill_restore_kyou を案内し、書き込み専用サーバの
// 「見つからない」が gkill_get_application_config / gkill_get_kyous を案内していた。
// どちらも「案内されたツールがそのサーバに無い」で、AI は存在しないツールを探す。
// 同じ穴なので、説明文とエラーメッセージをまとめて機械検査する。
describe("tool names in descriptions exist on the same server", () => {
  const WRITE_SERVER_READ_TOOLS = READ_TOOLS.filter((tool) => WRITE_SERVER_READ_TOOL_NAMES.has(tool.name));
  // 書き込み専用サーバは read を数本に絞る設計なので、3サーバで共有している
  // 説明文が検索の口(gkill_get_kyous)に触れるのは避けられない。ここで検査すると
  // 説明を全部書き換えることになり、read / readwrite 側の案内が劣化する。
  // 代わりに、実害の出たランタイム文言（「見つからない」）を下の別テストで固定する。
  const SERVERS = [
    ["read", [...READ_TOOLS, ...PLUGIN_TOOLS]],
    ["readwrite", [...READ_TOOLS, ...WRITE_TOOLS, ...PLUGIN_TOOLS]],
  ];
  const ALL_TOOL_NAMES = new Set([...READ_TOOLS, ...WRITE_TOOLS, ...PLUGIN_TOOLS].map((tool) => tool.name));

  // 説明文の中の gkill_* を全部拾う。inputSchema の中の description も見る
  // （restore の案内は find-query-schema.mjs 側、つまりスキーマの奥にあった）。
  function collectMentions(node, found = new Set()) {
    if (typeof node === "string") {
      for (const match of node.matchAll(/gkill_[a-z_]+/g)) {
        found.add(match[0]);
      }
      return found;
    }
    if (Array.isArray(node)) {
      for (const child of node) collectMentions(child, found);
      return found;
    }
    if (node && typeof node === "object") {
      for (const child of Object.values(node)) collectMentions(child, found);
    }
    return found;
  }

  test.each(SERVERS)("%s server", (_label, tools) => {
    const available = new Set(tools.map((tool) => tool.name));
    const missing = [];
    for (const tool of tools) {
      for (const mentioned of collectMentions(tool)) {
        // gkill_kftl / gkill_mcp_readwrite のような create_app 値は素通しする。
        // 判定したいのは「どこかのサーバに実在するツール名なのに、ここには無い」だけ。
        if (!ALL_TOOL_NAMES.has(mentioned)) continue;
        if (CROSS_SERVER_TOOL_MENTIONS.has(mentioned)) continue;
        if (!available.has(mentioned)) missing.push(`${tool.name} -> ${mentioned}`);
      }
    }
    expect(missing).toEqual([]);
  });

  // 「見つからない」はランタイムの文言なので、スキーマ検査では拾えない。
  // 書き込み専用サーバはここで案内されるツールを持っている必要がある。
  test("entityNotFoundMessage names only tools the write-only server has", () => {
    const available = new Set([
      ...WRITE_TOOLS.map((tool) => tool.name),
      ...WRITE_SERVER_READ_TOOLS.map((tool) => tool.name),
      ...PLUGIN_TOOLS.map((tool) => tool.name),
    ]);
    for (const mentioned of collectMentions(entityNotFoundMessage("x1", "kmemo"))) {
      if (!ALL_TOOL_NAMES.has(mentioned)) continue;
      expect(available.has(mentioned)).toBe(true);
    }
  });
});

// ---------------------------------------------------------------------------
// read と readwrite は同じ read ツールを配ること
// ---------------------------------------------------------------------------
//
// 「readwrite にだけ user_id が無い」「group_by が違う」という報告が繰り返し来る。
// 実体は毎回「古いプロセスが昨日の定義を配っていた」だったが、コード側で
// 保証しているのは「同じ配列を spread している」という書き方だけで、テストは
// 本数と名前しか見ていなかった。スキーマそのものの同一性を固定する。
describe("read and readwrite serve identical read tools", () => {
  test("same names and same inputSchema", () => {
    const readTools = [...READ_TOOLS, ...PLUGIN_TOOLS];
    const readwriteTools = [...READ_TOOLS, ...WRITE_TOOLS, ...PLUGIN_TOOLS];
    const readwriteByName = new Map(readwriteTools.map((tool) => [tool.name, tool]));

    for (const tool of readTools) {
      const counterpart = readwriteByName.get(tool.name);
      expect(counterpart).toBeDefined();
      expect(counterpart.description).toBe(tool.description);
      expect(counterpart.inputSchema).toEqual(tool.inputSchema);
    }
  });
});
