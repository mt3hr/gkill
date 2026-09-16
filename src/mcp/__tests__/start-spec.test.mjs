/**
 * Tests for the START_SPEC declarations of the three MCP server entry scripts.
 *
 * scope の正本は各エントリスクリプトの START_SPEC 1箇所で、bootstrap がそこから
 * OAuth の metadata・認可既定値・トークン発行・受理検証まで全部を生成する
 * （lib/mcp-server-bootstrap.mjs）。ReadWrite サーバが gkill:read を広告していた事故
 * （2026-08-30 修正）は、テストが自前の値で OAuthServer を組み立てていて
 * エントリスクリプトの宣言値を誰も読んでいなかったために出荷まで漏れた。
 * ここでは宣言値そのものを固定する。
 */

import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, test, expect } from "vitest";
import { START_SPEC as READ_SPEC } from "../gkill-read-server.mjs";
import { START_SPEC as WRITE_SPEC } from "../gkill-write-server.mjs";
import { START_SPEC as READWRITE_SPEC } from "../gkill-readwrite-server.mjs";
import { startInfo } from "../lib/mcp-server-bootstrap.mjs";

const SPECS = [
  ["read", READ_SPEC, { scope: "gkill:read", defaultPort: 8808, enableFileLinks: true }],
  ["write", WRITE_SPEC, { scope: "gkill:write", defaultPort: 8809, enableFileLinks: false }],
  ["readwrite", READWRITE_SPEC, { scope: "gkill:readwrite", defaultPort: 8810, enableFileLinks: true }],
];

describe("server START_SPEC declarations", () => {
  test.each(SPECS)("%s server declares its own scope / port / file-link policy", (_name, spec, expected) => {
    expect(spec.scope).toBe(expected.scope);
    expect(spec.defaultPort).toBe(expected.defaultPort);
    // 書き込み専用サーバだけ file-link URL を発行しない、という宣言もここが正本。
    expect(spec.enableFileLinks).toBe(expected.enableFileLinks);
    // 起動後にどこかが書き換える形へ戻さない（宣言は不変）。
    expect(Object.isFrozen(spec)).toBe(true);
  });

  test("three servers never share a scope, port, script name, log file, or OAuth state file", () => {
    for (const key of ["scope", "defaultPort", "scriptName", "logFileName", "oauthStateFileName"]) {
      const values = SPECS.map(([, spec]) => spec[key]);
      expect(new Set(values).size, `duplicated ${key}`).toBe(values.length);
    }
  });

  test("bootstrap forwards spec.scope to OAuthServer (single source of scope)", () => {
    // startMcpServer はポートを開きログファイルを作るため直接は呼ばない。
    // かわりに「OAuthServer へ spec.scope を渡す」行の存在をソースで固定する
    // —— この行が消えると3サーバ全部の scope が宣言と無関係になり、
    // 宣言値のテスト（上）が緑のまま実挙動だけずれる。
    const bootstrapSource = readFileSync(
      resolve(dirname(fileURLToPath(import.meta.url)), "../lib/mcp-server-bootstrap.mjs"),
      "utf8",
    );
    expect(bootstrapSource).toContain("scope: spec.scope");
  });
});

// server_start ログの世代情報。「ソースは直っているのに AI からは古い」の切り分けは
// ここと gkill_status の schema_revision を見比べて行う（プロセスが古いのか、クライアントの
// 一覧が古いのか）。schema_revision 自体は status-tool.test.mjs が固定する。
describe("server_start log carries the tool-list generation", () => {
  test("startInfo reports pid, schema_revision and tool_count of the running server", () => {
    const info = startInfo({ schemaRevision: "abc123def456", tools: [{}, {}, {}] });
    expect(info).toEqual({ pid: process.pid, schema_revision: "abc123def456", tool_count: 3 });
  });

  test("both transports log server_start with startInfo (stdio and http)", () => {
    const bootstrapSource = readFileSync(
      resolve(dirname(fileURLToPath(import.meta.url)), "../lib/mcp-server-bootstrap.mjs"),
      "utf8",
    );
    const lines = bootstrapSource.split("\n").filter((line) => line.includes('"server_start"'));
    expect(lines.length, "stdio と http の2箇所で server_start を出す").toBe(2);
    for (const line of lines) {
      expect(line).toContain("...startInfo(server)");
    }
  });
});
