#!/usr/bin/env node

// 読み書き両用の gkill MCP サーバ。
//
// **read + write そのもの。** ツール一覧は read サーバと同じ READ_TOOLS 配列を
// そのまま spread し、WRITE_TOOLS を足すだけ（両者の名前集合は重ならない）。
// ディスパッチと起動処理も3サーバ共通なので、このファイルが持つのは表の1行だけ。
// 正本: lib/mcp-server-base.mjs（handleToolCall）と lib/mcp-server-bootstrap.mjs（起動）。

import { PLUGIN_TOOLS } from "./lib/plugin-tools.mjs";
import { OAuthServer } from "./lib/oauth-server.mjs";
import { McpServerBase } from "./lib/mcp-server-base.mjs";
import { GkillClient } from "./lib/gkill-client.mjs";
import { READ_TOOLS } from "./lib/read-tools.mjs";
import { WRITE_TOOLS } from "./lib/write-tools.mjs";
import { HttpTransport } from "./lib/http-transport.mjs";
import { isDirectRun, readPackageVersion, startMcpServer } from "./lib/mcp-server-bootstrap.mjs";

// Tool definitions (READ_TOOLS + WRITE_TOOLS + PLUGIN_TOOLS。数の正本は verify_docs の実測)
const TOOLS = [
  ...READ_TOOLS,
  ...WRITE_TOOLS,
  ...PLUGIN_TOOLS,
];

class McpServer extends McpServerBase {
  constructor(client, accessLog = null) {
    super(client, accessLog, {
      serverName: "gkill-readwrite-mcp",
      serverVersion: readPackageVersion(),
      tools: TOOLS,
      // READ_TOOLS を全部載せる（read サーバと同じ）。加えて書き込みも持つ。
      readToolNames: null,
      writeAppName: "gkill_mcp_readwrite",
    });
  }
}

// Entry point — guarded so importing this module for tests does not start a transport.
if (isDirectRun(import.meta.url)) {
  startMcpServer({
    ServerClass: McpServer,
    client: new GkillClient(),
    scriptName: "gkill-readwrite-server.mjs",
    logFileName: "gkill_mcp_readwrite_access.log",
    oauthStateFileName: "mcp_oauth_readwrite_state.json",
    defaultPort: 8810,
    scope: "gkill:readwrite",
    enableFileLinks: true,
  });
}

export { GkillClient, McpServer, OAuthServer, HttpTransport };
