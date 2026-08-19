#!/usr/bin/env node

import crypto from "node:crypto";
import { readFileSync } from "node:fs";
import { dirname as _dirname, resolve as _resolvePath } from "node:path";
import process from "node:process";
import { fileURLToPath as _fileURLToPath } from "node:url";

import { GkillApiError } from "./lib/errors.mjs";
import { normalizeLocaleOnlyArgs } from "./lib/normalization.mjs";
import {
  normalizeKmemoArgs,
  normalizeUrlogArgs,
  normalizeNlogArgs,
  normalizeLantanaArgs,
  normalizeTimeIsArgs,
  normalizeMiArgs,
  normalizeKcArgs,
  normalizeTagArgs,
  normalizeTextArgs,
  normalizeKftlArgs,
  normalizeDeleteArgs,
  normalizeUpdateKmemoArgs,
  normalizeUpdateUrlogArgs,
  normalizeUpdateNlogArgs,
  normalizeUpdateLantanaArgs,
  normalizeUpdateTimeIsArgs,
  normalizeUpdateMiArgs,
  normalizeUpdateKcArgs,
  normalizeUpdateTagArgs,
  normalizeUpdateTextArgs,
} from "./lib/write-normalization.mjs";
import {
  PLUGIN_TOOLS,
  handlePluginToolCall,
  isPluginToolName,
  summarizePluginToolPayload,
} from "./lib/plugin-tools.mjs";
import { OAuthServer } from "./lib/oauth-server.mjs";
import { McpAccessLog, parseMcpLogLevel } from "./lib/access-log.mjs";
import { StdioTransport } from "./lib/stdio-transport.mjs";
import { McpServerBase } from "./lib/mcp-server-base.mjs";
import { GkillClient as GkillWriteClient } from "./lib/gkill-client.mjs";
import { READ_TOOLS } from "./lib/read-tools.mjs";
import { WRITE_TOOLS } from "./lib/write-tools.mjs";

// 書き込みサーバにも載せる読み取りツール。書き込みの前に名前を引くためだけのもの
const WRITE_SERVER_READ_TOOL_NAMES = new Set([
  "gkill_get_all_rep_names",
  "gkill_get_mi_board_list",
  "gkill_get_all_tag_names",
]);
import { HttpTransport } from "./lib/http-transport.mjs";
import { summarizeToolError } from "./lib/payload.mjs";

const _thisFile = _fileURLToPath(import.meta.url);
const _thisDir = _dirname(_thisFile);
const _pkg = JSON.parse(readFileSync(_resolvePath(_thisDir, "../../package.json"), "utf8"));

const SERVER_NAME = "gkill-write-mcp";
const SERVER_VERSION = _pkg.version;

const WRITE_APP_NAME = "gkill_mcp_write";
const WRITE_DEVICE = "mcp";

// ---------------------------------------------------------------------------
// Delete endpoint mapping
// ---------------------------------------------------------------------------

const DELETE_ENDPOINT_MAP = {
  kmemo:   { endpoint: "/api/update_kmemo",   key: "kmemo",   responseKey: "updated_kmemo" },
  urlog:   { endpoint: "/api/update_urlog",   key: "urlog",   responseKey: "updated_urlog" },
  nlog:    { endpoint: "/api/update_nlog",    key: "nlog",    responseKey: "updated_nlog" },
  lantana: { endpoint: "/api/update_lantana", key: "lantana", responseKey: "updated_lantana" },
  timeis:  { endpoint: "/api/update_timeis",  key: "timeis",  responseKey: "updated_timeis" },
  mi:      { endpoint: "/api/update_mi",      key: "mi",      responseKey: "updated_mi" },
  kc:      { endpoint: "/api/update_kc",      key: "kc",      responseKey: "updated_kc" },
  tag:     { endpoint: "/api/update_tag",     key: "tag",     responseKey: "updated_tag" },
  text:    { endpoint: "/api/update_text",    key: "text",    responseKey: "updated_text" },
};

// ---------------------------------------------------------------------------
// Get endpoint mapping (for patch-style delete)
// ---------------------------------------------------------------------------

const GET_ENDPOINT_MAP = {
  kmemo: { endpoint: "/api/get_kmemo", historiesKey: "kmemo_histories" },
  urlog: { endpoint: "/api/get_urlog", historiesKey: "urlog_histories" },
  nlog: { endpoint: "/api/get_nlog", historiesKey: "nlog_histories" },
  lantana: { endpoint: "/api/get_lantana", historiesKey: "lantana_histories" },
  timeis: { endpoint: "/api/get_timeis", historiesKey: "timeis_histories" },
  mi: { endpoint: "/api/get_mi", historiesKey: "mi_histories" },
  kc: { endpoint: "/api/get_kc", historiesKey: "kc_histories" },
  tag: { endpoint: "/api/get_tag_histories_by_tag_id", historiesKey: "tag_histories" },
  text: { endpoint: "/api/get_text_histories_by_text_id", historiesKey: "text_histories" },
};

// ---------------------------------------------------------------------------
// Tool payload summarizers
// ---------------------------------------------------------------------------

function summarizeWritePayload(name, payload) {
  const pluginSummary = summarizePluginToolPayload(name, payload);
  if (pluginSummary !== null) {
    return pluginSummary;
  }
  switch (name) {
    case "gkill_add_kmemo":
      return `Created kmemo: ${payload.added_kmemo?.id || "unknown"}`;
    case "gkill_add_urlog":
      return `Created urlog: ${payload.added_urlog?.id || "unknown"}`;
    case "gkill_add_nlog":
      return `Created nlog: ${payload.added_nlog?.id || "unknown"}`;
    case "gkill_add_lantana":
      return `Created lantana: ${payload.added_lantana?.id || "unknown"}`;
    case "gkill_add_timeis":
      return `Created timeis: ${payload.added_timeis?.id || "unknown"}`;
    case "gkill_add_mi":
      return `Created mi: ${payload.added_mi?.id || "unknown"}`;
    case "gkill_add_kc":
      return `Created kc: ${payload.added_kc?.id || "unknown"}`;
    case "gkill_add_tag":
      return `Added tag: ${payload.added_tag?.id || "unknown"}`;
    case "gkill_add_text":
      return `Added text: ${payload.added_text?.id || "unknown"}`;
    case "gkill_submit_kftl":
      return `KFTL submitted: ${Array.isArray(payload.messages) ? payload.messages.length : 0} messages.`;
    case "gkill_delete_kyou": {
      const keys = Object.keys(payload).filter((k) => k.startsWith("updated_"));
      return `Deleted (soft): ${keys.length > 0 ? keys.join(", ") : "completed"}`;
    }
    // Update tools
    case "gkill_update_kmemo":
      return `Updated kmemo: ${payload.updated_kmemo?.id || "unknown"}`;
    case "gkill_update_urlog":
      return `Updated urlog: ${payload.updated_urlog?.id || "unknown"}`;
    case "gkill_update_nlog":
      return `Updated nlog: ${payload.updated_nlog?.id || "unknown"}`;
    case "gkill_update_lantana":
      return `Updated lantana: ${payload.updated_lantana?.id || "unknown"}`;
    case "gkill_update_timeis":
      return `Updated timeis: ${payload.updated_timeis?.id || "unknown"}`;
    case "gkill_update_mi":
      return `Updated mi: ${payload.updated_mi?.id || "unknown"}`;
    case "gkill_update_kc":
      return `Updated kc: ${payload.updated_kc?.id || "unknown"}`;
    case "gkill_update_tag":
      return `Updated tag: ${payload.updated_tag?.id || "unknown"}`;
    case "gkill_update_text":
      return `Updated text: ${payload.updated_text?.id || "unknown"}`;
    // Read convenience tools
    case "gkill_get_mi_board_list":
      return `Fetched ${Array.isArray(payload.boards) ? payload.boards.length : 0} Mi boards.`;
    case "gkill_get_all_tag_names":
      return `Fetched ${Array.isArray(payload.tag_names) ? payload.tag_names.length : 0} tag names.`;
    case "gkill_get_all_rep_names":
      return `Fetched ${Array.isArray(payload.rep_names) ? payload.rep_names.length : 0} repository names.`;
    default:
      return "Tool call completed.";
  }
}

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------

const TOOLS = [
  ...WRITE_TOOLS,
  // --- Read convenience tools ---
  // 書き込みの前に rep名 / 板名 / タグ名を引けるように、読み取りツールの一部だけを載せる。
  // 定義は読み取りサーバと同じものを使う（以前は description が食い違っていた）
  ...READ_TOOLS.filter((tool) => WRITE_SERVER_READ_TOOL_NAMES.has(tool.name)),
  ...PLUGIN_TOOLS,
];

// ---------------------------------------------------------------------------
// GkillWriteClient
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// McpWriteServer: transport-independent JSON-RPC handler
// ---------------------------------------------------------------------------

class McpWriteServer extends McpServerBase {
  constructor(client, accessLog = null) {
    super(client, accessLog, { serverName: SERVER_NAME, serverVersion: SERVER_VERSION, tools: TOOLS });
  }


  buildToolResult(name, payload, isError = false) {
    const summary = isError
      ? summarizeToolError(name, payload?.error || "Unknown tool error", payload?.detail || null)
      : summarizeWritePayload(name, payload);

    let jsonText;
    if (payload !== undefined) {
      jsonText = JSON.stringify(payload, null, 2);
    }

    const result = {
      content: [{ type: "text", text: jsonText ? `${summary}\n\n${jsonText}` : summary }],
      isError,
    };
    if (payload !== undefined) {
      result.structuredContent = payload;
    }
    return result;
  }

  async handleToolCall(name, args) {
    const sid = this.currentSessionId;
    const userId = this.currentUserId || this.client.userId;

    if (isPluginToolName(name)) {
      return handlePluginToolCall(
        (pathname, body) => this.client.callApi(pathname, body, true, sid),
        name,
        args,
      );
    }

    switch (name) {
      // ----- Write tools -----
      case "gkill_add_kmemo": {
        const normalized = normalizeKmemoArgs(args);
        const now = new Date().toISOString();
        const kmemo = {
          id: crypto.randomUUID(),
          rep_name: "",
          related_time: normalized.related_time || now,
          content: normalized.content,
          data_type: "kmemo",
          create_time: now, create_app: WRITE_APP_NAME,
          create_device: WRITE_DEVICE, create_user: userId,
          update_time: now, update_app: WRITE_APP_NAME,
          update_device: WRITE_DEVICE, update_user: userId,
          is_deleted: false,
        };
        const response = await this.client.callApi(
          "/api/add_kmemo",
          { kmemo, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { added_kmemo: response.added_kmemo || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_urlog": {
        const normalized = normalizeUrlogArgs(args);
        const now = new Date().toISOString();
        const urlog = {
          id: crypto.randomUUID(),
          rep_name: "",
          related_time: normalized.related_time || now,
          url: normalized.url,
          title: normalized.title || "",
          image_base64: "",
          data_type: "urlog",
          create_time: now, create_app: WRITE_APP_NAME,
          create_device: WRITE_DEVICE, create_user: userId,
          update_time: now, update_app: WRITE_APP_NAME,
          update_device: WRITE_DEVICE, update_user: userId,
          is_deleted: false,
        };
        const response = await this.client.callApi(
          "/api/add_urlog",
          { urlog, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { added_urlog: response.added_urlog || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_nlog": {
        const normalized = normalizeNlogArgs(args);
        const now = new Date().toISOString();
        const nlog = {
          id: crypto.randomUUID(),
          rep_name: "",
          related_time: normalized.related_time || now,
          shop: normalized.shop || "",
          title: normalized.title,
          amount: normalized.amount,
          data_type: "nlog",
          create_time: now, create_app: WRITE_APP_NAME,
          create_device: WRITE_DEVICE, create_user: userId,
          update_time: now, update_app: WRITE_APP_NAME,
          update_device: WRITE_DEVICE, update_user: userId,
          is_deleted: false,
        };
        const response = await this.client.callApi(
          "/api/add_nlog",
          { nlog, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { added_nlog: response.added_nlog || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_lantana": {
        const normalized = normalizeLantanaArgs(args);
        const now = new Date().toISOString();
        const lantana = {
          id: crypto.randomUUID(),
          rep_name: "",
          related_time: normalized.related_time || now,
          mood: normalized.mood,
          data_type: "lantana",
          create_time: now, create_app: WRITE_APP_NAME,
          create_device: WRITE_DEVICE, create_user: userId,
          update_time: now, update_app: WRITE_APP_NAME,
          update_device: WRITE_DEVICE, update_user: userId,
          is_deleted: false,
        };
        const response = await this.client.callApi(
          "/api/add_lantana",
          { lantana, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { added_lantana: response.added_lantana || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_timeis": {
        const normalized = normalizeTimeIsArgs(args);
        const now = new Date().toISOString();
        const timeis = {
          id: crypto.randomUUID(),
          rep_name: "",
          title: normalized.title,
          start_time: normalized.start_time || now,
          end_time: normalized.end_time || null,
          data_type: "timeis",
          create_time: now, create_app: WRITE_APP_NAME,
          create_device: WRITE_DEVICE, create_user: userId,
          update_time: now, update_app: WRITE_APP_NAME,
          update_device: WRITE_DEVICE, update_user: userId,
          is_deleted: false,
        };
        const response = await this.client.callApi(
          "/api/add_timeis",
          { timeis, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { added_timeis: response.added_timeis || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_mi": {
        const normalized = normalizeMiArgs(args);
        const now = new Date().toISOString();
        const mi = {
          id: crypto.randomUUID(),
          rep_name: "",
          title: normalized.title,
          is_checked: normalized.is_checked,
          board_name: normalized.board_name,
          limit_time: normalized.limit_time || null,
          estimate_start_time: normalized.estimate_start_time || null,
          estimate_end_time: normalized.estimate_end_time || null,
          data_type: "mi",
          create_time: now, create_app: WRITE_APP_NAME,
          create_device: WRITE_DEVICE, create_user: userId,
          update_time: now, update_app: WRITE_APP_NAME,
          update_device: WRITE_DEVICE, update_user: userId,
          is_deleted: false,
        };
        const response = await this.client.callApi(
          "/api/add_mi",
          { mi, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { added_mi: response.added_mi || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_kc": {
        const normalized = normalizeKcArgs(args);
        const now = new Date().toISOString();
        const kc = {
          id: crypto.randomUUID(),
          rep_name: "",
          related_time: normalized.related_time || now,
          title: normalized.title,
          num_value: normalized.num_value,
          data_type: "kc",
          create_time: now, create_app: WRITE_APP_NAME,
          create_device: WRITE_DEVICE, create_user: userId,
          update_time: now, update_app: WRITE_APP_NAME,
          update_device: WRITE_DEVICE, update_user: userId,
          is_deleted: false,
        };
        const response = await this.client.callApi(
          "/api/add_kc",
          { kc, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { added_kc: response.added_kc || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_tag": {
        const normalized = normalizeTagArgs(args);
        const now = new Date().toISOString();
        const tag = {
          id: crypto.randomUUID(),
          rep_name: "",
          target_id: normalized.target_id,
          tag: normalized.tag,
          data_type: "tag",
          create_time: now, create_app: WRITE_APP_NAME,
          create_device: WRITE_DEVICE, create_user: userId,
          update_time: now, update_app: WRITE_APP_NAME,
          update_device: WRITE_DEVICE, update_user: userId,
          is_deleted: false,
        };
        const response = await this.client.callApi(
          "/api/add_tag",
          { tag, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { added_tag: response.added_tag || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_text": {
        const normalized = normalizeTextArgs(args);
        const now = new Date().toISOString();
        const text = {
          id: crypto.randomUUID(),
          rep_name: "",
          target_id: normalized.target_id,
          text: normalized.text,
          data_type: "text",
          create_time: now, create_app: WRITE_APP_NAME,
          create_device: WRITE_DEVICE, create_user: userId,
          update_time: now, update_app: WRITE_APP_NAME,
          update_device: WRITE_DEVICE, update_user: userId,
          is_deleted: false,
        };
        const response = await this.client.callApi(
          "/api/add_text",
          { text, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { added_text: response.added_text || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_submit_kftl": {
        const normalized = normalizeKftlArgs(args);
        const response = await this.client.callApi(
          "/api/submit_kftl_text",
          { kftl_text: normalized.kftl_text, locale_name: normalized.locale_name },
          true, sid,
        );
        return { messages: response.messages || [] };
      }

      case "gkill_delete_kyou": {
        const normalized = normalizeDeleteArgs(args);
        const deleteMapping = DELETE_ENDPOINT_MAP[normalized.data_type];
        const getMapping = GET_ENDPOINT_MAP[normalized.data_type];
        if (!deleteMapping || !getMapping) {
          throw new GkillApiError(`Unsupported data_type for delete: ${normalized.data_type}`);
        }
        // 1. Fetch current entity to preserve all data fields
        const getResponse = await this.client.callApi(
          getMapping.endpoint, { id: normalized.id }, true, sid,
        );
        const histories = getResponse[getMapping.historiesKey];
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Entity not found: ${normalized.id}`);
        }
        const current = histories[0];
        // 2. Set is_deleted + update metadata
        const now = new Date().toISOString();
        current.is_deleted = true;
        current.update_time = now;
        current.update_app = WRITE_APP_NAME;
        current.update_device = WRITE_DEVICE;
        current.update_user = userId;
        // 3. Send update
        const response = await this.client.callApi(
          deleteMapping.endpoint,
          { [deleteMapping.key]: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        // 4. Return current with is_deleted=true and all data preserved
        const result = {};
        result[deleteMapping.responseKey] = current;
        if (response.updated_kyou) result.updated_kyou = response.updated_kyou;
        return result;
      }

      // ----- Update tools -----
      case "gkill_update_kmemo": {
        const normalized = normalizeUpdateKmemoArgs(args);
        const getResponse = await this.client.callApi(
          "/api/get_kmemo", { id: normalized.id }, true, sid,
        );
        const histories = getResponse.kmemo_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Kmemo not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.content !== undefined) current.content = normalized.content;
        if (normalized.related_time !== undefined) current.related_time = normalized.related_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = WRITE_APP_NAME;
        current.update_device = WRITE_DEVICE;
        current.update_user = userId;
        const response = await this.client.callApi(
          "/api/update_kmemo",
          { kmemo: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { updated_kmemo: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_urlog": {
        const normalized = normalizeUpdateUrlogArgs(args);
        const getResponse = await this.client.callApi(
          "/api/get_urlog", { id: normalized.id }, true, sid,
        );
        const histories = getResponse.urlog_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Urlog not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.url !== undefined) current.url = normalized.url;
        if (normalized.title !== undefined) current.title = normalized.title;
        if (normalized.related_time !== undefined) current.related_time = normalized.related_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = WRITE_APP_NAME;
        current.update_device = WRITE_DEVICE;
        current.update_user = userId;
        const response = await this.client.callApi(
          "/api/update_urlog",
          { urlog: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { updated_urlog: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_nlog": {
        const normalized = normalizeUpdateNlogArgs(args);
        const getResponse = await this.client.callApi(
          "/api/get_nlog", { id: normalized.id }, true, sid,
        );
        const histories = getResponse.nlog_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Nlog not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.title !== undefined) current.title = normalized.title;
        if (normalized.amount !== undefined) current.amount = normalized.amount;
        if (normalized.shop !== undefined) current.shop = normalized.shop;
        if (normalized.related_time !== undefined) current.related_time = normalized.related_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = WRITE_APP_NAME;
        current.update_device = WRITE_DEVICE;
        current.update_user = userId;
        const response = await this.client.callApi(
          "/api/update_nlog",
          { nlog: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { updated_nlog: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_lantana": {
        const normalized = normalizeUpdateLantanaArgs(args);
        const getResponse = await this.client.callApi(
          "/api/get_lantana", { id: normalized.id }, true, sid,
        );
        const histories = getResponse.lantana_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Lantana not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.mood !== undefined) current.mood = normalized.mood;
        if (normalized.related_time !== undefined) current.related_time = normalized.related_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = WRITE_APP_NAME;
        current.update_device = WRITE_DEVICE;
        current.update_user = userId;
        const response = await this.client.callApi(
          "/api/update_lantana",
          { lantana: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { updated_lantana: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_timeis": {
        const normalized = normalizeUpdateTimeIsArgs(args);
        const getResponse = await this.client.callApi(
          "/api/get_timeis", { id: normalized.id }, true, sid,
        );
        const histories = getResponse.timeis_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`TimeIs not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.title !== undefined) current.title = normalized.title;
        if (normalized.start_time !== undefined) current.start_time = normalized.start_time;
        if (normalized.end_time !== undefined) current.end_time = normalized.end_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = WRITE_APP_NAME;
        current.update_device = WRITE_DEVICE;
        current.update_user = userId;
        const response = await this.client.callApi(
          "/api/update_timeis",
          { timeis: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { updated_timeis: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_mi": {
        const normalized = normalizeUpdateMiArgs(args);
        const getResponse = await this.client.callApi(
          "/api/get_mi", { id: normalized.id }, true, sid,
        );
        const histories = getResponse.mi_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Mi not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.title !== undefined) current.title = normalized.title;
        if (normalized.board_name !== undefined) current.board_name = normalized.board_name;
        if (normalized.is_checked !== undefined) current.is_checked = normalized.is_checked;
        if (normalized.limit_time !== undefined) current.limit_time = normalized.limit_time;
        if (normalized.estimate_start_time !== undefined) current.estimate_start_time = normalized.estimate_start_time;
        if (normalized.estimate_end_time !== undefined) current.estimate_end_time = normalized.estimate_end_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = WRITE_APP_NAME;
        current.update_device = WRITE_DEVICE;
        current.update_user = userId;
        const response = await this.client.callApi(
          "/api/update_mi",
          { mi: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { updated_mi: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_kc": {
        const normalized = normalizeUpdateKcArgs(args);
        const getResponse = await this.client.callApi(
          "/api/get_kc", { id: normalized.id }, true, sid,
        );
        const histories = getResponse.kc_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`KC not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.title !== undefined) current.title = normalized.title;
        if (normalized.num_value !== undefined) current.num_value = normalized.num_value;
        if (normalized.related_time !== undefined) current.related_time = normalized.related_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = WRITE_APP_NAME;
        current.update_device = WRITE_DEVICE;
        current.update_user = userId;
        const response = await this.client.callApi(
          "/api/update_kc",
          { kc: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { updated_kc: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_tag": {
        const normalized = normalizeUpdateTagArgs(args);
        const getResponse = await this.client.callApi(
          "/api/get_tag_histories_by_tag_id", { id: normalized.id }, true, sid,
        );
        const histories = getResponse.tag_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Tag not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.tag !== undefined) current.tag = normalized.tag;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = WRITE_APP_NAME;
        current.update_device = WRITE_DEVICE;
        current.update_user = userId;
        const response = await this.client.callApi(
          "/api/update_tag",
          { tag: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { updated_tag: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_text": {
        const normalized = normalizeUpdateTextArgs(args);
        const getResponse = await this.client.callApi(
          "/api/get_text_histories_by_text_id", { id: normalized.id }, true, sid,
        );
        const histories = getResponse.text_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Text not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.text !== undefined) current.text = normalized.text;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = WRITE_APP_NAME;
        current.update_device = WRITE_DEVICE;
        current.update_user = userId;
        const response = await this.client.callApi(
          "/api/update_text",
          { text: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, sid,
        );
        return { updated_text: current, updated_kyou: response.updated_kyou || null };
      }

      // ----- Read convenience tools -----
      case "gkill_get_all_rep_names": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await this.client.callApi("/api/get_all_rep_names", normalized, true, sid);
        return { rep_names: Array.isArray(response.rep_names) ? response.rep_names : [] };
      }

      case "gkill_get_mi_board_list": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await this.client.callApi("/api/get_mi_board_list", normalized, true, sid);
        return { boards: Array.isArray(response.boards) ? response.boards : [] };
      }

      case "gkill_get_all_tag_names": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await this.client.callApi("/api/get_all_tag_names", normalized, true, sid);
        return { tag_names: Array.isArray(response.tag_names) ? response.tag_names : [] };
      }

      default:
        throw new GkillApiError(`Unknown tool: ${name}`);
    }
  }

}

// ---------------------------------------------------------------------------
// StdioTransport
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// HttpTransport
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

const _isDirectRun =
  typeof process !== "undefined" &&
  process.argv[1] &&
  _resolvePath(process.argv[1]) === _thisFile;

if (_isDirectRun) {
  const client = new GkillWriteClient();

  const transport = (process.env.MCP_TRANSPORT || "stdio").toLowerCase();
  const gkillHome = process.env.GKILL_HOME || _resolvePath(process.env.HOME || process.env.USERPROFILE || ".", "gkill");
  const mcpLogLevel = parseMcpLogLevel(process.env.MCP_LOG);
  const accessLog = new McpAccessLog(
    _resolvePath(gkillHome, "logs", "gkill_mcp_write_access.log"),
    mcpLogLevel,
    "gkill-write-server.mjs",
  );

  const server = new McpWriteServer(client, accessLog);

  if (transport === "http") {
    const port = parseInt(process.env.MCP_PORT || "8809", 10);
    const issuer = process.env.MCP_OAUTH_ISSUER || `http://localhost:${port}`;
    const persistPath = _resolvePath(gkillHome, "configs", "mcp_oauth_write_state.json");
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
    new HttpTransport(server, port, oauthServer, { scope: "gkill:write", enableFileLinks: false }).start();
  } else {
    server.currentUserId = client.userId || null;
    accessLog.info("server_start", {
      transport, log_level: mcpLogLevel,
    });
    new StdioTransport(server).start();
  }
}

export { GkillWriteClient, McpWriteServer, OAuthServer };
