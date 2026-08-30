/**
 * Input normalization for gkill write MCP tools.
 *
 * Each normalizer validates required fields and normalizes optional datetime
 * strings.  Reuses assertTrimmedString / assertNumber / assertInteger /
 * assertBoolean / assertKnownKeys from ./validation.mjs and
 * normalizeDateTimeString from ./normalization.mjs.
 */

import { isPlainObject, invalidArgument } from "./errors.mjs";
import {
  assertTrimmedString,
  assertNumber,
  assertInteger,
  assertBoolean,
  assertKnownKeys,
} from "./validation.mjs";
import { ENTITY_DATA_TYPE_VALUES, toEntityDataType, MAX_DELETE_TARGETS } from "./constants.mjs";
import { normalizeDateTimeString, reviveStaleSchemaArgs } from "./normalization.mjs";

// ---------------------------------------------------------------------------
// Allowed data_type values for gkill_delete_kyou
// ---------------------------------------------------------------------------

// 語彙の正本は constants.mjs の ENTITY_TARGETS。ここは互換のための派生。
export const DELETE_DATA_TYPES = new Set(ENTITY_DATA_TYPE_VALUES);

// 後から足した targets（オブジェクトの配列）。古いスキーマを掴んだクライアントは
// 未知の引数を正規JSON文字列にして送ってくるので、型を復元する
// （is_video で実際に起きた事故と同じクラス。gkill-mcp スキル参照）。
const DELETE_STALE_SCHEMA_ARG_KINDS = new Map([["targets", "object_array"]]);

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// 日付だけを渡されたとき「その日の終わり」へ展開するフィールド。
//
// 締切や見積終了は「その日じゅう」の意味なので、00:00:00 に丸めると
// **8/25締切が25日の開始時点で期限切れ**になる。読み取り側は既に
// calendar_end_date / GPS の end_date をフィールドごとに endOfDay 指定しており
// (constants.mjs)、書き込み側だけが一律 false のままだった
// (2026-08-24 の実利用レビュー)。
const END_OF_DAY_DATETIME_FIELDS = new Set(["limit_time", "estimate_end_time"]);

/**
 * Validate and normalize an optional datetime argument.
 * Returns the normalized RFC 3339 string, or undefined if the field is absent.
 */
function optionalDatetime(args, field) {
  if (args[field] === undefined || args[field] === null) return undefined;
  return normalizeDateTimeString(args[field], field, {
    allowDateOnly: true,
    endOfDay: END_OF_DAY_DATETIME_FIELDS.has(field),
  });
}

/**
 * Validate that args is a plain object.  Throws if not.
 */
function assertArgs(args) {
  if (!isPlainObject(args)) {
    throw invalidArgument("arguments", "must be an object", args);
  }
}

// ---------------------------------------------------------------------------
// Normalizers
// ---------------------------------------------------------------------------

// URL からスキームが抜けていると gkill はページ取得すら試みず、
// title が空のまま保存される（エラーは出ない）。呼び出し側からは
// 「タイトルの自動補完が効かなかった」としか見えないので入口で弾く。
const URL_SCHEME_REGEX = /^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//;

/** @param {unknown} value @param {string} field */
function assertUrlWithScheme(value, field) {
  const url = assertTrimmedString(value, field);
  if (!URL_SCHEME_REGEX.test(url)) {
    throw invalidArgument(field, 'must include a scheme, e.g. "https://example.com/page"', value);
  }
  return url;
}

// 終わりが始まりより前の TimeIs は作れてしまい、長さが負になる。
/** @param {string|undefined} startTime @param {string|undefined} endTime */
function assertTimeIsOrder(startTime, endTime) {
  // null は「終了を消す」の意思表示なので比較しない
  if (startTime === undefined || endTime === undefined || endTime === null) return;
  // normalizeDateTimeString は妥当な RFC3339 をそのまま返すので "…Z" と "…+09:00" が
  // 混ざりうる。文字列比較だと offset の違いで前後を取り違える。
  if (Date.parse(startTime) > Date.parse(endTime)) {
    throw invalidArgument(
      "end_time",
      `must not be before start_time (${startTime}); the interval would have a negative length`,
      endTime,
    );
  }
}

// ---------------------------------------------------------------------------
// エンティティのフィールド表
// ---------------------------------------------------------------------------

// 追加(add)と更新(update)は同じ手順で、型ごとに違うのは
// **フィールド名・種別・追加時に必須か**だけ。かつては型ごとに add / update の
// 2関数を手書きしており、計18本が並んでいた。
//
// その並びは実際にずれた: gkill_add_urlog は assertUrlWithScheme で
// スキーム付きURLを要求するのに、gkill_update_urlog は assertTrimmedString のままで、
// assertUrlWithScheme はこのファイル全体で**追加側の1箇所でしか呼ばれていなかった**。
// スキームの無いURLで更新すると gkill はページ取得を試みず、title が空のまま
// エラーも出さずに保存される（2026-08-25 の実利用レビューを機に発見）。
//
// 表にすれば種別の定義が1箇所になるので、この種のずれは書けなくなる（ADR-0611）。
//
// kind:
//   string   … assertTrimmedString
//   url      … assertUrlWithScheme（スキーム必須。add / update とも）
//   integer  … assertInteger（range を渡せる）
//   number   … assertNumber
//   boolean  … assertBoolean
//   datetime … optionalDatetime（未指定なら触らない。END_OF_DAY_DATETIME_FIELDS も見る）
// requiredOnAdd … add のとき必須。update では常に任意（patch セマンティクス）
// defaultOnAdd  … add で未指定のときに入れる値
// addOnly       … add でしか受け付けない（update のスキーマにも無い）
// nullClears    … update のとき null を「消す」の意思表示として通す
// revivesStaleBoolean … 後付けの boolean 引数。古いスキーマのクライアントからは
//                 正規JSON文字列 ("true"/"false") で届くので型を復元する。
//                 **boolean をスキーマへ後から足すときは必ずこれを立て、
//                 normalization.mjs の STALE_SCHEMA_ARG_KINDS_BY_TOOL へも載せること**
const ENTITY_FIELD_SPECS = {
  kmemo: {
    fields: [
      { name: "content", kind: "string", requiredOnAdd: true },
      { name: "related_time", kind: "datetime" },
    ],
  },
  urlog: {
    fields: [
      { name: "url", kind: "url", requiredOnAdd: true },
      { name: "title", kind: "string" },
      { name: "related_time", kind: "datetime" },
      // 外向き取得の抑止 (2026-08-30 MCPレビュー)。add 専用 —— update で再取得が走るのは
      // リクエストに re_get_urlog_content:true を明示したときだけで (handle_update_urlog.go)、
      // MCP の runUpdate はそのキーを送らない。つまり update 経路には抑止すべき取得が無い。
      // エンティティには載せない (write-handlers がリクエストの skip_fetch_* へ写す)。
      { name: "fetch_metadata", kind: "boolean", defaultOnAdd: true, addOnly: true, revivesStaleBoolean: true },
      { name: "fetch_favicon", kind: "boolean", defaultOnAdd: true, addOnly: true, revivesStaleBoolean: true },
    ],
  },
  nlog: {
    fields: [
      { name: "title", kind: "string", requiredOnAdd: true },
      { name: "amount", kind: "integer", requiredOnAdd: true },
      { name: "shop", kind: "string" },
      { name: "related_time", kind: "datetime" },
    ],
  },
  lantana: {
    fields: [
      { name: "mood", kind: "integer", requiredOnAdd: true, range: { min: 0, max: 10 } },
      { name: "related_time", kind: "datetime" },
    ],
  },
  timeis: {
    fields: [
      { name: "title", kind: "string", requiredOnAdd: true },
      { name: "start_time", kind: "datetime" },
      // end_time だけは null に意味がある。「終了を取り消して進行中へ戻す」の唯一の手段で、
      // これが無いと一度終わらせた TimeIs を MCP から二度と進行中にできない
      // (Go 側 reps.TimeIs.EndTime は *time.Time なので nil を保存できる)。
      // 未指定 = 触らない、null = 消す、値 = その時刻にする、の3値。
      { name: "end_time", kind: "datetime", nullClears: true },
    ],
    // patch なので両方揃ったときだけ比べる（片側だけの更新は既存値と突き合わせられない）
    after: (normalized) => assertTimeIsOrder(normalized.start_time, normalized.end_time),
  },
  mi: {
    fields: [
      { name: "title", kind: "string", requiredOnAdd: true },
      // board_name は add でも省略可。未指定のときは呼び出し側 (write-handlers) が
      // ApplicationConfig の mi_default_board を引いて埋める。
      // ここで必須にすると、スキーマが optional と宣言しているフィールドを省略しただけで
      // 「must be a string」の型エラーになり、呼び出し側は自分の入力ミスだと誤診する。
      { name: "board_name", kind: "string" },
      { name: "is_checked", kind: "boolean", defaultOnAdd: false },
      { name: "limit_time", kind: "datetime" },
      { name: "estimate_start_time", kind: "datetime" },
      { name: "estimate_end_time", kind: "datetime" },
      // false のとき、実在しない board_name を「新しい板の作成」ではなく typo として弾く
      // (実在確認は write-handlers が板一覧と照合する。エンティティには載せない)。
      // 既定 true = 従来どおり未知の板名は新しい板を作る (2026-08-30 MCPレビュー、フラグ追加)。
      { name: "allow_create_board", kind: "boolean", defaultOnAdd: true, revivesStaleBoolean: true },
    ],
  },
  kc: {
    fields: [
      { name: "title", kind: "string", requiredOnAdd: true },
      { name: "num_value", kind: "number", requiredOnAdd: true },
      { name: "related_time", kind: "datetime" },
    ],
  },
  tag: {
    fields: [
      { name: "tag", kind: "string", requiredOnAdd: true },
      // 付け替えはできない。update_tag はタグ名だけを変える。
      { name: "target_id", kind: "string", requiredOnAdd: true, addOnly: true },
    ],
  },
  text: {
    fields: [
      { name: "text", kind: "string", requiredOnAdd: true },
      { name: "target_id", kind: "string", requiredOnAdd: true, addOnly: true },
    ],
  },
};

/** @param {{name: string, kind: string, range?: object}} field @param {unknown} value */
function assertFieldValue(field, value) {
  switch (field.kind) {
    case "string":
      return assertTrimmedString(value, field.name);
    case "url":
      return assertUrlWithScheme(value, field.name);
    case "integer":
      return assertInteger(value, field.name, field.range);
    case "number":
      return assertNumber(value, field.name, field.range);
    case "boolean":
      return assertBoolean(value, field.name);
    default:
      throw new Error(`unknown field kind: ${field.kind}`);
  }
}

/**
 * normalizeEntityArgs は add / update の引数を ENTITY_FIELD_SPECS に従って検証する。
 * mode="add" は requiredOnAdd を必須にし、mode="update" は id だけ必須の patch。
 *
 * @param {string} dataType ENTITY_FIELD_SPECS のキー
 * @param {unknown} args
 * @param {"add"|"update"} mode
 */
function normalizeEntityArgs(dataType, args, mode) {
  const spec = ENTITY_FIELD_SPECS[dataType];
  assertArgs(args);

  const fields = spec.fields.filter((field) => mode === "add" || !field.addOnly);
  const allowedKeys = new Set(["locale_name"]);
  if (mode === "update") {
    allowedKeys.add("id");
  }
  for (const field of fields) {
    allowedKeys.add(field.name);
  }
  assertKnownKeys(args, allowedKeys);

  const normalized = {};
  if (mode === "update") {
    normalized.id = assertTrimmedString(args.id, "id");
  }
  for (const field of fields) {
    if (field.kind === "datetime") {
      normalized[field.name] =
        mode === "update" && field.nullClears && args[field.name] === null
          ? null
          : optionalDatetime(args, field.name);
      continue;
    }
    // 後から足した boolean 引数は、古いツールスキーマを掴んだクライアントから
    // 正規JSON文字列 ("true" / "false") で届く。表で宣言したフィールドだけ型を復元する
    // (read側 reviveStaleSchemaArgs の書き込み版。古さの検出と警告は
    // appendStaleSchemaWarning が生の引数から行うので、ここは黙って直してよい)。
    // trim してから比べるのは検出器 (parseCanonicalJSONValue) と受理範囲を揃えるため ——
    // ずれると「stale 警告は出るのに型エラーで落ちる」入力が生まれる。
    let value = args[field.name];
    if (field.revivesStaleBoolean && typeof value === "string") {
      const trimmed = value.trim();
      if (trimmed === "true" || trimmed === "false") {
        value = trimmed === "true";
      }
    }
    // add の必須フィールドは値の有無を見ずに検証へ通す。未指定なら
    // 「must be a string」等でその欄の名前つきに落ちる（従来と同じ文言）。
    if (mode === "add" && field.requiredOnAdd) {
      normalized[field.name] = assertFieldValue(field, value);
      continue;
    }
    if (value === undefined) {
      normalized[field.name] = mode === "add" ? field.defaultOnAdd : undefined;
      continue;
    }
    normalized[field.name] = assertFieldValue(field, value);
  }
  normalized.locale_name =
    args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;

  if (spec.after) {
    spec.after(normalized);
  }
  return normalized;
}

// ---------------------------------------------------------------------------
// Normalizers（表から作る。名前つきの口は呼び出し側とテストのために残す）
// ---------------------------------------------------------------------------

/** @param {unknown} args */
export function normalizeKmemoArgs(args) {
  return normalizeEntityArgs("kmemo", args, "add");
}

/** @param {unknown} args */
export function normalizeUrlogArgs(args) {
  return normalizeEntityArgs("urlog", args, "add");
}

/** @param {unknown} args */
export function normalizeNlogArgs(args) {
  return normalizeEntityArgs("nlog", args, "add");
}

/** @param {unknown} args */
export function normalizeLantanaArgs(args) {
  return normalizeEntityArgs("lantana", args, "add");
}

/** @param {unknown} args */
export function normalizeTimeIsArgs(args) {
  return normalizeEntityArgs("timeis", args, "add");
}

/** @param {unknown} args */
export function normalizeMiArgs(args) {
  return normalizeEntityArgs("mi", args, "add");
}

/** @param {unknown} args */
export function normalizeKcArgs(args) {
  return normalizeEntityArgs("kc", args, "add");
}

/** @param {unknown} args */
export function normalizeTagArgs(args) {
  return normalizeEntityArgs("tag", args, "add");
}

/** @param {unknown} args */
export function normalizeTextArgs(args) {
  return normalizeEntityArgs("text", args, "add");
}

// ---------------------------------------------------------------------------
// Update normalizers (Patch-style: only id is required)
// ---------------------------------------------------------------------------

/** @param {unknown} args */
export function normalizeUpdateKmemoArgs(args) {
  return normalizeEntityArgs("kmemo", args, "update");
}

/** @param {unknown} args */
export function normalizeUpdateUrlogArgs(args) {
  return normalizeEntityArgs("urlog", args, "update");
}

/** @param {unknown} args */
export function normalizeUpdateNlogArgs(args) {
  return normalizeEntityArgs("nlog", args, "update");
}

/** @param {unknown} args */
export function normalizeUpdateLantanaArgs(args) {
  return normalizeEntityArgs("lantana", args, "update");
}

/** @param {unknown} args */
export function normalizeUpdateTimeIsArgs(args) {
  return normalizeEntityArgs("timeis", args, "update");
}

/** @param {unknown} args */
export function normalizeUpdateMiArgs(args) {
  return normalizeEntityArgs("mi", args, "update");
}

/** @param {unknown} args */
export function normalizeUpdateKcArgs(args) {
  return normalizeEntityArgs("kc", args, "update");
}

/** @param {unknown} args */
export function normalizeUpdateTagArgs(args) {
  return normalizeEntityArgs("tag", args, "update");
}

/** @param {unknown} args */
export function normalizeUpdateTextArgs(args) {
  return normalizeEntityArgs("text", args, "update");
}

/** @param {unknown} args */
export function normalizeKftlArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["kftl_text", "locale_name", "idempotency_key"]));
  const kftl_text = assertTrimmedString(args.kftl_text, "kftl_text");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  // KFTL は DB トランザクションではないので、途中で失敗すると先に書けたぶんが残る。
  // 同じ鍵で再送すれば二重登録にならない（サーバ側の受け口は前からあり、
  // Wear OS は送っていたのに MCP だけ送っていなかった。リトライのたびに
  // 孤児 kmemo が積んでいた実測あり。2026-08-25 の実利用レビュー）。
  const idempotency_key =
    args.idempotency_key !== undefined ? assertTrimmedString(args.idempotency_key, "idempotency_key") : undefined;
  return { kftl_text, locale_name, idempotency_key };
}

// 削除・復活の「対象」引数を1本にまとめる。
//
// 単件は {id, data_type}、一括は {targets:[{id, data_type}, ...]}。
// 一括を足したのは、KFTL が1回で5件作れるのに後始末が1件ずつ2往復で、
// 検証の後片付けに11往復かかったという報告があったため（2026-08-24 の実利用レビュー）。
// gkill_submit_kftl の created[] が既に {id, data_type} の配列なので、
// **応答をそのまま入力へ渡せる**形にしてある。
function normalizeDeleteTargets(args, verb) {
  const hasSingle = args.id !== undefined || args.data_type !== undefined;
  const hasBatch = args.targets !== undefined;
  if (hasSingle && hasBatch) {
    throw invalidArgument("targets", `cannot be combined with id / data_type; pass one form or the other`, args.targets);
  }
  if (!hasSingle && !hasBatch) {
    throw invalidArgument("id", `is required (or pass targets:[{id, data_type}] to ${verb} several entries)`, args.id);
  }
  if (hasSingle) {
    const id = assertTrimmedString(args.id, "id");
    // 検索結果や add_* の応答が返すのは射影名（mi_create / timeis_start）で、
    // ここが受理するのはエンティティ種別（mi / timeis）。応答をそのまま渡せるよう寄せる。
    const data_type = toEntityDataType(assertTrimmedString(args.data_type, "data_type"));
    if (!DELETE_DATA_TYPES.has(data_type)) {
      throw invalidArgument("data_type", `must be one of: ${[...DELETE_DATA_TYPES].join(", ")}`, data_type);
    }
    return [{ id, data_type }];
  }
  if (!Array.isArray(args.targets)) {
    throw invalidArgument("targets", "must be an array of {id, data_type}", args.targets);
  }
  if (args.targets.length === 0) {
    throw invalidArgument("targets", "must not be empty", args.targets);
  }
  if (args.targets.length > MAX_DELETE_TARGETS) {
    throw invalidArgument("targets", `must have at most ${MAX_DELETE_TARGETS} entries`, args.targets.length);
  }
  return args.targets.map((target, index) => {
    if (!isPlainObject(target)) {
      throw invalidArgument(`targets[${index}]`, "must be an object with id and data_type", target);
    }
    assertKnownKeys(target, new Set(["id", "data_type"]), `targets[${index}]`);
    const id = assertTrimmedString(target.id, `targets[${index}].id`);
    const data_type = toEntityDataType(assertTrimmedString(target.data_type, `targets[${index}].data_type`));
    if (!DELETE_DATA_TYPES.has(data_type)) {
      throw invalidArgument(
        `targets[${index}].data_type`,
        `must be one of: ${[...DELETE_DATA_TYPES].join(", ")}`,
        data_type,
      );
    }
    return { id, data_type };
  });
}

/** @param {unknown} args */
export function normalizeRestoreArgs(args) {
  assertArgs(args);
  const source = reviveStaleSchemaArgs(args, DELETE_STALE_SCHEMA_ARG_KINDS);
  assertKnownKeys(source, new Set(["id", "data_type", "targets", "locale_name"]));
  const targets = normalizeDeleteTargets(source, "restore");
  const locale_name = source.locale_name !== undefined ? assertTrimmedString(source.locale_name, "locale_name") : undefined;
  return { targets, batch: source.targets !== undefined, locale_name };
}

/** @param {unknown} args */
export function normalizeDeleteArgs(args) {
  assertArgs(args);
  const source = reviveStaleSchemaArgs(args, DELETE_STALE_SCHEMA_ARG_KINDS);
  assertKnownKeys(source, new Set(["id", "data_type", "targets", "locale_name"]));
  const targets = normalizeDeleteTargets(source, "delete");
  const locale_name = source.locale_name !== undefined ? assertTrimmedString(source.locale_name, "locale_name") : undefined;
  return { targets, batch: source.targets !== undefined, locale_name };
}
