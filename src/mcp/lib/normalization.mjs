// Normalization functions extracted from gkill-read-server.mjs.

import { invalidArgument, GkillApiError } from "./errors.mjs";
import { THUMB_QUERY_REGEX, MAX_THUMB_SIZE } from "./payload.mjs";
import {
  assertObject,
  assertBoolean,
  assertNumber,
  assertInteger,
  assertTrimmedString,
  assertStringArray,
  assertIntegerArray,
  assertKnownKeys,
} from "./validation.mjs";
import {
  RFC3339_REGEX,
  DATE_ONLY_REGEX,
  ISO_DATETIME_DESC,
  DATE_ONLY_DESC,
  DEFAULT_KYOUS_LIMIT,
  DEFAULT_KYOUS_MAX_SIZE_MB,
  DEFAULT_KYOUS_INCLUDE_TIMEIS,
  KYOUS_TOP_LEVEL_FIELDS,
  KYOUS_QUERY_BOOLEAN_FIELDS,
  KYOUS_QUERY_STRING_ARRAY_FIELDS,
  KYOUS_QUERY_NUMBER_FIELDS,
  KYOUS_QUERY_INTEGER_FIELDS,
  KYOUS_QUERY_DATETIME_FIELDS,
  KYOUS_QUERY_ALL_FIELDS,
  LEGACY_USE_FLAG_KEYS,
  MI_CHECK_STATES,
  MI_SORT_TYPES,
  PLUGIN_CONTENT_FORMATS,
  DEFAULT_PLUGIN_CONTENT_FORMAT,
  MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
  DEFAULT_INCLUDE_PLUGIN_CONTENT,
  DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
  KYOUS_GROUP_BY_VALUES,
  KYOUS_IDF_KIND_VALUES,
  MAX_CURSOR_LENGTH,
  APP_CONFIG_FIELDS,
  DEFAULT_GPS_LIMIT,
  MAX_GPS_LIMIT,
  GPS_GROUP_BY_VALUES,
  ENTITY_DATA_TYPE_VALUES,
  DEFAULT_KYOU_HISTORY_LIMIT,
  MAX_KYOU_HISTORY_LIMIT,
} from "./constants.mjs";

export function pad2(value) {
  return String(value).padStart(2, "0");
}

export function formatLocalRfc3339(date) {
  const offsetMinutes = -date.getTimezoneOffset();
  const sign = offsetMinutes >= 0 ? "+" : "-";
  const offsetHours = Math.floor(Math.abs(offsetMinutes) / 60);
  const offsetRemainder = Math.abs(offsetMinutes) % 60;
  return `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())}T${pad2(date.getHours())}:${pad2(date.getMinutes())}:${pad2(date.getSeconds())}${sign}${pad2(offsetHours)}:${pad2(offsetRemainder)}`;
}

export function normalizeDateOnlyToRfc3339(value, { endOfDay = false } = {}) {
  const match = DATE_ONLY_REGEX.exec(value);
  if (!match) {
    return null;
  }
  const year = Number.parseInt(match[0].slice(0, 4), 10);
  const month = Number.parseInt(match[0].slice(5, 7), 10);
  const day = Number.parseInt(match[0].slice(8, 10), 10);
  const date = endOfDay
    ? new Date(year, month - 1, day, 23, 59, 59, 0)
    : new Date(year, month - 1, day, 0, 0, 0, 0);
  if (
    Number.isNaN(date.getTime()) ||
    date.getFullYear() !== year ||
    date.getMonth() !== month - 1 ||
    date.getDate() !== day
  ) {
    return null;
  }
  return formatLocalRfc3339(date);
}

export function normalizeDateTimeString(value, field, { allowDateOnly = false, endOfDay = false } = {}) {
  const trimmed = assertTrimmedString(value, field);
  if (RFC3339_REGEX.test(trimmed) && Number.isFinite(Date.parse(trimmed))) {
    return trimmed;
  }
  if (allowDateOnly && DATE_ONLY_REGEX.test(trimmed)) {
    const normalized = normalizeDateOnlyToRfc3339(trimmed, { endOfDay });
    if (normalized) {
      return normalized;
    }
  }
  const allowedFormat = allowDateOnly ? `${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}` : ISO_DATETIME_DESC;
  throw invalidArgument(field, `must be ${allowedFormat}`, value);
}

// 旧 use_X フラグと、そのフラグが束ねていた値キーの対応表。
// use_X:false は「そのグループを使わない」の意思表示なので値キーを落とし、
// use_X:true は捨てるだけでよい (値キーの非null存在がフィルタを活性化する)。
// use_include_id は対応する値キーを持たない (ids は常に include リスト)。
// use_timeis は複合ゲート (timeis_words/timeis_not_words の存在で活性、
// TimeIsタグはそのゲート内でだけ効く) を束ねていたので、false のときは
// timeis_tags も一緒に落とす。
const LEGACY_USE_FLAG_VALUE_KEYS = new Map([
  ["use_tags", ["tags"]],
  ["use_reps", ["reps"]],
  ["use_rep_types", ["rep_types"]],
  ["use_ids", ["ids"]],
  ["use_include_id", []],
  // 値キーは落とさない (Go の移行実装もフラグ削除のみ)
  ["use_mi_sort_type", []],
  ["use_mi_check_state", []],
  ["use_words", ["words", "not_words"]],
  ["use_timeis", ["timeis_words", "timeis_not_words", "timeis_tags"]],
  ["use_timeis_tags", ["timeis_tags"]],
  ["use_calendar", ["calendar_start_date", "calendar_end_date"]],
  ["use_map", ["map_latitude", "map_longitude", "map_radius"]],
  ["use_plaing", ["plaing_time"]],
  ["use_update_time", ["update_time"]],
  ["use_mi_board_name", ["mi_board_name"]],
  [
    "use_period_of_time",
    ["period_of_time_start_time_second", "period_of_time_end_time_second", "period_of_time_week_of_days"],
  ],
]);

export function normalizeKyouQuery(query) {
  const source = assertObject(query, "query");
  const normalized = {};
  const disabledLegacyFlags = [];

  for (const [key, value] of Object.entries(source)) {
    const field = `query.${key}`;
    // null はキー欠落と同義 (フィルタ未使用)。normalized には積まない。
    if (value === null) {
      continue;
    }
    // 廃止済みの use_X フラグは後方互換のために受理だけする。
    // false は「そのグループを使わない」なので、束ねていた値キーをループ後に落とす。
    // true は捨てる (値キーの存在自体がフィルタを活性化する)。
    if (LEGACY_USE_FLAG_KEYS.has(key)) {
      if (assertBoolean(value, field) === false) {
        disabledLegacyFlags.push(key);
      }
      continue;
    }
    if (KYOUS_QUERY_BOOLEAN_FIELDS.has(key)) {
      normalized[key] = assertBoolean(value, field);
      continue;
    }
    if (KYOUS_QUERY_STRING_ARRAY_FIELDS.has(key)) {
      normalized[key] = assertStringArray(value, field);
      continue;
    }
    if (KYOUS_QUERY_NUMBER_FIELDS.has(key)) {
      normalized[key] = assertNumber(value, field);
      continue;
    }
    if (KYOUS_QUERY_INTEGER_FIELDS.has(key)) {
      normalized[key] = assertInteger(value, field, KYOUS_QUERY_INTEGER_FIELDS.get(key));
      continue;
    }
    if (KYOUS_QUERY_DATETIME_FIELDS.has(key)) {
      if (value === "") continue; // skip empty datetime strings (ChatGPT sends "" for unused fields)
      // plaing_time はリテラル "now" を受け、現在時刻のRFC3339へ展開する
      if (key === "plaing_time" && typeof value === "string" && value.trim() === "now") {
        normalized[key] = formatLocalRfc3339(new Date());
        continue;
      }
      normalized[key] = normalizeDateTimeString(value, field, KYOUS_QUERY_DATETIME_FIELDS.get(key));
      continue;
    }
    if (key === "period_of_time_week_of_days") {
      normalized[key] = assertIntegerArray(value, field, { min: 0, max: 6 });
      continue;
    }
    if (key === "mi_board_name") {
      normalized[key] = assertTrimmedString(value, field);
      continue;
    }
    if (key === "mi_check_state") {
      const state = assertTrimmedString(value, field);
      if (!MI_CHECK_STATES.has(state)) {
        throw invalidArgument(field, `must be one of: ${Array.from(MI_CHECK_STATES).join(", ")}`, value);
      }
      normalized[key] = state;
      continue;
    }
    if (key === "mi_sort_type") {
      const sortType = assertTrimmedString(value, field);
      if (!MI_SORT_TYPES.has(sortType)) {
        throw invalidArgument(field, `must be one of: ${Array.from(MI_SORT_TYPES).join(", ")}`, value);
      }
      normalized[key] = sortType;
      continue;
    }
    // 未知キーは受け付けない。
    // query は MCP 経由で AI クライアントが自由に組み立てられるため、
    // 検証していないキーでの動的な書き込みを残すとプロトタイプ汚染の経路になる。
    throw invalidArgument(field, "is not supported", value, {
      allowed: Array.from(KYOUS_QUERY_ALL_FIELDS).sort(),
    });
  }

  // 旧 use_X:false は「そのグループを使わない」だったので、束ねていた値キーを取り除く。
  // フラグと値キーの出現順に依存しないよう、ループ後にまとめて落とす。
  for (const flagKey of disabledLegacyFlags) {
    for (const valueKey of LEGACY_USE_FLAG_VALUE_KEYS.get(flagKey)) {
      delete normalized[valueKey];
    }
  }

  // TimeIsタグフィルタはサーバ側で「timeis_words/timeis_not_words の非null存在」を
  // ゲートにしている (HasTimeIsFilter && TimeIsTags != nil)。timeis_tags だけを
  // 指定したクエリでもゲートを満たすよう、timeis_words: []
  // (「任意のTimeIsに覆われたKyou」) を補完する。
  if (
    normalized.timeis_tags !== undefined &&
    normalized.timeis_words === undefined &&
    normalized.timeis_not_words === undefined
  ) {
    normalized.timeis_words = [];
  }

  normalized.only_latest_data = true;
  return normalized;
}

// 旧スキーマキャッシュ救済（2026-08-23 本番実測）: ツール一覧のスキーマを
// 更新前に取得したままのクライアント（claude.ai コネクタ等はセッション中
// スキーマを再取得しない）は、後から追加されたトップレベル引数の型を知らず、
// 値を正規のJSON符号化の文字列のまま送ってくる（"true"・'["nlog"]' など）。
// その符号化に限り本来の型へ復元する。曖昧な文字列は復元せずそのまま残し、
// 後段の型検証で従来どおりエラーにする。対象はスキーマ追加後の引数のみ
// （昔からある引数は常に型付きで届くため、緩めない）。
function parseCanonicalJSONValue(value, kind) {
  const trimmed = value.trim();
  if (trimmed === "") {
    return undefined;
  }
  let parsed;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return undefined;
  }
  if (kind === "boolean" && typeof parsed === "boolean") {
    return parsed;
  }
  if (kind === "number" && typeof parsed === "number" && Number.isFinite(parsed)) {
    return parsed;
  }
  if (kind === "string_array" && Array.isArray(parsed) && parsed.every((item) => typeof item === "string")) {
    return parsed;
  }
  return undefined;
}

export function reviveStaleSchemaArgs(source, kindsByKey) {
  let revived = null;
  for (const [key, kind] of kindsByKey) {
    if (!Object.prototype.hasOwnProperty.call(source, key)) {
      continue;
    }
    const value = source[key];
    if (typeof value !== "string") {
      continue;
    }
    const parsed = parseCanonicalJSONValue(value, kind);
    if (parsed === undefined) {
      continue;
    }
    if (revived === null) {
      revived = { ...source };
    }
    revived[key] = parsed;
  }
  return revived ?? source;
}

// v2 (ADR-0053) で追加したトップレベル引数。string 型の group_by / cursor は
// 旧スキーマ経由でも壊れないので対象外。
const KYOUS_STALE_SCHEMA_ARG_KINDS = new Map([
  ["count_only", "boolean"],
  ["data_types", "string_array"],
  ["num_min", "number"],
  ["num_max", "number"],
  ["idf_kinds", "string_array"],
  ["include_file_size", "boolean"],
]);

const GPS_STALE_SCHEMA_ARG_KINDS = new Map([
  ["limit", "number"],
  ["count_only", "boolean"],
]);

const APP_CONFIG_STALE_SCHEMA_ARG_KINDS = new Map([
  ["fields", "string_array"],
  ["include_ui_state", "boolean"],
]);

export function normalizeKyouArgs(args) {
  const source = reviveStaleSchemaArgs(
    args == null ? {} : assertObject(args, "arguments"),
    KYOUS_STALE_SCHEMA_ARG_KINDS,
  );
  assertKnownKeys(source, KYOUS_TOP_LEVEL_FIELDS, "arguments");

  const normalized = {
    query: normalizeKyouQuery(Object.prototype.hasOwnProperty.call(source, "query") ? source.query : {}),
    limit: DEFAULT_KYOUS_LIMIT,
    max_size_mb: DEFAULT_KYOUS_MAX_SIZE_MB,
    is_include_timeis: DEFAULT_KYOUS_INCLUDE_TIMEIS,
    include_plugin_content: DEFAULT_INCLUDE_PLUGIN_CONTENT,
    plugin_content_max_text_length: DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
    plugin_content_format: DEFAULT_PLUGIN_CONTENT_FORMAT,
  };

  if (Object.prototype.hasOwnProperty.call(source, "locale_name") && source.locale_name !== undefined) {
    normalized.locale_name = assertTrimmedString(source.locale_name, "locale_name");
  }
  if (Object.prototype.hasOwnProperty.call(source, "limit") && source.limit !== undefined) {
    normalized.limit = assertInteger(source.limit, "limit", { min: 1, max: 1000 });
  }
  if (Object.prototype.hasOwnProperty.call(source, "cursor") && source.cursor !== undefined) {
    // v2: カーソルは不透明文字列（複合形式 {RFC3339Nano}::{ID}）。
    // 旧実装はここで日時として正規化しており、複合形式を渡すと壊していた。
    // 解釈と旧形式の受理はサーバ(parseMCPCursor)の責務。Node は素通しする。
    const cursor = assertTrimmedString(source.cursor, "cursor");
    if (cursor.length > MAX_CURSOR_LENGTH) {
      throw new GkillApiError(`cursor is too long (${cursor.length} > ${MAX_CURSOR_LENGTH})`);
    }
    normalized.cursor = cursor;
  }
  if (Object.prototype.hasOwnProperty.call(source, "max_size_mb") && source.max_size_mb !== undefined) {
    normalized.max_size_mb = assertNumber(source.max_size_mb, "max_size_mb", { minExclusive: 0 });
  }
  if (Object.prototype.hasOwnProperty.call(source, "is_include_timeis") && source.is_include_timeis !== undefined) {
    normalized.is_include_timeis = assertBoolean(source.is_include_timeis, "is_include_timeis");
  }
  // include_id / include_rep_name は v2 で廃止（id/rep_name は常時付与）。
  // 旧クライアント救済のため受理はするが、値は使わない（型検証だけ行う）。
  if (Object.prototype.hasOwnProperty.call(source, "include_id") && source.include_id !== undefined) {
    assertBoolean(source.include_id, "include_id");
  }
  if (Object.prototype.hasOwnProperty.call(source, "include_rep_name") && source.include_rep_name !== undefined) {
    assertBoolean(source.include_rep_name, "include_rep_name");
  }
  // ---- v2 (ADR-0053) ----
  if (Object.prototype.hasOwnProperty.call(source, "count_only") && source.count_only !== undefined) {
    normalized.count_only = assertBoolean(source.count_only, "count_only");
  }
  if (Object.prototype.hasOwnProperty.call(source, "group_by") && source.group_by !== undefined) {
    const groupBy = assertTrimmedString(source.group_by, "group_by");
    if (!KYOUS_GROUP_BY_VALUES.has(groupBy)) {
      throw new GkillApiError(
        `Invalid group_by: ${JSON.stringify(groupBy)} (valid: ${[...KYOUS_GROUP_BY_VALUES].join(", ")})`,
      );
    }
    normalized.group_by = groupBy;
  }
  if (Object.prototype.hasOwnProperty.call(source, "data_types") && source.data_types !== undefined) {
    normalized.data_types = assertStringArray(source.data_types, "data_types");
  }
  if (Object.prototype.hasOwnProperty.call(source, "num_min") && source.num_min !== undefined) {
    normalized.num_min = assertNumber(source.num_min, "num_min", {});
  }
  if (Object.prototype.hasOwnProperty.call(source, "num_max") && source.num_max !== undefined) {
    normalized.num_max = assertNumber(source.num_max, "num_max", {});
  }
  if (Object.prototype.hasOwnProperty.call(source, "idf_kinds") && source.idf_kinds !== undefined) {
    const idfKinds = assertStringArray(source.idf_kinds, "idf_kinds");
    for (const kind of idfKinds) {
      if (!KYOUS_IDF_KIND_VALUES.has(kind)) {
        throw new GkillApiError(
          `Invalid idf_kind: ${JSON.stringify(kind)} (valid: ${[...KYOUS_IDF_KIND_VALUES].join(", ")})`,
        );
      }
    }
    normalized.idf_kinds = idfKinds;
  }
  if (Object.prototype.hasOwnProperty.call(source, "include_file_size") && source.include_file_size !== undefined) {
    normalized.include_file_size = assertBoolean(source.include_file_size, "include_file_size");
  }
  if (
    Object.prototype.hasOwnProperty.call(source, "include_plugin_content") &&
    source.include_plugin_content !== undefined
  ) {
    normalized.include_plugin_content = assertBoolean(source.include_plugin_content, "include_plugin_content");
  }
  if (
    Object.prototype.hasOwnProperty.call(source, "plugin_content_max_text_length") &&
    source.plugin_content_max_text_length !== undefined
  ) {
    normalized.plugin_content_max_text_length = assertInteger(
      source.plugin_content_max_text_length,
      "plugin_content_max_text_length",
      { min: 1, max: MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH },
    );
  }
  if (
    Object.prototype.hasOwnProperty.call(source, "plugin_content_format") &&
    source.plugin_content_format !== undefined
  ) {
    const format = assertTrimmedString(source.plugin_content_format, "plugin_content_format").toLowerCase();
    if (!PLUGIN_CONTENT_FORMATS.has(format)) {
      throw invalidArgument(
        "plugin_content_format",
        `must be one of: ${Array.from(PLUGIN_CONTENT_FORMATS).join(", ")}`,
        source.plugin_content_format,
      );
    }
    normalized.plugin_content_format = format;
  }

  return normalized;
}

export function normalizeLocaleOnlyArgs(args) {
  const source = args == null ? {} : assertObject(args, "arguments");
  assertKnownKeys(source, new Set(["locale_name"]), "arguments");
  if (!Object.prototype.hasOwnProperty.call(source, "locale_name") || source.locale_name === undefined) {
    return {};
  }
  return { locale_name: assertTrimmedString(source.locale_name, "locale_name") };
}

export function normalizeGpsArgs(args) {
  const source = reviveStaleSchemaArgs(
    args == null ? {} : assertObject(args, "arguments"),
    GPS_STALE_SCHEMA_ARG_KINDS,
  );
  assertKnownKeys(
    source,
    new Set(["start_date", "end_date", "locale_name", "limit", "cursor", "count_only", "group_by"]),
    "arguments",
  );
  const normalized = {
    start_date: normalizeDateTimeString(source.start_date, "start_date", { allowDateOnly: true, endOfDay: false }),
    end_date: normalizeDateTimeString(source.end_date, "end_date", { allowDateOnly: true, endOfDay: true }),
    ...(Object.prototype.hasOwnProperty.call(source, "locale_name") && source.locale_name !== undefined
      ? { locale_name: assertTrimmedString(source.locale_name, "locale_name") }
      : {}),
  };
  // ページングは Node 側実装（gkill は全件を返す。将来 Go 側へ移す余地あり）
  normalized.limit = DEFAULT_GPS_LIMIT;
  if (Object.prototype.hasOwnProperty.call(source, "limit") && source.limit !== undefined) {
    normalized.limit = assertInteger(source.limit, "limit", { min: 1, max: MAX_GPS_LIMIT });
  }
  if (Object.prototype.hasOwnProperty.call(source, "cursor") && source.cursor !== undefined) {
    const cursor = assertTrimmedString(source.cursor, "cursor");
    if (cursor.length > MAX_CURSOR_LENGTH) {
      throw new GkillApiError(`cursor is too long (${cursor.length} > ${MAX_CURSOR_LENGTH})`);
    }
    normalized.cursor = cursor;
  }
  if (Object.prototype.hasOwnProperty.call(source, "count_only") && source.count_only !== undefined) {
    normalized.count_only = assertBoolean(source.count_only, "count_only");
  }
  if (Object.prototype.hasOwnProperty.call(source, "group_by") && source.group_by !== undefined) {
    const groupBy = assertTrimmedString(source.group_by, "group_by");
    if (!GPS_GROUP_BY_VALUES.has(groupBy)) {
      throw new GkillApiError(
        `Invalid group_by: ${JSON.stringify(groupBy)} (valid: ${[...GPS_GROUP_BY_VALUES].join(", ")})`,
      );
    }
    normalized.group_by = groupBy;
  }
  return normalized;
}

// normalizeAppConfigArgs は gkill_get_application_config の引数を検証する。
// fields は射影（許可値のみ）、include_ui_state は struct ツリーの UI 状態キーを残すか。
export function normalizeAppConfigArgs(args) {
  const source = reviveStaleSchemaArgs(
    args == null ? {} : assertObject(args, "arguments"),
    APP_CONFIG_STALE_SCHEMA_ARG_KINDS,
  );
  assertKnownKeys(source, new Set(["locale_name", "fields", "include_ui_state"]), "arguments");
  const normalized = {};
  if (Object.prototype.hasOwnProperty.call(source, "locale_name") && source.locale_name !== undefined) {
    normalized.locale_name = assertTrimmedString(source.locale_name, "locale_name");
  }
  if (Object.prototype.hasOwnProperty.call(source, "fields") && source.fields !== undefined) {
    const fields = assertStringArray(source.fields, "fields");
    for (const field of fields) {
      if (!APP_CONFIG_FIELDS.has(field)) {
        throw new GkillApiError(
          `Invalid field: ${JSON.stringify(field)} (valid: ${[...APP_CONFIG_FIELDS].join(", ")})`,
        );
      }
    }
    normalized.fields = fields;
  }
  normalized.include_ui_state = false;
  if (Object.prototype.hasOwnProperty.call(source, "include_ui_state") && source.include_ui_state !== undefined) {
    normalized.include_ui_state = assertBoolean(source.include_ui_state, "include_ui_state");
  }
  return normalized;
}

export function normalizeIdfFileArgs(args) {
  const source = args == null ? {} : assertObject(args, "arguments");
  assertKnownKeys(
    source,
    new Set(["rep_name", "file_name", "thumb", "is_video", "locale_name"]),
    "arguments",
  );
  const normalized = {
    rep_name: assertTrimmedString(source.rep_name, "rep_name"),
    file_name: assertTrimmedString(source.file_name, "file_name"),
  };
  if (Object.prototype.hasOwnProperty.call(source, "thumb") && source.thumb !== undefined) {
    const thumb = assertTrimmedString(source.thumb, "thumb");
    if (!THUMB_QUERY_REGEX.test(thumb)) {
      throw invalidArgument("thumb", 'must be "<width>x<height>", e.g. "1024x1024"', source.thumb);
    }
    // Go 側は上限超えだとサムネを作らず原本をそのまま返す。ここで弾かないと
    // 「縮小したつもりで原寸が返る」という静かな取り違えになる。
    const [width, height] = thumb.split("x").map((size) => parseInt(size, 10));
    if (width > MAX_THUMB_SIZE || height > MAX_THUMB_SIZE) {
      throw invalidArgument(
        "thumb",
        `must not exceed ${MAX_THUMB_SIZE} per side (larger values make the server return the original instead)`,
        source.thumb,
      );
    }
    normalized.thumb = thumb;
  }
  if (Object.prototype.hasOwnProperty.call(source, "is_video") && source.is_video !== undefined) {
    normalized.is_video = assertBoolean(source.is_video, "is_video");
  }
  // is_video は thumb と組でしか意味を持たない (Go 側は thumb 未指定なら即座に原本を返す)。
  // 単独で受け付けると、数百MB の動画が丸ごと返ってサイズ上限に当たるだけになる。
  if (normalized.is_video && normalized.thumb === undefined) {
    throw invalidArgument(
      "is_video",
      "requires thumb (a video frame is only extracted when a thumbnail size is given)",
      source.is_video,
    );
  }
  if (Object.prototype.hasOwnProperty.call(source, "locale_name") && source.locale_name !== undefined) {
    normalized.locale_name = assertTrimmedString(source.locale_name, "locale_name");
  }
  return normalized;
}

// normalizeKyouHistoryArgs は gkill_get_kyou_history の引数を検証する。
//
// data_type は必須。型非依存の /api/get_kyou へ落とすフォールバックは置かない
// （constants.mjs の ENTITY_TARGETS のコメント参照: UnWrap() でキャッシュを全バイパスする）。
export function normalizeKyouHistoryArgs(args) {
  const source = assertObject(args, "arguments", { allowUndefined: true }) ?? {};
  assertKnownKeys(source, new Set(["id", "data_type", "limit", "locale_name"]), "arguments");
  const id = assertTrimmedString(source.id, "id");
  const data_type = assertTrimmedString(source.data_type, "data_type");
  if (!ENTITY_DATA_TYPE_VALUES.includes(data_type)) {
    throw invalidArgument("data_type", `must be one of: ${ENTITY_DATA_TYPE_VALUES.join(", ")}`, data_type);
  }
  const limit = source.limit !== undefined
    ? assertInteger(source.limit, "limit", { min: 1, max: MAX_KYOU_HISTORY_LIMIT })
    : DEFAULT_KYOU_HISTORY_LIMIT;
  const locale_name = source.locale_name !== undefined ? assertTrimmedString(source.locale_name, "locale_name") : undefined;
  return { id, data_type, limit, locale_name };
}
