package find

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func decodeMigratedObject(t *testing.T, migrated []byte) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(migrated))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("migrated JSON decode failed: %v", err)
	}
	obj, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("migrated JSON is not an object: %T", value)
	}
	return obj
}

func assertKeyAbsent(t *testing.T, obj map[string]any, key string) {
	t.Helper()
	if _, exist := obj[key]; exist {
		t.Errorf("key %q should be removed but exists (value=%v)", key, obj[key])
	}
}

func assertNull(t *testing.T, obj map[string]any, key string) {
	t.Helper()
	value, exist := obj[key]
	if !exist {
		t.Errorf("key %q should exist as null but is absent", key)
		return
	}
	if value != nil {
		t.Errorf("key %q should be null but is %v", key, value)
	}
}

func assertEmptyArray(t *testing.T, obj map[string]any, key string) {
	t.Helper()
	value, exist := obj[key]
	if !exist {
		t.Errorf("key %q should exist as [] but is absent", key)
		return
	}
	arr, ok := value.([]any)
	if !ok {
		t.Errorf("key %q should be [] but is %T(%v)", key, value, value)
		return
	}
	if len(arr) != 0 {
		t.Errorf("key %q should be empty array but has %d elements", key, len(arr))
	}
}

func assertStringArray(t *testing.T, obj map[string]any, key string, want []string) {
	t.Helper()
	value, exist := obj[key]
	if !exist {
		t.Fatalf("key %q is absent", key)
	}
	arr, ok := value.([]any)
	if !ok {
		t.Fatalf("key %q is not an array: %T", key, value)
	}
	if len(arr) != len(want) {
		t.Fatalf("key %q length: got %d, want %d", key, len(arr), len(want))
	}
	for i := range want {
		if arr[i] != want[i] {
			t.Errorf("key %q[%d]: got %v, want %q", key, i, arr[i], want[i])
		}
	}
}

// use_X=false のグループは値が null 化され、use_* キーが消えること
func TestMigrateLegacyFindQueryJSONDisabledGroups(t *testing.T) {
	legacy := `{
		"query_id": "q1",
		"use_words": false, "words": ["w1"], "not_words": ["n1"], "words_and": true,
		"use_tags": false, "tags": ["t1", "t2"], "hide_tags": ["h1"], "tags_and": false,
		"use_reps": false, "reps": ["r1"],
		"use_rep_types": false, "rep_types": ["kmemo"],
		"use_ids": false, "ids": ["id1"],
		"use_timeis": false, "timeis_words": ["tw"], "timeis_not_words": ["tnw"],
		"use_timeis_tags": true, "timeis_tags": ["tt1"],
		"use_calendar": false, "calendar_start_date": "2020-01-01T00:00:00+09:00", "calendar_end_date": "2020-01-02T00:00:00+09:00",
		"use_map": false, "map_radius": 500, "map_latitude": 35.6, "map_longitude": 139.7,
		"use_plaing": false, "plaing_time": "2020-01-01T00:00:00+09:00",
		"use_update_time": false, "update_time": "2020-01-01T00:00:00+09:00",
		"use_mi_board_name": false, "mi_board_name": "board",
		"use_period_of_time": false, "period_of_time_start_time_second": 3600, "period_of_time_end_time_second": 7200, "period_of_time_week_of_days": [0, 1, 2]
	}`

	migrated, changed, err := MigrateLegacyFindQueryJSON([]byte(legacy))
	if err != nil {
		t.Fatalf("MigrateLegacyFindQueryJSON failed: %v", err)
	}
	if !changed {
		t.Fatal("changed should be true")
	}

	obj := decodeMigratedObject(t, migrated)

	for _, key := range legacyUseFlagKeys {
		assertKeyAbsent(t, obj, key)
	}

	for _, key := range []string{
		"words", "not_words", "tags", "reps", "rep_types", "ids",
		"timeis_words", "timeis_not_words", "timeis_tags",
		"calendar_start_date", "calendar_end_date",
		"map_radius", "map_latitude", "map_longitude",
		"plaing_time", "update_time", "mi_board_name",
		"period_of_time_start_time_second", "period_of_time_end_time_second", "period_of_time_week_of_days",
	} {
		assertNull(t, obj, key)
	}

	// クライアント専用キーと and 系フラグは不変
	if obj["query_id"] != "q1" {
		t.Errorf("query_id should be preserved: %v", obj["query_id"])
	}
	if obj["words_and"] != true {
		t.Errorf("words_and should be preserved: %v", obj["words_and"])
	}
	assertStringArray(t, obj, "hide_tags", []string{"h1"})
}

// use_X=true のグループは値が維持され、null/欠落の配列は [] に物質化されること
func TestMigrateLegacyFindQueryJSONEnabledGroups(t *testing.T) {
	legacy := `{
		"use_words": true, "words": ["w1"], "not_words": null,
		"use_tags": true, "tags": ["t1"],
		"use_reps": true,
		"use_rep_types": true, "rep_types": null,
		"use_ids": true, "ids": [],
		"use_timeis": true, "timeis_words": null, "timeis_not_words": ["tnw"],
		"use_timeis_tags": true, "timeis_tags": null,
		"use_calendar": true, "calendar_start_date": "2020-01-01T00:00:00+09:00", "calendar_end_date": null,
		"use_map": true, "map_radius": 0, "map_latitude": 35.6, "map_longitude": 139.7,
		"use_plaing": true, "plaing_time": "2021-05-05T12:00:00+09:00",
		"use_update_time": true, "update_time": "2022-02-02T00:00:00+09:00",
		"use_mi_board_name": true,
		"use_period_of_time": true, "period_of_time_start_time_second": 3600, "period_of_time_end_time_second": null, "period_of_time_week_of_days": null
	}`

	migrated, changed, err := MigrateLegacyFindQueryJSON([]byte(legacy))
	if err != nil {
		t.Fatalf("MigrateLegacyFindQueryJSON failed: %v", err)
	}
	if !changed {
		t.Fatal("changed should be true")
	}

	obj := decodeMigratedObject(t, migrated)

	for _, key := range legacyUseFlagKeys {
		assertKeyAbsent(t, obj, key)
	}

	assertStringArray(t, obj, "words", []string{"w1"})
	assertEmptyArray(t, obj, "not_words")
	assertStringArray(t, obj, "tags", []string{"t1"})
	assertEmptyArray(t, obj, "reps")
	assertEmptyArray(t, obj, "rep_types")
	assertEmptyArray(t, obj, "ids")
	assertEmptyArray(t, obj, "timeis_words")
	assertStringArray(t, obj, "timeis_not_words", []string{"tnw"})
	assertEmptyArray(t, obj, "timeis_tags")
	if obj["calendar_start_date"] != "2020-01-01T00:00:00+09:00" {
		t.Errorf("calendar_start_date should be preserved: %v", obj["calendar_start_date"])
	}
	assertNull(t, obj, "calendar_end_date")
	if obj["map_radius"] != json.Number("0") {
		t.Errorf("map_radius should be preserved as 0: %v", obj["map_radius"])
	}
	if obj["plaing_time"] != "2021-05-05T12:00:00+09:00" {
		t.Errorf("plaing_time should be preserved: %v", obj["plaing_time"])
	}
	if obj["update_time"] != "2022-02-02T00:00:00+09:00" {
		t.Errorf("update_time should be preserved: %v", obj["update_time"])
	}
	if obj["mi_board_name"] != "" {
		t.Errorf("mi_board_name should be materialized to empty string: %v", obj["mi_board_name"])
	}
	if obj["period_of_time_start_time_second"] != json.Number("3600") {
		t.Errorf("period_of_time_start_time_second should be preserved: %v", obj["period_of_time_start_time_second"])
	}
	assertNull(t, obj, "period_of_time_end_time_second")
	assertEmptyArray(t, obj, "period_of_time_week_of_days")
}

// 旧ゲートは UseTimeIs && UseTimeIsTags の複合なので、
// use_timeis=false なら use_timeis_tags=true でも timeis_tags は null になること
func TestMigrateLegacyFindQueryJSONTimeIsDisabledNullsTags(t *testing.T) {
	legacy := `{"use_timeis": false, "use_timeis_tags": true, "timeis_tags": ["tt1", "tt2"], "timeis_words": ["tw"]}`

	migrated, changed, err := MigrateLegacyFindQueryJSON([]byte(legacy))
	if err != nil {
		t.Fatalf("MigrateLegacyFindQueryJSON failed: %v", err)
	}
	if !changed {
		t.Fatal("changed should be true")
	}

	obj := decodeMigratedObject(t, migrated)
	assertNull(t, obj, "timeis_words")
	assertNull(t, obj, "timeis_tags")
}

// クライアント専用フラグ use_mi_sort_type / use_mi_check_state / use_include_id は
// キー削除のみで、値の mi_sort_type / mi_check_state は不変であること
func TestMigrateLegacyFindQueryJSONClientOnlyFlags(t *testing.T) {
	legacy := `{
		"use_include_id": true,
		"use_mi_sort_type": false, "mi_sort_type": "estimate_start_time",
		"use_mi_check_state": true, "mi_check_state": "uncheck",
		"keywords": "hello world",
		"devices_in_sidebar": ["pc1"]
	}`

	migrated, changed, err := MigrateLegacyFindQueryJSON([]byte(legacy))
	if err != nil {
		t.Fatalf("MigrateLegacyFindQueryJSON failed: %v", err)
	}
	if !changed {
		t.Fatal("changed should be true")
	}

	obj := decodeMigratedObject(t, migrated)
	assertKeyAbsent(t, obj, "use_include_id")
	assertKeyAbsent(t, obj, "use_mi_sort_type")
	assertKeyAbsent(t, obj, "use_mi_check_state")
	if obj["mi_sort_type"] != "estimate_start_time" {
		t.Errorf("mi_sort_type should be preserved: %v", obj["mi_sort_type"])
	}
	if obj["mi_check_state"] != "uncheck" {
		t.Errorf("mi_check_state should be preserved: %v", obj["mi_check_state"])
	}
	if obj["keywords"] != "hello world" {
		t.Errorf("keywords should be preserved: %v", obj["keywords"])
	}
	assertStringArray(t, obj, "devices_in_sidebar", []string{"pc1"})
}

// RYUU 設定のようにネストの奥に FindQuery を抱える JSON も変換されること
func TestMigrateLegacyFindQueryJSONNested(t *testing.T) {
	legacy := `[
		{"title": "item1", "find_kyou_query": {"use_tags": false, "tags": ["t1"], "use_words": true, "words": ["w1"]}},
		{"title": "item2", "find_kyou_query": null},
		{"title": "item3", "nested": {"deep": [{"find_kyou_query": {"use_calendar": false, "calendar_start_date": "2020-01-01T00:00:00+09:00"}}]}}
	]`

	migrated, changed, err := MigrateLegacyFindQueryJSON([]byte(legacy))
	if err != nil {
		t.Fatalf("MigrateLegacyFindQueryJSON failed: %v", err)
	}
	if !changed {
		t.Fatal("changed should be true")
	}

	var items []any
	decoder := json.NewDecoder(bytes.NewReader(migrated))
	decoder.UseNumber()
	if err := decoder.Decode(&items); err != nil {
		t.Fatalf("migrated JSON decode failed: %v", err)
	}

	item1Query := items[0].(map[string]any)["find_kyou_query"].(map[string]any)
	assertKeyAbsent(t, item1Query, "use_tags")
	assertKeyAbsent(t, item1Query, "use_words")
	assertNull(t, item1Query, "tags")
	assertStringArray(t, item1Query, "words", []string{"w1"})

	if items[1].(map[string]any)["find_kyou_query"] != nil {
		t.Errorf("null find_kyou_query should stay null")
	}

	item3Query := items[2].(map[string]any)["nested"].(map[string]any)["deep"].([]any)[0].(map[string]any)["find_kyou_query"].(map[string]any)
	assertKeyAbsent(t, item3Query, "use_calendar")
	assertNull(t, item3Query, "calendar_start_date")
}

// 冪等性: 1回目の出力にもう一度かけると changed=false かつバイト同一であること
func TestMigrateLegacyFindQueryJSONIdempotent(t *testing.T) {
	legacy := `{"use_tags": true, "tags": ["t1"], "use_words": false, "words": ["w1"], "not_words": []}`

	first, changed, err := MigrateLegacyFindQueryJSON([]byte(legacy))
	if err != nil {
		t.Fatalf("first migration failed: %v", err)
	}
	if !changed {
		t.Fatal("first migration should report changed")
	}

	second, changed, err := MigrateLegacyFindQueryJSON(first)
	if err != nil {
		t.Fatalf("second migration failed: %v", err)
	}
	if changed {
		t.Error("second migration should report changed=false")
	}
	if !bytes.Equal(first, second) {
		t.Errorf("second migration should return identical bytes:\nfirst:  %s\nsecond: %s", first, second)
	}
}

// 旧形式キーが無い JSON はバイト同一のまま返ること（非オブジェクトも含む）
func TestMigrateLegacyFindQueryJSONNoLegacyKeys(t *testing.T) {
	for _, raw := range []string{
		`{"tags": ["t1"], "words": null, "keywords": "abc"}`,
		`null`,
		`[]`,
		`"text"`,
		`123`,
		``,
		`   `,
	} {
		migrated, changed, err := MigrateLegacyFindQueryJSON([]byte(raw))
		if err != nil {
			t.Fatalf("MigrateLegacyFindQueryJSON(%q) failed: %v", raw, err)
		}
		if changed {
			t.Errorf("MigrateLegacyFindQueryJSON(%q) should report changed=false", raw)
		}
		if !bytes.Equal([]byte(raw), migrated) {
			t.Errorf("MigrateLegacyFindQueryJSON(%q) should return input unchanged, got %q", raw, migrated)
		}
	}
}

// json.Number により大きな整数の精度が保たれること
func TestMigrateLegacyFindQueryJSONNumberPrecision(t *testing.T) {
	legacy := `{"use_words": false, "words": [], "big_number": 9007199254740993, "period_of_time_start_time_second": 86399}`

	migrated, _, err := MigrateLegacyFindQueryJSON([]byte(legacy))
	if err != nil {
		t.Fatalf("MigrateLegacyFindQueryJSON failed: %v", err)
	}
	if !strings.Contains(string(migrated), "9007199254740993") {
		t.Errorf("big integer should keep exact precision: %s", migrated)
	}
	if !strings.Contains(string(migrated), "86399") {
		t.Errorf("period_of_time_start_time_second should keep exact value: %s", migrated)
	}
}

// use_plaing=true かつ plaing_time が null/欠落の行は null のまま移行されること
func TestMigrateLegacyFindQueryJSONPlaingWithoutTime(t *testing.T) {
	for _, raw := range []string{
		`{"use_plaing": true, "plaing_time": null}`,
		`{"use_plaing": true}`,
	} {
		migrated, changed, err := MigrateLegacyFindQueryJSON([]byte(raw))
		if err != nil {
			t.Fatalf("MigrateLegacyFindQueryJSON(%q) failed: %v", raw, err)
		}
		if !changed {
			t.Fatalf("MigrateLegacyFindQueryJSON(%q) should report changed", raw)
		}
		obj := decodeMigratedObject(t, migrated)
		assertKeyAbsent(t, obj, "use_plaing")
		assertNull(t, obj, "plaing_time")
	}
}

// 壊れた JSON はエラーになること
func TestMigrateLegacyFindQueryJSONBrokenJSON(t *testing.T) {
	_, _, err := MigrateLegacyFindQueryJSON([]byte(`{"use_tags": tru`))
	if err == nil {
		t.Fatal("broken JSON should return error")
	}
}
