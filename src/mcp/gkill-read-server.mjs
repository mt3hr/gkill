#!/usr/bin/env node

// 読み取り専用の gkill MCP サーバ。
//
// ディスパッチ（plugin → read → write）と起動処理は3サーバ共通で、
// このファイルが持つのは「どのツールを載せ、どの名前とポートで名乗るか」という表の1行だけ。
// 正本: lib/mcp-server-base.mjs（handleToolCall）と lib/mcp-server-bootstrap.mjs（起動）。

import { PLUGIN_TOOLS } from "./lib/plugin-tools.mjs";
import { OAuthServer } from "./lib/oauth-server.mjs";
import { McpServerBase } from "./lib/mcp-server-base.mjs";
import { GkillClient as GkillReadClient } from "./lib/gkill-client.mjs";
import { READ_TOOLS } from "./lib/read-tools.mjs";
import { HttpTransport } from "./lib/http-transport.mjs";
import { isDirectRun, readPackageVersion, startMcpServer } from "./lib/mcp-server-bootstrap.mjs";

const TOOLS = [
  ...READ_TOOLS,
  ...PLUGIN_TOOLS,
];

// McpServer: transport-independent JSON-RPC handler.
// handleMessage() / handleToolCall() は McpServerBase が持つ。
class McpServer extends McpServerBase {
  constructor(client, accessLog = null) {
    super(client, accessLog, {
      serverName: "gkill-read-mcp",
      serverVersion: readPackageVersion(),
      tools: TOOLS,
      // READ_TOOLS を全部載せる。書き込みツールは持たない。
      readToolNames: null,
      writeAppName: null,
    });
  }
}

// Entry point — guarded so importing this module for tests does not start a transport.
if (isDirectRun(import.meta.url)) {
  startMcpServer({
    ServerClass: McpServer,
    client: new GkillReadClient(),
    scriptName: "gkill-read-server.mjs",
    logFileName: "gkill_mcp_read_access.log",
    oauthStateFileName: "mcp_oauth_read_state.json",
    defaultPort: 8808,
    scope: "gkill:read",
    enableFileLinks: true,
  });
}

export { GkillReadClient, McpServer, OAuthServer, HttpTransport };
