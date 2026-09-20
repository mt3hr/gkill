package mcp

// normalization.go（読み取りツールの引数正規化）の検査。

import (
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// containsValue は配列に値が含まれるか（toContain）。
func containsValue(items []any, want any) bool {
	for _, item := range items {
		if jsonobj.Equal(item, want) {
			return true
		}
	}
	return false
}

// mustNormalize(t)(Normalize...(...)) の形で使う: 正規化が通ることを要求して結果を返す。
func mustNormalize(t testing.TB) func(*jsonobj.Object, error) *jsonobj.Object {
	return func(result *jsonobj.Object, err error) *jsonobj.Object {
		t.Helper()
		expectNoError(t, err)
		return result
	}
}

// mustFail(t)(Normalize...(...)) の形で使う: GkillApiError で落ちることを要求してそのエラーを返す。
func mustFail(t testing.TB) func(*jsonobj.Object, error) *GkillApiError {
	return func(_ *jsonobj.Object, err error) *GkillApiError {
		t.Helper()
		return expectGkillApiError(t, err)
	}
}

// ---------------------------------------------------------------------------
// pad2
// ---------------------------------------------------------------------------
func TestPad2(t *testing.T) {
	t.Run("pads single digit", func(t *testing.T) {
		expectEqual(t, Pad2(1), "01")
		expectEqual(t, Pad2(0), "00")
		expectEqual(t, Pad2(9), "09")
	})

	t.Run("does not pad double digit", func(t *testing.T) {
		expectEqual(t, Pad2(10), "10")
		expectEqual(t, Pad2(59), "59")
	})

	t.Run("handles string input", func(t *testing.T) {
		expectEqual(t, Pad2("5"), "05")
	})
}

// ---------------------------------------------------------------------------
// formatLocalRfc3339
// ---------------------------------------------------------------------------
func TestFormatLocalRfc3339(t *testing.T) {
	t.Run("formats a date into RFC3339 with local timezone offset", func(t *testing.T) {
		result := FormatLocalRfc3339(time.Date(2026, 1, 15, 10, 30, 0, 0, time.Local))
		// Should match pattern YYYY-MM-DDTHH:MM:SS+HH:MM or -HH:MM
		mustMatch(t, result, `^2026-01-15T10:30:00[+-]\d{2}:\d{2}$`)
	})

	t.Run("formats midnight correctly", func(t *testing.T) {
		result := FormatLocalRfc3339(time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local))
		mustMatch(t, result, `^2026-06-01T00:00:00[+-]\d{2}:\d{2}$`)
	})

	t.Run("formats end of day correctly", func(t *testing.T) {
		result := FormatLocalRfc3339(time.Date(2026, 12, 31, 23, 59, 59, 0, time.Local))
		mustMatch(t, result, `^2026-12-31T23:59:59[+-]\d{2}:\d{2}$`)
	})
}

// ---------------------------------------------------------------------------
// normalizeDateOnlyToRfc3339
// ---------------------------------------------------------------------------
func TestNormalizeDateOnlyToRfc3339(t *testing.T) {
	t.Run("converts date-only to RFC3339 at start of day", func(t *testing.T) {
		result, ok := NormalizeDateOnlyToRfc3339("2026-03-15", false)
		expectTrue(t, ok, "returned null")
		mustMatch(t, result, `^2026-03-15T00:00:00[+-]\d{2}:\d{2}$`)
	})

	t.Run("converts date-only to RFC3339 at end of day", func(t *testing.T) {
		result, ok := NormalizeDateOnlyToRfc3339("2026-03-15", true)
		expectTrue(t, ok, "returned null")
		mustMatch(t, result, `^2026-03-15T23:59:59[+-]\d{2}:\d{2}$`)
	})

	t.Run("returns null for non-matching input", func(t *testing.T) {
		_, ok := NormalizeDateOnlyToRfc3339("not-a-date", false)
		expectTrue(t, !ok, "accepted")
	})

	t.Run("returns null for invalid date like Feb 30", func(t *testing.T) {
		_, ok := NormalizeDateOnlyToRfc3339("2026-02-30", false)
		expectTrue(t, !ok, "accepted")
	})

	t.Run("returns null for month 13", func(t *testing.T) {
		_, ok := NormalizeDateOnlyToRfc3339("2026-13-01", false)
		expectTrue(t, !ok, "accepted")
	})

	t.Run("handles leap year Feb 29", func(t *testing.T) {
		// 2024 is a leap year
		result, ok := NormalizeDateOnlyToRfc3339("2024-02-29", false)
		expectTrue(t, ok, "returned null")
		mustMatch(t, result, `^2024-02-29T00:00:00[+-]\d{2}:\d{2}$`)
	})

	t.Run("rejects non-leap year Feb 29", func(t *testing.T) {
		_, ok := NormalizeDateOnlyToRfc3339("2026-02-29", false)
		expectTrue(t, !ok, "accepted")
	})
}

// ---------------------------------------------------------------------------
// normalizeDateTimeString
// ---------------------------------------------------------------------------
func TestNormalizeDateTimeString(t *testing.T) {
	t.Run("accepts valid RFC3339 datetime", func(t *testing.T) {
		input := "2026-02-25T10:30:00+09:00"
		result, err := NormalizeDateTimeString(input, "field", DateTimeOptions{})
		expectNoError(t, err)
		expectEqual(t, result, input)
	})

	t.Run("accepts UTC datetime with Z", func(t *testing.T) {
		input := "2026-01-01T00:00:00Z"
		result, err := NormalizeDateTimeString(input, "field", DateTimeOptions{})
		expectNoError(t, err)
		expectEqual(t, result, input)
	})

	t.Run("accepts datetime with negative offset", func(t *testing.T) {
		input := "2026-06-15T08:00:00-05:00"
		result, err := NormalizeDateTimeString(input, "field", DateTimeOptions{})
		expectNoError(t, err)
		expectEqual(t, result, input)
	})

	t.Run("accepts datetime with fractional seconds", func(t *testing.T) {
		input := "2026-03-01T12:00:00.123+09:00"
		result, err := NormalizeDateTimeString(input, "field", DateTimeOptions{})
		expectNoError(t, err)
		expectEqual(t, result, input)
	})

	t.Run("throws for date-only when allowDateOnly is false", func(t *testing.T) {
		_, err := NormalizeDateTimeString("2026-03-15", "field", DateTimeOptions{})
		expectGkillApiError(t, err)
	})

	t.Run("converts date-only when allowDateOnly is true", func(t *testing.T) {
		result, err := NormalizeDateTimeString("2026-03-15", "field", DateTimeOptions{AllowDateOnly: true})
		expectNoError(t, err)
		mustMatch(t, result, `^2026-03-15T00:00:00[+-]\d{2}:\d{2}$`)
	})

	t.Run("converts date-only to end of day when endOfDay is true", func(t *testing.T) {
		result, err := NormalizeDateTimeString("2026-03-15", "field", DateTimeOptions{AllowDateOnly: true, EndOfDay: true})
		expectNoError(t, err)
		mustMatch(t, result, `^2026-03-15T23:59:59[+-]\d{2}:\d{2}$`)
	})

	t.Run("throws for non-string", func(t *testing.T) {
		_, err := NormalizeDateTimeString(12345, "field", DateTimeOptions{})
		expectGkillApiError(t, err)
	})

	t.Run("throws for empty string", func(t *testing.T) {
		_, err := NormalizeDateTimeString("", "field", DateTimeOptions{})
		expectGkillApiError(t, err)
	})

	t.Run("throws for invalid format", func(t *testing.T) {
		_, err := NormalizeDateTimeString("March 15, 2026", "field", DateTimeOptions{})
		expectGkillApiError(t, err)
	})

	t.Run("throws for invalid date-only (Feb 30) even with allowDateOnly", func(t *testing.T) {
		_, err := NormalizeDateTimeString("2026-02-30", "field", DateTimeOptions{AllowDateOnly: true})
		expectGkillApiError(t, err)
	})
}

// ---------------------------------------------------------------------------
// normalizeKyouQuery
// ---------------------------------------------------------------------------
func TestNormalizeKyouQuery(t *testing.T) {
	t.Run("returns object with only_latest_data=true for empty query", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouQuery(obj()))
		expectEqual(t, result, obj("only_latest_data", true))
	})

	t.Run("validates boolean fields", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouQuery(obj("tags_and", true, "include_deleted_data", false)))
		expectEqual(t, result.Value("tags_and"), true)
		expectEqual(t, result.Value("include_deleted_data"), false)
		expectEqual(t, result.Value("only_latest_data"), true)
	})

	t.Run("throws for non-boolean in boolean field", func(t *testing.T) {
		mustFail(t)(NormalizeKyouQuery(obj("words_and", "yes")))
	})

	// is_deleted / hide_timeis_tags は Kyou 検索で一度も実装されたことがなく、
	// 送っているクライアントも実在しなかったので語彙ごと落とした。
	// 受理して黙って無視すると「フィルタを指定したつもりで何も起きない」になるため、
	// 未知キーとして弾き、allowed に正しい候補を添えて返すのが正しい。
	t.Run("rejects the never-implemented is_deleted / hide_timeis_tags", func(t *testing.T) {
		mustFail(t)(NormalizeKyouQuery(obj("is_deleted", true)))
		mustFail(t)(NormalizeKyouQuery(obj("hide_timeis_tags", strs("x"))))
		apiErr := mustFail(t)(NormalizeKyouQuery(obj("is_deleted", true)))
		allowed := arrAt(t, apiErr.Detail, "allowed")
		expectTrue(t, containsValue(allowed, "include_deleted_data"), "allowed lacks include_deleted_data")
		expectTrue(t, !containsValue(allowed, "is_deleted"), "allowed contains is_deleted")
	})

	t.Run("validates string array fields", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouQuery(obj("tags", strs("tag1", " tag2 "))))
		expectEqual(t, result.Value("tags"), strs("tag1", "tag2"))
	})

	t.Run("throws for non-array in string array field", func(t *testing.T) {
		mustFail(t)(NormalizeKyouQuery(obj("tags", "single-tag")))
	})

	t.Run("validates number fields", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouQuery(obj("map_latitude", 35.6762, "map_longitude", 139.6503, "map_radius", 500)))
		expectEqual(t, result.Value("map_latitude"), 35.6762)
		expectEqual(t, result.Value("map_longitude"), 139.6503)
		expectEqual(t, result.Value("map_radius"), 500)
	})

	t.Run("throws for non-number in number field", func(t *testing.T) {
		mustFail(t)(NormalizeKyouQuery(obj("map_latitude", "35.6762", "map_longitude", 139.65, "map_radius", 500)))
	})

	// 地図条件は3値揃わないと gkill が黙って無視する（map_latitude だけでも通常検索と同じ件数が返る。
	// 2026-09-18 の実利用報告）。入口で欠けた欄を名指しして断る（ADR-0625）。
	t.Run("rejects a partial map filter and names the missing fields", func(t *testing.T) {
		thrown := mustFail(t)(NormalizeKyouQuery(obj("map_latitude", 35.6762)))
		expectEqual(t, thrown.Detail.Value("field"), "query.map_longitude")
		mustContain(t, thrown.Message, "map_longitude, map_radius")
		mustContain(t, thrown.Message, "meters")
		_, err := NormalizeKyouQuery(obj("map_latitude", 35.6762, "map_longitude", 139.65))
		expectErrorMatches(t, err, `query.map_radius`)
	})

	t.Run("rejects out-of-range map values", func(t *testing.T) {
		_, err := NormalizeKyouQuery(obj("map_latitude", 95, "map_longitude", 139.65, "map_radius", 500))
		expectErrorMatches(t, err, `between -90 and 90`)
		_, err = NormalizeKyouQuery(obj("map_latitude", 35, "map_longitude", 200, "map_radius", 500))
		expectErrorMatches(t, err, `between -180 and 180`)
		_, err = NormalizeKyouQuery(obj("map_latitude", 35, "map_longitude", 139.65, "map_radius", 0))
		expectErrorMatches(t, err, `greater than 0`)
	})

	// 未知キーは全部集めて1回で返し、廃止済み（only_latest_data / use_*）は allowed に載せない。
	t.Run("reports every unknown query key at once and hides deprecated keys from allowed", func(t *testing.T) {
		thrown := mustFail(t)(NormalizeKyouQuery(obj("wordz", strs("a"), "tagz", strs("b"))))
		expectEqual(t, thrown.Detail.Value("field"), "query.wordz")
		expectEqual(t, thrown.Detail.Value("unknown"), strs("wordz", "tagz"))
		mustContain(t, thrown.Message, "2 unknown names")
		allowed := arrAt(t, thrown.Detail, "allowed")
		expectTrue(t, !containsValue(allowed, "only_latest_data"), "allowed contains only_latest_data")
		expectTrue(t, !containsValue(allowed, "use_tags"), "allowed contains use_tags")
		expectTrue(t, containsValue(allowed, "words"), "allowed lacks words")
	})

	t.Run("names the missing timezone offset instead of just saying ISO-8601", func(t *testing.T) {
		thrown := mustFail(t)(NormalizeKyouQuery(obj("calendar_start_date", "2026-09-18T00:00:00")))
		mustContain(t, thrown.Message, "no timezone offset")
		mustContain(t, thrown.Message, "2026-09-18T00:00:00+09:00")
	})

	t.Run("validates integer fields with min/max", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouQuery(obj("period_of_time_start_time_second", 0)))
		expectEqual(t, result.Value("period_of_time_start_time_second"), 0)
	})

	t.Run("throws for out-of-range integer field", func(t *testing.T) {
		mustFail(t)(NormalizeKyouQuery(obj("period_of_time_start_time_second", 86400)))
		mustFail(t)(NormalizeKyouQuery(obj("period_of_time_start_time_second", -1)))
	})

	t.Run("validates datetime fields", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouQuery(obj("calendar_start_date", "2026-01-01")))
		mustMatch(t, strAt(t, result, "calendar_start_date"), `^2026-01-01T00:00:00[+-]\d{2}:\d{2}$`)
	})

	t.Run("calendar_end_date uses endOfDay", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouQuery(obj("calendar_end_date", "2026-12-31")))
		mustMatch(t, strAt(t, result, "calendar_end_date"), `^2026-12-31T23:59:59[+-]\d{2}:\d{2}$`)
	})

	t.Run("skips empty string datetime fields", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouQuery(obj("playing_time", "", "update_time", "")))
		expectTrue(t, !result.Has("playing_time"), "playing_time kept")
		expectTrue(t, !result.Has("update_time"), "update_time kept")
	})

	t.Run("skips empty string datetime alongside valid fields", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouQuery(obj(
			"calendar_start_date", "2026-03-18",
			"playing_time", "",
			"update_time", "",
		)))
		expectTrue(t, jsTruthy(result.Value("calendar_start_date")), "calendar_start_date missing")
		expectTrue(t, !result.Has("playing_time"), "playing_time kept")
		expectTrue(t, !result.Has("update_time"), "update_time kept")
	})

	t.Run("validates period_of_time_week_of_days as integer array", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouQuery(obj("period_of_time_week_of_days", arr(0, 3, 6))))
		expectEqual(t, result.Value("period_of_time_week_of_days"), arr(0, 3, 6))
	})

	t.Run("throws for out-of-range weekday", func(t *testing.T) {
		mustFail(t)(NormalizeKyouQuery(obj("period_of_time_week_of_days", arr(7))))
	})

	t.Run("validates mi_board_name", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouQuery(obj("mi_board_name", " board1 ")))
		expectEqual(t, result.Value("mi_board_name"), "board1")
	})

	t.Run("validates mi_check_state", func(t *testing.T) {
		for _, state := range []string{"all", "checked", "uncheck"} {
			expectEqual(t, mustNormalize(t)(NormalizeKyouQuery(obj("mi_check_state", state))).Value("mi_check_state"), state)
		}
	})

	t.Run("throws for invalid mi_check_state", func(t *testing.T) {
		mustFail(t)(NormalizeKyouQuery(obj("mi_check_state", "unknown")))
	})

	t.Run("validates mi_sort_type", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouQuery(obj("mi_sort_type", "create_time"))).Value("mi_sort_type"), "create_time")
		expectEqual(t, mustNormalize(t)(NormalizeKyouQuery(obj("mi_sort_type", "limit_time"))).Value("mi_sort_type"), "limit_time")
	})

	t.Run("throws for invalid mi_sort_type", func(t *testing.T) {
		mustFail(t)(NormalizeKyouQuery(obj("mi_sort_type", "invalid")))
	})

	t.Run("always forces only_latest_data=true even if set to false", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouQuery(obj("only_latest_data", false)))
		expectEqual(t, result.Value("only_latest_data"), true)
	})

	t.Run("throws for unknown keys", func(t *testing.T) {
		mustFail(t)(NormalizeKyouQuery(obj("some_future_field", "value")))
	})

	t.Run("throws for prototype polluting keys", func(t *testing.T) {
		// 実際の経路と同じく JSON パースで作る
		mustFail(t)(NormalizeKyouQuery(parseObj(t, `{"__proto__": {"polluted": true}}`)))
		mustFail(t)(NormalizeKyouQuery(obj("constructor", obj())))
		mustFail(t)(NormalizeKyouQuery(obj("prototype", obj())))
	})

	t.Run("throws for non-object query", func(t *testing.T) {
		mustFail(t)(NormalizeKyouQuery("not-an-object"))
		mustFail(t)(NormalizeKyouQuery(nil))
	})

	// --- null は「キー欠落」と同義 ---
	t.Run("null value handling", func(t *testing.T) {
		t.Run("skips null values entirely (filter not used)", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj(
				"tags", nil,
				"words", nil,
				"calendar_start_date", nil,
				"mi_board_name", nil,
				"map_latitude", nil,
				"playing_time", nil,
				"period_of_time_week_of_days", nil,
			)))
			expectEqual(t, result, obj("only_latest_data", true))
		})

		t.Run("keeps non-null values alongside skipped nulls", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("tags", strs("tagA"), "words", nil)))
			expectEqual(t, result.Value("tags"), strs("tagA"))
			expectTrue(t, !result.Has("words"), "words kept")
		})

		t.Run("preserves empty arrays (filter enabled but matches nothing)", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("tags", arr(), "reps", arr(), "ids", arr())))
			expectEqual(t, result.Value("tags"), arr())
			expectEqual(t, result.Value("reps"), arr())
			expectEqual(t, result.Value("ids"), arr())
		})
	})

	// --- playing_time の "now" 展開 ---
	t.Run("playing_time literal now", func(t *testing.T) {
		t.Run(`expands "now" to the current local RFC3339 time`, func(t *testing.T) {
			before := time.Now()
			result := mustNormalize(t)(NormalizeKyouQuery(obj("playing_time", "now")))
			after := time.Now()
			mustMatch(t, strAt(t, result, "playing_time"), `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$`)
			parsed, err := time.Parse(time.RFC3339, strAt(t, result, "playing_time"))
			expectNoError(t, err)
			// 秒未満切り捨てぶんの1秒を許容する
			expectTrue(t, !parsed.Before(before.Add(-time.Second)), "too early")
			expectTrue(t, !parsed.After(after.Add(time.Second)), "too late")
		})

		t.Run(`trims surrounding whitespace before matching "now"`, func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("playing_time", "  now  ")))
			mustMatch(t, strAt(t, result, "playing_time"), `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$`)
		})

		t.Run(`does not expand "now" for other datetime fields`, func(t *testing.T) {
			mustFail(t)(NormalizeKyouQuery(obj("update_time", "now")))
		})
	})

	// --- timeis_words の自動補完 (サーバの複合ゲート対策) ---
	t.Run("timeis_words auto-completion", func(t *testing.T) {
		t.Run("adds timeis_words: [] when only timeis_tags is set", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("timeis_tags", strs("tagB"))))
			expectEqual(t, result.Value("timeis_tags"), strs("tagB"))
			expectEqual(t, result.Value("timeis_words"), arr())
		})

		t.Run("also applies when timeis_tags is an empty array", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("timeis_tags", arr())))
			expectEqual(t, result.Value("timeis_words"), arr())
		})

		t.Run("does not overwrite an explicit timeis_words", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("timeis_tags", strs("tagB"), "timeis_words", strs("keyword"))))
			expectEqual(t, result.Value("timeis_words"), strs("keyword"))
		})

		t.Run("does not add timeis_words when timeis_not_words already opens the gate", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("timeis_tags", strs("tagB"), "timeis_not_words", strs("exclude"))))
			expectTrue(t, !result.Has("timeis_words"), "timeis_words added")
		})

		t.Run("does not add timeis_words when legacy use_timeis:false removed timeis_tags", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("timeis_tags", strs("tagB"), "use_timeis", false)))
			expectTrue(t, !result.Has("timeis_tags"), "timeis_tags kept")
			expectTrue(t, !result.Has("timeis_words"), "timeis_words added")
		})
	})

	// --- 旧 use_X フラグの後方互換受理 ---
	t.Run("legacy use_X flag acceptance", func(t *testing.T) {
		t.Run("use_X:true is dropped and the values stay authoritative", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("use_tags", true, "tags", strs("tagA"), "use_words", true, "words", strs("w"))))
			expectTrue(t, !result.Has("use_tags"), "use_tags kept")
			expectTrue(t, !result.Has("use_words"), "use_words kept")
			expectEqual(t, result.Value("tags"), strs("tagA"))
			expectEqual(t, result.Value("words"), strs("w"))
		})

		t.Run("use_tags:false removes tags", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("use_tags", false, "tags", strs("tagA"))))
			expectTrue(t, !result.Has("use_tags"), "use_tags kept")
			expectTrue(t, !result.Has("tags"), "tags kept")
		})

		t.Run("use_words:false removes words and not_words", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("use_words", false, "words", strs("w"), "not_words", strs("x"))))
			expectTrue(t, !result.Has("words"), "words kept")
			expectTrue(t, !result.Has("not_words"), "not_words kept")
		})

		t.Run("use_reps:false and use_rep_types:false remove their lists", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj(
				"use_reps", false,
				"reps", strs("rep1"),
				"use_rep_types", false,
				"rep_types", strs("kmemo"),
			)))
			expectTrue(t, !result.Has("reps"), "reps kept")
			expectTrue(t, !result.Has("rep_types"), "rep_types kept")
		})

		t.Run("use_ids:false removes ids", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("use_ids", false, "ids", strs("abc123"))))
			expectTrue(t, !result.Has("ids"), "ids kept")
		})

		t.Run("use_include_id is dropped without touching ids", func(t *testing.T) {
			expectEqual(t, mustNormalize(t)(NormalizeKyouQuery(obj("use_include_id", true, "ids", strs("abc123")))).Value("ids"), strs("abc123"))
			expectEqual(t, mustNormalize(t)(NormalizeKyouQuery(obj("use_include_id", false, "ids", strs("abc123")))).Value("ids"), strs("abc123"))
		})

		t.Run("use_timeis:false removes timeis_words, timeis_not_words and timeis_tags", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj(
				"use_timeis", false,
				"timeis_words", strs("a"),
				"timeis_not_words", strs("b"),
				"timeis_tags", strs("c"),
			)))
			expectTrue(t, !result.Has("timeis_words"), "timeis_words kept")
			expectTrue(t, !result.Has("timeis_not_words"), "timeis_not_words kept")
			expectTrue(t, !result.Has("timeis_tags"), "timeis_tags kept")
		})

		t.Run("use_timeis_tags:false removes only timeis_tags", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj(
				"use_timeis_tags", false,
				"timeis_words", strs("a"),
				"timeis_tags", strs("c"),
			)))
			expectEqual(t, result.Value("timeis_words"), strs("a"))
			expectTrue(t, !result.Has("timeis_tags"), "timeis_tags kept")
		})

		t.Run("use_calendar:false removes calendar_start_date and calendar_end_date", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj(
				"use_calendar", false,
				"calendar_start_date", "2026-01-01",
				"calendar_end_date", "2026-12-31",
			)))
			expectTrue(t, !result.Has("calendar_start_date"), "calendar_start_date kept")
			expectTrue(t, !result.Has("calendar_end_date"), "calendar_end_date kept")
		})

		t.Run("use_map:false removes map_latitude, map_longitude and map_radius", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj(
				"use_map", false,
				"map_latitude", 35.0,
				"map_longitude", 135.0,
				"map_radius", 100,
			)))
			expectTrue(t, !result.Has("map_latitude"), "map_latitude kept")
			expectTrue(t, !result.Has("map_longitude"), "map_longitude kept")
			expectTrue(t, !result.Has("map_radius"), "map_radius kept")
		})

		t.Run("use_playing:false removes playing_time", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("use_playing", false, "playing_time", "2026-01-01")))
			expectTrue(t, !result.Has("playing_time"), "playing_time kept")
		})

		t.Run("use_update_time:false removes update_time", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("use_update_time", false, "update_time", "2026-01-01")))
			expectTrue(t, !result.Has("update_time"), "update_time kept")
		})

		t.Run("use_mi_board_name:false removes mi_board_name", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("use_mi_board_name", false, "mi_board_name", "board1")))
			expectTrue(t, !result.Has("mi_board_name"), "mi_board_name kept")
		})

		t.Run("use_period_of_time:false removes the period_of_time value keys", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj(
				"use_period_of_time", false,
				"period_of_time_start_time_second", 0,
				"period_of_time_end_time_second", 3600,
				"period_of_time_week_of_days", arr(0, 6),
			)))
			expectTrue(t, !result.Has("period_of_time_start_time_second"), "start kept")
			expectTrue(t, !result.Has("period_of_time_end_time_second"), "end kept")
			expectTrue(t, !result.Has("period_of_time_week_of_days"), "week_of_days kept")
		})

		t.Run("flag position relative to the values does not matter", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("tags", strs("tagA"), "use_tags", false)))
			expectTrue(t, !result.Has("tags"), "tags kept")
		})

		t.Run("null legacy flag is ignored (treated as omitted)", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj("use_tags", nil, "tags", strs("tagA"))))
			expectEqual(t, result.Value("tags"), strs("tagA"))
		})

		t.Run("throws for a non-boolean legacy flag value", func(t *testing.T) {
			mustFail(t)(NormalizeKyouQuery(obj("use_tags", "yes")))
		})

		t.Run("unknown keys still throw despite legacy acceptance", func(t *testing.T) {
			mustFail(t)(NormalizeKyouQuery(obj("use_unknown_thing", true)))
		})

		// use_mi_sort_type / use_mi_check_state は値キーを束ねないので、受理し損ねると
		// 「未知キー」として throw してしまう。旧クライアントが送る16キーはすべて受理する
		t.Run("use_mi_sort_type and use_mi_check_state are accepted without dropping their values", func(t *testing.T) {
			result := mustNormalize(t)(NormalizeKyouQuery(obj(
				"use_mi_sort_type", false,
				"mi_sort_type", "limit_time",
				"use_mi_check_state", false,
				"mi_check_state", "checked",
			)))
			expectTrue(t, !result.Has("use_mi_sort_type"), "use_mi_sort_type kept")
			expectTrue(t, !result.Has("use_mi_check_state"), "use_mi_check_state kept")
			expectEqual(t, result.Value("mi_sort_type"), "limit_time")
			expectEqual(t, result.Value("mi_check_state"), "checked")
		})

		t.Run("every legacy flag key is accepted (no unknown-key throw)", func(t *testing.T) {
			for _, key := range LegacyUseFlagKeys.Values() {
				_, err := NormalizeKyouQuery(obj(key, true))
				expectTrue(t, err == nil, "%s が受理されない: %v", key, err)
				_, err = NormalizeKyouQuery(obj(key, false))
				expectTrue(t, err == nil, "%s が受理されない: %v", key, err)
			}
		})
	})
}

// ---------------------------------------------------------------------------
// normalizeKyouArgs
// ---------------------------------------------------------------------------
func TestNormalizeKyouArgs(t *testing.T) {
	t.Run("returns defaults for null/undefined args", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouArgs(nil))
		expectEqual(t, result.Value("limit"), 20)
		expectEqual(t, result.Value("max_size_mb"), 0.25)
		expectEqual(t, result.Value("is_include_timeis"), false)
		expectEqual(t, result.Value("query"), obj("only_latest_data", true))
	})

	t.Run("returns defaults for empty object", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouArgs(obj()))
		expectEqual(t, result.Value("limit"), 20)
		expectEqual(t, result.Value("max_size_mb"), 0.25)
		expectEqual(t, result.Value("is_include_timeis"), false)
	})

	t.Run("accepts custom limit", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("limit", 50))).Value("limit"), 50)
	})

	t.Run("throws for limit below 1", func(t *testing.T) {
		mustFail(t)(NormalizeKyouArgs(obj("limit", 0)))
	})

	t.Run("throws for limit above 1000", func(t *testing.T) {
		mustFail(t)(NormalizeKyouArgs(obj("limit", 1001)))
	})

	t.Run("accepts custom max_size_mb", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("max_size_mb", 1.5))).Value("max_size_mb"), 1.5)
	})

	t.Run("throws for max_size_mb <= 0", func(t *testing.T) {
		mustFail(t)(NormalizeKyouArgs(obj("max_size_mb", 0)))
		mustFail(t)(NormalizeKyouArgs(obj("max_size_mb", -1)))
	})

	t.Run("accepts is_include_timeis", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("is_include_timeis", true))).Value("is_include_timeis"), true)
	})

	// v2: include_id / include_rep_name は廃止（id/rep_name は常時付与）。
	// 旧クライアント救済のため受理はするが、正規化結果には載せない（型検証のみ）。
	t.Run("accepts but ignores deprecated include_rep_name", func(t *testing.T) {
		expectTrue(t, !mustNormalize(t)(NormalizeKyouArgs(obj("include_rep_name", true))).Defined("include_rep_name"), "include_rep_name kept")
		expectTrue(t, !mustNormalize(t)(NormalizeKyouArgs(obj("include_rep_name", false))).Defined("include_rep_name"), "include_rep_name kept")
		expectTrue(t, !mustNormalize(t)(NormalizeKyouArgs(obj("include_id", true))).Defined("include_id"), "include_id kept")
	})

	t.Run("throws for non-boolean include_rep_name", func(t *testing.T) {
		mustFail(t)(NormalizeKyouArgs(obj("include_rep_name", "yes")))
	})

	t.Run("accepts locale_name", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("locale_name", "en"))).Value("locale_name"), "en")
	})

	t.Run("accepts cursor as RFC3339", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("cursor", "2026-01-01T00:00:00+09:00"))).Value("cursor"), "2026-01-01T00:00:00+09:00")
	})

	// v2: カーソルは不透明文字列。日時として解釈せず素通しする
	// （複合形式 {RFC3339Nano}::{ID} を日時正規化に掛けると壊れるため）。
	// 旧形式（日付のみ等）の受理はサーバ(parseMCPCursor)の責務。
	t.Run("passes cursor through verbatim (opaque, v2)", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("cursor", "2026-01-01"))).Value("cursor"), "2026-01-01")
		composite := "2026-08-01T20:00:00.123456789+09:00::3f9e40c1"
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("cursor", composite))).Value("cursor"), composite)
		mustFail(t)(NormalizeKyouArgs(obj("cursor", strings.Repeat("x", 600))))
	})

	t.Run("passes query through normalizeKyouQuery", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouArgs(obj("query", obj("use_tags", true, "tags", strs("test")))))
		query := objAt(t, result, "query")
		expectTrue(t, !query.Has("use_tags"), "use_tags kept")
		expectEqual(t, query.Value("tags"), strs("test"))
		expectEqual(t, query.Value("only_latest_data"), true)
	})

	t.Run("throws for unknown top-level key", func(t *testing.T) {
		mustFail(t)(NormalizeKyouArgs(obj("unknown_field", true)))
	})

	t.Run("defaults the inline plugin content args", func(t *testing.T) {
		normalized := mustNormalize(t)(NormalizeKyouArgs(obj()))
		expectEqual(t, normalized.Value("include_plugin_content"), DefaultIncludePluginContent)
		expectEqual(t, normalized.Value("plugin_content_max_text_length"), DefaultInlinePluginContentMaxTextLength)
		expectEqual(t, normalized.Value("plugin_content_format"), DefaultPluginContentFormat)
	})

	t.Run("accepts include_plugin_content", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("include_plugin_content", true))).Value("include_plugin_content"), true)
	})

	t.Run("throws for a non-boolean include_plugin_content", func(t *testing.T) {
		mustFail(t)(NormalizeKyouArgs(obj("include_plugin_content", "yes")))
	})

	t.Run("accepts an in-range plugin_content_max_text_length", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("plugin_content_max_text_length", 100))).Value("plugin_content_max_text_length"), 100)
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("plugin_content_max_text_length", MaxPluginContentMaxTextLength))).Value("plugin_content_max_text_length"), MaxPluginContentMaxTextLength)
	})

	t.Run("throws for an out-of-range plugin_content_max_text_length", func(t *testing.T) {
		mustFail(t)(NormalizeKyouArgs(obj("plugin_content_max_text_length", 0)))
		mustFail(t)(NormalizeKyouArgs(obj("plugin_content_max_text_length", MaxPluginContentMaxTextLength+1)))
	})

	t.Run("lower-cases plugin_content_format", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("plugin_content_format", "HTML"))).Value("plugin_content_format"), "html")
	})

	t.Run("throws for an unknown plugin_content_format", func(t *testing.T) {
		_, err := NormalizeKyouArgs(obj("plugin_content_format", "markdown"))
		expectErrorMatches(t, err, `plugin_content_format`)
	})

	t.Run("throws for non-object args", func(t *testing.T) {
		mustFail(t)(NormalizeKyouArgs("string"))
	})

	t.Run("ignores undefined locale_name", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouArgs(obj("locale_name", jsonobj.Undefined)))
		expectTrue(t, !result.Defined("locale_name"), "locale_name set")
	})
}

// ---------------------------------------------------------------------------
// normalizeLocaleOnlyArgs
// ---------------------------------------------------------------------------
func TestNormalizeLocaleOnlyArgs(t *testing.T) {
	t.Run("returns empty object for null args", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeLocaleOnlyArgs(nil)), obj())
	})

	t.Run("returns empty object for undefined args", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeLocaleOnlyArgs(jsonobj.Undefined)), obj())
	})

	t.Run("returns empty object for empty object", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeLocaleOnlyArgs(obj())), obj())
	})

	t.Run("returns locale_name when provided", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeLocaleOnlyArgs(obj("locale_name", "ja"))), obj("locale_name", "ja"))
	})

	t.Run("trims locale_name", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeLocaleOnlyArgs(obj("locale_name", "  en  "))), obj("locale_name", "en"))
	})

	t.Run("returns empty object when locale_name is undefined", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeLocaleOnlyArgs(obj("locale_name", jsonobj.Undefined))), obj())
	})

	t.Run("throws for unknown key", func(t *testing.T) {
		mustFail(t)(NormalizeLocaleOnlyArgs(obj("unknown", "x")))
	})

	t.Run("throws for non-object args", func(t *testing.T) {
		mustFail(t)(NormalizeLocaleOnlyArgs("string"))
	})

	t.Run("throws for empty locale_name", func(t *testing.T) {
		mustFail(t)(NormalizeLocaleOnlyArgs(obj("locale_name", "")))
	})
}

// ---------------------------------------------------------------------------
// normalizeGpsArgs
// ---------------------------------------------------------------------------
func TestNormalizeGpsArgs(t *testing.T) {
	t.Run("normalizes date-only start_date and end_date", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeGpsArgs(obj("start_date", "2026-01-01", "end_date", "2026-01-31")))
		mustMatch(t, strAt(t, result, "start_date"), `^2026-01-01T00:00:00[+-]\d{2}:\d{2}$`)
		mustMatch(t, strAt(t, result, "end_date"), `^2026-01-31T23:59:59[+-]\d{2}:\d{2}$`)
	})

	t.Run("accepts RFC3339 start_date and end_date", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeGpsArgs(obj(
			"start_date", "2026-01-01T00:00:00+09:00",
			"end_date", "2026-01-31T23:59:59+09:00",
		)))
		expectEqual(t, result.Value("start_date"), "2026-01-01T00:00:00+09:00")
		expectEqual(t, result.Value("end_date"), "2026-01-31T23:59:59+09:00")
	})

	t.Run("includes locale_name when provided", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeGpsArgs(obj(
			"start_date", "2026-01-01",
			"end_date", "2026-01-31",
			"locale_name", "en",
		)))
		expectEqual(t, result.Value("locale_name"), "en")
	})

	t.Run("omits locale_name when not provided", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeGpsArgs(obj("start_date", "2026-01-01", "end_date", "2026-01-31")))
		expectTrue(t, !result.Has("locale_name"), "locale_name set")
	})

	t.Run("omits locale_name when undefined", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeGpsArgs(obj(
			"start_date", "2026-01-01",
			"end_date", "2026-01-31",
			"locale_name", jsonobj.Undefined,
		)))
		expectTrue(t, !result.Defined("locale_name"), "locale_name set")
	})

	t.Run("throws for missing start_date", func(t *testing.T) {
		mustFail(t)(NormalizeGpsArgs(obj("end_date", "2026-01-31")))
	})

	t.Run("throws for missing end_date", func(t *testing.T) {
		mustFail(t)(NormalizeGpsArgs(obj("start_date", "2026-01-01")))
	})

	t.Run("throws for null args (missing required dates)", func(t *testing.T) {
		mustFail(t)(NormalizeGpsArgs(nil))
	})

	t.Run("throws for unknown key", func(t *testing.T) {
		mustFail(t)(NormalizeGpsArgs(obj("start_date", "2026-01-01", "end_date", "2026-01-31", "unknown", "x")))
	})

	t.Run("end_date uses endOfDay for date-only input", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeGpsArgs(obj("start_date", "2026-06-15", "end_date", "2026-06-15")))
		mustMatch(t, strAt(t, result, "start_date"), `T00:00:00`)
		mustMatch(t, strAt(t, result, "end_date"), `T23:59:59`)
	})
}

// ---------------------------------------------------------------------------
// normalizeIdfFileArgs
// ---------------------------------------------------------------------------
func TestNormalizeIdfFileArgs(t *testing.T) {
	t.Run("accepts valid rep_name and file_name", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeIdfFileArgs(obj("rep_name", "my_repo", "file_name", "photo.jpg")))
		expectEqual(t, result.Value("rep_name"), "my_repo")
		expectEqual(t, result.Value("file_name"), "photo.jpg")
		expectTrue(t, !result.Defined("locale_name"), "locale_name set")
	})

	t.Run("includes locale_name when provided", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeIdfFileArgs(obj("rep_name", "repo", "file_name", "f.txt", "locale_name", "en")))
		expectEqual(t, result.Value("locale_name"), "en")
	})

	t.Run("throws when rep_name is missing", func(t *testing.T) {
		_, err := NormalizeIdfFileArgs(obj("file_name", "f.txt"))
		expectTrue(t, err != nil, "accepted")
	})

	t.Run("throws when file_name is missing", func(t *testing.T) {
		_, err := NormalizeIdfFileArgs(obj("rep_name", "repo"))
		expectTrue(t, err != nil, "accepted")
	})

	t.Run("throws when rep_name is empty", func(t *testing.T) {
		_, err := NormalizeIdfFileArgs(obj("rep_name", "  ", "file_name", "f.txt"))
		expectTrue(t, err != nil, "accepted")
	})

	t.Run("throws when file_name is empty", func(t *testing.T) {
		_, err := NormalizeIdfFileArgs(obj("rep_name", "repo", "file_name", ""))
		expectTrue(t, err != nil, "accepted")
	})

	t.Run("trims whitespace from strings", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeIdfFileArgs(obj("rep_name", "  repo  ", "file_name", "  file.txt  ")))
		expectEqual(t, result.Value("rep_name"), "repo")
		expectEqual(t, result.Value("file_name"), "file.txt")
	})

	t.Run("rejects unknown keys", func(t *testing.T) {
		_, err := NormalizeIdfFileArgs(obj("rep_name", "r", "file_name", "f", "unknown", 1))
		expectTrue(t, err != nil, "accepted")
	})

	t.Run("accepts null args as empty object and throws for missing required", func(t *testing.T) {
		_, err := NormalizeIdfFileArgs(nil)
		expectTrue(t, err != nil, "accepted")
	})

	t.Run("accepts thumb in the WxH form the file route expects", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeIdfFileArgs(obj("rep_name", "repo", "file_name", "photo.png", "thumb", "1024x1024")))
		expectEqual(t, result.Value("thumb"), "1024x1024")
	})

	t.Run("rejects thumb that is not WxH", func(t *testing.T) {
		for _, thumb := range []string{"1024", "1024*1024", "axb", "1024x", "12345x100"} {
			mustFail(t)(NormalizeIdfFileArgs(obj("rep_name", "r", "file_name", "f", "thumb", thumb)))
		}
	})

	t.Run("rejects thumb over the server cap instead of silently getting the original", func(t *testing.T) {
		// Go 側は 1024 超えを受け取るとサムネを作らず原本を返す。
		_, err := NormalizeIdfFileArgs(obj("rep_name", "r", "file_name", "f", "thumb", "2048x2048"))
		expectErrorMatches(t, err, `1024 per side`)
		_, err = NormalizeIdfFileArgs(obj("rep_name", "r", "file_name", "f", "thumb", "100x2048"))
		expectErrorMatches(t, err, `1024 per side`)
	})

	t.Run("accepts is_video together with thumb", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeIdfFileArgs(obj(
			"rep_name", "repo",
			"file_name", "clip.mp4",
			"thumb", "400x400",
			"is_video", true,
		)))
		expectEqual(t, result.Value("is_video"), true)
		expectEqual(t, result.Value("thumb"), "400x400")
	})

	t.Run("rejects is_video without thumb", func(t *testing.T) {
		_, err := NormalizeIdfFileArgs(obj("rep_name", "repo", "file_name", "clip.mp4", "is_video", true))
		expectErrorMatches(t, err, `requires thumb`)
	})

	t.Run("allows is_video:false on its own (it is a no-op, not a request for a frame)", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeIdfFileArgs(obj("rep_name", "repo", "file_name", "f.txt", "is_video", false)))
		expectEqual(t, result.Value("is_video"), false)
		expectTrue(t, !result.Defined("thumb"), "thumb set")
	})
}

// ---------------------------------------------------------------------------
// stale-schema revival (clients holding a pre-upgrade cached tool schema send
// post-upgrade top-level params as canonical-JSON strings; observed live on
// 2026-08-23 via the claude.ai connector)
// ---------------------------------------------------------------------------

func TestStaleSchemaArgumentRevival(t *testing.T) {
	t.Run("revives canonical JSON strings for v2 get_kyous params", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouArgs(obj(
			"count_only", "true",
			"data_types", `["nlog","kc"]`,
			"num_min", "3.5",
			"num_max", "10",
			"idf_kinds", `["image"]`,
			"include_file_size", "false",
		)))
		expectEqual(t, result.Value("count_only"), true)
		expectEqual(t, result.Value("data_types"), strs("nlog", "kc"))
		expectEqual(t, result.Value("num_min"), 3.5)
		expectEqual(t, result.Value("num_max"), 10)
		expectEqual(t, result.Value("idf_kinds"), strs("image"))
		expectEqual(t, result.Value("include_file_size"), false)
	})

	t.Run("properly typed values still pass unchanged", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouArgs(obj("count_only", true, "num_min", 2)))
		expectEqual(t, result.Value("count_only"), true)
		expectEqual(t, result.Value("num_min"), 2)
	})

	t.Run("non-canonical strings still fail type validation", func(t *testing.T) {
		mustFail(t)(NormalizeKyouArgs(obj("count_only", "TRUE")))
		mustFail(t)(NormalizeKyouArgs(obj("count_only", "yes")))
		mustFail(t)(NormalizeKyouArgs(obj("num_min", "abc")))
		mustFail(t)(NormalizeKyouArgs(obj("data_types", "nlog")))
		mustFail(t)(NormalizeKyouArgs(obj("data_types", `["nlog",1]`)))
		mustFail(t)(NormalizeKyouArgs(obj("data_types", "[broken")))
	})

	t.Run("revived idf_kinds still go through the allowed-value check", func(t *testing.T) {
		mustFail(t)(NormalizeKyouArgs(obj("idf_kinds", `["bogus"]`)))
	})

	t.Run("pre-v2 params are NOT revived (limit stays strict on get_kyous)", func(t *testing.T) {
		mustFail(t)(NormalizeKyouArgs(obj("limit", "50")))
	})

	t.Run("revives gps paging params (limit is v2-added on the gps tool)", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeGpsArgs(obj(
			"start_date", "2026-07-01",
			"end_date", "2026-07-02",
			"limit", "250",
			"count_only", "true",
		)))
		expectEqual(t, result.Value("limit"), 250)
		expectEqual(t, result.Value("count_only"), true)
	})

	t.Run("revived gps limit still rejects non-integers and out-of-range values", func(t *testing.T) {
		mustFail(t)(NormalizeGpsArgs(obj("start_date", "2026-07-01", "end_date", "2026-07-02", "limit", "2.5")))
		mustFail(t)(NormalizeGpsArgs(obj("start_date", "2026-07-01", "end_date", "2026-07-02", "limit", "999999")))
	})

	t.Run("revives app config projection params", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeAppConfigArgs(obj("fields", `["tag_struct"]`, "include_ui_state", "true")))
		expectEqual(t, result.Value("fields"), strs("tag_struct"))
		expectEqual(t, result.Value("include_ui_state"), true)
	})

	t.Run("revived app config fields still go through the allowed-value check", func(t *testing.T) {
		mustFail(t)(NormalizeAppConfigArgs(obj("fields", `["bogus_field"]`)))
	})
}

func TestNormalizeIdfFileArgsStaleSchemaRevival(t *testing.T) {
	// MCP のツール一覧はクライアントのセッション寿命で固定される。b303de73 で足した
	// is_video は boolean なので、旧スキーマのクライアントからは正規JSON文字列で届く。
	// reviveStaleSchemaArgs を通さないと、既存セッションからは動画サムネへ到達できない。
	t.Run(`revives is_video sent as the string "true" by a stale-schema client`, func(t *testing.T) {
		result := mustNormalize(t)(NormalizeIdfFileArgs(obj(
			"rep_name", "repo",
			"file_name", "clip.mp4",
			"thumb", "400x400",
			"is_video", "true",
		)))
		expectEqual(t, result.Value("is_video"), true)
	})

	t.Run(`revives is_video sent as the string "false"`, func(t *testing.T) {
		result := mustNormalize(t)(NormalizeIdfFileArgs(obj(
			"rep_name", "repo",
			"file_name", "photo.png",
			"thumb", "400x400",
			"is_video", "false",
		)))
		expectEqual(t, result.Value("is_video"), false)
	})

	t.Run("still rejects values that are not a canonical JSON boolean", func(t *testing.T) {
		for _, value := range []string{"TRUE", "yes", "1", ""} {
			mustFail(t)(NormalizeIdfFileArgs(obj("rep_name", "r", "file_name", "f", "thumb", "10x10", "is_video", value)))
		}
	})

	t.Run("a revived is_video still requires thumb", func(t *testing.T) {
		_, err := NormalizeIdfFileArgs(obj("rep_name", "r", "file_name", "f", "is_video", "true"))
		expectErrorMatches(t, err, `requires thumb`)
	})
}

func TestNormalizeGpsArgsInvertedPeriod(t *testing.T) {
	t.Run("rejects start_date after end_date instead of silently matching nothing", func(t *testing.T) {
		_, err := NormalizeGpsArgs(obj("start_date", "2026-08-24", "end_date", "2026-08-01"))
		expectErrorMatches(t, err, `must not be after end_date`)
	})

	t.Run("compares instants, not strings, so mixed offsets are judged correctly", func(t *testing.T) {
		// 文字列比較だと "2026-08-24T00:00:00Z" > "2026-08-24T08:00:00+09:00" に見えるが、
		// 実際には後者(= 2026-08-23T23:00:00Z)のほうが前。
		_, err := NormalizeGpsArgs(obj("start_date", "2026-08-24T00:00:00Z", "end_date", "2026-08-24T08:00:00+09:00"))
		expectErrorMatches(t, err, `must not be after end_date`)
	})

	t.Run("accepts an ordinary range and a single day", func(t *testing.T) {
		mustContain(t, strAt(t, mustNormalize(t)(NormalizeGpsArgs(obj("start_date", "2026-08-01", "end_date", "2026-08-24"))), "start_date"), "2026-08-01")
		_, err := NormalizeGpsArgs(obj("start_date", "2026-08-24", "end_date", "2026-08-24"))
		expectNoError(t, err)
	})
}

func TestCursorShape(t *testing.T) {
	// 壊れたカーソルは gkill 側で ERR000352「記録の取得に失敗しました」に畳まれ、
	// カーソルが原因だと分からなくなっていた。
	t.Run("rejects a cursor that is neither an RFC3339 time nor time::id", func(t *testing.T) {
		_, err := NormalizeKyouArgs(obj("cursor", "garbage-not-a-cursor"))
		expectErrorMatches(t, err, `next_cursor`)
		_, err = NormalizeGpsArgs(obj("start_date", "2026-08-01", "end_date", "2026-08-24", "cursor", "nonsense"))
		expectErrorMatches(t, err, `next_cursor`)
	})

	t.Run("accepts the v2 composite cursor and the legacy plain datetime", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("cursor", "2026-08-24T03:00:00+09:00::abc-123"))).Value("cursor"), "2026-08-24T03:00:00+09:00::abc-123")
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("cursor", "2026-08-24T03:00:00+09:00"))).Value("cursor"), "2026-08-24T03:00:00+09:00")
	})
}

func TestCreateAppsUpdateApps(t *testing.T) {
	// 「MCP で書いた記録」だけを絞る手段が無く、create_app は全レコードに入っているのに
	// 引く口だけが無かった。FindQuery には足していない（リポストのワード委譲が
	// 利用者クエリをそのまま下位検索へ流すので、SQL へ降ろすと黙って消える）
	t.Run("accepts create_apps and update_apps as string arrays", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouArgs(obj(
			"create_apps", strs("gkill_mcp_readwrite"),
			"update_apps", strs("gkill_kftl"),
		)))
		expectEqual(t, result.Value("create_apps"), strs("gkill_mcp_readwrite"))
		expectEqual(t, result.Value("update_apps"), strs("gkill_kftl"))
	})

	t.Run("keeps the null semantics: omitted is no filter, [] matches nothing", func(t *testing.T) {
		expectTrue(t, !mustNormalize(t)(NormalizeKyouArgs(obj())).Defined("create_apps"), "create_apps set")
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("create_apps", arr()))).Value("create_apps"), arr())
	})

	t.Run("revives them when a stale-schema client sends them as a JSON string", func(t *testing.T) {
		// 配列の新引数は revival の表に載せないと、旧スキーマのクライアントから
		// 文字列で届いて必ず型エラーになる
		result := mustNormalize(t)(NormalizeKyouArgs(obj("create_apps", `["gkill_mcp_write"]`)))
		expectEqual(t, result.Value("create_apps"), strs("gkill_mcp_write"))
	})

	t.Run("rejects a non-array", func(t *testing.T) {
		mustFail(t)(NormalizeKyouArgs(obj("create_apps", "gkill")))
	})
}

// ---------------------------------------------------------------------------
// GPS cursor validation
//
// The GPS cursor is a base64url token, NOT the Go composite
// `{RFC3339Nano}::{ID}` token that gkill_get_kyous uses. normalizeGpsArgs used to
// carry a copy-pasted RFC3339 check, so passing next_cursor back verbatim — exactly
// what the tool description instructs — failed 100% of the time.
// 発行側と受理側が別実装だったのが原因なので、実物の encode 出力で検証する。
// ---------------------------------------------------------------------------

func TestNormalizeGpsArgsCursor(t *testing.T) {
	period := func() *jsonobj.Object { return obj("start_date", "2026-08-23", "end_date", "2026-08-24") }

	t.Run("accepts a cursor produced by encodeGpsCursor", func(t *testing.T) {
		cursor := EncodeGpsCursor("2026-08-23T23:51:41.000+00:00", 1)
		result := mustNormalize(t)(NormalizeGpsArgs(period().Set("cursor", cursor)))
		expectEqual(t, result.Value("cursor"), cursor)
	})

	t.Run("rejects garbage and the get_kyous composite cursor", func(t *testing.T) {
		mustFail(t)(NormalizeGpsArgs(period().Set("cursor", "nonsense")))
		// Kyou 用の複合カーソルを GPS へ渡すのは呼び出し側の取り違えなので弾く
		mustFail(t)(NormalizeGpsArgs(period().Set("cursor", "2026-08-24T17:00:00+09:00::cd97a1f1")))
	})
}

// ---------------------------------------------------------------------------
// rep name filtering
// ---------------------------------------------------------------------------

func TestNormalizeRepNamesArgs(t *testing.T) {
	t.Run("defaults limit and accepts contains", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeRepNamesArgs(obj("contains", "Fit"))), obj("limit", 200, "contains", "Fit"))
	})

	t.Run("rejects unknown keys and out-of-range limit", func(t *testing.T) {
		mustFail(t)(NormalizeRepNamesArgs(obj("bogus", 1)))
		mustFail(t)(NormalizeRepNamesArgs(obj("limit", 0)))
		mustFail(t)(NormalizeRepNamesArgs(obj("limit", 2001)))
	})

	t.Run("revives a stale-schema limit string", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeRepNamesArgs(obj("limit", "50"))).Value("limit"), 50)
		mustFail(t)(NormalizeRepNamesArgs(obj("limit", "fifty")))
	})
}

// gkill_get_kyou_history の limit も他の後付け引数と同じ穴を持っていた
// （救済表に載っておらず、旧セッションからは文字列で届いて型エラーになる）
func TestNormalizeKyouHistoryArgsStaleSchemaRevival(t *testing.T) {
	t.Run("revives limit sent as a canonical JSON string", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouHistoryArgs(obj("id", "abc", "data_type", "kmemo", "limit", "5")))
		expectEqual(t, result.Value("limit"), 5)
	})

	t.Run("non-canonical strings still fail", func(t *testing.T) {
		mustFail(t)(NormalizeKyouHistoryArgs(obj("id", "abc", "data_type", "kmemo", "limit", "five")))
	})
}

// ---------------------------------------------------------------------------
// stale tool schema detection
//
// MCP tool schemas are frozen for the lifetime of a client session, so a
// server-side fix stays invisible until the client reconnects. Only emit the
// warning when staleness is PROVEN — a guess would make it constant noise.
// ---------------------------------------------------------------------------

func sortedCopy(items []string) []string {
	out := append([]string{}, items...)
	sort.Strings(out)
	return out
}

func TestDetectStaleSchemaSignals(t *testing.T) {
	t.Run("reports arguments that arrived as JSON strings", func(t *testing.T) {
		signals := DetectStaleSchemaSignals("gkill_get_kyous", obj("data_types", `["nlog"]`, "count_only", "true"))
		expectTrue(t, signals != nil, "no signals")
		expectEqual(t, sortedCopy(signals.Revived), []string{"count_only", "data_types"})
		expectEqual(t, signals.Deprecated, []string{})
	})

	t.Run("reports deprecated arguments, including legacy use_X query flags", func(t *testing.T) {
		signals := DetectStaleSchemaSignals("gkill_get_kyous", obj(
			"include_rep_name", false,
			"query", obj("use_reps", true, "only_latest_data", false, "words", strs("x")),
		))
		expectTrue(t, signals != nil, "no signals")
		expectEqual(t, signals.Revived, []string{})
		expectEqual(t, sortedCopy(signals.Deprecated), []string{"include_rep_name", "query.only_latest_data", "query.use_reps"})
	})

	t.Run("returns null for a current-schema call", func(t *testing.T) {
		expectTrue(t, DetectStaleSchemaSignals("gkill_get_kyous", obj("data_types", strs("nlog"), "query", obj("words", strs("x")))) == nil, "signals for a current call")
		expectTrue(t, DetectStaleSchemaSignals("gkill_get_gps_log", obj("start_date", "2026-08-01", "end_date", "2026-08-02")) == nil, "signals for a current gps call")
	})

	// string 型の引数は旧スキーマでも素通しするので、証拠にはならない
	t.Run("does not treat string-typed arguments as evidence", func(t *testing.T) {
		expectTrue(t, DetectStaleSchemaSignals("gkill_get_kyous", obj("group_by", "day", "cursor", "x")) == nil, "string args treated as evidence")
	})

	t.Run("covers every tool that has a revival table (table-driven, cannot fall behind)", func(t *testing.T) {
		// 以前はツールをハードコード列挙していて、表へ足した分だけこのテストが黙って
		// 古びていた (13 エントリ中 5 ツールしか見ていなかった)。表そのものを回し、
		// 全 (tool, key, kind) が「正規JSON文字列で届いたら古さの証拠として報告される」
		// ことを固定する。表へ1行足せば自動でここの検査対象になる。
		sampleByKind := map[string]string{
			"boolean":      "true",
			"number":       "5",
			"string_array": `["x"]`,
			"object_array": `[{"id":"a","data_type":"kmemo"}]`,
		}
		expectTrue(t, len(StaleSchemaArgKindsByTool) >= 13, "only %d tools in the revival table", len(StaleSchemaArgKindsByTool))
		for _, entry := range StaleSchemaArgKindsByTool {
			for _, kind := range entry.Kinds {
				sample, ok := sampleByKind[kind.Kind]
				expectTrue(t, ok, `no sample value for kind "%s" — add one to sampleByKind`, kind.Kind)
				signals := DetectStaleSchemaSignals(entry.Tool, obj(kind.Key, sample))
				revived := []string{}
				if signals != nil {
					revived = signals.Revived
				}
				expectTrue(t, NewStringSet(revived...).Has(kind.Key), "%s.%s (%s) not reported as revivable", entry.Tool, kind.Key, kind.Kind)
			}
		}
	})
}

// gkill_get_kyou_history も同じ2語彙の橋を通す。
// ツール説明が「gkill_get_kyous の結果から data_type を取れ」と案内しているのに、
// その結果は射影名なので、案内どおりにすると落ちていた。
func TestNormalizeKyouHistoryArgsProjectionDataType(t *testing.T) {
	t.Run("folds projection names to entity types", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouHistoryArgs(obj("id", "abc", "data_type", "mi_create"))).Value("data_type"), "mi")
		expectEqual(t, mustNormalize(t)(NormalizeKyouHistoryArgs(obj("id", "abc", "data_type", "timeis_start"))).Value("data_type"), "timeis")
		expectEqual(t, mustNormalize(t)(NormalizeKyouHistoryArgs(obj("id", "abc", "data_type", "mirekyou_end"))).Value("data_type"), "mirekyou")
	})

	t.Run("entity types are unchanged and unknown values still throw", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouHistoryArgs(obj("id", "abc", "data_type", "kmemo"))).Value("data_type"), "kmemo")
		mustFail(t)(NormalizeKyouHistoryArgs(obj("id", "abc", "data_type", "bogus")))
	})
}

// ---------------------------------------------------------------------------
// 逆さまのカレンダー期間
//
// gkill 側では0件になるだけで警告も出ず、「その期間に記録が無い」と読めてしまう。
// GPS 側 (NormalizeGpsArgs) は元から入口で弾いていたので、そちらと揃える
// （2026-08-25 の実利用レビュー）。
// ---------------------------------------------------------------------------

func TestNormalizeKyouQueryInvertedCalendarRange(t *testing.T) {
	t.Run("rejects start after end", func(t *testing.T) {
		mustFail(t)(NormalizeKyouQuery(obj("calendar_start_date", "2026-08-17", "calendar_end_date", "2026-08-11")))
	})

	t.Run("accepts the same day on both ends", func(t *testing.T) {
		// 日付のみは start=00:00:00 / end=23:59:59 へ展開されるので、同日は逆さまではない
		result := mustNormalize(t)(NormalizeKyouQuery(obj("calendar_start_date", "2026-08-11", "calendar_end_date", "2026-08-11")))
		mustMatch(t, strAt(t, result, "calendar_start_date"), `^2026-08-11T00:00:00`)
		mustMatch(t, strAt(t, result, "calendar_end_date"), `^2026-08-11T23:59:59`)
	})

	t.Run("accepts a normal range and one-sided ranges", func(t *testing.T) {
		_, err := NormalizeKyouQuery(obj("calendar_start_date", "2026-08-11", "calendar_end_date", "2026-08-17"))
		expectNoError(t, err)
		_, err = NormalizeKyouQuery(obj("calendar_start_date", "2026-08-17"))
		expectNoError(t, err)
		_, err = NormalizeKyouQuery(obj("calendar_end_date", "2026-08-11"))
		expectNoError(t, err)
	})
}

func TestNormalizeKyouArgsAggregationWithCursor(t *testing.T) {
	// 併用は gkill 側でも弾かれるが、返るのは ERR000352「記録の取得に失敗しました」という
	// 汎用文で、理由が本文に一切乗らない（実測 2026-08-25: 検索失敗と区別が付かなかった）。
	// GPS 側は前から MCP 層で理由つきに弾いており、get_kyous だけが素通しだった。
	t.Run("count_only + cursor を MCP 層で弾く", func(t *testing.T) {
		_, err := NormalizeKyouArgs(obj("count_only", true, "cursor", "2026-08-24T17:25:56+09:00::abc"))
		expectErrorMatches(t, err, `(?s)count_only.*cannot be combined with cursor`)
	})

	t.Run("group_by + cursor を MCP 層で弾く", func(t *testing.T) {
		_, err := NormalizeKyouArgs(obj("group_by", "day", "cursor", "2026-08-24T17:25:56+09:00::abc"))
		expectErrorMatches(t, err, `(?s)group_by.*cannot be combined with cursor`)
	})

	t.Run("cursor 単独・集計単独は通る", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("cursor", "2026-08-24T17:25:56+09:00::abc"))).Value("cursor"), "2026-08-24T17:25:56+09:00::abc")
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("count_only", true))).Value("count_only"), true)
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("group_by", "day"))).Value("group_by"), "day")
	})

	t.Run("count_only:false は cursor と併用できる", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouArgs(obj("count_only", false, "cursor", "2026-08-24T17:25:56+09:00::abc")))
		expectEqual(t, result.Value("count_only"), false)
		expectEqual(t, result.Value("cursor"), "2026-08-24T17:25:56+09:00::abc")
	})

	// count_only + group_by は、以前は count_only の早期 return が group_by を黙って捨て、
	// buckets の無い応答が「集計できた」顔で返っていた（2026-09-18 の実利用報告）。
	t.Run("count_only + group_by を MCP 層で弾く（cursor 併用と同じ扱い）", func(t *testing.T) {
		_, err := NormalizeKyouArgs(obj("count_only", true, "group_by", "data_type"))
		expectErrorMatches(t, err, `(?s)count_only.*cannot be combined with group_by`)
		// count_only:false は「集計しない」なので group_by と併用できる
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("count_only", false, "group_by", "data_type"))).Value("group_by"), "data_type")
	})

	t.Run("count_only + group_by も GPS と同じ文言で弾く", func(t *testing.T) {
		_, gpsErr := PaginateGpsLogs(arr(), obj("count_only", true, "group_by", "day", "limit", 1))
		_, kyouErr := NormalizeKyouArgs(obj("count_only", true, "group_by", "day"))
		expectTrue(t, gpsErr != nil, "gps accepted")
		expectTrue(t, kyouErr != nil, "kyou accepted")
		expectEqual(t, gpsErr.Error(), kyouErr.Error())
	})

	t.Run("GPS と get_kyous が同じ規則・同じ文言で弾く", func(t *testing.T) {
		_, gpsErr := PaginateGpsLogs(arr(), obj("count_only", true, "cursor", EncodeGpsCursor("2026-08-24T00:00:00+09:00", 0)))
		_, kyouErr := NormalizeKyouArgs(obj("count_only", true, "cursor", "2026-08-24T17:25:56+09:00::abc"))
		expectTrue(t, gpsErr != nil, "gps accepted")
		expectTrue(t, kyouErr != nil, "kyou accepted")
		expectEqual(t, gpsErr.Error(), kyouErr.Error())
	})
}

func TestNormalizeRepInfosArgsDataKinds(t *testing.T) {
	// attached_data_reps は歴代端末ぶんの Tag_ / Text_ / Notification_ / GPSLogs_ が並ぶので
	// 本番で約120件。fields は「配列を返すか返さないか」しか選べなかった。
	t.Run("正しい data_kind を受理する", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeRepInfosArgs(obj("data_kinds", strs("tag", "gpslog")))).Value("data_kinds"), strs("tag", "gpslog"))
	})

	// 語彙が4つしかないので、黙って0件になると「その種別が無い」と読めてしまう。
	t.Run("綴り違いは0件ではなくエラー", func(t *testing.T) {
		_, err := NormalizeRepInfosArgs(obj("data_kinds", strs("tags")))
		expectErrorMatches(t, err, `must be one of`)
		_, err = NormalizeRepInfosArgs(obj("data_kinds", strs("Tag")))
		expectErrorMatches(t, err, `must be one of`)
	})

	t.Run("省略時は絞り込まない", func(t *testing.T) {
		expectTrue(t, !mustNormalize(t)(NormalizeRepInfosArgs(obj())).Defined("data_kinds"), "data_kinds set")
	})
}

// 行の絞り込み（2026-09-18 の実利用報告: fields で rep_infos[] を落としても Archived Git の rep 名・
// 歴代端末の GPSLogs_ / Tag_ / Text_ だけで数百行残り、「gkill_add_tag はどこへ書くか」に全部を読んでいた）。
func TestNormalizeRepInfosArgsRowFiltering(t *testing.T) {
	t.Run("writable_only / rep_types / rep_names / contains を受理する", func(t *testing.T) {
		normalized := mustNormalize(t)(NormalizeRepInfosArgs(obj(
			"writable_only", true,
			"rep_types", strs("kmemo", "directory"),
			"rep_names", strs("Kmemo_pc"),
			"contains", "tag_",
		)))
		expectEqual(t, normalized, obj(
			"writable_only", true,
			"rep_types", strs("kmemo", "directory"),
			"rep_names", strs("Kmemo_pc"),
			"contains", "tag_",
		))
	})

	t.Run("型違いは弾く", func(t *testing.T) {
		mustFail(t)(NormalizeRepInfosArgs(obj("writable_only", "yes")))
		mustFail(t)(NormalizeRepInfosArgs(obj("rep_types", "kmemo")))
		mustFail(t)(NormalizeRepInfosArgs(obj("rep_names", arr(1))))
		mustFail(t)(NormalizeRepInfosArgs(obj("contains", "")))
	})

	// 古い一覧を握るクライアントからは配列 / boolean が正規JSON文字列で届く。
	// data_kinds は 2026-08-25 の追加時に救済表へ載せ忘れていた。
	t.Run("古スキーマ経由の文字列を復元する（data_kinds の載せ忘れも含む）", func(t *testing.T) {
		normalized := mustNormalize(t)(NormalizeRepInfosArgs(obj(
			"data_kinds", `["tag"]`,
			"writable_only", "true",
			"rep_types", `["kmemo"]`,
			"rep_names", `["Kmemo_pc"]`,
		)))
		expectEqual(t, normalized.Value("data_kinds"), strs("tag"))
		expectEqual(t, normalized.Value("writable_only"), true)
		expectEqual(t, normalized.Value("rep_types"), strs("kmemo"))
		expectEqual(t, normalized.Value("rep_names"), strs("Kmemo_pc"))
	})
}

func TestNormalizeKftlArgsIdempotencyKey(t *testing.T) {
	// KFTL は DB トランザクションではないので、失敗して再送するたびに
	// 手前で書けたぶんが積む（実測で孤児 kmemo 4件）。受け口はサーバに前からあり、
	// Wear OS は送っていたのに MCP だけ送っていなかった。
	t.Run("任意で受け取り、そのまま渡す", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKftlArgs(obj("kftl_text", "メモ", "idempotency_key", "k-1"))).Value("idempotency_key"), "k-1")
	})

	t.Run("省略できる", func(t *testing.T) {
		expectTrue(t, !mustNormalize(t)(NormalizeKftlArgs(obj("kftl_text", "メモ"))).Defined("idempotency_key"), "idempotency_key set")
	})
}

// ---------------------------------------------------------------------------
// 未知の引数名（2026-09-14 レビュー P0）
//
// ChatGPT が改名前の検索条件名を、握ったままの古い一覧どおりに送って
// 「is not supported」だけを受け取った。タイプミスと古い一覧はサーバから区別できないので、
// 両方の可能性と再接続・gkill_status の照合を案内する。文言はトップレベルと query で同じ正本。
// ---------------------------------------------------------------------------
func TestUnknownArgumentNamesPointAtAPossiblyStaleToolList(t *testing.T) {
	const expected = `(?s)is not supported.*misspelled.*tool list.*stale.*reconnect the MCP client.*gkill_status.*schema_revision`

	t.Run("query-level unknown key", func(t *testing.T) {
		caught := mustFail(t)(NormalizeKyouQuery(obj("no_such_field", "2026-09-14")))
		mustMatch(t, caught.Message, `^Invalid argument 'query\.no_such_field': `)
		mustMatch(t, caught.Message, expected)
		expectEqual(t, caught.Detail.Value("field"), "query.no_such_field")
		expectTrue(t, containsValue(arrAt(t, caught.Detail, "allowed"), "playing_time"), "allowed lacks playing_time")
	})

	t.Run("top-level unknown key uses the same wording", func(t *testing.T) {
		caught := mustFail(t)(NormalizeKyouArgs(obj("no_such_arg", true)))
		mustMatch(t, caught.Message, `^Invalid argument 'arguments\.no_such_arg': `)
		mustMatch(t, caught.Message, expected)
	})

	t.Run("the stale-schema warning also names gkill_status as the way to compare revisions", func(t *testing.T) {
		text := StaleSchemaWarning(&StaleSchemaSignals{Revived: []string{"count_only"}, Deprecated: []string{}})
		mustMatch(t, text, `(?s)gkill_status.*schema_revision`)
	})
}

// ---------------------------------------------------------------------------
// 廃止済み引数は公開スキーマに載せず、受理と古さの検出だけ残す（ADR-0620）
// ---------------------------------------------------------------------------
func TestDeprecatedArgumentsAreAcceptedAtRuntimeButAbsentFromTheAdvertisedSchema(t *testing.T) {
	kyousTool := findTool(ReadTools, "gkill_get_kyous")

	t.Run("include_id / include_rep_name / only_latest_data are not advertised", func(t *testing.T) {
		properties := objAt(t, kyousTool, "inputSchema", "properties")
		expectTrue(t, !properties.Has("include_id"), "include_id advertised")
		expectTrue(t, !properties.Has("include_rep_name"), "include_rep_name advertised")
		queryProperties := objAt(t, FindQuerySchema, "properties")
		expectTrue(t, !queryProperties.Has("only_latest_data"), "only_latest_data advertised")
		for _, key := range queryProperties.Keys() {
			expectTrue(t, !strings.HasPrefix(key, "use_"), "legacy flag %s must not be advertised", key)
		}
		// 説明文も旧 use_X フラグを説明しない（「受理する」と書けば AI は使う）
		mustNotMatch(t, strAt(t, FindQuerySchema, "description"), `use_X`)
	})

	t.Run("they are still accepted, and their arrival is reported as stale-schema evidence", func(t *testing.T) {
		normalized := mustNormalize(t)(NormalizeKyouArgs(obj(
			"include_id", true,
			"include_rep_name", false,
			"query", obj("only_latest_data", false, "use_words", true, "words", strs("x")),
		)))
		expectEqual(t, normalized.Value("query"), obj("words", strs("x"), "only_latest_data", true))
		payload := AppendStaleSchemaWarning(obj("kyous", arr()), "gkill_get_kyous", obj(
			"include_id", true,
			"include_rep_name", false,
			"query", obj("only_latest_data", false, "use_words", true, "words", strs("x")),
		))
		warnings := arrAt(t, payload, "warnings")
		expectTrue(t, len(warnings) == 1, "%d warnings", len(warnings))
		mustMatch(t, jsString(warnings[0]), `deprecated arguments were sent: include_id, include_rep_name, query\.only_latest_data, query\.use_words`)
	})
}

// ---------------------------------------------------------------------------
// normalizeStatusArgs — 引数を取らない
// ---------------------------------------------------------------------------
func TestNormalizeStatusArgs(t *testing.T) {
	t.Run("accepts undefined / null / {}", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeStatusArgs(jsonobj.Undefined)), obj())
		expectEqual(t, mustNormalize(t)(NormalizeStatusArgs(nil)), obj())
		expectEqual(t, mustNormalize(t)(NormalizeStatusArgs(obj())), obj())
	})

	t.Run("rejects any argument (so the tool never lands in the stale-schema revival table)", func(t *testing.T) {
		_, err := NormalizeStatusArgs(obj("locale_name", "ja"))
		expectErrorMatches(t, err, `(?s)arguments\.locale_name.*is not supported`)
		_, ok := staleSchemaArgKindsFor("gkill_status")
		expectTrue(t, !ok, "gkill_status is in the revival table")
	})
}

// ---------------------------------------------------------------------------
// 2026-09-19 の MCP 実利用報告への対応（ADR-0626 / 0627 / 0629 / 0630）
// ---------------------------------------------------------------------------
func TestForMiDefaultProjectionOptInFlagsAndHistoryOffset(t *testing.T) {
	t.Run("normalizeKyouArgs assumes include_create_mi for a bare for_mi and records a note", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKyouArgs(obj("query", obj("for_mi", true))))
		expectEqual(t, objAt(t, result, "query").Value("include_create_mi"), true)
		notes := arrAt(t, result, "notes")
		expectTrue(t, len(notes) == 1, "%d notes", len(notes))
		mustContain(t, jsString(notes[0]), "include_create_mi:true was assumed")
		expectTrue(t, !mustNormalize(t)(NormalizeKyouArgs(obj("query", obj("for_mi", true, "include_check_mi", true)))).Defined("notes"), "notes set")
		expectTrue(t, !objAt(t, mustNormalize(t)(NormalizeKyouArgs(obj("query", obj("for_mi", true, "include_check_mi", true)))), "query").Defined("include_create_mi"), "include_create_mi assumed")
		expectTrue(t, !mustNormalize(t)(NormalizeKyouArgs(obj("query", obj()))).Defined("notes"), "notes set for an empty query")
	})

	t.Run("normalizeKyouArgs defaults include_attached_ids / include_file_urls to false and revives stale strings", func(t *testing.T) {
		defaults := mustNormalize(t)(NormalizeKyouArgs(obj()))
		expectEqual(t, defaults.Value("include_attached_ids"), false)
		expectEqual(t, defaults.Value("include_file_urls"), false)
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("include_attached_ids", true, "include_file_urls", true))).Value("include_file_urls"), true)
		mustFail(t)(NormalizeKyouArgs(obj("include_file_urls", "yes")))
		// 古い一覧を握るクライアントからは正規JSON文字列で届く
		expectEqual(t, mustNormalize(t)(NormalizeKyouArgs(obj("include_attached_ids", "true"))).Value("include_attached_ids"), true)
	})

	t.Run("normalizeKyouHistoryArgs accepts offset (also as a stale-schema string) and rejects a negative one", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeKyouHistoryArgs(obj("id", "a", "data_type", "kmemo"))).Value("offset"), 0)
		expectEqual(t, mustNormalize(t)(NormalizeKyouHistoryArgs(obj("id", "a", "data_type", "kmemo", "offset", 40))).Value("offset"), 40)
		expectEqual(t, mustNormalize(t)(NormalizeKyouHistoryArgs(obj("id", "a", "data_type", "kmemo", "offset", "40"))).Value("offset"), 40)
		mustFail(t)(NormalizeKyouHistoryArgs(obj("id", "a", "data_type", "kmemo", "offset", -1)))
	})

	t.Run("top-level unknown keys hide deprecated names from allowed", func(t *testing.T) {
		thrown := mustFail(t)(NormalizeKyouArgs(obj("limitt", 3)))
		allowed := arrAt(t, thrown.Detail, "allowed")
		expectTrue(t, !containsValue(allowed, "include_id"), "allowed contains include_id")
		expectTrue(t, containsValue(allowed, "include_attached_ids"), "allowed lacks include_attached_ids")
	})
}
