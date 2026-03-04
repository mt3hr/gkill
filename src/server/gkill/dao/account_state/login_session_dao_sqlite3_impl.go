package account_state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

const CURRENT_SCHEMA_VERSION_LOGIN_SESSION_DAO = "1.0.0"

type loginSessionDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.RWMutex
}

func NewLoginSessionDAOSQLite3Impl(ctx context.Context, filename string) (LoginSessionDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=1&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	if isOld, oldVerDAO, err := checkAndResolveDataSchemaLoginSessionDAO(ctx, db); err != nil {
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

	sql := `
CREATE TABLE IF NOT EXISTS "LOGIN_SESSION" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  APPLICATION_NAME NOT NULL,
  SESSION_ID NOT NULL,
  CLIENT_IP_ADDRESS NOT NULL,
  LOGIN_TIME NOT NULL,
  EXPIRATION_TIME NOT NULL,
  IS_LOCAL_APP_USER NOT NULL
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create LOGIN_SESSION table statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create LOGIN_SESSION table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_LOGIN_SESSION ON LOGIN_SESSION (SESSION_ID);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create LOGIN_SESSION index statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := indexStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create LOGIN_SESSION index to %s: %w", filename, err)
		return nil, err
	}

	if gkill_options.Optimize {
		err = sqlite3impl.Optimize(db)
		if err != nil {
			err = fmt.Errorf("error at optimize db %w", err)
			return nil, err
		}
	}

	return &loginSessionDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.RWMutex{},
	}, nil
}

func (l *loginSessionDAOSQLite3Impl) GetAllLoginSessions(ctx context.Context) ([]*LoginSession, error) {
	l.m.RLock()
	defer l.m.RUnlock()
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  APPLICATION_NAME,
  SESSION_ID,
  CLIENT_IP_ADDRESS,
  LOGIN_TIME,
  EXPIRATION_TIME,
  IS_LOCAL_APP_USER
FROM LOGIN_SESSION
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all login sessions sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
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

	loginSessions := []*LoginSession{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			loginSession := &LoginSession{}
			loginTimeStr := ""
			expriationTimeStr := ""
			err = rows.Scan(
				&loginSession.ID,
				&loginSession.UserID,
				&loginSession.Device,
				&loginSession.ApplicationName,
				&loginSession.SessionID,
				&loginSession.ClientIPAddress,
				&loginTimeStr,
				&expriationTimeStr,
				&loginSession.IsLocalAppUser,
			)
			if err != nil {
				err = fmt.Errorf("error at scan login session %s: %w", loginSession.ID, err)
				return nil, err
			}

			loginSession.LoginTime, err = time.Parse(sqlite3impl.TimeLayout, loginTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file upload time %s at %s in LOGIN_SESSION: %w", loginTimeStr, loginSession.ID, err)
				return nil, err
			}

			loginSession.ExpirationTime, err = time.Parse(sqlite3impl.TimeLayout, expriationTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file upload time %s at %s in LOGIN_SESSION: %w", expriationTimeStr, loginSession.ID, err)
				return nil, err
			}

			loginSessions = append(loginSessions, loginSession)
		}
	}
	return loginSessions, nil
}

func (l *loginSessionDAOSQLite3Impl) GetLoginSessions(ctx context.Context, userID string, device string) ([]*LoginSession, error) {
	l.m.RLock()
	defer l.m.RUnlock()
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  APPLICATION_NAME,
  SESSION_ID,
  CLIENT_IP_ADDRESS,
  LOGIN_TIME,
  EXPIRATION_TIME,
  IS_LOCAL_APP_USER
FROM LOGIN_SESSION
WHERE USER_ID = ? AND DEVICE = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get login sessions sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		userID,
		device,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
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

	loginSessions := []*LoginSession{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			loginSession := &LoginSession{}
			loginTimeStr := ""
			expriationTimeStr := ""
			err = rows.Scan(
				&loginSession.ID,
				&loginSession.UserID,
				&loginSession.Device,
				&loginSession.ApplicationName,
				&loginSession.SessionID,
				&loginSession.ClientIPAddress,
				&loginTimeStr,
				&expriationTimeStr,
				&loginSession.IsLocalAppUser,
			)
			if err != nil {
				err = fmt.Errorf("error at scan login session %s: %w", userID, err)
				return nil, err
			}

			loginSession.LoginTime, err = time.Parse(sqlite3impl.TimeLayout, loginTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file upload time %s at %s in LOGIN_SESSION: %w", loginTimeStr, loginSession.ID, err)
				return nil, err
			}

			loginSession.ExpirationTime, err = time.Parse(sqlite3impl.TimeLayout, expriationTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file upload time %s at %s in LOGIN_SESSION: %w", expriationTimeStr, loginSession.ID, err)
				return nil, err
			}

			loginSessions = append(loginSessions, loginSession)
		}
	}
	return loginSessions, nil
}

func (l *loginSessionDAOSQLite3Impl) GetLoginSession(ctx context.Context, sessionID string) (*LoginSession, error) {
	l.m.RLock()
	defer l.m.RUnlock()
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  APPLICATION_NAME,
  SESSION_ID,
  CLIENT_IP_ADDRESS,
  LOGIN_TIME,
  EXPIRATION_TIME,
  IS_LOCAL_APP_USER
FROM LOGIN_SESSION
WHERE SESSION_ID = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get login sessions sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		sessionID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
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

	loginSessions := []*LoginSession{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			loginSession := &LoginSession{}
			loginTimeStr := ""
			expriationTimeStr := ""
			err = rows.Scan(
				&loginSession.ID,
				&loginSession.UserID,
				&loginSession.Device,
				&loginSession.ApplicationName,
				&loginSession.SessionID,
				&loginSession.ClientIPAddress,
				&loginTimeStr,
				&expriationTimeStr,
				&loginSession.IsLocalAppUser,
			)
			if err != nil {
				err = fmt.Errorf("error at scan login session %s: %w", sessionID, err)
				return nil, err
			}

			loginSession.LoginTime, err = time.Parse(sqlite3impl.TimeLayout, loginTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file upload time %s at %s in LOGIN_SESSION: %w", loginTimeStr, loginSession.ID, err)
				return nil, err
			}

			loginSession.ExpirationTime, err = time.Parse(sqlite3impl.TimeLayout, expriationTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file upload time %s at %s in LOGIN_SESSION: %w", expriationTimeStr, loginSession.ID, err)
				return nil, err
			}

			loginSessions = append(loginSessions, loginSession)
		}
	}
	if len(loginSessions) == 0 {
		return nil, nil
	}
	return loginSessions[0], nil
}

func (l *loginSessionDAOSQLite3Impl) AddLoginSession(ctx context.Context, loginSession *LoginSession) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	sql := `
INSERT INTO LOGIN_SESSION (
  ID,
  USER_ID,
  DEVICE,
  APPLICATION_NAME,
  SESSION_ID,
  CLIENT_IP_ADDRESS,
  LOGIN_TIME,
  EXPIRATION_TIME,
  IS_LOCAL_APP_USER
) VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update login sessions sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		loginSession.ID,
		loginSession.UserID,
		loginSession.Device,
		loginSession.ApplicationName,
		loginSession.SessionID,
		loginSession.ClientIPAddress,
		loginSession.LoginTime.Format(sqlite3impl.TimeLayout),
		loginSession.ExpirationTime.Format(sqlite3impl.TimeLayout),
		loginSession.IsLocalAppUser,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *loginSessionDAOSQLite3Impl) UpdateLoginSession(ctx context.Context, loginSession *LoginSession) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	sql := `
UPDATE LOGIN_SESSION SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  APPLICATION_NAME = ?,
  SESSION_ID = ?,
  CLIENT_IP_ADDRESS = ?,
  LOGIN_TIME = ?,
  EXPIRATION_TIME = ?,
  IS_LOCAL_APP_USER = ?
WHERE ID = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add login sessions sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		loginSession.ID,
		loginSession.UserID,
		loginSession.Device,
		loginSession.ApplicationName,
		loginSession.SessionID,
		loginSession.ClientIPAddress,
		loginSession.LoginTime.Format(sqlite3impl.TimeLayout),
		loginSession.ExpirationTime.Format(sqlite3impl.TimeLayout),
		loginSession.IsLocalAppUser,
		loginSession.ID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *loginSessionDAOSQLite3Impl) DeleteLoginSession(ctx context.Context, sessionID string) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	sql := `
DELETE FROM LOGIN_SESSION
WHERE SESSION_ID = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete login session sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		sessionID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *loginSessionDAOSQLite3Impl) Close(ctx context.Context) error {
	l.m.Lock()
	defer l.m.Unlock()
	return l.db.Close()
}

func checkAndResolveDataSchemaLoginSessionDAO(ctx context.Context, db *sql.DB) (isOld bool, oldVerDAO LoginSessionDAO, err error) {
	schemaVersionKey := "SCHEMA_VERSION_LOGIN_SESSION"
	currentSchemaVersion := CURRENT_SCHEMA_VERSION_LOGIN_SESSION_DAO

	// テーブルとインデックスがなければ作る
	createTableSQL := `
CREATE TABLE IF NOT EXISTS GKILL_META_INFO (
  KEY NOT NULL,
  VALUE,
  PRIMARY KEY(KEY)
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", createTableSQL)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", createTableSQL)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info table: %w", err)
		return false, nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_GKILL_META_INFO ON GKILL_META_INFO (KEY);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
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

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info index: %w", err)
		return false, nil, err
	}

	// スキーマのージョンを取得する
	selectSchemaVersionSQL := `
SELECT 
  VALUE
FROM GKILL_META_INFO
WHERE KEY = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", selectSchemaVersionSQL)
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
	queryArgs := []interface{}{schemaVersionKey}
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", selectSchemaVersionSQL, "query", queryArgs)
	err = selectSchemaVersionStmt.QueryRowContext(ctx, queryArgs...).Scan(&dbSchemaVersion)
	if err != nil {
		// データがなかったら今のバージョンをいれる
		if errors.Is(err, sql.ErrNoRows) {
			insertCurrentVersionSQL := `
INSERT INTO GKILL_META_INFO(KEY, VALUE)
VALUES(?, ?)`
			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", insertCurrentVersionSQL)
			insertCurrentVersionStmt, err := db.PrepareContext(ctx, insertCurrentVersionSQL)
			if err != nil {
				err = fmt.Errorf("error at get schema version sql: %w", err)
				err = fmt.Errorf("error at insert schema version sql: %w", err)
				return false, nil, err
			}
			defer func() {
				err := insertCurrentVersionStmt.Close()
				if err != nil {
					slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
				}
			}()
			queryArgs := []interface{}{schemaVersionKey, currentSchemaVersion}
			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", insertCurrentVersionSQL, queryArgs)
			_, err = insertCurrentVersionStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at get schema version sql: %w", err)
				err = fmt.Errorf("error at query :%w", err)
				return false, nil, err
			}

			queryArgs = []interface{}{schemaVersionKey}
			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", selectSchemaVersionSQL, "query", queryArgs)
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
			// 過去のDAOを作って返す or 最新のDAOに変換して返す
		}
		err = fmt.Errorf("invalid db schema version %s", dbSchemaVersion)
		return true, nil, err
	}
	// ここまで 過去バージョンのスキーマだった場合の対応

	return false, nil, nil
}
