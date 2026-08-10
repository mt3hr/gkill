// Shared constants for gkill MCP validation/normalization.

export const ISO_DATETIME_DESC = "ISO-8601 datetime string, e.g. 2026-02-25T10:30:00+09:00";
export const DATE_ONLY_DESC = "YYYY-MM-DD date string";
export const DEFAULT_KYOUS_LIMIT = 20;
export const DEFAULT_KYOUS_MAX_SIZE_MB = 0.25;
export const DEFAULT_KYOUS_INCLUDE_TIMEIS = false;
// gkill_get_idf_file が返すファイルの上限。base64はJSON-RPCレスポンスに素で載るため、
// 上限がないと大きな動画などで応答が破裂する。超えた場合はローカルパス経由の取得を案内する。
export const MAX_IDF_FILE_BYTES = Math.max(
  1,
  Number(process.env.GKILL_MCP_MAX_FILE_BYTES) || 8 * 1024 * 1024,
);
export const RFC3339_REGEX = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/;
export const DATE_ONLY_REGEX = /^\d{4}-\d{2}-\d{2}$/;

export const KYOUS_TOP_LEVEL_FIELDS = new Set([
  "query",
  "locale_name",
  "limit",
  "cursor",
  "max_size_mb",
  "is_include_timeis",
  "include_id",
  "include_plugin_content",
  "plugin_content_max_text_length",
  "plugin_content_format",
]);

export const KYOUS_QUERY_BOOLEAN_FIELDS = new Set([
  "update_cache",
  "is_deleted",
  "words_and",
  "tags_and",
  "timeis_words_and",
  "timeis_tags_and",
  "include_create_mi",
  "include_check_mi",
  "include_limit_mi",
  "include_start_mi",
  "include_end_mi",
  "include_end_timeis",
  "is_image_only",
  "for_mi",
  "only_latest_data",
]);

// 廃止された旧 use_X フラグ (フィルタの活性化は値フィールドの非null存在で決まる)。
// 後方互換のために受理だけする: use_X:false は対応グループの値キーを落とし、
// use_X:true は捨てる。normalized クエリには決して積まれない。
export const LEGACY_USE_FLAG_KEYS = new Set([
  "use_tags",
  "use_reps",
  "use_rep_types",
  "use_ids",
  "use_include_id",
  "use_words",
  "use_timeis",
  "use_timeis_tags",
  "use_calendar",
  "use_map",
  "use_plaing",
  "use_update_time",
  "use_mi_board_name",
  "use_period_of_time",
  // 値キーを束ねない (フラグだけ落とす) が、受理しないと未知キーとして throw してしまい
  // 「後方互換で受け付ける」という約束が破れるので、Go の移行実装と同じ16キーを揃える
  "use_mi_sort_type",
  "use_mi_check_state",
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

// normalizeKyouQuery が受け付けるクエリキー全体。
// 個別の集合から導出するので、フィールドを足すときはここを触らなくてよい。
export const KYOUS_QUERY_ALL_FIELDS = new Set([
  ...KYOUS_QUERY_BOOLEAN_FIELDS,
  ...KYOUS_QUERY_STRING_ARRAY_FIELDS,
  ...KYOUS_QUERY_NUMBER_FIELDS,
  ...KYOUS_QUERY_INTEGER_FIELDS.keys(),
  ...KYOUS_QUERY_DATETIME_FIELDS.keys(),
  // 上の集合に属さない、個別に検証しているキー
  "period_of_time_week_of_days",
  "mi_board_name",
  "mi_check_state",
  "mi_sort_type",
]);

// gkill_get_kyous の plugin_content_format 引数。既定は text。
// プラグインのコンテンツHTMLは表示用のCSS/JSでほとんどが埋まっているため、
// 生HTMLを既定で返すとトークンを浪費するだけになる。
export const PLUGIN_CONTENT_FORMATS = new Set(["text", "html", "both"]);
export const DEFAULT_PLUGIN_CONTENT_FORMAT = "text";

// ここから下は gkill_get_kyous の include_plugin_content 用。
// プラグインKyouの本文はgkillに保存されておらず、1件ずつプラグインプロセスに
// 問い合わせるしかない。レスポンスに直接埋めるので、単発取得より1件あたりの
// 上限を小さく取り、さらに合計・件数・時間にも上限を設ける。
export const DEFAULT_INCLUDE_PLUGIN_CONTENT = false;
// 1件あたりのテキスト上限。既定値は「20件並べても常識的なサイズに収まる」値。
export const DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH = 4000;
// 1件だけを対象にして全文を取りたいケースがあるので、上限は大きめに許す。
export const MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH = 200000;
// 1回のget_kyousで本文を埋めるKyouの最大件数。
export const MAX_INLINE_PLUGIN_CONTENT_KYOUS = 20;
// 埋め込むテキストの合計上限。1件を最大長で取っても収まる値にしてある。
// buildToolResultがペイロードを2回直列化するため、実際の転送量はこの約2倍になる。
export const INLINE_PLUGIN_CONTENT_TOTAL_TEXT_LENGTH = 200000;
// 並列に叩くプラグイン (rep_name) の数。同一プラグイン内は必ず直列にする。
export const INLINE_PLUGIN_CONTENT_REP_CONCURRENCY = 4;
// 本文取得全体の打ち切り時間。これを過ぎたら新しいリクエストを「始めない」だけで、
// 実行中のリクエストはabortしない。現在のgkillはabortされてもプラグインプロセスを
// 回収しないが、MCPサーバは古いgkillにも接続しうる (古い実装ではabortがプロセスkillになる)。
export const INLINE_PLUGIN_CONTENT_DEADLINE_MS = 30000;
// htmlToTextに渡す前にHTMLを切り詰める上限。stripTagsが1文字ずつ走査するため、
// 巨大なHTMLをそのまま流すと変換だけで時間を食う。
export const MAX_INLINE_PLUGIN_CONTENT_HTML_LENGTH = 400000;

export const MI_CHECK_STATES = new Set(["all", "checked", "uncheck"]);
export const MI_SORT_TYPES = new Set(["create_time", "estimate_start_time", "estimate_end_time", "limit_time"]);
