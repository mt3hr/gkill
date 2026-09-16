// Read before editing: .claude/skills/gkill-mcp/SKILL.md (invariants for this area)
// 3つのMCPサーバの起動ブロック。
//
// read / write / readwrite の3ファイルは、起動処理が**7つの値以外まったく同じ**だった
// （client クラス / ログ名 / スクリプト名 / 既定ポート / OAuth 状態ファイル名 / scope /
// file-link の可否）。同じ手順が3形態あると、片方だけ直して静かにずれる。
// ここが正本で、3ファイルは表の1行だけを持つ（ADR-0611）。
//
// ここへ足すもの: 全サーバで同じでなければならない起動時の手順。
// ここへ足さないもの: サーバごとに違う「どのツールを持つか」（それは McpServerBase の options）。

import { readFileSync } from "node:fs";
import { dirname as _dirname, resolve as _resolvePath } from "node:path";
import process from "node:process";
import { fileURLToPath as _fileURLToPath } from "node:url";

import { OAuthServer } from "./oauth-server.mjs";
import { McpAccessLog, parseMcpLogLevel } from "./access-log.mjs";
import { StdioTransport } from "./stdio-transport.mjs";
import { HttpTransport } from "./http-transport.mjs";
import { makeOAuthAuthenticateUser } from "./mcp-server-base.mjs";

const _libDir = _dirname(_fileURLToPath(import.meta.url));

// readPackageVersion はサーバ版として名乗る値をリポジトリ直下の package.json から読む。
// 3ファイルが各自 readFileSync していたので、相対パスの起点が1つずれるだけで
// 片方のサーバだけ版が読めなくなる形だった。
export function readPackageVersion() {
  return JSON.parse(readFileSync(_resolvePath(_libDir, "../../../package.json"), "utf8")).version;
}

// isDirectRun は「このファイルが node の引数として直接起動されたか」を返す。
// テストで import しただけのときにトランスポートを立ち上げないためのガード。
export function isDirectRun(importMetaUrl) {
  if (typeof process === "undefined" || !process.argv[1]) {
    return false;
  }
  return _resolvePath(process.argv[1]) === _fileURLToPath(importMetaUrl);
}

// startInfo は server_start ログに載せる「このプロセスがどの世代のツール一覧を配るか」。
// 「ソースは直っているのに AI からは古い」の切り分けは、まずここと gkill_status の
// schema_revision を見比べる（プロセスが古いのか、クライアントの一覧が古いのか）。
export function startInfo(server) {
  return { pid: process.pid, schema_revision: server.schemaRevision, tool_count: server.tools.length };
}

// startMcpServer はアクセスログを開き、サーバを作り、MCP_TRANSPORT に従って
// stdio か HTTP のトランスポートを起動する。
//
// spec:
//   ServerClass         … McpServerBase の派生
//   client              … GkillClient
//   scriptName          … アクセスログに載せる発生源のファイル名
//   logFileName         … $GKILL_HOME/logs/ 配下のファイル名
//   oauthStateFileName  … $GKILL_HOME/configs/ 配下のファイル名（HTTPのときだけ使う）
//   defaultPort         … MCP_PORT 未指定時のポート
//   scope               … OAuth のスコープ名
//   enableFileLinks     … file-link URL を発行するか（書き込み専用サーバは false）
export function startMcpServer(spec) {
  const transport = (process.env.MCP_TRANSPORT || "stdio").toLowerCase();
  const gkillHome =
    process.env.GKILL_HOME || _resolvePath(process.env.HOME || process.env.USERPROFILE || ".", "gkill");
  const mcpLogLevel = parseMcpLogLevel(process.env.MCP_LOG);
  const accessLog = new McpAccessLog(
    _resolvePath(gkillHome, "logs", spec.logFileName),
    mcpLogLevel,
    spec.scriptName,
    // ERROR は gkill_mcp_error.log にも複製する。障害を探す運用者は
    // "*_access.log" を開かないので、そこにしか無いと見つからない。
    _resolvePath(gkillHome, "logs", "gkill_mcp_error.log"),
  );

  const server = new spec.ServerClass(spec.client, accessLog);

  if (transport === "http") {
    const port = parseInt(process.env.MCP_PORT || String(spec.defaultPort), 10);
    const issuer = process.env.MCP_OAUTH_ISSUER || `http://localhost:${port}`;
    const persistPath = _resolvePath(gkillHome, "configs", spec.oauthStateFileName);
    const oauthServer = new OAuthServer({
      issuer,
      // scope の正本はここ1箇所。metadata・認可既定値・トークン発行・Bearer 受理の
      // 全てがこの値から生成される (transport は oauthServer.scope を読む)。
      scope: spec.scope,
      persistPath,
      accessLog,
      authenticateUser: makeOAuthAuthenticateUser(spec.client, accessLog),
    });
    accessLog.info("server_start", { transport, log_level: mcpLogLevel, port, ...startInfo(server) });
    new HttpTransport(server, port, oauthServer, {
      enableFileLinks: spec.enableFileLinks,
    }).start();
  } else {
    server.currentUserId = spec.client.userId || null;
    accessLog.info("server_start", { transport, log_level: mcpLogLevel, ...startInfo(server) });
    new StdioTransport(server).start();
  }

  return server;
}
