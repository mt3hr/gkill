// ˅
package account_state

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

// ˄

type loginSessionDAOSQLite3Impl struct {
	// ˅
	filename string
	db       *sql.DB
	m        *sync.Mutex
	// ˄
}

// ˅
func NewLoginSessionDAOSQLite3Impl(ctx context.Context, filename string) (LoginSessionDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
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
  IS_LOCAL_APP_USER NOT NULL,
);`
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create LOGIN_SESSION table statement %s: %w", filename, err)
		return nil, err
	}

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create LOGIN_SESSION table to %s: %w", filename, err)
		return nil, err
	}

	return &loginSessionDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}

func (l *loginSessionDAOSQLite3Impl) GetAllLoginSessions(ctx context.Context) ([]*LoginSession, error) {
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
  IS_LOCAL_APP_USER,
FROM LOGIN_SESSION
`
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all login sessions sql: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}

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
				loginSession.ID,
				loginSession.UserID,
				loginSession.Device,
				loginSession.ApplicationName,
				loginSession.SessionID,
				loginSession.ClientIPAddress,
				&loginTimeStr,
				&expriationTimeStr,
				loginSession.IsLocalAppUser,
			)

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
  IS_LOCAL_APP_USER,
FROM LOGIN_SESSION
WHERE USER_ID = ? AND DEVICE = ?
`
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get login sessions sql: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, userID, device)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}

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
				loginSession.ID,
				loginSession.UserID,
				loginSession.Device,
				loginSession.ApplicationName,
				loginSession.SessionID,
				loginSession.ClientIPAddress,
				&loginTimeStr,
				&expriationTimeStr,
				loginSession.IsLocalAppUser,
			)

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

func (l *loginSessionDAOSQLite3Impl) AddLoginSession(ctx context.Context, loginSession *LoginSession) (bool, error) {
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
  IS_LOCAL_APP_USER,
VALUES (
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
WHERE ID = ?
`
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update login sessions sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
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
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *loginSessionDAOSQLite3Impl) UpdateLoginSession(ctx context.Context, loginSession *LoginSession) (bool, error) {
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
  IS_LOCAL_APP_USER = ?,
WHERE ID = ?
`
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add login sessions sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
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
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *loginSessionDAOSQLite3Impl) DeleteLoginSession(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE LOGIN_SESSION
WHERE ID = ?
`
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete login session sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

// ˄
