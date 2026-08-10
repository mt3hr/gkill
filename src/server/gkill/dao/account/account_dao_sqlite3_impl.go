package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// 1.1.0でPASSWORD_SHA256をPASSWORD_HASH (Argon2idのPHC文字列) に置き換え、
// PASSWORD_RESET_TOKEN_EXPIRATIONを追加した。
const CURRENT_SCHEMA_VERSION_ACCOUNT_DAO = "1.1.0"

// formatPasswordResetTokenExpiration はリセットトークンの期限をDBに入れる形に変換する。
// 未設定のときはNULLを入れる。
func formatPasswordResetTokenExpiration(expiration *time.Time) any {
	if expiration == nil {
		return nil
	}
	return expiration.Format(sqlite3impl.TimeLayout)
}

type accountDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.RWMutex
}

func NewAccountDAOSQLite3Impl(ctx context.Context, filename string) (AccountDAO, error) {
	var err error
	db, err := sql.Open("sqlite", "file:"+filename+"?_pragma=busy_timeout(6000)&_pragma=synchronous(NORMAL)&_pragma=journal_mode(DELETE)")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	if isOld, oldVerDAO, err := checkAndResolveDataSchemaAccountDAO(ctx, db); err != nil {
		return nil, err
	} else if isOld {
		if oldVerDAO != nil {
			return oldVerDAO, nil
		} else {
			err = fmt.Errorf("error at load database schema %s", filename)
			return nil, err
		}
	}

	if gkill_options.Optimize {
		err = sqlite3impl.DeleteAllIndex(db)
		if err != nil {
			err = fmt.Errorf("error at delete all index %w", err)
			return nil, err
		}
	}

	// PASSWORD_HASHにはArgon2idのPHC文字列が入る (password_hash.go を参照)。
	// 旧スキーマのPASSWORD_SHA256は無塩SHA-256をそのまま保持していたが、
	// スキーマ1.1.0への移行でカラム名を変えたうえで全アカウントのパスワードを無効化している。
	sql := `
CREATE TABLE IF NOT EXISTS "ACCOUNT" (
  USER_ID PRIMARY KEY NOT NULL,
  PASSWORD_HASH,
  IS_ADMIN NOT NULL,
  IS_ENABLE NOT NULL,
  PASSWORD_RESET_TOKEN,
  PASSWORD_RESET_TOKEN_EXPIRATION
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create ACCOUNT table statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create ACCOUNT table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_ACCOUNT ON ACCOUNT (USER_ID);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create ACCOUNT index statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := indexStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create ACCOUNT index to %s: %w", filename, err)
		return nil, err
	}

	if gkill_options.Optimize {
		err = sqlite3impl.Optimize(db)
		if err != nil {
			err = fmt.Errorf("error at optimize db %w", err)
			return nil, err
		}
	}

	accountDAO := &accountDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.RWMutex{},
	}
	return accountDAO, nil
}
func (a *accountDAOSQLite3Impl) GetAllAccounts(ctx context.Context) ([]*Account, error) {
	a.m.RLock()
	defer a.m.RUnlock()
	sql := `
SELECT
  USER_ID,
  PASSWORD_HASH,
  IS_ADMIN,
  IS_ENABLE,
  PASSWORD_RESET_TOKEN,
  PASSWORD_RESET_TOKEN_EXPIRATION
FROM ACCOUNT
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all accounts sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	accounts := []*Account{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			account := &Account{}
			var resetTokenExpirationStr *string
			err = rows.Scan(
				&account.UserID,
				&account.PasswordHash,
				&account.IsAdmin,
				&account.IsEnable,
				&account.PasswordResetToken,
				&resetTokenExpirationStr,
			)
			if err != nil {
				err = fmt.Errorf("error at scan account: %w", err)
				return nil, err
			}
			if resetTokenExpirationStr != nil && *resetTokenExpirationStr != "" {
				resetTokenExpiration, err := time.Parse(sqlite3impl.TimeLayout, *resetTokenExpirationStr)
				if err != nil {
					err = fmt.Errorf("error at parse password reset token expiration %s at %s in ACCOUNT: %w", *resetTokenExpirationStr, account.UserID, err)
					return nil, err
				}
				account.PasswordResetTokenExpiration = &resetTokenExpiration
			}
			accounts = append(accounts, account)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return accounts, nil
}
func (a *accountDAOSQLite3Impl) GetAccount(ctx context.Context, userID string) (*Account, error) {
	a.m.RLock()
	defer a.m.RUnlock()
	sql := `
SELECT
  USER_ID,
  PASSWORD_HASH,
  IS_ADMIN,
  IS_ENABLE,
  PASSWORD_RESET_TOKEN,
  PASSWORD_RESET_TOKEN_EXPIRATION
FROM ACCOUNT
WHERE USER_ID = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get account sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []any{
		userID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	accounts := []*Account{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			account := &Account{}
			var resetTokenExpirationStr *string
			err = rows.Scan(
				&account.UserID,
				&account.PasswordHash,
				&account.IsAdmin,
				&account.IsEnable,
				&account.PasswordResetToken,
				&resetTokenExpirationStr,
			)
			if err != nil {
				err = fmt.Errorf("error at scan account: %w", err)
				return nil, err
			}
			if resetTokenExpirationStr != nil && *resetTokenExpirationStr != "" {
				resetTokenExpiration, err := time.Parse(sqlite3impl.TimeLayout, *resetTokenExpirationStr)
				if err != nil {
					err = fmt.Errorf("error at parse password reset token expiration %s at %s in ACCOUNT: %w", *resetTokenExpirationStr, account.UserID, err)
					return nil, err
				}
				account.PasswordResetTokenExpiration = &resetTokenExpiration
			}
			accounts = append(accounts, account)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, nil
	} else if len(accounts) == 1 {
		return accounts[0], nil
	}
	return nil, fmt.Errorf("複数のアカウントが見つかりました。%s", userID)
}
func (a *accountDAOSQLite3Impl) AddAccount(ctx context.Context, account *Account) (bool, error) {
	a.m.Lock()
	defer a.m.Unlock()
	sql := `
INSERT INTO ACCOUNT (
  USER_ID,
  PASSWORD_HASH,
  IS_ADMIN,
  IS_ENABLE,
  PASSWORD_RESET_TOKEN,
  PASSWORD_RESET_TOKEN_EXPIRATION
)
VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add account sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []any{
		account.UserID,
		account.PasswordHash,
		account.IsAdmin,
		account.IsEnable,
		account.PasswordResetToken,
		formatPasswordResetTokenExpiration(account.PasswordResetTokenExpiration),
	}
	// パスワードハッシュ・リセットトークンはログに出さない
	queryArgsForLog := []any{
		account.UserID,
		"***",
		account.IsAdmin,
		account.IsEnable,
		"***",
		formatPasswordResetTokenExpiration(account.PasswordResetTokenExpiration),
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgsForLog)))
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}
func (a *accountDAOSQLite3Impl) UpdateAccount(ctx context.Context, account *Account) (bool, error) {
	a.m.Lock()
	defer a.m.Unlock()
	sql := `
UPDATE ACCOUNT SET
  USER_ID = ?,
  PASSWORD_HASH = ?,
  IS_ADMIN = ?,
  IS_ENABLE = ?,
  PASSWORD_RESET_TOKEN = ?,
  PASSWORD_RESET_TOKEN_EXPIRATION = ?
WHERE USER_ID = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update account sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []any{
		account.UserID,
		account.PasswordHash,
		account.IsAdmin,
		account.IsEnable,
		account.PasswordResetToken,
		formatPasswordResetTokenExpiration(account.PasswordResetTokenExpiration),
		account.UserID,
	}
	// パスワードハッシュ・リセットトークンはログに出さない
	queryArgsForLog := []any{
		account.UserID,
		"***",
		account.IsAdmin,
		account.IsEnable,
		"***",
		formatPasswordResetTokenExpiration(account.PasswordResetTokenExpiration),
		account.UserID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgsForLog)))
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}
func (a *accountDAOSQLite3Impl) DeleteAccount(ctx context.Context, userID string) (bool, error) {
	a.m.Lock()
	defer a.m.Unlock()
	sql := `
DELETE FROM ACCOUNT
WHERE USER_ID = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete account sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	queryArgs := []any{
		userID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (a *accountDAOSQLite3Impl) Close(ctx context.Context) error {
	a.m.Lock()
	defer a.m.Unlock()
	return a.db.Close()
}

func checkAndResolveDataSchemaAccountDAO(ctx context.Context, db *sql.DB) (isOld bool, oldVerDAO AccountDAO, err error) {
	schemaVersionKey := "SCHEMA_VERSION_ACCOUNT"
	currentSchemaVersion := CURRENT_SCHEMA_VERSION_ACCOUNT_DAO

	// テーブルとインデックスがなければ作る
	createTableSQL := `
CREATE TABLE IF NOT EXISTS GKILL_META_INFO (
  KEY NOT NULL,
  VALUE,
  PRIMARY KEY(KEY)
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", createTableSQL))
	stmt, err := db.PrepareContext(ctx, createTableSQL)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info table statement: %w", err)
		return false, nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", createTableSQL))
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info table: %w", err)
		return false, nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_GKILL_META_INFO ON GKILL_META_INFO (KEY);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info index statement: %w", err)
		return false, nil, err
	}
	defer func() {
		err := indexStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info index: %w", err)
		return false, nil, err
	}

	// スキーマのバージョンを取得する
	selectSchemaVersionSQL := `
SELECT 
  VALUE
FROM GKILL_META_INFO
WHERE KEY = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", selectSchemaVersionSQL))
	selectSchemaVersionStmt, err := db.PrepareContext(ctx, selectSchemaVersionSQL)
	if err != nil {
		err = fmt.Errorf("error at get schema version sql: %w", err)
		return false, nil, err
	}
	defer func() {
		err := selectSchemaVersionStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	dbSchemaVersion := ""
	queryArgs := []any{schemaVersionKey}
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", selectSchemaVersionSQL), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	err = selectSchemaVersionStmt.QueryRowContext(ctx, queryArgs...).Scan(&dbSchemaVersion)
	if err != nil {
		// データがなかったら今のバージョンをいれる。
		// ただし GKILL_META_INFO 自体が無かった時代のDBがあるので、
		// 「行が無い＝新規DB」と決めつけずに実際のテーブルの形を見る。
		// 決めつけると、旧スキーマのまま現行版と記録してしまい、
		// 以降 PASSWORD_HASH を参照する全クエリが
		// 「no such column」で落ち続ける（移行も二度と走らない）。
		if errors.Is(err, sql.ErrNoRows) {
			hasOldPasswordColumn, columnErr := columnExists(ctx, db, "ACCOUNT", "PASSWORD_SHA256")
			if columnErr != nil {
				return false, nil, columnErr
			}
			if hasOldPasswordColumn {
				// バージョン行が無いだけの 1.0.0 のDB。移行してから現行版として記録する
				if migrateErr := migrateAccountSchemaFrom100(ctx, db, schemaVersionKey, currentSchemaVersion); migrateErr != nil {
					return true, nil, migrateErr
				}
				return false, nil, nil
			}

			insertCurrentVersionSQL := `
INSERT INTO GKILL_META_INFO(KEY, VALUE)
VALUES(?, ?)`
			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", insertCurrentVersionSQL))
			insertCurrentVersionStmt, err := db.PrepareContext(ctx, insertCurrentVersionSQL)
			if err != nil {
				err = fmt.Errorf("error at prepare insert schema version sql: %w", err)
				return false, nil, err
			}
			defer func() {
				err := insertCurrentVersionStmt.Close()
				if err != nil {
					slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
				}
			}()
			queryArgs := []any{schemaVersionKey, currentSchemaVersion}
			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", insertCurrentVersionSQL), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
			_, err = insertCurrentVersionStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at exec insert schema version sql: %w", err)
				return false, nil, err
			}

			queryArgs = []any{schemaVersionKey}
			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", selectSchemaVersionSQL), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
			err = selectSchemaVersionStmt.QueryRowContext(ctx, queryArgs...).Scan(&dbSchemaVersion)
			if err != nil {
				err = fmt.Errorf("error at get schema version sql: %w", err)
				return false, nil, err
			}
		} else {
			err = fmt.Errorf("error at query :%w", err)
			return false, nil, err
		}
	}

	// ここから 過去バージョンのスキーマだった場合の対応
	if currentSchemaVersion != dbSchemaVersion {
		switch dbSchemaVersion {
		case "1.0.0":
			// 1.0.0はパスワードを無塩SHA-256のまま保持していた。
			// つまりDBの値がそのままログイン資格情報として通用する状態だったので、
			// 中身を作り直すのではなく全アカウントのパスワードを無効化して
			// 設定しなおしてもらう。
			if err := migrateAccountSchemaFrom100(ctx, db, schemaVersionKey, currentSchemaVersion); err != nil {
				return true, nil, err
			}
			return false, nil, nil
		}
		err = fmt.Errorf("invalid db schema version %s", dbSchemaVersion)
		return true, nil, err
	}
	// ここまで 過去バージョンのスキーマだった場合の対応

	return false, nil, nil
}

// migrateAccountSchemaFrom100 はスキーマ1.0.0のACCOUNTテーブルを1.1.0へ移行する。
//
//   - PASSWORD_SHA256をPASSWORD_HASHにリネームする
//   - PASSWORD_RESET_TOKEN_EXPIRATIONを追加する
//   - 全アカウントのパスワードを無効化し、リセットトークンを発行しなおす
//
// 1.0.0の保存値は無塩SHA-256をそのまま持っていて、それ自体がログインに使える
// 資格情報だった。Argon2idで包み直しても「DBを読めた者がログインできる」状態が
// 続いてしまうため、包み直しではなく全員に再設定してもらう。
// 移行後は誰もログインできなくなるので、発行したリセットURLを標準出力に印字する。
func migrateAccountSchemaFrom100(ctx context.Context, db *sql.DB, schemaVersionKey string, currentSchemaVersion string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error at begin tx for account schema migration: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback account schema migration", "error", err)
			}
		}
	}()

	hasPasswordHash, err := columnExistsInTx(ctx, tx, "ACCOUNT", "PASSWORD_HASH")
	if err != nil {
		return err
	}
	if !hasPasswordHash {
		renameSQL := `ALTER TABLE ACCOUNT RENAME COLUMN PASSWORD_SHA256 TO PASSWORD_HASH`
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", renameSQL))
		if _, err := tx.ExecContext(ctx, renameSQL); err != nil {
			return fmt.Errorf("error at rename PASSWORD_SHA256 to PASSWORD_HASH: %w", err)
		}
	}

	hasExpiration, err := columnExistsInTx(ctx, tx, "ACCOUNT", "PASSWORD_RESET_TOKEN_EXPIRATION")
	if err != nil {
		return err
	}
	if !hasExpiration {
		addColumnSQL := `ALTER TABLE ACCOUNT ADD COLUMN PASSWORD_RESET_TOKEN_EXPIRATION`
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", addColumnSQL))
		if _, err := tx.ExecContext(ctx, addColumnSQL); err != nil {
			return fmt.Errorf("error at add PASSWORD_RESET_TOKEN_EXPIRATION column: %w", err)
		}
	}

	selectSQL := `SELECT USER_ID FROM ACCOUNT`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", selectSQL))
	rows, err := tx.QueryContext(ctx, selectSQL)
	if err != nil {
		return fmt.Errorf("error at get user ids for account schema migration: %w", err)
	}
	userIDs := []string{}
	for rows.Next() {
		userID := ""
		if err := rows.Scan(&userID); err != nil {
			rows.Close()
			return fmt.Errorf("error at scan user id for account schema migration: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("error at iterate user ids for account schema migration: %w", err)
	}
	rows.Close()

	updateSQL := `
UPDATE ACCOUNT SET
  PASSWORD_HASH = NULL,
  PASSWORD_RESET_TOKEN = ?,
  PASSWORD_RESET_TOKEN_EXPIRATION = ?
WHERE USER_ID = ?
`
	expiration := time.Now().Add(PasswordResetTokenTTL)
	resetTokens := map[string]string{}
	for _, userID := range userIDs {
		token := sqlite3impl.GenerateNewID()
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", updateSQL), "query", fmt.Sprintf("%q", fmt.Sprint([]any{"***", expiration.Format(sqlite3impl.TimeLayout), userID})))
		if _, err := tx.ExecContext(ctx, updateSQL, token, expiration.Format(sqlite3impl.TimeLayout), userID); err != nil {
			return fmt.Errorf("error at reset password for account schema migration user id = %s: %w", userID, err)
		}
		resetTokens[userID] = token
	}

	// バージョン管理の仕組みが入る前のDBには行そのものが無い。
	// UPDATE だけだと黙って0行更新になり、次の起動でまた移行を試みることになるので
	// 行が無ければ作る
	updateVersionSQL := `
INSERT INTO GKILL_META_INFO(KEY, VALUE)
VALUES(?, ?)
ON CONFLICT(KEY) DO UPDATE SET VALUE = excluded.VALUE`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", updateVersionSQL), "query", fmt.Sprintf("%q", fmt.Sprint([]any{schemaVersionKey, currentSchemaVersion})))
	if _, err := tx.ExecContext(ctx, updateVersionSQL, schemaVersionKey, currentSchemaVersion); err != nil {
		return fmt.Errorf("error at update account schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit account schema migration: %w", err)
	}
	committed = true

	printMigrationNotice(len(userIDs), expiration)
	return nil
}

// printMigrationNotice は移行で何が起きたかを標準出力に知らせる。
//
// 実際のリセットURLはサーバ起動時に printInitialSetupURLs が出す。
// あちらはホストとポートを知っているのでそのまま開けるURLになる。ここでは理由だけ伝える。
func printMigrationNotice(accountCount int, expiration time.Time) {
	if accountCount == 0 {
		return
	}
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("================================================================\n")
	sb.WriteString("gkill: パスワードの保存方式をArgon2idに変更しました。\n")
	sb.WriteString("以前の保存方式ではDBの値がそのままログインに使えてしまうため、\n")
	fmt.Fprintf(&sb, "全アカウント (%d件) のパスワードを無効化しました。\n", accountCount)
	sb.WriteString("このあと表示されるURLからパスワードを設定しなおしてください。\n")
	fmt.Fprintf(&sb, "リセットトークンの有効期限: %s\n", expiration.Format(sqlite3impl.TimeLayout))
	sb.WriteString("================================================================\n")
	os.Stdout.WriteString(sb.String())
}

// columnExists は指定したテーブルに指定した名前のカラムがあるかを返す。
// テーブル自体が無いときは false を返す（PRAGMA は空を返すだけでエラーにならない）。
func columnExists(ctx context.Context, db *sql.DB, tableName string, columnName string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, fmt.Errorf("error at get table info %s: %w", tableName, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	return scanColumnExists(rows, tableName, columnName)
}

// columnExistsInTx は指定したテーブルに指定した名前のカラムがあるかを返す。
func columnExistsInTx(ctx context.Context, tx *sql.Tx, tableName string, columnName string) (bool, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, fmt.Errorf("error at get table info %s: %w", tableName, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	return scanColumnExists(rows, tableName, columnName)
}

// scanColumnExists は PRAGMA table_info の結果から目的のカラムを探す。
func scanColumnExists(rows *sql.Rows, tableName string, columnName string) (bool, error) {
	for rows.Next() {
		var cid int
		var name string
		var columnType any
		var notNull any
		var defaultValue any
		var primaryKey any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, fmt.Errorf("error at scan table info %s: %w", tableName, err)
		}
		if name == columnName {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("error at iterate table info %s: %w", tableName, err)
	}
	return false, nil
}
