package gkill_notification

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type gkillNotificateTargetDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewGkillNotificateTargetDAOSQLite3Impl(ctx context.Context, filename string) (GkillNotificateTargetDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS "NOTIFICATION" (
  ID NOT NULL,
  USER_ID NOT NULL,
  PUBLIC_KEY NOT NULL,
  SUBSCRIPTION NOT NULL
)`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table to %s: %w", filename, err)
		return nil, err
	}

	return &gkillNotificateTargetDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}

func (m *gkillNotificateTargetDAOSQLite3Impl) GetAllGkillNotificationTargets(ctx context.Context) ([]*GkillNotificateTarget, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  PUBLIC_KEY,
  SUBSCRIPTION
FROM NOTIFICATION
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all mi notificate target sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	gkillNotificateTargets := []*GkillNotificateTarget{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			gkillNotificateTarget := &GkillNotificateTarget{}
			err = rows.Scan(
				&gkillNotificateTarget.ID,
				&gkillNotificateTarget.UserID,
				&gkillNotificateTarget.PublicKey,
				&gkillNotificateTarget.Subscription,
			)
			gkillNotificateTargets = append(gkillNotificateTargets, gkillNotificateTarget)
		}
	}
	return gkillNotificateTargets, nil
}

func (m *gkillNotificateTargetDAOSQLite3Impl) GetGkillNotificationTargets(ctx context.Context, userID string, publicKey string) ([]*GkillNotificateTarget, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  PUBLIC_KEY,
  SUBSCRIPTION
FROM NOTIFICATION
WHERE USER_ID = ? AND PUBLIC_KEY = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get mi notificate target sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
		publicKey,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	gkillNotificateTargets := []*GkillNotificateTarget{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			gkillNotificateTarget := &GkillNotificateTarget{}
			err = rows.Scan(
				&gkillNotificateTarget.ID,
				&gkillNotificateTarget.UserID,
				&gkillNotificateTarget.PublicKey,
				&gkillNotificateTarget.Subscription,
			)
			gkillNotificateTargets = append(gkillNotificateTargets, gkillNotificateTarget)
		}
	}
	return gkillNotificateTargets, nil
}

func (m *gkillNotificateTargetDAOSQLite3Impl) AddGkillNotificationTarget(ctx context.Context, gkillNotificateTarget *GkillNotificateTarget) (bool, error) {
	sql := `
INSERT INTO NOTIFICATION(
  ID,
  USER_ID,
  PUBLIC_KEY,
  SUBSCRIPTION
)
VALUES (
  ?,
  ?,
  ?,
  ?
)
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add mi notificate target sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		gkillNotificateTarget.ID,
		gkillNotificateTarget.UserID,
		gkillNotificateTarget.PublicKey,
		gkillNotificateTarget.Subscription,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (m *gkillNotificateTargetDAOSQLite3Impl) UpdateGkillNotificationTarget(ctx context.Context, gkillNotificateTarget *GkillNotificateTarget) (bool, error) {
	sql := `
UPDATE NOTIFICATION
  ID = ?,
  USER_ID = ?,
  PUBLIC_KEY = ?,
  SUBSCRIPTION = ?
WHERE ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update mi notificate target sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		gkillNotificateTarget.ID,
		gkillNotificateTarget.UserID,
		gkillNotificateTarget.PublicKey,
		gkillNotificateTarget.Subscription,
		gkillNotificateTarget.ID,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (m *gkillNotificateTargetDAOSQLite3Impl) DeleteGkillNotificationTarget(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE FROM NOTIFICATION
WHERE ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete mi notification target sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		id,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (m *gkillNotificateTargetDAOSQLite3Impl) Close(ctx context.Context) error {
	return m.db.Close()
}
