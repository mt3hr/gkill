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
import { normalizeKyouArgs, normalizeLocaleOnlyArgs, normalizeGpsArgs, normalizeIdfFileArgs } from "./lib/normalization.mjs";
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
import { OAuthServer } from "./lib/oauth-server.mjs";
import { McpAccessLog, parseMcpLogLevel } from "./lib/access-log.mjs";

const _thisFile = _fileURLToPath(import.meta.url);
const _thisDir = _dirname(_thisFile);
const _pkg = JSON.parse(readFileSync(_resolvePath(_thisDir, "../../package.json"), "utf8"));

const SERVER_NAME = "gkill-readwrite-mcp";
const SERVER_VERSION = _pkg.version;

const WRITE_APP_NAME = "gkill_mcp_readwrite";
const WRITE_DEVICE = "mcp";

const AUTH_ERROR_CODES = new Set([
  "ERR000002", // AccountNotFoundError
  "ERR000013", // AccountSessionNotFoundError
  "ERR000238", // AccountDisabledError
]);

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
// Find query schema (from read server)
// ---------------------------------------------------------------------------

const FIND_QUERY_SCHEMA = {
  type: "object",
  description:
    "gkill find query. Omitted fields follow server defaults. Datetime fields use ISO-8601 strings. " +
    "General rule: each filter group requires its use_X flag set to true to activate (e.g., use_calendar:true activates calendar_start/end_date; use_words:true activates words). Without the flag, the related fields are ignored. " +
    "Recommended filtering strategy: fetch ApplicationConfig and all tag names first, then build a visible-tag allowlist — a tag is visible when is_force_hide=false AND check_when_inited=true in ApplicationConfig tag_struct. Pass visible tags via tags/timeis_tags with use_tags/use_timeis_tags=true. For repositories, prefer checked leaf rep_types from ApplicationConfig and treat unchecked leaf rep_type leaves as inferred hidden sources. " +
    "Payload varies by data_type: kmemo body is in texts[], lantana has mood (0-10), nlog has title/shop/amount, timeis has title/start_time/end_time, mi has title/is_checked/board_name/limit_time, urlog has title/url, kc has title/num_value, idf has file_name/is_image/is_video/is_audio/rep_name/mime_type (use gkill_get_idf_file tool with rep_name and file_name to fetch actual file content), git_commit_log has commit_message.",
  properties: {
    update_cache: { type: "boolean", description: "Force cache refresh before query." },
    is_deleted: { type: "boolean", description: "Include soft-deleted entries." },
    use_tags: { type: "boolean", description: "Activate tag filtering (tags, hide_tags, tags_and)." },
    use_reps: { type: "boolean", description: "Activate repository name filtering (reps)." },
    use_rep_types: { type: "boolean", description: "Activate rep-type filtering (rep_types)." },
    rep_types: {
      type: "array",
      description:
        "Allowed rep-type names. These values are backend-specific and may be case-sensitive. Do not assume ApplicationConfig display labels map 1:1 to accepted query values. In some deployments, lower-case values such as \"kmemo\" work where title-case labels such as \"Kmemo\" do not. If unsure, omit use_rep_types first, confirm the search works, then add rep_types gradually.",
      items: { type: "string" },
    },
    use_ids: { type: "boolean", description: "Activate ID filtering (ids)." },
    use_include_id: { type: "boolean", description: "When true, ids is an include-list; when false, an exclude-list." },
    ids: { type: "array", description: "Entry IDs to include or exclude.", items: { type: "string" } },
    use_words: { type: "boolean", description: "Activate keyword filtering (words, not_words, words_and)." },
    words: { type: "array", description: "Keywords to match.", items: { type: "string" } },
    words_and: { type: "boolean", description: "AND logic for words (true=all must match, false=any)." },
    not_words: { type: "array", description: "Keywords to exclude.", items: { type: "string" } },
    reps: {
      type: "array",
      description:
        "Allowed rep names. Use this as an allowlist when you already know the visible repos to include. If rep_struct (from ApplicationConfig) is unavailable, infer hidden repos from unchecked rep_type leaves and keep this list aligned with visible sources only.",
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
    tags_and: { type: "boolean", description: "AND logic for tags (true=all must match, false=any)." },
    use_timeis: { type: "boolean", description: "Activate TimeIs keyword filtering (timeis_words, timeis_not_words)." },
    timeis_words: { type: "array", description: "Keywords to match in TimeIs titles.", items: { type: "string" } },
    timeis_not_words: { type: "array", description: "Keywords to exclude from TimeIs titles.", items: { type: "string" } },
    timeis_words_and: { type: "boolean", description: "AND logic for timeis_words." },
    use_timeis_tags: { type: "boolean", description: "Activate TimeIs tag filtering." },
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
    timeis_tags_and: { type: "boolean", description: "AND logic for timeis_tags." },
    use_calendar: { type: "boolean", description: "Activate date range filtering (calendar_start/end_date)." },
    calendar_start_date: { type: "string", description: `${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}` },
    calendar_end_date: { type: "string", description: `${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}` },
    use_map: { type: "boolean", description: "Activate geographic filtering (map_latitude, map_longitude, map_radius)." },
    map_radius: { type: "number", description: "Search radius in meters." },
    map_latitude: { type: "number", description: "Center latitude." },
    map_longitude: { type: "number", description: "Center longitude." },
    include_create_mi: { type: "boolean", description: "Include Mi tasks in 'created' state. Effective only when for_mi=true." },
    include_check_mi: { type: "boolean", description: "Include Mi tasks in 'checked' (completed) state. Effective only when for_mi=true." },
    include_limit_mi: { type: "boolean", description: "Include Mi tasks that have a deadline (limit_time). Effective only when for_mi=true." },
    include_start_mi: { type: "boolean", description: "Include Mi tasks that have an estimate_start_time. Effective only when for_mi=true." },
    include_end_mi: { type: "boolean", description: "Include Mi tasks that have an estimate_end_time. Effective only when for_mi=true." },
    include_end_timeis: { type: "boolean", description: "Include TimeIs entries that have ended (have end_time)." },
    use_plaing: { type: "boolean", description: "Activate plaing time filtering — shows what was happening at a specific moment (e.g., which TimeIs was running, which records existed). Unlike calendar range, this is a point-in-time snapshot." },
    plaing_time: { type: "string", description: `Target time for plaing view. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}` },
    use_update_time: { type: "boolean", description: "Activate update-time filtering (records updated after this time)." },
    update_time: { type: "string", description: `Filter by last update time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}` },
    is_image_only: { type: "boolean", description: "Return only entries that have images attached." },
    for_mi: { type: "boolean", description: "Query Mi (task) entries specifically." },
    use_mi_board_name: { type: "boolean", description: "Activate Mi board filtering (mi_board_name)." },
    use_period_of_time: { type: "boolean", description: "Activate time-of-day/weekday filtering." },
    period_of_time_start_time_second: {
      type: "integer",
      description: "Start of time-of-day window, seconds from 00:00:00 (0-86399).",
    },
    period_of_time_end_time_second: {
      type: "integer",
      description: "End of time-of-day window, seconds from 00:00:00 (0-86399).",
    },
    period_of_time_week_of_days: {
      type: "array",
      description: "Weekdays to include: Sunday=0 ... Saturday=6.",
      items: { type: "integer", minimum: 0, maximum: 6 },
    },
    mi_board_name: { type: "string", description: "Filter Mi tasks by board name." },
    mi_check_state: {
      type: "string",
      description: "Filter Mi tasks by check state.",
      enum: ["all", "checked", "uncheck"],
    },
    mi_sort_type: {
      type: "string",
      description: "Sort order for Mi tasks.",
      enum: ["create_time", "estimate_start_time", "estimate_end_time", "limit_time"],
    },
    only_latest_data: { type: "boolean", description: "Return only the latest version of each entry (server default: true)." },
  },
  additionalProperties: true,
};

// ---------------------------------------------------------------------------
// Tool payload summarizers (merged)
// ---------------------------------------------------------------------------

function summarizeToolPayload(name, payload) {
  switch (name) {
    // Read tools
    case "gkill_get_kyous": {
      const returnedCount = payload.returned_count ?? 0;
      const totalCount = payload.total_count ?? returnedCount;
      const remaining = totalCount - returnedCount;
      if (payload.has_more && payload.next_cursor) {
        return `Returned ${returnedCount} of ${totalCount} kyou entries (${remaining} remaining). Next page: cursor="${payload.next_cursor}".`;
      }
      return `Returned ${returnedCount} of ${totalCount} kyou entries (all results returned).`;
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
    // Write tools
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

// ---------------------------------------------------------------------------
// Tool definitions (7 read + 11 write + 9 update = 27 tools)
// ---------------------------------------------------------------------------

const TOOLS = [
  // --- Read tools ---
  {
    name: "gkill_get_kyous",
    description:
      "Search life-log entries (kyou) with optional filters and return enriched results including tags, texts, notifications, and typed payload inline. " +
      "Each result contains data_type, related_time, tags[], texts[], notifications[], timeis[] (attached TimeIs), and payload (type-specific fields). " +
      "Supports cursor-based pagination via next_cursor / cursor parameters. " +
      "Use limit and max_size_mb to control response size. " +
      "Available data_type values: kmemo (text memo), kc (numeric record), timeis (time stamp start/end), nlog (expense/income), lantana (mood 0-10), urlog (URL/bookmark), idf (file/image — use gkill_get_idf_file to fetch file content), git_commit_log (git commit), mi (task). " +
      "Most used query fields: use_calendar + calendar_start/end_date, use_words + words, use_tags + tags, for_mi. Advanced: use_map, use_plaing, use_period_of_time, use_update_time. " +
      "Common query patterns: " +
      "Date range: {use_calendar:true, calendar_start_date:\"2026-03-01\", calendar_end_date:\"2026-03-07\"}. " +
      "Keyword search: {use_words:true, words:[\"keyword\"]}. " +
      "Tag filter: {use_tags:true, tags:[\"tagname\"]}. " +
      "Mi tasks: {for_mi:true, mi_check_state:\"uncheck\"}. " +
      "Practical recommendation: start with a minimal query, keep limit small, and add filters gradually. Hidden tags can be searched intentionally by passing them directly in query.tags or query.timeis_tags. rep_types are backend-specific and may be case-sensitive, so do not assume ApplicationConfig display labels map 1:1 to accepted query values. " +
      "If a query fails, first retry with fewer query fields, a smaller limit, and is_include_timeis=false; then add rep_types or TimeIs expansion back step by step. " +
      "The server always applies only_latest_data=true. " +
      "Results are returned in reverse chronological order (newest first, by related_time). " +
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
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_mi_board_list",
    description:
      "Get the list of Mi (task) board names configured in gkill. Boards are like Kanban columns that organize tasks. " +
      "Call this before gkill_add_mi or gkill_update_mi to discover existing board names. Any string can be used as board_name — non-existent names create new boards. " +
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
    description: "Get all tag names defined in gkill. Use this to discover available tags for filtering in gkill_get_kyous via query.tags (with use_tags:true) or query.timeis_tags (with use_timeis_tags:true).",
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
    description: "Get all repository names configured in gkill. Use this to discover rep names for filtering in gkill_get_kyous via query.reps (with use_reps:true).",
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
      "Response fields: tag_struct (tag parent-child hierarchy with check_when_inited, is_force_hide, children), mi_board_struct (task board hierarchy), rep_struct (repository hierarchy), rep_type_struct (repository type hierarchy), device_struct (device hierarchy), kftl_template_struct (KFTL templates), mi_default_board (default board name, e.g. \"Inbox\"), show_tags_in_list (boolean). " +
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
      "For images, the content is returned as an MCP image content block that AI can view directly.",
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
  // --- Write tools ---
  {
    name: "gkill_add_kmemo",
    description:
      "Create a text memo (kmemo) in gkill — the most general-purpose record type for free-form text notes, diary entries, or any textual life-log data. " +
      "The repository where the memo is stored is determined automatically by the server based on user configuration. " +
      "Response fields: added_kmemo (full Kmemo entity with id, rep_name, content, related_time, create_time, etc.), added_kyou (parent Kyou wrapper with id, data_type, related_time). " +
      "Use the returned id as target_id for gkill_add_tag to categorize the memo, or gkill_add_text to attach additional annotations. " +
      "Typical workflow: create a memo with gkill_add_kmemo → tag it with gkill_add_tag using the returned id. " +
      "If related_time is omitted, defaults to the current timestamp. " +
      "For structured multi-record creation (e.g., memo + mood + expense in one shot), consider gkill_submit_kftl instead.",
    inputSchema: {
      type: "object",
      properties: {
        content: { type: "string", description: "Memo text content. Supports any free-form text including multi-line." },
        related_time: { type: "string", description: `When this memo relates to (not when it was created — that is auto-set). ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["content"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_urlog",
    description:
      "Create a bookmark/URL record (urlog) in gkill for saving web links with optional titles. " +
      "Useful for bookmarking articles, documentation, or any web resource as part of the life-log. " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_urlog (full URLog entity with id, url, title, rep_name, related_time, etc.), added_kyou (parent Kyou wrapper). " +
      "Use the returned id as target_id for gkill_add_tag or gkill_add_text to annotate the bookmark. " +
      "If title is omitted, only the URL is stored. The server does not automatically fetch page titles.",
    inputSchema: {
      type: "object",
      properties: {
        url: { type: "string", description: "Full URL to bookmark (e.g., https://example.com/article)." },
        title: { type: "string", description: "Human-readable title for the bookmark. Optional — if omitted, only the URL is stored." },
        related_time: { type: "string", description: `When this bookmark relates to. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["url"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_nlog",
    description:
      "Create an expense/income record (nlog) in gkill for tracking financial transactions. " +
      "Each record has a title (what was purchased or received), an amount (negative for expense/spending, positive for income/refund), and an optional shop name. " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_nlog (full Nlog entity with id, title, shop, amount, rep_name, related_time, etc.), added_kyou (parent Kyou wrapper). " +
      "Use the returned id as target_id for gkill_add_tag (e.g., tag with category like \"food\", \"transport\") to organize expenses.",
    inputSchema: {
      type: "object",
      properties: {
        title: { type: "string", description: "Description of the expense/income (e.g., \"lunch\", \"train ticket\", \"freelance payment\")." },
        amount: { type: "integer", description: "Monetary amount (integer only, e.g. -1500 for expense, 200 for income). Must be a valid integer — empty or non-integer values are rejected by the server." },
        shop: { type: "string", description: "Shop, store, or source name (e.g., \"Starbucks\", \"Amazon\"). Optional." },
        related_time: { type: "string", description: `When the transaction occurred. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["title", "amount"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_lantana",
    description:
      "Create a mood record (lantana) in gkill for tracking emotional state over time. " +
      "Mood is an integer from 0 (lowest/worst) to 10 (highest/best), representing a subjective self-assessment of well-being. " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_lantana (full Lantana entity with id, mood, rep_name, related_time, etc.), added_kyou (parent Kyou wrapper). " +
      "Use the returned id as target_id for gkill_add_tag or gkill_add_text to add context (e.g., tag with reason like \"exercise\", annotate with notes about why the mood is high/low). " +
      "Typical usage: record mood periodically (e.g., morning, evening) to build a mood timeline.",
    inputSchema: {
      type: "object",
      properties: {
        mood: { type: "integer", description: "Mood level: 0 (lowest) to 10 (highest). Must be an integer.", minimum: 0, maximum: 10 },
        related_time: { type: "string", description: `When this mood assessment relates to. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["mood"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_timeis",
    description:
      "Create a time interval record (timeis) in gkill for tracking what you were doing during a specific period. " +
      "Each timeis has a title (the activity label) and a start/end time range. " +
      "Omit end_time to create an ongoing (open-ended) interval — it can be closed later. " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_timeis (full TimeIs entity with id, title, start_time, end_time, rep_name, etc.), added_kyou (parent Kyou wrapper). " +
      "TimeIs records are used by gkill's plaing view to show what was happening at any given moment. " +
      "Multiple timeis can overlap (e.g., \"work\" and \"meeting\" can run simultaneously). " +
      "Use the returned id as target_id for gkill_add_tag to categorize the activity.",
    inputSchema: {
      type: "object",
      properties: {
        title: { type: "string", description: "Activity title/label (e.g., \"work\", \"meeting\", \"sleep\", \"exercise\")." },
        start_time: { type: "string", description: `When the activity started. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
        end_time: { type: "string", description: `When the activity ended. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit for an ongoing interval that hasn't ended yet.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["title"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_mi",
    description:
      "Create a task (mi) in gkill's task management system. Tasks are organized into boards (like Kanban columns). " +
      "Use gkill_get_mi_board_list to discover existing board names. board_name can be any string — a non-existent board name will be created and the task is saved under that name. If board_name is omitted, the account's default board is used automatically. " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_mi (full Mi entity with id, title, is_checked, board_name, limit_time, estimate_start_time, estimate_end_time, rep_name, etc.), added_kyou (parent Kyou wrapper). " +
      "Tasks can have optional scheduling fields: limit_time (deadline), estimate_start_time, estimate_end_time. " +
      "Use the returned id as target_id for gkill_add_tag to categorize (e.g., \"urgent\", \"bugfix\") or gkill_add_text to add detailed notes. " +
      "Typical workflow: gkill_get_mi_board_list → pick a board → gkill_add_mi → optionally tag/annotate.",
    inputSchema: {
      type: "object",
      properties: {
        title: { type: "string", description: "Task title/description. Be concise but descriptive." },
        board_name: { type: "string", description: "Board name to place the task on. Use gkill_get_mi_board_list to discover existing names. Any string is accepted — a non-existent name creates a new board. If omitted, the account's default board is used." },
        is_checked: { type: "boolean", description: "Whether the task is already completed. Default: false. Set to true to create a pre-completed task (e.g., logging past work)." },
        limit_time: { type: "string", description: `Deadline for the task. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Optional.` },
        estimate_start_time: { type: "string", description: `Estimated start time for scheduling. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Optional.` },
        estimate_end_time: { type: "string", description: `Estimated end time for scheduling. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Optional.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["title"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_kc",
    description:
      "Create a numeric record (kc) in gkill for tracking any quantitative measurement over time. " +
      "Use cases: step counts, body weight, temperature, water intake, study hours, or any custom metric. " +
      "Each record has a title (what is being measured) and a num_value (the measurement). " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_kc (full KC entity with id, title, num_value, rep_name, related_time, etc.), added_kyou (parent Kyou wrapper). " +
      "Use the returned id as target_id for gkill_add_tag to categorize (e.g., tag with \"health\", \"fitness\").",
    inputSchema: {
      type: "object",
      properties: {
        title: { type: "string", description: "What is being measured (e.g., \"steps\", \"weight\", \"temperature\", \"study hours\")." },
        num_value: { type: "number", description: "Numeric measurement value. Integer or decimal (e.g., 10000, 72.5, -3)." },
        related_time: { type: "string", description: `When this measurement was taken. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["title", "num_value"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_tag",
    description:
      "Add a tag to an existing entry in gkill. Tags are the primary way to categorize and organize life-log data. " +
      "The target_id must be the ID of an existing kyou entry — obtain this from the response of any gkill_add_* tool (e.g., added_kmemo.id, added_mi.id). " +
      "Tags are free-form strings. Use gkill_get_all_tag_names to discover existing tags and maintain consistency. " +
      "You can add multiple tags to the same entry by calling this tool multiple times with the same target_id but different tag values. " +
      "The repository for the tag is determined automatically by the server. " +
      "Response fields: added_tag (full Tag entity with id, tag, target_id, rep_name, etc.), added_kyou (parent Kyou wrapper). " +
      "Typical workflow: create an entry (e.g., gkill_add_kmemo) → use the returned id → gkill_add_tag to categorize it.",
    inputSchema: {
      type: "object",
      properties: {
        tag: { type: "string", description: "Tag name string. Free-form text (e.g., \"work\", \"personal\", \"important\", \"recipe\")." },
        target_id: { type: "string", description: "ID of the existing kyou entry to tag. Obtain from the response of gkill_add_kmemo, gkill_add_mi, or any other gkill_add_* tool." },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["tag", "target_id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_text",
    description:
      "Add a text annotation to an existing entry in gkill. Text annotations provide supplementary notes or details attached to a parent record. " +
      "Unlike tags (short labels), text annotations are for longer-form content such as descriptions, comments, or context. " +
      "The target_id must be the ID of an existing kyou entry — obtain this from the response of any gkill_add_* tool. " +
      "You can add multiple text annotations to the same entry by calling this tool multiple times. " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_text (full Text entity with id, text, target_id, rep_name, etc.), added_kyou (parent Kyou wrapper). " +
      "Typical workflow: create an entry → gkill_add_text to attach detailed notes.",
    inputSchema: {
      type: "object",
      properties: {
        text: { type: "string", description: "Text annotation content. Supports free-form text including multi-line." },
        target_id: { type: "string", description: "ID of the existing kyou entry to annotate. Obtain from the response of any gkill_add_* tool." },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["text", "target_id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_submit_kftl",
    description:
      "Submit KFTL-formatted text for batch processing. KFTL is gkill's line-based text format that creates multiple records from a single text block. " +
      "CRITICAL parsing rules: " +
      "(1) Text is split by newlines (\\n). Each line is processed independently. " +
      "(2) Prefixes MUST be on their own line with NOTHING else on that line. The prefix line and the data value MUST be on SEPARATE lines. " +
      "For example, '/mood' must be alone on one line, and '8' on the next line. '/mood 8' on one line does NOT work — it becomes a kmemo. " +
      "(3) Lines without a recognized prefix are treated as kmemo (text memo) content. Adjacent non-prefixed lines are merged into a single kmemo. " +
      "(4) To create SEPARATE records, insert a separator line (、 or ,) between them. Without separators, consecutive lines merge into one kmemo. " +
      "Supported prefix lines (must be the ENTIRE line, not part of a line): " +
      "/mi or ーみ → next line is task title, " +
      "/mood or ーら → next line is mood value (0-10), " +
      "/expense or ーん → next 3 lines: shop name, then title/description, then amount as integer (IMPORTANT: prefix is /expense, NOT /nlog), " +
      "/url or ーう → next line is URL, " +
      "/num or ーか → next line is title then value, " +
      "/start or ーた → next line is timeis start label, " +
      "/end or ーえ → end current timeis, " +
      "/timeis or ーち → timeis shorthand, " +
      "/end? or ーいえ → end timeis if exists, " +
      "/endt or ーたえ → end timeis by tag, " +
      "/endt? or ーいたえ → end timeis by tag if exists, " +
      "# or 。 → tag (attach to previous record), " +
      "? or ？ → related time, " +
      "-- or ーー → text block start/end, " +
      "! or ！ → stop processing, " +
      "(no prefix) → kmemo text content. " +
      "Separator lines: 、 or , → separate into a new entity; 、、 or ,, → separate + increment time by 1 second. " +
      "Example (creates 3 records: kmemo + mood + expense): " +
      "\"今日はいい天気だった\\n、\\n/mood\\n8\\n、\\n/expense\\nカフェ\\nアイスコーヒー\\n-500\\n!\" " +
      "Important: unlike individual gkill_add_* tools, KFTL does not return created entity IDs. If you need IDs for tagging/updating, use individual gkill_add_* tools instead. " +
      "Response fields: messages[] (server processing messages).",
    inputSchema: {
      type: "object",
      properties: {
        kftl_text: { type: "string", description: "KFTL formatted text block. Multi-line (\\n separated). CRITICAL: Each prefix (/mood, /expense, /mi, etc.) MUST be the ENTIRE line by itself — do NOT put data values on the same line as the prefix. The data goes on the NEXT line(s). Use 、 or , on its own line to separate entities." },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["kftl_text"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_delete_kyou",
    description:
      "Soft-delete an existing entry by setting is_deleted=true. The entry is not physically removed — it is marked as deleted and hidden from normal queries. " +
      "Requires the entry's ID (from a previous gkill_add_* response or from gkill_get_kyous) and its data_type. " +
      "Valid data_type values: kmemo (text memo), urlog (bookmark), nlog (expense), lantana (mood), timeis (time interval), mi (task), kc (numeric), tag, text. " +
      "The appropriate update endpoint is selected automatically based on data_type. " +
      "Response fields: updated_{data_type} (the entity with is_deleted=true), updated_kyou (parent Kyou wrapper). " +
      "Note: this is a soft-delete. The data remains in the database and can potentially be recovered by clearing the is_deleted flag. " +
      "Note: idf (file) and git_commit_log entries cannot be deleted via this tool — they are managed by the file system and git repositories respectively.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the entry to soft-delete. Obtain from gkill_add_* responses or from gkill_get_kyous." },
        data_type: {
          type: "string",
          description: "Data type of the entry to delete. Must match the actual type of the entry.",
          enum: ["kmemo", "urlog", "nlog", "lantana", "timeis", "mi", "kc", "tag", "text"],
        },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id", "data_type"],
      additionalProperties: false,
    },
  },
  // --- Update tools ---
  {
    name: "gkill_update_kmemo",
    description:
      "Update an existing text memo (kmemo) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity by ID, merges your changes, updates metadata (update_time, update_app, update_device, update_user), and sends the update to the backend. " +
      "To obtain the entity ID: use the id from a previous gkill_add_kmemo response (added_kmemo.id), or search with gkill_get_kyous (include_id:true) to find existing entries and their IDs. " +
      "Response fields: updated_kmemo (full Kmemo entity after update, with id, rep_name, content, related_time, create_time, update_time, etc.), updated_kyou (parent Kyou wrapper). " +
      "Typical workflow: gkill_get_kyous({include_id:true, query:{use_words:true, words:[\"keyword\"]}}) → find the entry → gkill_update_kmemo({id: found_id, content: \"updated text\"}).",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the kmemo to update. Obtain from gkill_add_kmemo response (added_kmemo.id) or gkill_get_kyous with include_id:true." },
        content: { type: "string", description: "New memo text content." },
        related_time: { type: "string", description: `New related time (when the memo relates to). ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id", "content"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_urlog",
    description:
      "Update an existing bookmark/URL record (urlog) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity, merges changes, and sends the update. " +
      "To obtain the entity ID: use the id from a previous gkill_add_urlog response, or search with gkill_get_kyous (include_id:true). " +
      "Response fields: updated_urlog (full URLog entity after update, with id, url, title, rep_name, related_time, etc.), updated_kyou (parent Kyou wrapper). " +
      "Use cases: correct a URL typo, add/change a title for a previously untitled bookmark, change related_time.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the urlog to update. Obtain from gkill_add_urlog response or gkill_get_kyous with include_id:true." },
        url: { type: "string", description: "New URL." },
        title: { type: "string", description: "New human-readable title. Omit to keep unchanged." },
        related_time: { type: "string", description: `New related time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id", "url"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_nlog",
    description:
      "Update an existing expense/income record (nlog) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity, merges changes, and sends the update. " +
      "To obtain the entity ID: use the id from a previous gkill_add_nlog response, or search with gkill_get_kyous (include_id:true). " +
      "Response fields: updated_nlog (full Nlog entity after update, with id, title, shop, amount, rep_name, related_time, etc.), updated_kyou (parent Kyou wrapper). " +
      "Use cases: correct an expense amount, change the shop name, update the description.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the nlog to update. Obtain from gkill_add_nlog response or gkill_get_kyous with include_id:true." },
        title: { type: "string", description: "New expense/income description." },
        amount: { type: "integer", description: "New monetary amount (integer only, e.g. -1500 for expense, 200 for income). Must be a valid integer." },
        shop: { type: "string", description: "New shop/store name. Omit to keep unchanged." },
        related_time: { type: "string", description: `New related time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id", "title", "amount"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_lantana",
    description:
      "Update an existing mood record (lantana) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity, merges changes, and sends the update. " +
      "To obtain the entity ID: use the id from a previous gkill_add_lantana response, or search with gkill_get_kyous (include_id:true). " +
      "Response fields: updated_lantana (full Lantana entity after update, with id, mood, rep_name, related_time, etc.), updated_kyou (parent Kyou wrapper). " +
      "Use cases: correct a mood value that was recorded incorrectly, adjust the related_time.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the lantana to update. Obtain from gkill_add_lantana response or gkill_get_kyous with include_id:true." },
        mood: { type: "integer", description: "New mood level: 0 (lowest) to 10 (highest). Must be an integer.", minimum: 0, maximum: 10 },
        related_time: { type: "string", description: `New related time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id", "mood"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_timeis",
    description:
      "Update an existing time interval record (timeis) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity, merges changes, and sends the update. " +
      "To obtain the entity ID: use the id from a previous gkill_add_timeis response, or search with gkill_get_kyous (include_id:true). " +
      "Response fields: updated_timeis (full TimeIs entity after update, with id, title, start_time, end_time, rep_name, etc.), updated_kyou (parent Kyou wrapper). " +
      "Common use case: close an open-ended timeis by setting end_time (e.g., gkill_update_timeis({id, end_time: \"2026-03-31T18:00:00+09:00\"})). " +
      "Also useful for: correcting start/end times, renaming an activity.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the timeis to update. Obtain from gkill_add_timeis response or gkill_get_kyous with include_id:true." },
        title: { type: "string", description: "New activity title/label." },
        start_time: { type: "string", description: `New start time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        end_time: { type: "string", description: `New end time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Set this to close an open-ended (ongoing) timeis. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id", "title"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_mi",
    description:
      "Update an existing task (mi) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity, merges changes, and sends the update. " +
      "To obtain the entity ID: use the id from a previous gkill_add_mi response, or search with gkill_get_kyous (include_id:true, query:{for_mi:true}). " +
      "Response fields: updated_mi (full Mi entity after update, with id, title, is_checked, board_name, limit_time, estimate_start_time, estimate_end_time, rep_name, etc.), updated_kyou (parent Kyou wrapper). " +
      "Common use cases: mark a task as completed (is_checked:true), move to a different board (board_name), update deadline (limit_time), rename a task. " +
      "Typical workflow: gkill_get_kyous({include_id:true, query:{for_mi:true, mi_check_state:\"uncheck\"}}) → find the task → gkill_update_mi({id, is_checked:true}).",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the mi to update. Obtain from gkill_add_mi response or gkill_get_kyous with include_id:true." },
        title: { type: "string", description: "New task title." },
        board_name: { type: "string", description: "New board name to move the task to. Any string accepted — non-existent names create new boards. Omit to keep the current board unchanged." },
        is_checked: { type: "boolean", description: "Set to true to mark as completed, false to reopen. Omit to keep unchanged." },
        limit_time: { type: "string", description: `New deadline. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        estimate_start_time: { type: "string", description: `New estimated start time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        estimate_end_time: { type: "string", description: `New estimated end time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id", "title"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_kc",
    description:
      "Update an existing numeric record (kc) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity, merges changes, and sends the update. " +
      "To obtain the entity ID: use the id from a previous gkill_add_kc response, or search with gkill_get_kyous (include_id:true). " +
      "Response fields: updated_kc (full KC entity after update, with id, title, num_value, rep_name, related_time, etc.), updated_kyou (parent Kyou wrapper). " +
      "Use cases: correct a measurement value, rename the metric title, adjust related_time.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the kc to update. Obtain from gkill_add_kc response or gkill_get_kyous with include_id:true." },
        title: { type: "string", description: "New measurement title (e.g., \"steps\", \"weight\")." },
        num_value: { type: "number", description: "New numeric value. Integer or decimal (e.g., 10000, 72.5)." },
        related_time: { type: "string", description: `New related time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id", "title", "num_value"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_tag",
    description:
      "Update an existing tag in gkill using patch semantics. Changes the tag name while keeping the tag attached to the same target entry. " +
      "The MCP server internally fetches the current tag entity via the tag history API (get_tag_histories_by_tag_id), merges the change, and sends the update. " +
      "To obtain the tag ID: use the id from a previous gkill_add_tag response (added_tag.id). Note: tags are separate entities from the entries they're attached to — each tag has its own ID distinct from the parent entry's ID. " +
      "Response fields: updated_tag (full Tag entity after update, with id, tag, target_id, rep_name, etc.), updated_kyou (parent Kyou wrapper). " +
      "Use case: rename a tag (e.g., fix a typo in a tag name, change \"wrk\" to \"work\"). To remove a tag entirely, use gkill_delete_kyou with data_type=\"tag\".",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the tag entity to update. This is the tag's own ID (added_tag.id), not the target entry's ID." },
        tag: { type: "string", description: "New tag name string." },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id", "tag"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_text",
    description:
      "Update an existing text annotation in gkill using patch semantics. Changes the text content while keeping the annotation attached to the same target entry. " +
      "The MCP server internally fetches the current text entity via the text history API (get_text_histories_by_text_id), merges the change, and sends the update. " +
      "To obtain the text ID: use the id from a previous gkill_add_text response (added_text.id). Note: text annotations are separate entities from the entries they're attached to — each has its own ID distinct from the parent entry's ID. " +
      "Response fields: updated_text (full Text entity after update, with id, text, target_id, rep_name, etc.), updated_kyou (parent Kyou wrapper). " +
      "Use case: edit a note or comment attached to an existing entry. To remove a text annotation entirely, use gkill_delete_kyou with data_type=\"text\".",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the text annotation entity to update. This is the text's own ID (added_text.id), not the target entry's ID." },
        text: { type: "string", description: "New text annotation content. Supports multi-line." },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id", "text"],
      additionalProperties: false,
    },
  },
];

// ---------------------------------------------------------------------------
// GkillClient (merged read + write)
// ---------------------------------------------------------------------------

class GkillClient {
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

  async callApi(pathname, requestBody, requiresAuth, sessionIdOverride = null) {
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

// ---------------------------------------------------------------------------
// McpServer: transport-independent JSON-RPC handler (merged read + write)
// ---------------------------------------------------------------------------

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
  }

  buildToolResult(name, payload, isError = false) {
    const summary = isError
      ? summarizeToolError(name, payload?.error || "Unknown tool error", payload?.detail || null)
      : summarizeToolPayload(name, payload);

    // For gkill_get_idf_file, exclude file_content_base64 from the text representation
    // to avoid bloating the text content (binary data is sent via image block instead).
    let jsonText;
    if (payload !== undefined) {
      if (name === "gkill_get_idf_file" && !isError && payload.file_content_base64) {
        const { file_content_base64: _file_content_base64, ...rest } = payload;
        jsonText = JSON.stringify(rest, null, 2);
      } else {
        jsonText = JSON.stringify(payload, null, 2);
      }
    }

    const result = {
      content: [{ type: "text", text: jsonText ? `${summary}\n\n${jsonText}` : summary }],
      isError,
    };
    if (
      name === "gkill_get_idf_file" &&
      !isError &&
      payload &&
      payload.is_image &&
      payload.file_content_base64
    ) {
      result.content.push({
        type: "image",
        data: payload.file_content_base64,
        mimeType: payload.mime_type,
      });
    }
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
    const sid = this.currentSessionId;
    const userId = this.currentUserId || this.client.userId;

    switch (name) {
      // ----- Read tools -----
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
          },
          true,
          sid,
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
        return {
          file_name: normalized.file_name,
          mime_type: contentType,
          file_size_bytes: buffer.length,
          is_image: contentType.startsWith("image/"),
          file_content_base64: buffer.toString("base64"),
        };
      }

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

// ---------------------------------------------------------------------------
// StdioTransport
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// HttpTransport: Streamable HTTP transport (MCP spec 2024-11-05)
// ---------------------------------------------------------------------------

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
      scopes_supported: ["gkill:readwrite"],
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

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

const _isDirectRun =
  typeof process !== "undefined" &&
  process.argv[1] &&
  _resolvePath(process.argv[1]) === _thisFile;

if (_isDirectRun) {
  const client = new GkillClient();

  const transport = (process.env.MCP_TRANSPORT || "stdio").toLowerCase();
  const gkillHome = process.env.GKILL_HOME || _resolvePath(process.env.HOME || process.env.USERPROFILE || ".", "gkill");
  const mcpLogLevel = parseMcpLogLevel(process.env.MCP_LOG);
  const accessLog = new McpAccessLog(
    _resolvePath(gkillHome, "logs", "gkill_mcp_readwrite_access.log"),
    mcpLogLevel,
    "gkill-readwrite-server.mjs",
  );

  const server = new McpServer(client, accessLog);

  if (transport === "http") {
    const port = parseInt(process.env.MCP_PORT || "8810", 10);
    const issuer = process.env.MCP_OAUTH_ISSUER || `http://localhost:${port}`;
    const persistPath = _resolvePath(gkillHome, "configs", "mcp_oauth_readwrite_state.json");
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

export { GkillClient, McpServer, OAuthServer };
