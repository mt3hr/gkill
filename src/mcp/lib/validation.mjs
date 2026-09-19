// Validation functions extracted from gkill-read-server.mjs.

import { isPlainObject, invalidArgument } from "./errors.mjs";

export function assertObject(value, field, { allowUndefined = false } = {}) {
  if (value === undefined && allowUndefined) {
    return undefined;
  }
  if (!isPlainObject(value)) {
    throw invalidArgument(field, "must be an object", value);
  }
  return value;
}

export function assertBoolean(value, field) {
  if (typeof value !== "boolean") {
    throw invalidArgument(field, "must be a boolean", value);
  }
  return value;
}

export function assertNumber(value, field, { minExclusive = null, min = null, max = null } = {}) {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    throw invalidArgument(field, "must be a finite number", value);
  }
  if (minExclusive !== null && value <= minExclusive) {
    throw invalidArgument(field, `must be greater than ${minExclusive}`, value);
  }
  if (min !== null && value < min) {
    throw invalidArgument(field, `must be greater than or equal to ${min}`, value);
  }
  if (max !== null && value > max) {
    throw invalidArgument(field, `must be less than or equal to ${max}`, value);
  }
  return value;
}

export function assertInteger(value, field, { min = null, max = null } = {}) {
  if (typeof value !== "number" || !Number.isInteger(value)) {
    throw invalidArgument(field, "must be an integer", value);
  }
  // 2^53 を超える整数は JSON を往復するだけで別の値に化ける。受理すると
  // 「保存した金額と読み戻した金額が違う」が例外もエラーも無しに起きる。
  if (!Number.isSafeInteger(value)) {
    throw invalidArgument(
      field,
      `must be within the safe integer range (at most ${Number.MAX_SAFE_INTEGER})`,
      value,
    );
  }
  if (min !== null && value < min) {
    throw invalidArgument(field, `must be greater than or equal to ${min}`, value);
  }
  if (max !== null && value > max) {
    throw invalidArgument(field, `must be less than or equal to ${max}`, value);
  }
  return value;
}

export function assertTrimmedString(value, field) {
  if (typeof value !== "string") {
    throw invalidArgument(field, "must be a string", value);
  }
  const trimmed = value.trim();
  if (!trimmed) {
    throw invalidArgument(field, "must not be empty", value);
  }
  return trimmed;
}

export function assertStringArray(value, field) {
  if (!Array.isArray(value)) {
    throw invalidArgument(field, "must be an array of strings", value);
  }
  return value.map((item, index) => assertTrimmedString(item, `${field}[${index}]`));
}

export function assertIntegerArray(value, field, { min = null, max = null } = {}) {
  if (!Array.isArray(value)) {
    throw invalidArgument(field, "must be an array of integers", value);
  }
  return value.map((item, index) => assertInteger(item, `${field}[${index}]`, { min, max }));
}

// unknownKeyMessage は「その名前の引数は無い」の文言。**未知キーを弾く全箇所がこれを使う。**
//
// 呼び出し側の書き間違いと、クライアントが握っている古いツール一覧（改名・削除前の
// 名前を今も載せている）とは、サーバからは区別できない。ツール一覧はクライアントの
// セッション寿命で固定されるので、サーバを直しても生きているセッションには届かない
// （2026-09-14 のレビューで、ChatGPT が改名前の検索条件名を一覧どおりに送って
// 「is not supported」だけを受け取り、行き止まりになった）。
// entityNotFoundMessage と同じく**区別できないことを言う**: 一覧にその名前があるなら
// 一覧が古い、と両方の可能性を示し、再接続と gkill_status の照合を案内する。
// 改名前の名前そのものは書かない（旧綴りは verify_docs が禁止する。ADR-0806）。
export function unknownKeyMessage() {
  return (
    "is not supported (see detail.allowed for the accepted names). Either the name is misspelled, or the tool list " +
    "your client holds is stale: tool lists are fetched once per client session, so a field renamed or removed on the " +
    "server stays in your list until you reconnect. If your tool schema lists this name, reconnect the MCP client; " +
    "gkill_status reports the server's current schema_revision to compare against the one in its description"
  );
}

// assertKnownKeys は未知のキーを弾く。
// field は「どのオブジェクトの中か」を示す接頭辞で、既定は "arguments"（ツール引数の直下）。
// 既定を置く前は write 側20箇所が第3引数を渡しておらず、
// エラーが `Invalid argument 'undefined.contnet'` と出て、
// 呼び出し側からは自分の書き間違いなのか実装の不具合なのか判別できなかった。
//
// 未知のキーは**全部集めて1回で**投げる（detail.unknown）。1件ずつ返すと、キーを3つ
// 間違えた呼び出しは3往復になる（2026-09-18 の実利用報告。KFTL は全行まとめて返す）。
// detail.field は先頭の未知キー（既存の呼び出し側とテストは1件の形を前提にしている）。
// hiddenKeys は「受理はするが detail.allowed に載せない」キー（廃止済み引数。公開スキーマにも無いので、
// 一覧に出すと「only_latest_data:false なら過去版が読めるのか」と考える余地を作るだけ。ADR-0620）。
export function assertKnownKeys(value, allowedKeys, field = "arguments", hiddenKeys = null) {
  const unknown = Object.keys(value).filter((key) => !allowedKeys.has(key) && !(hiddenKeys && hiddenKeys.has(key)));
  if (unknown.length === 0) {
    return;
  }
  const suffix = unknown.length > 1 ? ` (${unknown.length} unknown names in this object: ${unknown.join(", ")})` : "";
  throw invalidArgument(`${field}.${unknown[0]}`, unknownKeyMessage() + suffix, value[unknown[0]], {
    allowed: Array.from(allowedKeys).filter((key) => !(hiddenKeys && hiddenKeys.has(key))).sort(),
    unknown,
  });
}
