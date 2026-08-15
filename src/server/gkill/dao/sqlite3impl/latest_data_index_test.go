package sqlite3impl

// 最新版判定（相関サブクエリ）が索引で引けていることの回帰テスト。
//
// GenerateFindSQLCommon は onlyLatestData のとき
//   UPDATE_TIME_UNIX = ( SELECT MAX(UPDATE_TIME_UNIX) FROM T AS INNER_TABLE WHERE ID = T.ID )
// を足す。これは行ごとに評価される相関サブクエリなので、ID で引ける索引が無いと
// 1行につき全表走査になり O(n^2) に落ちる。実データ規模（IDFで数十万行）では
// 索引が外れた瞬間に検索が数十分コースになるが、エラーは出ず「ただ遅い」だけなので
// 気づけない。プランを直接検査して固定する。
//
// 2026-08-16 の実測（22万行・全列取得＋行走査）:
//   相関サブクエリ(この形)  636ms   ← 現状。被覆索引で引けている
//   GROUP BY 結合へ書き換え  789ms   ← 24%遅い。書き換えても改善しない
//   ウィンドウ関数           720ms   ← 一時B-treeを作るぶん更に遅い
//   (ID, UPDATE_TIME_UNIX) の索引追加は効果が無く、索引作成が297ms→1,379msに増えるだけ。
// つまりこのSQLは既に妥当で、重いのはSQLではなく取得件数そのもの。

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
)

// newLatestDataTestDB はキャッシュrepと同じ形の表と索引を作る。
func newLatestDataTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE T (IS_DELETED, ID NOT NULL, TARGET_FILE, RELATED_TIME_UNIX, UPDATE_TIME_UNIX)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	// キャッシュrepが張っているものと同じ索引
	if _, err := db.Exec(`CREATE INDEX INDEX_T_UNIX ON T (ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX)`); err != nil {
		t.Fatalf("create index: %v", err)
	}
	// プランを決めるのに統計が要るので、ある程度の行数を入れてANALYZEする
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO T VALUES (0, ?, 'file.jpg', ?, ?)`)
	if err != nil {
		t.Fatalf("prepare insert: %v", err)
	}
	for i := range 2000 {
		id := fmt.Sprintf("id-%06d", i)
		if _, err := stmt.Exec(id, 1500000000+i, 1500000000+i); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	if err := stmt.Close(); err != nil {
		t.Fatalf("close insert stmt: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := db.Exec(`ANALYZE`); err != nil {
		t.Fatalf("analyze: %v", err)
	}
	return db
}

func explainPlan(t *testing.T, db *sql.DB, query string) []string {
	t.Helper()

	rows, err := db.Query("EXPLAIN QUERY PLAN " + query)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer func() { _ = rows.Close() }()

	plans := []string{}
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		plans = append(plans, detail)
	}
	return plans
}

// TestLatestDataSubqueryUsesIndex は最新版判定の相関サブクエリが
// 索引検索(SEARCH)になっていることを固定する。
// ここが SCAN に落ちると、1行ごとの全表走査になって検索全体が O(n^2) になる。
func TestLatestDataSubqueryUsesIndex(t *testing.T) {
	db := newLatestDataTestDB(t)

	query := `SELECT IS_DELETED, ID, TARGET_FILE, RELATED_TIME_UNIX, UPDATE_TIME_UNIX FROM T ` +
		`WHERE UPDATE_TIME_UNIX = ( SELECT MAX(UPDATE_TIME_UNIX) FROM T AS INNER_TABLE WHERE ID = T.ID )`
	plans := explainPlan(t, db, query)

	innerPlanFound := false
	for _, plan := range plans {
		if !strings.Contains(plan, "INNER_TABLE") {
			continue
		}
		innerPlanFound = true
		if !strings.Contains(plan, "SEARCH") || !strings.Contains(plan, "INDEX") {
			t.Errorf("最新版判定のサブクエリが索引で引けていない（1行ごとの全表走査になる）: %q", plan)
		}
	}
	if !innerPlanFound {
		t.Errorf("サブクエリのプランが見つからない: %v", plans)
	}
}
