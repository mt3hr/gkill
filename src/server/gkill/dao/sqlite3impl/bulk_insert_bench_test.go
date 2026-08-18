package sqlite3impl

// キャッシュ再構築の INSERT を1行ずつ実行するのと multi-row にまとめるのの比較。
//
// **結論: multi-row INSERT にしてはいけない。** このベンチはその根拠を残すためにある。
//
// 再構築は全13型が共有する書き込みロックを保持したまま走るので、その所要時間は
// そのまま「全種類の検索が止まる時間」になる。1行ずつ ExecContext するのは
// Go↔SQLite の境界越えが行数ぶん起きるので遅そうに見えるが、実際は逆だった。
//
// modernc.org/sqlite は SQLite を Go へ変換した実装なので、
// プレースホルダを数千個持つ文の準備とバインドが非常に高くつく。
// 1行ぶんのプリペアド文を reset して使い回すほうが桁で速い。
//
// 実測(2万行, go test -run '^$' -bench BenchmarkInsert -benchmem -benchtime 5x, 2026-08-18)
//
//	書き方              ns/op          B/op        allocs/op
//	------------------  -------------  ----------  ---------
//	1行ずつ(現状)         79,963,800   24,173,108    399,642
//	500行ずつまとめる  1,376,614,280   21,979,544    200,591   ← 17倍遅い
//	 50行ずつまとめる  1,102,178,620   20,461,145    205,657   ← 14倍遅い
//
// 確保回数は半分になるが、所要時間が桁で悪化するので割に合わない。
// 「1行ずつのExecは無駄だから束ねよう」と思ったら、まずこのベンチを回すこと。

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

const benchInsertRowCount = 20_000

func newBenchInsertDB(tb testing.TB) *sql.DB {
	tb.Helper()
	db, err := sql.Open("sqlite", "file:bench_bulk_insert?mode=memory&cache=shared&_pragma=journal_mode(MEMORY)&_pragma=synchronous(OFF)")
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS BENCH_ROWS (
  IS_DELETED NOT NULL, ID NOT NULL, CONTENT NOT NULL,
  CREATE_APP NOT NULL, CREATE_DEVICE NOT NULL, CREATE_USER NOT NULL,
  UPDATE_APP NOT NULL, UPDATE_DEVICE NOT NULL, UPDATE_USER NOT NULL,
  REP_NAME NOT NULL, RELATED_TIME_UNIX NOT NULL, CREATE_TIME_UNIX NOT NULL, UPDATE_TIME_UNIX NOT NULL)`); err != nil {
		tb.Fatal(err)
	}
	return db
}

func benchInsertRow(i int) []any {
	return []any{
		false, fmt.Sprintf("id-%08d", i), fmt.Sprintf("content %d", i),
		"app", "device", "user", "app", "device", "user", "rep",
		int64(1700000000 + i), int64(1700000000 + i), int64(1700000000 + i),
	}
}

// BenchmarkInsertPerRow は現状のかたち。
func BenchmarkInsertPerRow(b *testing.B) {
	db := newBenchInsertDB(b)
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM BENCH_ROWS"); err != nil {
			b.Fatal(err)
		}
		stmt, err := tx.PrepareContext(ctx, "INSERT INTO BENCH_ROWS VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)")
		if err != nil {
			b.Fatal(err)
		}
		for i := range benchInsertRowCount {
			if _, err := stmt.ExecContext(ctx, benchInsertRow(i)...); err != nil {
				b.Fatal(err)
			}
		}
		stmt.Close()
		if err := tx.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkInsertMultiRow は500行ずつまとめたかたち。
func BenchmarkInsertMultiRow(b *testing.B) {
	db := newBenchInsertDB(b)
	ctx := context.Background()
	const chunk = 50
	placeholders := "(" + strings.TrimSuffix(strings.Repeat("?,", 13), ",") + ")"
	chunkSQL := "INSERT INTO BENCH_ROWS VALUES " + strings.TrimSuffix(strings.Repeat(placeholders+",", chunk), ",")
	b.ReportAllocs()
	for b.Loop() {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM BENCH_ROWS"); err != nil {
			b.Fatal(err)
		}
		stmt, err := tx.PrepareContext(ctx, chunkSQL)
		if err != nil {
			b.Fatal(err)
		}
		args := make([]any, 0, chunk*13)
		for i := range benchInsertRowCount {
			args = append(args, benchInsertRow(i)...)
			if len(args) == chunk*13 {
				if _, err := stmt.ExecContext(ctx, args...); err != nil {
					b.Fatal(err)
				}
				args = args[:0]
			}
		}
		stmt.Close()
		if err := tx.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}
