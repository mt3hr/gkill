// Read before editing: .claude/skills/gkill-mcp/SKILL.md (invariants for this area)
// tools/list のバイト量の予算。
//
// AI クライアントは接続時に tools/list を丸ごとコンテキストへ抱える。「昨日の記録を見せて」
// のような要求でも、全ツールの説明文とスキーマぶんのトークンを毎回払う。
// 事故対策を説明文へ書き足すたびに一覧は太り（2026-09-14 時点で readwrite が約 94KB）、
// 個々の追記は正しいのに合計だけが誰にも見えていなかった（2026-09-14 のレビュー P0）。
//
// 予算は「現状の実測」で、閾値を頭で決めない。増やすときは `npm run mcp:schema-budget -- --update`
// で予算ファイルを更新し、コミットメッセージに理由を書く。減ったときも追随させる
// （緩んだ予算が残ると、次の増加を吸収して検知が効かなくなる）。
// 計測はサーバが実際に配る tools（schema_revision 焼き込み後。印は固定長なので決定的）。
//
// このファイルはサーバ3本を import するので lib/ には置かない（lib はサーバに依存しない）。

import { readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { McpServer as ReadServer } from "./gkill-read-server.mjs";
import { McpWriteServer } from "./gkill-write-server.mjs";
import { McpServer as ReadWriteServer } from "./gkill-readwrite-server.mjs";

const here = dirname(fileURLToPath(import.meta.url));

// 予算ファイル。追跡する（CI がこの値と実測を比べる）。
export const TOOL_SCHEMA_BUDGET_PATH = resolve(here, "tool-schema-budget.json");

// 予算を下回ってよい幅。これより小さくなったら予算のほうを下げさせる。
export const TOOL_SCHEMA_BUDGET_SLACK_BYTES = 1024;

export const SERVER_KINDS = ["read", "write", "readwrite"];

// measureToolSchemaBytes は3サーバの tools/list の JSON バイト数を返す。
// コンストラクタは client を保持するだけなので null でよい（起動はしない）。
export function measureToolSchemaBytes() {
  const servers = {
    read: new ReadServer(null),
    write: new McpWriteServer(null),
    readwrite: new ReadWriteServer(null),
  };
  const out = {};
  for (const kind of SERVER_KINDS) {
    out[kind] = Buffer.byteLength(JSON.stringify(servers[kind].tools), "utf8");
  }
  return out;
}

export function readToolSchemaBudget(path = TOOL_SCHEMA_BUDGET_PATH) {
  return JSON.parse(readFileSync(path, "utf8"));
}

export function writeToolSchemaBudget(budget, path = TOOL_SCHEMA_BUDGET_PATH) {
  writeFileSync(path, `${JSON.stringify(budget, null, 2)}\n`);
}

/**
 * compareToolSchemaBudget は実測と予算を突き合わせる。
 *
 * @param {Record<string, number>} current measureToolSchemaBytes の値。
 * @param {Record<string, number>} budget 予算ファイルの値。
 * @returns {Array<{kind: string, current: number, budget: number|null, delta: number|null, verdict: "ok"|"over"|"under"|"missing"}>}
 */
export function compareToolSchemaBudget(current, budget) {
  return SERVER_KINDS.map((kind) => {
    const measured = current[kind];
    const allowed = typeof budget?.[kind] === "number" ? budget[kind] : null;
    if (allowed === null) {
      return { kind, current: measured, budget: null, delta: null, verdict: "missing" };
    }
    const delta = measured - allowed;
    let verdict = "ok";
    if (delta > 0) verdict = "over";
    else if (-delta > TOOL_SCHEMA_BUDGET_SLACK_BYTES) verdict = "under";
    return { kind, current: measured, budget: allowed, delta, verdict };
  });
}

// describeBudgetRow は1行の判定を人が読む文にする（テストの失敗文とスクリプトの表示で共用）。
export function describeBudgetRow(row) {
  const sign = row.delta === null ? "" : row.delta >= 0 ? "+" : "";
  switch (row.verdict) {
    case "over":
      return (
        `${row.kind}: tools/list is ${row.current} bytes, ${sign}${row.delta} over the budget of ${row.budget}. ` +
        "Every byte here is paid by every AI session on every request. If the growth is intended, run " +
        "`npm run mcp:schema-budget -- --update` and say why in the commit message; otherwise trim the description."
      );
    case "under":
      return (
        `${row.kind}: tools/list is ${row.current} bytes, ${sign}${row.delta} under the budget of ${row.budget} ` +
        `(more than ${TOOL_SCHEMA_BUDGET_SLACK_BYTES} bytes of slack). Run \`npm run mcp:schema-budget -- --update\` ` +
        "so the budget follows the reduction instead of quietly absorbing the next increase."
      );
    case "missing":
      return `${row.kind}: no budget recorded (current ${row.current} bytes). Run \`npm run mcp:schema-budget -- --update\`.`;
    default:
      return `${row.kind}: ${row.current} bytes (budget ${row.budget}, ${sign}${row.delta}).`;
  }
}
