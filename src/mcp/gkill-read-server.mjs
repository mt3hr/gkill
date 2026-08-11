#!/usr/bin/env node

import crypto from "node:crypto";
import { readFileSync } from "node:fs";
import http from "node:http";
import { dirname as _dirname, resolve as _resolvePath } from "node:path";
import process from "node:process";
import { fileURLToPath as _fileURLToPath } from "node:url";
import { Agent, fetch } from "undici";

import { GkillApiError, isPlainObject, invalidArgument } from "./lib/errors.mjs";
import { assertTrimmedString } from "./lib/validation.mjs";
import {
  ISO_DATETIME_DESC,
  DATE_ONLY_DESC,
  DEFAULT_KYOUS_LIMIT,
  DEFAULT_KYOUS_MAX_SIZE_MB,
  DEFAULT_KYOUS_INCLUDE_TIMEIS,
  MAX_IDF_FILE_BYTES,
  MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
  DEFAULT_PLUGIN_CONTENT_FORMAT,
  DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
  MAX_INLINE_PLUGIN_CONTENT_KYOUS,
  INLINE_PLUGIN_CONTENT_TOTAL_TEXT_LENGTH,
} from "./lib/constants.mjs";
import { normalizeKyouArgs, normalizeLocaleOnlyArgs, normalizeGpsArgs, normalizeIdfFileArgs } from "./lib/normalization.mjs";
import { FIND_QUERY_SCHEMA } from "./lib/find-query-schema.mjs";
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
import { FileLinkStore } from "./lib/file-link-store.mjs";

// リモート向け file_url の既定サムネサイズ (gkillの ?thumb=WxH に渡す。長辺上限1024)。
const DEFAULT_FILE_LINK_THUMB = "1024x1024";
// 配信ルートが受け付けるサムネ指定の検証用。
const THUMB_QUERY_REGEX = /^\d{1,4}x\d{1,4}$/;

const _thisFile = _fileURLToPath(import.meta.url);
const _thisDir = _dirname(_thisFile);
const _pkg = JSON.parse(readFileSync(_resolvePath(_thisDir, "../../package.json"), "utf8"));

const SERVER_NAME = "gkill-read-mcp";
const SERVER_VERSION = _pkg.version;

const AUTH_ERROR_CODES = new Set([
  "ERR000002", // AccountNotFoundError
  "ERR000013", // AccountSessionNotFoundError
  "ERR000238", // AccountDisabledError
]);

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

// Content-Type ヘッダから "; charset=..." などのパラメータを落とし、MIME型だけにする。
function normalizeMimeType(contentType) {
  return String(contentType || "").split(";")[0].trim();
}

// file_path はこのマシン上の絶対パス。同一マシンで動くクライアント (stdio) にしか意味がなく、
// リモートクライアントに渡すとユーザのディレクトリ構造を漏らすことになるので取り除く。
function stripFilePaths(value) {
  if (Array.isArray(value)) {
    for (const item of value) stripFilePaths(item);
    return value;
  }
  if (value !== null && typeof value === "object") {
    delete value.file_path;
    for (const key of Object.keys(value)) stripFilePaths(value[key]);
  }
  return value;
}

// idfペイロード (rep_name + file_name を持つ) を判定する。
function isIdfPayload(value) {
  return (
    value !== null &&
    typeof value === "object" &&
    typeof value.rep_name === "string" &&
    typeof value.file_name === "string"
  );
}

// リモートクライアント向けに、idfペイロードへ期限付きの公開ファイルURLを注入する。
// 実パス (file_path) は同時に取り除く。ローカルクライアント (stdio) では呼ばない。
// ctx = { publicBaseUrl, store }, gkillSessionId は発行元のOAuthセッション。
function applyFileLinks(value, ctx, gkillSessionId) {
  if (Array.isArray(value)) {
    for (const item of value) applyFileLinks(item, ctx, gkillSessionId);
    return value;
  }
  if (value === null || typeof value !== "object") {
    return value;
  }
  if (isIdfPayload(value)) {
    delete value.file_path;
    const token = ctx.store.mint({
      gkillSessionId,
      repName: value.rep_name,
      fileName: value.file_name,
      isImage: Boolean(value.is_image),
    });
    const base = `${ctx.publicBaseUrl}/files/${token}`;
    if (value.is_image) {
      // 既定は軽量なサムネ、原寸は file_url_full で別途取得できる
      value.file_url = `${base}?thumb=${DEFAULT_FILE_LINK_THUMB}`;
      value.file_url_full = base;
    } else {
      value.file_url = base;
    }
    return value;
  }
  for (const key of Object.keys(value)) applyFileLinks(value[key], ctx, gkillSessionId);
  return value;
}

function summarizeToolError(name, error, detail) {
  const prefix = name ? `${name} failed` : "Tool call failed";
  if (detail && detail.field) {
    return `${prefix}: ${error} (field: ${detail.field})`;
  }
  return `${prefix}: ${error}`;
}

const TOOLS = [
  {
    name: "gkill_get_kyous",
    description:
      "Search life-log entries (kyou) with optional filters and return enriched results including tags, texts, notifications, and typed payload inline. " +
      "Each result contains data_type, related_time, tags[], texts[], notifications[], timeis[] (attached TimeIs), and payload (type-specific fields). " +
      "Supports cursor-based pagination via next_cursor / cursor parameters. " +
      "Use limit and max_size_mb to control response size. " +
      "Available data_type values: kmemo (text memo), kc (numeric record), nlog (expense/income), lantana (mood 0-10), urlog (URL/bookmark), idf (file/image — use gkill_get_idf_file to fetch file content), git_commit_log (git commit), rekyou (repost of another entry), " +
      "timeis_start / timeis_end (time stamp), mi_create / mi_check / mi_limit / mi_start / mi_end (task, one value per projection — which one you get follows query.mi_sort_type), mirekyou_create / mirekyou_check / mirekyou_limit / mirekyou_start / mirekyou_end (an existing entry turned into a task). Plugins add their own data_type values (e.g. claude_conversation) — list them with gkill_get_plugin_list, and set include_plugin_content:true to read their bodies in this same response. " +
      "A filter activates simply by being present and non-null in the query; omit (or pass null for) filters you don't use. " +
      "Most used query fields: calendar_start_date/calendar_end_date, words, tags, for_mi. Advanced: map_latitude/map_longitude/map_radius, plaing_time, period_of_time_*, update_time. " +
      "Common query patterns: " +
      "Date range: {calendar_start_date:\"2026-03-01\", calendar_end_date:\"2026-03-07\"}. " +
      "Keyword search: {words:[\"keyword\"]}. " +
      "Tag filter: {tags:[\"tagname\"]}. " +
      "Mi tasks: {for_mi:true, mi_check_state:\"uncheck\", include_create_mi:true} — for_mi needs at least one include_*_mi flag or it returns nothing. " +
      "Practical recommendation: start with a minimal query, keep limit small, and add filters gradually. Hidden tags can be searched intentionally by passing them directly in query.tags or query.timeis_tags. rep_types are backend-specific and may be case-sensitive, so do not assume ApplicationConfig display labels map 1:1 to accepted query values. " +
      "If a query fails, first retry with fewer query fields, a smaller limit, and is_include_timeis=false; then add rep_types or TimeIs expansion back step by step. " +
      "The server always applies only_latest_data=true. " +
      "Results are returned in reverse chronological order (newest first, by related_time). " +
      "Response fields: kyous[], total_count, returned_count, has_more, next_cursor, plugin_content (inline-content counts; present only when include_plugin_content is true).",
    inputSchema: {
      type: "object",
      properties: {
        query: FIND_QUERY_SCHEMA,
        locale_name: {
          type: "string",
          description: "Locale, e.g. ja/en.",
        },
        limit: {
          type: "integer",
          description: `Max number of entries to return. Default: ${DEFAULT_KYOUS_LIMIT}.`,
          default: DEFAULT_KYOUS_LIMIT,
        },
        cursor: {
          type: "string",
          description:
            `Pagination cursor. Pass the next_cursor value from the previous response to fetch the next page. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}`,
        },
        max_size_mb: {
          type: "number",
          description: `Max response size in MB. Default: ${DEFAULT_KYOUS_MAX_SIZE_MB}.`,
          default: DEFAULT_KYOUS_MAX_SIZE_MB,
        },
        is_include_timeis: {
          type: "boolean",
          description: `Include attached TimeIs (plaing) data for each kyou — i.e., which TimeIs was running when each record was created. Default: ${DEFAULT_KYOUS_INCLUDE_TIMEIS}. Note: this does NOT filter out TimeIs-type kyous from results; those always appear regardless of this flag. Only controls inline plaing attachment on other data types.`,
          default: DEFAULT_KYOUS_INCLUDE_TIMEIS,
        },
        include_id: {
          type: "boolean",
          description:
            "Include entity ID (UUID) in each result object. Default: false (IDs omitted to reduce response size). " +
            "Set to true when you need IDs for subsequent operations such as gkill_update_* (patch update), gkill_delete_kyou (soft-delete), " +
            "gkill_add_tag (tagging by target_id), or gkill_add_text (annotating by target_id). " +
            "When true, each result includes an 'id' field at the top level of the kyou object.",
          default: false,
        },
        include_rep_name: {
          type: "boolean",
          description:
            "Include the source repository name in each result object. Default: false (omitted to reduce response size). " +
            "When true, each result includes a 'rep_name' field at the top level of the kyou object — the repository the entry came from, " +
            "which is the value you pass to query.reps to narrow later searches. " +
            "Note idf and plugin payloads already carry their own rep_name inside payload regardless of this flag.",
          default: false,
        },
        include_plugin_content: {
          type: "boolean",
          description:
            "Inline the body of plugin kyous (payload.kind='plugin') into this response, so you do not need a " +
            "separate follow-up call per entry. Default: false. " +
            "When true, each plugin payload gains content_status ('ok' | 'truncated' | 'skipped' | 'error'), plus " +
            "content_text (and content_html when plugin_content_format includes html) when the body was fetched, " +
            "content_skipped_reason ('max_kyous' | 'budget' | 'deadline' | 'rep_error') when it was skipped, and " +
            "content_error when the fetch failed. Only content_status='ok' means the body is complete. " +
            `At most ${MAX_INLINE_PLUGIN_CONTENT_KYOUS} plugin kyous per call are inlined, and ${INLINE_PLUGIN_CONTENT_TOTAL_TEXT_LENGTH} characters in total. ` +
            "To read one long body in full, narrow the query to that single entry (query.ids) and " +
            "raise plugin_content_max_text_length. " +
            "Enable this only when you actually intend to read plugin bodies: it costs one extra request per plugin kyou.",
          default: false,
        },
        plugin_content_max_text_length: {
          type: "integer",
          description:
            "Max characters of inlined text per plugin kyou. Only used when include_plugin_content is true. " +
            `Default: ${DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH}, max: ${MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH}. ` +
            "Longer bodies are cut and content_status becomes 'truncated'. Raising this reduces how many entries fit " +
            "the shared total-text budget, so raise it only when fetching a small number of long records.",
          default: DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
        },
        plugin_content_format: {
          type: "string",
          description:
            "Format of the inlined plugin body. Only used when include_plugin_content is true. " +
            "'text' (default) converts the plugin's HTML to plain text into content_text, 'html' puts the raw HTML " +
            "into content_html, 'both' fills both. Prefer 'text': plugin content HTML is mostly presentation CSS/JS.",
          enum: ["text", "html", "both"],
          default: DEFAULT_PLUGIN_CONTENT_FORMAT,
        },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_mi_board_list",
    description:
      "Get the list of Mi (task) board names configured in gkill. Boards are like Kanban columns that organize tasks. " +
      "Use this to discover existing board names for Mi queries (query.mi_board_name). " +
      "Response fields: boards[] (array of board name strings).",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string", description: "Locale, e.g. ja/en." },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_all_tag_names",
    description: "Get all tag names defined in gkill. Use this to discover available tags for filtering in gkill_get_kyous via query.tags or query.timeis_tags.",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string", description: "Locale, e.g. ja/en." },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_all_rep_names",
    description: "Get all repository names configured in gkill. Use this to discover rep names for filtering in gkill_get_kyous via query.reps.",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string", description: "Locale, e.g. ja/en." },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_gps_log",
    description: "Get GPS log entries in a date range. Returns array of GPS log objects with latitude, longitude, timestamp, and related metadata. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        start_date: {
          type: "string",
          description: `Required ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}`,
        },
        end_date: {
          type: "string",
          description: `Required ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}`,
        },
        locale_name: { type: "string", description: "Locale, e.g. ja/en." },
      },
      required: ["start_date", "end_date"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_application_config",
    description:
      "Get application configuration including tag hierarchy, task board structure, repository structure, and KFTL templates. " +
      "Recommended first call: use this before gkill_get_kyous to understand the data organization, visible tags, and board names. " +
      "Response fields: tag_struct (tag parent-child hierarchy with check_when_inited, is_force_hide, children), " +
      "mi_board_struct (task board hierarchy), rep_struct (repository hierarchy), rep_type_struct (repository type hierarchy), " +
      "device_struct (device hierarchy), kftl_template_struct (KFTL templates), mi_default_board (default board name, e.g. \"Inbox\"), show_tags_in_list (boolean). " +
      "Note that display labels in this config may not map 1:1 to accepted rep_types query values.",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: {
          type: "string",
          description: "Locale, e.g. ja/en.",
        },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_idf_file",
    description:
      "Retrieve actual file content for an IDF (file/image/video/audio) kyou entry. " +
      "First use gkill_get_kyous to find IDF entries (data_type 'idf'), then call this tool " +
      "with the rep_name and file_name from the IDF payload to get the file content as base64. " +
      "For images, the content is returned as an MCP image content block that AI can view directly. " +
      "PREFER PATH OR URL INSTEAD WHEN AVAILABLE: if the IDF payload carries a 'file_path' (local clients) " +
      "read it directly from the filesystem; if it carries a 'file_url' (remote clients) fetch that URL to " +
      "get the bytes with no auth. Both avoid base64 transfer and work at any size. Use this tool only as a " +
      "fallback when neither is available; it is capped by GKILL_MCP_MAX_FILE_BYTES.",
    inputSchema: {
      type: "object",
      properties: {
        rep_name: {
          type: "string",
          description: "Repository name from the IDF payload's rep_name field.",
        },
        file_name: {
          type: "string",
          description: "File name from the IDF payload's file_name field.",
        },
        locale_name: {
          type: "string",
          description: "Locale, e.g. ja/en.",
        },
      },
      required: ["rep_name", "file_name"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_idf_file_path",
    description:
      "Resolve the absolute local filesystem path of an IDF file from its rep_name and file_name. " +
      "Available only when gkill runs on the same machine as this MCP server and you are connected " +
      "over stdio; otherwise it returns an error. Once you have the path, read the file directly from " +
      "the filesystem — for images this lets you view them without any base64 transfer, at any file size. " +
      "gkill_get_kyous already includes 'file_path' in IDF payloads when available, so call this tool " +
      "only when you have a rep_name/file_name but not the path. Returns exists=false if the file is " +
      "not registered in the repository.",
    inputSchema: {
      type: "object",
      properties: {
        rep_name: {
          type: "string",
          description: "Repository name from the IDF payload's rep_name field.",
        },
        file_name: {
          type: "string",
          description: "File name from the IDF payload's file_name field.",
        },
        locale_name: {
          type: "string",
          description: "Locale, e.g. ja/en.",
        },
      },
      required: ["rep_name", "file_name"],
      additionalProperties: false,
    },
  },
  ...PLUGIN_TOOLS,
];

class GkillReadClient {
  constructor() {
    this.baseUrl = process.env.GKILL_BASE_URL || "http://127.0.0.1:9999";
    this.userId = process.env.GKILL_USER || "";
    this.passwordSha256 = process.env.GKILL_PASSWORD_SHA256 || "";
    this.password = process.env.GKILL_PASSWORD || "";
    this.defaultLocale = process.env.GKILL_LOCALE || "ja";
    this.sessionId = process.env.GKILL_SESSION_ID || "";
    const insecure = process.env.GKILL_INSECURE === "true" || process.env.GKILL_INSECURE === "1";
    this.dispatcher = insecure ? new Agent({ connect: { rejectUnauthorized: false } }) : null;
  }

  resolvePasswordSha256() {
    if (this.passwordSha256) {
      return this.passwordSha256;
    }
    if (this.password) {
      return crypto.createHash("sha256").update(this.password).digest("hex");
    }
    return "";
  }

  buildApiUrl(pathname) {
    return new URL(pathname, this.baseUrl).toString();
  }

  hasErrors(responseBody) {
    return Boolean(responseBody && Array.isArray(responseBody.errors) && responseBody.errors.length > 0);
  }

  hasAuthErrors(responseBody) {
    if (!this.hasErrors(responseBody)) {
      return false;
    }
    return responseBody.errors.some((err) => AUTH_ERROR_CODES.has(err.error_code));
  }

  formatErrors(responseBody) {
    if (!this.hasErrors(responseBody)) {
      return "";
    }
    return responseBody.errors
      .map((err) => `${err.error_code ?? "UNKNOWN"}: ${err.error_message ?? "unknown error"}`)
      .join("; ");
  }

  async post(pathname, body) {
    const url = this.buildApiUrl(pathname);
    const timeoutMs = parseInt(process.env.GKILL_FETCH_TIMEOUT_MS || "120000", 10);
    let response;
    try {
      const fetchOptions = {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(body),
        signal: AbortSignal.timeout(timeoutMs),
      };
      if (this.dispatcher) {
        fetchOptions.dispatcher = this.dispatcher;
      }
      response = await fetch(url, fetchOptions);
    } catch (error) {
      throw new GkillApiError(`Network error at ${pathname}.`, {
        url,
        message: error instanceof Error ? error.message : String(error),
        cause:
          error && typeof error === "object" && "cause" in error
            ? String(error.cause && error.cause.message ? error.cause.message : error.cause)
            : null,
      });
    }

    let jsonBody;
    try {
      jsonBody = await response.json();
    } catch (error) {
      throw new GkillApiError(`Failed to parse JSON response from ${pathname}.`, {
        cause: String(error),
      });
    }

    if (!response.ok) {
      throw new GkillApiError(`HTTP ${response.status} from ${pathname}.`, {
        status: response.status,
        body: jsonBody,
      });
    }

    return jsonBody;
  }

  async login() {
    if (this.sessionId) {
      return this.sessionId;
    }

    const passwordSha256 = this.resolvePasswordSha256();
    if (!this.userId || !passwordSha256) {
      throw new GkillApiError(
        "Missing login credentials. Set GKILL_USER and GKILL_PASSWORD_SHA256 (or GKILL_PASSWORD).",
      );
    }

    const response = await this.post("/api/login", {
      user_id: this.userId,
      password_sha256: passwordSha256,
      locale_name: this.defaultLocale,
    });

    if (this.hasErrors(response)) {
      throw new GkillApiError(`Login failed: ${this.formatErrors(response)}`, response);
    }
    if (!response.session_id) {
      throw new GkillApiError("Login succeeded but session_id is missing.", response);
    }

    this.sessionId = response.session_id;
    return this.sessionId;
  }

  async callRead(pathname, requestBody, requiresAuth, sessionIdOverride = null) {
    const localeName = requestBody.locale_name || this.defaultLocale;
    const body = {
      ...requestBody,
      locale_name: localeName,
    };

    if (requiresAuth) {
      body.session_id = sessionIdOverride || body.session_id || (await this.login());
    }

    let response = await this.post(pathname, body);
    if (requiresAuth && this.hasAuthErrors(response)) {
      this.sessionId = "";
      body.session_id = await this.login();
      response = await this.post(pathname, body);
    }

    if (this.hasErrors(response)) {
      throw new GkillApiError(`API error at ${pathname}: ${this.formatErrors(response)}`, response);
    }
    return response;
  }

  async fetchFile(filePath, sessionId) {
    const url = this.buildApiUrl(filePath);
    const timeoutMs = parseInt(process.env.GKILL_FETCH_TIMEOUT_MS || "120000", 10);
    const fetchOptions = {
      method: "GET",
      headers: {
        Cookie: `gkill_session_id=${sessionId}`,
      },
      signal: AbortSignal.timeout(timeoutMs),
    };
    if (this.dispatcher) {
      fetchOptions.dispatcher = this.dispatcher;
    }
    let response;
    try {
      response = await fetch(url, fetchOptions);
    } catch (error) {
      throw new GkillApiError(`Network error fetching file ${filePath}.`, {
        url,
        message: error instanceof Error ? error.message : String(error),
      });
    }
    if (!response.ok) {
      throw new GkillApiError(`HTTP ${response.status} fetching file ${filePath}.`, {
        status: response.status,
      });
    }
    const contentType = response.headers.get("content-type") || "application/octet-stream";
    const buffer = Buffer.from(await response.arrayBuffer());
    return { buffer, contentType };
  }
}

// McpServer: transport-independent JSON-RPC handler.
// handleMessage() returns a response object (or null for notifications).
class McpServer {
  constructor(client, accessLog = null) {
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

  async handlePayload(payload) {
    if (!Array.isArray(payload)) {
      return this.handleMessage(payload);
    }
    if (payload.length === 0) {
      return { jsonrpc: "2.0", id: null, error: { code: -32600, message: "Invalid Request" } };
    }
    const responses = [];
    for (const message of payload) {
      const response = await this.handleMessage(message);
      if (response !== null) {
        responses.push(response);
      }
    }
    return responses.length === 0 ? null : responses;
  }

  async handleToolCall(name, args) {
    const sid = this.currentSessionId;
    if (isPluginToolName(name)) {
      return handlePluginToolCall(
        (pathname, body) => this.client.callRead(pathname, body, true, sid),
        name,
        args,
      );
    }
    switch (name) {
      case "gkill_get_kyous": {
        const normalized = normalizeKyouArgs(args);
        const response = await this.client.callRead(
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
            (pathname, body) => this.client.callRead(pathname, body, true, sid),
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
        const response = await this.client.callRead("/api/get_mi_board_list", normalized, true, sid);
        return {
          boards: Array.isArray(response.boards) ? response.boards : [],
        };
      }
      case "gkill_get_all_tag_names": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await this.client.callRead("/api/get_all_tag_names", normalized, true, sid);
        return {
          tag_names: Array.isArray(response.tag_names) ? response.tag_names : [],
        };
      }
      case "gkill_get_all_rep_names": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await this.client.callRead("/api/get_all_rep_names", normalized, true, sid);
        return {
          rep_names: Array.isArray(response.rep_names) ? response.rep_names : [],
        };
      }
      case "gkill_get_gps_log": {
        const normalized = normalizeGpsArgs(args);
        const response = await this.client.callRead(
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
        const response = await this.client.callRead(
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
        const response = await this.client.callRead(
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

  async handleMessage(message) {
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
          serverInfo: { name: SERVER_NAME, version: SERVER_VERSION },
        },
      };
    }

    if (method === "ping") {
      if (!hasId) return null;
      return { jsonrpc: "2.0", id, result: {} };
    }

    if (method === "tools/list") {
      if (!hasId) return null;
      return { jsonrpc: "2.0", id, result: { tools: TOOLS } };
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
        const response = await this.handleToolCall(toolName, toolArgs);
        this.accessLog.info("tool_call", {
          tool: toolName,
          user_id: this.currentUserId || null,
          remote_addr: this.currentRemoteAddr || null,
          duration: `${Date.now() - toolStart}ms`,
        });
        return { jsonrpc: "2.0", id, result: this.buildToolResult(toolName, response, false) };
      } catch (error) {
        const detail = error instanceof GkillApiError ? error.detail : null;
        const messageText = error instanceof Error ? error.message : "Unknown tool error";
        this.accessLog.error("tool_call_error", {
          tool: params.name,
          user_id: this.currentUserId || null,
          remote_addr: this.currentRemoteAddr || null,
          duration: `${Date.now() - toolStart}ms`,
          error: messageText,
        });
        return {
          jsonrpc: "2.0",
          id,
          result: this.buildToolResult(params.name, { error: messageText, detail }, true),
        };
      }
    }

    if (!hasId) return null;
    return { jsonrpc: "2.0", id, error: { code: -32601, message: `Method not found: ${method}` } };
  }
}

// StdioTransport: reads JSON-RPC from stdin (LSP or NDJSON framing), writes NDJSON to stdout.
class StdioTransport {
  constructor(server) {
    this.server = server;
    this.buffer = Buffer.alloc(0);
    // stdioで話す相手はこのマシン上のプロセスなので、ファイルの絶対パスを渡してよい
    this.server.isLocalTransport = true;
  }

  start() {
    process.stdin.on("data", (chunk) => this.onData(chunk));
    process.stdin.on("error", (e) => this.logError("stdin error", e));
    process.stdin.resume();
  }

  logError(message, error) {
    process.stderr.write(`${message}: ${String(error)}\n`);
  }

  writeMessage(message) {
    const json = JSON.stringify(message);
    process.stdout.write(`${json}\n`);
  }

  async dispatch(message) {
    try {
      const response = await this.server.handlePayload(message);
      if (response) this.writeMessage(response);
    } catch (error) {
      this.logError("unhandled request error", error);
      if (message && !Array.isArray(message) && Object.prototype.hasOwnProperty.call(message, "id")) {
        this.writeMessage({ jsonrpc: "2.0", id: message.id, error: { code: -32603, message: "Internal error" } });
      }
    }
  }

  onData(chunk) {
    this.buffer = Buffer.concat([this.buffer, chunk]);

    while (true) {
      // LSP-style framing: "Content-Length: N\r\n\r\n{...}"
      const headerEnd = this.buffer.indexOf("\r\n\r\n");
      if (headerEnd !== -1) {
        const headerText = this.buffer.subarray(0, headerEnd).toString("utf8");
        const headers = headerText.split("\r\n");
        let contentLength = null;
        for (const line of headers) {
          const idx = line.indexOf(":");
          if (idx === -1) continue;
          const key = line.slice(0, idx).trim().toLowerCase();
          const value = line.slice(idx + 1).trim();
          if (key === "content-length") {
            contentLength = Number.parseInt(value, 10);
          }
        }

        if (!Number.isFinite(contentLength) || contentLength < 0) {
          this.logError("invalid content-length header", headerText);
          this.buffer = Buffer.alloc(0);
          return;
        }

        const totalLength = headerEnd + 4 + contentLength;
        if (this.buffer.length < totalLength) return;

        const bodyBuffer = this.buffer.subarray(headerEnd + 4, totalLength);
        this.buffer = this.buffer.subarray(totalLength);

        let message;
        try {
          message = JSON.parse(bodyBuffer.toString("utf8"));
        } catch (error) {
          this.logError("invalid json body", error);
          continue;
        }

        this.dispatch(message);
        continue;
      }

      // NDJSON-style framing: one JSON-RPC message per line.
      const lf = this.buffer.indexOf("\n");
      if (lf === -1) return;
      const line = this.buffer.subarray(0, lf).toString("utf8").trim();
      this.buffer = this.buffer.subarray(lf + 1);
      if (line.length === 0) continue;

      let message;
      try {
        message = JSON.parse(line);
      } catch (_error) {
        continue;
      }
      this.dispatch(message);
    }
  }
}

// HttpTransport: Streamable HTTP transport (MCP spec 2024-11-05).
// Supports POST /mcp (requests), GET /mcp (SSE stream), DELETE /mcp (session end).
// OAuth 2.1 endpoints for ChatGPT and Claude.ai MCP connectors.
class HttpTransport {
  /**
   * @param {McpServer} server
   * @param {number} port
   * @param {OAuthServer} oauthServer
   */
  constructor(server, port, oauthServer) {
    this.server = server;
    this.port = port;
    this.oauthServer = oauthServer;
    // HTTP越しのクライアントは別マシン (例: クラウド上のAI) でありうる。
    // このMCPサーバ自身がgkillと同居していても、絶対パスを渡してよい相手ではない。
    this.server.isLocalTransport = false;
    // リモートクライアントには実パスの代わりに期限付きの公開ファイルURLを渡す。
    // issuer は MCP_OAUTH_ISSUER (公開URL) で、この配信ルート自身の基点になる。
    this.fileLinkStore = new FileLinkStore();
    this.fileLinkStore.startCleanup();
    this.server.fileLinkContext = {
      publicBaseUrl: this.oauthServer.issuer,
      store: this.fileLinkStore,
    };
  }

  start() {
    const httpServer = http.createServer((req, res) => this.handleRequest(req, res));
    httpServer.listen(this.port, "0.0.0.0", () => {
      process.stderr.write(`MCP HTTP server listening on http://0.0.0.0:${this.port}/mcp [OAuth issuer: ${this.oauthServer.issuer}]\n`);
    });
  }

  parseRoute(req) {
    const url = new URL(req.url, "http://localhost");
    const pathname = url.pathname;
    const query = Object.fromEntries(url.searchParams);

    // Protected Resource Metadata (RFC 9728)
    if (pathname === "/.well-known/oauth-protected-resource" ||
        pathname === "/.well-known/oauth-protected-resource/mcp") {
      return { type: "oauth-protected-resource", pathname };
    }

    // OAuth Authorization Server Metadata (RFC 8414)
    if (pathname === "/.well-known/oauth-authorization-server") {
      return { type: "oauth-metadata", pathname, query };
    }

    // OAuth endpoints — /oauth/* canonical, /* fallback for Claude.ai (known bug: ignores metadata endpoints)
    if (pathname === "/oauth/authorize" || pathname === "/authorize") {
      return { type: "oauth-authorize", pathname, query };
    }
    if (pathname === "/oauth/token" || pathname === "/token") {
      return { type: "oauth-token", pathname };
    }
    if (pathname === "/oauth/register" || pathname === "/register") {
      return { type: "oauth-register", pathname };
    }

    // MCP endpoint
    if (pathname === "/mcp") {
      return { type: "mcp", pathname };
    }

    // Public file delivery — token in the path, no Bearer needed (image fetchers
    // cannot send auth headers). The token itself is the security boundary.
    if (pathname.startsWith("/files/")) {
      return { type: "file", pathname, token: decodeURIComponent(pathname.slice("/files/".length)), query };
    }

    return null;
  }

  logRequest(req, extra = {}) {
    const payload = {
      method: req.method,
      path: req.url,
      sessionId: req.headers["mcp-session-id"] || null,
      ...extra,
    };
    process.stderr.write(`[${new Date().toISOString()}] MCP HTTP ${JSON.stringify(payload)}\n`);

    // Also write to access log file
    const statusCode = extra.statusCode || 0;
    const level = statusCode >= 400 ? "warn" : "info";
    this.server.accessLog[level]("http_request", {
      remote_addr: req.socket?.remoteAddress || null,
      method: req.method,
      path: req.url,
      status: statusCode,
      ...(extra.methods ? { methods: extra.methods } : {}),
      ...(extra.reason ? { reason: extra.reason } : {}),
      ...(extra.responseBytes !== undefined ? { response_bytes: extra.responseBytes } : {}),
    });
  }

  sendJson(res, statusCode, payload, headers = {}) {
    const body = payload === undefined ? "" : JSON.stringify(payload);
    const baseHeaders = body
      ? { "Content-Type": "application/json" }
      : {};
    res.writeHead(statusCode, { ...baseHeaders, ...headers });
    res.end(body);
    return Buffer.byteLength(body, "utf8");
  }

  summarizeJsonRpcMethods(payload) {
    if (Array.isArray(payload)) {
      return payload
        .map((item) => (item && typeof item === "object" && "method" in item ? item.method : "invalid"))
        .join(",");
    }
    if (payload && typeof payload === "object" && "method" in payload) {
      return payload.method;
    }
    return "invalid";
  }

  // 公開ファイル配信。トークンからファイルを解決し、gkillからバイトを取って返す。
  // Bearer不要 (画像取得は認証ヘッダを付けられない) なので、トークンが唯一の防御線。
  async handleFileServe(req, res, token, query) {
    const store = this.server.fileLinkContext?.store;
    const link = store ? store.resolve(token) : null;
    if (!link) {
      this.logRequest(req, { statusCode: 404, reason: "file_token_invalid" });
      this.sendJson(res, 404, { error: "Not Found" });
      return;
    }

    let gkillPath =
      "/files/" +
      encodeURIComponent(link.repName) +
      "/" +
      link.fileName
        .split("/")
        .map((s) => encodeURIComponent(s))
        .join("/");
    // サムネ指定は画像のときだけ、かつ WxH 形式に限って gkill に転送する。
    if (link.isImage && typeof query?.thumb === "string" && THUMB_QUERY_REGEX.test(query.thumb)) {
      gkillPath += `?thumb=${query.thumb}`;
    }

    try {
      const { buffer, contentType } = await this.server.client.fetchFile(gkillPath, link.gkillSessionId);
      this.logRequest(req, { statusCode: 200, responseBytes: buffer.length });
      res.writeHead(200, {
        "Content-Type": normalizeMimeType(contentType) || "application/octet-stream",
        "Content-Length": buffer.length,
        "Cache-Control": "private, max-age=300",
        "Access-Control-Allow-Origin": "*",
      });
      res.end(buffer);
    } catch (error) {
      this.logRequest(req, { statusCode: 502, reason: "file_fetch_failed" });
      this.server.accessLog.error("file_fetch_error", {
        remote_addr: req.socket?.remoteAddress || null,
        error: error instanceof Error ? error.message : String(error),
      });
      this.sendJson(res, 502, { error: "Bad Gateway" });
    }
  }

  handleRequest(req, res) {
    const route = this.parseRoute(req);
    if (!route) {
      this.logRequest(req, { statusCode: 404, reason: "route_not_found" });
      this.sendJson(res, 404, { error: "Not Found. Use POST /mcp" });
      return;
    }

    // OAuth discovery/auth endpoints — no Bearer auth required
    if (route.type === "oauth-protected-resource") {
      return this.handleProtectedResourceMetadata(req, res);
    }
    if (route.type === "oauth-metadata") {
      return this.handleOAuthMetadata(req, res);
    }
    if (route.type === "oauth-authorize") {
      return this.handleOAuthAuthorize(req, res, route.query);
    }
    if (route.type === "oauth-token") {
      return this.handleOAuthToken(req, res);
    }
    if (route.type === "oauth-register") {
      return this.handleOAuthRegister(req, res);
    }

    // Public file delivery — Bearer 不要。トークンが防御線。
    if (route.type === "file") {
      if (req.method === "OPTIONS") {
        this.logRequest(req, { statusCode: 204 });
        res.writeHead(204, {
          "Access-Control-Allow-Origin": "*",
          "Access-Control-Allow-Methods": "GET, OPTIONS",
        });
        res.end();
        return;
      }
      if (req.method !== "GET") {
        this.logRequest(req, { statusCode: 405, reason: "method_not_allowed" });
        this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "GET, OPTIONS" });
        return;
      }
      return this.handleFileServe(req, res, route.token, route.query);
    }

    // MCP endpoint — require OAuth Bearer token
    const bearerToken = OAuthServer.extractBearerToken(req.headers["authorization"] || "");
    const tokenData = bearerToken ? this.oauthServer.validateAccessToken(bearerToken) : null;

    if (!tokenData) {
      this.logRequest(req, { statusCode: 401, reason: "unauthorized" });
      this.server.accessLog.warn("token_rejected", {
        remote_addr: req.socket?.remoteAddress || null,
        method: req.method, path: req.url,
      });
      const resourceMetadataUrl = `${this.oauthServer.issuer}/.well-known/oauth-protected-resource`;
      this.sendJson(res, 401, {
        error: "Unauthorized",
        error_description: "Bearer token required",
      }, {
        "WWW-Authenticate": `Bearer resource_metadata="${resourceMetadataUrl}"`,
      });
      return;
    }

    // NOTE: _lastTokenUserId は handlePost 内で server.currentUserId に転記される。
    // HTTP/1.1 直列処理を前提としており、HTTP/2 並行リクエスト時はリクエスト間でリークする可能性がある。
    this._lastTokenUserId = tokenData.userId || null;
    switch (req.method) {
      case "POST":
        return this.handlePost(req, res, tokenData.gkillSessionId);
      case "GET":
        return this.handleGet(req, res);
      case "DELETE":
        return this.handleDelete(req, res);
      default:
        this.logRequest(req, { statusCode: 405, reason: "method_not_allowed" });
        this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "GET, POST, DELETE" });
    }
  }

  handlePost(req, res, oauthSessionId = null) {
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", async () => {
      let payload;
      try {
        payload = JSON.parse(Buffer.concat(chunks).toString("utf8"));
      } catch {
        this.logRequest(req, { statusCode: 400, reason: "parse_error" });
        this.sendJson(res, 400, { jsonrpc: "2.0", id: null, error: { code: -32700, message: "Parse error" } });
        return;
      }

      try {
        // Set session override for OAuth-authenticated requests
        this.server.currentSessionId = oauthSessionId;
        this.server.currentUserId = this._lastTokenUserId || null;
        this.server.currentRemoteAddr = req.socket?.remoteAddress || null;
        const response = await this.server.handlePayload(payload);
        this.server.currentSessionId = null;
        this.server.currentUserId = null;
        this.server.currentRemoteAddr = null;
        const methods = this.summarizeJsonRpcMethods(payload);

        if (response === null) {
          this.logRequest(req, { methods, statusCode: 202, responseBytes: 0 });
          res.writeHead(202);
          res.end();
          return;
        }
        const responseBytes = this.sendJson(res, 200, response);
        this.logRequest(req, { methods, statusCode: 200, responseBytes });
      } catch (error) {
        this.server.currentSessionId = null;
        this.server.currentUserId = null;
        this.server.currentRemoteAddr = null;
        process.stderr.write(`HTTP handler error: ${String(error)}\n`);
        const id =
          payload && !Array.isArray(payload) && Object.prototype.hasOwnProperty.call(payload, "id") ? payload.id : null;
        const responseBytes = this.sendJson(res, 200, {
          jsonrpc: "2.0",
          id,
          error: { code: -32603, message: "Internal error" },
        });
        this.logRequest(req, {
          methods: this.summarizeJsonRpcMethods(payload),
          statusCode: 200,
          responseBytes,
          reason: "internal_error",
        });
      }
    });
  }

  handleGet(req, res) {
    // SSE endpoint for server-initiated notifications.
    // Currently gkill has no server-push notifications, so just hold the connection open.
    const accept = req.headers["accept"] || "";
    if (!accept.includes("text/event-stream")) {
      this.logRequest(req, { statusCode: 406, reason: "missing_sse_accept_header" });
      this.sendJson(res, 406, { error: "Not Acceptable. Use Accept: text/event-stream" });
      return;
    }

    res.writeHead(200, {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache",
      Connection: "keep-alive",
    });
    // Keep connection alive with periodic comments
    const keepAlive = setInterval(() => {
      res.write(": keepalive\n\n");
    }, 30000);
    this.logRequest(req, { statusCode: 200, reason: "sse_open" });
    req.on("close", () => {
      clearInterval(keepAlive);
      this.logRequest(req, { statusCode: 200, reason: "sse_closed" });
    });
  }

  handleDelete(req, res) {
    // Stateless mode: DELETE is accepted as a no-op for clients that still send session cleanup.
    const responseBytes = this.sendJson(res, 200, { ok: true });
    this.logRequest(req, { statusCode: 200, responseBytes, reason: "stateless_delete_noop" });
  }

  // --- OAuth endpoint handlers ---

  handleProtectedResourceMetadata(req, res) {
    if (req.method !== "GET") {
      this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "GET" });
      return;
    }
    const issuer = this.oauthServer.issuer;
    const body = {
      resource: `${issuer}/mcp`,
      authorization_servers: [issuer],
      scopes_supported: ["gkill:read"],
      bearer_methods_supported: ["header"],
    };
    this.sendJson(res, 200, body);
    this.logRequest(req, { statusCode: 200, reason: "oauth_protected_resource" });
  }

  handleOAuthMetadata(req, res) {
    if (req.method !== "GET") {
      this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "GET" });
      return;
    }
    const meta = this.oauthServer.getMetadata();
    this.sendJson(res, 200, meta);
    this.logRequest(req, { statusCode: 200, reason: "oauth_metadata" });
  }

  handleOAuthAuthorize(req, res, query) {
    if (req.method === "GET") {
      const result = this.oauthServer.handleAuthorizeGet(query);
      this._sendOAuthResult(req, res, result, "oauth_authorize_get");
      return;
    }
    if (req.method === "POST") {
      const chunks = [];
      req.on("data", (chunk) => chunks.push(chunk));
      req.on("end", async () => {
        try {
          const bodyStr = Buffer.concat(chunks).toString("utf8");
          const formData = Object.fromEntries(new URLSearchParams(bodyStr));
          const result = await this.oauthServer.handleAuthorizePost(formData);
          this._sendOAuthResult(req, res, result, "oauth_authorize_post");
        } catch (error) {
          process.stderr.write(`OAuth authorize error: ${String(error)}\n`);
          this.sendJson(res, 500, { error: "Internal Server Error" });
        }
      });
      return;
    }
    this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "GET, POST" });
  }

  handleOAuthToken(req, res) {
    if (req.method !== "POST") {
      this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "POST" });
      return;
    }
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", () => {
      try {
        const bodyStr = Buffer.concat(chunks).toString("utf8");
        // Token endpoint accepts both application/x-www-form-urlencoded and application/json
        let body;
        const contentType = req.headers["content-type"] || "";
        if (contentType.includes("application/json")) {
          body = JSON.parse(bodyStr);
        } else {
          body = Object.fromEntries(new URLSearchParams(bodyStr));
        }
        const result = this.oauthServer.handleTokenRequest(body);
        this.sendJson(res, result.status, result.body);
        this.logRequest(req, { statusCode: result.status, reason: "oauth_token" });
      } catch (error) {
        process.stderr.write(`OAuth token error: ${String(error)}\n`);
        this.sendJson(res, 500, { error: "server_error", error_description: "Internal Server Error" });
      }
    });
  }

  handleOAuthRegister(req, res) {
    if (req.method !== "POST") {
      this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "POST" });
      return;
    }
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", () => {
      try {
        const body = JSON.parse(Buffer.concat(chunks).toString("utf8"));
        const result = this.oauthServer.handleRegister(body);
        this.sendJson(res, result.status, result.body);
        this.logRequest(req, { statusCode: result.status, reason: "oauth_register" });
      } catch (error) {
        process.stderr.write(`OAuth register error: ${String(error)}\n`);
        this.sendJson(res, 400, { error: "invalid_client_metadata", error_description: "Invalid JSON" });
      }
    });
  }

  /** Send an OAuth result (HTML, redirect, or JSON). */
  _sendOAuthResult(req, res, result, reason) {
    if (result.redirect) {
      res.writeHead(result.status, { Location: result.redirect });
      res.end();
      this.logRequest(req, { statusCode: result.status, reason, redirect: result.redirect });
      return;
    }
    if (result.contentType === "text/html") {
      const body = result.body;
      res.writeHead(result.status, { "Content-Type": "text/html; charset=utf-8" });
      res.end(body);
      this.logRequest(req, { statusCode: result.status, reason });
      return;
    }
    this.sendJson(res, result.status, result.body);
    this.logRequest(req, { statusCode: result.status, reason });
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
    new HttpTransport(server, port, oauthServer).start();
  } else {
    server.currentUserId = client.userId || null;
    accessLog.info("server_start", {
      transport, log_level: mcpLogLevel,
    });
    new StdioTransport(server).start();
  }
}

export { GkillReadClient, McpServer, OAuthServer, HttpTransport };
