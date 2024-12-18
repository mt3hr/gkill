package account_state

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type latestDataRepositoryAddressSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewLatestDataRepositoryAddressSQLite3Impl(ctx context.Context, filename string) (LatestDataRepositoryAddressDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "LATEST_DATA_REPOSITORY_ADDRESS" (
  TARGET_ID NOT NULL,
  LATEST_DATA_REPOSITORY_NAME NOT NULL,
  DATA_UPDATE_TIME NOT NULL,
  PRIMARY KEY(TARGET_ID)
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create table LATEST_DATA_REPOSITORY_ADDRESS statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create LATEST_DATA_REPOSITORY_ADDRESS table to %s: %w", filename, err)
		return nil, err
	}

	return &latestDataRepositoryAddressSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) GetAllLatestDataRepositoryAddresses(ctx context.Context) ([]*LatestDataRepositoryAddress, error) {
	sql := `
SELECT 
  TARGET_ID,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME
FROM LATEST_DATA_REPOSITORY_ADDRESS
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all latest data repository addresses sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	latestDataRepositoryAddresses := []*LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			latestDataRepositoryAddress := &LatestDataRepositoryAddress{}
			dataUpdateTimeStr := ""
			err = rows.Scan(
				&latestDataRepositoryAddress.TargetID,
				&latestDataRepositoryAddress.LatestDataRepositoryName,
				&dataUpdateTimeStr,
			)

			latestDataRepositoryAddress.DataUpdateTime, err = time.Parse(sqlite3impl.TimeLayout, dataUpdateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file data update time %s at %s in LATEST_DATA_REPOSITORY_ADDREDD: %w", dataUpdateTimeStr, latestDataRepositoryAddress.TargetID, err)
				return nil, err
			}

			latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
		}
	}
	return latestDataRepositoryAddresses, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) GetLatestDataRepositoryAddressesByRepName(ctx context.Context, repName string) ([]*LatestDataRepositoryAddress, error) {
	sql := `
SELECT 
  TARGET_ID,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME
FROM LATEST_DATA_REPOSITORY_ADDRESS
WHERE LATEST_DATA_REPOSITORY_NAME = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all latest data repository addresses sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		repName,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	latestDataRepositoryAddresses := []*LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			latestDataRepositoryAddress := &LatestDataRepositoryAddress{}
			dataUpdateTimeStr := ""
			err = rows.Scan(
				&latestDataRepositoryAddress.TargetID,
				&latestDataRepositoryAddress.LatestDataRepositoryName,
				&dataUpdateTimeStr,
			)

			latestDataRepositoryAddress.DataUpdateTime, err = time.Parse(sqlite3impl.TimeLayout, dataUpdateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file data update time %s at %s in LATEST_DATA_REPOSITORY_ADDREDD: %w", dataUpdateTimeStr, latestDataRepositoryAddress.TargetID, err)
				return nil, err
			}

			latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
		}
	}
	return latestDataRepositoryAddresses, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, targetID string) (*LatestDataRepositoryAddress, error) {
	sql := `
SELECT 
  TARGET_ID,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME
FROM LATEST_DATA_REPOSITORY_ADDRESS
WHERE TARGET_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all latest data repository addresses sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		targetID,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	latestDataRepositoryAddresses := []*LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			latestDataRepositoryAddress := &LatestDataRepositoryAddress{}
			dataUpdateTimeStr := ""
			err = rows.Scan(
				&latestDataRepositoryAddress.TargetID,
				&latestDataRepositoryAddress.LatestDataRepositoryName,
				&dataUpdateTimeStr,
			)

			latestDataRepositoryAddress.DataUpdateTime, err = time.Parse(sqlite3impl.TimeLayout, dataUpdateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file data update time %s at %s in LATEST_DATA_REPOSITORY_ADDREDD: %w", dataUpdateTimeStr, latestDataRepositoryAddress.TargetID, err)
				return nil, err
			}

			latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
		}
	}
	if len(latestDataRepositoryAddresses) == 0 {
		return nil, nil
	}
	return latestDataRepositoryAddresses[0], nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) AddLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	sql := `
INSERT INTO LATEST_DATA_REPOSITORY_ADDRESS (
  TARGET_ID,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME
) VALUES (
  ?,
  ?,
  ?
)
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add latest data repoisitory address sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		latestDataRepositoryAddress.TargetID,
		latestDataRepositoryAddress.LatestDataRepositoryName,
		latestDataRepositoryAddress.DataUpdateTime.Format(sqlite3impl.TimeLayout),
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) UpdateLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	sql := `
UPDATE LATEST_DATA_REPOSITORY_ADDRESS SET
  TARGET_ID = ?,
  LATEST_DATA_REPOSITORY_NAME = ?,
  DATA_UPDATE_TIME = ?
WHERE TARGET_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update latest data repository address sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		latestDataRepositoryAddress.TargetID,
		latestDataRepositoryAddress.LatestDataRepositoryName,
		latestDataRepositoryAddress.DataUpdateTime.Format(sqlite3impl.TimeLayout),
		latestDataRepositoryAddress.TargetID,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) UpdateOrAddLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error) {
	existLatestDataRepositoryAddress, err := l.GetLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress.TargetID)
	if err == nil && existLatestDataRepositoryAddress != nil { // データが存在する場合は更新する
		_, err := l.UpdateLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress)
		if err != nil {
			err = fmt.Errorf("error at update latest data repository address %s: %w", latestDataRepositoryAddress.TargetID, err)
			return false, err
		}
	} else { // データが存在しない場合は作成する
		_, err := l.AddLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress)
		if err != nil {
			err = fmt.Errorf("error at add latest data repository address %s: %w", latestDataRepositoryAddress.TargetID, err)
			return false, err
		}
	}
	return true, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) UpdateOrAddLatestDataRepositoryAddresses(ctx context.Context, latestDataRepositoryAddresses []*LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()

	updateSQL := `
UPDATE LATEST_DATA_REPOSITORY_ADDRESS SET
  TARGET_ID = ?,
  LATEST_DATA_REPOSITORY_NAME = ?,
  DATA_UPDATE_TIME = ?
WHERE TARGET_ID = ?
`
	insertSQL := `
INSERT INTO LATEST_DATA_REPOSITORY_ADDRESS (
  TARGET_ID,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME
) VALUES (
  ?,
  ?,
  ?
)
`

	tx, err := l.db.Begin()
	if err != nil {
		fmt.Errorf("error at begin: %w", err)
		return false, err
	}

	for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
		existRecords, err := l.GetLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress.TargetID)
		if err == nil && existRecords != nil { // データが存在する場合は更新する
			_, err := l.UpdateLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress)
			gkill_log.TraceSQL.Printf("sql: %s", updateSQL)
			stmt, err := tx.PrepareContext(ctx, updateSQL)
			if err != nil {
				err = fmt.Errorf("error at update latest data repository address sql: %w", err)
				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					err = fmt.Errorf("%w: %w", err, rollbackErr)
				}
				return false, err
			}

			queryArgs := []interface{}{
				latestDataRepositoryAddress.TargetID,
				latestDataRepositoryAddress.LatestDataRepositoryName,
				latestDataRepositoryAddress.DataUpdateTime.Format(sqlite3impl.TimeLayout),
				latestDataRepositoryAddress.TargetID,
			}
			gkill_log.TraceSQL.Printf("sql: %s query: %#v", updateSQL, queryArgs)
			_, err = stmt.ExecContext(ctx, queryArgs...)

			if err != nil {
				err = fmt.Errorf("error at query :%w", err)
				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					err = fmt.Errorf("%w: %w", err, rollbackErr)
				}
				return false, err
			}
		} else { // データが存在しない場合は作成する
			gkill_log.TraceSQL.Printf("sql: %s", insertSQL)
			stmt, err := tx.PrepareContext(ctx, insertSQL)
			if err != nil {
				err = fmt.Errorf("error at add latest data repoisitory address sql: %w", err)
				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					err = fmt.Errorf("%w: %w", err, rollbackErr)
				}
				return false, err
			}

			queryArgs := []interface{}{
				latestDataRepositoryAddress.TargetID,
				latestDataRepositoryAddress.LatestDataRepositoryName,
				latestDataRepositoryAddress.DataUpdateTime.Format(sqlite3impl.TimeLayout),
			}
			gkill_log.TraceSQL.Printf("sql: %s query: %#v", insertSQL, queryArgs)
			_, err = stmt.ExecContext(ctx, queryArgs...)

			if err != nil {
				err = fmt.Errorf("error at query :%w", err)
				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					err = fmt.Errorf("%w: %w", err, rollbackErr)
				}
				return false, err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		fmt.Errorf("error at commit: %w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	return true, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) DeleteLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	sql := `
DELETE FROM LATEST_DATA_REPOSITORY_ADDRESS
WHERE TARGET_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete latest data repository address sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		latestDataRepositoryAddress.TargetID,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) DeleteAllLatestDataRepositoryAddress(ctx context.Context) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	sql := `
DELETE FROM LATEST_DATA_REPOSITORY_ADDRESS
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete all latest data repository address sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) DeleteLatestDataRepositoryAddressInRep(ctx context.Context, repName string) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	sql := `
DELETE FROM LATEST_DATA_REPOSITORY_ADDRESS
WHERE LATEST_DATA_REPOSITORY_NAME  = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete all latest data repository address sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		repName,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) Close(ctx context.Context) error {
	return l.db.Close()
}
