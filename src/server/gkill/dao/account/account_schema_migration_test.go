package account

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// newSchema100AccountDB はスキーマ1.0.0相当のaccount.dbを作る。
// 1.0.0はPASSWORD_SHA256に無塩SHA-256をそのまま入れていた。
func newSchema100AccountDB(t *testing.T, accounts map[string]*string) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "account.db")
	db, err := sql.Open("sqlite", "file:"+filename)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	stmts := []string{
		`CREATE TABLE "ACCOUNT" (
  USER_ID PRIMARY KEY NOT NULL,
  PASSWORD_SHA256,
  IS_ADMIN NOT NULL,
  IS_ENABLE NOT NULL,
  PASSWORD_RESET_TOKEN
);`,
		`CREATE TABLE GKILL_META_INFO (KEY NOT NULL, VALUE, PRIMARY KEY(KEY));`,
		`INSERT INTO GKILL_META_INFO(KEY, VALUE) VALUES('SCHEMA_VERSION_ACCOUNT', '1.0.0');`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("failed to exec %q: %v", stmt, err)
		}
	}

	for userID, passwordSha256 := range accounts {
		_, err := db.ExecContext(ctx,
			`INSERT INTO ACCOUNT(USER_ID, PASSWORD_SHA256, IS_ADMIN, IS_ENABLE, PASSWORD_RESET_TOKEN) VALUES(?, ?, ?, ?, NULL)`,
			userID, passwordSha256, userID == "admin", true)
		if err != nil {
			t.Fatalf("failed to insert account %s: %v", userID, err)
		}
	}
	return filename
}

func TestMigrateFrom100ResetsAllPasswords(t *testing.T) {
	oldHash := testCredential
	filename := newSchema100AccountDB(t, map[string]*string{
		"admin":     &oldHash,
		"e2e_user":  &oldHash,
		"no_passwd": nil,
	})

	ctx := context.Background()
	dao, err := NewAccountDAOSQLite3Impl(ctx, filename)
	if err != nil {
		t.Fatalf("NewAccountDAOSQLite3Impl failed: %v", err)
	}
	defer dao.Close(ctx)

	accounts, err := dao.GetAllAccounts(ctx)
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}
	if len(accounts) != 3 {
		t.Fatalf("len(accounts) = %d, want 3", len(accounts))
	}

	seenTokens := map[string]string{}
	for _, target := range accounts {
		if target.PasswordHash != nil {
			t.Errorf("%s: PasswordHash = %q, want nil (移行で全員のパスワードを無効化するはず)", target.UserID, *target.PasswordHash)
		}
		if target.PasswordResetToken == nil {
			t.Fatalf("%s: PasswordResetToken is nil, want a freshly issued token", target.UserID)
		}
		if target.PasswordResetTokenExpiration == nil {
			t.Errorf("%s: PasswordResetTokenExpiration is nil, want an expiration", target.UserID)
		}
		if other, ok := seenTokens[*target.PasswordResetToken]; ok {
			t.Errorf("%s と %s に同じリセットトークンが発行された", target.UserID, other)
		}
		seenTokens[*target.PasswordResetToken] = target.UserID

		// 旧パスワードでログインできないこと
		ok, err := target.VerifyPassword(oldHash)
		if err != nil {
			t.Fatalf("%s: VerifyPassword failed: %v", target.UserID, err)
		}
		if ok {
			t.Errorf("%s: 移行後も旧パスワードでログインできてしまった", target.UserID)
		}
	}

	// IS_ADMIN / IS_ENABLE は保たれていること
	for _, target := range accounts {
		if !target.IsEnable {
			t.Errorf("%s: IsEnable = false, want true", target.UserID)
		}
		if (target.UserID == "admin") != target.IsAdmin {
			t.Errorf("%s: IsAdmin = %v", target.UserID, target.IsAdmin)
		}
	}
}

// newSchema100AccountDBWithoutVersionRow は GKILL_META_INFO を持たない
// 1.0.0 相当の account.db を作る。バージョン管理の仕組みが入る前のDBはこの形。
func newSchema100AccountDBWithoutVersionRow(t *testing.T, accounts map[string]*string) string {
	t.Helper()
	filename := newSchema100AccountDB(t, accounts)

	db, err := sql.Open("sqlite", "file:"+filename)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), `DROP TABLE GKILL_META_INFO`); err != nil {
		t.Fatalf("failed to drop GKILL_META_INFO: %v", err)
	}
	return filename
}

// TestMigrateFrom100WithoutVersionRow は、バージョン行が無いだけの旧DBを
// 「新規DB」と誤認せずに移行することを確認する。
//
// 誤認すると、旧スキーマのまま現行版として記録してしまい、以降
// PASSWORD_HASH を参照する全クエリが「no such column」で落ち続ける。
// しかもバージョンは現行値なので移行も二度と走らない（同梱のサンプルデータが
// まさにこの状態で、ログインどころかアカウント一覧の取得すらできなかった）。
func TestMigrateFrom100WithoutVersionRow(t *testing.T) {
	oldHash := testCredential
	filename := newSchema100AccountDBWithoutVersionRow(t, map[string]*string{"admin": &oldHash})

	ctx := context.Background()
	dao, err := NewAccountDAOSQLite3Impl(ctx, filename)
	if err != nil {
		t.Fatalf("NewAccountDAOSQLite3Impl failed: %v", err)
	}
	defer func() {
		if err := dao.Close(ctx); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	// 移行後のスキーマで読めること（旧スキーマのままだとここで no such column になる）
	accounts, err := dao.GetAllAccounts(ctx)
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("accounts = %d件, want 1件", len(accounts))
	}
	// 1.0.0 は無塩SHA-256をそのまま資格情報にしていたので、移行時に無効化される
	if accounts[0].PasswordHash != nil && *accounts[0].PasswordHash != "" {
		t.Errorf("PasswordHash = %q, 移行時に無効化されるはず", *accounts[0].PasswordHash)
	}

	db, err := sql.Open("sqlite", "file:"+filename)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	version := ""
	if err := db.QueryRowContext(ctx, "SELECT VALUE FROM GKILL_META_INFO WHERE KEY = 'SCHEMA_VERSION_ACCOUNT'").Scan(&version); err != nil {
		t.Fatalf("failed to get schema version: %v", err)
	}
	if version != CURRENT_SCHEMA_VERSION_ACCOUNT_DAO {
		t.Errorf("schema version = %q, want %q", version, CURRENT_SCHEMA_VERSION_ACCOUNT_DAO)
	}
}

// TestFreshDBWithoutVersionRowIsNotTreatedAsOld は、本当に新規のDB
// （ACCOUNTテーブルがまだ無い）を旧スキーマ扱いしないことを確認する。
// 上の判定を「バージョン行が無ければ移行」だけにすると、新規DBで移行が走ってしまう。
func TestFreshDBWithoutVersionRowIsNotTreatedAsOld(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "account.db")

	ctx := context.Background()
	dao, err := NewAccountDAOSQLite3Impl(ctx, filename)
	if err != nil {
		t.Fatalf("NewAccountDAOSQLite3Impl failed: %v", err)
	}
	defer func() {
		if err := dao.Close(ctx); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	accounts, err := dao.GetAllAccounts(ctx)
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("accounts = %d件, want 0件", len(accounts))
	}
}

func TestMigrateFrom100RenamesColumnAndBumpsVersion(t *testing.T) {
	oldHash := testCredential
	filename := newSchema100AccountDB(t, map[string]*string{"admin": &oldHash})

	ctx := context.Background()
	dao, err := NewAccountDAOSQLite3Impl(ctx, filename)
	if err != nil {
		t.Fatalf("NewAccountDAOSQLite3Impl failed: %v", err)
	}
	if err := dao.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	db, err := sql.Open("sqlite", "file:"+filename)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	columns := map[string]bool{}
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(ACCOUNT)")
	if err != nil {
		t.Fatalf("PRAGMA table_info failed: %v", err)
	}
	for rows.Next() {
		var cid int
		var name string
		var columnType, notNull, defaultValue, primaryKey any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			t.Fatalf("scan table info failed: %v", err)
		}
		columns[name] = true
	}
	rows.Close()

	if columns["PASSWORD_SHA256"] {
		t.Error("PASSWORD_SHA256 が残っている。PASSWORD_HASH にリネームされるはず")
	}
	if !columns["PASSWORD_HASH"] {
		t.Error("PASSWORD_HASH が無い")
	}
	if !columns["PASSWORD_RESET_TOKEN_EXPIRATION"] {
		t.Error("PASSWORD_RESET_TOKEN_EXPIRATION が無い")
	}

	version := ""
	if err := db.QueryRowContext(ctx, "SELECT VALUE FROM GKILL_META_INFO WHERE KEY = 'SCHEMA_VERSION_ACCOUNT'").Scan(&version); err != nil {
		t.Fatalf("failed to get schema version: %v", err)
	}
	if version != CURRENT_SCHEMA_VERSION_ACCOUNT_DAO {
		t.Errorf("schema version = %q, want %q", version, CURRENT_SCHEMA_VERSION_ACCOUNT_DAO)
	}
}

func TestMigrateFrom100IsIdempotentAcrossReopen(t *testing.T) {
	oldHash := testCredential
	filename := newSchema100AccountDB(t, map[string]*string{"admin": &oldHash})
	ctx := context.Background()

	dao, err := NewAccountDAOSQLite3Impl(ctx, filename)
	if err != nil {
		t.Fatalf("first open failed: %v", err)
	}
	first, err := dao.GetAllAccounts(ctx)
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}
	if err := dao.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// 2回目以降の起動では移行は走らず、発行済みのトークンも変わらないこと
	dao2, err := NewAccountDAOSQLite3Impl(ctx, filename)
	if err != nil {
		t.Fatalf("second open failed: %v", err)
	}
	defer dao2.Close(ctx)
	second, err := dao2.GetAllAccounts(ctx)
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}

	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("len(first) = %d, len(second) = %d, want 1 and 1", len(first), len(second))
	}
	if *first[0].PasswordResetToken != *second[0].PasswordResetToken {
		t.Error("再起動でリセットトークンが再発行された。移行が毎回走っている")
	}
}

func TestSetPasswordAfterMigration(t *testing.T) {
	oldHash := testCredential
	filename := newSchema100AccountDB(t, map[string]*string{"admin": &oldHash})
	ctx := context.Background()

	dao, err := NewAccountDAOSQLite3Impl(ctx, filename)
	if err != nil {
		t.Fatalf("NewAccountDAOSQLite3Impl failed: %v", err)
	}
	defer dao.Close(ctx)

	target, err := dao.GetAccount(ctx, "admin")
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}

	newHash, err := HashPassword(testCredential)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	target.PasswordHash = &newHash
	target.PasswordResetToken = nil
	target.PasswordResetTokenExpiration = nil
	if _, err := dao.UpdateAccount(ctx, target); err != nil {
		t.Fatalf("UpdateAccount failed: %v", err)
	}

	got, err := dao.GetAccount(ctx, "admin")
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if got.PasswordResetToken != nil {
		t.Error("PasswordResetToken should be nil after setting a password")
	}
	if got.PasswordResetTokenExpiration != nil {
		t.Error("PasswordResetTokenExpiration should be nil after setting a password")
	}
	ok, err := got.VerifyPassword(testCredential)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !ok {
		t.Error("設定しなおしたパスワードでログインできない")
	}
}
