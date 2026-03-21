import { GkillApiError } from "../lib/errors.mjs";
import {
  pad2,
  formatLocalRfc3339,
  normalizeDateOnlyToRfc3339,
  normalizeDateTimeString,
  normalizeKyouQuery,
  normalizeKyouArgs,
  normalizeLocaleOnlyArgs,
  normalizeGpsArgs,
} from "../lib/normalization.mjs";

// ---------------------------------------------------------------------------
// pad2
// ---------------------------------------------------------------------------
describe("pad2", () => {
  test("pads single digit", () => {
    expect(pad2(1)).toBe("01");
    expect(pad2(0)).toBe("00");
    expect(pad2(9)).toBe("09");
  });

  test("does not pad double digit", () => {
    expect(pad2(10)).toBe("10");
    expect(pad2(59)).toBe("59");
  });

  test("handles string input", () => {
    expect(pad2("5")).toBe("05");
  });
});

// ---------------------------------------------------------------------------
// formatLocalRfc3339
// ---------------------------------------------------------------------------
describe("formatLocalRfc3339", () => {
  test("formats a date into RFC3339 with local timezone offset", () => {
    const result = formatLocalRfc3339(new Date(2026, 0, 15, 10, 30, 0));
    // Should match pattern YYYY-MM-DDTHH:MM:SS+HH:MM or -HH:MM
    expect(result).toMatch(/^2026-01-15T10:30:00[+-]\d{2}:\d{2}$/);
  });

  test("formats midnight correctly", () => {
    const result = formatLocalRfc3339(new Date(2026, 5, 1, 0, 0, 0));
    expect(result).toMatch(/^2026-06-01T00:00:00[+-]\d{2}:\d{2}$/);
  });

  test("formats end of day correctly", () => {
    const result = formatLocalRfc3339(new Date(2026, 11, 31, 23, 59, 59));
    expect(result).toMatch(/^2026-12-31T23:59:59[+-]\d{2}:\d{2}$/);
  });
});

// ---------------------------------------------------------------------------
// normalizeDateOnlyToRfc3339
// ---------------------------------------------------------------------------
describe("normalizeDateOnlyToRfc3339", () => {
  test("converts date-only to RFC3339 at start of day", () => {
    const result = normalizeDateOnlyToRfc3339("2026-03-15");
    expect(result).toMatch(/^2026-03-15T00:00:00[+-]\d{2}:\d{2}$/);
  });

  test("converts date-only to RFC3339 at end of day", () => {
    const result = normalizeDateOnlyToRfc3339("2026-03-15", { endOfDay: true });
    expect(result).toMatch(/^2026-03-15T23:59:59[+-]\d{2}:\d{2}$/);
  });

  test("returns null for non-matching input", () => {
    expect(normalizeDateOnlyToRfc3339("not-a-date")).toBeNull();
  });

  test("returns null for invalid date like Feb 30", () => {
    expect(normalizeDateOnlyToRfc3339("2026-02-30")).toBeNull();
  });

  test("returns null for month 13", () => {
    expect(normalizeDateOnlyToRfc3339("2026-13-01")).toBeNull();
  });

  test("handles leap year Feb 29", () => {
    // 2024 is a leap year
    const result = normalizeDateOnlyToRfc3339("2024-02-29");
    expect(result).toMatch(/^2024-02-29T00:00:00[+-]\d{2}:\d{2}$/);
  });

  test("rejects non-leap year Feb 29", () => {
    expect(normalizeDateOnlyToRfc3339("2026-02-29")).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// normalizeDateTimeString
// ---------------------------------------------------------------------------
describe("normalizeDateTimeString", () => {
  test("accepts valid RFC3339 datetime", () => {
    const input = "2026-02-25T10:30:00+09:00";
    expect(normalizeDateTimeString(input, "field")).toBe(input);
  });

  test("accepts UTC datetime with Z", () => {
    const input = "2026-01-01T00:00:00Z";
    expect(normalizeDateTimeString(input, "field")).toBe(input);
  });

  test("accepts datetime with negative offset", () => {
    const input = "2026-06-15T08:00:00-05:00";
    expect(normalizeDateTimeString(input, "field")).toBe(input);
  });

  test("accepts datetime with fractional seconds", () => {
    const input = "2026-03-01T12:00:00.123+09:00";
    expect(normalizeDateTimeString(input, "field")).toBe(input);
  });

  test("throws for date-only when allowDateOnly is false", () => {
    expect(() => normalizeDateTimeString("2026-03-15", "field")).toThrow(GkillApiError);
  });

  test("converts date-only when allowDateOnly is true", () => {
    const result = normalizeDateTimeString("2026-03-15", "field", { allowDateOnly: true });
    expect(result).toMatch(/^2026-03-15T00:00:00[+-]\d{2}:\d{2}$/);
  });

  test("converts date-only to end of day when endOfDay is true", () => {
    const result = normalizeDateTimeString("2026-03-15", "field", { allowDateOnly: true, endOfDay: true });
    expect(result).toMatch(/^2026-03-15T23:59:59[+-]\d{2}:\d{2}$/);
  });

  test("throws for non-string", () => {
    expect(() => normalizeDateTimeString(12345, "field")).toThrow(GkillApiError);
  });

  test("throws for empty string", () => {
    expect(() => normalizeDateTimeString("", "field")).toThrow(GkillApiError);
  });

  test("throws for invalid format", () => {
    expect(() => normalizeDateTimeString("March 15, 2026", "field")).toThrow(GkillApiError);
  });

  test("throws for invalid date-only (Feb 30) even with allowDateOnly", () => {
    expect(() => normalizeDateTimeString("2026-02-30", "field", { allowDateOnly: true })).toThrow(GkillApiError);
  });
});

// ---------------------------------------------------------------------------
// normalizeKyouQuery
// ---------------------------------------------------------------------------
describe("normalizeKyouQuery", () => {
  test("returns object with only_latest_data=true for empty query", () => {
    const result = normalizeKyouQuery({});
    expect(result).toEqual({ only_latest_data: true });
  });

  test("validates boolean fields", () => {
    const result = normalizeKyouQuery({ use_tags: true, is_deleted: false });
    expect(result.use_tags).toBe(true);
    expect(result.is_deleted).toBe(false);
    expect(result.only_latest_data).toBe(true);
  });

  test("throws for non-boolean in boolean field", () => {
    expect(() => normalizeKyouQuery({ use_tags: "yes" })).toThrow(GkillApiError);
  });

  test("validates string array fields", () => {
    const result = normalizeKyouQuery({ tags: ["tag1", " tag2 "] });
    expect(result.tags).toEqual(["tag1", "tag2"]);
  });

  test("throws for non-array in string array field", () => {
    expect(() => normalizeKyouQuery({ tags: "single-tag" })).toThrow(GkillApiError);
  });

  test("validates number fields", () => {
    const result = normalizeKyouQuery({ map_latitude: 35.6762 });
    expect(result.map_latitude).toBe(35.6762);
  });

  test("throws for non-number in number field", () => {
    expect(() => normalizeKyouQuery({ map_latitude: "35.6762" })).toThrow(GkillApiError);
  });

  test("validates integer fields with min/max", () => {
    const result = normalizeKyouQuery({ period_of_time_start_time_second: 0 });
    expect(result.period_of_time_start_time_second).toBe(0);
  });

  test("throws for out-of-range integer field", () => {
    expect(() => normalizeKyouQuery({ period_of_time_start_time_second: 86400 })).toThrow(GkillApiError);
    expect(() => normalizeKyouQuery({ period_of_time_start_time_second: -1 })).toThrow(GkillApiError);
  });

  test("validates datetime fields", () => {
    const result = normalizeKyouQuery({ calendar_start_date: "2026-01-01" });
    expect(result.calendar_start_date).toMatch(/^2026-01-01T00:00:00[+-]\d{2}:\d{2}$/);
  });

  test("calendar_end_date uses endOfDay", () => {
    const result = normalizeKyouQuery({ calendar_end_date: "2026-12-31" });
    expect(result.calendar_end_date).toMatch(/^2026-12-31T23:59:59[+-]\d{2}:\d{2}$/);
  });

  test("validates period_of_time_week_of_days as integer array", () => {
    const result = normalizeKyouQuery({ period_of_time_week_of_days: [0, 3, 6] });
    expect(result.period_of_time_week_of_days).toEqual([0, 3, 6]);
  });

  test("throws for out-of-range weekday", () => {
    expect(() => normalizeKyouQuery({ period_of_time_week_of_days: [7] })).toThrow(GkillApiError);
  });

  test("validates mi_board_name", () => {
    const result = normalizeKyouQuery({ mi_board_name: " board1 " });
    expect(result.mi_board_name).toBe("board1");
  });

  test("validates mi_check_state", () => {
    expect(normalizeKyouQuery({ mi_check_state: "all" }).mi_check_state).toBe("all");
    expect(normalizeKyouQuery({ mi_check_state: "checked" }).mi_check_state).toBe("checked");
    expect(normalizeKyouQuery({ mi_check_state: "uncheck" }).mi_check_state).toBe("uncheck");
  });

  test("throws for invalid mi_check_state", () => {
    expect(() => normalizeKyouQuery({ mi_check_state: "unknown" })).toThrow(GkillApiError);
  });

  test("validates mi_sort_type", () => {
    expect(normalizeKyouQuery({ mi_sort_type: "create_time" }).mi_sort_type).toBe("create_time");
    expect(normalizeKyouQuery({ mi_sort_type: "limit_time" }).mi_sort_type).toBe("limit_time");
  });

  test("throws for invalid mi_sort_type", () => {
    expect(() => normalizeKyouQuery({ mi_sort_type: "invalid" })).toThrow(GkillApiError);
  });

  test("always forces only_latest_data=true even if set to false", () => {
    const result = normalizeKyouQuery({ only_latest_data: false });
    expect(result.only_latest_data).toBe(true);
  });

  test("passes through unknown keys unchanged", () => {
    const result = normalizeKyouQuery({ some_future_field: "value" });
    expect(result.some_future_field).toBe("value");
    expect(result.only_latest_data).toBe(true);
  });

  test("throws for non-object query", () => {
    expect(() => normalizeKyouQuery("not-an-object")).toThrow(GkillApiError);
    expect(() => normalizeKyouQuery(null)).toThrow(GkillApiError);
  });
});

// ---------------------------------------------------------------------------
// normalizeKyouArgs
// ---------------------------------------------------------------------------
describe("normalizeKyouArgs", () => {
  test("returns defaults for null/undefined args", () => {
    const result = normalizeKyouArgs(null);
    expect(result.limit).toBe(20);
    expect(result.max_size_mb).toBe(0.25);
    expect(result.is_include_timeis).toBe(false);
    expect(result.query).toEqual({ only_latest_data: true });
  });

  test("returns defaults for empty object", () => {
    const result = normalizeKyouArgs({});
    expect(result.limit).toBe(20);
    expect(result.max_size_mb).toBe(0.25);
    expect(result.is_include_timeis).toBe(false);
  });

  test("accepts custom limit", () => {
    const result = normalizeKyouArgs({ limit: 50 });
    expect(result.limit).toBe(50);
  });

  test("throws for limit below 1", () => {
    expect(() => normalizeKyouArgs({ limit: 0 })).toThrow(GkillApiError);
  });

  test("throws for limit above 1000", () => {
    expect(() => normalizeKyouArgs({ limit: 1001 })).toThrow(GkillApiError);
  });

  test("accepts custom max_size_mb", () => {
    const result = normalizeKyouArgs({ max_size_mb: 1.5 });
    expect(result.max_size_mb).toBe(1.5);
  });

  test("throws for max_size_mb <= 0", () => {
    expect(() => normalizeKyouArgs({ max_size_mb: 0 })).toThrow(GkillApiError);
    expect(() => normalizeKyouArgs({ max_size_mb: -1 })).toThrow(GkillApiError);
  });

  test("accepts is_include_timeis", () => {
    const result = normalizeKyouArgs({ is_include_timeis: true });
    expect(result.is_include_timeis).toBe(true);
  });

  test("accepts locale_name", () => {
    const result = normalizeKyouArgs({ locale_name: "en" });
    expect(result.locale_name).toBe("en");
  });

  test("accepts cursor as RFC3339", () => {
    const result = normalizeKyouArgs({ cursor: "2026-01-01T00:00:00+09:00" });
    expect(result.cursor).toBe("2026-01-01T00:00:00+09:00");
  });

  test("accepts cursor as date-only", () => {
    const result = normalizeKyouArgs({ cursor: "2026-01-01" });
    expect(result.cursor).toMatch(/^2026-01-01T00:00:00[+-]\d{2}:\d{2}$/);
  });

  test("passes query through normalizeKyouQuery", () => {
    const result = normalizeKyouArgs({ query: { use_tags: true, tags: ["test"] } });
    expect(result.query.use_tags).toBe(true);
    expect(result.query.tags).toEqual(["test"]);
    expect(result.query.only_latest_data).toBe(true);
  });

  test("throws for unknown top-level key", () => {
    expect(() => normalizeKyouArgs({ unknown_field: true })).toThrow(GkillApiError);
  });

  test("throws for non-object args", () => {
    expect(() => normalizeKyouArgs("string")).toThrow(GkillApiError);
  });

  test("ignores undefined locale_name", () => {
    const result = normalizeKyouArgs({ locale_name: undefined });
    expect(result).not.toHaveProperty("locale_name");
  });
});

// ---------------------------------------------------------------------------
// normalizeLocaleOnlyArgs
// ---------------------------------------------------------------------------
describe("normalizeLocaleOnlyArgs", () => {
  test("returns empty object for null args", () => {
    expect(normalizeLocaleOnlyArgs(null)).toEqual({});
  });

  test("returns empty object for undefined args", () => {
    expect(normalizeLocaleOnlyArgs(undefined)).toEqual({});
  });

  test("returns empty object for empty object", () => {
    expect(normalizeLocaleOnlyArgs({})).toEqual({});
  });

  test("returns locale_name when provided", () => {
    expect(normalizeLocaleOnlyArgs({ locale_name: "ja" })).toEqual({ locale_name: "ja" });
  });

  test("trims locale_name", () => {
    expect(normalizeLocaleOnlyArgs({ locale_name: "  en  " })).toEqual({ locale_name: "en" });
  });

  test("returns empty object when locale_name is undefined", () => {
    expect(normalizeLocaleOnlyArgs({ locale_name: undefined })).toEqual({});
  });

  test("throws for unknown key", () => {
    expect(() => normalizeLocaleOnlyArgs({ unknown: "x" })).toThrow(GkillApiError);
  });

  test("throws for non-object args", () => {
    expect(() => normalizeLocaleOnlyArgs("string")).toThrow(GkillApiError);
  });

  test("throws for empty locale_name", () => {
    expect(() => normalizeLocaleOnlyArgs({ locale_name: "" })).toThrow(GkillApiError);
  });
});

// ---------------------------------------------------------------------------
// normalizeGpsArgs
// ---------------------------------------------------------------------------
describe("normalizeGpsArgs", () => {
  test("normalizes date-only start_date and end_date", () => {
    const result = normalizeGpsArgs({ start_date: "2026-01-01", end_date: "2026-01-31" });
    expect(result.start_date).toMatch(/^2026-01-01T00:00:00[+-]\d{2}:\d{2}$/);
    expect(result.end_date).toMatch(/^2026-01-31T23:59:59[+-]\d{2}:\d{2}$/);
  });

  test("accepts RFC3339 start_date and end_date", () => {
    const result = normalizeGpsArgs({
      start_date: "2026-01-01T00:00:00+09:00",
      end_date: "2026-01-31T23:59:59+09:00",
    });
    expect(result.start_date).toBe("2026-01-01T00:00:00+09:00");
    expect(result.end_date).toBe("2026-01-31T23:59:59+09:00");
  });

  test("includes locale_name when provided", () => {
    const result = normalizeGpsArgs({
      start_date: "2026-01-01",
      end_date: "2026-01-31",
      locale_name: "en",
    });
    expect(result.locale_name).toBe("en");
  });

  test("omits locale_name when not provided", () => {
    const result = normalizeGpsArgs({ start_date: "2026-01-01", end_date: "2026-01-31" });
    expect(result).not.toHaveProperty("locale_name");
  });

  test("omits locale_name when undefined", () => {
    const result = normalizeGpsArgs({
      start_date: "2026-01-01",
      end_date: "2026-01-31",
      locale_name: undefined,
    });
    expect(result).not.toHaveProperty("locale_name");
  });

  test("throws for missing start_date", () => {
    expect(() => normalizeGpsArgs({ end_date: "2026-01-31" })).toThrow(GkillApiError);
  });

  test("throws for missing end_date", () => {
    expect(() => normalizeGpsArgs({ start_date: "2026-01-01" })).toThrow(GkillApiError);
  });

  test("throws for null args (missing required dates)", () => {
    expect(() => normalizeGpsArgs(null)).toThrow(GkillApiError);
  });

  test("throws for unknown key", () => {
    expect(() =>
      normalizeGpsArgs({ start_date: "2026-01-01", end_date: "2026-01-31", unknown: "x" }),
    ).toThrow(GkillApiError);
  });

  test("end_date uses endOfDay for date-only input", () => {
    const result = normalizeGpsArgs({ start_date: "2026-06-15", end_date: "2026-06-15" });
    expect(result.start_date).toMatch(/T00:00:00/);
    expect(result.end_date).toMatch(/T23:59:59/);
  });
});
