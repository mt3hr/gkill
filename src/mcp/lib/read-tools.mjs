// 読み取りツールの定義。read / readwrite の2サーバが共有する。
// 書き込み専用サーバも rep名 / 板名 / タグ名の3つだけをここから取る。
//
// 以前はサーバごとに逐語コピーされていて、同じツールの description が
// 接続先サーバによって違うという状態になっていた（locale_name の説明、
// gkill_get_mi_board_list / gkill_get_all_tag_names / gkill_get_all_rep_names の本文）。

import { FIND_QUERY_SCHEMA } from "./find-query-schema.mjs";
import {
  ENTITY_DATA_TYPE_VALUES,
  DEFAULT_KYOU_HISTORY_LIMIT,
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
      "Results are returned in reverse chronological order (newest first, by related_time; ties break by id ascending). " +
      "For counts and histograms use count_only / group_by instead of fetching records — they skip all payload construction. " +
      "calendar_start_date/calendar_end_date are both INCLUSIVE, so adjacent hand-made windows double-count the boundary day; prefer group_by. " +
      "Unknown filter values (rep_types / tags / reps / data_types typos) are reported in warnings[] instead of silently matching nothing; " +
      "canonical rep_types values come from gkill_get_rep_infos. " +
      "Every entry always carries id and rep_name (v2). limit and max_size_mb are strict caps. " +
      "Response fields: kyous[], total_count (only on cursor-less responses), returned_count, remaining_count, has_more, next_cursor, " +
      "buckets (group_by only), warnings, plugin_content (inline-content counts; present only when include_plugin_content is true).",
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
            "Opaque pagination cursor. Pass next_cursor from the previous response verbatim — do not construct or edit it " +
            "(v2 cursors are composite time+ID tokens; plain ISO-8601 datetimes from older clients are still accepted). " +
            "Responses with a cursor omit total_count (use remaining_count); the first page carries total_count.",
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
            "Deprecated (v2): entity IDs are always included now. Accepted for backward compatibility and ignored.",
          default: true,
        },
        include_rep_name: {
          type: "boolean",
          description:
            "Deprecated (v2): rep_name is always included now. Accepted for backward compatibility and ignored.",
          default: true,
        },
        count_only: {
          type: "boolean",
          default: false,
          description:
            "Return only total_count (no kyous[], no attached data, no payloads). The cheapest way to size a query " +
            "before fetching, and the right tool for building histograms with repeated narrow queries is group_by instead. " +
            "Cannot be combined with cursor.",
        },
        group_by: {
          type: "string",
          enum: ["month", "day", "week_of_day", "hour", "data_type", "rep_name", "url_domain", "file_extension"],
          description:
            "Aggregate matching entries server-side and return buckets:[{key,count}] plus total_count instead of kyous[]. " +
            "Time keys use the server's local timezone. url_domain covers only urlog entries and file_extension only idf " +
            "entries (others are excluded with a warning). At most 1000 buckets; overflow folds into \"(other)\". " +
            "This replaces manual window-splitting (which double-counts boundary days because calendar bounds are inclusive). " +
            "Cannot be combined with cursor.",
        },
        data_types: {
          type: "array",
          items: { type: "string" },
          description:
            "Allowlist of data_type strings exactly as they appear in results (e.g. [\"nlog\"], [\"mi_create\"], " +
            "plugin types like [\"claude_conversation\"]). This is how you separate Mi from MiReKyou projections and " +
            "how you filter plugin records (rep_types cannot). Unknown values produce warnings, not errors. " +
            "null/omitted = no filter, [] = match nothing.",
        },
        num_min: {
          type: "number",
          description:
            "Lower bound (inclusive) on the numeric payload value: nlog amount, kc num_value, lantana mood. " +
            "When num_min/num_max is set, entries of other kinds are excluded from results.",
        },
        num_max: {
          type: "number",
          description: "Upper bound (inclusive). See num_min.",
        },
        idf_kinds: {
          type: "array",
          items: { type: "string", enum: ["image", "video", "audio", "zip", "other"] },
          description:
            "Filter idf (file) entries by kind. When set, non-idf entries are excluded from results. " +
            "null/omitted = no filter, [] = match nothing.",
        },
        include_file_size: {
          type: "boolean",
          default: false,
          description:
            "Add file_size (bytes, from the filesystem) to idf payloads on the returned page. " +
            "Missing when the file cannot be stat-ed.",
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
    description:
      "Get GPS log points in a date range (deduplicated, newest first). Supports cursor pagination and aggregation: " +
      "use count_only to size a range, group_by:\"day\" for daily coverage buckets, and limit/cursor to page through points. " +
      "Response fields: gps_logs[], total_count (cursor-less responses), returned_count, remaining_count, has_more, next_cursor, buckets (group_by only). Read-only.",
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
        limit: {
          type: "integer",
          default: 500,
          description: "Max GPS points per page (1-5000).",
        },
        cursor: {
          type: "string",
          description: "Opaque cursor. Pass next_cursor from the previous response verbatim.",
        },
        count_only: {
          type: "boolean",
          default: false,
          description: "Return only total_count. Cannot be combined with cursor.",
        },
        group_by: {
          type: "string",
          enum: ["day"],
          description: "Return buckets:[{key:\"YYYY-MM-DD\",count}] — daily coverage instead of points. Cannot be combined with cursor.",
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
      "Note that display labels in this config may not map 1:1 to accepted rep_types query values — canonical query values come from gkill_get_rep_infos. " +
      "The full config is large (~90k chars even after UI-state stripping); prefer narrowing with fields, e.g. fields:[\"tag_struct\"].",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: {
          type: "string",
          description: "Locale, e.g. ja/en.",
        },
        fields: {
          type: "array",
          items: {
            type: "string",
            enum: ["tag_struct", "mi_board_struct", "rep_struct", "rep_type_struct", "device_struct", "kftl_template_struct", "mi_default_board", "show_tags_in_list"],
          },
          description: "Return only these fields. Default: all.",
        },
        include_ui_state: {
          type: "boolean",
          default: false,
          description:
            "When false (default), transient tree-editor keys (is_checked, indeterminate, key, seq, seq_in_parent, " +
            "is_open_default, parent_folder_id, id) are stripped from struct nodes. check_when_inited and is_force_hide are always kept.",
        },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_rep_infos",
    description:
      "List repositories with structured metadata: rep_infos[] ({rep_name, rep_type}), canonical_rep_types[] (the exact " +
      "strings query.rep_types accepts — e.g. files/images live under \"directory\", not \"idf\"), and plugins[] " +
      "({rep_name, data_type, plugin_name} — plugins are matched via query.reps or data_types, never rep_types). " +
      "Call this instead of guessing rep_types casing; ApplicationConfig display labels do not map 1:1 to query values.",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string", description: "Locale, e.g. ja/en." },
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
    name: "gkill_get_kyou_history",
    description:
      "Read every stored version of ONE entry, including versions that are soft-deleted. " +
      "gkill is append-only: an update adds a new version and a delete adds a version with is_deleted=true. " +
      "Ordinary searches (gkill_get_kyous) only ever return the latest version of entries that are not deleted, " +
      "and there is no query flag that changes that — this tool is the only way to see what an entry used to say, " +
      "or to read back something you deleted by mistake. " +
      "Requires both the id and its data_type: the lookup is per-type, and there is no safe type-agnostic fallback. " +
      "Response fields: id, data_type, latest_is_deleted (whether the newest version is deleted — check this first " +
      "when an entry has vanished from search results), version_count, returned_count, has_more, and versions[] " +
      "newest first, each with update_time, is_deleted, update_app, update_device, update_user, rep_name and the " +
      "type-specific fields. Use gkill_restore_kyou to bring back a deleted entry. " +
      "Caveat: update_time is stored at one-second resolution, so two versions written within the same second " +
      "collapse into one and a history may be missing a version.",
    inputSchema: {
      type: "object",
      properties: {
        id: {
          type: "string",
          description: "ID of the entry. Obtain from gkill_add_* responses or from gkill_get_kyous results.",
        },
        data_type: {
          type: "string",
          description: "Data type of the entry. Must match the actual type — a mismatch reports the entry as not found.",
          enum: ENTITY_DATA_TYPE_VALUES,
        },
        limit: {
          type: "integer",
          description: `Max versions to return, newest first. Default: ${DEFAULT_KYOU_HISTORY_LIMIT}. Histories are unbounded — every edit appends one.`,
          default: DEFAULT_KYOU_HISTORY_LIMIT,
        },
        locale_name: { type: "string", description: "Locale, e.g. ja/en." },
      },
      required: ["id", "data_type"],
      additionalProperties: false,
    },
  },
];
