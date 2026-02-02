package reps

import (
	"context"
	"fmt"
	"sync"
	"time"

	"database/sql"
	sqllib "database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type miTempRepositorySQLite3Impl miRepositorySQLite3Impl

func NewMiTempRepositorySQLite3Impl(ctx context.Context, db *sql.DB, m *sync.Mutex) (MiTempRepository, error) {
	filename := "mi_temp"

	sql := `
CREATE TABLE IF NOT EXISTS "MI" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TITLE NOT NULL,
  IS_CHECKED NOT NULL,
  BOARD_NAME NOT NULL,
  LIMIT_TIME,
  ESTIMATE_START_TIME,
  ESTIMATE_END_TIME,
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
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create MI table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MI table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_MI ON MI (ID, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create MI index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MI index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create MI table to %s: %w", filename, err)
		return nil, err
	}

	return &miTempRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        m,
	}, nil
}
func (m *miTempRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	impl := miRepositorySQLite3Impl(*m)
	return impl.FindKyous(ctx, query)
}

func (m *miTempRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	impl := miRepositorySQLite3Impl(*m)
	return impl.GetKyou(ctx, id, updateTime)
}

func (m *miTempRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	impl := miRepositorySQLite3Impl(*m)
	return impl.GetKyouHistories(ctx, id)
}

func (m *miTempRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("miTempRepositorySQLite3Impl does not support GetPath")
}

func (m *miTempRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	impl := miRepositorySQLite3Impl(*m)
	return impl.UpdateCache(ctx)
}

func (m *miTempRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "mi_temp", nil
}

func (m *miTempRepositorySQLite3Impl) Close(ctx context.Context) error {
	impl := miRepositorySQLite3Impl(*m)
	return impl.Close(ctx)
}

func (m *miTempRepositorySQLite3Impl) FindMi(ctx context.Context, query *find.FindQuery) ([]*Mi, error) {
	impl := miRepositorySQLite3Impl(*m)
	return impl.FindMi(ctx, query)
}

func (m *miTempRepositorySQLite3Impl) GetMi(ctx context.Context, id string, updateTime *time.Time) (*Mi, error) {
	impl := miRepositorySQLite3Impl(*m)
	return impl.GetMi(ctx, id, updateTime)
}

func (m *miTempRepositorySQLite3Impl) GetMiHistories(ctx context.Context, id string) ([]*Mi, error) {
	impl := miRepositorySQLite3Impl(*m)
	return impl.GetMiHistories(ctx, id)
}

func (m *miTempRepositorySQLite3Impl) AddMiInfo(ctx context.Context, mi *Mi, txID string, userID string, device string) error {
	m.m.Lock()
	defer m.m.Unlock()
	sql := `
INSERT INTO MI (
  IS_DELETED,
  ID,
  TITLE,
  IS_CHECKED,
  BOARD_NAME,
  LIMIT_TIME,
  ESTIMATE_START_TIME,
  ESTIMATE_END_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
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
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add mi sql %s: %w", mi.ID, err)
		return err
	}
	defer stmt.Close()

	var limitTimeStr interface{}
	if mi.LimitTime == nil {
		limitTimeStr = nil
	} else {
		limitTimeStr = mi.LimitTime.Format(sqlite3impl.TimeLayout)
	}
	var startTimeStr interface{}
	if mi.EstimateStartTime == nil {
		startTimeStr = nil
	} else {
		startTimeStr = mi.EstimateStartTime.Format(sqlite3impl.TimeLayout)
	}
	var endTimeStr interface{}
	if mi.EstimateEndTime == nil {
		endTimeStr = nil
	} else {
		endTimeStr = mi.EstimateEndTime.Format(sqlite3impl.TimeLayout)
	}

	queryArgs := []interface{}{
		mi.IsDeleted,
		mi.ID,
		mi.Title,
		mi.IsChecked,
		mi.BoardName,
		limitTimeStr,
		startTimeStr,
		endTimeStr,
		mi.CreateTime.Format(sqlite3impl.TimeLayout),
		mi.CreateApp,
		mi.CreateDevice,
		mi.CreateUser,
		mi.UpdateTime.Format(sqlite3impl.TimeLayout),
		mi.UpdateApp,
		mi.UpdateDevice,
		mi.UpdateUser,
		userID,
		device,
		txID,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to mi %s: %w", mi.ID, err)
		return err
	}
	return nil
}

func (m *miTempRepositorySQLite3Impl) GetBoardNames(ctx context.Context) ([]string, error) {
	impl := miRepositorySQLite3Impl(*m)
	return impl.GetBoardNames(ctx)
}

func (m *miTempRepositorySQLite3Impl) GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error) {
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
FROM MI
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`

	repName, err := m.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at mi temp: %w", err)
		return nil, err
	}

	dataType := "mi"
	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyous by TXID sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from mi temp: %w", err)
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
				err = fmt.Errorf("error at scan from mi temp: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in mi temp: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in mi temp: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in mi temp: %w", updateTimeStr, err)
				return nil, err
			}

			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (m *miTempRepositorySQLite3Impl) GetMisByTXID(ctx context.Context, txID string, userID string, device string) ([]*Mi, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TITLE,
  IS_CHECKED,
  BOARD_NAME,
  LIMIT_TIME,
  ESTIMATE_START_TIME,
  ESTIMATE_END_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM MI
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	repName, err := m.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MI: %w", err)
		return nil, err
	}

	dataType := "mi"

	queryArgsForCreate := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get mi by txid sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{}
	queryArgs = append(queryArgs, queryArgsForCreate...)

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from MI: %w", err)
		return nil, err
	}
	defer rows.Close()

	mis := []*Mi{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			mi := &Mi{}
			mi.RepName = repName
			createTimeStr, updateTimeStr := "", ""
			limitTime, estimateStartTime, estimateEndTime := sqllib.NullString{}, sqllib.NullString{}, sqllib.NullString{}

			err = rows.Scan(
				&mi.IsDeleted,
				&mi.ID,
				&mi.Title,
				&mi.IsChecked,
				&mi.BoardName,
				&limitTime,
				&estimateStartTime,
				&estimateEndTime,
				&createTimeStr,
				&mi.CreateApp,
				&mi.CreateDevice,
				&mi.CreateUser,
				&updateTimeStr,
				&mi.UpdateApp,
				&mi.UpdateDevice,
				&mi.UpdateUser,
				&mi.RepName,
				&mi.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan mi: %w", err)
				return nil, err
			}

			mi.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in MI: %w", createTimeStr, err)
				return nil, err
			}
			mi.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in MI: %w", updateTimeStr, err)
				return nil, err
			}
			if limitTime.Valid {
				parsedLimitTime, _ := time.Parse(sqlite3impl.TimeLayout, limitTime.String)
				mi.LimitTime = &parsedLimitTime
			}
			if estimateStartTime.Valid {
				parsedEstimateStartTime, _ := time.Parse(sqlite3impl.TimeLayout, estimateStartTime.String)
				mi.EstimateStartTime = &parsedEstimateStartTime
			}
			if estimateEndTime.Valid {
				parsedEstimateEndTime, _ := time.Parse(sqlite3impl.TimeLayout, estimateEndTime.String)
				mi.EstimateEndTime = &parsedEstimateEndTime
			}
			mis = append(mis, mi)
		}
	}
	return mis, nil
}

func (m *miTempRepositorySQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	sql := `
DELETE FROM MI
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete temp mi kyou by TXID sql: %w", err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		txID,
		userID,
		device,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at delete temp mi kyou by TXID sql: %w", err)
		return err
	}
	return nil
}

func (m *miTempRepositorySQLite3Impl) UnWrapTyped() ([]MiTempRepository, error) {
	return []MiTempRepository{m}, nil
}

func (m *miTempRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{m}, nil
}

func (m *miTempRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error) {
	return nil, fmt.Errorf("not implements GetLatestDataRepositoryAddress at temp rep")
}
