package account

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type accountDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewAccountDAOSQLite3Impl(ctx context.Context, filename string) (AccountDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=1&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "ACCOUNT" (
  USER_ID PRIMARY KEY NOT NULL,
  PASSWORD_SHA256,
  IS_ADMIN NOT NULL,
  IS_ENABLE NOT NULL,
  PASSWORD_RESET_TOKEN
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create ACCOUNT table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create ACCOUNT table to %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_ACCOUNT ON ACCOUNT (USER_ID);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create ACCOUNT index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create ACCOUNT index to %s: %w", filename, err)
		return nil, err
	}

	accountDAO := &accountDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}
	return accountDAO, nil
}
func (a *accountDAOSQLite3Impl) GetAllAccounts(ctx context.Context) ([]*Account, error) {
	sql := `
SELECT 
  USER_ID,
  PASSWORD_SHA256,
  IS_ADMIN,
  IS_ENABLE,
  PASSWORD_RESET_TOKEN
FROM ACCOUNT
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all accounts sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	accounts := []*Account{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			account := &Account{}
			err = rows.Scan(
				&account.UserID,
				&account.PasswordSha256,
				&account.IsAdmin,
				&account.IsEnable,
				&account.PasswordResetToken,
			)
			if err != nil {
				err = fmt.Errorf("error at scan account: %w", err)
				return nil, err
			}
			accounts = append(accounts, account)
		}
	}
	return accounts, nil
}
func (a *accountDAOSQLite3Impl) GetAccount(ctx context.Context, userID string) (*Account, error) {
	sql := `
SELECT 
  USER_ID,
  PASSWORD_SHA256,
  IS_ADMIN,
  IS_ENABLE,
  PASSWORD_RESET_TOKEN
FROM ACCOUNT
WHERE USER_ID = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get account sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	accounts := []*Account{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			account := &Account{}
			err = rows.Scan(
				&account.UserID,
				&account.PasswordSha256,
				&account.IsAdmin,
				&account.IsEnable,
				&account.PasswordResetToken,
			)
			accounts = append(accounts, account)
		}
	}
	if len(accounts) == 0 {
		return nil, nil
	} else if len(accounts) == 1 {
		return accounts[0], nil
	}
	return nil, fmt.Errorf("複数のアカウントが見つかりました。%s: %w", userID, err)
}
func (a *accountDAOSQLite3Impl) AddAccount(ctx context.Context, account *Account) (bool, error) {
	sql := `
INSERT INTO ACCOUNT (
  USER_ID,
  PASSWORD_SHA256,
  IS_ADMIN,
  IS_ENABLE,
  PASSWORD_RESET_TOKEN
)
VALUES (
  ?,
  ?,
  ?,
  ?,
  ?
)
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add account sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		account.UserID,
		account.PasswordSha256,
		account.IsAdmin,
		account.IsEnable,
		account.PasswordResetToken,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}
func (a *accountDAOSQLite3Impl) UpdateAccount(ctx context.Context, account *Account) (bool, error) {
	sql := `
UPDATE ACCOUNT SET
  USER_ID = ?,
  PASSWORD_SHA256 = ?,
  IS_ADMIN = ?,
  IS_ENABLE = ?,
  PASSWORD_RESET_TOKEN = ?
WHERE USER_ID = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update account sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		account.UserID,
		account.PasswordSha256,
		account.IsAdmin,
		account.IsEnable,
		account.PasswordResetToken,
		account.UserID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}
func (a *accountDAOSQLite3Impl) DeleteAccount(ctx context.Context, userID string) (bool, error) {
	sql := `
DELETE FROM ACCOUNT
WHERE USER_ID = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete account sql: %w", err)
		return false, err
	}
	defer stmt.Close()
	queryArgs := []interface{}{
		userID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (a *accountDAOSQLite3Impl) Close(ctx context.Context) error {
	return a.db.Close()
}
