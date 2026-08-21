// 3つのMCPサーバに共通する JSON-RPC の受け口。
//
// handleMessage / handlePayload / constructor は3本とも1文字違わず同じだった。
// 違うのは「どのツールを持つか」と「結果をどう組み立てるか」だけなので、
// そこだけを継承側の buildToolResult / handleToolCall に残す。
//
// サーバ名・版・ツール一覧はコンストラクタの options で受ける。

import { GkillApiError, isPlainObject, invalidArgument } from "./errors.mjs";
import { assertTrimmedString } from "./validation.mjs";

export class McpServerBase {
  constructor(client, accessLog, options) {
    this.serverName = options.serverName;
    this.serverVersion = options.serverVersion;
    this.tools = options.tools;
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
        this.accessLog.error("tool_call_error", {
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
