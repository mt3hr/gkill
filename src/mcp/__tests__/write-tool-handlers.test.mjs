/**
 * Tests for MCP write tool definitions and shared helpers.
 *
 * C1 から summarize / ディスパッチ / 削除の対応表は lib/write-handlers.mjs に一本化され
 * export されたので、ここは再実装のミラーではなく**実物を直接 import して**検証する
 * （ミラーは実装が変わっても緑のまま古び、二重管理の温床だった。
 *  実際このファイルと readwrite-tool-handlers.test.mjs は
 *  「統合サーバは29ツール」「read は8件」という実測と違う値を
 *  自分のハードコード配列に対して検証し続けていた）。
 *
 * lib/read-tools.mjs 側の同じ形は tool-handlers.test.mjs にある。
 */

import { describe, test, expect } from "vitest";

import { WRITE_TOOLS } from "../lib/write-tools.mjs";
import { isWriteToolName, summarizeWriteToolPayload } from "../lib/write-handlers.mjs";
import {
  ENTITY_TARGETS,
  ENTITY_AND_PROJECTION_DATA_TYPE_VALUES,
  PROJECTION_TO_ENTITY_DATA_TYPE,
  toEntityDataType,
} from "../lib/constants.mjs";
import { DELETE_DATA_TYPES } from "../lib/write-normalization.mjs";
import { summarizeToolError } from "../lib/payload.mjs";

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------
describe("Tool definitions", () => {
  test("write server exposes 21 write tools", () => {
    expect(WRITE_TOOLS).toHaveLength(21);
  });

  test("write tool names are the current set", () => {
    expect(WRITE_TOOLS.map((tool) => tool.name)).toEqual([
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
      "gkill_restore_kyou",
    ]);
  });

  test("isWriteToolName matches the definitions", () => {
    for (const tool of WRITE_TOOLS) {
      expect(isWriteToolName(tool.name)).toBe(true);
    }
    expect(isWriteToolName("gkill_get_kyous")).toBe(false);
    expect(isWriteToolName("gkill_get_plugin_list")).toBe(false);
  });

  test("every tool has an object inputSchema with additionalProperties: false", () => {
    // 未知キーを黙って捨てないための不変条件。read 側は tool-handlers.test.mjs が同じ検査をしている
    for (const tool of WRITE_TOOLS) {
      expect(tool.inputSchema.type).toBe("object");
      expect(tool.inputSchema.additionalProperties).toBe(false);
    }
  });

  test("update tools require only id (patch semantics)", () => {
    // 正規化層は id 以外すべて optional の patch なのに、スキーマの required だけが
    // title などを強制していた。スキーマに忠実な AI は
    // (a) title を取りに1往復増やすか (b) 推測して送って既存値を静かに上書きする。
    // update_mi の説明文の例 {id, is_checked:true} も自分の required に違反していた
    for (const tool of WRITE_TOOLS) {
      if (!tool.name.startsWith("gkill_update_")) continue;
      expect(tool.inputSchema.required).toEqual(["id"]);
    }
  });

  test("no description advertises the deprecated include_id argument", () => {
    // ADR-0053 で ID は常時付与になった。案内が残っていると AI が必須引数だと学習する
    for (const tool of WRITE_TOOLS) {
      expect(tool.description).not.toContain("include_id");
      expect(JSON.stringify(tool.inputSchema)).not.toContain("include_id");
    }
  });

  test("every tool has a non-empty description", () => {
    for (const tool of WRITE_TOOLS) {
      expect(typeof tool.description).toBe("string");
      expect(tool.description.length).toBeGreaterThan(0);
    }
  });
});

// ---------------------------------------------------------------------------
// Delete data_type: 語彙が constants.mjs の1箇所から派生していること
// ---------------------------------------------------------------------------
describe("delete_kyou data_type vocabulary", () => {
  test("schema enum accepts projections while DELETE_DATA_TYPES stays the folded vocabulary", () => {
    // 語彙が食い違うと「スキーマは受理するのにディスパッチで落ちる」
    // （あるいはその逆）になる。畳んだ後の正本は constants.mjs の ENTITY_TARGETS。
    // ただし入口は toEntityDataType を通すので、スキーマ側は射影名も許さないと
    // 「応答の data_type をそのまま次のツールへ渡せる」という説明と食い違う
    // （enum を畳んだ後の語彙だけにすると、送信前に検証するクライアントが
    // mi_start をサーバへ届く前に弾く。2026-08-25 の実利用レビュー）。
    const deleteTool = WRITE_TOOLS.find((tool) => tool.name === "gkill_delete_kyou");
    const canonical = Object.keys(ENTITY_TARGETS).sort();
    const accepted = [...ENTITY_AND_PROJECTION_DATA_TYPE_VALUES].sort();

    expect([...deleteTool.inputSchema.properties.data_type.enum].sort()).toEqual(accepted);
    // 畳んだ後の語彙は従来どおり ENTITY_TARGETS と一致する
    expect([...DELETE_DATA_TYPES].sort()).toEqual(canonical);
    // スキーマが受理する射影名は、すべて畳んだ先が語彙にあること
    for (const projection of PROJECTION_TO_ENTITY_DATA_TYPE.keys()) {
      expect(DELETE_DATA_TYPES.has(toEntityDataType(projection))).toBe(true);
    }
  });

  test("every delete target has both a get and an update endpoint", () => {
    for (const target of Object.values(ENTITY_TARGETS)) {
      expect(target.getEndpoint).toMatch(/^\/api\//);
      expect(target.historiesKey).toMatch(/_histories$/);
      expect(target.updateEndpoint).toMatch(/^\/api\/update_/);
      expect(target.responseKey).toMatch(/^updated_/);
      expect(typeof target.requestKey).toBe("string");
    }
  });
});

// ---------------------------------------------------------------------------
// summarizeWriteToolPayload
// ---------------------------------------------------------------------------
describe("summarizeWriteToolPayload", () => {
  test("add tools report the created id", () => {
    expect(summarizeWriteToolPayload("gkill_add_kmemo", { added_kmemo: { id: "k1" } }))
      .toBe("Created kmemo: k1");
    expect(summarizeWriteToolPayload("gkill_add_mi", { added_mi: { id: "m1" } }))
      .toBe("Created mi: m1");
    expect(summarizeWriteToolPayload("gkill_add_tag", { added_tag: { id: "t1" } }))
      .toBe("Added tag: t1");
    expect(summarizeWriteToolPayload("gkill_add_text", { added_text: { id: "x1" } }))
      .toBe("Added text: x1");
  });

  test("add tools fall back to unknown when the entity is missing", () => {
    expect(summarizeWriteToolPayload("gkill_add_kmemo", {})).toBe("Created kmemo: unknown");
  });

  test("update tools report the updated id", () => {
    expect(summarizeWriteToolPayload("gkill_update_kmemo", { updated_kmemo: { id: "k1" } }))
      .toBe("Updated kmemo: k1");
    expect(summarizeWriteToolPayload("gkill_update_mi", { updated_mi: { id: "m1" } }))
      .toBe("Updated mi: m1");
  });

  test("gkill_submit_kftl reports what was written, not how many messages came back", () => {
    // 「N messages」はサーバの定型文の本数でしかなく、何が作られたかを伝えていなかった
    expect(
      summarizeWriteToolPayload("gkill_submit_kftl", {
        messages: [{}],
        created: [
          { id: "a", data_type: "kmemo" },
          { id: "b", data_type: "lantana" },
        ],
      }),
    ).toBe("KFTL submitted: wrote 2 record(s) — kmemo, lantana.");
  });

  test("gkill_submit_kftl groups repeats and marks updates", () => {
    // 打刻の終了は新規作成ではなく既存レコードの更新
    expect(
      summarizeWriteToolPayload("gkill_submit_kftl", {
        created: [
          { id: "a", data_type: "kmemo" },
          { id: "b", data_type: "kmemo" },
          { id: "c", data_type: "timeis", updated: true },
        ],
      }),
    ).toBe("KFTL submitted: wrote 3 record(s) — kmemo x2, timeis (updated).");
  });

  test("gkill_submit_kftl says so when nothing was written", () => {
    // 空行だけのテキストと、冪等キーで畳まれた再送はどちらも何も書かない
    expect(summarizeWriteToolPayload("gkill_submit_kftl", { messages: [{}] }))
      .toBe("KFTL submitted: nothing was written (blank lines and idempotent replays write nothing).");
  });

  test("gkill_delete_kyou lists the updated_* keys", () => {
    expect(summarizeWriteToolPayload("gkill_delete_kyou", { updated_kmemo: {}, updated_kyou: {} }))
      .toBe("Deleted (soft): updated_kmemo, updated_kyou");
    expect(summarizeWriteToolPayload("gkill_delete_kyou", {}))
      .toBe("Deleted (soft): completed");
  });

  test("returns null for tools it does not own (server falls through to read/plugin)", () => {
    // null を返さないと、基底の3段フォールバックが read の要約を上書きしてしまう
    expect(summarizeWriteToolPayload("gkill_get_kyous", {})).toBeNull();
    expect(summarizeWriteToolPayload("gkill_get_plugin_list", {})).toBeNull();
    expect(summarizeWriteToolPayload("unknown_tool", {})).toBeNull();
  });

  test("covers every tool in WRITE_TOOLS", () => {
    // ツールを足して要約の case を忘れると "Tool call completed." に落ちて静かに劣化する
    for (const tool of WRITE_TOOLS) {
      expect(summarizeWriteToolPayload(tool.name, {})).not.toBeNull();
    }
  });
});

// ---------------------------------------------------------------------------
// summarizeToolError (payload.mjs の実物)
// ---------------------------------------------------------------------------
describe("summarizeToolError", () => {
  test("includes tool name and error", () => {
    const result = summarizeToolError("gkill_add_kmemo", "Connection refused", null);
    expect(result).toContain("gkill_add_kmemo");
    expect(result).toContain("Connection refused");
  });

  test("includes field when present", () => {
    const result = summarizeToolError("gkill_add_kmemo", "Invalid", { field: "content" });
    expect(result).toContain("content");
  });

  test("handles empty tool name", () => {
    expect(summarizeToolError("", "Timeout", null)).toContain("Timeout");
  });
});
