package reps

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"database/sql"
	sqllib "database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type timeIsTempRepositorySQLite3Impl timeIsRepositorySQLite3Impl

func NewTimeIsTempRepositorySQLite3Impl(ctx context.Context, db *sql.DB, m *sync.Mutex) (TimeIsTempRepository, error) {
	filename := "time_is_temp"
	sql := `
CREATE TABLE IF NOT EXISTS "TIMEIS" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TITLE NOT NULL,
  START_TIME NOT NULL,
  END_TIME,
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
		err = fmt.Errorf("error at create TIMEIS table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_TIMEIS ON TIMEIS (ID, UPDATE_TIME);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS index to %s: %w", filename, err)
		return nil, err
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create TIMEIS table to %s: %w", filename, err)
		return nil, err
	}

	return &timeIsTempRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        m,
	}, nil
}
func (t *timeIsTempRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	impl := timeIsRepositorySQLite3Impl(*t)
	return impl.FindKyous(ctx, query)
}

func (t *timeIsTempRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	impl := timeIsRepositorySQLite3Impl(*t)
	return impl.GetKyou(ctx, id, updateTime)
}

func (t *timeIsTempRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	impl := timeIsRepositorySQLite3Impl(*t)
	return impl.GetKyouHistories(ctx, id)
}

func (t *timeIsTempRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("GetPath is not implemented for timeis temp repository")
}

func (t *timeIsTempRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	impl := timeIsRepositorySQLite3Impl(*t)
	return impl.UpdateCache(ctx)
}

func (t *timeIsTempRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "timeis_temp", nil
}

func (t *timeIsTempRepositorySQLite3Impl) Close(ctx context.Context) error {
	impl := timeIsRepositorySQLite3Impl(*t)
	return impl.Close(ctx)
}

func (t *timeIsTempRepositorySQLite3Impl) FindTimeIs(ctx context.Context, query *find.FindQuery) ([]*TimeIs, error) {
	impl := timeIsRepositorySQLite3Impl(*t)
	return impl.FindTimeIs(ctx, query)
}

func (t *timeIsTempRepositorySQLite3Impl) GetTimeIs(ctx context.Context, id string, updateTime *time.Time) (*TimeIs, error) {
	impl := timeIsRepositorySQLite3Impl(*t)
	return impl.GetTimeIs(ctx, id, updateTime)
}

func (t *timeIsTempRepositorySQLite3Impl) GetTimeIsHistories(ctx context.Context, id string) ([]*TimeIs, error) {
	impl := timeIsRepositorySQLite3Impl(*t)
	return impl.GetTimeIsHistories(ctx, id)
}

func (t *timeIsTempRepositorySQLite3Impl) AddTimeIsInfo(ctx context.Context, timeis *TimeIs, txID string, userID string, device string) error {
	t.m.Lock()
	defer t.m.Unlock()
	sql := `
INSERT INTO TIMEIS (
  IS_DELETED,
  ID,
  TITLE,
  START_TIME,
  END_TIME,
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
  ?
)`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add timeis sql %s: %w", timeis.ID, err)
		return err
	}
	defer stmt.Close()

	var endTimeStr interface{}
	if timeis.EndTime == nil {
		endTimeStr = nil
	} else {
		endTimeStr = timeis.EndTime.Format(sqlite3impl.TimeLayout)
	}

	queryArgs := []interface{}{
		timeis.IsDeleted,
		timeis.ID,
		timeis.Title,
		timeis.StartTime.Format(sqlite3impl.TimeLayout),
		endTimeStr,
		timeis.CreateTime.Format(sqlite3impl.TimeLayout),
		timeis.CreateApp,
		timeis.CreateDevice,
		timeis.CreateUser,
		timeis.UpdateTime.Format(sqlite3impl.TimeLayout),
		timeis.UpdateApp,
		timeis.UpdateDevice,
		timeis.UpdateUser,
		userID,
		device,
		txID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to timeis %s: %w", timeis.ID, err)
		return err
	}
	return nil
}

func (t *timeIsTempRepositorySQLite3Impl) GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error) {
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
FROM TIMEIS
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at timeis temp: %w", err)
		return nil, err
	}

	dataType := "timeis"
	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyous by TXID sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from timeis temp: %w", err)
		return nil, err
	}
	defer rows.Close()

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
				err = fmt.Errorf("error at scan from timeis temp: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in timeis temp: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in timeis temp: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in timeis temp: %w", updateTimeStr, err)
				return nil, err
			}

			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (t *timeIsTempRepositorySQLite3Impl) GetTimeIssByTXID(ctx context.Context, txID string, userID string, device string) ([]*TimeIs, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TITLE,
  START_TIME,
  END_TIME,
  CREATE_TIME AS RELATED_TIME,
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
FROM TIMEIS 
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at TIMEIS: %w", err)
		return nil, err
	}

	dataType := "timeis"

	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get time is by tx id sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from TIMEIS: %w", err)
		return nil, err
	}
	defer rows.Close()

	timeiss := []*TimeIs{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			timeis := &TimeIs{}
			timeis.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			startTimeStr, endTime := "", sqllib.NullString{}

			err = rows.Scan(
				&timeis.IsDeleted,
				&timeis.ID,
				&timeis.Title,
				&startTimeStr,
				&endTime,
				&relatedTimeStr,
				&createTimeStr,
				&timeis.CreateApp,
				&timeis.CreateDevice,
				&timeis.CreateUser,
				&updateTimeStr,
				&timeis.UpdateApp,
				&timeis.UpdateDevice,
				&timeis.UpdateUser,
				&timeis.RepName,
				&timeis.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan timeis: %w", err)
				return nil, err
			}

			timeis.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TIMEIS: %w", createTimeStr, err)
				return nil, err
			}
			timeis.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TIMEIS: %w", updateTimeStr, err)
				return nil, err
			}
			timeis.StartTime, err = time.Parse(sqlite3impl.TimeLayout, startTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse start time %s in TIMEIS: %w", updateTimeStr, err)
				return nil, err
			}
			if endTime.Valid {
				parsedEndTime, _ := time.Parse(sqlite3impl.TimeLayout, endTime.String)
				timeis.EndTime = &parsedEndTime
			}
			timeiss = append(timeiss, timeis)
		}
	}
	return timeiss, nil
}

func (t *timeIsTempRepositorySQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	sql := `
DELETE FROM TIMEIS
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete temp timeis kyou by TXID sql: %w", err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		txID,
		userID,
		device,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at delete temp timeis kyou by TXID sql: %w", err)
		return err
	}
	return nil
}

func (t *timeIsTempRepositorySQLite3Impl) UnWrapTyped() ([]TimeIsTempRepository, error) {
	return []TimeIsTempRepository{t}, nil
}

func (t *timeIsTempRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{t}, nil
}
