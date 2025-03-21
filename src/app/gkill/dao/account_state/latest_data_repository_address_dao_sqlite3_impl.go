package account_state

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/dao/memory_db"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type latestDataRepositoryAddressSQLite3Impl struct {
	db        *sql.DB
	m         *sync.Mutex
	userID    string
	tableName string
}

func NewLatestDataRepositoryAddressSQLite3Impl(userID string) (LatestDataRepositoryAddressDAO, error) {
	var err error

	latestDataRepositoryAddress := &latestDataRepositoryAddressSQLite3Impl{
		m:         &sync.Mutex{},
		userID:    userID,
		tableName: fmt.Sprintf("LATEST_DATA_REPOSITORY_ADDRESS_%s", userID),
	}

	ctx := context.Background()

	if gkill_options.IsCacheInMemory {
		latestDataRepositoryAddress.db = memory_db.MemoryDB
	} else {
		latestDataRepositoryAddress.db, err = sql.Open("sqlite3", os.ExpandEnv(filepath.Join(gkill_options.CacheDir, latestDataRepositoryAddress.tableName+".db?_timeout=60000&_journal=DELETE")))
		if err != nil {
			err = fmt.Errorf("error at open database: %w", err)
			return nil, err
		}
	}

	sql := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  IS_DELETED NOT NULL,
  TARGET_ID NOT NULL,
  LATEST_DATA_REPOSITORY_NAME NOT NULL,
  DATA_UPDATE_TIME NOT NULL,
  PRIMARY KEY(TARGET_ID)
);`, latestDataRepositoryAddress.tableName)
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := latestDataRepositoryAddress.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at CREATE TABLE LATEST_DATA_REPOSITORY_ADDRESS statement %s: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create LATEST_DATA_REPOSITORY_ADDRESS table to %s: %w", err)
		return nil, err
	}

	return latestDataRepositoryAddress, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) GetAllLatestDataRepositoryAddresses(ctx context.Context) (map[string]*LatestDataRepositoryAddress, error) {
	l.m.Lock()
	l.m.Unlock()
	sql := fmt.Sprintf(`
SELECT 
  IS_DELETED,
  TARGET_ID,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME
FROM %s
`, l.tableName)
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

	latestDataRepositoryAddresses := map[string]*LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			latestDataRepositoryAddress := &LatestDataRepositoryAddress{}
			dataUpdateTimeStr := ""
			err = rows.Scan(
				&latestDataRepositoryAddress.IsDeleted,
				&latestDataRepositoryAddress.TargetID,
				&latestDataRepositoryAddress.LatestDataRepositoryName,
				&dataUpdateTimeStr,
			)

			latestDataRepositoryAddress.DataUpdateTime, err = time.Parse(sqlite3impl.TimeLayout, dataUpdateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file data update time %s at %s in LATEST_DATA_REPOSITORY_ADDREDD: %w", dataUpdateTimeStr, latestDataRepositoryAddress.TargetID, err)
				return nil, err
			}

			latestDataRepositoryAddresses[latestDataRepositoryAddress.TargetID] = latestDataRepositoryAddress
		}
	}
	return latestDataRepositoryAddresses, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) GetLatestDataRepositoryAddressesByRepName(ctx context.Context, repName string) (map[string]*LatestDataRepositoryAddress, error) {
	l.m.Lock()
	l.m.Unlock()
	sql := fmt.Sprintf(`
SELECT 
  IS_DELETED,
  TARGET_ID,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME
FROM %s
WHERE LATEST_DATA_REPOSITORY_NAME = ?
`, l.tableName)
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all data repository by rep name addresses sql: %w", err)
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

	latestDataRepositoryAddresses := map[string]*LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			latestDataRepositoryAddress := &LatestDataRepositoryAddress{}
			dataUpdateTimeStr := ""
			err = rows.Scan(
				&latestDataRepositoryAddress.IsDeleted,
				&latestDataRepositoryAddress.TargetID,
				&latestDataRepositoryAddress.LatestDataRepositoryName,
				&dataUpdateTimeStr,
			)

			latestDataRepositoryAddress.DataUpdateTime, err = time.Parse(sqlite3impl.TimeLayout, dataUpdateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file data update time %s at %s in LATEST_DATA_REPOSITORY_ADDREDD: %w", dataUpdateTimeStr, latestDataRepositoryAddress.TargetID, err)
				return nil, err
			}

			latestDataRepositoryAddresses[latestDataRepositoryAddress.TargetID] = latestDataRepositoryAddress
		}
	}
	return latestDataRepositoryAddresses, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, targetID string) (*LatestDataRepositoryAddress, error) {
	l.m.Lock()
	l.m.Unlock()
	sql := fmt.Sprintf(`
SELECT 
  IS_DELETED,
  TARGET_ID,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME
FROM %s
WHERE TARGET_ID = ?
`, l.tableName)
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get latest data repository addresses sql: %w", err)
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
				&latestDataRepositoryAddress.IsDeleted,
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

func (l *latestDataRepositoryAddressSQLite3Impl) AddOrUpdateLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	deleteSQL := fmt.Sprintf(`
DELETE FROM %s
WHERE TARGET_ID = ?`, l.tableName)

	insertSQL := fmt.Sprintf(`
INSERT INTO %s (
  IS_DELETED,
  TARGET_ID,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME
) VALUES (
  ?,
  ?,
  ?,
  ?
)
`, l.tableName)

	gkill_log.TraceSQL.Printf("sql: %s", deleteSQL)
	deleteStmt, err := l.db.PrepareContext(ctx, deleteSQL)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repoisitory address delete sql: %w", err)
		return false, err
	}
	defer deleteStmt.Close()

	deleteQueryArgs := []interface{}{
		latestDataRepositoryAddress.TargetID,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", deleteSQL, deleteQueryArgs)
	_, err = deleteStmt.ExecContext(ctx, deleteQueryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", insertSQL)
	insertStmt, err := l.db.PrepareContext(ctx, insertSQL)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repoisitory insert address sql: %w", err)
		return false, err
	}
	defer insertStmt.Close()

	insertQueryArgs := []interface{}{
		latestDataRepositoryAddress.IsDeleted,
		latestDataRepositoryAddress.TargetID,
		latestDataRepositoryAddress.LatestDataRepositoryName,
		latestDataRepositoryAddress.DataUpdateTime.Format(sqlite3impl.TimeLayout),
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", insertSQL, insertQueryArgs)
	_, err = insertStmt.ExecContext(ctx, insertQueryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) AddOrUpdateLatestDataRepositoryAddresses(ctx context.Context, latestDataRepositoryAddresses []*LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	deleteSQL := fmt.Sprintf(`
DELETE FROM %s
WHERE TARGET_ID = ?`, l.tableName)

	insertSQL := fmt.Sprintf(`
INSERT INTO %s (
  IS_DELETED,
  TARGET_ID,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME
) VALUES (
  ?,
  ?,
  ?,
  ?
)
`, l.tableName)

	tx, err := l.db.Begin()
	if err != nil {
		fmt.Errorf("error at begin: %w", err)
		return false, err
	}

	for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
		_, err := func() (bool, error) {
			gkill_log.TraceSQL.Printf("sql: %s", deleteSQL)
			deleteStmt, err := tx.PrepareContext(ctx, deleteSQL)
			if err != nil {
				err = fmt.Errorf("error at add or update latest data repoisitory address delete sql: %w", err)
				errTx := tx.Rollback()
				if errTx != nil {
					err = fmt.Errorf("error at rollback: %w: %w", err, errTx)
				}
				return false, err
			}
			defer deleteStmt.Close()

			deleteQueryArgs := []interface{}{
				latestDataRepositoryAddress.TargetID,
			}
			gkill_log.TraceSQL.Printf("sql: %s query: %#v", deleteSQL, deleteQueryArgs)
			_, err = deleteStmt.ExecContext(ctx, deleteQueryArgs...)
			if err != nil {
				err = fmt.Errorf("error at query :%w", err)
				errTx := tx.Rollback()
				if errTx != nil {
					err = fmt.Errorf("error at rollback: %w: %w", err, errTx)
				}
				return false, err
			}

			gkill_log.TraceSQL.Printf("sql: %s", insertSQL)
			insertStmt, err := tx.PrepareContext(ctx, insertSQL)
			if err != nil {
				err = fmt.Errorf("error at add or update latest data repoisitory address insert sql: %w", err)
				errTx := tx.Rollback()
				if errTx != nil {
					err = fmt.Errorf("error at rollback: %w: %w", err, errTx)
				}
				return false, err
			}
			defer insertStmt.Close()
			insertQueryArgs := []interface{}{
				latestDataRepositoryAddress.IsDeleted,
				latestDataRepositoryAddress.TargetID,
				latestDataRepositoryAddress.LatestDataRepositoryName,
				latestDataRepositoryAddress.DataUpdateTime.Format(sqlite3impl.TimeLayout),
			}

			gkill_log.TraceSQL.Printf("sql: %s query: %#v", insertSQL, insertQueryArgs)
			_, err = insertStmt.ExecContext(ctx, insertQueryArgs...)
			if err != nil {
				err = fmt.Errorf("error at query :%w", err)
				errTx := tx.Rollback()
				if errTx != nil {
					err = fmt.Errorf("error at rollback: %w: %w", err, errTx)
				}
				return false, err
			}
			return true, nil
		}()
		if err != nil {
			return false, err
		}
	}

	errCommit := tx.Commit()
	if errCommit != nil {
		err = fmt.Errorf("error at commit: %w: %w", err, errCommit)
		return false, err
	}
	return true, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) DeleteLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	sql := fmt.Sprintf(`
DELETE FROM %s
WHERE TARGET_ID = ?
`, l.tableName)
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
	sql := fmt.Sprintf(`
DELETE FROM %s
`, l.tableName)
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
	sql := fmt.Sprintf(`
DELETE FROM %s
WHERE LATEST_DATA_REPOSITORY_NAME  = ?
`, l.tableName)
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
	if !gkill_options.IsCacheInMemory {
		return l.db.Close()
	}
	return nil
}
