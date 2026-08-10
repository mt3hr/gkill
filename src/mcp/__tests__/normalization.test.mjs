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
  normalizeIdfFileArgs,
} from "../lib/normalization.mjs";
import {
  DEFAULT_INCLUDE_PLUGIN_CONTENT,
  DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
  DEFAULT_PLUGIN_CONTENT_FORMAT,
  LEGACY_USE_FLAG_KEYS,
  MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
} from "../lib/constants.mjs";

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
    const result = normalizeKyouQuery({ tags_and: true, is_deleted: false });
    expect(result.tags_and).toBe(true);
    expect(result.is_deleted).toBe(false);
    expect(result.only_latest_data).toBe(true);
  });

  test("throws for non-boolean in boolean field", () => {
    expect(() => normalizeKyouQuery({ words_and: "yes" })).toThrow(GkillApiError);
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

  test("skips empty string datetime fields", () => {
    const result = normalizeKyouQuery({ plaing_time: "", update_time: "" });
    expect(result).not.toHaveProperty("plaing_time");
    expect(result).not.toHaveProperty("update_time");
  });

  test("skips empty string datetime alongside valid fields", () => {
    const result = normalizeKyouQuery({
      calendar_start_date: "2026-03-18",
      plaing_time: "",
      update_time: "",
    });
    expect(result.calendar_start_date).toBeTruthy();
    expect(result).not.toHaveProperty("plaing_time");
    expect(result).not.toHaveProperty("update_time");
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

  test("throws for unknown keys", () => {
    expect(() => normalizeKyouQuery({ some_future_field: "value" })).toThrow(GkillApiError);
  });

  test("throws for prototype polluting keys", () => {
    // オブジェクトリテラルの __proto__ はプロトタイプ設定になり自身のプロパティにならないため、
    // 実際の経路と同じくJSONパースで作る
    expect(() => normalizeKyouQuery(JSON.parse('{"__proto__": {"polluted": true}}'))).toThrow(GkillApiError);
    expect(() => normalizeKyouQuery({ constructor: {} })).toThrow(GkillApiError);
    expect(() => normalizeKyouQuery({ prototype: {} })).toThrow(GkillApiError);
  });

  test("throws for non-object query", () => {
    expect(() => normalizeKyouQuery("not-an-object")).toThrow(GkillApiError);
    expect(() => normalizeKyouQuery(null)).toThrow(GkillApiError);
  });

  // --- null は「キー欠落」と同義 ---
  describe("null value handling", () => {
    test("skips null values entirely (filter not used)", () => {
      const result = normalizeKyouQuery({
        tags: null,
        words: null,
        calendar_start_date: null,
        mi_board_name: null,
        map_latitude: null,
        plaing_time: null,
        period_of_time_week_of_days: null,
      });
      expect(result).toEqual({ only_latest_data: true });
    });

    test("keeps non-null values alongside skipped nulls", () => {
      const result = normalizeKyouQuery({ tags: ["tagA"], words: null });
      expect(result.tags).toEqual(["tagA"]);
      expect(result).not.toHaveProperty("words");
    });

    test("preserves empty arrays (filter enabled but matches nothing)", () => {
      const result = normalizeKyouQuery({ tags: [], reps: [], ids: [] });
      expect(result.tags).toEqual([]);
      expect(result.reps).toEqual([]);
      expect(result.ids).toEqual([]);
    });
  });

  // --- plaing_time の "now" 展開 ---
  describe("plaing_time literal now", () => {
    test("expands \"now\" to the current local RFC3339 time", () => {
      const before = Date.now();
      const result = normalizeKyouQuery({ plaing_time: "now" });
      const after = Date.now();
      expect(result.plaing_time).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$/);
      const parsed = Date.parse(result.plaing_time);
      // 秒未満切り捨てぶんの1秒を許容する
      expect(parsed).toBeGreaterThanOrEqual(before - 1000);
      expect(parsed).toBeLessThanOrEqual(after + 1000);
    });

    test("trims surrounding whitespace before matching \"now\"", () => {
      const result = normalizeKyouQuery({ plaing_time: "  now  " });
      expect(result.plaing_time).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$/);
    });

    test("does not expand \"now\" for other datetime fields", () => {
      expect(() => normalizeKyouQuery({ update_time: "now" })).toThrow(GkillApiError);
    });
  });

  // --- timeis_words の自動補完 (サーバの複合ゲート対策) ---
  describe("timeis_words auto-completion", () => {
    test("adds timeis_words: [] when only timeis_tags is set", () => {
      const result = normalizeKyouQuery({ timeis_tags: ["tagB"] });
      expect(result.timeis_tags).toEqual(["tagB"]);
      expect(result.timeis_words).toEqual([]);
    });

    test("also applies when timeis_tags is an empty array", () => {
      const result = normalizeKyouQuery({ timeis_tags: [] });
      expect(result.timeis_words).toEqual([]);
    });

    test("does not overwrite an explicit timeis_words", () => {
      const result = normalizeKyouQuery({ timeis_tags: ["tagB"], timeis_words: ["keyword"] });
      expect(result.timeis_words).toEqual(["keyword"]);
    });

    test("does not add timeis_words when timeis_not_words already opens the gate", () => {
      const result = normalizeKyouQuery({ timeis_tags: ["tagB"], timeis_not_words: ["exclude"] });
      expect(result).not.toHaveProperty("timeis_words");
    });

    test("does not add timeis_words when legacy use_timeis:false removed timeis_tags", () => {
      const result = normalizeKyouQuery({ timeis_tags: ["tagB"], use_timeis: false });
      expect(result).not.toHaveProperty("timeis_tags");
      expect(result).not.toHaveProperty("timeis_words");
    });
  });

  // --- 旧 use_X フラグの後方互換受理 ---
  describe("legacy use_X flag acceptance", () => {
    test("use_X:true is dropped and the values stay authoritative", () => {
      const result = normalizeKyouQuery({ use_tags: true, tags: ["tagA"], use_words: true, words: ["w"] });
      expect(result).not.toHaveProperty("use_tags");
      expect(result).not.toHaveProperty("use_words");
      expect(result.tags).toEqual(["tagA"]);
      expect(result.words).toEqual(["w"]);
    });

    test("use_tags:false removes tags", () => {
      const result = normalizeKyouQuery({ use_tags: false, tags: ["tagA"] });
      expect(result).not.toHaveProperty("use_tags");
      expect(result).not.toHaveProperty("tags");
    });

    test("use_words:false removes words and not_words", () => {
      const result = normalizeKyouQuery({ use_words: false, words: ["w"], not_words: ["x"] });
      expect(result).not.toHaveProperty("words");
      expect(result).not.toHaveProperty("not_words");
    });

    test("use_reps:false and use_rep_types:false remove their lists", () => {
      const result = normalizeKyouQuery({
        use_reps: false,
        reps: ["rep1"],
        use_rep_types: false,
        rep_types: ["kmemo"],
      });
      expect(result).not.toHaveProperty("reps");
      expect(result).not.toHaveProperty("rep_types");
    });

    test("use_ids:false removes ids", () => {
      const result = normalizeKyouQuery({ use_ids: false, ids: ["abc123"] });
      expect(result).not.toHaveProperty("ids");
    });

    test("use_include_id is dropped without touching ids", () => {
      expect(normalizeKyouQuery({ use_include_id: true, ids: ["abc123"] }).ids).toEqual(["abc123"]);
      expect(normalizeKyouQuery({ use_include_id: false, ids: ["abc123"] }).ids).toEqual(["abc123"]);
    });

    test("use_timeis:false removes timeis_words, timeis_not_words and timeis_tags", () => {
      const result = normalizeKyouQuery({
        use_timeis: false,
        timeis_words: ["a"],
        timeis_not_words: ["b"],
        timeis_tags: ["c"],
      });
      expect(result).not.toHaveProperty("timeis_words");
      expect(result).not.toHaveProperty("timeis_not_words");
      expect(result).not.toHaveProperty("timeis_tags");
    });

    test("use_timeis_tags:false removes only timeis_tags", () => {
      const result = normalizeKyouQuery({
        use_timeis_tags: false,
        timeis_words: ["a"],
        timeis_tags: ["c"],
      });
      expect(result.timeis_words).toEqual(["a"]);
      expect(result).not.toHaveProperty("timeis_tags");
    });

    test("use_calendar:false removes calendar_start_date and calendar_end_date", () => {
      const result = normalizeKyouQuery({
        use_calendar: false,
        calendar_start_date: "2026-01-01",
        calendar_end_date: "2026-12-31",
      });
      expect(result).not.toHaveProperty("calendar_start_date");
      expect(result).not.toHaveProperty("calendar_end_date");
    });

    test("use_map:false removes map_latitude, map_longitude and map_radius", () => {
      const result = normalizeKyouQuery({
        use_map: false,
        map_latitude: 35.0,
        map_longitude: 135.0,
        map_radius: 100,
      });
      expect(result).not.toHaveProperty("map_latitude");
      expect(result).not.toHaveProperty("map_longitude");
      expect(result).not.toHaveProperty("map_radius");
    });

    test("use_plaing:false removes plaing_time", () => {
      const result = normalizeKyouQuery({ use_plaing: false, plaing_time: "2026-01-01" });
      expect(result).not.toHaveProperty("plaing_time");
    });

    test("use_update_time:false removes update_time", () => {
      const result = normalizeKyouQuery({ use_update_time: false, update_time: "2026-01-01" });
      expect(result).not.toHaveProperty("update_time");
    });

    test("use_mi_board_name:false removes mi_board_name", () => {
      const result = normalizeKyouQuery({ use_mi_board_name: false, mi_board_name: "board1" });
      expect(result).not.toHaveProperty("mi_board_name");
    });

    test("use_period_of_time:false removes the period_of_time value keys", () => {
      const result = normalizeKyouQuery({
        use_period_of_time: false,
        period_of_time_start_time_second: 0,
        period_of_time_end_time_second: 3600,
        period_of_time_week_of_days: [0, 6],
      });
      expect(result).not.toHaveProperty("period_of_time_start_time_second");
      expect(result).not.toHaveProperty("period_of_time_end_time_second");
      expect(result).not.toHaveProperty("period_of_time_week_of_days");
    });

    test("flag position relative to the values does not matter", () => {
      const result = normalizeKyouQuery({ tags: ["tagA"], use_tags: false });
      expect(result).not.toHaveProperty("tags");
    });

    test("null legacy flag is ignored (treated as omitted)", () => {
      const result = normalizeKyouQuery({ use_tags: null, tags: ["tagA"] });
      expect(result.tags).toEqual(["tagA"]);
    });

    test("throws for a non-boolean legacy flag value", () => {
      expect(() => normalizeKyouQuery({ use_tags: "yes" })).toThrow(GkillApiError);
    });

    test("unknown keys still throw despite legacy acceptance", () => {
      expect(() => normalizeKyouQuery({ use_unknown_thing: true })).toThrow(GkillApiError);
    });

    // use_mi_sort_type / use_mi_check_state は値キーを束ねないので、受理し損ねると
    // 「未知キー」として throw してしまう。旧クライアントが送る16キーはすべて受理する
    test("use_mi_sort_type and use_mi_check_state are accepted without dropping their values", () => {
      const result = normalizeKyouQuery({
        use_mi_sort_type: false,
        mi_sort_type: "limit_time",
        use_mi_check_state: false,
        mi_check_state: "checked",
      });
      expect(result).not.toHaveProperty("use_mi_sort_type");
      expect(result).not.toHaveProperty("use_mi_check_state");
      expect(result.mi_sort_type).toBe("limit_time");
      expect(result.mi_check_state).toBe("checked");
    });

    test("every legacy flag key is accepted (no unknown-key throw)", () => {
      for (const key of LEGACY_USE_FLAG_KEYS) {
        expect(() => normalizeKyouQuery({ [key]: true }), `${key} が受理されない`).not.toThrow();
        expect(() => normalizeKyouQuery({ [key]: false }), `${key} が受理されない`).not.toThrow();
      }
    });
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
    expect(result.query).not.toHaveProperty("use_tags");
    expect(result.query.tags).toEqual(["test"]);
    expect(result.query.only_latest_data).toBe(true);
  });

  test("throws for unknown top-level key", () => {
    expect(() => normalizeKyouArgs({ unknown_field: true })).toThrow(GkillApiError);
  });

  test("defaults the inline plugin content args", () => {
    const normalized = normalizeKyouArgs({});
    expect(normalized.include_plugin_content).toBe(DEFAULT_INCLUDE_PLUGIN_CONTENT);
    expect(normalized.plugin_content_max_text_length).toBe(DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH);
    expect(normalized.plugin_content_format).toBe(DEFAULT_PLUGIN_CONTENT_FORMAT);
  });

  test("accepts include_plugin_content", () => {
    expect(normalizeKyouArgs({ include_plugin_content: true }).include_plugin_content).toBe(true);
  });

  test("throws for a non-boolean include_plugin_content", () => {
    expect(() => normalizeKyouArgs({ include_plugin_content: "yes" })).toThrow(GkillApiError);
  });

  test("accepts an in-range plugin_content_max_text_length", () => {
    expect(normalizeKyouArgs({ plugin_content_max_text_length: 100 }).plugin_content_max_text_length).toBe(100);
    expect(
      normalizeKyouArgs({ plugin_content_max_text_length: MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH })
        .plugin_content_max_text_length,
    ).toBe(MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH);
  });

  test("throws for an out-of-range plugin_content_max_text_length", () => {
    expect(() => normalizeKyouArgs({ plugin_content_max_text_length: 0 })).toThrow(GkillApiError);
    expect(() =>
      normalizeKyouArgs({ plugin_content_max_text_length: MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH + 1 }),
    ).toThrow(GkillApiError);
  });

  test("lower-cases plugin_content_format", () => {
    expect(normalizeKyouArgs({ plugin_content_format: "HTML" }).plugin_content_format).toBe("html");
  });

  test("throws for an unknown plugin_content_format", () => {
    expect(() => normalizeKyouArgs({ plugin_content_format: "markdown" })).toThrow(/plugin_content_format/);
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

// ---------------------------------------------------------------------------
// normalizeIdfFileArgs
// ---------------------------------------------------------------------------
describe("normalizeIdfFileArgs", () => {
  test("accepts valid rep_name and file_name", () => {
    const result = normalizeIdfFileArgs({ rep_name: "my_repo", file_name: "photo.jpg" });
    expect(result.rep_name).toBe("my_repo");
    expect(result.file_name).toBe("photo.jpg");
    expect(result.locale_name).toBeUndefined();
  });

  test("includes locale_name when provided", () => {
    const result = normalizeIdfFileArgs({ rep_name: "repo", file_name: "f.txt", locale_name: "en" });
    expect(result.locale_name).toBe("en");
  });

  test("throws when rep_name is missing", () => {
    expect(() => normalizeIdfFileArgs({ file_name: "f.txt" })).toThrow();
  });

  test("throws when file_name is missing", () => {
    expect(() => normalizeIdfFileArgs({ rep_name: "repo" })).toThrow();
  });

  test("throws when rep_name is empty", () => {
    expect(() => normalizeIdfFileArgs({ rep_name: "  ", file_name: "f.txt" })).toThrow();
  });

  test("throws when file_name is empty", () => {
    expect(() => normalizeIdfFileArgs({ rep_name: "repo", file_name: "" })).toThrow();
  });

  test("trims whitespace from strings", () => {
    const result = normalizeIdfFileArgs({ rep_name: "  repo  ", file_name: "  file.txt  " });
    expect(result.rep_name).toBe("repo");
    expect(result.file_name).toBe("file.txt");
  });

  test("rejects unknown keys", () => {
    expect(() => normalizeIdfFileArgs({ rep_name: "r", file_name: "f", unknown: 1 })).toThrow();
  });

  test("accepts null args as empty object and throws for missing required", () => {
    expect(() => normalizeIdfFileArgs(null)).toThrow();
  });
});
