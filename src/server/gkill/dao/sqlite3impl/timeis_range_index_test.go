package sqlite3impl

// TIMEISキャッシュ表の START_TIME_UNIX / END_TIME_UNIX 索引が、なぜ「入れる」判断なのかを固定するテスト。
//
// この索引は2つの経路に逆向きの影響を与えるので、実行時間だけ見て足したり外したりすると
// 判断が揺れる(計測環境によってはベンチのns/opが同一コードで10ms〜17msまでぶれる)。
// クエリプランは決定的なので、そちらで根拠を残す。
//
//	経路                          索引なし   索引あり
//	----------------------------  ---------  ------------------------------------------
//	期間絞り込み(検索のたび)        SCAN T     SEARCH T USING INDEX (START>? AND START<?)
//	plaing判定(ダイアログを開く時)  SCAN T     SEARCH T USING INDEX (START<?)
//
// 期間絞り込みは範囲が閉じているので、走査する行が「その期間ぶん」に縮む(2万行→千行)。
// plaing判定は `? >= START_TIME_UNIX` が開いた範囲(表の約半分に当たる)なので、
// 索引を辿ってから表を引く形になり、素直に全表を舐めるより不利になる。
//
// それでも索引を採るのは、期間絞り込みが**すべての検索**で走るのに対し、
// plaing判定は show_attached_timeis が真の面(ダイアログ・詳細ペイン)でしか飛ばないため。
// 一覧の行では飛ばない(kyou-list-view.vue が false を渡している)。

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func timeIsPlanFor(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		t.Fatalf("failed to explain: %v", err)
	}
	defer rows.Close()
	details := []string{}
	for rows.Next() {
		var selectID, order, from int
		var detail string
		if err := rows.Scan(&selectID, &order, &from, &detail); err != nil {
			t.Fatalf("failed to scan plan: %v", err)
		}
		details = append(details, detail)
	}
	return strings.Join(details, " | ")
}

func TestTimeIsRangeIndexIsUsedByCalendarFilter(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:timeis_range_index_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	mustExec := func(query string) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Fatalf("failed to exec %q: %v", query, err)
		}
	}
	mustExec(`CREATE TABLE T (IS_DELETED NOT NULL, ID NOT NULL, TITLE NOT NULL, REP_NAME NOT NULL,
	  START_TIME_UNIX NOT NULL, END_TIME_UNIX, CREATE_TIME_UNIX NOT NULL, UPDATE_TIME_UNIX NOT NULL)`)
	mustExec(`CREATE INDEX IDX_ID_UPD ON T (ID, UPDATE_TIME_UNIX)`)

	const rowCount = 20000
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin: %v", err)
	}
	insertStmt, err := tx.PrepareContext(ctx, `INSERT INTO T VALUES (0,?,?,'rep',?,?,?,?)`)
	if err != nil {
		t.Fatalf("failed to prepare: %v", err)
	}
	for i := range rowCount {
		base := int64(1700000000 + i*60)
		if _, err := insertStmt.ExecContext(ctx, fmt.Sprintf("id-%08d", i), "title", base, base+30, base, base); err != nil {
			t.Fatalf("failed to seed: %v", err)
		}
	}
	insertStmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// GenerateFindSQLCommon が出す形(期間絞り込み + 最新版の相関サブクエリ)。
	// FindKyous では START_TIME_UNIX に RELATED_TIME_UNIX という別名が付くが、
	// SQLiteは別名を基底列へ解決するので索引の使われ方は同じ。
	calendarSQL := `SELECT ID FROM T WHERE START_TIME_UNIX >= ? AND START_TIME_UNIX <= ?` +
		` AND UPDATE_TIME_UNIX = (SELECT MAX(UPDATE_TIME_UNIX) FROM T AS INNER_TABLE WHERE ID = T.ID)`
	from, to := int64(1700000000+19000*60), int64(1700000000+20000*60)

	mustExec("PRAGMA analysis_limit=400")
	mustExec("ANALYZE")
	planWithout := timeIsPlanFor(t, db, calendarSQL, from, to)
	if !strings.Contains(planWithout, "SCAN T") {
		t.Fatalf("索引が無いときは全表走査のはず。実際のプラン: %s", planWithout)
	}

	mustExec(`CREATE INDEX IDX_START ON T (START_TIME_UNIX DESC)`)
	mustExec(`CREATE INDEX IDX_END ON T (END_TIME_UNIX DESC)`)
	mustExec("PRAGMA analysis_limit=400")
	mustExec("ANALYZE")

	planWith := timeIsPlanFor(t, db, calendarSQL, from, to)
	if !strings.Contains(planWith, "SEARCH T USING INDEX IDX_START") {
		t.Errorf("期間絞り込みがSTART_TIME_UNIXの索引を使っていない。プラン: %s", planWith)
	}
	if strings.Contains(planWith, "SCAN T") {
		t.Errorf("期間絞り込みで全表走査が残っている。プラン: %s", planWith)
	}
}
