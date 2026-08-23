#!/usr/bin/env node

import { readFileSync } from "node:fs";
import { dirname as _dirname, resolve as _resolvePath } from "node:path";
import process from "node:process";
import { fileURLToPath as _fileURLToPath } from "node:url";

import { GkillApiError } from "./lib/errors.mjs";
import {
  PLUGIN_TOOLS,
  handlePluginToolCall,
  isPluginToolName,
} from "./lib/plugin-tools.mjs";
import { OAuthServer } from "./lib/oauth-server.mjs";
import { McpAccessLog, parseMcpLogLevel } from "./lib/access-log.mjs";
import { StdioTransport } from "./lib/stdio-transport.mjs";
import { McpServerBase } from "./lib/mcp-server-base.mjs";
import { GkillClient as GkillReadClient } from "./lib/gkill-client.mjs";
import { READ_TOOLS } from "./lib/read-tools.mjs";
import { isReadToolName, handleReadToolCall } from "./lib/read-handlers.mjs";
import { HttpTransport } from "./lib/http-transport.mjs";

const _thisFile = _fileURLToPath(import.meta.url);
const _thisDir = _dirname(_thisFile);
const _pkg = JSON.parse(readFileSync(_resolvePath(_thisDir, "../../package.json"), "utf8"));

const SERVER_NAME = "gkill-read-mcp";
const SERVER_VERSION = _pkg.version;

const TOOLS = [
  ...READ_TOOLS,
  ...PLUGIN_TOOLS,
];

// McpServer: transport-independent JSON-RPC handler.
// handleMessage() returns a response object (or null for notifications).
class McpServer extends McpServerBase {
  constructor(client, accessLog = null) {
    super(client, accessLog, { serverName: SERVER_NAME, serverVersion: SERVER_VERSION, tools: TOOLS });
  }

  async handleToolCall(name, args, ctx = null) {
    const sid = ctx ? ctx.sessionId : this.currentSessionId;
    if (isPluginToolName(name)) {
      return handlePluginToolCall(
        (pathname, body) => this.client.callApi(pathname, body, true, sid),
        name,
        args,
      );
    }
    if (isReadToolName(name)) {
      return handleReadToolCall(
        { client: this.client, sid, isLocalTransport: this.isLocalTransport },
        name,
        args,
      );
    }
    throw new GkillApiError(`Unknown tool: ${name}`);
  }

}

// Entry point — guarded so importing this module for tests does not start a transport.
const _isDirectRun =
  typeof process !== "undefined" &&
  process.argv[1] &&
  _resolvePath(process.argv[1]) === _thisFile;

if (_isDirectRun) {
  const client = new GkillReadClient();

  const transport = (process.env.MCP_TRANSPORT || "stdio").toLowerCase();
  const gkillHome = process.env.GKILL_HOME || _resolvePath(process.env.HOME || process.env.USERPROFILE || ".", "gkill");
  const mcpLogLevel = parseMcpLogLevel(process.env.MCP_LOG);
  const accessLog = new McpAccessLog(
    _resolvePath(gkillHome, "logs", "gkill_mcp_read_access.log"),
    mcpLogLevel,
    "gkill-read-server.mjs",
  );

  const server = new McpServer(client, accessLog);

  if (transport === "http") {
    const port = parseInt(process.env.MCP_PORT || "8808", 10);
    const issuer = process.env.MCP_OAUTH_ISSUER || `http://localhost:${port}`;
    const persistPath = _resolvePath(gkillHome, "configs", "mcp_oauth_read_state.json");
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
    new HttpTransport(server, port, oauthServer, { scope: "gkill:read", enableFileLinks: true }).start();
  } else {
    server.currentUserId = client.userId || null;
    accessLog.info("server_start", {
      transport, log_level: mcpLogLevel,
    });
    new StdioTransport(server).start();
  }
}

export { GkillReadClient, McpServer, OAuthServer, HttpTransport };
