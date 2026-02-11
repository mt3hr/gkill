package reps

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type urlogTempRepositorySQLite3Impl urlogRepositorySQLite3Impl

func NewURLogTempRepositorySQLite3Impl(ctx context.Context, db *sql.DB, m *sync.RWMutex) (URLogTempRepository, error) {
	filename := "urlog_temp"
	sql := `
CREATE TABLE IF NOT EXISTS "URLOG" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  URL NOT NULL,
  TITLE NOT NULL,
  DESCRIPTION NOT NULL,
  FAVICON_IMAGE NOT NULL,
  THUMBNAIL_IMAGE NOT NULL,
  RELATED_TIME NOT NULL,
  CREATE_TIME NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_TIME NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  TX_ID NOT NULL
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table statement %s: %w", filename, err)
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
		err = fmt.Errorf("error at create URLOG table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_URLOG ON URLOG (ID, RELATED_TIME, UPDATE_TIME);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create URLOG index statement %s: %w", filename, err)
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
		err = fmt.Errorf("error at create URLOG index to %s: %w", filename, err)
		return nil, err
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create URLOG table to %s: %w", filename, err)
		return nil, err
	}

	return &urlogTempRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        m,
	}, nil
}

func (u *urlogTempRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	impl := urlogRepositorySQLite3Impl(*u)
	return impl.FindKyous(ctx, query)
}

func (u *urlogTempRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	impl := urlogRepositorySQLite3Impl(*u)
	return impl.GetKyou(ctx, id, updateTime)
}

func (u *urlogTempRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	impl := urlogRepositorySQLite3Impl(*u)
	return impl.GetKyouHistories(ctx, id)
}

func (u *urlogTempRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("GetPath is not implemented for urlog temp repository")
}

func (u *urlogTempRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	u.m.Lock()
	defer u.m.Unlock()
	impl := urlogRepositorySQLite3Impl(*u)
	return impl.UpdateCache(ctx)
}

func (u *urlogTempRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "urlog_temp", nil
}

func (u *urlogTempRepositorySQLite3Impl) Close(ctx context.Context) error {
	u.m.Lock()
	defer u.m.Unlock()
	impl := urlogRepositorySQLite3Impl(*u)
	return impl.Close(ctx)
}

func (u *urlogTempRepositorySQLite3Impl) FindURLog(ctx context.Context, query *find.FindQuery) ([]*URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	impl := urlogRepositorySQLite3Impl(*u)
	return impl.FindURLog(ctx, query)
}

func (u *urlogTempRepositorySQLite3Impl) GetURLog(ctx context.Context, id string, updateTime *time.Time) (*URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	impl := urlogRepositorySQLite3Impl(*u)
	return impl.GetURLog(ctx, id, updateTime)
}

func (u *urlogTempRepositorySQLite3Impl) GetURLogHistories(ctx context.Context, id string) ([]*URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	impl := urlogRepositorySQLite3Impl(*u)
	return impl.GetURLogHistories(ctx, id)
}

func (u *urlogTempRepositorySQLite3Impl) AddURLogInfo(ctx context.Context, urlog *URLog, txID string, userID string, device string) error {
	u.m.Lock()
	defer u.m.Unlock()
	sql := `
INSERT INTO URLOG (
  IS_DELETED,
  ID,
  URL,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
  THUMBNAIL_IMAGE,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  USER_ID,
  DEVICE,
  TX_ID
) VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := u.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add urlog sql %s: %w", urlog.ID, err)
		return err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		urlog.IsDeleted,
		urlog.ID,
		urlog.URL,
		urlog.Title,
		urlog.Description,
		urlog.FaviconImage,
		urlog.ThumbnailImage,
		urlog.RelatedTime.Format(sqlite3impl.TimeLayout),
		urlog.CreateTime.Format(sqlite3impl.TimeLayout),
		urlog.CreateApp,
		urlog.CreateDevice,
		urlog.CreateUser,
		urlog.UpdateTime.Format(sqlite3impl.TimeLayout),
		urlog.UpdateApp,
		urlog.UpdateDevice,
		urlog.UpdateUser,
		userID,
		device,
		txID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to URLog %s: %w", urlog.ID, err)
		return err
	}
	return nil
}

func (u *urlogTempRepositorySQLite3Impl) GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	var err error
	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM URLOG
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`

	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at urlog temp: %w", err)
		return nil, err
	}

	dataType := "urlog"
	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := u.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyous by TXID sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from urlog temp: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	kyous := []*Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := &Kyou{}
			kyou.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			targetRepName := ""

			err = rows.Scan(
				&kyou.IsDeleted,
				&kyou.ID,
				&targetRepName,
				&relatedTimeStr,
				&createTimeStr,
				&kyou.CreateApp,
				&kyou.CreateDevice,
				&kyou.CreateUser,
				&updateTimeStr,
				&kyou.UpdateApp,
				&kyou.UpdateDevice,
				&kyou.UpdateUser,
				&kyou.RepName,
				&kyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from urlog temp: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in urlog temp: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in urlog temp: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in urlog temp: %w", updateTimeStr, err)
				return nil, err
			}

			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (u *urlogTempRepositorySQLite3Impl) GetURLogsByTXID(ctx context.Context, txID string, userID string, device string) ([]*URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at URLOG: %w", err)
		return nil, err
	}

	sql := `
SELECT 
  IS_DELETED,
  ID,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  URL,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
  THUMBNAIL_IMAGE,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM URLOG
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`

	dataType := "urlog"

	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := u.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get urlog by tx id sql %s: %w", txID, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query ")
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	urlogs := []*URLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			urlog := &URLog{}
			urlog.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(
				&urlog.IsDeleted,
				&urlog.ID,
				&relatedTimeStr,
				&createTimeStr,
				&urlog.CreateApp,
				&urlog.CreateDevice,
				&urlog.CreateUser,
				&updateTimeStr,
				&urlog.UpdateApp,
				&urlog.UpdateDevice,
				&urlog.UpdateUser,
				&urlog.URL,
				&urlog.Title,
				&urlog.Description,
				&urlog.FaviconImage,
				&urlog.ThumbnailImage,
				&urlog.RepName,
				&urlog.DataType,
			)

			if err != nil {
				err = fmt.Errorf("error at scan from URLOG %s: %w", txID, err)
				return nil, err
			}

			urlog.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in URLog: %w", relatedTimeStr, txID, err)
				return nil, err
			}
			urlog.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in URLog: %w", createTimeStr, txID, err)
				return nil, err
			}
			urlog.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in URLog: %w", updateTimeStr, txID, err)
				return nil, err
			}
			urlogs = append(urlogs, urlog)
		}
	}
	return urlogs, nil
}

func (u *urlogTempRepositorySQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	u.m.Lock()
	defer u.m.Unlock()
	sql := `
DELETE FROM URLOG
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := u.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete temp urlog kyou by TXID sql: %w", err)
		return err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		txID,
		userID,
		device,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at delete temp urlog kyou by TXID sql: %w", err)
		return err
	}
	return nil
}

func (u *urlogTempRepositorySQLite3Impl) UnWrapTyped() ([]URLogTempRepository, error) {
	return []URLogTempRepository{u}, nil
}

func (u *urlogTempRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{u}, nil
}
