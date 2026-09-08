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
  "include_rep_name",
  "include_plugin_content",
  "plugin_content_max_text_length",
  "plugin_content_format",
  "count_only",
  "group_by",
  "data_types",
  "create_apps",
  "update_apps",
  "num_min",
  "num_max",
  "idf_kinds",
  "include_file_size",
]);

// v2 (ADR-0604) の集計・絞り込みの列挙値。サーバ側(get_kyous_mcp_helpers.go)と揃えること。
export const KYOUS_GROUP_BY_VALUES = new Set([
  "month",
  "day",
  "week_of_day",
  "hour",
  "data_type",
  "rep_name",
  "create_app",
  "update_app",
  "url_domain",
  "file_extension",
]);

export const KYOUS_IDF_KIND_VALUES = new Set(["image", "video", "audio", "zip", "other"]);

// カーソルは v2 から不透明文字列（複合形式 {RFC3339Nano}::{ID}）。
// Node側は解釈せず素通しする。長さだけ制限して事故を防ぐ。
export const MAX_CURSOR_LENGTH = 512;

// gkill_get_application_config の fields 射影の許可値。
export const APP_CONFIG_FIELDS = new Set([
  // 接続先の識別。gkill が返す user_id / device をそのまま通す。
  // read サーバと readwrite サーバが別アカウントを向いていても、AI からは
  // 区別する手段が無く「同じAPIなのに件数が違う」と誤診されていた
  // （2026-08-24 の実利用レビュー）。環境変数の値（GKILL_BASE_URL 等）は
  // 端末の情報なので出さない（ADR-0707）。出すのは gkill 由来のこの2つだけ。
  "user_id",
  "device",
  "tag_struct",
  "mi_board_struct",
  "rep_struct",
  "rep_type_struct",
  "device_struct",
  "kftl_template_struct",
  "mi_default_board",
  "show_tags_in_list",
]);

// gkill_delete_kyou / gkill_restore_kyou の一括指定の上限。
// 1件につき取得+更新の2往復なので、上限が無いと1リクエストで数百往復になりうる。
export const MAX_DELETE_TARGETS = 100;

// gkill_get_rep_infos の fields 射影の許可値。
// 本番では rep_infos[] だけで数百件になる（rep は rep_type ごとに重複して載るので
// rep 数より膨らむ）。一方「正準値と対応表だけ欲しい」呼び出しが多く、
// 一番よく使う形が一番大きい応答になっていた（2026-08-24 の実利用レビュー）。
export const REP_INFOS_FIELDS = new Set([
  "rep_infos",
  "canonical_rep_types",
  "plugins",
  "attached_data_reps",
]);

// attached_data_reps の data_kind。歴代端末ぶんの Tag_ / Text_ / Notification_ / GPSLogs_ が
// 並ぶので本番では約120件になり、fields で丸ごと落とすか丸ごと取るかの2択だった
// （2026-08-25 の実利用レビュー）。生成側は handle_get_rep_infos_mcp.go の
// appendAttachedDataRep が渡す4値。
export const ATTACHED_DATA_KINDS = new Set(["tag", "text", "notification", "gpslog"]);

// struct ツリーから既定で剥がす UI 状態キー（ツリーエディタの一時状態。
// 実測で応答が 193.6k→93.0k 字に減る）。check_when_inited / is_force_hide は
// 可視タグ判定に必要なので**絶対に剥がさない**。
export const APP_CONFIG_UI_STATE_KEYS = new Set([
  "is_checked",
  "indeterminate",
  "key",
  "seq",
  "seq_in_parent",
  "is_open_default",
  "parent_folder_id",
  "id",
]);

// gkill_get_gps_log のページング（Node側実装。gkillは全件を返す）。
export const DEFAULT_GPS_LIMIT = 500;
export const MAX_GPS_LIMIT = 5000;
export const GPS_GROUP_BY_VALUES = new Set(["day"]);

// gkill_get_all_rep_names の絞り込み（Node側実装。gkillは全件を返す）。
// rep が数百ある環境では「その名前の rep があるか」を確かめるだけで
// 全件を読むことになっていた。
export const DEFAULT_REP_NAMES_LIMIT = 200;
export const MAX_REP_NAMES_LIMIT = 2000;

// タグ名も rep 名と同じ理由で絞り込める必要がある。タグは数百〜数千に育つので、
// 「autolog 系のタグはあるか」を確かめるために全件を受け取るのは無駄が大きい。
// rep 名側にだけ contains / limit があり、タグ名だけ全件返しだった。
export const DEFAULT_TAG_NAMES_LIMIT = 200;
export const MAX_TAG_NAMES_LIMIT = 2000;

export const KYOUS_QUERY_BOOLEAN_FIELDS = new Set([
  "update_cache",
  "include_deleted_data",
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
  // use_plaing は旧綴りのまま。我々の綴りではなく、過去の gkill が書き出したデータのキー名（ADR-0806）
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
]);

export const KYOUS_QUERY_NUMBER_FIELDS = new Set(["map_radius", "map_latitude", "map_longitude"]);

export const KYOUS_QUERY_INTEGER_FIELDS = new Map([
  ["period_of_time_start_time_second", { min: 0, max: 86399 }],
  ["period_of_time_end_time_second", { min: 0, max: 86399 }],
]);

export const KYOUS_QUERY_DATETIME_FIELDS = new Map([
  ["calendar_start_date", { allowDateOnly: true, endOfDay: false }],
  ["calendar_end_date", { allowDateOnly: true, endOfDay: true }],
  ["playing_time", { allowDateOnly: true, endOfDay: false }],
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

// ---------------------------------------------------------------------------
// 型別エンティティの取得/更新エンドポイント
// ---------------------------------------------------------------------------
//
// **削除の語彙はここだけに置くこと。**
// 以前は gkill_delete_kyou の enum (write-tools.mjs)、DELETE_DATA_TYPES
// (write-normalization.mjs)、DELETE_ENDPOINT_MAP、GET_ENDPOINT_MAP の4箇所に
// 同じ9値が別々に書かれており、しかもサーバ2本ぶんに複製されていた。
// 1つ足し忘れると「スキーマは受理するのにディスパッチで落ちる」
// (あるいはその逆で、正規化を素通りしてエンドポイント未定義で落ちる) になる。
//
// gkill に専用の削除APIは無い。現在値を取って is_deleted を立て、
// 同じ型の更新APIへ送り直す patch 方式なので、種別ごとに取得と更新の対が要る。
// 削除・復活・版履歴の3ツールが同じ対応表を使う。
//
// **型非依存の /api/get_kyou を使ってはいけない。**
// Repositories.GetKyouHistoriesByRepName は冒頭で UnWrap() を呼び、
// キャッシュrepを丸ごとバイパスして 11rep→約940rep に膨れる（実測20.7秒）。
// 型別の XxxRepositories.GetXxxHistoriesByRepName はキャッシュrepを直接回るので安全。
export const ENTITY_TARGETS = {
  kmemo:   { getEndpoint: "/api/get_kmemo",   historiesKey: "kmemo_histories",   updateEndpoint: "/api/update_kmemo",   requestKey: "kmemo",   responseKey: "updated_kmemo" },
  urlog:   { getEndpoint: "/api/get_urlog",   historiesKey: "urlog_histories",   updateEndpoint: "/api/update_urlog",   requestKey: "urlog",   responseKey: "updated_urlog" },
  nlog:    { getEndpoint: "/api/get_nlog",    historiesKey: "nlog_histories",    updateEndpoint: "/api/update_nlog",    requestKey: "nlog",    responseKey: "updated_nlog" },
  lantana: { getEndpoint: "/api/get_lantana", historiesKey: "lantana_histories", updateEndpoint: "/api/update_lantana", requestKey: "lantana", responseKey: "updated_lantana" },
  timeis:  { getEndpoint: "/api/get_timeis",  historiesKey: "timeis_histories",  updateEndpoint: "/api/update_timeis",  requestKey: "timeis",  responseKey: "updated_timeis" },
  mi:      { getEndpoint: "/api/get_mi",      historiesKey: "mi_histories",      updateEndpoint: "/api/update_mi",      requestKey: "mi",      responseKey: "updated_mi" },
  kc:      { getEndpoint: "/api/get_kc",      historiesKey: "kc_histories",      updateEndpoint: "/api/update_kc",      requestKey: "kc",      responseKey: "updated_kc" },
  tag:     { getEndpoint: "/api/get_tag_histories_by_tag_id",   historiesKey: "tag_histories",  updateEndpoint: "/api/update_tag",  requestKey: "tag",  responseKey: "updated_tag" },
  text:    { getEndpoint: "/api/get_text_histories_by_text_id", historiesKey: "text_histories", updateEndpoint: "/api/update_text", requestKey: "text", responseKey: "updated_text" },
  // rekyou / mirekyou / notification は rep が実在するのに MCP から作成も削除もできなかった。
  // 作成(add)は依然として無いが、Web クライアントや他端末が作ったものを
  // 消す・履歴を見る・戻すことはできるべきなので対応表へ入れる。
  rekyou:       { getEndpoint: "/api/get_rekyou",   historiesKey: "rekyou_histories",   updateEndpoint: "/api/update_rekyou",   requestKey: "rekyou",   responseKey: "updated_rekyou" },
  mirekyou:     { getEndpoint: "/api/get_mirekyou", historiesKey: "mirekyou_histories", updateEndpoint: "/api/update_mirekyou", requestKey: "mirekyou", responseKey: "updated_mirekyou" },
  notification: { getEndpoint: "/api/get_gkill_notification_histories_by_notification_id", historiesKey: "notification_histories", updateEndpoint: "/api/update_gkill_notification", requestKey: "notification", responseKey: "updated_notification" },
};

// gkill_get_kyou_history が1回に返す版の既定件数。
// 履歴は編集のたびに1件伸びるので必ず上限を掛ける。
export const DEFAULT_KYOU_HISTORY_LIMIT = 20;
export const MAX_KYOU_HISTORY_LIMIT = 200;

// data_type として受理する値。削除・復活・版履歴のスキーマ enum と正規化がこれを見る。
export const ENTITY_DATA_TYPE_VALUES = Object.keys(ENTITY_TARGETS);

// 射影名 → エンティティ種別の対応。**語彙が2つある**ことへの唯一の橋。
//
// gkill_get_kyous の DTO が返す data_type は「射影名」（mi_create / timeis_start …）で、
// delete / restore / history が受理するのは ENTITY_TARGETS のキー＝「エンティティ種別」
// （mi / timeis …）。この違いはどこにも書かれておらず、
// **gkill_add_mi の応答 data_type:"mi_create" をそのまま gkill_delete_kyou へ渡すと落ちる**
// （KFTL の created[] だけはエンティティ語彙なので通る、という三者三様だった。
// 2026-08-24 の実利用レビュー）。
// 応答をそのまま次のツールへ渡せるよう、射影名も受理して正規化する。
export const PROJECTION_TO_ENTITY_DATA_TYPE = new Map([
  ["mi_create", "mi"],
  ["mi_check", "mi"],
  ["mi_limit", "mi"],
  ["mi_start", "mi"],
  ["mi_end", "mi"],
  ["mirekyou_create", "mirekyou"],
  ["mirekyou_check", "mirekyou"],
  ["mirekyou_limit", "mirekyou"],
  ["mirekyou_start", "mirekyou"],
  ["mirekyou_end", "mirekyou"],
  ["timeis_start", "timeis"],
  ["timeis_end", "timeis"],
  // idf は Kyou 側の data_type。エンティティとしての削除/履歴は未対応なので写さない。
]);

/**
 * ENTITY_AND_PROJECTION_DATA_TYPE_VALUES は data_type を受け取る口の enum。
 *
 * 入口は toEntityDataType を通すので射影名(mi_start / timeis_start …)でも動くのに、
 * enum を畳んだ後の語彙だけにしていると、JSON Schema を送信前に検証する
 * クライアントが mi_start をサーバへ届く前に弾く。説明文が「応答の data_type を
 * そのまま渡せる」と約束しているので正面から食い違っていた(2026-08-25 の実利用レビュー)。
 * 畳んだ後の語彙(ENTITY_DATA_TYPE_VALUES)は検証側で使うので、両方を残す。
 */
// CROSS_SERVER_TOOL_MENTIONS は「そのサーバに載っていなくても説明文に出てよい」ツール名。
//
// read / write は同じデータを別のサーバで扱うので、読み取り側の説明が
// 書き込み側のツールに触れること自体は正しい —— 「この id は gkill_update_text へ
// 渡すためのものだ」は、その場で呼べという案内ではなく、id を捨てるなという話。
//
// 禁じたいのはその逆で、**「今これを呼べ」と書いておいて、そのサーバに無い**こと。
// read の履歴説明が gkill_restore_kyou を「使え」と書いていたのが実例で、
// AI は載っていないツールを探しにいく。区別は機械では付かないので、
// 参照情報としての言及だけをここへ明示する。**案内を足すときは表ではなく
// ツールの搭載側を直すこと。**
export const CROSS_SERVER_TOOL_MENTIONS = new Set([
  "gkill_restore_kyou",
  "gkill_update_text",
  "gkill_delete_kyou",
  "gkill_submit_kftl",
  "gkill_add_mi",
  "gkill_update_mi",
  "gkill_add_tag",
]);

export const ENTITY_AND_PROJECTION_DATA_TYPE_VALUES = [
  ...ENTITY_DATA_TYPE_VALUES,
  ...PROJECTION_TO_ENTITY_DATA_TYPE.keys(),
];

/**
 * toEntityDataType は射影名を受け取ったらエンティティ種別へ寄せる。
 * エンティティ種別・未知の値はそのまま返す（判定は呼び出し側の許可リストに任せる）。
 *
 * @param {string} dataType 射影名またはエンティティ種別。
 * @returns {string} エンティティ種別。
 */
export function toEntityDataType(dataType) {
  return PROJECTION_TO_ENTITY_DATA_TYPE.get(dataType) ?? dataType;
}

// 消したツールの案内。MCP のツール一覧は**クライアントのセッション寿命で固定**されるので、
// サーバから消しても既存セッションは呼び続ける (2026-08-24 の再監査で live コネクタから再現)。
// しかもそのクライアントが握っている古い説明文は「パスを優先しろ」と、
// まさにこの消えたツールへ誘導している。名前だけ返すと行き止まりになる。
const REMOVED_TOOL_HINTS = new Map([
  [
    "gkill_get_idf_file_path",
    "removed on 2026-08-24: it could never be reached from an HTTP client, and on stdio it was a strict " +
      "subset of the file_path that gkill_get_kyous already puts into IDF payloads. On stdio read " +
      "payload.file_path directly; otherwise call gkill_get_idf_file, which is the only way to get an " +
      'image in front of the model (add thumb:"1024x1024" to stay under the size cap).',
  ],
  [
    "gkill_get_plugin_content",
    "removed: pass include_plugin_content:true to gkill_get_kyous instead. It inlines plugin bodies into " +
      "the same response rather than costing one round trip per entry.",
  ],
]);

// 未知ツールのエラー文。消したツールなら代替まで案内する。
export function unknownToolMessage(name) {
  const hint = REMOVED_TOOL_HINTS.get(name);
  return hint ? `Unknown tool: ${name} — ${hint}` : `Unknown tool: ${name}`;
}
