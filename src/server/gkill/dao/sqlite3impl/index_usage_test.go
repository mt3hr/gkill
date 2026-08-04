package sqlite3impl

// 追加した索引が実際に使われている（SCANではなくSEARCHになる）ことの回帰テスト。
//
// 索引はズレていてもエラーにならず黙って全走査に戻るだけなので、
// プランを直接検査しないと退行に気づけない。

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

func queryPlan(t *testing.T, db *sql.DB, sqlText string, args ...any) string {
	t.Helper()
	rows, err := db.Query("EXPLAIN QUERY PLAN "+sqlText, args...)
	if err != nil {
		t.Fatalf("EXPLAIN QUERY PLAN failed: %v / sql=%q", err, sqlText)
	}
	defer func() { _ = rows.Close() }()

	plan := &strings.Builder{}
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		plan.WriteString(detail)
		plan.WriteString(" | ")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate plan: %v", err)
	}
	return plan.String()
}

func newIndexTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// 一時表は WHERE TX_ID = ? AND USER_ID = ? AND DEVICE = ? で引く。
// commit_tx は1コミットにつき13repへこれを2種類ずつ投げるので、
// 索引が無いと1コミットあたり26回の全表スキャンになる。
func TestEnsureTxIDIndex_UsedByTxIDLookup(t *testing.T) {
	ctx := context.Background()
	db := newIndexTestDB(t)
	if _, err := db.Exec(`CREATE TABLE KMEMO (IS_DELETED, ID, CONTENT, RELATED_TIME, UPDATE_TIME, USER_ID, DEVICE, TX_ID)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := EnsureTxIDIndex(ctx, db, "KMEMO"); err != nil {
		t.Fatalf("EnsureTxIDIndex: %v", err)
	}

	plan := queryPlan(t, db,
		`SELECT ID FROM KMEMO WHERE TX_ID = ? AND USER_ID = ? AND DEVICE = ?`,
		"tx-1", "user-1", "device-1")

	if !strings.Contains(plan, "SEARCH") {
		t.Errorf("TX_ID索引が使われていない。plan=%q", plan)
	}
}

// MI は5射影のUNIONで射影ごとに別の時刻列を使う。
// CREATE_TIME だけ索引があった頃は残り3本が全走査だった。
func TestEnsureUnixepochIndex_UsedByEachMiTimeColumn(t *testing.T) {
	ctx := context.Background()
	db := newIndexTestDB(t)
	if _, err := db.Exec(`CREATE TABLE MI (ID, TITLE, BOARD_NAME, IS_CHECKED, LIMIT_TIME, ESTIMATE_START_TIME, ESTIMATE_END_TIME, CREATE_TIME, UPDATE_TIME)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := EnsureUnixepochIndex(ctx, db, "MI", "CREATE_TIME", "LIMIT_TIME", "ESTIMATE_START_TIME", "ESTIMATE_END_TIME"); err != nil {
		t.Fatalf("EnsureUnixepochIndex: %v", err)
	}

	for _, column := range []string{"CREATE_TIME", "LIMIT_TIME", "ESTIMATE_START_TIME", "ESTIMATE_END_TIME"} {
		// GenerateFindSQLCommon が生成するのと同じ式で引く
		plan := queryPlan(t, db,
			`SELECT ID FROM MI WHERE unixepoch(`+column+`) >= unixepoch(?) AND unixepoch(`+column+`) <= unixepoch(?)`,
			"2026-07-01T00:00:00+09:00", "2026-08-01T00:00:00+09:00")
		if !strings.Contains(plan, "SEARCH") {
			t.Errorf("%s の式インデックスが使われていない。plan=%q", column, plan)
		}
	}
}

// 差分同期はこの列で絞る。LIMITが実質無制限なので索引が無いと全表スキャンになる。
func TestEnsureUnixColumnIndex_UsedByUpdatedTimeLookup(t *testing.T) {
	ctx := context.Background()
	db := newIndexTestDB(t)
	if _, err := db.Exec(`CREATE TABLE ADDR (TARGET_ID PRIMARY KEY, LATEST_DATA_REPOSITORY_NAME, LATEST_DATA_REPOSITORY_ADDRESS_UPDATED_TIME_UNIX)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := EnsureUnixColumnIndex(ctx, db, "ADDR", "LATEST_DATA_REPOSITORY_ADDRESS_UPDATED_TIME_UNIX"); err != nil {
		t.Fatalf("EnsureUnixColumnIndex: %v", err)
	}

	plan := queryPlan(t, db,
		`SELECT TARGET_ID FROM ADDR WHERE LATEST_DATA_REPOSITORY_ADDRESS_UPDATED_TIME_UNIX >= ?`, 0)
	if !strings.Contains(plan, "SEARCH") {
		t.Errorf("更新時刻の索引が使われていない。plan=%q", plan)
	}
}
