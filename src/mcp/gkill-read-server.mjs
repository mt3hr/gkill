#!/usr/bin/env node

import crypto from "node:crypto";
import http from "node:http";
import process from "node:process";
import { Agent } from "undici";

const SERVER_NAME = "gkill-read-mcp";
const SERVER_VERSION = "0.3.2";

const AUTH_ERROR_CODES = new Set([
  "ERR000002", // AccountNotFoundError
  "ERR000013", // AccountSessionNotFoundError
  "ERR000238", // AccountDisabledError
]);

const ISO_DATETIME_DESC = "ISO-8601 datetime string, e.g. 2026-02-25T10:30:00+09:00";

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
    calendar_start_date: { type: "string", description: ISO_DATETIME_DESC },
    calendar_end_date: { type: "string", description: ISO_DATETIME_DESC },
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
    plaing_time: { type: "string", description: ISO_DATETIME_DESC },
    use_update_time: { type: "boolean" },
    update_time: { type: "string", description: ISO_DATETIME_DESC },
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
          description: "Max number of entries to return. Default: 50.",
          default: 50,
        },
        cursor: {
          type: "string",
          description:
            `Pagination cursor. Pass the next_cursor value from the previous response to fetch the next page. ${ISO_DATETIME_DESC}`,
        },
        max_size_mb: {
          type: "number",
          description: "Max response size in MB. Default: 1.0.",
          default: 1.0,
        },
        is_include_timeis: {
          type: "boolean",
          description: "Include attached TimeIs (plaing) data for each kyou. Default: true. Useful for enrichment, but when debugging a failing query it is often simpler to retry once with this set to false first.",
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
          description: `Required ${ISO_DATETIME_DESC}`,
        },
        end_date: {
          type: "string",
          description: `Required ${ISO_DATETIME_DESC}`,
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

class GkillApiError extends Error {
  constructor(message, detail = null) {
    super(message);
    this.name = "GkillApiError";
    this.detail = detail;
  }
}

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

  toolResultText(text, isError = false) {
    return {
      content: [{ type: "text", text }],
      isError,
    };
  }

  async handleToolCall(name, args) {
    const p = args || {};
    const requireString = (key) => {
      if (typeof p[key] !== "string" || p[key].trim() === "") {
        throw new GkillApiError(`Missing required string argument: ${key}`);
      }
      return p[key];
    };

    switch (name) {
      case "gkill_get_kyous":
        return this.client.callRead(
          "/api/get_kyous_mcp",
          {
            query: p.query || {},
            locale_name: p.locale_name,
            limit: p.limit,
            cursor: p.cursor,
            max_size_mb: p.max_size_mb,
            is_include_timeis: p.is_include_timeis,
          },
          true,
        );
      case "gkill_get_mi_board_list":
        return this.client.callRead("/api/get_mi_board_list", { locale_name: p.locale_name }, true);
      case "gkill_get_all_tag_names":
        return this.client.callRead("/api/get_all_tag_names", { locale_name: p.locale_name }, true);
      case "gkill_get_all_rep_names":
        return this.client.callRead("/api/get_all_rep_names", { locale_name: p.locale_name }, true);
      case "gkill_get_gps_log":
        requireString("start_date");
        requireString("end_date");
        return this.client.callRead(
          "/api/get_gps_log",
          {
            start_date: p.start_date,
            end_date: p.end_date,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_application_config": {
        const response = await this.client.callRead(
          "/api/get_application_config",
          { locale_name: p.locale_name },
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
      if (message && Object.prototype.hasOwnProperty.call(message, "id")) {
        return { jsonrpc: "2.0", id: message.id, error: { code: -32600, message: "Invalid Request" } };
      }
      return null;
    }

    const hasId = Object.prototype.hasOwnProperty.call(message, "id");
    const id = message.id;
    const method = message.method;
    const params = message.params || {};

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
        const toolName = params.name;
        const toolArgs = params.arguments || {};
        const response = await this.handleToolCall(toolName, toolArgs);
        return { jsonrpc: "2.0", id, result: this.toolResultText(JSON.stringify(response), false) };
      } catch (error) {
        const detail = error instanceof GkillApiError ? error.detail : null;
        const messageText = error instanceof Error ? error.message : "Unknown tool error";
        return {
          jsonrpc: "2.0",
          id,
          result: this.toolResultText(JSON.stringify({ error: messageText, detail }), true),
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
      const response = await this.server.handleMessage(message);
      if (response) this.writeMessage(response);
    } catch (error) {
      this.logError("unhandled request error", error);
      if (message && Object.prototype.hasOwnProperty.call(message, "id")) {
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
    this.sessions = new Map(); // sessionId -> { createdAt }
  }

  generateSessionId() {
    return crypto.randomUUID();
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

  handleRequest(req, res) {
    const route = this.parseRoute(req);
    if (!route) {
      res.writeHead(404, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: "Not Found. Use POST /mcp" }));
      return;
    }

    if (!checkApiKey(req, route.pathKey)) {
      res.writeHead(401, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: "Unauthorized" }));
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
        res.writeHead(405, { "Content-Type": "application/json", Allow: "GET, POST, DELETE" });
        res.end(JSON.stringify({ error: "Method Not Allowed" }));
    }
  }

  handlePost(req, res) {
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", async () => {
      let message;
      try {
        message = JSON.parse(Buffer.concat(chunks).toString("utf8"));
      } catch {
        res.writeHead(400, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ jsonrpc: "2.0", id: null, error: { code: -32700, message: "Parse error" } }));
        return;
      }

      try {
        const response = await this.server.handleMessage(message);

        // If this is an initialize response, create a new session
        if (message && message.method === "initialize" && response) {
          const sessionId = this.generateSessionId();
          this.sessions.set(sessionId, { createdAt: Date.now() });
          if (response === null) {
            res.writeHead(202, { "Mcp-Session-Id": sessionId });
            res.end();
          } else {
            res.writeHead(200, { "Content-Type": "application/json", "Mcp-Session-Id": sessionId });
            res.end(JSON.stringify(response));
          }
          return;
        }

        if (response === null) {
          // Notification (no id) — acknowledge with 202
          res.writeHead(202);
          res.end();
        } else {
          res.writeHead(200, { "Content-Type": "application/json" });
          res.end(JSON.stringify(response));
        }
      } catch (error) {
        process.stderr.write(`HTTP handler error: ${String(error)}\n`);
        const id = message && Object.prototype.hasOwnProperty.call(message, "id") ? message.id : null;
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ jsonrpc: "2.0", id, error: { code: -32603, message: "Internal error" } }));
      }
    });
  }

  handleGet(req, res) {
    // SSE endpoint for server-initiated notifications.
    // Currently gkill has no server-push notifications, so just hold the connection open.
    const accept = req.headers["accept"] || "";
    if (!accept.includes("text/event-stream")) {
      res.writeHead(406, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: "Not Acceptable. Use Accept: text/event-stream" }));
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
    req.on("close", () => clearInterval(keepAlive));
  }

  handleDelete(req, res) {
    // Session termination
    const sessionId = req.headers["mcp-session-id"];
    if (sessionId && this.sessions.has(sessionId)) {
      this.sessions.delete(sessionId);
    }
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ ok: true }));
  }
}

// Entry point
const client = new GkillReadClient();
const server = new McpServer(client);

const transport = (process.env.MCP_TRANSPORT || "stdio").toLowerCase();
if (transport === "http") {
  new HttpTransport(server, parseInt(process.env.MCP_PORT || "8808", 10)).start();
} else {
  new StdioTransport(server).start();
}
