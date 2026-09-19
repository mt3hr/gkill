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
  normalizeAppConfigArgs,
  normalizeIdfFileArgs,
  normalizeRepNamesArgs,
  normalizeRepInfosArgs,
  normalizeKyouHistoryArgs,
  detectStaleSchemaSignals,
  staleSchemaWarning,
  appendStaleSchemaWarning,
  normalizeStatusArgs,
  STALE_SCHEMA_ARG_KINDS_BY_TOOL,
} from "../lib/normalization.mjs";
import { READ_TOOLS } from "../lib/read-tools.mjs";
import { FIND_QUERY_SCHEMA } from "../lib/find-query-schema.mjs";
import { encodeGpsCursor } from "../lib/gps-cursor.mjs";
import { paginateGpsLogs } from "../lib/read-handlers.mjs";
import {
  DEFAULT_INCLUDE_PLUGIN_CONTENT,
  DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
  DEFAULT_PLUGIN_CONTENT_FORMAT,
  LEGACY_USE_FLAG_KEYS,
  MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
} from "../lib/constants.mjs";
import { normalizeKftlArgs } from "../lib/write-normalization.mjs";

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
    const result = normalizeKyouQuery({ tags_and: true, include_deleted_data: false });
    expect(result.tags_and).toBe(true);
    expect(result.include_deleted_data).toBe(false);
    expect(result.only_latest_data).toBe(true);
  });

  test("throws for non-boolean in boolean field", () => {
    expect(() => normalizeKyouQuery({ words_and: "yes" })).toThrow(GkillApiError);
  });

  // is_deleted / hide_timeis_tags は Kyou 検索で一度も実装されたことがなく、
  // 送っているクライアントも実在しなかったので語彙ごと落とした。
  // 受理して黙って無視すると「フィルタを指定したつもりで何も起きない」になるため、
  // 未知キーとして弾き、allowed に正しい候補を添えて返すのが正しい。
  test("rejects the never-implemented is_deleted / hide_timeis_tags", () => {
    expect(() => normalizeKyouQuery({ is_deleted: true })).toThrow(GkillApiError);
    expect(() => normalizeKyouQuery({ hide_timeis_tags: ["x"] })).toThrow(GkillApiError);
    try {
      normalizeKyouQuery({ is_deleted: true });
    } catch (error) {
      expect(error.detail.allowed).toContain("include_deleted_data");
      expect(error.detail.allowed).not.toContain("is_deleted");
    }
  });

  test("validates string array fields", () => {
    const result = normalizeKyouQuery({ tags: ["tag1", " tag2 "] });
    expect(result.tags).toEqual(["tag1", "tag2"]);
  });

  test("throws for non-array in string array field", () => {
    expect(() => normalizeKyouQuery({ tags: "single-tag" })).toThrow(GkillApiError);
  });

  test("validates number fields", () => {
    const result = normalizeKyouQuery({ map_latitude: 35.6762, map_longitude: 139.6503, map_radius: 500 });
    expect(result.map_latitude).toBe(35.6762);
    expect(result.map_longitude).toBe(139.6503);
    expect(result.map_radius).toBe(500);
  });

  test("throws for non-number in number field", () => {
    expect(() => normalizeKyouQuery({ map_latitude: "35.6762", map_longitude: 139.65, map_radius: 500 })).toThrow(GkillApiError);
  });

  // 地図条件は3値揃わないと gkill が黙って無視する（map_latitude だけでも通常検索と同じ件数が返る。
  // 2026-09-18 の実利用報告）。入口で欠けた欄を名指しして断る（ADR-0625）。
  test("rejects a partial map filter and names the missing fields", () => {
    let thrown;
    try {
      normalizeKyouQuery({ map_latitude: 35.6762 });
    } catch (e) {
      thrown = e;
    }
    expect(thrown).toBeInstanceOf(GkillApiError);
    expect(thrown.detail?.field).toBe("query.map_longitude");
    expect(thrown.message).toContain("map_longitude, map_radius");
    expect(thrown.message).toContain("meters");
    expect(() => normalizeKyouQuery({ map_latitude: 35.6762, map_longitude: 139.65 })).toThrow(/query.map_radius/);
  });

  test("rejects out-of-range map values", () => {
    expect(() => normalizeKyouQuery({ map_latitude: 95, map_longitude: 139.65, map_radius: 500 })).toThrow(/between -90 and 90/);
    expect(() => normalizeKyouQuery({ map_latitude: 35, map_longitude: 200, map_radius: 500 })).toThrow(/between -180 and 180/);
    expect(() => normalizeKyouQuery({ map_latitude: 35, map_longitude: 139.65, map_radius: 0 })).toThrow(/greater than 0/);
  });

  // 未知キーは全部集めて1回で返し、廃止済み（only_latest_data / use_*）は allowed に載せない。
  test("reports every unknown query key at once and hides deprecated keys from allowed", () => {
    let thrown;
    try {
      normalizeKyouQuery({ wordz: ["a"], tagz: ["b"] });
    } catch (e) {
      thrown = e;
    }
    expect(thrown).toBeInstanceOf(GkillApiError);
    expect(thrown.detail?.field).toBe("query.wordz");
    expect(thrown.detail?.unknown).toEqual(["wordz", "tagz"]);
    expect(thrown.message).toContain("2 unknown names");
    expect(thrown.detail?.allowed).not.toContain("only_latest_data");
    expect(thrown.detail?.allowed).not.toContain("use_tags");
    expect(thrown.detail?.allowed).toContain("words");
  });

  test("names the missing timezone offset instead of just saying ISO-8601", () => {
    let thrown;
    try {
      normalizeKyouQuery({ calendar_start_date: "2026-09-18T00:00:00" });
    } catch (e) {
      thrown = e;
    }
    expect(thrown).toBeInstanceOf(GkillApiError);
    expect(thrown.message).toContain("no timezone offset");
    expect(thrown.message).toContain("2026-09-18T00:00:00+09:00");
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
    const result = normalizeKyouQuery({ playing_time: "", update_time: "" });
    expect(result).not.toHaveProperty("playing_time");
    expect(result).not.toHaveProperty("update_time");
  });

  test("skips empty string datetime alongside valid fields", () => {
    const result = normalizeKyouQuery({
      calendar_start_date: "2026-03-18",
      playing_time: "",
      update_time: "",
    });
    expect(result.calendar_start_date).toBeTruthy();
    expect(result).not.toHaveProperty("playing_time");
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
        playing_time: null,
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

  // --- playing_time の "now" 展開 ---
  describe("playing_time literal now", () => {
    test("expands \"now\" to the current local RFC3339 time", () => {
      const before = Date.now();
      const result = normalizeKyouQuery({ playing_time: "now" });
      const after = Date.now();
      expect(result.playing_time).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$/);
      const parsed = Date.parse(result.playing_time);
      // 秒未満切り捨てぶんの1秒を許容する
      expect(parsed).toBeGreaterThanOrEqual(before - 1000);
      expect(parsed).toBeLessThanOrEqual(after + 1000);
    });

    test("trims surrounding whitespace before matching \"now\"", () => {
      const result = normalizeKyouQuery({ playing_time: "  now  " });
      expect(result.playing_time).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$/);
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

    test("use_playing:false removes playing_time", () => {
      const result = normalizeKyouQuery({ use_playing: false, playing_time: "2026-01-01" });
      expect(result).not.toHaveProperty("playing_time");
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

  // v2: include_id / include_rep_name は廃止（id/rep_name は常時付与）。
  // 旧クライアント救済のため受理はするが、正規化結果には載せない（型検証のみ）。
  test("accepts but ignores deprecated include_rep_name", () => {
    expect(normalizeKyouArgs({ include_rep_name: true }).include_rep_name).toBeUndefined();
    expect(normalizeKyouArgs({ include_rep_name: false }).include_rep_name).toBeUndefined();
    expect(normalizeKyouArgs({ include_id: true }).include_id).toBeUndefined();
  });

  test("throws for non-boolean include_rep_name", () => {
    expect(() => normalizeKyouArgs({ include_rep_name: "yes" })).toThrow(GkillApiError);
  });

  test("accepts locale_name", () => {
    const result = normalizeKyouArgs({ locale_name: "en" });
    expect(result.locale_name).toBe("en");
  });

  test("accepts cursor as RFC3339", () => {
    const result = normalizeKyouArgs({ cursor: "2026-01-01T00:00:00+09:00" });
    expect(result.cursor).toBe("2026-01-01T00:00:00+09:00");
  });

  // v2: カーソルは不透明文字列。Node は日時として解釈せず素通しする
  // （複合形式 {RFC3339Nano}::{ID} を日時正規化に掛けると壊れるため）。
  // 旧形式（日付のみ等）の受理はサーバ(parseMCPCursor)の責務。
  test("passes cursor through verbatim (opaque, v2)", () => {
    expect(normalizeKyouArgs({ cursor: "2026-01-01" }).cursor).toBe("2026-01-01");
    const composite = "2026-08-01T20:00:00.123456789+09:00::3f9e40c1";
    expect(normalizeKyouArgs({ cursor: composite }).cursor).toBe(composite);
    expect(() => normalizeKyouArgs({ cursor: "x".repeat(600) })).toThrow(GkillApiError);
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

  test("accepts thumb in the WxH form the file route expects", () => {
    const result = normalizeIdfFileArgs({ rep_name: "repo", file_name: "photo.png", thumb: "1024x1024" });
    expect(result.thumb).toBe("1024x1024");
  });

  test("rejects thumb that is not WxH", () => {
    for (const thumb of ["1024", "1024*1024", "axb", "1024x", "12345x100"]) {
      expect(() => normalizeIdfFileArgs({ rep_name: "r", file_name: "f", thumb })).toThrow(GkillApiError);
    }
  });

  test("rejects thumb over the server cap instead of silently getting the original", () => {
    // Go 側は 1024 超えを受け取るとサムネを作らず原本を返す。
    expect(() => normalizeIdfFileArgs({ rep_name: "r", file_name: "f", thumb: "2048x2048" })).toThrow(
      /1024 per side/,
    );
    expect(() => normalizeIdfFileArgs({ rep_name: "r", file_name: "f", thumb: "100x2048" })).toThrow(
      /1024 per side/,
    );
  });

  test("accepts is_video together with thumb", () => {
    const result = normalizeIdfFileArgs({
      rep_name: "repo",
      file_name: "clip.mp4",
      thumb: "400x400",
      is_video: true,
    });
    expect(result.is_video).toBe(true);
    expect(result.thumb).toBe("400x400");
  });

  test("rejects is_video without thumb", () => {
    expect(() =>
      normalizeIdfFileArgs({ rep_name: "repo", file_name: "clip.mp4", is_video: true }),
    ).toThrow(/requires thumb/);
  });

  test("allows is_video:false on its own (it is a no-op, not a request for a frame)", () => {
    const result = normalizeIdfFileArgs({ rep_name: "repo", file_name: "f.txt", is_video: false });
    expect(result.is_video).toBe(false);
    expect(result.thumb).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// stale-schema revival (clients holding a pre-upgrade cached tool schema send
// post-upgrade top-level params as canonical-JSON strings; observed live on
// 2026-08-23 via the claude.ai connector)
// ---------------------------------------------------------------------------

describe("stale-schema argument revival", () => {
  test("revives canonical JSON strings for v2 get_kyous params", () => {
    const result = normalizeKyouArgs({
      count_only: "true",
      data_types: '["nlog","kc"]',
      num_min: "3.5",
      num_max: "10",
      idf_kinds: '["image"]',
      include_file_size: "false",
    });
    expect(result.count_only).toBe(true);
    expect(result.data_types).toEqual(["nlog", "kc"]);
    expect(result.num_min).toBe(3.5);
    expect(result.num_max).toBe(10);
    expect(result.idf_kinds).toEqual(["image"]);
    expect(result.include_file_size).toBe(false);
  });

  test("properly typed values still pass unchanged", () => {
    const result = normalizeKyouArgs({ count_only: true, num_min: 2 });
    expect(result.count_only).toBe(true);
    expect(result.num_min).toBe(2);
  });

  test("non-canonical strings still fail type validation", () => {
    expect(() => normalizeKyouArgs({ count_only: "TRUE" })).toThrow(GkillApiError);
    expect(() => normalizeKyouArgs({ count_only: "yes" })).toThrow(GkillApiError);
    expect(() => normalizeKyouArgs({ num_min: "abc" })).toThrow(GkillApiError);
    expect(() => normalizeKyouArgs({ data_types: "nlog" })).toThrow(GkillApiError);
    expect(() => normalizeKyouArgs({ data_types: '["nlog",1]' })).toThrow(GkillApiError);
    expect(() => normalizeKyouArgs({ data_types: "[broken" })).toThrow(GkillApiError);
  });

  test("revived idf_kinds still go through the allowed-value check", () => {
    expect(() => normalizeKyouArgs({ idf_kinds: '["bogus"]' })).toThrow(GkillApiError);
  });

  test("pre-v2 params are NOT revived (limit stays strict on get_kyous)", () => {
    expect(() => normalizeKyouArgs({ limit: "50" })).toThrow(GkillApiError);
  });

  test("revives gps paging params (limit is v2-added on the gps tool)", () => {
    const result = normalizeGpsArgs({
      start_date: "2026-07-01",
      end_date: "2026-07-02",
      limit: "250",
      count_only: "true",
    });
    expect(result.limit).toBe(250);
    expect(result.count_only).toBe(true);
  });

  test("revived gps limit still rejects non-integers and out-of-range values", () => {
    expect(() =>
      normalizeGpsArgs({ start_date: "2026-07-01", end_date: "2026-07-02", limit: "2.5" }),
    ).toThrow(GkillApiError);
    expect(() =>
      normalizeGpsArgs({ start_date: "2026-07-01", end_date: "2026-07-02", limit: "999999" }),
    ).toThrow(GkillApiError);
  });

  test("revives app config projection params", () => {
    const result = normalizeAppConfigArgs({ fields: '["tag_struct"]', include_ui_state: "true" });
    expect(result.fields).toEqual(["tag_struct"]);
    expect(result.include_ui_state).toBe(true);
  });

  test("revived app config fields still go through the allowed-value check", () => {
    expect(() => normalizeAppConfigArgs({ fields: '["bogus_field"]' })).toThrow(GkillApiError);
  });
});

describe("normalizeIdfFileArgs — 旧スキーマ救済 (2026-08-24 再監査 Q-01)", () => {
  // MCP のツール一覧はクライアントのセッション寿命で固定される。b303de73 で足した
  // is_video は boolean なので、旧スキーマのクライアントからは正規JSON文字列で届く。
  // reviveStaleSchemaArgs を通さないと、既存セッションからは動画サムネへ到達できない。
  test('revives is_video sent as the string "true" by a stale-schema client', () => {
    const result = normalizeIdfFileArgs({
      rep_name: "repo",
      file_name: "clip.mp4",
      thumb: "400x400",
      is_video: "true",
    });
    expect(result.is_video).toBe(true);
  });

  test('revives is_video sent as the string "false"', () => {
    const result = normalizeIdfFileArgs({
      rep_name: "repo",
      file_name: "photo.png",
      thumb: "400x400",
      is_video: "false",
    });
    expect(result.is_video).toBe(false);
  });

  test("still rejects values that are not a canonical JSON boolean", () => {
    for (const value of ["TRUE", "yes", "1", ""]) {
      expect(() =>
        normalizeIdfFileArgs({ rep_name: "r", file_name: "f", thumb: "10x10", is_video: value }),
      ).toThrow(GkillApiError);
    }
  });

  test("a revived is_video still requires thumb", () => {
    expect(() =>
      normalizeIdfFileArgs({ rep_name: "r", file_name: "f", is_video: "true" }),
    ).toThrow(/requires thumb/);
  });
});

describe("normalizeGpsArgs — 逆さまの期間 (2026-08-24 再監査 P-12)", () => {
  test("rejects start_date after end_date instead of silently matching nothing", () => {
    expect(() => normalizeGpsArgs({ start_date: "2026-08-24", end_date: "2026-08-01" })).toThrow(
      /must not be after end_date/,
    );
  });

  test("compares instants, not strings, so mixed offsets are judged correctly", () => {
    // 文字列比較だと "2026-08-24T00:00:00Z" > "2026-08-24T08:00:00+09:00" に見えるが、
    // 実際には後者(= 2026-08-23T23:00:00Z)のほうが前。
    expect(() =>
      normalizeGpsArgs({ start_date: "2026-08-24T00:00:00Z", end_date: "2026-08-24T08:00:00+09:00" }),
    ).toThrow(/must not be after end_date/);
  });

  test("accepts an ordinary range and a single day", () => {
    expect(normalizeGpsArgs({ start_date: "2026-08-01", end_date: "2026-08-24" }).start_date).toContain("2026-08-01");
    expect(() => normalizeGpsArgs({ start_date: "2026-08-24", end_date: "2026-08-24" })).not.toThrow();
  });
});

describe("cursor の形 (2026-08-24 再監査 P-36)", () => {
  // 壊れたカーソルは gkill 側で ERR000352「記録の取得に失敗しました」に畳まれ、
  // カーソルが原因だと分からなくなっていた。
  test("rejects a cursor that is neither an RFC3339 time nor time::id", () => {
    expect(() => normalizeKyouArgs({ cursor: "garbage-not-a-cursor" })).toThrow(/next_cursor/);
    expect(() =>
      normalizeGpsArgs({ start_date: "2026-08-01", end_date: "2026-08-24", cursor: "nonsense" }),
    ).toThrow(/next_cursor/);
  });

  test("accepts the v2 composite cursor and the legacy plain datetime", () => {
    expect(normalizeKyouArgs({ cursor: "2026-08-24T03:00:00+09:00::abc-123" }).cursor).toBe(
      "2026-08-24T03:00:00+09:00::abc-123",
    );
    expect(normalizeKyouArgs({ cursor: "2026-08-24T03:00:00+09:00" }).cursor).toBe(
      "2026-08-24T03:00:00+09:00",
    );
  });
});

describe("create_apps / update_apps (2026-08-24 再監査 事象7)", () => {
  // 「MCP で書いた記録」だけを絞る手段が無く、create_app は全レコードに入っているのに
  // 引く口だけが無かった。FindQuery には足していない（リポストのワード委譲が
  // 利用者クエリをそのまま下位検索へ流すので、SQL へ降ろすと黙って消える）
  test("accepts create_apps and update_apps as string arrays", () => {
    const result = normalizeKyouArgs({
      create_apps: ["gkill_mcp_readwrite"],
      update_apps: ["gkill_kftl"],
    });
    expect(result.create_apps).toEqual(["gkill_mcp_readwrite"]);
    expect(result.update_apps).toEqual(["gkill_kftl"]);
  });

  test("keeps the null semantics: omitted is no filter, [] matches nothing", () => {
    expect(normalizeKyouArgs({}).create_apps).toBeUndefined();
    expect(normalizeKyouArgs({ create_apps: [] }).create_apps).toEqual([]);
  });

  test("revives them when a stale-schema client sends them as a JSON string", () => {
    // 配列の新引数は revival の表に載せないと、旧スキーマのクライアントから
    // 文字列で届いて必ず型エラーになる
    const result = normalizeKyouArgs({ create_apps: '["gkill_mcp_write"]' });
    expect(result.create_apps).toEqual(["gkill_mcp_write"]);
  });

  test("rejects a non-array", () => {
    expect(() => normalizeKyouArgs({ create_apps: "gkill" })).toThrow(GkillApiError);
  });
});

// ---------------------------------------------------------------------------
// GPS cursor validation
//
// The GPS cursor is a Node-side base64url token, NOT the Go composite
// `{RFC3339Nano}::{ID}` token that gkill_get_kyous uses. normalizeGpsArgs used to
// carry a copy-pasted RFC3339 check, so passing next_cursor back verbatim — exactly
// what the tool description instructs — failed 100% of the time.
// 発行側と受理側が別実装だったのが原因なので、実物の encode 出力で検証する。
// ---------------------------------------------------------------------------

describe("normalizeGpsArgs — cursor", () => {
  const period = { start_date: "2026-08-23", end_date: "2026-08-24" };

  test("accepts a cursor produced by encodeGpsCursor", () => {
    const cursor = encodeGpsCursor("2026-08-23T23:51:41.000+00:00", 1);
    const result = normalizeGpsArgs({ ...period, cursor });
    expect(result.cursor).toBe(cursor);
  });

  test("rejects garbage and the get_kyous composite cursor", () => {
    expect(() => normalizeGpsArgs({ ...period, cursor: "nonsense" })).toThrow(GkillApiError);
    // Kyou 用の複合カーソルを GPS へ渡すのは呼び出し側の取り違えなので弾く
    expect(() =>
      normalizeGpsArgs({ ...period, cursor: "2026-08-24T17:00:00+09:00::cd97a1f1" }),
    ).toThrow(GkillApiError);
  });
});

// ---------------------------------------------------------------------------
// rep name filtering
// ---------------------------------------------------------------------------

describe("normalizeRepNamesArgs", () => {
  test("defaults limit and accepts contains", () => {
    expect(normalizeRepNamesArgs({ contains: "Fit" })).toEqual({ limit: 200, contains: "Fit" });
  });

  test("rejects unknown keys and out-of-range limit", () => {
    expect(() => normalizeRepNamesArgs({ bogus: 1 })).toThrow(GkillApiError);
    expect(() => normalizeRepNamesArgs({ limit: 0 })).toThrow(GkillApiError);
    expect(() => normalizeRepNamesArgs({ limit: 2001 })).toThrow(GkillApiError);
  });

  test("revives a stale-schema limit string", () => {
    expect(normalizeRepNamesArgs({ limit: "50" }).limit).toBe(50);
    expect(() => normalizeRepNamesArgs({ limit: "fifty" })).toThrow(GkillApiError);
  });
});

// gkill_get_kyou_history の limit も他の後付け引数と同じ穴を持っていた
// （救済表に載っておらず、旧セッションからは文字列で届いて型エラーになる）
describe("normalizeKyouHistoryArgs — stale-schema revival", () => {
  test("revives limit sent as a canonical JSON string", () => {
    const result = normalizeKyouHistoryArgs({ id: "abc", data_type: "kmemo", limit: "5" });
    expect(result.limit).toBe(5);
  });

  test("non-canonical strings still fail", () => {
    expect(() => normalizeKyouHistoryArgs({ id: "abc", data_type: "kmemo", limit: "five" })).toThrow(GkillApiError);
  });
});

// ---------------------------------------------------------------------------
// stale tool schema detection
//
// MCP tool schemas are frozen for the lifetime of a client session, so a
// server-side fix stays invisible until the client reconnects. Only emit the
// warning when staleness is PROVEN — a guess would make it constant noise.
// ---------------------------------------------------------------------------

describe("detectStaleSchemaSignals", () => {
  test("reports arguments that arrived as JSON strings", () => {
    const signals = detectStaleSchemaSignals("gkill_get_kyous", { data_types: '["nlog"]', count_only: "true" });
    expect(signals.revived.sort()).toEqual(["count_only", "data_types"]);
    expect(signals.deprecated).toEqual([]);
  });

  test("reports deprecated arguments, including legacy use_X query flags", () => {
    const signals = detectStaleSchemaSignals("gkill_get_kyous", {
      include_rep_name: false,
      query: { use_reps: true, only_latest_data: false, words: ["x"] },
    });
    expect(signals.revived).toEqual([]);
    expect(signals.deprecated.sort()).toEqual(["include_rep_name", "query.only_latest_data", "query.use_reps"]);
  });

  test("returns null for a current-schema call", () => {
    expect(detectStaleSchemaSignals("gkill_get_kyous", { data_types: ["nlog"], query: { words: ["x"] } })).toBeNull();
    expect(detectStaleSchemaSignals("gkill_get_gps_log", { start_date: "2026-08-01", end_date: "2026-08-02" })).toBeNull();
  });

  // string 型の引数は旧スキーマでも素通しするので、証拠にはならない
  test("does not treat string-typed arguments as evidence", () => {
    expect(detectStaleSchemaSignals("gkill_get_kyous", { group_by: "day", cursor: "x" })).toBeNull();
  });

  test("covers every tool that has a revival table (table-driven, cannot fall behind)", () => {
    // 以前はツールをハードコード列挙していて、表へ足した分だけこのテストが黙って
    // 古びていた (13 エントリ中 5 ツールしか見ていなかった)。表そのものを回し、
    // 全 (tool, key, kind) が「正規JSON文字列で届いたら古さの証拠として報告される」
    // ことを固定する。表へ1行足せば自動でここの検査対象になる。
    const SAMPLE_BY_KIND = {
      boolean: "true",
      number: "5",
      string_array: '["x"]',
      object_array: '[{"id":"a","data_type":"kmemo"}]',
    };
    expect(STALE_SCHEMA_ARG_KINDS_BY_TOOL.size).toBeGreaterThanOrEqual(13);
    for (const [tool, kinds] of STALE_SCHEMA_ARG_KINDS_BY_TOOL) {
      for (const [key, kind] of kinds) {
        const sample = SAMPLE_BY_KIND[kind];
        expect(sample, `no sample value for kind "${kind}" — add one to SAMPLE_BY_KIND`).toBeDefined();
        const signals = detectStaleSchemaSignals(tool, { [key]: sample });
        expect(signals?.revived ?? [], `${tool}.${key} (${kind}) not reported as revivable`).toContain(key);
      }
    }
  });
});

// gkill_get_kyou_history も同じ2語彙の橋を通す。
// ツール説明が「gkill_get_kyous の結果から data_type を取れ」と案内しているのに、
// その結果は射影名なので、案内どおりにすると落ちていた。
describe("normalizeKyouHistoryArgs — projection data_type", () => {
  test("folds projection names to entity types", () => {
    expect(normalizeKyouHistoryArgs({ id: "abc", data_type: "mi_create" }).data_type).toBe("mi");
    expect(normalizeKyouHistoryArgs({ id: "abc", data_type: "timeis_start" }).data_type).toBe("timeis");
    expect(normalizeKyouHistoryArgs({ id: "abc", data_type: "mirekyou_end" }).data_type).toBe("mirekyou");
  });

  test("entity types are unchanged and unknown values still throw", () => {
    expect(normalizeKyouHistoryArgs({ id: "abc", data_type: "kmemo" }).data_type).toBe("kmemo");
    expect(() => normalizeKyouHistoryArgs({ id: "abc", data_type: "bogus" })).toThrow(GkillApiError);
  });
});

// ---------------------------------------------------------------------------
// 逆さまのカレンダー期間
//
// gkill 側では0件になるだけで警告も出ず、「その期間に記録が無い」と読めてしまう。
// GPS 側 (normalizeGpsArgs) は元から入口で弾いていたので、そちらと揃える
// （2026-08-25 の実利用レビュー）。
// ---------------------------------------------------------------------------

describe("normalizeKyouQuery — inverted calendar range", () => {
  test("rejects start after end", () => {
    expect(() =>
      normalizeKyouQuery({ calendar_start_date: "2026-08-17", calendar_end_date: "2026-08-11" }),
    ).toThrow(GkillApiError);
  });

  test("accepts the same day on both ends", () => {
    // 日付のみは start=00:00:00 / end=23:59:59 へ展開されるので、同日は逆さまではない
    const result = normalizeKyouQuery({ calendar_start_date: "2026-08-11", calendar_end_date: "2026-08-11" });
    expect(result.calendar_start_date).toMatch(/^2026-08-11T00:00:00/);
    expect(result.calendar_end_date).toMatch(/^2026-08-11T23:59:59/);
  });

  test("accepts a normal range and one-sided ranges", () => {
    expect(() =>
      normalizeKyouQuery({ calendar_start_date: "2026-08-11", calendar_end_date: "2026-08-17" }),
    ).not.toThrow();
    expect(() => normalizeKyouQuery({ calendar_start_date: "2026-08-17" })).not.toThrow();
    expect(() => normalizeKyouQuery({ calendar_end_date: "2026-08-11" })).not.toThrow();
  });
});

describe("normalizeKyouArgs — 集計と cursor の併用", () => {
  // 併用は gkill 側でも弾かれるが、返るのは ERR000352「記録の取得に失敗しました」という
  // 汎用文で、理由が本文に一切乗らない（実測 2026-08-25: 検索失敗と区別が付かなかった）。
  // GPS 側は前から MCP 層で理由つきに弾いており、get_kyous だけが素通しだった。
  test("count_only + cursor を MCP 層で弾く", () => {
    expect(() => normalizeKyouArgs({ count_only: true, cursor: "2026-08-24T17:25:56+09:00::abc" }))
      .toThrow(/count_only[\s\S]*cannot be combined with cursor/);
  });

  test("group_by + cursor を MCP 層で弾く", () => {
    expect(() => normalizeKyouArgs({ group_by: "day", cursor: "2026-08-24T17:25:56+09:00::abc" }))
      .toThrow(/group_by[\s\S]*cannot be combined with cursor/);
  });

  test("cursor 単独・集計単独は通る", () => {
    expect(normalizeKyouArgs({ cursor: "2026-08-24T17:25:56+09:00::abc" }).cursor).toBe(
      "2026-08-24T17:25:56+09:00::abc",
    );
    expect(normalizeKyouArgs({ count_only: true }).count_only).toBe(true);
    expect(normalizeKyouArgs({ group_by: "day" }).group_by).toBe("day");
  });

  test("count_only:false は cursor と併用できる", () => {
    const result = normalizeKyouArgs({ count_only: false, cursor: "2026-08-24T17:25:56+09:00::abc" });
    expect(result.count_only).toBe(false);
    expect(result.cursor).toBe("2026-08-24T17:25:56+09:00::abc");
  });

  // count_only + group_by は、以前は count_only の早期 return が group_by を黙って捨て、
  // buckets の無い応答が「集計できた」顔で返っていた（2026-09-18 の実利用報告）。
  test("count_only + group_by を MCP 層で弾く（cursor 併用と同じ扱い）", () => {
    expect(() => normalizeKyouArgs({ count_only: true, group_by: "data_type" }))
      .toThrow(/count_only[\s\S]*cannot be combined with group_by/);
    // count_only:false は「集計しない」なので group_by と併用できる
    expect(normalizeKyouArgs({ count_only: false, group_by: "data_type" }).group_by).toBe("data_type");
  });

  test("count_only + group_by も GPS と同じ文言で弾く", () => {
    const gpsError = (() => {
      try {
        paginateGpsLogs([], { count_only: true, group_by: "day", limit: 1 });
      } catch (error) {
        return error;
      }
      return null;
    })();
    const kyouError = (() => {
      try {
        normalizeKyouArgs({ count_only: true, group_by: "day" });
      } catch (error) {
        return error;
      }
      return null;
    })();
    expect(gpsError).not.toBeNull();
    expect(kyouError).not.toBeNull();
    expect(gpsError.message).toBe(kyouError.message);
  });

  test("GPS と get_kyous が同じ規則・同じ文言で弾く", () => {
    const gpsError = (() => {
      try {
        paginateGpsLogs([], { count_only: true, cursor: encodeGpsCursor("2026-08-24T00:00:00+09:00", 0) });
      } catch (error) {
        return error;
      }
      return null;
    })();
    const kyouError = (() => {
      try {
        normalizeKyouArgs({ count_only: true, cursor: "2026-08-24T17:25:56+09:00::abc" });
      } catch (error) {
        return error;
      }
      return null;
    })();
    expect(gpsError).not.toBeNull();
    expect(kyouError).not.toBeNull();
    expect(gpsError.message).toBe(kyouError.message);
  });
});

describe("normalizeRepInfosArgs — data_kinds", () => {
  // attached_data_reps は歴代端末ぶんの Tag_ / Text_ / Notification_ / GPSLogs_ が並ぶので
  // 本番で約120件。fields は「配列を返すか返さないか」しか選べなかった。
  test("正しい data_kind を受理する", () => {
    expect(normalizeRepInfosArgs({ data_kinds: ["tag", "gpslog"] }).data_kinds).toEqual(["tag", "gpslog"]);
  });

  // 語彙が4つしかないので、黙って0件になると「その種別が無い」と読めてしまう。
  test("綴り違いは0件ではなくエラー", () => {
    expect(() => normalizeRepInfosArgs({ data_kinds: ["tags"] })).toThrow(/must be one of/);
    expect(() => normalizeRepInfosArgs({ data_kinds: ["Tag"] })).toThrow(/must be one of/);
  });

  test("省略時は絞り込まない", () => {
    expect(normalizeRepInfosArgs({}).data_kinds).toBeUndefined();
  });
});

// 行の絞り込み（2026-09-18 の実利用報告: fields で rep_infos[] を落としても Archived Git の rep 名・
// 歴代端末の GPSLogs_ / Tag_ / Text_ だけで数百行残り、「gkill_add_tag はどこへ書くか」に全部を読んでいた）。
describe("normalizeRepInfosArgs — 行の絞り込み", () => {
  test("writable_only / rep_types / rep_names / contains を受理する", () => {
    const normalized = normalizeRepInfosArgs({
      writable_only: true,
      rep_types: ["kmemo", "directory"],
      rep_names: ["Kmemo_pc"],
      contains: "tag_",
    });
    expect(normalized).toEqual({
      writable_only: true,
      rep_types: ["kmemo", "directory"],
      rep_names: ["Kmemo_pc"],
      contains: "tag_",
    });
  });

  test("型違いは弾く", () => {
    expect(() => normalizeRepInfosArgs({ writable_only: "yes" })).toThrow(GkillApiError);
    expect(() => normalizeRepInfosArgs({ rep_types: "kmemo" })).toThrow(GkillApiError);
    expect(() => normalizeRepInfosArgs({ rep_names: [1] })).toThrow(GkillApiError);
    expect(() => normalizeRepInfosArgs({ contains: "" })).toThrow(GkillApiError);
  });

  // 古い一覧を握るクライアントからは配列 / boolean が正規JSON文字列で届く。
  // data_kinds は 2026-08-25 の追加時に救済表へ載せ忘れていた。
  test("古スキーマ経由の文字列を復元する（data_kinds の載せ忘れも含む）", () => {
    const normalized = normalizeRepInfosArgs({
      data_kinds: '["tag"]',
      writable_only: "true",
      rep_types: '["kmemo"]',
      rep_names: '["Kmemo_pc"]',
    });
    expect(normalized.data_kinds).toEqual(["tag"]);
    expect(normalized.writable_only).toBe(true);
    expect(normalized.rep_types).toEqual(["kmemo"]);
    expect(normalized.rep_names).toEqual(["Kmemo_pc"]);
  });
});

describe("normalizeKftlArgs — idempotency_key", () => {
  // KFTL は DB トランザクションではないので、失敗して再送するたびに
  // 手前で書けたぶんが積む（実測で孤児 kmemo 4件）。受け口はサーバに前からあり、
  // Wear OS は送っていたのに MCP だけ送っていなかった。
  test("任意で受け取り、そのまま渡す", () => {
    expect(normalizeKftlArgs({ kftl_text: "メモ", idempotency_key: "k-1" }).idempotency_key).toBe("k-1");
  });

  test("省略できる", () => {
    expect(normalizeKftlArgs({ kftl_text: "メモ" }).idempotency_key).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// 未知の引数名（2026-09-14 レビュー P0）
//
// ChatGPT が改名前の検索条件名を、握ったままの古い一覧どおりに送って
// 「is not supported」だけを受け取った。タイプミスと古い一覧はサーバから区別できないので、
// 両方の可能性と再接続・gkill_status の照合を案内する。文言はトップレベルと query で同じ正本。
// ---------------------------------------------------------------------------
describe("unknown argument names point at a possibly stale tool list", () => {
  const EXPECTED = /is not supported.*misspelled.*tool list.*stale.*reconnect the MCP client.*gkill_status.*schema_revision/s;

  test("query-level unknown key", () => {
    let caught;
    try {
      normalizeKyouQuery({ no_such_field: "2026-09-14" });
    } catch (error) {
      caught = error;
    }
    expect(caught).toBeInstanceOf(GkillApiError);
    expect(caught.message).toMatch(/^Invalid argument 'query\.no_such_field': /);
    expect(caught.message).toMatch(EXPECTED);
    expect(caught.detail.field).toBe("query.no_such_field");
    expect(caught.detail.allowed).toContain("playing_time");
  });

  test("top-level unknown key uses the same wording", () => {
    let caught;
    try {
      normalizeKyouArgs({ no_such_arg: true });
    } catch (error) {
      caught = error;
    }
    expect(caught.message).toMatch(/^Invalid argument 'arguments\.no_such_arg': /);
    expect(caught.message).toMatch(EXPECTED);
  });

  test("the stale-schema warning also names gkill_status as the way to compare revisions", () => {
    const text = staleSchemaWarning({ revived: ["count_only"], deprecated: [] });
    expect(text).toMatch(/gkill_status.*schema_revision/s);
  });
});

// ---------------------------------------------------------------------------
// 廃止済み引数は公開スキーマに載せず、受理と古さの検出だけ残す（ADR-0620）
// ---------------------------------------------------------------------------
describe("deprecated arguments are accepted at runtime but absent from the advertised schema", () => {
  const kyousTool = READ_TOOLS.find((tool) => tool.name === "gkill_get_kyous");

  test("include_id / include_rep_name / only_latest_data are not advertised", () => {
    expect(kyousTool.inputSchema.properties).not.toHaveProperty("include_id");
    expect(kyousTool.inputSchema.properties).not.toHaveProperty("include_rep_name");
    expect(FIND_QUERY_SCHEMA.properties).not.toHaveProperty("only_latest_data");
    for (const key of Object.keys(FIND_QUERY_SCHEMA.properties)) {
      expect(key.startsWith("use_"), `legacy flag ${key} must not be advertised`).toBe(false);
    }
    // 説明文も旧 use_X フラグを説明しない（「受理する」と書けば AI は使う）
    expect(FIND_QUERY_SCHEMA.description).not.toMatch(/use_X/);
  });

  test("they are still accepted, and their arrival is reported as stale-schema evidence", () => {
    const normalized = normalizeKyouArgs({
      include_id: true,
      include_rep_name: false,
      query: { only_latest_data: false, use_words: true, words: ["x"] },
    });
    expect(normalized.query).toEqual({ words: ["x"], only_latest_data: true });
    const payload = appendStaleSchemaWarning({ kyous: [] }, "gkill_get_kyous", {
      include_id: true,
      include_rep_name: false,
      query: { only_latest_data: false, use_words: true, words: ["x"] },
    });
    expect(payload.warnings).toHaveLength(1);
    expect(payload.warnings[0]).toMatch(/deprecated arguments were sent: include_id, include_rep_name, query\.only_latest_data, query\.use_words/);
  });
});

// ---------------------------------------------------------------------------
// normalizeStatusArgs — 引数を取らない
// ---------------------------------------------------------------------------
describe("normalizeStatusArgs", () => {
  test("accepts undefined / null / {}", () => {
    expect(normalizeStatusArgs(undefined)).toEqual({});
    expect(normalizeStatusArgs(null)).toEqual({});
    expect(normalizeStatusArgs({})).toEqual({});
  });

  test("rejects any argument (so the tool never lands in the stale-schema revival table)", () => {
    expect(() => normalizeStatusArgs({ locale_name: "ja" })).toThrow(/arguments\.locale_name.*is not supported/);
    expect(STALE_SCHEMA_ARG_KINDS_BY_TOOL.has("gkill_status")).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// 2026-09-19 の MCP 実利用報告への対応（ADR-0626 / 0627 / 0629 / 0630）
// ---------------------------------------------------------------------------
describe("for_mi default projection, opt-in flags and history offset", () => {
  test("normalizeKyouArgs assumes include_create_mi for a bare for_mi and records a note", () => {
    const result = normalizeKyouArgs({ query: { for_mi: true } });
    expect(result.query.include_create_mi).toBe(true);
    expect(result.notes).toHaveLength(1);
    expect(result.notes[0]).toContain("include_create_mi:true was assumed");
    expect(normalizeKyouArgs({ query: { for_mi: true, include_check_mi: true } }).notes).toBeUndefined();
    expect(normalizeKyouArgs({ query: { for_mi: true, include_check_mi: true } }).query.include_create_mi).toBeUndefined();
    expect(normalizeKyouArgs({ query: {} }).notes).toBeUndefined();
  });

  test("normalizeKyouArgs defaults include_attached_ids / include_file_urls to false and revives stale strings", () => {
    const defaults = normalizeKyouArgs({});
    expect(defaults.include_attached_ids).toBe(false);
    expect(defaults.include_file_urls).toBe(false);
    expect(normalizeKyouArgs({ include_attached_ids: true, include_file_urls: true }).include_file_urls).toBe(true);
    expect(() => normalizeKyouArgs({ include_file_urls: "yes" })).toThrow(GkillApiError);
    // 古い一覧を握るクライアントからは正規JSON文字列で届く
    expect(normalizeKyouArgs({ include_attached_ids: "true" }).include_attached_ids).toBe(true);
  });

  test("normalizeKyouHistoryArgs accepts offset (also as a stale-schema string) and rejects a negative one", () => {
    expect(normalizeKyouHistoryArgs({ id: "a", data_type: "kmemo" }).offset).toBe(0);
    expect(normalizeKyouHistoryArgs({ id: "a", data_type: "kmemo", offset: 40 }).offset).toBe(40);
    expect(normalizeKyouHistoryArgs({ id: "a", data_type: "kmemo", offset: "40" }).offset).toBe(40);
    expect(() => normalizeKyouHistoryArgs({ id: "a", data_type: "kmemo", offset: -1 })).toThrow(GkillApiError);
  });

  test("top-level unknown keys hide deprecated names from allowed", () => {
    let thrown;
    try {
      normalizeKyouArgs({ limitt: 3 });
    } catch (e) {
      thrown = e;
    }
    expect(thrown).toBeInstanceOf(GkillApiError);
    expect(thrown.detail.allowed).not.toContain("include_id");
    expect(thrown.detail.allowed).toContain("include_attached_ids");
  });
});
