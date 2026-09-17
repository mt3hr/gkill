// Shared FindQuery JSON schema for gkill_get_kyous.
//
// read / readwrite の2サーバで同一定義を使う。フィルタの活性化は
// 「値フィールドが非nullで存在すること」で決まり、旧 use_X フラグは廃止済み
// (後方互換の受理変換は lib/normalization.mjs の normalizeKyouQuery が行う)。
// 廃止済みのキー（only_latest_data / use_X）はこのスキーマには載せない。
// **properties のキー集合は normalizeKyouQuery の受理集合（KYOUS_QUERY_ALL_FIELDS）から
// 廃止済みを引いたものと一致させること** —— schema-contract.test.mjs が固定する。
// 片方だけ改名すると「tools/list どおりに呼ぶと未知キーで拒否される」になる。

import { ISO_DATETIME_DESC, DATE_ONLY_DESC } from "./constants.mjs";

export const FIND_QUERY_SCHEMA = {
  type: "object",
  // 本文（推奨する絞り込み手順・ペイロード形・idf の読み方・プラグイン）は help topic の
  // search / mi / idf / plugin へ移した（ADR-0622）。ここは活性化の規則だけ。
  description:
    "gkill find query. A filter activates when its value field is present and non-null; omit (or pass null for) fields " +
    "you don't filter by. [] means 'filter enabled but matches nothing' (except timeis_words: [], which means 'only kyous " +
    "covered by any TimeIs'). Datetime fields use ISO-8601 strings. Payload shapes per data_type, the Mi / TimeIs " +
    "projections and how to read idf files: gkill_get_mcp_help topics search, mi and idf.",
  properties: {
    update_cache: {
      type: "boolean",
      description:
        "Force cache refresh before query. Does NOT change any user data, but it is not free either: " +
        "it triggers a rebuild of the derived caches (I/O and wait time), so a 'read-only' call with this flag " +
        "still has an operational side effect. Leave it off unless a record you know exists is missing from results.",
    },
    include_deleted_data: { type: "boolean", description: "Also return soft-deleted entries (they carry is_deleted:true). Default false. A deleted entry is indexed by the time it was DELETED, not its original related_time, and rekyou / mirekyou stay hidden even with this flag. Reading or restoring one: gkill_get_mcp_help topic:deleted." },
    rep_types: {
      type: "array",
      description:
        "Allowed rep-type names; omit or pass null for no filter, [] matches nothing. Case-sensitive canonical values come from gkill_get_rep_infos canonical_rep_types[] (files/images are \"directory\") — do not guess from ApplicationConfig labels.",
      items: { type: "string" },
    },
    ids: {
      type: "array",
      description:
        "Entry IDs to include (an include-list: only these entries are returned). Omit or pass null for no ID filter, [] matches nothing. Ids that match nothing are reported in warnings[] (the search cannot tell a missing id from a deleted or otherwise excluded one).",
      items: { type: "string" },
    },
    words: {
      type: "array",
      description:
        "Keywords to match (case-insensitive substring). Matched against each type's text fields (kmemo content; urlog url/title/description; nlog title/shop/amount; timeis title; kc title/value; mi title/board name; lantana mood value as text; idf file path and .md/.txt body; git commit message; plugin-defined text) and against attached texts; an entry whose ID starts with the keyword also matches. Omit or pass null for no keyword filter; [] (or only blank strings) applies no keyword condition, i.e. everything passes.",
      items: { type: "string" },
    },
    words_and: { type: "boolean", description: "AND logic for words (true=all must match, false=any)." },
    not_words: {
      type: "array",
      description:
        "Keywords to exclude: drops entries whose text fields (same fields as words) or attached texts contain any of them. IDs are never matched. With words omitted or empty, the result is everything minus these matches.",
      items: { type: "string" },
    },
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
    timeis_tags_and: { type: "boolean", description: "AND logic for timeis_tags." },
    calendar_start_date: {
      type: "string",
      description: `Start of the date-range filter (INCLUSIVE); set to activate. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}`,
    },
    calendar_end_date: {
      type: "string",
      description: `End of the date-range filter (INCLUSIVE - adjacent hand-made windows double-count the boundary day; prefer top-level group_by for histograms). Date-only values expand to 23:59:59 local. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}`,
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
    include_create_mi: { type: "boolean", description: "Include the created-time projection of Mi tasks (data_type mi_create). Effective only when for_mi=true. The five include_*_mi flags select which projections supply rows — with all five false (the default) a for_mi search returns ZERO entries, so set at least one (include_create_mi:true is the usual start)." },
    include_check_mi: { type: "boolean", description: "Include the checked-time projection (data_type mi_check). Effective only when for_mi=true; see include_create_mi." },
    include_limit_mi: { type: "boolean", description: "Include the deadline projection — only tasks with a limit_time (data_type mi_limit). Effective only when for_mi=true; see include_create_mi." },
    include_start_mi: { type: "boolean", description: "Include the estimated-start projection — only tasks with an estimate_start_time (data_type mi_start). Effective only when for_mi=true; see include_create_mi." },
    include_end_mi: { type: "boolean", description: "Include the estimated-end projection — only tasks with an estimate_end_time (data_type mi_end). Effective only when for_mi=true; see include_create_mi." },
    include_end_timeis: { type: "boolean", description: "Also index ended TimeIs entries by their end time (data_type timeis_end). Default false, so a TimeIs is only found on the day it STARTED — an overnight sleep that began yesterday does not appear in today's calendar range unless you set this to true." },
    playing_time: {
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
    for_mi: { type: "boolean", description: "Restrict the search to task entries — BOTH Mi and MiReKyou. Requires at least one include_*_mi flag, otherwise zero entries. Also decides which projection you see (with for_mi: mi_sort_type; without it: one collapsed representative per task). Details: gkill_get_mcp_help topic:mi." },
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
      description:
        "Which time of a Mi task to use. NOT only a sort order: it is the timestamp the calendar / time-of-day / " +
        "weekday filters match against and it sets the data_type of the results, so it changes WHICH tasks come back. " +
        "It only takes effect through the matching include_*_mi projection (limit_time needs include_limit_mi, and so on).",
      enum: ["create_time", "estimate_start_time", "estimate_end_time", "limit_time"],
    },
    // only_latest_data（MCP 層が常に true へ強制する）と旧 use_X フラグはここに載せない。
    // normalizeKyouQuery は今までどおり受理し、届いたら古いスキーマの証拠として警告する（ADR-0620）。
  },
  additionalProperties: true,
};
