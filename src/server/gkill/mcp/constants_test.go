package mcp

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// ---------------------------------------------------------------------------
// DEFAULT_KYOUS_LIMIT
// ---------------------------------------------------------------------------
func TestDefaultKyousLimit(t *testing.T) {
	t.Run("equals 20", func(t *testing.T) {
		expectEqual(t, DefaultKyousLimit, 20)
	})
}

// ---------------------------------------------------------------------------
// DEFAULT_KYOUS_MAX_SIZE_MB
// ---------------------------------------------------------------------------
func TestDefaultKyousMaxSizeMB(t *testing.T) {
	t.Run("equals 0.25", func(t *testing.T) {
		expectEqual(t, DefaultKyousMaxSizeMB, 0.25)
	})
}

// ---------------------------------------------------------------------------
// DEFAULT_KYOUS_INCLUDE_TIMEIS
// ---------------------------------------------------------------------------
func TestDefaultKyousIncludeTimeIs(t *testing.T) {
	t.Run("equals false", func(t *testing.T) {
		expectEqual(t, DefaultKyousIncludeTimeIs, false)
	})
}

// ---------------------------------------------------------------------------
// RFC3339_REGEX
// ---------------------------------------------------------------------------
func TestRFC3339Regex(t *testing.T) {
	t.Run("is a RegExp", func(t *testing.T) {
		expectTrue(t, RFC3339Regex != nil, "nil regexp")
	})

	t.Run("matches valid RFC 3339 datetime with timezone offset", func(t *testing.T) {
		expectTrue(t, RFC3339Regex.MatchString("2026-02-25T10:30:00+09:00"), "no match")
	})

	t.Run("matches valid RFC 3339 datetime with Z", func(t *testing.T) {
		expectTrue(t, RFC3339Regex.MatchString("2026-02-25T10:30:00Z"), "no match")
	})

	t.Run("matches datetime with fractional seconds", func(t *testing.T) {
		expectTrue(t, RFC3339Regex.MatchString("2026-02-25T10:30:00.123+09:00"), "no match")
	})

	t.Run("matches datetime with negative offset", func(t *testing.T) {
		expectTrue(t, RFC3339Regex.MatchString("2026-02-25T10:30:00-05:00"), "no match")
	})

	t.Run("does not match date-only string", func(t *testing.T) {
		expectTrue(t, !RFC3339Regex.MatchString("2026-02-25"), "matched")
	})

	t.Run("does not match empty string", func(t *testing.T) {
		expectTrue(t, !RFC3339Regex.MatchString(""), "matched")
	})

	t.Run("does not match plain text", func(t *testing.T) {
		expectTrue(t, !RFC3339Regex.MatchString("not-a-date"), "matched")
	})
}

// ---------------------------------------------------------------------------
// DATE_ONLY_REGEX
// ---------------------------------------------------------------------------
func TestDateOnlyRegex(t *testing.T) {
	t.Run("is a RegExp", func(t *testing.T) {
		expectTrue(t, DateOnlyRegex != nil, "nil regexp")
	})

	t.Run("matches YYYY-MM-DD", func(t *testing.T) {
		expectTrue(t, DateOnlyRegex.MatchString("2026-02-25"), "no match")
	})

	t.Run("does not match full datetime", func(t *testing.T) {
		expectTrue(t, !DateOnlyRegex.MatchString("2026-02-25T10:30:00+09:00"), "matched")
	})

	t.Run("does not match empty string", func(t *testing.T) {
		expectTrue(t, !DateOnlyRegex.MatchString(""), "matched")
	})

	t.Run("does not match partial date", func(t *testing.T) {
		expectTrue(t, !DateOnlyRegex.MatchString("2026-02"), "matched")
	})
}

// ---------------------------------------------------------------------------
// KYOUS_TOP_LEVEL_FIELDS
// ---------------------------------------------------------------------------
func TestKyousTopLevelFields(t *testing.T) {
	t.Run("is a non-empty Set", func(t *testing.T) {
		expectTrue(t, KyousTopLevelFields.Len() > 0, "empty")
	})

	t.Run("contains expected fields", func(t *testing.T) {
		for _, key := range []string{"query", "locale_name", "limit", "cursor", "max_size_mb", "is_include_timeis"} {
			expectTrue(t, KyousTopLevelFields.Has(key), "missing %s", key)
		}
	})

	t.Run("contains the inline plugin content fields", func(t *testing.T) {
		for _, key := range []string{"include_plugin_content", "plugin_content_max_text_length", "plugin_content_format"} {
			expectTrue(t, KyousTopLevelFields.Has(key), "missing %s", key)
		}
	})
}

// ---------------------------------------------------------------------------
// インライン本文取得の定数
// ---------------------------------------------------------------------------
func TestInlinePluginContentConstants(t *testing.T) {
	t.Run("defaults to opt-in", func(t *testing.T) {
		expectEqual(t, DefaultIncludePluginContent, false)
	})

	t.Run("keeps the format enum and its default", func(t *testing.T) {
		expectEqual(t, PluginContentFormats.Values(), []string{"text", "html", "both"})
		expectTrue(t, PluginContentFormats.Has(DefaultPluginContentFormat), "default is not a format")
		expectEqual(t, DefaultPluginContentFormat, "text")
	})

	t.Run("orders the per-entry default below the per-entry maximum", func(t *testing.T) {
		expectTrue(t, DefaultInlinePluginContentMaxTextLength > 0, "default not positive")
		expectTrue(t, DefaultInlinePluginContentMaxTextLength <= MaxPluginContentMaxTextLength, "default above max")
	})

	t.Run("lets a single entry at the maximum length fit the total budget", func(t *testing.T) {
		expectTrue(t, InlinePluginContentTotalTextLength >= MaxPluginContentMaxTextLength, "total below per-entry max")
	})

	t.Run("uses sane counts, concurrency and deadline", func(t *testing.T) {
		expectTrue(t, MaxInlinePluginContentKyous >= 1, "kyous < 1")
		expectTrue(t, InlinePluginContentRepConcurrency >= 1, "concurrency < 1")
		expectTrue(t, InlinePluginContentDeadlineMS > 0, "deadline <= 0")
		expectTrue(t, MaxInlinePluginContentHTMLLength > MaxPluginContentMaxTextLength, "html max <= text max")
	})
}

// ---------------------------------------------------------------------------
// KYOUS_QUERY_BOOLEAN_FIELDS
// ---------------------------------------------------------------------------
func TestKyousQueryBooleanFields(t *testing.T) {
	t.Run("is a non-empty Set", func(t *testing.T) {
		expectTrue(t, KyousQueryBooleanFields.Len() > 0, "empty")
	})

	t.Run("contains expected boolean fields", func(t *testing.T) {
		for _, key := range []string{"update_cache", "include_deleted_data", "is_image_only", "only_latest_data"} {
			expectTrue(t, KyousQueryBooleanFields.Has(key), "missing %s", key)
		}
	})

	// 実装されたことが一度も無く、送っているクライアントも実在しなかった検索条件。
	// 受理して黙って無視するより、未知キーとして弾いて「そのフィルタは無い」と伝えるほうがよい。
	t.Run("no longer contains the never-implemented is_deleted", func(t *testing.T) {
		expectTrue(t, !KyousQueryBooleanFields.Has("is_deleted"), "is_deleted is present")
	})

	t.Run("no longer contains the removed use_X flags", func(t *testing.T) {
		for _, key := range []string{"use_tags", "use_words", "use_calendar", "use_include_id"} {
			expectTrue(t, !KyousQueryBooleanFields.Has(key), "%s is present", key)
		}
	})
}

// ---------------------------------------------------------------------------
// LEGACY_USE_FLAG_KEYS
// ---------------------------------------------------------------------------
func TestLegacyUseFlagKeys(t *testing.T) {
	// Go (find/find_query_legacy_json.go) と client
	// (classes/api/find_query/normalize-legacy-find-kyou-query-json.ts) と同じ16キー。
	// ここが欠けると、その旧フラグを送ってきたクライアントが未知キー扱いで弾かれる
	t.Run("contains exactly the 16 removed use_X flags", func(t *testing.T) {
		want := []string{
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
			"use_playing",
			"use_update_time",
			"use_mi_board_name",
			"use_period_of_time",
			"use_mi_sort_type",
			"use_mi_check_state",
		}
		sort.Strings(want)
		expectEqual(t, LegacyUseFlagKeys.Sorted(), want)
	})

	t.Run("does not overlap KYOUS_QUERY_BOOLEAN_FIELDS", func(t *testing.T) {
		for _, key := range LegacyUseFlagKeys.Values() {
			expectTrue(t, !KyousQueryBooleanFields.Has(key), "%s overlaps", key)
		}
	})

	t.Run("derived KYOUS_QUERY_ALL_FIELDS follows the removal (no use_X keys advertised)", func(t *testing.T) {
		for _, key := range LegacyUseFlagKeys.Values() {
			expectTrue(t, !KyousQueryAllFields.Has(key), "%s advertised", key)
		}
	})
}

// ---------------------------------------------------------------------------
// KYOUS_QUERY_STRING_ARRAY_FIELDS
// ---------------------------------------------------------------------------
func TestKyousQueryStringArrayFields(t *testing.T) {
	t.Run("is a non-empty Set", func(t *testing.T) {
		expectTrue(t, KyousQueryStringArrayFields.Len() > 0, "empty")
	})

	t.Run("contains expected string array fields", func(t *testing.T) {
		for _, key := range []string{"rep_types", "ids", "words", "tags", "reps"} {
			expectTrue(t, KyousQueryStringArrayFields.Has(key), "missing %s", key)
		}
	})
}

// ---------------------------------------------------------------------------
// KYOUS_QUERY_NUMBER_FIELDS
// ---------------------------------------------------------------------------
func TestKyousQueryNumberFields(t *testing.T) {
	t.Run("is a non-empty Set", func(t *testing.T) {
		expectTrue(t, KyousQueryNumberFields.Len() > 0, "empty")
	})

	t.Run("contains map-related number fields", func(t *testing.T) {
		for _, key := range []string{"map_radius", "map_latitude", "map_longitude"} {
			expectTrue(t, KyousQueryNumberFields.Has(key), "missing %s", key)
		}
	})
}

// ---------------------------------------------------------------------------
// KYOUS_QUERY_INTEGER_FIELDS
// ---------------------------------------------------------------------------
func TestKyousQueryIntegerFields(t *testing.T) {
	t.Run("is a non-empty Map", func(t *testing.T) {
		expectTrue(t, len(KyousQueryIntegerFields) > 0, "empty")
	})

	t.Run("contains period_of_time fields with min/max", func(t *testing.T) {
		startTime, ok := kyousQueryIntegerField("period_of_time_start_time_second")
		expectTrue(t, ok, "start field missing")
		expectEqual(t, startTime.Min, 0)
		expectEqual(t, startTime.Max, 86399)

		endTime, ok := kyousQueryIntegerField("period_of_time_end_time_second")
		expectTrue(t, ok, "end field missing")
		expectEqual(t, endTime.Min, 0)
		expectEqual(t, endTime.Max, 86399)
	})
}

// ---------------------------------------------------------------------------
// KYOUS_QUERY_DATETIME_FIELDS
// ---------------------------------------------------------------------------
func TestKyousQueryDateTimeFields(t *testing.T) {
	t.Run("is a non-empty Map", func(t *testing.T) {
		expectTrue(t, len(KyousQueryDateTimeFields) > 0, "empty")
	})

	t.Run("contains calendar and playing datetime fields", func(t *testing.T) {
		for _, key := range []string{"calendar_start_date", "calendar_end_date", "playing_time", "update_time"} {
			_, ok := kyousQueryDateTimeField(key)
			expectTrue(t, ok, "missing %s", key)
		}
	})

	t.Run("calendar_end_date has endOfDay=true", func(t *testing.T) {
		endDate, _ := kyousQueryDateTimeField("calendar_end_date")
		expectTrue(t, endDate.EndOfDay, "endOfDay is false")
		expectTrue(t, endDate.AllowDateOnly, "allowDateOnly is false")
	})

	t.Run("calendar_start_date has endOfDay=false", func(t *testing.T) {
		startDate, _ := kyousQueryDateTimeField("calendar_start_date")
		expectTrue(t, !startDate.EndOfDay, "endOfDay is true")
		expectTrue(t, startDate.AllowDateOnly, "allowDateOnly is false")
	})
}

// ---------------------------------------------------------------------------
// MI_CHECK_STATES
// ---------------------------------------------------------------------------
func TestMiCheckStates(t *testing.T) {
	t.Run("is a non-empty Set", func(t *testing.T) {
		expectTrue(t, MiCheckStates.Len() > 0, "empty")
	})

	t.Run("contains all, checked, uncheck", func(t *testing.T) {
		for _, key := range []string{"all", "checked", "uncheck"} {
			expectTrue(t, MiCheckStates.Has(key), "missing %s", key)
		}
	})

	t.Run("has exactly 3 entries", func(t *testing.T) {
		expectEqual(t, MiCheckStates.Len(), 3)
	})
}

// ---------------------------------------------------------------------------
// MI_SORT_TYPES
// ---------------------------------------------------------------------------
func TestMiSortTypes(t *testing.T) {
	t.Run("is a non-empty Set", func(t *testing.T) {
		expectTrue(t, MiSortTypes.Len() > 0, "empty")
	})

	t.Run("contains expected sort types", func(t *testing.T) {
		for _, key := range []string{"create_time", "estimate_start_time", "estimate_end_time", "limit_time"} {
			expectTrue(t, MiSortTypes.Has(key), "missing %s", key)
		}
	})

	t.Run("has exactly 4 entries", func(t *testing.T) {
		expectEqual(t, MiSortTypes.Len(), 4)
	})
}

// ---------------------------------------------------------------------------
// Go 側の列挙値との一致
//
// group_by の有効値は Go(get_kyous_mcp_helpers.go の mcpGroupByValues)、
// MCP の正規化(KyousGroupByValues)、スキーマの enum(read_tools.go) の3箇所にある。
// 「サーバ側と揃えること」とコメントされているだけで機械検査が無く、
// MCP 側に足し忘れると **Go に届く前に MCP が弾く**、Go 側に足し忘れると
// 「スキーマは受理するのにサーバがエラーを返す」になる。
// ここで実ファイルを読んで突き合わせ、ズレたら落とす。
// ---------------------------------------------------------------------------

func readSourceFile(t testing.TB, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(relPath)
	if err != nil {
		t.Fatalf("read %s: %v", relPath, err)
	}
	return string(data)
}

func TestGroupByVocabularyIsIdenticalAcrossGoNormalizationAndSchema(t *testing.T) {
	goSource := readSourceFile(t, "../api/gkill_server_api/get_kyous_mcp_helpers.go")

	goGroupByValues := func() []string {
		match := regexp.MustCompile(`var mcpGroupByValues = \[\]string\{([^}]*)\}`).FindStringSubmatch(goSource)
		if match == nil {
			t.Fatalf("mcpGroupByValues が Go 側に見つからない（変数名が変わった？）")
		}
		values := []string{}
		for _, part := range strings.Split(match[1], ",") {
			part = strings.Trim(strings.TrimSpace(part), `"`)
			if part != "" {
				values = append(values, part)
			}
		}
		sort.Strings(values)
		return values
	}

	t.Run("Go and KYOUS_GROUP_BY_VALUES agree", func(t *testing.T) {
		expectEqual(t, KyousGroupByValues.Sorted(), goGroupByValues())
	})

	t.Run("the tool schema enum agrees too", func(t *testing.T) {
		tool := findTool(ReadTools, "gkill_get_kyous")
		enum := arrAt(t, tool, "inputSchema", "properties", "group_by", "enum")
		values := []string{}
		for _, v := range enum {
			values = append(values, v.(string))
		}
		sort.Strings(values)
		expectEqual(t, values, goGroupByValues())
	})
}

// undefinedValue は JS の undefined（存在しない値）。
func undefinedValue() any { return jsonobj.Undefined }
