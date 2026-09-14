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
      serverKind: "read",
      serverVersion: readPackageVersion(),
      tools: TOOLS,
      // READ_TOOLS を全部載せる。書き込みツールは持たない。
      readToolNames: null,
      writeAppName: null,
    });
  }
}

// 起動 spec の静的な部分。scope / ポート / file-link 可否は「このサーバが何者か」の宣言で、
// OAuth の scope はこの1値から metadata・認可既定値・トークン発行・受理検証まで全部が生成される
// （lib/mcp-server-bootstrap.mjs）。テストが宣言値そのものを固定できるよう export する
// —— ReadWrite サーバが gkill:read を広告していた事故は、テストが自前の値で
// OAuthServer を組み立てていて、ここの宣言を誰も読んでいなかったために漏れた。
// ServerClass と client は起動時にだけ作る（import 時に環境変数を読む副作用を避ける）。
export const START_SPEC = Object.freeze({
  scriptName: "gkill-read-server.mjs",
  logFileName: "gkill_mcp_read_access.log",
  oauthStateFileName: "mcp_oauth_read_state.json",
  defaultPort: 8808,
  scope: "gkill:read",
  enableFileLinks: true,
});

// Entry point — guarded so importing this module for tests does not start a transport.
if (isDirectRun(import.meta.url)) {
  startMcpServer({
    ServerClass: McpServer,
    client: new GkillReadClient(),
    ...START_SPEC,
  });
}

export { GkillReadClient, McpServer, OAuthServer, HttpTransport };
