#!/usr/bin/env node

import crypto from "node:crypto";
import { readFileSync } from "node:fs";
import http from "node:http";
import { dirname as _dirname, resolve as _resolvePath } from "node:path";
import process from "node:process";
import { fileURLToPath as _fileURLToPath } from "node:url";
import { Agent } from "undici";

import { GkillApiError, isPlainObject, invalidArgument } from "./lib/errors.mjs";
import { assertTrimmedString } from "./lib/validation.mjs";
import {
  ISO_DATETIME_DESC,
  DATE_ONLY_DESC,
  DEFAULT_KYOUS_LIMIT,
  DEFAULT_KYOUS_MAX_SIZE_MB,
  DEFAULT_KYOUS_INCLUDE_TIMEIS,
} from "./lib/constants.mjs";
import { normalizeKyouArgs, normalizeLocaleOnlyArgs, normalizeGpsArgs } from "./lib/normalization.mjs";

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

const FIND_QUERY_SCHEMA = {
  type: "object",
  description:
    "gkill find query. Omitted fields follow server defaults. Datetime fields use ISO-8601 strings. Recommended filtering strategy: fetch ApplicationConfig and all tag names first, build a visible-tag allowlist by removing force-hidden or unchecked tags, then pass that allowlist via tags/timeis_tags with use_tags/use_timeis_tags=true. For repositories, prefer checked leaf rep_types from ApplicationConfig and treat unchecked leaf rep_type leaves as inferred hidden sources.",
  properties: {
    update_cache: { type: "boolean" },
    is_deleted: { type: "boolean" },
    use_tags: { type: "boolean" },
    use_reps: { type: "boolean" },
    use_rep_types: { type: "boolean" },
    rep_types: {
      type: "array",
      description:
        "Allowed rep-type names. These values are backend-specific and may be case-sensitive. Do not assume ApplicationConfig display labels map 1:1 to accepted query values. In some deployments, lower-case values such as \"kmemo\" work where title-case labels such as \"Kmemo\" do not. If unsure, omit use_rep_types first, confirm the search works, then add rep_types gradually.",
      items: { type: "string" },
    },
    use_ids: { type: "boolean" },
    use_include_id: { type: "boolean" },
    ids: { type: "array", items: { type: "string" } },
    use_words: { type: "boolean" },
    words: { type: "array", items: { type: "string" } },
    words_and: { type: "boolean" },
    not_words: { type: "array", items: { type: "string" } },
    reps: {
      type: "array",
      description:
        "Allowed rep names. Use this as an allowlist when you already know the visible repos to include. If rep_struct is unavailable, infer hidden repos from unchecked rep_type leaves and keep this list aligned with visible sources only.",
      items: { type: "string" },
    },
    tags: {
      type: "array",
      description:
        "Allowed tag names. For ordinary browsing, you may build a visible-tag allowlist from ApplicationConfig. If you intentionally need a hidden tag, you can pass it here directly with use_tags=true instead of excluding it from the query.",
      items: { type: "string" },
    },
    hide_tags: {
      type: "array",
      description:
        "Explicit tag exclusion list. Prefer a visible-tag allowlist in tags when you need to exclude hidden tags reliably.",
      items: { type: "string" },
    },
    tags_and: { type: "boolean" },
    use_timeis: { type: "boolean" },
    timeis_words: { type: "array", items: { type: "string" } },
    timeis_not_words: { type: "array", items: { type: "string" } },
    timeis_words_and: { type: "boolean" },
    use_timeis_tags: { type: "boolean" },
    timeis_tags: {
      type: "array",
      description:
        "Allowed TimeIs tag names. For ordinary browsing, you may use the same visible-tag allowlist strategy as tags. If you intentionally need a hidden tag, you can pass it here directly with use_timeis_tags=true.",
      items: { type: "string" },
    },
    hide_timeis_tags: {
      type: "array",
      description:
        "Explicit TimeIs tag exclusion list. Prefer a visible-tag allowlist in timeis_tags when you need to exclude hidden tags reliably.",
      items: { type: "string" },
    },
    timeis_tags_and: { type: "boolean" },
    use_calendar: { type: "boolean" },
    calendar_start_date: { type: "string", description: `${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}` },
    calendar_end_date: { type: "string", description: `${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}` },
    use_map: { type: "boolean" },
    map_radius: { type: "number" },
    map_latitude: { type: "number" },
    map_longitude: { type: "number" },
    include_create_mi: { type: "boolean" },
    include_check_mi: { type: "boolean" },
    include_limit_mi: { type: "boolean" },
    include_start_mi: { type: "boolean" },
    include_end_mi: { type: "boolean" },
    include_end_timeis: { type: "boolean" },
    use_plaing: { type: "boolean" },
    plaing_time: { type: "string", description: `${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}` },
    use_update_time: { type: "boolean" },
    update_time: { type: "string", description: `${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}` },
    is_image_only: { type: "boolean" },
    for_mi: { type: "boolean" },
    use_mi_board_name: { type: "boolean" },
    use_period_of_time: { type: "boolean" },
    period_of_time_start_time_second: {
      type: "integer",
      description: "Seconds from 00:00:00 (0-86399).",
    },
    period_of_time_end_time_second: {
      type: "integer",
      description: "Seconds from 00:00:00 (0-86399).",
    },
    period_of_time_week_of_days: {
      type: "array",
      description: "Weekdays: Sunday=0 ... Saturday=6",
      items: { type: "integer", minimum: 0, maximum: 6 },
    },
    mi_board_name: { type: "string" },
    mi_check_state: {
      type: "string",
      enum: ["all", "checked", "uncheck"],
    },
    mi_sort_type: {
      type: "string",
      enum: ["create_time", "estimate_start_time", "estimate_end_time", "limit_time"],
    },
    only_latest_data: { type: "boolean" },
  },
  additionalProperties: true,
};


function summarizeToolPayload(name, payload) {
  switch (name) {
    case "gkill_get_kyous": {
      const returnedCount = payload.returned_count ?? 0;
      const totalCount = payload.total_count ?? returnedCount;
      const suffix = payload.has_more && payload.next_cursor ? ` More results are available via next_cursor ${payload.next_cursor}.` : "";
      return `Returned ${returnedCount} of ${totalCount} kyou entries.${suffix}`;
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
    default:
      return "Tool call completed.";
  }
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
      "Available data_type values: kmemo, kc, timeis, nlog, lantana, urlog, idf, git_commit_log, mi. " +
      "Practical recommendation: start with a minimal query, keep limit small, and add filters gradually. Hidden tags can be searched intentionally by passing them directly in query.tags or query.timeis_tags. rep_types are backend-specific and may be case-sensitive, so do not assume ApplicationConfig display labels map 1:1 to accepted query values. " +
      "If a query fails, first retry with fewer query fields, a smaller limit, and is_include_timeis=false; then add rep_types or TimeIs expansion back step by step. " +
      "The server always applies only_latest_data=true. " +
      "Response fields: kyous[], total_count, returned_count, has_more, next_cursor.",
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
          description: `Include attached TimeIs (plaing) data for each kyou. Default: ${DEFAULT_KYOUS_INCLUDE_TIMEIS}. Useful for enrichment, but ChatGPT is usually more stable when this stays false unless you explicitly need TimeIs context.`,
          default: DEFAULT_KYOUS_INCLUDE_TIMEIS,
        },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_mi_board_list",
    description: "Get the list of Mi (task) board names. Use this to discover board names for use in Mi queries.",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string" },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_all_tag_names",
    description: "Get all tag names defined in gkill. Use this to discover available tags for filtering.",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string" },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_all_rep_names",
    description: "Get all repository names configured in gkill. Use this to discover rep names for filtering.",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string" },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_gps_log",
    description: "Get GPS log entries in a date range. Read-only.",
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
        locale_name: { type: "string" },
      },
      required: ["start_date", "end_date"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_application_config",
    description:
      "Get application configuration including tag hierarchy (parent-child relationships, default check states, force-hide settings), task board structure, repository structure, and KFTL templates. Call this before gkill_get_kyous to understand the data organization and build better queries, but note that display labels in this config may not map 1:1 to accepted rep_types query values.",
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

  async callRead(pathname, requestBody, requiresAuth) {
    const localeName = requestBody.locale_name || this.defaultLocale;
    const body = {
      ...requestBody,
      locale_name: localeName,
    };

    if (requiresAuth) {
      body.session_id = body.session_id || (await this.login());
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
}

// McpServer: transport-independent JSON-RPC handler.
// handleMessage() returns a response object (or null for notifications).
class McpServer {
  constructor(client) {
    this.client = client;
  }

  buildToolResult(name, payload, isError = false) {
    const summary = isError
      ? summarizeToolError(name, payload?.error || "Unknown tool error", payload?.detail || null)
      : summarizeToolPayload(name, payload);
    const result = {
      content: [{ type: "text", text: summary }],
      isError,
    };
    if (payload !== undefined) {
      result.structuredContent = payload;
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
          },
          true,
        );
        return {
          kyous: Array.isArray(response.kyous) ? response.kyous : [],
          total_count: response.total_count ?? 0,
          returned_count: response.returned_count ?? 0,
          has_more: Boolean(response.has_more),
          ...(response.next_cursor ? { next_cursor: response.next_cursor } : {}),
        };
      }
      case "gkill_get_mi_board_list": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await this.client.callRead("/api/get_mi_board_list", normalized, true);
        return {
          boards: Array.isArray(response.boards) ? response.boards : [],
        };
      }
      case "gkill_get_all_tag_names": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await this.client.callRead("/api/get_all_tag_names", normalized, true);
        return {
          tag_names: Array.isArray(response.tag_names) ? response.tag_names : [],
        };
      }
      case "gkill_get_all_rep_names": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await this.client.callRead("/api/get_all_rep_names", normalized, true);
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
      try {
        if (!isPlainObject(params)) {
          throw invalidArgument("params", "must be an object", params);
        }
        const toolName = assertTrimmedString(params.name, "name");
        const toolArgs = Object.prototype.hasOwnProperty.call(params, "arguments") ? params.arguments : {};
        const response = await this.handleToolCall(toolName, toolArgs);
        return { jsonrpc: "2.0", id, result: this.buildToolResult(toolName, response, false) };
      } catch (error) {
        const detail = error instanceof GkillApiError ? error.detail : null;
        const messageText = error instanceof Error ? error.message : "Unknown tool error";
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

// API key verification using timing-safe comparison.
// Accepts either:
//   1. Authorization: Bearer <key> header (Claude.ai Connectors)
//   2. Path segment: POST /mcp/<key> (ChatGPT — no-auth mode with URL-embedded key)
function checkApiKey(req, pathKey) {
  const expected = process.env.MCP_API_KEY;
  if (!expected) return false;

  // Try Authorization header first
  const auth = req.headers["authorization"] || "";
  let token = auth.startsWith("Bearer ") ? auth.slice(7) : "";

  // Fall back to path segment key
  if (!token && pathKey) {
    token = pathKey;
  }

  if (token.length !== expected.length) return false;
  try {
    return crypto.timingSafeEqual(Buffer.from(token, "utf8"), Buffer.from(expected, "utf8"));
  } catch {
    return false;
  }
}

// HttpTransport: Streamable HTTP transport (MCP spec 2024-11-05).
// Supports POST /mcp (requests), GET /mcp (SSE stream), DELETE /mcp (session end).
class HttpTransport {
  constructor(server, port) {
    this.server = server;
    this.port = port;
  }

  start() {
    if (!process.env.MCP_API_KEY) {
      process.stderr.write("ERROR: MCP_API_KEY environment variable is required for HTTP transport.\n");
      process.exit(1);
    }

    const httpServer = http.createServer((req, res) => this.handleRequest(req, res));
    httpServer.listen(this.port, "0.0.0.0", () => {
      process.stderr.write(`MCP HTTP server listening on http://0.0.0.0:${this.port}/mcp\n`);
    });
  }

  parseRoute(req) {
    const pathname = new URL(req.url, "http://localhost").pathname;
    let pathKey = null;
    if (pathname === "/mcp") {
      // key from Authorization header only
    } else if (pathname.startsWith("/mcp/")) {
      pathKey = pathname.slice("/mcp/".length);
    } else {
      return null;
    }
    return { pathname, pathKey };
  }

  logRequest(req, extra = {}) {
    const payload = {
      method: req.method,
      path: req.url,
      sessionId: req.headers["mcp-session-id"] || null,
      ...extra,
    };
    process.stderr.write(`[${new Date().toISOString()}] MCP HTTP ${JSON.stringify(payload)}\n`);
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

  handleRequest(req, res) {
    const route = this.parseRoute(req);
    if (!route) {
      this.logRequest(req, { statusCode: 404, reason: "route_not_found" });
      this.sendJson(res, 404, { error: "Not Found. Use POST /mcp" });
      return;
    }

    if (!checkApiKey(req, route.pathKey)) {
      this.logRequest(req, { statusCode: 401, reason: "unauthorized" });
      this.sendJson(res, 401, { error: "Unauthorized" });
      return;
    }

    switch (req.method) {
      case "POST":
        return this.handlePost(req, res);
      case "GET":
        return this.handleGet(req, res);
      case "DELETE":
        return this.handleDelete(req, res);
      default:
        this.logRequest(req, { statusCode: 405, reason: "method_not_allowed" });
        this.sendJson(res, 405, { error: "Method Not Allowed" }, { Allow: "GET, POST, DELETE" });
    }
  }

  handlePost(req, res) {
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
        const response = await this.server.handlePayload(payload);
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
}

// Entry point — guarded so importing this module for tests does not start a transport.
const _isDirectRun =
  typeof process !== "undefined" &&
  process.argv[1] &&
  _resolvePath(process.argv[1]) === _thisFile;

if (_isDirectRun) {
  const client = new GkillReadClient();
  const server = new McpServer(client);

  const transport = (process.env.MCP_TRANSPORT || "stdio").toLowerCase();
  if (transport === "http") {
    new HttpTransport(server, parseInt(process.env.MCP_PORT || "8808", 10)).start();
  } else {
    new StdioTransport(server).start();
  }
}

export { GkillReadClient, McpServer };
