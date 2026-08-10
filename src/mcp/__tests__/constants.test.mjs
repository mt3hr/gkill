import {
  DEFAULT_KYOUS_LIMIT,
  DEFAULT_KYOUS_MAX_SIZE_MB,
  DEFAULT_KYOUS_INCLUDE_TIMEIS,
  RFC3339_REGEX,
  DATE_ONLY_REGEX,
  KYOUS_TOP_LEVEL_FIELDS,
  KYOUS_QUERY_BOOLEAN_FIELDS,
  KYOUS_QUERY_ALL_FIELDS,
  LEGACY_USE_FLAG_KEYS,
  KYOUS_QUERY_STRING_ARRAY_FIELDS,
  KYOUS_QUERY_NUMBER_FIELDS,
  KYOUS_QUERY_INTEGER_FIELDS,
  KYOUS_QUERY_DATETIME_FIELDS,
  MI_CHECK_STATES,
  MI_SORT_TYPES,
  PLUGIN_CONTENT_FORMATS,
  DEFAULT_PLUGIN_CONTENT_FORMAT,
  DEFAULT_INCLUDE_PLUGIN_CONTENT,
  DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
  MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
  MAX_INLINE_PLUGIN_CONTENT_KYOUS,
  INLINE_PLUGIN_CONTENT_TOTAL_TEXT_LENGTH,
  INLINE_PLUGIN_CONTENT_REP_CONCURRENCY,
  INLINE_PLUGIN_CONTENT_DEADLINE_MS,
  MAX_INLINE_PLUGIN_CONTENT_HTML_LENGTH,
} from "../lib/constants.mjs";

// ---------------------------------------------------------------------------
// DEFAULT_KYOUS_LIMIT
// ---------------------------------------------------------------------------
describe("DEFAULT_KYOUS_LIMIT", () => {
  test("equals 20", () => {
    expect(DEFAULT_KYOUS_LIMIT).toBe(20);
  });
});

// ---------------------------------------------------------------------------
// DEFAULT_KYOUS_MAX_SIZE_MB
// ---------------------------------------------------------------------------
describe("DEFAULT_KYOUS_MAX_SIZE_MB", () => {
  test("equals 0.25", () => {
    expect(DEFAULT_KYOUS_MAX_SIZE_MB).toBe(0.25);
  });
});

// ---------------------------------------------------------------------------
// DEFAULT_KYOUS_INCLUDE_TIMEIS
// ---------------------------------------------------------------------------
describe("DEFAULT_KYOUS_INCLUDE_TIMEIS", () => {
  test("equals false", () => {
    expect(DEFAULT_KYOUS_INCLUDE_TIMEIS).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// RFC3339_REGEX
// ---------------------------------------------------------------------------
describe("RFC3339_REGEX", () => {
  test("is a RegExp", () => {
    expect(RFC3339_REGEX).toBeInstanceOf(RegExp);
  });

  test("matches valid RFC 3339 datetime with timezone offset", () => {
    expect(RFC3339_REGEX.test("2026-02-25T10:30:00+09:00")).toBe(true);
  });

  test("matches valid RFC 3339 datetime with Z", () => {
    expect(RFC3339_REGEX.test("2026-02-25T10:30:00Z")).toBe(true);
  });

  test("matches datetime with fractional seconds", () => {
    expect(RFC3339_REGEX.test("2026-02-25T10:30:00.123+09:00")).toBe(true);
  });

  test("matches datetime with negative offset", () => {
    expect(RFC3339_REGEX.test("2026-02-25T10:30:00-05:00")).toBe(true);
  });

  test("does not match date-only string", () => {
    expect(RFC3339_REGEX.test("2026-02-25")).toBe(false);
  });

  test("does not match empty string", () => {
    expect(RFC3339_REGEX.test("")).toBe(false);
  });

  test("does not match plain text", () => {
    expect(RFC3339_REGEX.test("not-a-date")).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// DATE_ONLY_REGEX
// ---------------------------------------------------------------------------
describe("DATE_ONLY_REGEX", () => {
  test("is a RegExp", () => {
    expect(DATE_ONLY_REGEX).toBeInstanceOf(RegExp);
  });

  test("matches YYYY-MM-DD", () => {
    expect(DATE_ONLY_REGEX.test("2026-02-25")).toBe(true);
  });

  test("does not match full datetime", () => {
    expect(DATE_ONLY_REGEX.test("2026-02-25T10:30:00+09:00")).toBe(false);
  });

  test("does not match empty string", () => {
    expect(DATE_ONLY_REGEX.test("")).toBe(false);
  });

  test("does not match partial date", () => {
    expect(DATE_ONLY_REGEX.test("2026-02")).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// KYOUS_TOP_LEVEL_FIELDS
// ---------------------------------------------------------------------------
describe("KYOUS_TOP_LEVEL_FIELDS", () => {
  test("is a non-empty Set", () => {
    expect(KYOUS_TOP_LEVEL_FIELDS).toBeInstanceOf(Set);
    expect(KYOUS_TOP_LEVEL_FIELDS.size).toBeGreaterThan(0);
  });

  test("contains expected fields", () => {
    expect(KYOUS_TOP_LEVEL_FIELDS.has("query")).toBe(true);
    expect(KYOUS_TOP_LEVEL_FIELDS.has("locale_name")).toBe(true);
    expect(KYOUS_TOP_LEVEL_FIELDS.has("limit")).toBe(true);
    expect(KYOUS_TOP_LEVEL_FIELDS.has("cursor")).toBe(true);
    expect(KYOUS_TOP_LEVEL_FIELDS.has("max_size_mb")).toBe(true);
    expect(KYOUS_TOP_LEVEL_FIELDS.has("is_include_timeis")).toBe(true);
  });

  test("contains the inline plugin content fields", () => {
    expect(KYOUS_TOP_LEVEL_FIELDS.has("include_plugin_content")).toBe(true);
    expect(KYOUS_TOP_LEVEL_FIELDS.has("plugin_content_max_text_length")).toBe(true);
    expect(KYOUS_TOP_LEVEL_FIELDS.has("plugin_content_format")).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// インライン本文取得の定数
// ---------------------------------------------------------------------------
describe("inline plugin content constants", () => {
  test("defaults to opt-in", () => {
    expect(DEFAULT_INCLUDE_PLUGIN_CONTENT).toBe(false);
  });

  test("keeps the format enum and its default", () => {
    expect(Array.from(PLUGIN_CONTENT_FORMATS)).toEqual(["text", "html", "both"]);
    expect(PLUGIN_CONTENT_FORMATS.has(DEFAULT_PLUGIN_CONTENT_FORMAT)).toBe(true);
    expect(DEFAULT_PLUGIN_CONTENT_FORMAT).toBe("text");
  });

  test("orders the per-entry default below the per-entry maximum", () => {
    expect(DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH).toBeGreaterThan(0);
    expect(DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH).toBeLessThanOrEqual(
      MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
    );
  });

  test("lets a single entry at the maximum length fit the total budget", () => {
    expect(INLINE_PLUGIN_CONTENT_TOTAL_TEXT_LENGTH).toBeGreaterThanOrEqual(
      MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
    );
  });

  test("uses sane counts, concurrency and deadline", () => {
    expect(MAX_INLINE_PLUGIN_CONTENT_KYOUS).toBeGreaterThanOrEqual(1);
    expect(INLINE_PLUGIN_CONTENT_REP_CONCURRENCY).toBeGreaterThanOrEqual(1);
    expect(INLINE_PLUGIN_CONTENT_DEADLINE_MS).toBeGreaterThan(0);
    expect(MAX_INLINE_PLUGIN_CONTENT_HTML_LENGTH).toBeGreaterThan(
      MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
    );
  });
});

// ---------------------------------------------------------------------------
// KYOUS_QUERY_BOOLEAN_FIELDS
// ---------------------------------------------------------------------------
describe("KYOUS_QUERY_BOOLEAN_FIELDS", () => {
  test("is a non-empty Set", () => {
    expect(KYOUS_QUERY_BOOLEAN_FIELDS).toBeInstanceOf(Set);
    expect(KYOUS_QUERY_BOOLEAN_FIELDS.size).toBeGreaterThan(0);
  });

  test("contains expected boolean fields", () => {
    expect(KYOUS_QUERY_BOOLEAN_FIELDS.has("update_cache")).toBe(true);
    expect(KYOUS_QUERY_BOOLEAN_FIELDS.has("is_deleted")).toBe(true);
    expect(KYOUS_QUERY_BOOLEAN_FIELDS.has("is_image_only")).toBe(true);
    expect(KYOUS_QUERY_BOOLEAN_FIELDS.has("only_latest_data")).toBe(true);
  });

  test("no longer contains the removed use_X flags", () => {
    expect(KYOUS_QUERY_BOOLEAN_FIELDS.has("use_tags")).toBe(false);
    expect(KYOUS_QUERY_BOOLEAN_FIELDS.has("use_words")).toBe(false);
    expect(KYOUS_QUERY_BOOLEAN_FIELDS.has("use_calendar")).toBe(false);
    expect(KYOUS_QUERY_BOOLEAN_FIELDS.has("use_include_id")).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// LEGACY_USE_FLAG_KEYS
// ---------------------------------------------------------------------------
describe("LEGACY_USE_FLAG_KEYS", () => {
  // Go (find/find_query_legacy_json.go) と client
  // (classes/api/find_query/normalize-legacy-find-kyou-query-json.ts) と同じ16キー。
  // ここが欠けると、その旧フラグを送ってきたクライアントが未知キー扱いで弾かれる
  test("contains exactly the 16 removed use_X flags", () => {
    expect(LEGACY_USE_FLAG_KEYS).toBeInstanceOf(Set);
    expect(Array.from(LEGACY_USE_FLAG_KEYS).sort()).toEqual(
      [
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
        "use_mi_sort_type",
        "use_mi_check_state",
      ].sort(),
    );
  });

  test("does not overlap KYOUS_QUERY_BOOLEAN_FIELDS", () => {
    for (const key of LEGACY_USE_FLAG_KEYS) {
      expect(KYOUS_QUERY_BOOLEAN_FIELDS.has(key)).toBe(false);
    }
  });

  test("derived KYOUS_QUERY_ALL_FIELDS follows the removal (no use_X keys advertised)", () => {
    for (const key of LEGACY_USE_FLAG_KEYS) {
      expect(KYOUS_QUERY_ALL_FIELDS.has(key)).toBe(false);
    }
  });
});

// ---------------------------------------------------------------------------
// KYOUS_QUERY_STRING_ARRAY_FIELDS
// ---------------------------------------------------------------------------
describe("KYOUS_QUERY_STRING_ARRAY_FIELDS", () => {
  test("is a non-empty Set", () => {
    expect(KYOUS_QUERY_STRING_ARRAY_FIELDS).toBeInstanceOf(Set);
    expect(KYOUS_QUERY_STRING_ARRAY_FIELDS.size).toBeGreaterThan(0);
  });

  test("contains expected string array fields", () => {
    expect(KYOUS_QUERY_STRING_ARRAY_FIELDS.has("rep_types")).toBe(true);
    expect(KYOUS_QUERY_STRING_ARRAY_FIELDS.has("ids")).toBe(true);
    expect(KYOUS_QUERY_STRING_ARRAY_FIELDS.has("words")).toBe(true);
    expect(KYOUS_QUERY_STRING_ARRAY_FIELDS.has("tags")).toBe(true);
    expect(KYOUS_QUERY_STRING_ARRAY_FIELDS.has("reps")).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// KYOUS_QUERY_NUMBER_FIELDS
// ---------------------------------------------------------------------------
describe("KYOUS_QUERY_NUMBER_FIELDS", () => {
  test("is a non-empty Set", () => {
    expect(KYOUS_QUERY_NUMBER_FIELDS).toBeInstanceOf(Set);
    expect(KYOUS_QUERY_NUMBER_FIELDS.size).toBeGreaterThan(0);
  });

  test("contains map-related number fields", () => {
    expect(KYOUS_QUERY_NUMBER_FIELDS.has("map_radius")).toBe(true);
    expect(KYOUS_QUERY_NUMBER_FIELDS.has("map_latitude")).toBe(true);
    expect(KYOUS_QUERY_NUMBER_FIELDS.has("map_longitude")).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// KYOUS_QUERY_INTEGER_FIELDS
// ---------------------------------------------------------------------------
describe("KYOUS_QUERY_INTEGER_FIELDS", () => {
  test("is a non-empty Map", () => {
    expect(KYOUS_QUERY_INTEGER_FIELDS).toBeInstanceOf(Map);
    expect(KYOUS_QUERY_INTEGER_FIELDS.size).toBeGreaterThan(0);
  });

  test("contains period_of_time fields with min/max", () => {
    const startTime = KYOUS_QUERY_INTEGER_FIELDS.get("period_of_time_start_time_second");
    expect(startTime).toBeDefined();
    expect(startTime.min).toBe(0);
    expect(startTime.max).toBe(86399);

    const endTime = KYOUS_QUERY_INTEGER_FIELDS.get("period_of_time_end_time_second");
    expect(endTime).toBeDefined();
    expect(endTime.min).toBe(0);
    expect(endTime.max).toBe(86399);
  });
});

// ---------------------------------------------------------------------------
// KYOUS_QUERY_DATETIME_FIELDS
// ---------------------------------------------------------------------------
describe("KYOUS_QUERY_DATETIME_FIELDS", () => {
  test("is a non-empty Map", () => {
    expect(KYOUS_QUERY_DATETIME_FIELDS).toBeInstanceOf(Map);
    expect(KYOUS_QUERY_DATETIME_FIELDS.size).toBeGreaterThan(0);
  });

  test("contains calendar and plaing datetime fields", () => {
    expect(KYOUS_QUERY_DATETIME_FIELDS.has("calendar_start_date")).toBe(true);
    expect(KYOUS_QUERY_DATETIME_FIELDS.has("calendar_end_date")).toBe(true);
    expect(KYOUS_QUERY_DATETIME_FIELDS.has("plaing_time")).toBe(true);
    expect(KYOUS_QUERY_DATETIME_FIELDS.has("update_time")).toBe(true);
  });

  test("calendar_end_date has endOfDay=true", () => {
    const endDate = KYOUS_QUERY_DATETIME_FIELDS.get("calendar_end_date");
    expect(endDate.endOfDay).toBe(true);
    expect(endDate.allowDateOnly).toBe(true);
  });

  test("calendar_start_date has endOfDay=false", () => {
    const startDate = KYOUS_QUERY_DATETIME_FIELDS.get("calendar_start_date");
    expect(startDate.endOfDay).toBe(false);
    expect(startDate.allowDateOnly).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// MI_CHECK_STATES
// ---------------------------------------------------------------------------
describe("MI_CHECK_STATES", () => {
  test("is a non-empty Set", () => {
    expect(MI_CHECK_STATES).toBeInstanceOf(Set);
    expect(MI_CHECK_STATES.size).toBeGreaterThan(0);
  });

  test("contains all, checked, uncheck", () => {
    expect(MI_CHECK_STATES.has("all")).toBe(true);
    expect(MI_CHECK_STATES.has("checked")).toBe(true);
    expect(MI_CHECK_STATES.has("uncheck")).toBe(true);
  });

  test("has exactly 3 entries", () => {
    expect(MI_CHECK_STATES.size).toBe(3);
  });
});

// ---------------------------------------------------------------------------
// MI_SORT_TYPES
// ---------------------------------------------------------------------------
describe("MI_SORT_TYPES", () => {
  test("is a non-empty Set", () => {
    expect(MI_SORT_TYPES).toBeInstanceOf(Set);
    expect(MI_SORT_TYPES.size).toBeGreaterThan(0);
  });

  test("contains expected sort types", () => {
    expect(MI_SORT_TYPES.has("create_time")).toBe(true);
    expect(MI_SORT_TYPES.has("estimate_start_time")).toBe(true);
    expect(MI_SORT_TYPES.has("estimate_end_time")).toBe(true);
    expect(MI_SORT_TYPES.has("limit_time")).toBe(true);
  });

  test("has exactly 4 entries", () => {
    expect(MI_SORT_TYPES.size).toBe(4);
  });
});
