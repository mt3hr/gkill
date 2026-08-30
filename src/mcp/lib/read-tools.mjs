// 読み取りツールの定義。read / readwrite の2サーバが共有する。
// 書き込み専用サーバも rep名 / 板名 / タグ名の3つだけをここから取る。
//
// 以前はサーバごとに逐語コピーされていて、同じツールの description が
// 接続先サーバによって違うという状態になっていた（locale_name の説明、
// gkill_get_mi_board_list / gkill_get_all_tag_names / gkill_get_all_rep_names の本文）。

import { FIND_QUERY_SCHEMA } from "./find-query-schema.mjs";
import {
  ENTITY_AND_PROJECTION_DATA_TYPE_VALUES,
  DEFAULT_KYOU_HISTORY_LIMIT,
  MAX_KYOU_HISTORY_LIMIT,
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
  DEFAULT_GPS_LIMIT,
  MAX_GPS_LIMIT,
  DEFAULT_REP_NAMES_LIMIT,
  DEFAULT_TAG_NAMES_LIMIT,
  MAX_TAG_NAMES_LIMIT,
  MAX_REP_NAMES_LIMIT,
} from "./constants.mjs";

export const READ_TOOLS = [
  {
    name: "gkill_get_kyous",
    description:
      "Search life-log entries (kyou) with optional filters and return enriched results including tags, texts, notifications, and typed payload inline. " +
      "Each result contains data_type, related_time, create_app / update_app (the app that wrote / last updated it — filter on these with the create_apps / update_apps parameters), tags[], texts[], notifications[], timeis[] (attached TimeIs), and payload (type-specific fields). " +
      "Supports cursor-based pagination via next_cursor / cursor parameters. " +
      "Use limit and max_size_mb to control response size. " +
      "Available data_type values: kmemo (text memo), kc (numeric record), nlog (expense/income), lantana (mood 0-10), urlog (URL/bookmark), idf (file/image — use gkill_get_idf_file to fetch file content), git_commit_log (git commit), rekyou (repost of another entry), " +
      "timeis_start / timeis_end (time stamp), mi_create / mi_check / mi_limit / mi_start / mi_end (task, one value per projection), mirekyou_create / mirekyou_check / mirekyou_limit / mirekyou_start / mirekyou_end (an existing entry turned into a task). Which Mi projection you see depends on query.for_mi: WITH it the data_type follows query.mi_sort_type (mi_create when unset), WITHOUT it the five collapse to one representative per record (mi_start wins, then mi_check), so a plain date search mostly shows mi_check / mi_start. mi_create still survives for tasks whose create_time falls in the window while their update_time does not, so it is rare but NOT absent — do not read its low count as proof that no task was created. Plugins add their own data_type values (e.g. claude_conversation) — list them with gkill_get_plugin_list, and set include_plugin_content:true to read their bodies in this same response. " +
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
      "buckets (group_by only), partial (true when some attached data — tags/texts/notifications/TimeIs — could not be fetched and the " +
      "returned entries are incomplete; details land in warnings), warnings (always inspect this array even when partial is false: a " +
      "repository may have failed to load, so its records are absent; do not put a repository named by that warning back into query.reps), " +
      "plugins (one description per plugin that appears in this " +
      "response — the per-entry payload carries only rep_name/plugin_name so the text is not repeated per record), " +
      "plugin_content (inline-content counts; present only when include_plugin_content is true). " +
      "Each entry also carries tag_entities[] / text_entities[] ({id, value}) alongside the plain tags[] / texts[] strings: those ids are the " +
      "annotation's OWN id, which is what gkill_update_text and gkill_delete_kyou(data_type:\"tag\"/\"text\") require — the plain string " +
      "arrays cannot be edited or deleted because they carry no id. notifications[] carries id for the same reason.",
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
          description: `Max number of entries to return (1-1000). Default: ${DEFAULT_KYOUS_LIMIT}.`,
          default: DEFAULT_KYOUS_LIMIT,
          minimum: 1,
          maximum: 1000,
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
          description: `Include attached TimeIs (plaing) data for each kyou — i.e., which TimeIs was running when each record was created. Each entry carries id, title, tags, start_time and end_time (absent while still running), so you can tell same-titled stamps apart and fetch one with query.ids. Default: ${DEFAULT_KYOUS_INCLUDE_TIMEIS}. Deleted stamps are excluded, by the same rule the search itself uses. A stamp with no end_time is still running by definition, so it covers every record after its start — an old stamp you forgot to close attaches to everything since, and that is data to clean up, not a bug. This is expensive in a way limit does not bound: every call reads the whole TimeIs history (tens of thousands of rows in a real account) and the attachment ignores query.reps / rep_types / the calendar range, so narrowing the search does not narrow what gets attached. Leave it off unless you actually need it. Note: this does NOT filter out TimeIs-type kyous from results; those always appear regardless of this flag. Only controls inline plaing attachment on other data types.`,
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
          enum: ["month", "day", "week_of_day", "hour", "data_type", "rep_name", "create_app", "update_app", "url_domain", "file_extension"],
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
            "Allowlist of data_type strings exactly as they appear in results (e.g. [\"nlog\"], " +
            "plugin types like [\"claude_conversation\"]). This is how you separate Mi from MiReKyou projections and " +
            "how you filter plugin records (rep_types cannot). Unknown values produce warnings, not errors. " +
            "Mi projections are collapsed to one representative per record unless query.for_mi=true (plus an " +
            "include_*_mi flag) is set, so filtering on [\"mi_create\"] without it returns far fewer rows than " +
            "the number of tasks actually created — a low count is NOT proof that no task was created. " +
            "The response says so in warnings. " +
            "Plugins may reuse a built-in data_type: [\"kc\"] can return step counts, account balances and " +
            "hand-entered measurements at once, because they are different repositories wearing the same type. " +
            "Add query.reps to separate them. " +
            "null/omitted = no filter, [] = match nothing.",
        },
        create_apps: {
          type: "array",
          items: { type: "string" },
          description:
            "Allowlist of the app that WROTE each entry, matched against the create_app now returned on every " +
            "result. Known values: \"gkill\" (web UI and uploads), \"gkill_kftl\" (the KFTL notepad — AND anything written through gkill_submit_kftl, which the server stamps with this same value), " +
            "\"gkill_mcp_readwrite\" / \"gkill_mcp_write\" (these MCP servers), \"urlog_bookmarklet\", " +
            "\"git\", and whatever a plugin sets. This finds records written by the gkill_add_* / gkill_update_* tools, but NOT ones written through gkill_submit_kftl: those carry \"gkill_kftl\" and are indistinguishable from hand-typed notepad entries (create_device is the server's device name for both). " +
            "null/omitted = no filter, [] = match nothing.",
        },
        update_apps: {
          type: "array",
          items: { type: "string" },
          description:
            "Same as create_apps but for the app that last UPDATED the entry (matched against update_app). " +
            "Use this to find entries edited through a particular client rather than created by it. " +
            "null/omitted = no filter, [] = match nothing.",
        },
        num_min: {
          type: "number",
          description:
            "Lower bound (inclusive) on the numeric payload value: nlog amount, kc num_value, lantana mood. " +
            "When num_min/num_max is set, entries of other kinds are excluded from results. " +
            "The comparison ignores units — yen, step counts and a 0-10 mood are all measured on the same " +
            "axis, so num_min:7 mixes them. Pair it with data_types (and query.reps for plugin-supplied kc) " +
            "whenever the number means something specific.",
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
      "Get the Mi (task) board names that are actually IN USE. Boards are like Kanban columns that organize tasks. " +
      "These are the distinct board_name values of live (latest, not deleted) Mi and MiReKyou records — not a configured list, " +
      "so an account with no tasks returns [] and a board whose last task was deleted disappears. " +
      "The default board (ApplicationConfig.mi_default_board, usually \"Inbox\") is NOT included unless a task actually sits there, " +
      "and the web UI's \"all boards\" entry is a client-side pseudo-board that never appears here. " +
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
    description:
      "Get tag names defined in gkill. Use this to discover available tags for filtering in " +
      "gkill_get_kyous via query.tags or query.timeis_tags. Accounts accumulate hundreds of tags, " +
      "so narrow with contains when you only need to confirm one exists, or to list one family " +
      "such as the autolog tags. Response fields: tag_names[], total_count (before limit), " +
      "returned_count, truncated. Only tags whose target still exists are listed.",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
        contains: {
          type: "string",
          description:
            "Case-insensitive substring filter on the tag name. Omit for no filter. " +
            "Matching is done on the full list, so total_count reflects the filter.",
        },
        limit: {
          type: "integer",
          default: DEFAULT_TAG_NAMES_LIMIT,
          minimum: 1,
          maximum: MAX_TAG_NAMES_LIMIT,
          description:
            `Max names to return after filtering (1-${MAX_TAG_NAMES_LIMIT}). Default: ${DEFAULT_TAG_NAMES_LIMIT}. ` +
            "When more matched, truncated is true and total_count tells you how many there were.",
        },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_all_rep_names",
    description:
      "Get repository names configured in gkill. Use this to discover rep names for filtering in " +
      "gkill_get_kyous via query.reps. Deployments can hold hundreds of repositories, so narrow with " +
      "contains when you only need to confirm one name exists. Response fields: rep_names[], total_count " +
      "(before limit), returned_count, truncated. " +
      "Only repositories that supply kyou appear here — a plugin that emits no kyou (for example a " +
      "GPS-only plugin) is absent by design, and its manifest rep_name is not a query.reps value. " +
      "For canonical rep_types values, per-repository index freshness and where attached data lives, " +
      "call gkill_get_rep_infos instead.",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
        contains: {
          type: "string",
          description:
            "Case-insensitive substring filter on the repository name. Omit for no filter. " +
            "Matching is done on the full list, so total_count reflects the filter.",
        },
        limit: {
          type: "integer",
          default: DEFAULT_REP_NAMES_LIMIT,
          minimum: 1,
          maximum: MAX_REP_NAMES_LIMIT,
          description:
            `Max names to return after filtering (1-${MAX_REP_NAMES_LIMIT}). Default: ${DEFAULT_REP_NAMES_LIMIT}. ` +
            "When more matched, truncated is true and total_count tells you how many there were.",
        },
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
          default: DEFAULT_GPS_LIMIT,
          minimum: 1,
          maximum: MAX_GPS_LIMIT,
          description: `Max GPS points per page (1-${MAX_GPS_LIMIT}). Default: ${DEFAULT_GPS_LIMIT}.`,
        },
        cursor: {
          type: "string",
          description:
            "Opaque pagination cursor. Pass next_cursor from the previous response verbatim — do not " +
            "construct or edit it. It is not an ISO-8601 datetime, and it is not interchangeable with " +
            "the gkill_get_kyous cursor.",
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
      "It also answers WHICH ACCOUNT this MCP server is connected to: user_id and device. " +
      "Different gkill MCP servers (read / write / readwrite) can be configured against different accounts, so a record " +
      "written through one may be invisible to another — check user_id before concluding that a search or an id lookup is broken. " +
      "fields:[\"user_id\",\"device\"] is the cheap way to ask. " +
      "Recommended first call: use this before gkill_get_kyous to understand the data organization, visible tags, and board names. " +
      "Response fields: tag_struct (tag parent-child hierarchy with check_when_inited, is_force_hide, children), mi_board_struct (task board hierarchy), rep_struct (repository hierarchy — this is the tree the web settings screen saves, so it is null until someone has pressed Apply there at least once; an account used only through MCP or the CLI will always see null, and that is not an error. For the actual list of repositories, call gkill_get_rep_infos instead), rep_type_struct (repository type hierarchy), device_struct (device hierarchy), kftl_template_struct (KFTL templates), mi_default_board (default board name, e.g. \"Inbox\"), show_tags_in_list (boolean). " +
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
            enum: ["user_id", "device", "tag_struct", "mi_board_struct", "rep_struct", "rep_type_struct", "device_struct", "kftl_template_struct", "mi_default_board", "show_tags_in_list"],
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
      "List repositories with structured metadata: rep_infos[] ({rep_name, rep_type, and indexed_at for repositories " +
      "that keep an index — when that index was last refreshed. Files dropped into a repository directory do not " +
      "appear in searches until the cache is updated, and nothing warns you, so a stale indexed_at is the reason " +
      "a file you know you added comes back as zero hits: pass query.update_cache=true or run the update_cache CLI), " +
      "Each rep_infos[] entry also carries use_to_write: whether that repository is the write target for " +
      "its type. A repository can be listed and searchable yet not writable, and then every write of that " +
      "type fails with a message that does not say why (the KFTL ~~ task-from-record line is the usual " +
      "victim). Check it before writing, not after. " +
      "canonical_rep_types[] (the exact " +
      "strings query.rep_types accepts — e.g. files/images live under \"directory\", not \"idf\". It is the " +
      "vocabulary, not an inventory: every value is listed whether or not this account has such a repository, " +
      "so filtering by one of them and getting zero hits is not an anomaly — check rep_infos[] for what exists here), and plugins[] " +
      "({rep_name, data_type, plugin_name} — plugins are matched via query.reps or data_types, never rep_types). " +
      "plugins[] lists only plugins that actually supply kyou: one that emits none (a GPS-only plugin, say) is " +
      "absent here by design, because its manifest rep_name would silently match nothing — look for it in " +
      "attached_data_reps[] instead, and call gkill_get_plugin_list to see every plugin with its emits_kyou/provides. " +
      "Call this instead of guessing rep_types casing; ApplicationConfig display labels do not map 1:1 to query values. " +
      "Also returns attached_data_reps[] ({rep_name, data_kind}) — where tags, texts, notifications and GPS logs are " +
      "stored. On the ReadWrite server that answers \"where does gkill_add_tag write?\" before you write, which " +
      "nothing else could (the Write-only server does not carry this tool). " +
      "IMPORTANT: these are NOT query.reps values. They hold attached data, not kyou entries, so passing one to " +
      "query.reps matches no kyou and silently returns zero results. Use rep_infos[] for filtering and " +
      "attached_data_reps[] only to know where attached data lives. " +
      "rep_infos[] alone can run to several hundred entries (a repository appears once per rep_type it supplies), " +
      "so narrow with fields when you only need the vocabulary and the lookup tables.",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string", description: "Locale, e.g. ja/en." },
        fields: {
          type: "array",
          items: {
            type: "string",
            enum: ["rep_infos", "canonical_rep_types", "plugins", "attached_data_reps"],
          },
          description:
            "Return only these top-level fields. Default: all. " +
            "fields:[\"canonical_rep_types\",\"plugins\",\"attached_data_reps\"] skips the large rep_infos[] list.",
        },
        data_kinds: {
          type: "array",
          items: {
            type: "string",
            enum: ["tag", "text", "notification", "gpslog"],
          },
          description:
            "Narrow attached_data_reps[] to these data_kind values. Omit for all of them. " +
            "That list carries one entry per repository per kind, so an account with a long device history " +
            "runs to a hundred or more entries even though only one kind is usually wanted " +
            "(e.g. data_kinds:[\"tag\"] to see where gkill_add_tag writes). " +
            "Unknown values are rejected rather than silently matching nothing. " +
            "Has no effect on rep_infos[] / canonical_rep_types[] / plugins[].",
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
      "For images the content also comes back as an MCP image content block. That block is what puts the " +
      "picture in front of the model and lets it be used as a reference image for image generation, and " +
      "this tool is its only producer. " +
      "The payload's 'file_url' is a link to hand a human (paste it in a reply, open it in a browser, " +
      "embed it in HTML): MCP never fetches it for you, and a client that can fetch URLs on its own still " +
      "ends up with bytes outside the conversation rather than a picture it can look at. " +
      "On stdio clients the payload carries 'file_path' instead; reading that from the filesystem avoids " +
      "base64 and has no size cap, so prefer it whenever it is present. " +
      "This tool is capped by GKILL_MCP_MAX_FILE_BYTES (default 8MB); pass thumb to stay under it " +
      "(and is_video:true alongside thumb to grab a frame out of a video). " +
      "Response fields: file_name, mime_type, file_size_bytes, is_image, thumb (echoed back only when a downscaled " +
      "version was returned — its absence means you got the original), and file_content_base64 (the file body; " +
      "for images it is also delivered as the MCP image content block).",
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
        thumb: {
          type: "string",
          description:
            'Return a downscaled JPEG instead of the original, sized "<width>x<height>" (e.g. "1024x1024"). ' +
            "At most 1024 per side. Reach for this when you only need to look at the picture, or when the " +
            "original exceeds the size cap. Files that are neither images nor (with is_video) videos come " +
            "back unchanged.",
        },
        is_video: {
          type: "boolean",
          description:
            "Extract a frame from a video and return that as the thumbnail. Requires thumb. Without it a " +
            "video is fetched whole, which normally blows the size cap.",
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
      "Ordinary searches (gkill_get_kyous) only ever return the LATEST version of an entry, and by default only " +
      "entries that are not deleted. query.include_deleted_data:true does surface deleted entries there, but still " +
      "only their latest version — no query flag reaches earlier versions. This tool is the only way to see what an " +
      "entry used to say, or to read back something you deleted by mistake. " +
      "Requires both the id and its data_type: the lookup is per-type, and there is no safe type-agnostic fallback. " +
      "Response fields: id, data_type, latest_is_deleted (whether the newest version is deleted — check this first " +
      "when an entry has vanished from search results), version_count, returned_count, has_more, and versions[] " +
      "newest first, each with update_time, is_deleted, update_app, update_device, update_user, rep_name and the " +
      "type-specific fields. On a server that exposes write tools, gkill_restore_kyou brings a deleted entry back " +
      "(this read-only server does not have it). " +
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
          description:
            "Data type of the entry. Must match the actual type — a mismatch reports the entry as not found. " +
            "Two vocabularies exist: search results and add_* / update_* responses carry PROJECTION names (mi_create / mi_check / mi_limit / mi_start / mi_end, mirekyou_*, timeis_start / timeis_end), while this parameter is the ENTITY type (mi / mirekyou / timeis). Projection names are accepted here and folded to their entity type, so a data_type copied straight out of a response works.",
          enum: ENTITY_AND_PROJECTION_DATA_TYPE_VALUES,
        },
        limit: {
          type: "integer",
          description: `Max versions to return, newest first (1-${MAX_KYOU_HISTORY_LIMIT}). Default: ${DEFAULT_KYOU_HISTORY_LIMIT}. Histories are unbounded — every edit appends one.`,
          default: DEFAULT_KYOU_HISTORY_LIMIT,
          minimum: 1,
          maximum: MAX_KYOU_HISTORY_LIMIT,
        },
        locale_name: { type: "string", description: "Locale, e.g. ja/en." },
      },
      required: ["id", "data_type"],
      additionalProperties: false,
    },
  },
];
