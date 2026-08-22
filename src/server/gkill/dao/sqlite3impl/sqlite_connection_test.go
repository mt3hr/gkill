package sqlite3impl

// journal_mode を DELETE のまま変えない理由（WAL のサイドカーが持ち回りを壊す）:
// documents/adr/0013-keep-journal-mode-delete.md

// 実データDBの接続設定の回帰テスト。
//
// とくに journal_mode は DELETE から変えてはいけない。
// WALにすると -wal / -shm のサイドカーができ、.db 単体をコピーしても
// 未チェックポイントの内容が落ちる。gkillはrepを端末ごとの .db ファイルとして
// 持ち回る作りで、バックアップも単純なファイルコピーで済ませたいため。

import (
	"context"
	"path/filepath"
	"testing"
)

func TestGetSQLiteDBConnection_AppliesExpectedPragmas(t *testing.T) {
	ctx := context.Background()
	db, err := GetSQLiteDBConnection(ctx, filepath.Join(t.TempDir(), "pragma.db"))
	if err != nil {
		t.Fatalf("GetSQLiteDBConnection: %v", err)
	}
	defer func() { _ = db.Close() }()

	// PRAGMAはファイルが実在してから確定するものがあるので、先に1つ作る
	if _, err := db.ExecContext(ctx, `CREATE TABLE T(A)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	cases := []struct {
		pragma string
		want   string
		reason string
	}{
		{"journal_mode", "delete", "WALにすると -wal/-shm ができて .db 単体のバックアップが壊れる"},
		{"synchronous", "1", "NORMAL のまま。変えると耐久性の意味が変わる"},
		{"cache_size", "-8000", "既定の2MBでは大きいDBに足りない"},
		{"temp_store", "2", "MEMORY。ORDER BY の一時ソートがディスクに落ちるのを防ぐ"},
		{"busy_timeout", "6000", ""},
	}
	for _, c := range cases {
		var got string
		if err := db.QueryRowContext(ctx, "PRAGMA "+c.pragma).Scan(&got); err != nil {
			t.Errorf("PRAGMA %s: %v", c.pragma, err)
			continue
		}
		if got != c.want {
			t.Errorf("PRAGMA %s = %q, want %q (%s)", c.pragma, got, c.want, c.reason)
		}
	}

	// mmap_size は 0 でなければよい（実測で90MBのDB全走査が156ms -> 5.6msになる）
	var mmapSize int64
	if err := db.QueryRowContext(ctx, "PRAGMA mmap_size").Scan(&mmapSize); err != nil {
		t.Errorf("PRAGMA mmap_size: %v", err)
	} else if mmapSize <= 0 {
		t.Errorf("mmap_size = %d。大きいDBの読み出しが遅くなる", mmapSize)
	}

	// 既定の MaxIdleConns(2) のままだと、超えた接続が使用後に即閉じられ
	// SQLiteのページキャッシュが毎回捨てられる
	stats := db.Stats()
	if stats.MaxOpenConnections <= 1 {
		t.Errorf("MaxOpenConnections = %d。読み取り並列ができない", stats.MaxOpenConnections)
	}
}
