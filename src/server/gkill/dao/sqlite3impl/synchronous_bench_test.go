package sqlite3impl

// synchronous を NORMAL から FULL へ上げたときの書き込みコストの実測。
//
// **このベンチは ADR-0215 の Evidence のためにある。**
// journal_mode=DELETE のまま synchronous=NORMAL にしていると、電源断・I/O断で
// DBそのものが壊れうる。その耐久性をいくらで買っているかを数字で残しておくためのもの。
// 「FULLは遅いからNORMALへ戻そう」と思ったら、まずこのベンチを回すこと。
//
// 隣の bulk_insert_bench_test.go のベンチは mode=memory + synchronous(OFF) なので、
// fsync のコストを一切測れない。こちらは t.TempDir() の実ファイルを開く。
//
// 裸INSERTとトランザクションの両方を測るのは、gkillの書き込みが2種類あるため。
// 記録の追加は1件ずつの裸INSERT（＝文ごとに暗黙のトランザクションになり毎回fsyncする）で、
// キャッシュ再構築はトランザクションでまとめる（commitで1回fsyncする）。
// synchronous が効くのは fsync の回数ぶんなので、前者で差が大きく出る。
//
// 実測(100行, go test -run '^$' -bench BenchmarkSynchronous -benchmem -benchtime 20x -count 3, 2026-08-30)
// Intel Core i7-10510U / 書き込み先は内蔵NVMe SSD
//
//	書き方                        NORMAL(1)   FULL(2)     差
//	----------------------------  ----------  ----------  ----
//	1件ずつの裸INSERT (100行)       442.5 ms    522.1 ms   +18%
//	トランザクション1回 (100行)        4.68 ms     5.52 ms   +18%
//
// 3回測って中央値。裸INSERTは文ごとに暗黙のトランザクションになるのでfsyncが100回、
// トランザクション側は1回。所要時間の絶対値は95倍違うが、増加率はどちらも+18%だった。
// 確保量・確保回数は変わらない（fsyncを待つ時間が増えるだけなので当然ではある）。

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

const benchSyncRowCount = 100

// benchSyncDSN は sqliteDataDSNParams の synchronous だけを差し替えたDSNを返す。
//
// 定数側を直接いじらずに組み立て直しているのは、比較の対象が
// 「synchronous 以外は本番と同じ条件」でなければ意味がないため。
// sqliteDataDSNParams を変更したらこちらも合わせること。
func benchSyncDSN(mode string) string {
	return "?_pragma=busy_timeout(6000)" +
		"&_pragma=synchronous(" + mode + ")" +
		"&_pragma=journal_mode(DELETE)" +
		"&_pragma=cache_size(-8000)" +
		"&_pragma=temp_store(MEMORY)" +
		"&_pragma=mmap_size(268435456)"
}

func newBenchSyncDB(tb testing.TB, mode string) *sql.DB {
	tb.Helper()
	// mmap_size を効かせたままにするため、実ファイルで開く
	path := filepath.Join(tb.TempDir(), "sync_bench.db")
	db, err := sql.Open("sqlite", "file:"+path+benchSyncDSN(mode))
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = db.Close() })

	// 仕込みが効いているかの確認。ここが食い違うと、
	// 2つのベンチが同じ条件を測っていることになり比較が無意味になる
	var got string
	if err := db.QueryRow("PRAGMA synchronous").Scan(&got); err != nil {
		tb.Fatal(err)
	}
	want := map[string]string{"NORMAL": "1", "FULL": "2"}[mode]
	if got != want {
		tb.Fatalf("PRAGMA synchronous = %q, want %q (mode=%s)", got, want, mode)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS BENCH_SYNC_ROWS (
  IS_DELETED NOT NULL, ID NOT NULL, CONTENT NOT NULL,
  REP_NAME NOT NULL, RELATED_TIME_UNIX NOT NULL)`); err != nil {
		tb.Fatal(err)
	}
	return db
}

func benchSyncRow(i int) []any {
	return []any{
		false, fmt.Sprintf("id-%08d", i), fmt.Sprintf("content %d", i),
		"rep", int64(1700000000 + i),
	}
}

// benchSyncBareInsert は記録を1件ずつ足すかたち（文ごとに暗黙のトランザクション）。
func benchSyncBareInsert(b *testing.B, mode string) {
	b.Helper()
	db := newBenchSyncDB(b, mode)
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := db.ExecContext(ctx, "DELETE FROM BENCH_SYNC_ROWS"); err != nil {
			b.Fatal(err)
		}
		stmt, err := db.PrepareContext(ctx, "INSERT INTO BENCH_SYNC_ROWS VALUES (?,?,?,?,?)")
		if err != nil {
			b.Fatal(err)
		}
		for i := range benchSyncRowCount {
			if _, err := stmt.ExecContext(ctx, benchSyncRow(i)...); err != nil {
				b.Fatal(err)
			}
		}
		_ = stmt.Close()
	}
}

// benchSyncTxInsert はキャッシュ再構築のかたち（commitで1回だけfsync）。
func benchSyncTxInsert(b *testing.B, mode string) {
	b.Helper()
	db := newBenchSyncDB(b, mode)
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM BENCH_SYNC_ROWS"); err != nil {
			b.Fatal(err)
		}
		stmt, err := tx.PrepareContext(ctx, "INSERT INTO BENCH_SYNC_ROWS VALUES (?,?,?,?,?)")
		if err != nil {
			b.Fatal(err)
		}
		for i := range benchSyncRowCount {
			if _, err := stmt.ExecContext(ctx, benchSyncRow(i)...); err != nil {
				b.Fatal(err)
			}
		}
		_ = stmt.Close()
		if err := tx.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSynchronousNormalBareInsert(b *testing.B) { benchSyncBareInsert(b, "NORMAL") }
func BenchmarkSynchronousFullBareInsert(b *testing.B)   { benchSyncBareInsert(b, "FULL") }
func BenchmarkSynchronousNormalTxInsert(b *testing.B)   { benchSyncTxInsert(b, "NORMAL") }
func BenchmarkSynchronousFullTxInsert(b *testing.B)     { benchSyncTxInsert(b, "FULL") }
