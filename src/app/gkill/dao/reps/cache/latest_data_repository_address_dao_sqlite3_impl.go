package gkill_cache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

const CURRENT_SCHEMA_VERSION_LATEST_DATA_REPOSITORY_ADDRESS_DAO = "1.0.0"

type latestDataRepositoryAddressSQLite3Impl struct {
	db        *sql.DB
	m         *sync.RWMutex
	userID    string
	tableName string
}

func NewLatestDataRepositoryAddressSQLite3Impl(userID string, db *sql.DB, mutex *sync.RWMutex) (LatestDataRepositoryAddressDAO, error) {
	var err error

	latestDataRepositoryAddress := &latestDataRepositoryAddressSQLite3Impl{
		m:         mutex,
		userID:    userID,
		tableName: fmt.Sprintf("LATEST_DATA_REPOSITORY_ADDRESS_%s", userID),
		db:        db,
	}

	ctx := context.Background()

	if isOld, oldVerDAO, err := checkAndResolveDataSchemaLatestDataRepositoryAddressDAO(ctx, db); err != nil {
		return nil, err
	} else if isOld {
		if oldVerDAO != nil {
			return oldVerDAO, nil
		} else {
			err = fmt.Errorf("error at load database schema latest data repository address dao")
			return nil, err
		}
	}

	sql := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  IS_DELETED NOT NULL,
  TARGET_ID_IN_DATA,
  TARGET_ID NOT NULL,
  LATEST_DATA_REPOSITORY_NAME NOT NULL,
  DATA_UPDATE_TIME_UNIX NOT NULL,
  LATEST_DATA_REPOSITORY_ADDRESS_UPDATED_TIME_UNIX NOT NULL,
  PRIMARY KEY(TARGET_ID)
);`, latestDataRepositoryAddress.tableName)
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := latestDataRepositoryAddress.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at CREATE TABLE LATEST_DATA_REPOSITORY_ADDRESS statement: %w", err)
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
		err = fmt.Errorf("error at create LATEST_DATA_REPOSITORY_ADDRESS table: %w", err)
		return nil, err
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create %s table: %w", latestDataRepositoryAddress.tableName, err)
		return nil, err
	}

	indexSQL := fmt.Sprintf("CREATE INDEX IF NOT EXISTS INDEX_%s ON %s (TARGET_ID);", latestDataRepositoryAddress.tableName, latestDataRepositoryAddress.tableName)
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	indexStmt, err := latestDataRepositoryAddress.db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create %s index statement: %w", latestDataRepositoryAddress.tableName, err)
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
		err = fmt.Errorf("error at create %s index: %w", latestDataRepositoryAddress.tableName, err)
		return nil, err
	}

	return latestDataRepositoryAddress, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) GetAllLatestDataRepositoryAddresses(ctx context.Context) (map[string]LatestDataRepositoryAddress, error) {
	l.m.RLock()
	defer l.m.RUnlock()
	sql := fmt.Sprintf(`
SELECT 
  IS_DELETED,
  TARGET_ID,
  TARGET_ID_IN_DATA,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME_UNIX,
  LATEST_DATA_REPOSITORY_ADDRESS_UPDATED_TIME_UNIX
FROM %s
`, l.tableName)
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all latest data repository addresses sql: %w", err)
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

	latestDataRepositoryAddresses := map[string]LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			latestDataRepositoryAddress := LatestDataRepositoryAddress{}
			dataUpdateTimeUnix := int64(0)
			latestDataRepositoryAddressUpdatedTimeUnix := int64(0)
			err = rows.Scan(
				&latestDataRepositoryAddress.IsDeleted,
				&latestDataRepositoryAddress.TargetID,
				&latestDataRepositoryAddress.TargetIDInData,
				&latestDataRepositoryAddress.LatestDataRepositoryName,
				&dataUpdateTimeUnix,
				&latestDataRepositoryAddressUpdatedTimeUnix,
			)
			if err != nil {
				err = fmt.Errorf("error at scan latest data repository address: %w", err)
				return nil, err
			}

			latestDataRepositoryAddress.DataUpdateTime = time.Unix(dataUpdateTimeUnix, int64(0))
			latestDataRepositoryAddress.LatestDataRepositoryAddressUpdatedTime = time.Unix(latestDataRepositoryAddressUpdatedTimeUnix, int64(0))

			latestDataRepositoryAddresses[latestDataRepositoryAddress.TargetID] = latestDataRepositoryAddress
		}
	}
	return latestDataRepositoryAddresses, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) GetLatestDataRepositoryAddressesByRepName(ctx context.Context, repName string) (map[string]LatestDataRepositoryAddress, error) {
	l.m.RLock()
	defer l.m.RUnlock()
	sql := fmt.Sprintf(`
SELECT 
  IS_DELETED,
  TARGET_ID,
  TARGET_ID_IN_DATA,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME_UNIX,
  LATEST_DATA_REPOSITORY_ADDRESS_UPDATED_TIME_UNIX
FROM %s
WHERE LATEST_DATA_REPOSITORY_NAME = ?
`, l.tableName)
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all data repository by rep name addresses sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		repName,
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

	latestDataRepositoryAddresses := map[string]LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			latestDataRepositoryAddress := LatestDataRepositoryAddress{}
			dataUpdateTimeUnix := int64(0)
			latestDataRepositoryAddressUpdatedTimeUnix := int64(0)
			err = rows.Scan(
				&latestDataRepositoryAddress.IsDeleted,
				&latestDataRepositoryAddress.TargetID,
				&latestDataRepositoryAddress.TargetIDInData,
				&latestDataRepositoryAddress.LatestDataRepositoryName,
				&dataUpdateTimeUnix,
				&latestDataRepositoryAddressUpdatedTimeUnix,
			)
			if err != nil {
				err = fmt.Errorf("error at scan latest data repository address: %w", err)
				return nil, err
			}

			latestDataRepositoryAddress.DataUpdateTime = time.Unix(dataUpdateTimeUnix, int64(0))
			latestDataRepositoryAddress.LatestDataRepositoryAddressUpdatedTime = time.Unix(latestDataRepositoryAddressUpdatedTimeUnix, int64(0))

			latestDataRepositoryAddresses[latestDataRepositoryAddress.TargetID] = latestDataRepositoryAddress
		}
	}
	return latestDataRepositoryAddresses, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, targetID string) (*LatestDataRepositoryAddress, error) {
	l.m.RLock()
	defer l.m.RUnlock()
	sql := fmt.Sprintf(`
SELECT 
  IS_DELETED,
  TARGET_ID,
  TARGET_ID_IN_DATA,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME_UNIX,
  LATEST_DATA_REPOSITORY_ADDRESS_UPDATED_TIME_UNIX
FROM %s
WHERE TARGET_ID = ?
`, l.tableName)
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get latest data repository addresses sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		targetID,
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

	latestDataRepositoryAddresses := []*LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			latestDataRepositoryAddress := &LatestDataRepositoryAddress{}
			dataUpdateTimeUnix := int64(0)
			latestDataRepositoryAddressUpdatedTimeUnix := int64(0)
			err = rows.Scan(
				&latestDataRepositoryAddress.IsDeleted,
				&latestDataRepositoryAddress.TargetID,
				&latestDataRepositoryAddress.TargetIDInData,
				&latestDataRepositoryAddress.LatestDataRepositoryName,
				&dataUpdateTimeUnix,
				&latestDataRepositoryAddressUpdatedTimeUnix,
			)
			if err != nil {
				err = fmt.Errorf("error at scan latest data repository address: %w", err)
				return nil, err
			}

			latestDataRepositoryAddress.DataUpdateTime = time.Unix(dataUpdateTimeUnix, int64(0))
			latestDataRepositoryAddress.LatestDataRepositoryAddressUpdatedTime = time.Unix(latestDataRepositoryAddressUpdatedTimeUnix, int64(0))

			latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
		}
	}
	if len(latestDataRepositoryAddresses) == 0 {
		return nil, nil
	}
	return latestDataRepositoryAddresses[0], nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) GetLatestDataRepositoryAddressByUpdateTimeAfter(ctx context.Context, updateTime time.Time, limit int64) (map[string]LatestDataRepositoryAddress, error) {
	l.m.RLock()
	defer l.m.RUnlock()
	sql := fmt.Sprintf(`
SELECT 
  IS_DELETED,
  TARGET_ID,
  TARGET_ID_IN_DATA,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME_UNIX,
  LATEST_DATA_REPOSITORY_ADDRESS_UPDATED_TIME_UNIX
FROM %s
WHERE LATEST_DATA_REPOSITORY_ADDRESS_UPDATED_TIME_UNIX >= ?
LIMIT ?
`, l.tableName)
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all latest data repository addresses sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		updateTime.Unix(),
		limit,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s queryArgs: %v", sql, queryArgs)
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

	latestDataRepositoryAddresses := map[string]LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			latestDataRepositoryAddress := LatestDataRepositoryAddress{}
			dataUpdateTimeUnix := int64(0)
			latestDataRepositoryAddressUpdatedTimeUnix := int64(0)
			err = rows.Scan(
				&latestDataRepositoryAddress.IsDeleted,
				&latestDataRepositoryAddress.TargetID,
				&latestDataRepositoryAddress.TargetIDInData,
				&latestDataRepositoryAddress.LatestDataRepositoryName,
				&dataUpdateTimeUnix,
				&latestDataRepositoryAddressUpdatedTimeUnix,
			)
			if err != nil {
				return nil, err
			}

			latestDataRepositoryAddress.DataUpdateTime = time.Unix(dataUpdateTimeUnix, int64(0))
			latestDataRepositoryAddress.LatestDataRepositoryAddressUpdatedTime = time.Unix(latestDataRepositoryAddressUpdatedTimeUnix, int64(0))

			latestDataRepositoryAddresses[latestDataRepositoryAddress.TargetID] = latestDataRepositoryAddress
		}
	}
	return latestDataRepositoryAddresses, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) AddOrUpdateLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	deleteSQL := fmt.Sprintf(`
DELETE FROM %s
WHERE TARGET_ID = ?`, l.tableName)

	insertSQL := fmt.Sprintf(`
INSERT INTO %s (
  IS_DELETED,
  TARGET_ID,
  TARGET_ID_IN_DATA,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME_UNIX,
  LATEST_DATA_REPOSITORY_ADDRESS_UPDATED_TIME_UNIX
) VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)
`, l.tableName)

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", deleteSQL)
	deleteStmt, err := l.db.PrepareContext(ctx, deleteSQL)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repoisitory address delete sql: %w", err)
		return false, err
	}
	defer func() {
		err := deleteStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	deleteQueryArgs := []interface{}{
		latestDataRepositoryAddress.TargetID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", deleteSQL, deleteQueryArgs)
	_, err = deleteStmt.ExecContext(ctx, deleteQueryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", insertSQL)
	insertStmt, err := l.db.PrepareContext(ctx, insertSQL)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repoisitory insert address sql: %w", err)
		return false, err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	insertQueryArgs := []interface{}{
		latestDataRepositoryAddress.IsDeleted,
		latestDataRepositoryAddress.TargetID,
		latestDataRepositoryAddress.TargetIDInData,
		latestDataRepositoryAddress.LatestDataRepositoryName,
		latestDataRepositoryAddress.DataUpdateTime.Unix(),
		latestDataRepositoryAddress.LatestDataRepositoryAddressUpdatedTime.Unix(),
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", insertSQL, insertQueryArgs)
	_, err = insertStmt.ExecContext(ctx, insertQueryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) AddOrUpdateLatestDataRepositoryAddresses(ctx context.Context, latestDataRepositoryAddresses []LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}

	deleteSQL := fmt.Sprintf(`
DELETE FROM %s
WHERE TARGET_ID = ?`, l.tableName)

	deleteStmt, err := tx.PrepareContext(ctx, deleteSQL)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repoisitory address delete sql: %w", err)
		errTx := tx.Rollback()
		if errTx != nil {
			err = fmt.Errorf("error at rollback: %w: %w", err, errTx)
		}
		return false, err
	}
	defer func() {
		err := deleteStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	insertSQL := fmt.Sprintf(`
INSERT INTO %s (
  IS_DELETED,
  TARGET_ID,
  TARGET_ID_IN_DATA,
  LATEST_DATA_REPOSITORY_NAME,
  DATA_UPDATE_TIME_UNIX,
  LATEST_DATA_REPOSITORY_ADDRESS_UPDATED_TIME_UNIX
) VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)
`, l.tableName)
	insertStmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repoisitory address insert sql: %w", err)
		errTx := tx.Rollback()
		if errTx != nil {
			err = fmt.Errorf("error at rollback: %w: %w", err, errTx)
		}
		return false, err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
		_, err := func() (bool, error) {
			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", deleteSQL)
			deleteQueryArgs := []interface{}{
				latestDataRepositoryAddress.TargetID,
			}
			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", deleteSQL, deleteQueryArgs)
			_, err = deleteStmt.ExecContext(ctx, deleteQueryArgs...)
			if err != nil {
				err = fmt.Errorf("error at query :%w", err)
				errTx := tx.Rollback()
				if errTx != nil {
					err = fmt.Errorf("error at rollback: %w: %w", err, errTx)
				}
				return false, err
			}

			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", insertSQL)
			insertQueryArgs := []interface{}{
				latestDataRepositoryAddress.IsDeleted,
				latestDataRepositoryAddress.TargetID,
				latestDataRepositoryAddress.TargetIDInData,
				latestDataRepositoryAddress.LatestDataRepositoryName,
				latestDataRepositoryAddress.DataUpdateTime.Unix(),
				latestDataRepositoryAddress.LatestDataRepositoryAddressUpdatedTime.Unix(),
			}

			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", insertSQL, insertQueryArgs)
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete latest data repository address sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		latestDataRepositoryAddress.TargetID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete all latest data repository address sql: %w", err)
		return false, err
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete all latest data repository address sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		repName,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) ExtructUpdatedLatestDataRepositoryAddressDatas(ctx context.Context, latestDataRepositoryAddresses []LatestDataRepositoryAddress) ([]LatestDataRepositoryAddress, error) {
	existlatestDataRepositoryAddresses, err := l.GetAllLatestDataRepositoryAddresses(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all latest data repository addresses: %w", err)
		return nil, err
	}

	latestDataRepositoryAddressMap := map[string]LatestDataRepositoryAddress{}
	for _, latestDataRepositoryAddress := range existlatestDataRepositoryAddresses {
		latestDataRepositoryAddressMap[latestDataRepositoryAddress.TargetID] = latestDataRepositoryAddress
	}

	// 内容が更新された（ハッシュ一が在しないデータのみを抽出する
	notExistsLatestDataRepositoryAddresses := []LatestDataRepositoryAddress{}
	for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
		if existLatestDataRepositoryAddress, exist := latestDataRepositoryAddressMap[latestDataRepositoryAddress.TargetID]; exist {
			if existLatestDataRepositoryAddress.DataUpdateTime.Before(latestDataRepositoryAddress.DataUpdateTime) {
				notExistsLatestDataRepositoryAddresses = append(notExistsLatestDataRepositoryAddresses, latestDataRepositoryAddress)
			}
		} else {
			notExistsLatestDataRepositoryAddresses = append(notExistsLatestDataRepositoryAddresses, latestDataRepositoryAddress)
		}
	}
	return notExistsLatestDataRepositoryAddresses, nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) UpdateLatestDataRepositoryAddressesData(ctx context.Context, latestDataRepositoryAddresses []LatestDataRepositoryAddress) error {
	notExistsLatestDataRepositoryAddresses, err := l.ExtructUpdatedLatestDataRepositoryAddressDatas(ctx, latestDataRepositoryAddresses)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository addresses: %w", err)
		return err
	}
	// いれる
	_, err = l.AddOrUpdateLatestDataRepositoryAddresses(ctx, notExistsLatestDataRepositoryAddresses)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository addresses: %w", err)
		return err
	}
	return nil
}

func (l *latestDataRepositoryAddressSQLite3Impl) Close(ctx context.Context) error {
	l.m.Lock()
	defer l.m.Unlock()
	if gkill_options.IsCacheInMemory {
		sql := fmt.Sprintf(`DROP TABLE %s `, l.tableName)
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
		stmt, err := l.db.PrepareContext(ctx, sql)
		if err != nil {
			err = fmt.Errorf("error at DROP TABLE LATEST_DATA_REPOSITORY_ADDRESS statement: %w", err)
			return err
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
			err = fmt.Errorf("error at drop table LATEST_DATA_REPOSITORY_ADDRESS table: %w", err)
			return err
		}
	} else {
		return l.db.Close()
	}
	return nil
}

func checkAndResolveDataSchemaLatestDataRepositoryAddressDAO(ctx context.Context, db *sql.DB) (isOld bool, oldVerDAO LatestDataRepositoryAddressDAO, err error) {
	schemaVersionKey := "SCHEMA_VERSION_LATEST_DATA_REPOSITORY_ADDRESS"
	currentSchemaVersion := CURRENT_SCHEMA_VERSION_LATEST_DATA_REPOSITORY_ADDRESS_DAO

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
