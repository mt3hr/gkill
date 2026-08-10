package find

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFindQueryJSONRoundTrip(t *testing.T) {
	words := []string{"test", "hello"}
	tags := []string{"tag1", "tag2"}
	reps := []string{"rep1"}
	now := time.Now().UTC().Truncate(time.Second)

	original := FindQuery{
		Words:             words,
		WordsAnd:          true,
		Tags:              tags,
		Reps:              reps,
		CalendarStartDate: &now,
		IsDeleted:         false,
		UpdateCache:       true,
		ForMi:             true,
		MiCheckState:      All,
		MiSortType:        CreateTime,
		OnlyLatestData:    true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored FindQuery
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(restored.Words) != len(original.Words) {
		t.Errorf("Words length: got %d, want %d", len(restored.Words), len(original.Words))
	}
	for i := range original.Words {
		if restored.Words[i] != original.Words[i] {
			t.Errorf("Words[%d]: got %q, want %q", i, restored.Words[i], original.Words[i])
		}
	}
	if len(restored.Tags) != len(original.Tags) {
		t.Errorf("Tags length: got %d, want %d", len(restored.Tags), len(original.Tags))
	}
	if len(restored.Reps) != len(original.Reps) {
		t.Errorf("Reps length: got %d, want %d", len(restored.Reps), len(original.Reps))
	}
	if restored.CalendarStartDate == nil || !restored.CalendarStartDate.Equal(now) {
		t.Errorf("CalendarStartDate: got %v, want %v", restored.CalendarStartDate, now)
	}
	if restored.UpdateCache != original.UpdateCache {
		t.Errorf("UpdateCache: got %v, want %v", restored.UpdateCache, original.UpdateCache)
	}
	if restored.ForMi != original.ForMi {
		t.Errorf("ForMi: got %v, want %v", restored.ForMi, original.ForMi)
	}
	if restored.OnlyLatestData != original.OnlyLatestData {
		t.Errorf("OnlyLatestData: got %v, want %v", restored.OnlyLatestData, original.OnlyLatestData)
	}
}

// ゼロ値のFindQueryは「全フィルタ未使用」であること
func TestFindQueryDefaultValues(t *testing.T) {
	var q FindQuery

	if q.IsDeleted {
		t.Error("IsDeleted should be false by default")
	}
	if q.ForMi {
		t.Error("ForMi should be false by default")
	}
	if q.Words != nil {
		t.Error("Words should be nil by default")
	}
	if q.Tags != nil {
		t.Error("Tags should be nil by default")
	}
	if q.Reps != nil {
		t.Error("Reps should be nil by default")
	}
	if q.CalendarStartDate != nil {
		t.Error("CalendarStartDate should be nil by default")
	}
	if q.CalendarEndDate != nil {
		t.Error("CalendarEndDate should be nil by default")
	}
	if q.MapRadius != nil {
		t.Error("MapRadius should be nil by default")
	}
	if q.PlaingTime != nil {
		t.Error("PlaingTime should be nil by default")
	}
	if q.MiBoardName != nil {
		t.Error("MiBoardName should be nil by default")
	}

	// ゼロ値では全ゲートが「未使用」
	if q.HasWordFilter() {
		t.Error("HasWordFilter should be false for zero value")
	}
	if q.HasTimeIsFilter() {
		t.Error("HasTimeIsFilter should be false for zero value")
	}
	if q.HasCalendarFilter() {
		t.Error("HasCalendarFilter should be false for zero value")
	}
	if q.HasMapFilter() {
		t.Error("HasMapFilter should be false for zero value")
	}
	if q.HasPeriodOfTimeFilter() {
		t.Error("HasPeriodOfTimeFilter should be false for zero value")
	}
}

// ゲートヘルパーが nil / 非nil空 / 値ありを正しく判定すること
func TestFindQueryGateHelpers(t *testing.T) {
	now := time.Now()
	radius := 500.0
	lat := 35.0
	lng := 139.0
	sec := int64(3600)

	// 非nil空スライスは「有効(空指定)」としてゲートが立つ
	emptyWords := FindQuery{Words: []string{}}
	if !emptyWords.HasWordFilter() {
		t.Error("HasWordFilter should be true for non-nil empty Words")
	}
	notWordsOnly := FindQuery{NotWords: []string{"x"}}
	if !notWordsOnly.HasWordFilter() {
		t.Error("HasWordFilter should be true when only NotWords is set")
	}

	emptyTimeIsWords := FindQuery{TimeIsWords: []string{}}
	if !emptyTimeIsWords.HasTimeIsFilter() {
		t.Error("HasTimeIsFilter should be true for non-nil empty TimeIsWords (任意のTimeIsに覆われたKyou)")
	}
	timeIsNotWordsOnly := FindQuery{TimeIsNotWords: []string{"x"}}
	if !timeIsNotWordsOnly.HasTimeIsFilter() {
		t.Error("HasTimeIsFilter should be true when only TimeIsNotWords is set")
	}

	startOnly := FindQuery{CalendarStartDate: &now}
	if !startOnly.HasCalendarFilter() {
		t.Error("HasCalendarFilter should be true when only start date is set")
	}
	endOnly := FindQuery{CalendarEndDate: &now}
	if !endOnly.HasCalendarFilter() {
		t.Error("HasCalendarFilter should be true when only end date is set")
	}

	// 地図は3値が揃って初めて有効
	fullMap := FindQuery{MapRadius: &radius, MapLatitude: &lat, MapLongitude: &lng}
	if !fullMap.HasMapFilter() {
		t.Error("HasMapFilter should be true when all three values are set")
	}
	partialMap := FindQuery{MapRadius: &radius, MapLatitude: &lat}
	if partialMap.HasMapFilter() {
		t.Error("HasMapFilter should be false when longitude is missing")
	}

	periodStart := FindQuery{PeriodOfTimeStartTimeSecond: &sec}
	if !periodStart.HasPeriodOfTimeFilter() {
		t.Error("HasPeriodOfTimeFilter should be true when start second is set")
	}
	periodWeek := FindQuery{PeriodOfTimeWeekOfDays: []WeekOfDays{}}
	if !periodWeek.HasPeriodOfTimeFilter() {
		t.Error("HasPeriodOfTimeFilter should be true for non-nil empty WeekOfDays")
	}
}

// JSONの null / キー欠落 / [] がそれぞれ nil / nil / 非nil空 に復元されること
func TestFindQueryNilVsEmptyJSON(t *testing.T) {
	// null と キー欠落 → nil
	var fromNull FindQuery
	if err := json.Unmarshal([]byte(`{"tags": null, "words": null}`), &fromNull); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if fromNull.Tags != nil {
		t.Error("Tags should be nil for JSON null")
	}
	if fromNull.Words != nil {
		t.Error("Words should be nil for JSON null")
	}

	var fromAbsent FindQuery
	if err := json.Unmarshal([]byte(`{}`), &fromAbsent); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if fromAbsent.Tags != nil || fromAbsent.Reps != nil || fromAbsent.IDs != nil {
		t.Error("absent keys should decode to nil slices")
	}

	// [] → 非nil空
	var fromEmpty FindQuery
	if err := json.Unmarshal([]byte(`{"tags": [], "timeis_words": []}`), &fromEmpty); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if fromEmpty.Tags == nil || len(fromEmpty.Tags) != 0 {
		t.Error("Tags should be non-nil empty for JSON []")
	}
	if fromEmpty.TimeIsWords == nil {
		t.Error("TimeIsWords should be non-nil empty for JSON []")
	}
	if !fromEmpty.HasTimeIsFilter() {
		t.Error("HasTimeIsFilter should be true after decoding timeis_words: []")
	}

	// Marshal側: nil → null、[] → []
	data, err := json.Marshal(FindQuery{Tags: []string{}, Words: nil})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !strings.Contains(string(data), `"tags":[]`) {
		t.Errorf("non-nil empty Tags should marshal to []: %s", data)
	}
	if !strings.Contains(string(data), `"words":null`) {
		t.Errorf("nil Words should marshal to null: %s", data)
	}
}

func TestMiCheckState_Values(t *testing.T) {
	if string(All) != "all" {
		t.Errorf("All: got %q, want %q", string(All), "all")
	}
	if string(Checked) != "checked" {
		t.Errorf("Checked: got %q, want %q", string(Checked), "checked")
	}
	if string(UncCheck) != "uncheck" {
		t.Errorf("UncCheck: got %q, want %q", string(UncCheck), "uncheck")
	}

	// 全列挙値が別々の文字列であること（同じ値になると絞り込みが黙って別状態を拾う）
	seen := map[MiCheckState]struct{}{}
	for _, state := range []MiCheckState{All, Checked, UncCheck} {
		if _, exist := seen[state]; exist {
			t.Errorf("Duplicate MiCheckState value: %q", string(state))
		}
		seen[state] = struct{}{}
	}
	if len(seen) != 3 {
		t.Errorf("Expected 3 distinct MiCheckState values, got %d", len(seen))
	}
}

// MiCheckState は JSON では素の文字列であること。
// MarshalJSON が []byte を marshal していた頃は base64 になり、UnmarshalJSON が
// 素の文字列を期待しているので往復で値が壊れていた。
// 「往復後に非空」だけを見ると base64 のままでも通ってしまうので、元の値との一致まで見る。
func TestMiCheckState_JSONRoundTrip(t *testing.T) {
	type wrapper struct {
		State MiCheckState `json:"state"`
	}

	for _, state := range []MiCheckState{All, Checked, UncCheck} {
		original := wrapper{State: state}
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal failed for %q: %v", string(state), err)
		}

		if want := `{"state":"` + string(state) + `"}`; string(data) != want {
			t.Errorf("Marshal result: got %s, want %s", data, want)
		}

		var restored wrapper
		err = json.Unmarshal(data, &restored)
		if err != nil {
			t.Fatalf("Unmarshal failed for %q: %v", string(state), err)
		}

		if restored.State != state {
			t.Errorf("MiCheckState round-trip: got %q, want %q", string(restored.State), string(state))
		}
	}
}

func TestMiSortType_Values(t *testing.T) {
	tests := []struct {
		got  MiSortType
		want string
	}{
		{CreateTime, "create_time"},
		{EstimateStartTime, "estimate_start_time"},
		{EstimateEndTime, "estimate_end_time"},
		{LimitTime, "limit_time"},
	}

	for _, tc := range tests {
		if string(tc.got) != tc.want {
			t.Errorf("MiSortType: got %q, want %q", string(tc.got), tc.want)
		}
	}

	// 全列挙値が別々の文字列であること（同じ値になると並び替えが黙って別キーになる）
	seen := map[MiSortType]struct{}{}
	for _, tc := range tests {
		if _, exist := seen[tc.got]; exist {
			t.Errorf("Duplicate MiSortType value: %q", string(tc.got))
		}
		seen[tc.got] = struct{}{}
	}
	if len(seen) != 4 {
		t.Errorf("Expected 4 distinct MiSortType values, got %d", len(seen))
	}
}

// MiSortType は JSON では素の文字列であること。理由は TestMiCheckState_JSONRoundTrip と同じ。
func TestMiSortType_JSONRoundTrip(t *testing.T) {
	type wrapper struct {
		Sort MiSortType `json:"sort"`
	}

	for _, sortType := range []MiSortType{CreateTime, EstimateStartTime, EstimateEndTime, LimitTime} {
		original := wrapper{Sort: sortType}
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal failed for %q: %v", string(sortType), err)
		}

		if want := `{"sort":"` + string(sortType) + `"}`; string(data) != want {
			t.Errorf("Marshal result: got %s, want %s", data, want)
		}

		var restored wrapper
		err = json.Unmarshal(data, &restored)
		if err != nil {
			t.Fatalf("Unmarshal failed for %q: %v", string(sortType), err)
		}

		if restored.Sort != sortType {
			t.Errorf("MiSortType round-trip: got %q, want %q", string(restored.Sort), string(sortType))
		}
	}
}

func TestWeekOfDays_Values(t *testing.T) {
	tests := []struct {
		day  WeekOfDays
		want int
	}{
		{SunDay, 0},
		{MonDay, 1},
		{TuesDay, 2},
		{WednesDay, 3},
		{ThursDay, 4},
		{FriDay, 5},
		{SaturDay, 6},
	}

	for _, tc := range tests {
		if int(tc.day) != tc.want {
			t.Errorf("WeekOfDays: got %d, want %d", int(tc.day), tc.want)
		}
	}
}

func TestWeekOfDays_AllDaysCovered(t *testing.T) {
	days := []WeekOfDays{SunDay, MonDay, TuesDay, WednesDay, ThursDay, FriDay, SaturDay}

	if len(days) != 7 {
		t.Fatalf("Expected 7 days, got %d", len(days))
	}

	seen := make(map[int]bool)
	for _, d := range days {
		val := int(d)
		if val < 0 || val > 6 {
			t.Errorf("WeekOfDays value %d out of range 0-6", val)
		}
		if seen[val] {
			t.Errorf("Duplicate WeekOfDays value: %d", val)
		}
		seen[val] = true
	}

	for i := 0; i <= 6; i++ {
		if !seen[i] {
			t.Errorf("Missing WeekOfDays value: %d", i)
		}
	}
}

func TestFindQueryWithCalendarDates(t *testing.T) {
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	original := FindQuery{
		CalendarStartDate: &startDate,
		CalendarEndDate:   &endDate,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored FindQuery
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !restored.HasCalendarFilter() {
		t.Error("HasCalendarFilter should be true")
	}
	if restored.CalendarStartDate == nil {
		t.Fatal("CalendarStartDate should not be nil")
	}
	if !restored.CalendarStartDate.Equal(startDate) {
		t.Errorf("CalendarStartDate: got %v, want %v", restored.CalendarStartDate, startDate)
	}
	if restored.CalendarEndDate == nil {
		t.Fatal("CalendarEndDate should not be nil")
	}
	if !restored.CalendarEndDate.Equal(endDate) {
		t.Errorf("CalendarEndDate: got %v, want %v", restored.CalendarEndDate, endDate)
	}
}

func TestFindQueryMiFields(t *testing.T) {
	boardName := "test_board"
	original := FindQuery{
		ForMi:           true,
		MiBoardName:     &boardName,
		MiCheckState:    Checked,
		MiSortType:      EstimateStartTime,
		IncludeCreateMi: true,
		IncludeCheckMi:  true,
		IncludeLimitMi:  false,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored FindQuery
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !restored.ForMi {
		t.Error("ForMi should be true")
	}
	if restored.MiBoardName == nil || *restored.MiBoardName != "test_board" {
		t.Errorf("MiBoardName: got %v, want %q", restored.MiBoardName, "test_board")
	}
	// MiCheckState / MiSortType はカスタムMarshalJSONを持つ。
	// 素の文字列で往復し、元の値と一致すること
	if restored.MiCheckState != Checked {
		t.Errorf("MiCheckState: got %q, want %q", string(restored.MiCheckState), string(Checked))
	}
	if restored.MiSortType != EstimateStartTime {
		t.Errorf("MiSortType: got %q, want %q", string(restored.MiSortType), string(EstimateStartTime))
	}
	if !restored.IncludeCreateMi {
		t.Error("IncludeCreateMi should be true")
	}
	if !restored.IncludeCheckMi {
		t.Error("IncludeCheckMi should be true")
	}
	if restored.IncludeLimitMi {
		t.Error("IncludeLimitMi should be false")
	}
}

func TestFindQuery_AllFiltersEnabled(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	startSec := int64(3600)
	endSec := int64(7200)
	radius := 500.5
	latitude := 35.681236
	longitude := 139.767125
	boardName := "all_filters_board"

	original := FindQuery{
		UpdateCache:                 true,
		IsDeleted:                   true,
		Tags:                        []string{"t1", "t2"},
		HideTags:                    []string{"ht1"},
		TagsAnd:                     true,
		Reps:                        []string{"r1"},
		RepTypes:                    []string{"kmemo", "nlog"},
		IDs:                         []string{"id1", "id2"},
		Words:                       []string{"w1", "w2"},
		WordsAnd:                    true,
		NotWords:                    []string{"nw1"},
		TimeIsWords:                 []string{"tiw1"},
		TimeIsNotWords:              []string{"tinw1"},
		TimeIsWordsAnd:              true,
		TimeIsTags:                  []string{"tit1"},
		HideTimeIsTags:              []string{"htit1"},
		TimeIsTagsAnd:               true,
		CalendarStartDate:           &now,
		CalendarEndDate:             &now,
		MapRadius:                   &radius,
		MapLatitude:                 &latitude,
		MapLongitude:                &longitude,
		IncludeCreateMi:             true,
		IncludeCheckMi:              true,
		IncludeLimitMi:              true,
		IncludeStartMi:              true,
		IncludeEndMi:                true,
		IncludeEndTimeIs:            true,
		PlaingTime:                  &now,
		UpdateTime:                  &now,
		IsImageOnly:                 true,
		ForMi:                       true,
		MiBoardName:                 &boardName,
		PeriodOfTimeStartTimeSecond: &startSec,
		PeriodOfTimeEndTimeSecond:   &endSec,
		PeriodOfTimeWeekOfDays:      []WeekOfDays{MonDay, FriDay},
		MiCheckState:                Checked,
		MiSortType:                  LimitTime,
		OnlyLatestData:              true,
		IncludeDeletedData:          true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored FindQuery
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify bool flags
	if !restored.UpdateCache {
		t.Error("UpdateCache should be true")
	}
	if !restored.IsDeleted {
		t.Error("IsDeleted should be true")
	}
	if !restored.WordsAnd {
		t.Error("WordsAnd should be true")
	}
	if !restored.TimeIsWordsAnd {
		t.Error("TimeIsWordsAnd should be true")
	}
	if !restored.TimeIsTagsAnd {
		t.Error("TimeIsTagsAnd should be true")
	}
	if !restored.IsImageOnly {
		t.Error("IsImageOnly should be true")
	}
	if !restored.ForMi {
		t.Error("ForMi should be true")
	}
	if !restored.OnlyLatestData {
		t.Error("OnlyLatestData should be true")
	}
	if !restored.IncludeDeletedData {
		t.Error("IncludeDeletedData should be true")
	}

	// Verify all gates enabled after round-trip
	if !restored.HasWordFilter() {
		t.Error("HasWordFilter should be true")
	}
	if !restored.HasTimeIsFilter() {
		t.Error("HasTimeIsFilter should be true")
	}
	if !restored.HasCalendarFilter() {
		t.Error("HasCalendarFilter should be true")
	}
	if !restored.HasMapFilter() {
		t.Error("HasMapFilter should be true")
	}
	if !restored.HasPeriodOfTimeFilter() {
		t.Error("HasPeriodOfTimeFilter should be true")
	}

	// Verify slices
	if len(restored.Tags) != 2 {
		t.Errorf("Tags length: got %d, want 2", len(restored.Tags))
	}
	if len(restored.RepTypes) != 2 {
		t.Errorf("RepTypes length: got %d, want 2", len(restored.RepTypes))
	}
	if len(restored.IDs) != 2 {
		t.Errorf("IDs length: got %d, want 2", len(restored.IDs))
	}

	// Verify pointer fields
	if restored.MapRadius == nil || *restored.MapRadius != 500.5 {
		t.Errorf("MapRadius: got %v, want 500.5", restored.MapRadius)
	}
	if restored.MapLatitude == nil || *restored.MapLatitude != 35.681236 {
		t.Errorf("MapLatitude: got %v, want 35.681236", restored.MapLatitude)
	}
	if restored.PlaingTime == nil || !restored.PlaingTime.Equal(now) {
		t.Errorf("PlaingTime: got %v, want %v", restored.PlaingTime, now)
	}
	if restored.MiBoardName == nil || *restored.MiBoardName != "all_filters_board" {
		t.Errorf("MiBoardName: got %v, want %q", restored.MiBoardName, "all_filters_board")
	}

	// Verify period of time
	if restored.PeriodOfTimeStartTimeSecond == nil || *restored.PeriodOfTimeStartTimeSecond != 3600 {
		t.Error("PeriodOfTimeStartTimeSecond mismatch")
	}
	if restored.PeriodOfTimeEndTimeSecond == nil || *restored.PeriodOfTimeEndTimeSecond != 7200 {
		t.Error("PeriodOfTimeEndTimeSecond mismatch")
	}
	if len(restored.PeriodOfTimeWeekOfDays) != 2 {
		t.Errorf("PeriodOfTimeWeekOfDays length: got %d, want 2", len(restored.PeriodOfTimeWeekOfDays))
	}
}

func TestFindQuery_NilDateHandling(t *testing.T) {
	original := FindQuery{
		CalendarStartDate: nil,
		CalendarEndDate:   nil,
		UpdateTime:        nil,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored FindQuery
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.CalendarStartDate != nil {
		t.Error("CalendarStartDate should be nil after round-trip")
	}
	if restored.CalendarEndDate != nil {
		t.Error("CalendarEndDate should be nil after round-trip")
	}
	if restored.UpdateTime != nil {
		t.Error("UpdateTime should be nil after round-trip")
	}
	if restored.PeriodOfTimeStartTimeSecond != nil {
		t.Error("PeriodOfTimeStartTimeSecond should be nil after round-trip")
	}
	if restored.PeriodOfTimeEndTimeSecond != nil {
		t.Error("PeriodOfTimeEndTimeSecond should be nil after round-trip")
	}
}

func TestFindQuery_EmptySlices(t *testing.T) {
	original := FindQuery{
		Words:    []string{},
		Tags:     []string{},
		Reps:     []string{},
		NotWords: []string{},
		IDs:      []string{},
		RepTypes: []string{},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored FindQuery
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Empty slices should survive round-trip as non-nil empty slices
	if restored.Words == nil {
		t.Error("Words should not be nil after round-trip of empty slice")
	}
	if len(restored.Words) != 0 {
		t.Errorf("Words length: got %d, want 0", len(restored.Words))
	}
	if restored.Tags == nil {
		t.Error("Tags should not be nil after round-trip of empty slice")
	}
	if len(restored.Tags) != 0 {
		t.Errorf("Tags length: got %d, want 0", len(restored.Tags))
	}
	if restored.Reps == nil {
		t.Error("Reps should not be nil after round-trip of empty slice")
	}
	if len(restored.Reps) != 0 {
		t.Errorf("Reps length: got %d, want 0", len(restored.Reps))
	}
	if restored.NotWords == nil {
		t.Error("NotWords should not be nil after round-trip of empty slice")
	}
	if len(restored.NotWords) != 0 {
		t.Errorf("NotWords length: got %d, want 0", len(restored.NotWords))
	}
	if restored.IDs == nil {
		t.Error("IDs should not be nil after round-trip of empty slice")
	}
	if len(restored.IDs) != 0 {
		t.Errorf("IDs length: got %d, want 0", len(restored.IDs))
	}
	if restored.RepTypes == nil {
		t.Error("RepTypes should not be nil after round-trip of empty slice")
	}
	if len(restored.RepTypes) != 0 {
		t.Errorf("RepTypes length: got %d, want 0", len(restored.RepTypes))
	}
}

func TestFindQuery_DataTypeFilter(t *testing.T) {
	original := FindQuery{
		RepTypes: []string{"kmemo", "timeis", "lantana", "kc", "nlog", "urlog", "mi"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored FindQuery
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.RepTypes == nil {
		t.Error("RepTypes should be non-nil (filter enabled)")
	}
	if len(restored.RepTypes) != 7 {
		t.Fatalf("RepTypes length: got %d, want 7", len(restored.RepTypes))
	}

	expectedTypes := []string{"kmemo", "timeis", "lantana", "kc", "nlog", "urlog", "mi"}
	for i, expected := range expectedTypes {
		if restored.RepTypes[i] != expected {
			t.Errorf("RepTypes[%d]: got %q, want %q", i, restored.RepTypes[i], expected)
		}
	}
}
