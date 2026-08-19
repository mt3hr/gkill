#!/usr/bin/env node

import { readFileSync } from "node:fs";
import { dirname as _dirname, resolve as _resolvePath } from "node:path";
import process from "node:process";
import { fileURLToPath as _fileURLToPath } from "node:url";

import { GkillApiError } from "./lib/errors.mjs";
import {
  MAX_IDF_FILE_BYTES,
} from "./lib/constants.mjs";
import { normalizeKyouArgs, normalizeLocaleOnlyArgs, normalizeGpsArgs, normalizeIdfFileArgs } from "./lib/normalization.mjs";
import {
  PLUGIN_TOOLS,
  handlePluginToolCall,
  isPluginToolName,
  summarizePluginToolPayload,
  inlinePluginContents,
  summarizeInlinePluginContent,
} from "./lib/plugin-tools.mjs";
import { OAuthServer } from "./lib/oauth-server.mjs";
import { McpAccessLog, parseMcpLogLevel } from "./lib/access-log.mjs";
import { StdioTransport } from "./lib/stdio-transport.mjs";
import { McpServerBase } from "./lib/mcp-server-base.mjs";
import { GkillClient as GkillReadClient } from "./lib/gkill-client.mjs";
import { READ_TOOLS } from "./lib/read-tools.mjs";
import { HttpTransport } from "./lib/http-transport.mjs";
import { applyFileLinks, normalizeMimeType, stripFilePaths, summarizeToolError } from "./lib/payload.mjs";

const _thisFile = _fileURLToPath(import.meta.url);
const _thisDir = _dirname(_thisFile);
const _pkg = JSON.parse(readFileSync(_resolvePath(_thisDir, "../../package.json"), "utf8"));

const SERVER_NAME = "gkill-read-mcp";
const SERVER_VERSION = _pkg.version;

function summarizeToolPayload(name, payload) {
  const pluginSummary = summarizePluginToolPayload(name, payload);
  if (pluginSummary !== null) {
    return pluginSummary;
  }
  switch (name) {
    case "gkill_get_kyous": {
      const returnedCount = payload.returned_count ?? 0;
      const totalCount = payload.total_count ?? returnedCount;
      const remaining = totalCount - returnedCount;
      const pluginSuffix = summarizeInlinePluginContent(payload.plugin_content);
      if (payload.has_more && payload.next_cursor) {
        return `Returned ${returnedCount} of ${totalCount} kyou entries (${remaining} remaining). Next page: cursor="${payload.next_cursor}".${pluginSuffix}`;
      }
      return `Returned ${returnedCount} of ${totalCount} kyou entries (all results returned).${pluginSuffix}`;
    }
    case "gkill_get_mi_board_list":
      return `Fetched ${Array.isArray(payload.boards) ? payload.boards.length : 0} Mi boards.`;
    case "gkill_get_all_tag_names":
      return `Fetched ${Array.isArray(payload.tag_names) ? payload.tag_names.length : 0} tag names.`;
    case "gkill_get_all_rep_names":
      return `Fetched ${Array.isArray(payload.rep_names) ? payload.rep_names.length : 0} repository names.`;
    case "gkill_get_gps_log":
      return `Fetched ${Array.isArray(payload.gps_logs) ? payload.gps_logs.length : 0} GPS log entries.`;
    case "gkill_get_application_config":
      return "Fetched application configuration.";
    case "gkill_get_idf_file":
      return `Retrieved file: ${payload.file_name} (${payload.file_size_bytes} bytes, ${payload.mime_type})`;
    case "gkill_get_idf_file_path":
      return payload.exists
        ? `Resolved local file path: ${payload.file_path}`
        : `File not found in repository (no local path available).`;
    default:
      return "Tool call completed.";
  }
}

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


  buildToolResult(name, payload, isError = false) {
    // ローカルクライアントには実パスを渡す。リモートには実パスを渡さず、
    // 代わりに期限付きの公開ファイルURLを注入する (発行できないときは実パスを消すだけ)。
    if (!this.isLocalTransport) {
      if (this.fileLinkContext && !isError) {
        applyFileLinks(payload, this.fileLinkContext, this.currentSessionId);
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

  async handleToolCall(name, args) {
    const sid = this.currentSessionId;
    if (isPluginToolName(name)) {
      return handlePluginToolCall(
        (pathname, body) => this.client.callApi(pathname, body, true, sid),
        name,
        args,
      );
    }
    switch (name) {
      case "gkill_get_kyous": {
        const normalized = normalizeKyouArgs(args);
        const response = await this.client.callApi(
          "/api/get_kyous_mcp",
          {
            query: normalized.query,
            locale_name: normalized.locale_name,
            limit: normalized.limit,
            cursor: normalized.cursor,
            max_size_mb: normalized.max_size_mb,
            is_include_timeis: normalized.is_include_timeis,
            include_id: normalized.include_id || false,
            include_rep_name: normalized.include_rep_name || false,
          },
          true,
          sid,
        );
        const payload = {
          kyous: Array.isArray(response.kyous) ? response.kyous : [],
          total_count: response.total_count ?? 0,
          returned_count: response.returned_count ?? 0,
          has_more: Boolean(response.has_more),
          ...(response.next_cursor ? { next_cursor: response.next_cursor } : {}),
        };
        if (normalized.include_plugin_content) {
          payload.plugin_content = await inlinePluginContents(
            (pathname, body) => this.client.callApi(pathname, body, true, sid),
            payload.kyous,
            {
              maxTextLength: normalized.plugin_content_max_text_length,
              format: normalized.plugin_content_format,
              localeName: normalized.locale_name,
            },
          );
        }
        return payload;
      }
      case "gkill_get_mi_board_list": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await this.client.callApi("/api/get_mi_board_list", normalized, true, sid);
        return {
          boards: Array.isArray(response.boards) ? response.boards : [],
        };
      }
      case "gkill_get_all_tag_names": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await this.client.callApi("/api/get_all_tag_names", normalized, true, sid);
        return {
          tag_names: Array.isArray(response.tag_names) ? response.tag_names : [],
        };
      }
      case "gkill_get_all_rep_names": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await this.client.callApi("/api/get_all_rep_names", normalized, true, sid);
        return {
          rep_names: Array.isArray(response.rep_names) ? response.rep_names : [],
        };
      }
      case "gkill_get_gps_log": {
        const normalized = normalizeGpsArgs(args);
        const response = await this.client.callApi(
          "/api/get_gps_log",
          {
            start_date: normalized.start_date,
            end_date: normalized.end_date,
            locale_name: normalized.locale_name,
          },
          true,
          sid,
        );
        return {
          gps_logs: Array.isArray(response.gps_logs) ? response.gps_logs : [],
        };
      }
      case "gkill_get_application_config": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await this.client.callApi(
          "/api/get_application_config",
          normalized,
          true,
          sid,
        );
        const config = response.application_config || {};
        return {
          tag_struct: config.tag_struct,
          mi_board_struct: config.mi_board_struct,
          rep_struct: config.rep_struct,
          rep_type_struct: config.rep_type_struct,
          device_struct: config.device_struct,
          kftl_template_struct: config.kftl_template_struct,
          mi_default_board: config.mi_default_board,
          show_tags_in_list: config.show_tags_in_list,
        };
      }
      case "gkill_get_idf_file": {
        const normalized = normalizeIdfFileArgs(args);
        const filePath =
          "/files/" +
          encodeURIComponent(normalized.rep_name) +
          "/" +
          normalized.file_name
            .split("/")
            .map((s) => encodeURIComponent(s))
            .join("/");
        const sid = this.currentSessionId || (await this.client.login());
        const { buffer, contentType } = await this.client.fetchFile(filePath, sid);
        // base64はJSON-RPCレスポンスに素で載るので、青天井にすると数百MBの動画で応答が破裂する
        if (buffer.length > MAX_IDF_FILE_BYTES) {
          throw new GkillApiError(
            `File is too large to return through MCP: ${buffer.length} bytes (limit ${MAX_IDF_FILE_BYTES}). ` +
              `Use gkill_get_idf_file_path to get the local path and read the file from the filesystem instead.`,
            {
              file_name: normalized.file_name,
              file_size_bytes: buffer.length,
              max_bytes: MAX_IDF_FILE_BYTES,
            },
          );
        }
        const mimeType = normalizeMimeType(contentType);
        return {
          file_name: normalized.file_name,
          mime_type: mimeType,
          file_size_bytes: buffer.length,
          is_image: mimeType.startsWith("image/"),
          file_content_base64: buffer.toString("base64"),
        };
      }
      case "gkill_get_idf_file_path": {
        const normalized = normalizeIdfFileArgs(args);
        // 絶対パスは同一マシンのクライアントにしか意味がない。
        // リモートクライアントに渡すとユーザのディレクトリ構造の漏洩になるので、gkillに問い合わせもしない。
        if (!this.isLocalTransport) {
          throw new GkillApiError(
            "Local file paths are available only to MCP clients running on the same machine (stdio transport). " +
              "Use gkill_get_idf_file to fetch the file content instead.",
          );
        }
        const response = await this.client.callApi(
          "/api/get_idf_file_path",
          {
            rep_name: normalized.rep_name,
            file_name: normalized.file_name,
            locale_name: normalized.locale_name,
          },
          true,
          sid,
        );
        return {
          rep_name: normalized.rep_name,
          file_name: normalized.file_name,
          file_path: response.file_path || "",
          exists: Boolean(response.exists),
        };
      }
      default:
        throw new GkillApiError(`Unknown tool: ${name}`);
    }
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
