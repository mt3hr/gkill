package message

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"modernc.org/sqlite"
)

// reason は「何が起きたか」を消費者へ伝える唯一の経路。
// 分類は errors.Is / errors.As だけで行い、%w で包まれていても・順序が違っても同じ答えになることを固定する。

// TestReasonOf_Classification は代表的な error の分類を表で固定する。
func TestReasonOf_Classification(t *testing.T) {
	wrap := func(err error) error { return fmt.Errorf("error at add kmemo user id = u device = d: %w", err) }

	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil は空", nil, ""},
		{"分類できない error は空", errors.New("something"), ""},
		{"Reasoner は自分の reason", &ReasonError{Msg: "x", Reason: ReasonPluginBusy}, ReasonPluginBusy},
		{"Reasoner は包まれていても拾う", wrap(&ReasonError{Msg: "x", Reason: ReasonWriteRepMissing}), ReasonWriteRepMissing},
		{"DeadlineExceeded は timeout", wrap(context.DeadlineExceeded), ReasonTimeout},
		{"Canceled は canceled", wrap(context.Canceled), ReasonCanceled},
		{"ErrNotExist は storage_unavailable", wrap(&fs.PathError{Op: "open", Path: "x.db", Err: fs.ErrNotExist}), ReasonStorageUnavailable},
		{"ErrPermission は storage_readonly", wrap(&fs.PathError{Op: "open", Path: "x.db", Err: fs.ErrPermission}), ReasonStorageReadonly},
		{"ENOSPC は storage_full", wrap(&fs.PathError{Op: "write", Path: "x.db", Err: syscall.ENOSPC}), ReasonStorageFull},
		{"url.Error は external_fetch_failed", wrap(&url.Error{Op: "Get", URL: "https://example.com", Err: errors.New("refused")}), ReasonExternalFetchFailed},
		{"net の timeout は timeout", wrap(&net.DNSError{Err: "x", IsTimeout: true}), ReasonTimeout},
		{"net のそれ以外は external_fetch_failed", wrap(&net.DNSError{Err: "x", IsNotFound: true}), ReasonExternalFetchFailed},
		{"Reasoner が context より優先", wrap(fmt.Errorf("%w: %w", &ReasonError{Msg: "x", Reason: ReasonPluginError}, context.DeadlineExceeded)), ReasonPluginError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ReasonOf(c.err); got != c.want {
				t.Errorf("ReasonOf = %q, want %q", got, c.want)
			}
		})
	}
}

// sqliteErrorWithCode は modernc.org/sqlite が実際に返す *sqlite.Error を起こして返す。
// 構造体のフィールドが非公開で組み立てられないので、本物の失敗を起こす。
func sqliteErrorWithCode(t *testing.T, want int) error {
	t.Helper()
	dir := t.TempDir()
	var err error
	switch want {
	case sqliteResultCantOpen:
		// 存在しないディレクトリの下のファイル。mode=rw なら作成しないので開けない。
		db := openSQLite(t, "file:"+filepath.ToSlash(filepath.Join(dir, "missing", "x.db"))+"?mode=rw")
		_, err = db.Exec("CREATE TABLE t (id INTEGER)")
	case sqliteResultNotADB:
		path := filepath.Join(dir, "garbage.db")
		if writeErr := os.WriteFile(path, []byte("this is not a sqlite database file at all, just text padding......"), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		db := openSQLite(t, "file:"+filepath.ToSlash(path))
		_, err = db.Exec("CREATE TABLE t (id INTEGER)")
	case sqliteResultReadonly:
		path := filepath.Join(dir, "ro.db")
		setup := openSQLite(t, "file:"+filepath.ToSlash(path))
		if _, setupErr := setup.Exec("CREATE TABLE t (id INTEGER)"); setupErr != nil {
			t.Fatal(setupErr)
		}
		_ = setup.Close()
		db := openSQLite(t, "file:"+filepath.ToSlash(path)+"?mode=ro")
		_, err = db.Exec("INSERT INTO t VALUES (1)")
	case sqliteResultBusy:
		path := filepath.Join(dir, "busy.db")
		writer := openSQLite(t, "file:"+filepath.ToSlash(path))
		if _, setupErr := writer.Exec("CREATE TABLE t (id INTEGER)"); setupErr != nil {
			t.Fatal(setupErr)
		}
		tx, txErr := writer.Begin()
		if txErr != nil {
			t.Fatal(txErr)
		}
		defer func() { _ = tx.Rollback() }()
		if _, lockErr := tx.Exec("INSERT INTO t VALUES (1)"); lockErr != nil {
			t.Fatal(lockErr)
		}
		other := openSQLite(t, "file:"+filepath.ToSlash(path)+"?_pragma=busy_timeout(0)")
		_, err = other.Exec("INSERT INTO t VALUES (2)")
	default:
		t.Fatalf("この結果コードを起こす手順が無い: %d", want)
	}
	if err == nil {
		t.Fatalf("結果コード %d の失敗が起きなかった", want)
	}
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		t.Fatalf("modernc.org/sqlite の *sqlite.Error ではない: %T %v", err, err)
	}
	if sqliteErr.Code()&0xff != want {
		t.Fatalf("結果コード = %d, want %d (%v)", sqliteErr.Code(), want, err)
	}
	return err
}

func openSQLite(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestReasonOf_SQLite はドライバが実際に返すエラーで、結果コード → reason の対応を固定する。
// 文字列ではなく Code() を見ているので、文言が変わっても分類は変わらない。
func TestReasonOf_SQLite(t *testing.T) {
	cases := []struct {
		name string
		code int
		want string
	}{
		{"CANTOPEN は storage_unavailable（USB が外れた等）", sqliteResultCantOpen, ReasonStorageUnavailable},
		{"NOTADB は storage_corrupted", sqliteResultNotADB, ReasonStorageCorrupted},
		{"READONLY は storage_readonly", sqliteResultReadonly, ReasonStorageReadonly},
		{"BUSY は db_busy", sqliteResultBusy, ReasonDBBusy},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := sqliteErrorWithCode(t, c.code)
			wrapped := fmt.Errorf("error at add kmemo: %w", err)
			if got := ReasonOf(wrapped); got != c.want {
				t.Errorf("ReasonOf = %q, want %q (%v)", got, c.want, err)
			}
		})
	}
}

// TestReasonForError_CodeFallback は Cause が無い（または分類できない）とき、コードの表から reason が決まることを固定する。
func TestReasonForError_CodeFallback(t *testing.T) {
	if got := reasonForError(LocalOnlyAccessDeniedError, nil); got != ReasonLocalOnlyAccess {
		t.Errorf("LocalOnlyAccessDeniedError の reason = %q, want %q", got, ReasonLocalOnlyAccess)
	}
	if got := reasonForError(WriteRepMissingError, errors.New("unclassified")); got != ReasonWriteRepMissing {
		t.Errorf("WriteRepMissingError + 分類不能な Cause の reason = %q, want %q", got, ReasonWriteRepMissing)
	}
	// Cause から分類できればそちらが勝つ
	if got := reasonForError(WriteRepMissingError, context.DeadlineExceeded); got != ReasonTimeout {
		t.Errorf("Cause 優先になっていない: %q", got)
	}
	if got := reasonForError(AddKmemoError, nil); got != "" {
		t.Errorf("表に無いコードで Cause も無いなら空のはず: %q", got)
	}
}

// TestReasonTokens_Vocabulary は語彙の一覧が定数と一致し、重複しないことを固定する。
// verify_docs が reasonTokens の件数を資料と突き合わせるので、定数を足したら一覧にも足すこと。
func TestReasonTokens_Vocabulary(t *testing.T) {
	seen := map[string]bool{}
	for _, token := range reasonTokens {
		if token == "" {
			t.Error("空のトークンがある")
		}
		if seen[token] {
			t.Errorf("重複: %q", token)
		}
		seen[token] = true
	}
	for code, reason := range errorCodeReason {
		if !seen[reason] {
			t.Errorf("errorCodeReason[%q] = %q が reasonTokens に無い", code, reason)
		}
	}
	if len(reasonTokens) != 14 {
		t.Errorf("reason の語彙数 = %d, want 14（資料の表と verify_docs も更新すること）", len(reasonTokens))
	}
}
