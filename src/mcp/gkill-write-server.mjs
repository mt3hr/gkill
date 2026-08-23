#!/usr/bin/env node

import { readFileSync } from "node:fs";
import { dirname as _dirname, resolve as _resolvePath } from "node:path";
import process from "node:process";
import { fileURLToPath as _fileURLToPath } from "node:url";

import {
  PLUGIN_TOOLS,
  handlePluginToolCall,
  isPluginToolName,
} from "./lib/plugin-tools.mjs";
import { OAuthServer } from "./lib/oauth-server.mjs";
import { McpAccessLog, parseMcpLogLevel } from "./lib/access-log.mjs";
import { StdioTransport } from "./lib/stdio-transport.mjs";
import { McpServerBase } from "./lib/mcp-server-base.mjs";
import { GkillClient as GkillWriteClient } from "./lib/gkill-client.mjs";
import { READ_TOOLS } from "./lib/read-tools.mjs";
import { handleReadToolCall } from "./lib/read-handlers.mjs";
import { WRITE_TOOLS } from "./lib/write-tools.mjs";
import { handleWriteToolCall } from "./lib/write-handlers.mjs";
import { HttpTransport } from "./lib/http-transport.mjs";

// 書き込みサーバにも載せる読み取りツール。
// 前3つは書き込みの前に rep名 / 板名 / タグ名を引くためのもの。
// gkill_get_kyou_history は gkill_delete_kyou / gkill_restore_kyou の相棒で、
// 「今なにを消したのか」「なにを戻そうとしているのか」を同じサーバから確かめられないと
// 削除の取り消しが当てずっぽうになるので、ここに載せる。
const WRITE_SERVER_READ_TOOL_NAMES = new Set([
  "gkill_get_all_rep_names",
  "gkill_get_mi_board_list",
  "gkill_get_all_tag_names",
  "gkill_get_kyou_history",
]);

const _thisFile = _fileURLToPath(import.meta.url);
const _thisDir = _dirname(_thisFile);
const _pkg = JSON.parse(readFileSync(_resolvePath(_thisDir, "../../package.json"), "utf8"));

const SERVER_NAME = "gkill-write-mcp";
const SERVER_VERSION = _pkg.version;

const WRITE_APP_NAME = "gkill_mcp_write";

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------

const TOOLS = [
  ...WRITE_TOOLS,
  // --- Read convenience tools ---
  // 書き込みの前に rep名 / 板名 / タグ名を引けるように、読み取りツールの一部だけを載せる。
  // 定義は読み取りサーバと同じものを使う（以前は description が食い違っていた）
  ...READ_TOOLS.filter((tool) => WRITE_SERVER_READ_TOOL_NAMES.has(tool.name)),
  ...PLUGIN_TOOLS,
];

// ---------------------------------------------------------------------------
// GkillWriteClient
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// McpWriteServer: transport-independent JSON-RPC handler
// ---------------------------------------------------------------------------

class McpWriteServer extends McpServerBase {
  constructor(client, accessLog = null) {
    super(client, accessLog, { serverName: SERVER_NAME, serverVersion: SERVER_VERSION, tools: TOOLS });
  }

  async handleToolCall(name, args, ctx = null) {
    const sid = ctx ? ctx.sessionId : this.currentSessionId;
    const userId = (ctx ? ctx.userId : this.currentUserId) || this.client.userId;

    if (isPluginToolName(name)) {
      return handlePluginToolCall(
        (pathname, body) => this.client.callApi(pathname, body, true, sid),
        name,
        args,
      );
    }

    // Read convenience tools（WRITE_SERVER_READ_TOOL_NAMES）は共有ディスパッチへ
    if (WRITE_SERVER_READ_TOOL_NAMES.has(name)) {
      return handleReadToolCall(
        { client: this.client, sid, isLocalTransport: this.isLocalTransport },
        name,
        args,
      );
    }

    return handleWriteToolCall(
      { client: this.client, sid, userId, appName: WRITE_APP_NAME },
      name,
      args,
    );
  }

}

// ---------------------------------------------------------------------------
// StdioTransport
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// HttpTransport
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

const _isDirectRun =
  typeof process !== "undefined" &&
  process.argv[1] &&
  _resolvePath(process.argv[1]) === _thisFile;

if (_isDirectRun) {
  const client = new GkillWriteClient();

  const transport = (process.env.MCP_TRANSPORT || "stdio").toLowerCase();
  const gkillHome = process.env.GKILL_HOME || _resolvePath(process.env.HOME || process.env.USERPROFILE || ".", "gkill");
  const mcpLogLevel = parseMcpLogLevel(process.env.MCP_LOG);
  const accessLog = new McpAccessLog(
    _resolvePath(gkillHome, "logs", "gkill_mcp_write_access.log"),
    mcpLogLevel,
    "gkill-write-server.mjs",
  );

  const server = new McpWriteServer(client, accessLog);

  if (transport === "http") {
    const port = parseInt(process.env.MCP_PORT || "8809", 10);
    const issuer = process.env.MCP_OAUTH_ISSUER || `http://localhost:${port}`;
    const persistPath = _resolvePath(gkillHome, "configs", "mcp_oauth_write_state.json");
    const oauthServer = new OAuthServer({
      issuer,
      persistPath,
      authenticateUser: async (userId, passwordSha256) => {
        try {
          const response = await client.post("/api/login", {
            user_id: userId,
            password_sha256: passwordSha256,
            locale_name: client.defaultLocale,
          });
          if (client.hasErrors(response) || !response.session_id) {
            accessLog.warn("auth_failure", { user_id: userId });
            return null;
          }
          accessLog.info("auth_success", { user_id: userId });
          return { sessionId: response.session_id };
        } catch {
          accessLog.warn("auth_failure", { user_id: userId });
          return null;
        }
      },
    });
    accessLog.info("server_start", {
      transport, log_level: mcpLogLevel, port,
    });
    new HttpTransport(server, port, oauthServer, { scope: "gkill:write", enableFileLinks: false }).start();
  } else {
    server.currentUserId = client.userId || null;
    accessLog.info("server_start", {
      transport, log_level: mcpLogLevel,
    });
    new StdioTransport(server).start();
  }
}

export { GkillWriteClient, McpWriteServer, OAuthServer };
