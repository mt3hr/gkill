package sqlite3impl

import (
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

func TestEscapeSQLite_SingleQuote(t *testing.T) {
	result := EscapeSQLite("it's")
	if result != "it''s" {
		t.Errorf("EscapeSQLite(\"it's\") = %q, want %q", result, "it''s")
	}
}

func TestEscapeSQLite_NoQuotes(t *testing.T) {
	result := EscapeSQLite("hello world")
	if result != "hello world" {
		t.Errorf("EscapeSQLite(\"hello world\") = %q, want %q", result, "hello world")
	}
}

func TestEscapeSQLite_MultipleQuotes(t *testing.T) {
	result := EscapeSQLite("it's a 'test'")
	if result != "it''s a ''test''" {
		t.Errorf("EscapeSQLite(\"it's a 'test'\") = %q, want %q", result, "it''s a ''test''")
	}
}

func TestEscapeSQLite_EmptyString(t *testing.T) {
	result := EscapeSQLite("")
	if result != "" {
		t.Errorf("EscapeSQLite(\"\") = %q, want %q", result, "")
	}
}

func TestEscapeSQLite_JapaneseText(t *testing.T) {
	result := EscapeSQLite("テスト'データ")
	if result != "テスト''データ" {
		t.Errorf("EscapeSQLite(\"テスト'データ\") = %q, want %q", result, "テスト''データ")
	}
}

func TestQuoteIdent_Simple(t *testing.T) {
	result := QuoteIdent("column_name")
	expected := `"column_name"`
	if result != expected {
		t.Errorf("QuoteIdent(\"column_name\") = %q, want %q", result, expected)
	}
}

func TestQuoteIdent_WithDoubleQuotes(t *testing.T) {
	result := QuoteIdent(`col"name`)
	expected := `"col""name"`
	if result != expected {
		t.Errorf("QuoteIdent(\"col\\\"name\") = %q, want %q", result, expected)
	}
}

func TestGenerateNewID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for range 100 {
		id := GenerateNewID()
		if id == "" {
			t.Fatal("GenerateNewID returned empty string")
		}
		if ids[id] {
			t.Fatalf("GenerateNewID produced duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestTimeLayout_IsValid(t *testing.T) {
	if TimeLayout == "" {
		t.Fatal("TimeLayout is empty")
	}
	// Should be Go RFC3339-like format
	expected := "2006-01-02T15:04:05-07:00"
	if TimeLayout != expected {
		t.Errorf("TimeLayout = %q, want %q", TimeLayout, expected)
	}
}

func TestGenerateFindSQLCommon_EmptyQuery(t *testing.T) {
	query := &find.FindQuery{}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty query should produce a tautology " 0 = 0 "
	if !strings.Contains(sql, "0 = 0") {
		t.Errorf("expected tautology '0 = 0' in sql, got %q", sql)
	}
	if len(queryArgs) != 0 {
		t.Errorf("expected 0 queryArgs for empty query, got %d", len(queryArgs))
	}
}

func TestGenerateFindSQLCommon_UseIDs(t *testing.T) {
	query := &find.FindQuery{
		UseIDs: true,
		IDs:    []string{"id-1", "id-2", "id-3"},
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "ID IN (") {
		t.Errorf("expected 'ID IN (' in sql, got %q", sql)
	}
	// Should have 3 query args for the 3 IDs
	if len(queryArgs) != 3 {
		t.Errorf("expected 3 queryArgs, got %d", len(queryArgs))
	}
	for i, expected := range []string{"id-1", "id-2", "id-3"} {
		if queryArgs[i] != expected {
			t.Errorf("queryArgs[%d] = %v, want %v", i, queryArgs[i], expected)
		}
	}
}

func TestGenerateFindSQLCommon_UseIDsEmpty(t *testing.T) {
	query := &find.FindQuery{
		UseIDs: true,
		IDs:    []string{},
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty IDs with UseIDs=true should produce a contradiction
	if !strings.Contains(sql, "0 = 1") {
		t.Errorf("expected '0 = 1' for empty IDs, got %q", sql)
	}
}

func TestGenerateFindSQLCommon_UseWords(t *testing.T) {
	query := &find.FindQuery{
		UseWords: true,
		Words:    []string{"hello"},
		WordsAnd: true,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "LIKE") {
		t.Errorf("expected LIKE in sql for word search, got %q", sql)
	}
	// Should have args for word search (TITLE LIKE and ID LIKE)
	if len(queryArgs) < 2 {
		t.Errorf("expected at least 2 queryArgs for word search, got %d", len(queryArgs))
	}
	// First arg should be the word wrapped with %
	if queryArgs[0] != "%hello%" {
		t.Errorf("queryArgs[0] = %v, want %%hello%%", queryArgs[0])
	}
}

func TestGenerateFindSQLCommon_UseWordsOr(t *testing.T) {
	query := &find.FindQuery{
		UseWords: true,
		Words:    []string{"foo", "bar"},
		WordsAnd: false,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "LIKE") {
		t.Errorf("expected LIKE in sql for OR word search, got %q", sql)
	}
	// Each word produces 2 args (column LIKE + ID LIKE), 2 words = 4 args
	if len(queryArgs) != 4 {
		t.Errorf("expected 4 queryArgs for 2-word OR search, got %d", len(queryArgs))
	}
}

func TestGenerateFindSQLCommon_UseCalendar(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.Local)
	query := &find.FindQuery{
		UseCalendar:       true,
		CalendarStartDate: &start,
		CalendarEndDate:   &end,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "datetime") {
		t.Errorf("expected 'datetime' in sql for calendar search, got %q", sql)
	}
	if !strings.Contains(sql, ">=") {
		t.Errorf("expected '>=' in sql for calendar start date, got %q", sql)
	}
	if !strings.Contains(sql, "<=") {
		t.Errorf("expected '<=' in sql for calendar end date, got %q", sql)
	}
	if len(queryArgs) != 2 {
		t.Errorf("expected 2 queryArgs for calendar range, got %d", len(queryArgs))
	}
}

func TestGenerateFindSQLCommon_OnlyLatestData(t *testing.T) {
	query := &find.FindQuery{}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		true, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "UPDATE_TIME = ( SELECT UPDATE_TIME FROM") || !strings.Contains(sql, "ORDER BY datetime(INNER_TABLE.UPDATE_TIME) DESC LIMIT 1") {
		t.Errorf("expected latest data subquery with datetime() in sql, got %q", sql)
	}
}

func TestGenerateFindSQLCommon_AppendOrderBy(t *testing.T) {
	query := &find.FindQuery{}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		true, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "ORDER BY RELATED_TIME DESC") {
		t.Errorf("expected ORDER BY in sql, got %q", sql)
	}
}

func TestGenerateFindSQLCommon_IgnoreCase(t *testing.T) {
	query := &find.FindQuery{
		UseWords: true,
		Words:    []string{"Test"},
		WordsAnd: true,
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, true, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "LOWER") {
		t.Errorf("expected LOWER in sql for case-insensitive search, got %q", sql)
	}
}

func TestGenerateFindSQLCommon_NotWords(t *testing.T) {
	query := &find.FindQuery{
		UseWords: true,
		NotWords: []string{"exclude"},
	}
	whereCounter := 0
	queryArgs := []any{}

	sql, err := GenerateFindSQLCommon(
		query, "MY_TABLE", "T", &whereCounter,
		false, "RELATED_TIME",
		[]string{"TITLE"}, true, false,
		false, false, &queryArgs,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "NOT LIKE") {
		t.Errorf("expected 'NOT LIKE' in sql for not-words, got %q", sql)
	}
}
