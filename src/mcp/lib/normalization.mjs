// Normalization functions extracted from gkill-read-server.mjs.

import { invalidArgument } from "./errors.mjs";
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
  MI_CHECK_STATES,
  MI_SORT_TYPES,
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

export function normalizeKyouQuery(query) {
  const source = assertObject(query, "query");
  const normalized = {};

  for (const [key, value] of Object.entries(source)) {
    const field = `query.${key}`;
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
    normalized[key] = value;
  }

  normalized.only_latest_data = true;
  return normalized;
}

export function normalizeKyouArgs(args) {
  const source = args == null ? {} : assertObject(args, "arguments");
  assertKnownKeys(source, KYOUS_TOP_LEVEL_FIELDS, "arguments");

  const normalized = {
    query: normalizeKyouQuery(Object.prototype.hasOwnProperty.call(source, "query") ? source.query : {}),
    limit: DEFAULT_KYOUS_LIMIT,
    max_size_mb: DEFAULT_KYOUS_MAX_SIZE_MB,
    is_include_timeis: DEFAULT_KYOUS_INCLUDE_TIMEIS,
  };

  if (Object.prototype.hasOwnProperty.call(source, "locale_name") && source.locale_name !== undefined) {
    normalized.locale_name = assertTrimmedString(source.locale_name, "locale_name");
  }
  if (Object.prototype.hasOwnProperty.call(source, "limit") && source.limit !== undefined) {
    normalized.limit = assertInteger(source.limit, "limit", { min: 1, max: 1000 });
  }
  if (Object.prototype.hasOwnProperty.call(source, "cursor") && source.cursor !== undefined) {
    normalized.cursor = normalizeDateTimeString(source.cursor, "cursor", { allowDateOnly: true, endOfDay: false });
  }
  if (Object.prototype.hasOwnProperty.call(source, "max_size_mb") && source.max_size_mb !== undefined) {
    normalized.max_size_mb = assertNumber(source.max_size_mb, "max_size_mb", { minExclusive: 0 });
  }
  if (Object.prototype.hasOwnProperty.call(source, "is_include_timeis") && source.is_include_timeis !== undefined) {
    normalized.is_include_timeis = assertBoolean(source.is_include_timeis, "is_include_timeis");
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
  const source = args == null ? {} : assertObject(args, "arguments");
  assertKnownKeys(source, new Set(["start_date", "end_date", "locale_name"]), "arguments");
  return {
    start_date: normalizeDateTimeString(source.start_date, "start_date", { allowDateOnly: true, endOfDay: false }),
    end_date: normalizeDateTimeString(source.end_date, "end_date", { allowDateOnly: true, endOfDay: true }),
    ...(Object.prototype.hasOwnProperty.call(source, "locale_name") && source.locale_name !== undefined
      ? { locale_name: assertTrimmedString(source.locale_name, "locale_name") }
      : {}),
  };
}

export function normalizeIdfFileArgs(args) {
  const source = args == null ? {} : assertObject(args, "arguments");
  assertKnownKeys(source, new Set(["rep_name", "file_name", "locale_name"]), "arguments");
  const normalized = {
    rep_name: assertTrimmedString(source.rep_name, "rep_name"),
    file_name: assertTrimmedString(source.file_name, "file_name"),
  };
  if (Object.prototype.hasOwnProperty.call(source, "locale_name") && source.locale_name !== undefined) {
    normalized.locale_name = assertTrimmedString(source.locale_name, "locale_name");
  }
  return normalized;
}
