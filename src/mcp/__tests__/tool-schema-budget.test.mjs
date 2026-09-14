/**
 * tools/list のバイト量の予算（2026-09-14 レビュー P0）。
 *
 * 予算ファイル src/mcp/tool-schema-budget.json は「現状の実測」で、増やすときは
 * `npm run mcp:schema-budget -- --update` で明示的に更新する。減ったときも追随させる。
 * 判定と文言の正本は src/mcp/tool-schema-budget.mjs（スクリプトと共用）。
 */

import { describe, test, expect } from "vitest";

import {
  SERVER_KINDS,
  TOOL_SCHEMA_BUDGET_SLACK_BYTES,
  compareToolSchemaBudget,
  describeBudgetRow,
  measureToolSchemaBytes,
  readToolSchemaBudget,
} from "../tool-schema-budget.mjs";

describe("tools/list byte budget", () => {
  const current = measureToolSchemaBytes();
  const budget = readToolSchemaBudget();
  const rows = compareToolSchemaBudget(current, budget);

  test.each(SERVER_KINDS)("%s server stays within its recorded budget", (kind) => {
    const row = rows.find((candidate) => candidate.kind === kind);
    expect(row.verdict, describeBudgetRow(row)).toBe("ok");
  });

  test("measurement is deterministic (the schema_revision mark is fixed-length)", () => {
    expect(measureToolSchemaBytes()).toEqual(current);
  });
});

describe("compareToolSchemaBudget", () => {
  test("classifies over / under / ok / missing", () => {
    const budget = { read: 1000, write: 1000, readwrite: 1000 };
    const rows = compareToolSchemaBudget(
      { read: 1001, write: 1000 - TOOL_SCHEMA_BUDGET_SLACK_BYTES - 1, readwrite: 1000 - TOOL_SCHEMA_BUDGET_SLACK_BYTES },
      budget,
    );
    expect(rows.map((row) => row.verdict)).toEqual(["over", "under", "ok"]);
    expect(describeBudgetRow(rows[0])).toMatch(/\+1 over the budget of 1000.*--update/s);
    expect(describeBudgetRow(rows[1])).toMatch(/under the budget.*--update/s);
    expect(compareToolSchemaBudget({ read: 1, write: 1, readwrite: 1 }, {})[0].verdict).toBe("missing");
    expect(compareToolSchemaBudget({ read: 1, write: 1, readwrite: 1 }, null)[0].verdict).toBe("missing");
  });
});
