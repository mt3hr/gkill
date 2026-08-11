// Shared FindQuery JSON schema for gkill_get_kyous.
//
// read / readwrite の2サーバで同一定義を使う。フィルタの活性化は
// 「値フィールドが非nullで存在すること」で決まり、旧 use_X フラグは廃止済み
// (後方互換の受理変換は lib/normalization.mjs の normalizeKyouQuery が行う)。

import { ISO_DATETIME_DESC, DATE_ONLY_DESC } from "./constants.mjs";

export const FIND_QUERY_SCHEMA = {
  type: "object",
  description:
    "gkill find query. Omitted fields follow server defaults. Datetime fields use ISO-8601 strings. " +
    "A filter group activates when its value field is present and non-null. Omit (or pass null for) fields you don't filter by. An empty array [] means 'filter enabled but matches nothing' (except timeis_words: [] which means 'only Kyous covered by any TimeIs'). Legacy use_X boolean flags are deprecated: accepted for backward compatibility (use_X:false removes that group's values; use_X:true is dropped). " +
    "Recommended filtering strategy: fetch ApplicationConfig and all tag names first, then build a visible-tag allowlist — a tag is visible when is_force_hide=false AND check_when_inited=true in ApplicationConfig tag_struct. Pass visible tags via tags/timeis_tags. For repositories, prefer checked leaf rep_types from ApplicationConfig and treat unchecked leaf rep_type leaves as inferred hidden sources. " +
    "Payload varies by data_type, and data_type is finer-grained than payload.kind: Mi, MiReKyou and TimeIs each surface under several data_type values (one per projection) that all share a single payload.kind. Mapping — " +
    "kmemo -> kind 'kmemo' (content; note texts[] is a separate list of annotations attached to the entry, not its body), " +
    "kc -> 'kc' (title, num_value), lantana -> 'lantana' (mood 0-10), nlog -> 'nlog' (title, shop, amount), urlog -> 'urlog' (title, url, description), git_commit_log -> 'git_commit_log' (commit_message, addition, deletion), " +
    "idf -> 'idf' (file_name, is_image, is_video, is_audio, is_zip, rep_name, mime_type), " +
    "timeis_start / timeis_end -> 'timeis' (title, start_time, end_time), " +
    "mi_create / mi_check / mi_limit / mi_start / mi_end -> 'mi' (title, is_checked, board_name, create_time, limit_time, estimate_start_time, estimate_end_time), " +
    "mirekyou_create / mirekyou_check / mirekyou_limit / mirekyou_start / mirekyou_end -> 'mirekyou' (an existing entry turned into a task: target_id, is_checked, board_name, create_time and the same scheduling fields — it has no title of its own, so pass target_id to query.ids to read the entry it points at), " +
    "rekyou -> 'rekyou' (a repost: target_id only; read the original the same way). " +
    "Which mi_* / mirekyou_* projection you get follows query.mi_sort_type, and related_time changes with it (mi_limit -> the deadline, mi_start -> the estimated start, ...), so take create_time from the payload when you need the creation time. " +
    "To view/read an idf file, prefer in this order: (1) file_path in the payload — read it directly from the local filesystem (local clients only); (2) file_url in the payload — fetch that URL to get the bytes, no auth needed, works for any size (images: file_url is a downscaled thumbnail, file_url_full is the original); (3) gkill_get_idf_file tool with rep_name and file_name — base64 fallback, capped in size. " +
    "Plugin-provided entries (any data_type that is not one of the built-ins above) have payload.kind='plugin' carrying data_type/rep_name/kyou_id/plugin_name; their body is not stored in gkill, so set include_plugin_content:true on this same call to get it inline as payload.content_text.",
  properties: {
    update_cache: { type: "boolean", description: "Force cache refresh before query." },
    is_deleted: { type: "boolean", description: "Include soft-deleted entries." },
    rep_types: {
      type: "array",
      description:
        "Allowed rep-type names; omit or pass null for no rep-type filter, [] matches nothing. These values are backend-specific and may be case-sensitive. Do not assume ApplicationConfig display labels map 1:1 to accepted query values. In some deployments, lower-case values such as \"kmemo\" work where title-case labels such as \"Kmemo\" do not. If unsure, omit rep_types first, confirm the search works, then add it gradually.",
      items: { type: "string" },
    },
    ids: {
      type: "array",
      description:
        "Entry IDs to include (an include-list: only these entries are returned). Omit or pass null for no ID filter, [] matches nothing.",
      items: { type: "string" },
    },
    words: {
      type: "array",
      description: "Keywords to match; omit or pass null for no keyword filter, [] matches nothing.",
      items: { type: "string" },
    },
    words_and: { type: "boolean", description: "AND logic for words (true=all must match, false=any)." },
    not_words: { type: "array", description: "Keywords to exclude.", items: { type: "string" } },
    reps: {
      type: "array",
      description:
        "Allowed rep names; omit or pass null for no rep filter, [] matches nothing. Use this as an allowlist when you already know the visible repos to include. If rep_struct (from ApplicationConfig) is unavailable, infer hidden repos from unchecked rep_type leaves and keep this list aligned with visible sources only.",
      items: { type: "string" },
    },
    tags: {
      type: "array",
      description:
        "Allowed tag names; omit or pass null for no tag filter, [] matches nothing. For ordinary browsing, you may build a visible-tag allowlist from ApplicationConfig. If you intentionally need a hidden tag, you can pass it here directly instead of excluding it from the query.",
      items: { type: "string" },
    },
    hide_tags: {
      type: "array",
      description:
        "Explicit tag exclusion list. Prefer a visible-tag allowlist in tags when you need to exclude hidden tags reliably.",
      items: { type: "string" },
    },
    tags_and: { type: "boolean", description: "AND logic for tags (true=all must match, false=any)." },
    timeis_words: {
      type: "array",
      description:
        "Keywords to match in TimeIs titles. [] means 'only Kyous covered by any TimeIs' (no keyword constraint). Omit or pass null (with timeis_not_words also absent) for no TimeIs filter.",
      items: { type: "string" },
    },
    timeis_not_words: { type: "array", description: "Keywords to exclude from TimeIs titles.", items: { type: "string" } },
    timeis_words_and: { type: "boolean", description: "AND logic for timeis_words." },
    timeis_tags: {
      type: "array",
      description:
        "Allowed TimeIs tag names; omit or pass null for no TimeIs tag filter, [] matches nothing. When set without timeis_words/timeis_not_words, timeis_words: [] is auto-added so the TimeIs filter activates. For ordinary browsing, you may use the same visible-tag allowlist strategy as tags. If you intentionally need a hidden tag, you can pass it here directly.",
      items: { type: "string" },
    },
    hide_timeis_tags: {
      type: "array",
      description:
        "Explicit TimeIs tag exclusion list. Prefer a visible-tag allowlist in timeis_tags when you need to exclude hidden tags reliably.",
      items: { type: "string" },
    },
    timeis_tags_and: { type: "boolean", description: "AND logic for timeis_tags." },
    calendar_start_date: {
      type: "string",
      description: `Start of the date-range filter; set to activate. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}`,
    },
    calendar_end_date: {
      type: "string",
      description: `End of the date-range filter; set to activate. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}`,
    },
    map_radius: {
      type: "number",
      description:
        "Search radius in meters. The map filter activates only when map_latitude, map_longitude and map_radius are all set.",
    },
    map_latitude: { type: "number", description: "Center latitude. See map_radius for activation." },
    map_longitude: { type: "number", description: "Center longitude. See map_radius for activation." },
    // 5つのinclude_*_miはMiのSQL射影そのものを選ぶスイッチで、既定(全false)では
    // 検索が0件になる。「絞り込み」ではなく「行の供給源」なので、
    // for_mi=trueのときは最低1つtrueにしないと何も返らないことを明記する。
    include_create_mi: { type: "boolean", description: "Include the created-time projection of Mi tasks (data_type mi_create). Effective only when for_mi=true. NOTE: the five include_*_mi flags select which projections supply rows, they do not narrow an existing result set — with all five false (the default) a for_mi search returns zero entries. Set at least one; include_create_mi:true is the usual starting point." },
    include_check_mi: { type: "boolean", description: "Include the checked-time projection of Mi tasks (data_type mi_check). Effective only when for_mi=true. See include_create_mi for why at least one of the five must be set." },
    include_limit_mi: { type: "boolean", description: "Include the deadline projection of Mi tasks — only tasks that have a limit_time (data_type mi_limit). Effective only when for_mi=true. See include_create_mi for why at least one of the five must be set." },
    include_start_mi: { type: "boolean", description: "Include the estimated-start projection of Mi tasks — only tasks that have an estimate_start_time (data_type mi_start). Effective only when for_mi=true. See include_create_mi for why at least one of the five must be set." },
    include_end_mi: { type: "boolean", description: "Include the estimated-end projection of Mi tasks — only tasks that have an estimate_end_time (data_type mi_end). Effective only when for_mi=true. See include_create_mi for why at least one of the five must be set." },
    include_end_timeis: { type: "boolean", description: "Include TimeIs entries that have ended (have end_time)." },
    plaing_time: {
      type: "string",
      description:
        "Set to search TimeIs entries running at that moment — a point-in-time snapshot of what was happening, unlike calendar range. " +
        `Accepts the literal "now" for the current time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}`,
    },
    update_time: {
      type: "string",
      description: `Filter by last update time (records updated after this time); set to activate. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}`,
    },
    is_image_only: { type: "boolean", description: "Return only entries that have images attached." },
    for_mi: { type: "boolean", description: "Query Mi (task) entries specifically. Requires at least one of include_create_mi / include_check_mi / include_limit_mi / include_start_mi / include_end_mi to be true, otherwise the search returns zero entries." },
    period_of_time_start_time_second: {
      type: "integer",
      description: "Start of time-of-day window, seconds from 00:00:00 (0-86399); set to activate time-of-day filtering.",
    },
    period_of_time_end_time_second: {
      type: "integer",
      description: "End of time-of-day window, seconds from 00:00:00 (0-86399); set to activate time-of-day filtering.",
    },
    period_of_time_week_of_days: {
      type: "array",
      description:
        "Weekdays to include: Sunday=0 ... Saturday=6. Omit or pass null for no weekday restriction, [] matches nothing, all 7 days = no restriction.",
      items: { type: "integer", minimum: 0, maximum: 6 },
    },
    mi_board_name: { type: "string", description: "Filter Mi tasks by board name; omit or pass null for all boards." },
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
