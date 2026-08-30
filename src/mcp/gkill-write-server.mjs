#!/usr/bin/env node

// 書き込み用の gkill MCP サーバ。
//
// ディスパッチ（plugin → read → write）と起動処理は3サーバ共通で、
// このファイルが持つのは「どのツールを載せ、どの名前とポートで名乗るか」という表の1行だけ。
// 正本: lib/mcp-server-base.mjs（handleToolCall）と lib/mcp-server-bootstrap.mjs（起動）。

import { PLUGIN_TOOLS } from "./lib/plugin-tools.mjs";
import { OAuthServer } from "./lib/oauth-server.mjs";
import { McpServerBase } from "./lib/mcp-server-base.mjs";
import { GkillClient as GkillWriteClient } from "./lib/gkill-client.mjs";
import { READ_TOOLS } from "./lib/read-tools.mjs";
import { WRITE_TOOLS } from "./lib/write-tools.mjs";
import { isDirectRun, readPackageVersion, startMcpServer } from "./lib/mcp-server-bootstrap.mjs";

// 書き込みサーバにも載せる読み取りツール。
// 前3つは書き込みの前に rep名 / 板名 / タグ名を引くためのもの。
// gkill_get_kyou_history は gkill_delete_kyou / gkill_restore_kyou の相棒で、
// 「今なにを消したのか」「なにを戻そうとしているのか」を同じサーバから確かめられないと
// 削除の取り消しが当てずっぽうになるので、ここに載せる。
//
// これは「載せる分の選抜集合」であって、「read ツールかどうか」の判定ではない
// （判定は read-handlers.mjs の isReadToolName が正本。McpServerBase が両方を見る）。
const WRITE_SERVER_READ_TOOL_NAMES = new Set([
  // 「どのアカウントへ書くのか」を書く前に確かめる唯一の手段。
  // 3サーバが別アカウントを向いていることがあり、これが無いと
  // 書き込み専用サーバだけが自分の接続先を答えられなかった
  // （しかも entityNotFoundMessage はこのツールを名指しで案内していた）。
  "gkill_get_application_config",
  "gkill_get_all_rep_names",
  "gkill_get_mi_board_list",
  "gkill_get_all_tag_names",
  "gkill_get_kyou_history",
]);

const TOOLS = [
  ...WRITE_TOOLS,
  // --- Read convenience tools ---
  // 定義は読み取りサーバと同じオブジェクトを使う（以前は description が食い違っていた）
  ...READ_TOOLS.filter((tool) => WRITE_SERVER_READ_TOOL_NAMES.has(tool.name)),
  ...PLUGIN_TOOLS,
];

class McpWriteServer extends McpServerBase {
  constructor(client, accessLog = null) {
    super(client, accessLog, {
      serverName: "gkill-write-mcp",
      serverVersion: readPackageVersion(),
      tools: TOOLS,
      readToolNames: WRITE_SERVER_READ_TOOL_NAMES,
      writeAppName: "gkill_mcp_write",
    });
  }
}

// 起動 spec の静的な部分。宣言値の意味と export の理由は gkill-read-server.mjs の同名定数を参照。
// enableFileLinks:false は「書き込み専用サーバは file-link URL を発行しない」の宣言で、
// bootstrap がそのまま HttpTransport へ渡す。
export const START_SPEC = Object.freeze({
  scriptName: "gkill-write-server.mjs",
  logFileName: "gkill_mcp_write_access.log",
  oauthStateFileName: "mcp_oauth_write_state.json",
  defaultPort: 8809,
  scope: "gkill:write",
  enableFileLinks: false,
});

// Entry point — guarded so importing this module for tests does not start a transport.
if (isDirectRun(import.meta.url)) {
  startMcpServer({
    ServerClass: McpWriteServer,
    client: new GkillWriteClient(),
    ...START_SPEC,
  });
}

export { GkillWriteClient, McpWriteServer, OAuthServer, WRITE_SERVER_READ_TOOL_NAMES };
