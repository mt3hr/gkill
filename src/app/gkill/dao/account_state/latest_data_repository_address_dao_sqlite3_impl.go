package account_state

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
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
	latestDataRepositoryAddress := &latestDataRepositoryAddressSQLite3Impl{
		m:      &sync.Mutex{},
		userID: userID,
	}
	err := latestDataRepositoryAddress.createTableIfNotExist()
	if err != nil {
		return nil, err
	}

	return latestDataRepositoryAddress, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) createTableIfNotExist() error {
	var err error
	ctx := context.Background()

	l.tableName = fmt.Sprintf("LATEST_DATA_REPOSITORY_ADDRESS_%s", l.userID)
	if gkill_options.IsCacheInMemory {
		l.db, err = sql.Open("sqlite3", "file::memory:?cache=shared")
		if err != nil {
			err = fmt.Errorf("error at open database: %w", err)
			return err
		}
	} else {
		l.db, err = sql.Open("sqlite3", filepath.Join(gkill_options.CacheDir, l.tableName+".db"))
		if err != nil {
			err = fmt.Errorf("error at open database: %w", err)
			return err
		}
	}

	sql := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  IS_DELETED NOT NULL,
  TARGET_ID NOT NULL,
  LATEST_DATA_REPOSITORY_NAME NOT NULL,
  DATA_UPDATE_TIME NOT NULL,
  PRIMARY KEY(TARGET_ID)
);`, l.tableName)
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at CREATE TABLE LATEST_DATA_REPOSITORY_ADDRESS statement %s: %w", err)
		return err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create LATEST_DATA_REPOSITORY_ADDRESS table to %s: %w", err)
		return err
	}
	return nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) GetAllLatestDataRepositoryAddresses(ctx context.Context) (map[string]*LatestDataRepositoryAddress, error) {
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

func (l *latestDataRepositoryAddressSQLite3Impl) AddLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	sql := fmt.Sprintf(`
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
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add latest data repoisitory address sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		latestDataRepositoryAddress.IsDeleted,
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

func (l *latestDataRepositoryAddressSQLite3Impl) AddLatestDataRepositoryAddresses(ctx context.Context, latestDataRepositoryAddresses []*LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()

	tx, err := l.db.Begin()
	if err != nil {
		fmt.Errorf("error at begin: %w", err)
		return false, err
	}

	sql := fmt.Sprintf(`
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
	for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
		gkill_log.TraceSQL.Printf("sql: %s", sql)
		stmt, err := tx.PrepareContext(ctx, sql)
		if err != nil {
			err = fmt.Errorf("error at add latest data repoisitory address sql: %w", err)
			errTx := tx.Rollback()
			if errTx != nil {
				err = fmt.Errorf("error at rollback: %w: %w", err, errTx)
			}
			return false, err
		}
		defer stmt.Close()

		queryArgs := []interface{}{
			latestDataRepositoryAddress.IsDeleted,
			latestDataRepositoryAddress.TargetID,
			latestDataRepositoryAddress.LatestDataRepositoryName,
			latestDataRepositoryAddress.DataUpdateTime.Format(sqlite3impl.TimeLayout),
		}
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
		_, err = stmt.ExecContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			errTx := tx.Rollback()
			if errTx != nil {
				err = fmt.Errorf("error at rollback: %w: %w", err, errTx)
			}
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

func (l *latestDataRepositoryAddressSQLite3Impl) UpdateLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	sql := fmt.Sprintf(`
UPDATE %s SET
  IS_DELETED = ?,
  TARGET_ID = ?,
  LATEST_DATA_REPOSITORY_NAME = ?,
  DATA_UPDATE_TIME = ?
WHERE TARGET_ID = ?
`, l.tableName)
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update latest data repository address sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		latestDataRepositoryAddress.IsDeleted,
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

	updateSQL := fmt.Sprintf(`
UPDATE %s SET
  IS_DELETED = ?,
  TARGET_ID = ?,
  LATEST_DATA_REPOSITORY_NAME = ?,
  DATA_UPDATE_TIME = ?
WHERE TARGET_ID = ?
`, l.tableName)
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
				latestDataRepositoryAddress.IsDeleted,
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
				latestDataRepositoryAddress.IsDeleted,
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
	return l.db.Close()
}
