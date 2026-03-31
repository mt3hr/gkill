// Shared constants for gkill MCP validation/normalization.

export const ISO_DATETIME_DESC = "ISO-8601 datetime string, e.g. 2026-02-25T10:30:00+09:00";
export const DATE_ONLY_DESC = "YYYY-MM-DD date string";
export const DEFAULT_KYOUS_LIMIT = 20;
export const DEFAULT_KYOUS_MAX_SIZE_MB = 0.25;
export const DEFAULT_KYOUS_INCLUDE_TIMEIS = false;
export const RFC3339_REGEX = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/;
export const DATE_ONLY_REGEX = /^\d{4}-\d{2}-\d{2}$/;

export const KYOUS_TOP_LEVEL_FIELDS = new Set(["query", "locale_name", "limit", "cursor", "max_size_mb", "is_include_timeis", "include_id"]);

export const KYOUS_QUERY_BOOLEAN_FIELDS = new Set([
  "update_cache",
  "is_deleted",
  "use_tags",
  "use_reps",
  "use_rep_types",
  "use_ids",
  "use_include_id",
  "use_words",
  "words_and",
  "tags_and",
  "use_timeis",
  "timeis_words_and",
  "use_timeis_tags",
  "timeis_tags_and",
  "use_calendar",
  "use_map",
  "include_create_mi",
  "include_check_mi",
  "include_limit_mi",
  "include_start_mi",
  "include_end_mi",
  "include_end_timeis",
  "use_plaing",
  "use_update_time",
  "is_image_only",
  "for_mi",
  "use_mi_board_name",
  "use_period_of_time",
  "only_latest_data",
]);

export const KYOUS_QUERY_STRING_ARRAY_FIELDS = new Set([
  "rep_types",
  "ids",
  "words",
  "not_words",
  "reps",
  "tags",
  "hide_tags",
  "timeis_words",
  "timeis_not_words",
  "timeis_tags",
  "hide_timeis_tags",
]);

export const KYOUS_QUERY_NUMBER_FIELDS = new Set(["map_radius", "map_latitude", "map_longitude"]);

export const KYOUS_QUERY_INTEGER_FIELDS = new Map([
  ["period_of_time_start_time_second", { min: 0, max: 86399 }],
  ["period_of_time_end_time_second", { min: 0, max: 86399 }],
]);

export const KYOUS_QUERY_DATETIME_FIELDS = new Map([
  ["calendar_start_date", { allowDateOnly: true, endOfDay: false }],
  ["calendar_end_date", { allowDateOnly: true, endOfDay: true }],
  ["plaing_time", { allowDateOnly: true, endOfDay: false }],
  ["update_time", { allowDateOnly: true, endOfDay: false }],
]);

export const MI_CHECK_STATES = new Set(["all", "checked", "uncheck"]);
export const MI_SORT_TYPES = new Set(["create_time", "estimate_start_time", "estimate_end_time", "limit_time"]);
