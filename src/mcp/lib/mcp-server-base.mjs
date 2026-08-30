// Read before editing: .claude/skills/gkill-mcp/SKILL.md (invariants for this area)
// 3つのMCPサーバに共通する JSON-RPC の受け口。
//
// handleMessage / handlePayload / constructor は3本とも1文字違わず同じだった。
// buildToolResult も read と readwrite は完全に同一で、write サーバだけが
// file-link 注入と IDF 画像ブロックを欠いた劣化コピーを持っていたので、
// 正しいほう (read / readwrite 版) をここへ引き上げて1本にする。
// handleToolCall も3本とも「plugin → read → write」の同じ順で、違うのは
// **write を持つか**と**read の選抜集合を持つか**の2つの値だけだったので、ここへ畳んだ。
// これで readwrite が read の上位集合であることが構造で担保される
// （2026-08-25 の実利用レビューが「read と readwrite でスキーマが違う」と報告したが、
//  実体は古いプロセスが昨日の定義を配っていただけで、コードは既に同一だった）。
// 継承側に残すのは options の表1行だけ（ADR-0611）。
//
// サーバ名・版・ツール一覧・read の選抜集合・write のアプリ名は options で受ける。
//
// server.current* へ書き戻してはいけない理由（並行要求の混線）:
// documents/adr/0601-mcp-request-context-immutable.md

import { GkillApiError, isPlainObject, invalidArgument } from "./errors.mjs";
import { assertTrimmedString } from "./validation.mjs";
import { applyFileLinks, normalizeMimeType, stripFilePaths, summarizeToolError } from "./payload.mjs";
import { summarizePluginToolPayload } from "./plugin-tools.mjs";
import { summarizeReadToolPayload, isReadToolName, handleReadToolCall } from "./read-handlers.mjs";
import { summarizeWriteToolPayload, handleWriteToolCall } from "./write-handlers.mjs";
import { handlePluginToolCall, isPluginToolName } from "./plugin-tools.mjs";
import { unknownToolMessage } from "./constants.mjs";

// summarizeToolPayload は結果の1行要約を返す。plugin → read → write の順に委ねる。
// 各要約器は対象外のツールに null を返すので、持っていないツールの分は素通りする
// (読み取り専用サーバは write の case に一致しない)。
function summarizeToolPayload(name, payload) {
  const pluginSummary = summarizePluginToolPayload(name, payload);
  if (pluginSummary !== null) {
    return pluginSummary;
  }
  const readSummary = summarizeReadToolPayload(name, payload);
  if (readSummary !== null) {
    return readSummary;
  }
  const writeSummary = summarizeWriteToolPayload(name, payload);
  if (writeSummary !== null) {
    return writeSummary;
  }
  return "Tool call completed.";
}

export class McpServerBase {
  constructor(client, accessLog, options) {
    this.serverName = options.serverName;
    this.serverVersion = options.serverVersion;
    this.tools = options.tools;
    // readToolNames: null なら READ_TOOLS 全部。Set ならそのうち載せる分だけ
    // （書き込み専用サーバは4本だけ載せる）。「read ツールか」の判定自体は
    // isReadToolName が正本で、ここは選抜集合。選抜集合だけで判定すると
    // READ_TOOLS から消えた名前がここに残っていても気づけない。
    this.readToolNames = options.readToolNames ?? null;
    // writeAppName: null なら書き込みツールを持たない（読み取り専用サーバ）。
    // 文字列なら書いた記録の create_app に載る値。
    this.writeAppName = options.writeAppName ?? null;
    this.client = client;
    this.accessLog = accessLog || { info() {}, warn() {}, error() {}, debug() {}, trace() {} };
    /** @type {string|null} Per-request session override set by HttpTransport for OAuth. */
    this.currentSessionId = null;
    /** @type {string|null} Per-request user id set by HttpTransport for OAuth. */
    this.currentUserId = null;
    /** @type {string|null} Per-request remote address set by HttpTransport. */
    this.currentRemoteAddr = null;
    /**
     * @type {boolean} True only when the MCP client runs on this machine (stdio transport).
     * Absolute filesystem paths are exposed to the client only when this is true.
     * Defaults to false so that a transport that forgets to opt in never leaks paths.
     */
    this.isLocalTransport = false;
    /**
     * @type {{publicBaseUrl: string, store: import("./lib/file-link-store.mjs").FileLinkStore}|null}
     * Set by HttpTransport so remote clients get a public file URL instead of a local path.
     * Null on stdio (local clients read the path directly).
     */
    this.fileLinkContext = null;
  }

  // handleToolCall は plugin → read → write の順で委譲する。3サーバ共通。
  async handleToolCall(name, args, ctx = null) {
    const sid = ctx ? ctx.sessionId : this.currentSessionId;

    if (isPluginToolName(name)) {
      return handlePluginToolCall(
        (pathname, body) => this.client.callApi(pathname, body, true, sid),
        name,
        args,
      );
    }

    if (isReadToolName(name) && (this.readToolNames === null || this.readToolNames.has(name))) {
      return handleReadToolCall(
        { client: this.client, sid, isLocalTransport: this.isLocalTransport },
        name,
        args,
      );
    }

    if (this.writeAppName === null) {
      throw new GkillApiError(unknownToolMessage(name));
    }

    // 書き込みに刻む user は、その要求を認証したアカウントでなければならない。
    // sid（どのアカウントのDBへ書くか）と userId（レコードに刻む名前）は
    // 出どころが別なので、放っておくと静かにずれる。
    // sid がトークン由来（HTTP/OAuth）なのに userId だけ取れないと、環境変数
    // GKILL_USER へ落ちて「別アカウントのセッションへ書いたのに create_user は
    // 手元の名前」という記録ができあがり、あとから誰が書いたのか追えなくなる。
    // stdio は sid を持たず環境変数で接続するので、そちらは従来どおり。
    const authenticatedUserId = ctx ? ctx.userId : this.currentUserId;
    if (sid && !authenticatedUserId) {
      throw new GkillApiError(
        "Cannot determine which user to record as the writer of this entry: " +
          "the session is authenticated but carries no user id, so create_user / update_user " +
          "would be taken from this server's environment instead of the account being written to. " +
          "Reconnect (re-authorize) this MCP server and retry.",
      );
    }
    const userId = authenticatedUserId || this.client.userId;
    return handleWriteToolCall(
      { client: this.client, sid, userId, appName: this.writeAppName },
      name,
      args,
    );
  }

  buildToolResult(name, payload, isError = false, ctx = null) {
    // ローカルクライアントには実パスを渡す。リモートには実パスを渡さず、
    // 代わりに期限付きの公開ファイルURLを注入する (発行できないときは実パスを消すだけ)。
    // file-link トークンは ctx.sessionId で鋳造する。ctx 未指定 (単体テスト) のみ this.currentSessionId。
    if (!this.isLocalTransport) {
      if (this.fileLinkContext && !isError) {
        applyFileLinks(payload, this.fileLinkContext, ctx ? ctx.sessionId : this.currentSessionId);
      } else {
        stripFilePaths(payload);
      }
    }

    const summary = isError
      ? summarizeToolError(name, payload?.error || "Unknown tool error", payload?.detail || null)
      : summarizeToolPayload(name, payload);

    const hasBase64 = name === "gkill_get_idf_file" && !isError && Boolean(payload?.file_content_base64);
    // 画像はimageブロックでバイト列を届ける
    const hasImageBlock = hasBase64 && Boolean(payload.is_image);

    // テキスト表現にbase64は載せない（読めないうえに肥大化するだけ）
    let textPayload = payload;
    if (hasBase64) {
      const { file_content_base64: _file_content_base64, ...rest } = payload;
      textPayload = rest;
    }
    // structuredContentからbase64を落とすのは、imageブロックで既にバイト列を届けている画像のときだけ。
    // 同じデータが1レスポンスに2回入ると、クライアント側のツール結果上限を超えて切り捨てられ、
    // 画像そのものが届かなくなる。非画像 (PDF等) はここが唯一のバイト列の渡し口なので残す。
    const structuredPayload = hasImageBlock ? textPayload : payload;

    const jsonText = textPayload !== undefined ? JSON.stringify(textPayload, null, 2) : undefined;

    const result = {
      content: [{ type: "text", text: jsonText ? `${summary}\n\n${jsonText}` : summary }],
      isError,
    };
    if (hasImageBlock) {
      result.content.push({
        type: "image",
        data: payload.file_content_base64,
        mimeType: normalizeMimeType(payload.mime_type),
      });
    }
    if (structuredPayload !== undefined) {
      result.structuredContent = structuredPayload;
    }
    return result;
  }

  // requestContext は HttpTransport が組む1リクエスト分の不変値
  // {sessionId,userId,remoteAddr}。以前は server.current* 共有フィールドに
  // 書いて await をまたいで読んでいたため、並行リクエストで別要求の
  // user/session が混線した。引数で末端まで流すことで混線を構造的に断つ。
  async handlePayload(payload, requestContext = null) {
    if (!Array.isArray(payload)) {
      return this.handleMessage(payload, requestContext);
    }
    if (payload.length === 0) {
      return { jsonrpc: "2.0", id: null, error: { code: -32600, message: "Invalid Request" } };
    }
    const responses = [];
    for (const message of payload) {
      const response = await this.handleMessage(message, requestContext);
      if (response !== null) {
        responses.push(response);
      }
    }
    return responses.length === 0 ? null : responses;
  }

  async handleMessage(message, requestContext = null) {
    // requestContext 未指定 (stdio / 単体テストの直接呼び出し) のときだけ
    // 起動時に一度設定される this.current* のスナップショットへフォールバックする。
    // HTTP 経路は必ず requestContext を渡すので、この分岐には入らない。
    const ctx = requestContext ?? Object.freeze({
      sessionId: this.currentSessionId,
      userId: this.currentUserId,
      remoteAddr: this.currentRemoteAddr,
    });
    if (!message || message.jsonrpc !== "2.0" || !message.method) {
      return {
        jsonrpc: "2.0",
        id: message && Object.prototype.hasOwnProperty.call(message, "id") ? message.id : null,
        error: { code: -32600, message: "Invalid Request" },
      };
    }

    const hasId = Object.prototype.hasOwnProperty.call(message, "id");
    const id = message.id;
    const method = message.method;
    const params = Object.prototype.hasOwnProperty.call(message, "params") ? message.params : {};

    if (method === "notifications/initialized") {
      return null;
    }

    if (method === "initialize") {
      if (!hasId) return null;
      return {
        jsonrpc: "2.0",
        id,
        result: {
          protocolVersion: "2024-11-05",
          capabilities: { tools: {} },
          serverInfo: { name: this.serverName, version: this.serverVersion },
        },
      };
    }

    if (method === "ping") {
      if (!hasId) return null;
      return { jsonrpc: "2.0", id, result: {} };
    }

    if (method === "tools/list") {
      if (!hasId) return null;
      return { jsonrpc: "2.0", id, result: { tools: this.tools } };
    }

    if (method === "tools/call") {
      if (!hasId) return null;
      const toolStart = Date.now();
      try {
        if (!isPlainObject(params)) {
          throw invalidArgument("params", "must be an object", params);
        }
        const toolName = assertTrimmedString(params.name, "name");
        const toolArgs = Object.prototype.hasOwnProperty.call(params, "arguments") ? params.arguments : {};
        const response = await this.handleToolCall(toolName, toolArgs, ctx);
        this.accessLog.info("tool_call", {
          tool: toolName,
          user_id: ctx.userId || null,
          remote_addr: ctx.remoteAddr || null,
          duration: `${Date.now() - toolStart}ms`,
        });
        return { jsonrpc: "2.0", id, result: this.buildToolResult(toolName, response, false, ctx) };
      } catch (error) {
        const detail = error instanceof GkillApiError ? error.detail : null;
        const messageText = error instanceof Error ? error.message : "Unknown tool error";
        // 引数の型違い（errors.mjs の invalidArgument）は呼び出し側の誤りで、HTTP でいう 400。
        // サーバ障害と同じ ERROR に混ぜると、ログの ERROR 件数が意味を持たなくなる。
        const isCallerError = detail !== null && typeof detail.field === "string";
        this.accessLog[isCallerError ? "warn" : "error"]("tool_call_error", {
          tool: params.name,
          user_id: ctx.userId || null,
          remote_addr: ctx.remoteAddr || null,
          duration: `${Date.now() - toolStart}ms`,
          error: messageText,
        });
        return {
          jsonrpc: "2.0",
          id,
          result: this.buildToolResult(params.name, { error: messageText, detail }, true, ctx),
        };
      }
    }

    if (!hasId) return null;
    return { jsonrpc: "2.0", id, error: { code: -32601, message: `Method not found: ${method}` } };
  }
}

// makeOAuthAuthenticateUser は OAuth の利用者認証コールバックを作る。
//
// read / write / readwrite の3サーバへ**逐語コピーされていた18行**（md5 一致）。
// 片方だけ直すと静かにずれるので1本にする。gkill のログイン応答は
// エンベロープ（errors / session_id）なので、HTTP ステータスではなく中身で判定する。
//
// 認証の失敗理由は**呼び出し側へ返さない**（総当たりの手掛かりになる）。
// アクセスログにだけ user_id つきで残す。
export function makeOAuthAuthenticateUser(client, accessLog) {
  return async (userId, passwordSha256) => {
    try {
      const response = await client.post("/api/login", {
        user_id: userId,
        password_sha256: passwordSha256,
        locale_name: client.defaultLocale,
      });
      if (client.hasErrors(response) || !response.session_id) {
        accessLog.warn("auth_failure", { user_id: userId, reason: "rejected_by_gkill" });
        return null;
      }
      accessLog.info("auth_success", { user_id: userId });
      return { sessionId: response.session_id };
    } catch (error) {
      // 応答には理由を返さない（総当たりの手がかりになる）が、ログには残す。
      // 捨てると「gkill へ届かない」のか「パスワードが違う」のかが区別できない。
      accessLog.warn("auth_failure", { user_id: userId, reason: "request_failed", error: String(error) });
      return null;
    }
  };
}
