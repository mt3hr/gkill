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
import { normalizeDateTimeString } from "./normalization.mjs";

// ---------------------------------------------------------------------------
// Allowed data_type values for gkill_delete_kyou
// ---------------------------------------------------------------------------

export const DELETE_DATA_TYPES = new Set([
  "kmemo", "urlog", "nlog", "lantana", "timeis", "mi", "kc", "tag", "text",
]);

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Validate and normalize an optional datetime argument.
 * Returns the normalized RFC 3339 string, or undefined if the field is absent.
 */
function optionalDatetime(args, field) {
  if (args[field] === undefined || args[field] === null) return undefined;
  return normalizeDateTimeString(args[field], field, { allowDateOnly: true, endOfDay: false });
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

/** @param {unknown} args */
export function normalizeKmemoArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["content", "related_time", "locale_name"]));
  const content = assertTrimmedString(args.content, "content");
  const related_time = optionalDatetime(args, "related_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { content, related_time, locale_name };
}

/** @param {unknown} args */
export function normalizeUrlogArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["url", "title", "related_time", "locale_name"]));
  const url = assertTrimmedString(args.url, "url");
  const title = args.title !== undefined ? assertTrimmedString(args.title, "title") : undefined;
  const related_time = optionalDatetime(args, "related_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { url, title, related_time, locale_name };
}

/** @param {unknown} args */
export function normalizeNlogArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["title", "amount", "shop", "related_time", "locale_name"]));
  const title = assertTrimmedString(args.title, "title");
  const amount = assertInteger(args.amount, "amount");
  const shop = args.shop !== undefined ? assertTrimmedString(args.shop, "shop") : undefined;
  const related_time = optionalDatetime(args, "related_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { title, amount, shop, related_time, locale_name };
}

/** @param {unknown} args */
export function normalizeLantanaArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["mood", "related_time", "locale_name"]));
  const mood = assertInteger(args.mood, "mood", { min: 0, max: 10 });
  const related_time = optionalDatetime(args, "related_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { mood, related_time, locale_name };
}

/** @param {unknown} args */
export function normalizeTimeIsArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["title", "start_time", "end_time", "locale_name"]));
  const title = assertTrimmedString(args.title, "title");
  const start_time = optionalDatetime(args, "start_time");
  const end_time = optionalDatetime(args, "end_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { title, start_time, end_time, locale_name };
}

/** @param {unknown} args */
export function normalizeMiArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set([
    "title", "board_name", "is_checked",
    "limit_time", "estimate_start_time", "estimate_end_time",
    "locale_name",
  ]));
  const title = assertTrimmedString(args.title, "title");
  const board_name = assertTrimmedString(args.board_name, "board_name");
  const is_checked = args.is_checked !== undefined ? assertBoolean(args.is_checked, "is_checked") : false;
  const limit_time = optionalDatetime(args, "limit_time");
  const estimate_start_time = optionalDatetime(args, "estimate_start_time");
  const estimate_end_time = optionalDatetime(args, "estimate_end_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { title, board_name, is_checked, limit_time, estimate_start_time, estimate_end_time, locale_name };
}

/** @param {unknown} args */
export function normalizeKcArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["title", "num_value", "related_time", "locale_name"]));
  const title = assertTrimmedString(args.title, "title");
  const num_value = assertNumber(args.num_value, "num_value");
  const related_time = optionalDatetime(args, "related_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { title, num_value, related_time, locale_name };
}

/** @param {unknown} args */
export function normalizeTagArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["tag", "target_id", "locale_name"]));
  const tag = assertTrimmedString(args.tag, "tag");
  const target_id = assertTrimmedString(args.target_id, "target_id");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { tag, target_id, locale_name };
}

/** @param {unknown} args */
export function normalizeTextArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["text", "target_id", "locale_name"]));
  const text = assertTrimmedString(args.text, "text");
  const target_id = assertTrimmedString(args.target_id, "target_id");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { text, target_id, locale_name };
}

// ---------------------------------------------------------------------------
// Update normalizers (Patch-style: only id is required)
// ---------------------------------------------------------------------------

/** @param {unknown} args */
export function normalizeUpdateKmemoArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["id", "content", "related_time", "locale_name"]));
  const id = assertTrimmedString(args.id, "id");
  const content = args.content !== undefined ? assertTrimmedString(args.content, "content") : undefined;
  const related_time = optionalDatetime(args, "related_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { id, content, related_time, locale_name };
}

/** @param {unknown} args */
export function normalizeUpdateUrlogArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["id", "url", "title", "related_time", "locale_name"]));
  const id = assertTrimmedString(args.id, "id");
  const url = args.url !== undefined ? assertTrimmedString(args.url, "url") : undefined;
  const title = args.title !== undefined ? assertTrimmedString(args.title, "title") : undefined;
  const related_time = optionalDatetime(args, "related_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { id, url, title, related_time, locale_name };
}

/** @param {unknown} args */
export function normalizeUpdateNlogArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["id", "title", "amount", "shop", "related_time", "locale_name"]));
  const id = assertTrimmedString(args.id, "id");
  const title = args.title !== undefined ? assertTrimmedString(args.title, "title") : undefined;
  const amount = args.amount !== undefined ? assertInteger(args.amount, "amount") : undefined;
  const shop = args.shop !== undefined ? assertTrimmedString(args.shop, "shop") : undefined;
  const related_time = optionalDatetime(args, "related_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { id, title, amount, shop, related_time, locale_name };
}

/** @param {unknown} args */
export function normalizeUpdateLantanaArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["id", "mood", "related_time", "locale_name"]));
  const id = assertTrimmedString(args.id, "id");
  const mood = args.mood !== undefined ? assertInteger(args.mood, "mood", { min: 0, max: 10 }) : undefined;
  const related_time = optionalDatetime(args, "related_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { id, mood, related_time, locale_name };
}

/** @param {unknown} args */
export function normalizeUpdateTimeIsArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["id", "title", "start_time", "end_time", "locale_name"]));
  const id = assertTrimmedString(args.id, "id");
  const title = args.title !== undefined ? assertTrimmedString(args.title, "title") : undefined;
  const start_time = optionalDatetime(args, "start_time");
  const end_time = optionalDatetime(args, "end_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { id, title, start_time, end_time, locale_name };
}

/** @param {unknown} args */
export function normalizeUpdateMiArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set([
    "id", "title", "board_name", "is_checked",
    "limit_time", "estimate_start_time", "estimate_end_time",
    "locale_name",
  ]));
  const id = assertTrimmedString(args.id, "id");
  const title = args.title !== undefined ? assertTrimmedString(args.title, "title") : undefined;
  const board_name = args.board_name !== undefined ? assertTrimmedString(args.board_name, "board_name") : undefined;
  const is_checked = args.is_checked !== undefined ? assertBoolean(args.is_checked, "is_checked") : undefined;
  const limit_time = optionalDatetime(args, "limit_time");
  const estimate_start_time = optionalDatetime(args, "estimate_start_time");
  const estimate_end_time = optionalDatetime(args, "estimate_end_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { id, title, board_name, is_checked, limit_time, estimate_start_time, estimate_end_time, locale_name };
}

/** @param {unknown} args */
export function normalizeUpdateKcArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["id", "title", "num_value", "related_time", "locale_name"]));
  const id = assertTrimmedString(args.id, "id");
  const title = args.title !== undefined ? assertTrimmedString(args.title, "title") : undefined;
  const num_value = args.num_value !== undefined ? assertNumber(args.num_value, "num_value") : undefined;
  const related_time = optionalDatetime(args, "related_time");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { id, title, num_value, related_time, locale_name };
}

/** @param {unknown} args */
export function normalizeUpdateTagArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["id", "tag", "locale_name"]));
  const id = assertTrimmedString(args.id, "id");
  const tag = args.tag !== undefined ? assertTrimmedString(args.tag, "tag") : undefined;
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { id, tag, locale_name };
}

/** @param {unknown} args */
export function normalizeUpdateTextArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["id", "text", "locale_name"]));
  const id = assertTrimmedString(args.id, "id");
  const text = args.text !== undefined ? assertTrimmedString(args.text, "text") : undefined;
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { id, text, locale_name };
}

/** @param {unknown} args */
export function normalizeKftlArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["kftl_text", "locale_name"]));
  const kftl_text = assertTrimmedString(args.kftl_text, "kftl_text");
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { kftl_text, locale_name };
}

/** @param {unknown} args */
export function normalizeDeleteArgs(args) {
  assertArgs(args);
  assertKnownKeys(args, new Set(["id", "data_type", "locale_name"]));
  const id = assertTrimmedString(args.id, "id");
  const data_type = assertTrimmedString(args.data_type, "data_type");
  if (!DELETE_DATA_TYPES.has(data_type)) {
    throw invalidArgument("data_type", `must be one of: ${[...DELETE_DATA_TYPES].join(", ")}`, data_type);
  }
  const locale_name = args.locale_name !== undefined ? assertTrimmedString(args.locale_name, "locale_name") : undefined;
  return { id, data_type, locale_name };
}
