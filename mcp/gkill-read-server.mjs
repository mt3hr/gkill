#!/usr/bin/env node

import crypto from "node:crypto";
import process from "node:process";

const SERVER_NAME = "gkill-read-mcp";
const SERVER_VERSION = "0.1.0";

const AUTH_ERROR_CODES = new Set([
  "ERR000002", // AccountNotFoundError
  "ERR000013", // AccountSessionNotFoundError
  "ERR000238", // AccountDisabledError
]);

const ISO_DATETIME_DESC = "ISO-8601 datetime string, e.g. 2026-02-25T10:30:00+09:00";

const FIND_QUERY_SCHEMA = {
  type: "object",
  description:
    "gkill find query. Omitted fields follow server defaults. Datetime fields use ISO-8601 strings.",
  properties: {
    update_cache: { type: "boolean" },
    is_deleted: { type: "boolean" },
    use_tags: { type: "boolean" },
    use_reps: { type: "boolean" },
    use_rep_types: { type: "boolean" },
    rep_types: { type: "array", items: { type: "string" } },
    use_ids: { type: "boolean" },
    use_include_id: { type: "boolean" },
    ids: { type: "array", items: { type: "string" } },
    use_words: { type: "boolean" },
    words: { type: "array", items: { type: "string" } },
    words_and: { type: "boolean" },
    not_words: { type: "array", items: { type: "string" } },
    reps: { type: "array", items: { type: "string" } },
    tags: { type: "array", items: { type: "string" } },
    hide_tags: { type: "array", items: { type: "string" } },
    tags_and: { type: "boolean" },
    use_timeis: { type: "boolean" },
    timeis_words: { type: "array", items: { type: "string" } },
    timeis_not_words: { type: "array", items: { type: "string" } },
    timeis_words_and: { type: "boolean" },
    use_timeis_tags: { type: "boolean" },
    timeis_tags: { type: "array", items: { type: "string" } },
    hide_timeis_tags: { type: "array", items: { type: "string" } },
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
      "Get kyou list by FindQuery. `query` shape follows Go find.FindQuery. If omitted, server defaults are used.",
    inputSchema: {
      type: "object",
      properties: {
        query: FIND_QUERY_SCHEMA,
        locale_name: {
          type: "string",
          description: "Locale, e.g. ja/en.",
        },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_kyou",
    description: "Get history by kyou id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        update_time: {
          type: "string",
          description: `Optional ${ISO_DATETIME_DESC}`,
        },
        rep_name: { type: "string" },
        locale_name: { type: "string" },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_kmemo",
    description: "Get kmemo history by id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        update_time: {
          type: "string",
          description: `Optional ${ISO_DATETIME_DESC}`,
        },
        rep_name: { type: "string" },
        locale_name: { type: "string" },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_kc",
    description: "Get kc history by id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        update_time: {
          type: "string",
          description: `Optional ${ISO_DATETIME_DESC}`,
        },
        rep_name: { type: "string" },
        locale_name: { type: "string" },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_urlog",
    description: "Get urlog history by id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        update_time: {
          type: "string",
          description: `Optional ${ISO_DATETIME_DESC}`,
        },
        rep_name: { type: "string" },
        locale_name: { type: "string" },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_nlog",
    description: "Get nlog history by id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        update_time: {
          type: "string",
          description: `Optional ${ISO_DATETIME_DESC}`,
        },
        rep_name: { type: "string" },
        locale_name: { type: "string" },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_timeis",
    description: "Get timeis history by id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        update_time: {
          type: "string",
          description: `Optional ${ISO_DATETIME_DESC}`,
        },
        rep_name: { type: "string" },
        locale_name: { type: "string" },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_mi",
    description: "Get mi history by id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        update_time: {
          type: "string",
          description: `Optional ${ISO_DATETIME_DESC}`,
        },
        rep_name: { type: "string" },
        locale_name: { type: "string" },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_lantana",
    description: "Get lantana history by id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        update_time: {
          type: "string",
          description: "Optional ISO-8601 timestamp.",
        },
        rep_name: { type: "string" },
        locale_name: { type: "string" },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_rekyou",
    description: "Get rekyou history by id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        update_time: {
          type: "string",
          description: "Optional ISO-8601 timestamp.",
        },
        rep_name: { type: "string" },
        locale_name: { type: "string" },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_git_commit_log",
    description: "Get git commit log history by id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        update_time: {
          type: "string",
          description: `Optional ${ISO_DATETIME_DESC}`,
        },
        locale_name: { type: "string" },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_idf_kyou",
    description: "Get idf_kyou history by id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        update_time: {
          type: "string",
          description: `Optional ${ISO_DATETIME_DESC}`,
        },
        rep_name: { type: "string" },
        locale_name: { type: "string" },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_mi_board_list",
    description: "Get mi board list. Read-only.",
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
    description: "Get all tag names. Read-only.",
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
    description: "Get all repository names. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string" },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_tags_by_target_id",
    description: "Get tags by target_id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        target_id: { type: "string" },
        locale_name: { type: "string" },
      },
      required: ["target_id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_texts_by_target_id",
    description: "Get texts by target_id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        target_id: { type: "string" },
        locale_name: { type: "string" },
      },
      required: ["target_id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_notification_histories_by_notification_id",
    description: "Get notification histories by notification id. Read-only.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        update_time: {
          type: "string",
          description: `Optional ${ISO_DATETIME_DESC}`,
        },
        locale_name: { type: "string" },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_gps_log",
    description: "Get gps logs in date range. Read-only.",
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
    let response;
    try {
      response = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(body),
      });
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

class McpServer {
  constructor(client) {
    this.client = client;
    this.buffer = Buffer.alloc(0);
    process.stdin.on("data", (chunk) => this.onData(chunk));
    process.stdin.on("error", (error) => this.logError("stdin error", error));
  }

  logError(message, error) {
    process.stderr.write(`${message}: ${String(error)}\n`);
  }

  writeMessage(message) {
    // Claude Desktop (current build) expects line-delimited JSON-RPC messages.
    // Do not use LSP "Content-Length" framing on output.
    const json = JSON.stringify(message);
    process.stdout.write(`${json}\n`);
  }

  sendResult(id, result) {
    this.writeMessage({ jsonrpc: "2.0", id, result });
  }

  sendError(id, code, message, data = undefined) {
    this.writeMessage({
      jsonrpc: "2.0",
      id,
      error: { code, message, data },
    });
  }

  toolResultText(text, isError = false) {
    return {
      content: [{ type: "text", text }],
      isError,
    };
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
          if (idx === -1) {
            continue;
          }
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
        if (this.buffer.length < totalLength) {
          return;
        }

        const bodyBuffer = this.buffer.subarray(headerEnd + 4, totalLength);
        this.buffer = this.buffer.subarray(totalLength);

        let message;
        try {
          message = JSON.parse(bodyBuffer.toString("utf8"));
        } catch (error) {
          this.logError("invalid json body", error);
          continue;
        }

        this.handleMessage(message).catch((error) => {
          this.logError("unhandled request error", error);
          if (message && Object.prototype.hasOwnProperty.call(message, "id")) {
            this.sendError(message.id, -32603, "Internal error");
          }
        });
        continue;
      }

      // NDJSON-style framing: one JSON-RPC message per line.
      const lf = this.buffer.indexOf("\n");
      if (lf === -1) {
        return;
      }
      const line = this.buffer.subarray(0, lf).toString("utf8").trim();
      this.buffer = this.buffer.subarray(lf + 1);
      if (line.length === 0) {
        continue;
      }

      let message;
      try {
        message = JSON.parse(line);
      } catch (_error) {
        // If this is not NDJSON, wait for more bytes (likely partial header/body).
        continue;
      }
      this.handleMessage(message).catch((error) => {
        this.logError("unhandled request error", error);
        if (message && Object.prototype.hasOwnProperty.call(message, "id")) {
          this.sendError(message.id, -32603, "Internal error");
        }
      });
    }
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
        return this.client.callRead("/api/get_kyous", { query: p.query || {}, locale_name: p.locale_name }, true);
      case "gkill_get_kyou":
        requireString("id");
        return this.client.callRead(
          "/api/get_kyou",
          {
            id: p.id,
            update_time: p.update_time,
            rep_name: p.rep_name,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_kmemo":
        requireString("id");
        return this.client.callRead(
          "/api/get_kmemo",
          {
            id: p.id,
            update_time: p.update_time,
            rep_name: p.rep_name,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_kc":
        requireString("id");
        return this.client.callRead(
          "/api/get_kc",
          {
            id: p.id,
            update_time: p.update_time,
            rep_name: p.rep_name,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_urlog":
        requireString("id");
        return this.client.callRead(
          "/api/get_urlog",
          {
            id: p.id,
            update_time: p.update_time,
            rep_name: p.rep_name,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_nlog":
        requireString("id");
        return this.client.callRead(
          "/api/get_nlog",
          {
            id: p.id,
            update_time: p.update_time,
            rep_name: p.rep_name,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_timeis":
        requireString("id");
        return this.client.callRead(
          "/api/get_timeis",
          {
            id: p.id,
            update_time: p.update_time,
            rep_name: p.rep_name,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_mi":
        requireString("id");
        return this.client.callRead(
          "/api/get_mi",
          {
            id: p.id,
            update_time: p.update_time,
            rep_name: p.rep_name,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_lantana":
        requireString("id");
        return this.client.callRead(
          "/api/get_lantana",
          {
            id: p.id,
            update_time: p.update_time,
            rep_name: p.rep_name,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_rekyou":
        requireString("id");
        return this.client.callRead(
          "/api/get_rekyou",
          {
            id: p.id,
            update_time: p.update_time,
            rep_name: p.rep_name,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_git_commit_log":
        requireString("id");
        return this.client.callRead(
          "/api/get_git_commit_log",
          {
            id: p.id,
            update_time: p.update_time,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_idf_kyou":
        requireString("id");
        return this.client.callRead(
          "/api/get_idf_kyou",
          {
            id: p.id,
            update_time: p.update_time,
            rep_name: p.rep_name,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_mi_board_list":
        return this.client.callRead("/api/get_mi_board_list", { locale_name: p.locale_name }, true);
      case "gkill_get_all_tag_names":
        return this.client.callRead("/api/get_all_tag_names", { locale_name: p.locale_name }, true);
      case "gkill_get_all_rep_names":
        return this.client.callRead("/api/get_all_rep_names", { locale_name: p.locale_name }, true);
      case "gkill_get_tags_by_target_id":
        requireString("target_id");
        return this.client.callRead(
          "/api/get_tags_by_id",
          {
            target_id: p.target_id,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_texts_by_target_id":
        requireString("target_id");
        return this.client.callRead(
          "/api/get_texts_by_id",
          {
            target_id: p.target_id,
            locale_name: p.locale_name,
          },
          true,
        );
      case "gkill_get_notification_histories_by_notification_id":
        requireString("id");
        return this.client.callRead(
          "/api/get_gkill_notification_histories_by_notification_id",
          {
            id: p.id,
            update_time: p.update_time,
            locale_name: p.locale_name,
          },
          true,
        );
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
      default:
        throw new GkillApiError(`Unknown tool: ${name}`);
    }
  }

  async handleMessage(message) {
    if (!message || message.jsonrpc !== "2.0" || !message.method) {
      if (message && Object.prototype.hasOwnProperty.call(message, "id")) {
        this.sendError(message.id, -32600, "Invalid Request");
      }
      return;
    }

    const hasId = Object.prototype.hasOwnProperty.call(message, "id");
    const id = message.id;
    const method = message.method;
    const params = message.params || {};

    if (method === "notifications/initialized") {
      return;
    }

    if (method === "initialize") {
      if (!hasId) {
        return;
      }
      this.sendResult(id, {
        protocolVersion: "2024-11-05",
        capabilities: {
          tools: {},
        },
        serverInfo: {
          name: SERVER_NAME,
          version: SERVER_VERSION,
        },
      });
      return;
    }

    if (method === "ping") {
      if (!hasId) {
        return;
      }
      this.sendResult(id, {});
      return;
    }

    if (method === "tools/list") {
      if (!hasId) {
        return;
      }
      this.sendResult(id, { tools: TOOLS });
      return;
    }

    if (method === "tools/call") {
      if (!hasId) {
        return;
      }
      try {
        const toolName = params.name;
        const toolArgs = params.arguments || {};
        const response = await this.handleToolCall(toolName, toolArgs);
        this.sendResult(id, this.toolResultText(JSON.stringify(response, null, 2), false));
      } catch (error) {
        const detail = error instanceof GkillApiError ? error.detail : null;
        const messageText = error instanceof Error ? error.message : "Unknown tool error";
        this.sendResult(
          id,
          this.toolResultText(
            JSON.stringify(
              {
                error: messageText,
                detail,
              },
              null,
              2,
            ),
            true,
          ),
        );
      }
      return;
    }

    if (!hasId) {
      return;
    }
    this.sendError(id, -32601, `Method not found: ${method}`);
  }
}

const client = new GkillReadClient();
const server = new McpServer(client);

process.stdin.resume();
