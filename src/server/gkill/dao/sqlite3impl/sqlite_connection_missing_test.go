package sqlite3impl

// 無いDBファイルを作らないことの回帰テスト。
// 作ってよいのは CreateSQLiteDBConnection と OpenRepositoryDBConnection(allowCreate=true) だけ。
// documents/adr/0223-never-recreate-a-missing-rep-db.md

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func assertNoFile(t *testing.T, filename string) {
	t.Helper()
	if _, err := os.Stat(filename); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s が作られている（stat err = %v）", filename, err)
	}
}

func TestGetSQLiteDBConnection_DoesNotCreateMissingFile(t *testing.T) {
	ctx := context.Background()
	missing := filepath.Join(t.TempDir(), "missing.db")
	db, err := GetSQLiteDBConnection(ctx, missing)
	if err == nil {
		_ = db.Close()
		t.Fatal("無いファイルを開けてしまった")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want os.ErrNotExist を包んだエラー", err)
	}
	assertNoFile(t, missing)
}

func TestOpenSQLiteDBConnectionForTx_DoesNotCreateMissingFile(t *testing.T) {
	ctx := context.Background()
	missing := filepath.Join(t.TempDir(), "missing.db")
	db, err := OpenSQLiteDBConnectionForTx(ctx, missing)
	if err == nil {
		_ = db.Close()
		t.Fatal("無いファイルを開けてしまった")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want os.ErrNotExist を包んだエラー", err)
	}
	assertNoFile(t, missing)
}

func TestOpenRepositoryDBConnection_CreatesOnlyWhenAllowed(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	missing := filepath.Join(dir, "readonly.db")
	if db, err := OpenRepositoryDBConnection(ctx, missing, false); err == nil {
		_ = db.Close()
		t.Fatal("allowCreate=false で無いファイルを開けてしまった")
	}
	assertNoFile(t, missing)

	created := filepath.Join(dir, "write.db")
	db, err := OpenRepositoryDBConnection(ctx, created, true)
	if err != nil {
		t.Fatalf("allowCreate=true で開けない: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.ExecContext(ctx, `CREATE TABLE T(A)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := os.Stat(created); err != nil {
		t.Errorf("allowCreate=true でファイルが作られていない: %v", err)
	}
}
