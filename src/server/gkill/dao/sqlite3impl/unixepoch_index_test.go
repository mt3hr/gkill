package sqlite3impl

// 時刻範囲検索が式インデックスを使えているかの回帰テスト。
//
// 列に関数を適用した条件（datetime(列,'localtime') >= ...）では
// どんな索引を足しても SEARCH にならないため、
// GenerateFindSQLCommon 側の式と EnsureUnixepochIndex が張る式は
// 完全に一致していなければならない。
// CAST を挟む・'auto' を足す・strftime('%s',...) で書くといった些細な違いでも
// エラーにならず黙って全走査に戻るので、プランを直接検査して固定する。

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

func newUnixepochTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE KMEMO (IS_DELETED, ID, CONTENT, RELATED_TIME, UPDATE_TIME)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := EnsureUnixepochIndex(context.Background(), db, "KMEMO", "RELATED_TIME"); err != nil {
		t.Fatalf("EnsureUnixepochIndex: %v", err)
	}
	return db
}

func calendarRangeSQL(t *testing.T, from, to time.Time) (string, []any) {
	t.Helper()
	query := &find.FindQuery{
		CalendarStartDate: &from,
		CalendarEndDate:   &to,
	}
	whereCounter := 0
	queryArgs := []any{}
	where, err := GenerateFindSQLCommon(
		query, "KMEMO", "KMEMO", &whereCounter,
		false, "RELATED_TIME",
		[]string{"CONTENT"}, true, false,
		true, true, &queryArgs,
	)
	if err != nil {
		t.Fatalf("GenerateFindSQLCommon: %v", err)
	}
	return `SELECT ID FROM KMEMO WHERE ` + where, queryArgs
}

// カレンダー範囲検索が SCAN ではなく SEARCH になること。
func TestCalendarRangeUsesUnixepochIndex(t *testing.T) {
	db := newUnixepochTestDB(t)
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	sqlText, args := calendarRangeSQL(t, from, to)

	rows, err := db.Query(`EXPLAIN QUERY PLAN `+sqlText, args...)
	if err != nil {
		t.Fatalf("EXPLAIN QUERY PLAN failed (生成SQLが実行できない): %v / sql=%q", err, sqlText)
	}
	defer func() { _ = rows.Close() }()

	plan := ""
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		plan += detail + " | "
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate plan: %v", err)
	}

	if !strings.Contains(plan, "SEARCH") {
		t.Errorf("式インデックスが使われていない。plan=%q sql=%q", plan, sqlText)
	}
	if strings.Contains(plan, "TEMP B-TREE") {
		t.Errorf("ORDER BY で一時ソートが発生している。plan=%q", plan)
	}
}

// オフセットが混在していても正しい範囲が返ること。
// 実データの TAG.RELATED_TIME は +00:00 と +09:00 が混在しており、
// 素朴な文字列比較にすると結果が変わってしまう。
func TestCalendarRangeHandlesMixedTimezoneOffsets(t *testing.T) {
	db := newUnixepochTestDB(t)

	// いずれも実時刻は 2026-07-15T00:00:00Z 前後。表記だけ違う。
	rowsToInsert := []struct {
		id          string
		relatedTime string
	}{
		{"utc-in-range", "2026-07-15T00:00:00+00:00"},
		{"jst-in-range", "2026-07-15T09:00:00+09:00"},    // = 同じ瞬間
		{"jst-just-before", "2026-07-01T08:59:59+09:00"}, // = 2026-06-30T23:59:59Z 範囲外
		{"jst-just-after", "2026-07-01T09:00:01+09:00"},  // = 2026-07-01T00:00:01Z 範囲内
		{"utc-out-of-range", "2026-09-01T00:00:00+00:00"},
	}
	for _, r := range rowsToInsert {
		if _, err := db.Exec(`INSERT INTO KMEMO VALUES(0, ?, 'c', ?, ?)`, r.id, r.relatedTime, r.relatedTime); err != nil {
			t.Fatalf("insert %s: %v", r.id, err)
		}
	}

	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	sqlText, args := calendarRangeSQL(t, from, to)

	rows, err := db.Query(sqlText, args...)
	if err != nil {
		t.Fatalf("query failed: %v / sql=%q", err, sqlText)
	}
	defer func() { _ = rows.Close() }()

	got := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[id] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate: %v", err)
	}

	want := []string{"utc-in-range", "jst-in-range", "jst-just-after"}
	notWant := []string{"jst-just-before", "utc-out-of-range"}
	for _, id := range want {
		if !got[id] {
			t.Errorf("%s が範囲に入っていない (got=%v)", id, got)
		}
	}
	for _, id := range notWant {
		if got[id] {
			t.Errorf("%s が範囲に入ってしまっている (got=%v)", id, got)
		}
	}
}
