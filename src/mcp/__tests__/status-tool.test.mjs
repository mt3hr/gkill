/**
 * lib/status-tool.mjs — ツール一覧の世代（schema_revision）と gkill_status への焼き込み。
 *
 * - 同じ一覧なら同じ値、1文字でも違えば別の値（手書きの版番号ではない）
 * - gkill_status 自身は計算対象に入らない（description に自分の値を焼き込むため）
 * - 焼き込みは静的な配列を書き換えず、gkill_status だけ差し替えた新しい配列を返す
 * - 焼き込みは冪等（二重に付かない）
 * - 3サーバはツール集合が違うので世代も違う
 */

import { describe, test, expect } from "vitest";

import {
  STATUS_TOOL_NAME,
  SCHEMA_REVISION_LENGTH,
  SCHEMA_REVISION_MARK_REGEX,
  computeSchemaRevision,
  stampSchemaRevision,
  stripSchemaRevisionMark,
} from "../lib/status-tool.mjs";
import { READ_TOOLS } from "../lib/read-tools.mjs";
import { WRITE_TOOLS } from "../lib/write-tools.mjs";
import { PLUGIN_TOOLS } from "../lib/plugin-tools.mjs";

const SAMPLE_TOOLS = [
  { name: STATUS_TOOL_NAME, description: "status", inputSchema: { type: "object", properties: {} } },
  { name: "gkill_x", description: "x", inputSchema: { type: "object", properties: { a: { type: "string" } } } },
];

describe("computeSchemaRevision", () => {
  test("is deterministic and 12 hex chars", () => {
    const first = computeSchemaRevision(SAMPLE_TOOLS);
    expect(first).toMatch(new RegExp(`^[0-9a-f]{${SCHEMA_REVISION_LENGTH}}$`));
    expect(computeSchemaRevision(SAMPLE_TOOLS)).toBe(first);
  });

  test("changes when any tool's description or schema changes", () => {
    const base = computeSchemaRevision(SAMPLE_TOOLS);
    const renamedField = structuredClone(SAMPLE_TOOLS);
    renamedField[1].inputSchema.properties = { b: { type: "string" } };
    expect(computeSchemaRevision(renamedField)).not.toBe(base);
    const editedDescription = structuredClone(SAMPLE_TOOLS);
    editedDescription[1].description = "x!";
    expect(computeSchemaRevision(editedDescription)).not.toBe(base);
  });

  test("ignores gkill_status itself, so stamping does not move the value", () => {
    const before = computeSchemaRevision(SAMPLE_TOOLS);
    const stamped = stampSchemaRevision(SAMPLE_TOOLS, before);
    expect(computeSchemaRevision(stamped)).toBe(before);
    // gkill_status の説明文だけ変えても世代は動かない（自己参照を避けるため）
    const statusEdited = structuredClone(SAMPLE_TOOLS);
    statusEdited[0].description = "another status text";
    expect(computeSchemaRevision(statusEdited)).toBe(before);
  });

  test("differs across the three real servers because their tool sets differ", () => {
    const read = computeSchemaRevision([...READ_TOOLS, ...PLUGIN_TOOLS]);
    const readwrite = computeSchemaRevision([...READ_TOOLS, ...WRITE_TOOLS, ...PLUGIN_TOOLS]);
    expect(read).not.toBe(readwrite);
  });
});

describe("stampSchemaRevision", () => {
  test("appends the mark to gkill_status only and leaves the source array untouched", () => {
    const revision = "0123456789ab";
    const stamped = stampSchemaRevision(SAMPLE_TOOLS, revision);
    expect(stamped).not.toBe(SAMPLE_TOOLS);
    expect(stamped[0].description).toBe(`status [schema_revision: ${revision}]`);
    expect(stamped[0].description).toMatch(SCHEMA_REVISION_MARK_REGEX);
    // 元の配列は書き換えない（read / write / readwrite が同じ READ_TOOLS を共有する）
    expect(SAMPLE_TOOLS[0].description).toBe("status");
    // gkill_status 以外は同じオブジェクト
    expect(stamped[1]).toBe(SAMPLE_TOOLS[1]);
  });

  test("is idempotent: re-stamping replaces the mark instead of appending a second one", () => {
    const once = stampSchemaRevision(SAMPLE_TOOLS, "0123456789ab");
    const twice = stampSchemaRevision(once, "ba9876543210");
    expect(twice[0].description).toBe("status [schema_revision: ba9876543210]");
    expect(stripSchemaRevisionMark(twice[0].description)).toBe("status");
  });

  test("the real gkill_status definition carries no mark until a server stamps it", () => {
    const status = READ_TOOLS.find((tool) => tool.name === STATUS_TOOL_NAME);
    expect(status).toBeDefined();
    expect(status.description).not.toMatch(SCHEMA_REVISION_MARK_REGEX);
    // 引数を取らない（引数を足すと古いスキーマの救済表の対象になり、
    // 「古さを確かめるツール自身が古いスキーマで壊れる」）
    expect(Object.keys(status.inputSchema.properties)).toEqual([]);
    expect(status.inputSchema.additionalProperties).toBe(false);
  });
});
