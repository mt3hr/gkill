// 読み取りツールの定義。read / readwrite の2サーバが共有する。
// 書き込み専用サーバも rep名 / 板名 / タグ名の3つだけをここから取る。
//
// 以前はサーバごとに逐語コピーされていて、同じツールの description が
// 接続先サーバによって違うという状態になっていた（locale_name の説明、
// gkill_get_mi_board_list / gkill_get_all_tag_names / gkill_get_all_rep_names の本文）。

import { FIND_QUERY_SCHEMA } from "./find-query-schema.mjs";
import {
  ISO_DATETIME_DESC,
  DATE_ONLY_DESC,
  DEFAULT_KYOUS_LIMIT,
  DEFAULT_KYOUS_MAX_SIZE_MB,
  DEFAULT_KYOUS_INCLUDE_TIMEIS,
  MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
  DEFAULT_PLUGIN_CONTENT_FORMAT,
  DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
  MAX_INLINE_PLUGIN_CONTENT_KYOUS,
  INLINE_PLUGIN_CONTENT_TOTAL_TEXT_LENGTH,
} from "./constants.mjs";

export const READ_TOOLS = [
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
      "Use this to discover existing board names for Mi queries (query.mi_board_name), and call it before gkill_add_mi / gkill_update_mi. Any string can be used as board_name — non-existent names create new boards. " +
      "Response fields: boards[] (array of board name strings).",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
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
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
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
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
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
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
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
];
