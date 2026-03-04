package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type repositoryDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.RWMutex
}

func NewRepositoryDAOSQLite3Impl(ctx context.Context, filename string) (RepositoryDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=1&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	if gkill_options.Optimize {
		err = sqlite3impl.DeleteAllIndex(db)
		if err != nil {
			err = fmt.Errorf("error at delete all index %w", err)
			return nil, err
		}
	}

	sql := `
CREATE TABLE IF NOT EXISTS "REPOSITORY" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  TYPE NOT NULL,
  FILE NOT NULL,
  USE_TO_WRITE NOT NULL,
  IS_EXECUTE_IDF_WHEN_RELOAD NOT NULL,
  IS_WATCH_TARGET_FOR_UPDATE_REP NOT NULL,
  IS_ENABLE NOT NULL
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create REPOSITORY table statement %s: %w", filename, err)
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
		err = fmt.Errorf("error at create REPOSITORY table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_REPOSITORY ON REPOSITORY (USER_ID);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create REPOSITORY index statement %s: %w", filename, err)
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
		err = fmt.Errorf("error at create REPOSITORY index to %s: %w", filename, err)
		return nil, err
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create REPOSITORY table to %s: %w", filename, err)
		return nil, err
	}

	if gkill_options.Optimize {
		err = sqlite3impl.Optimize(db)
		if err != nil {
			err = fmt.Errorf("error at optimize db %w", err)
			return nil, err
		}
	}

	return &repositoryDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.RWMutex{},
	}, nil
}

func (r *repositoryDAOSQLite3Impl) GetAllRepositories(ctx context.Context) ([]*Repository, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  TYPE,
  FILE,
  USE_TO_WRITE,
  IS_EXECUTE_IDF_WHEN_RELOAD,
  IS_WATCH_TARGET_FOR_UPDATE_REP,
  IS_ENABLE
FROM REPOSITORY
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all repositories sql: %w", err)
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

	repositories := []*Repository{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			repository := &Repository{}
			err = rows.Scan(
				&repository.ID,
				&repository.UserID,
				&repository.Device,
				&repository.Type,
				&repository.File,
				&repository.UseToWrite,
				&repository.IsExecuteIDFWhenReload,
				&repository.IsWatchTargetForUpdateRep,
				&repository.IsEnable,
			)
			if err != nil {
				return nil, err
			}
			base := filepath.Base(repository.File)
			ext := filepath.Ext(base)
			withoutExt := base[:len(base)-len(ext)]
			repository.RepName = withoutExt
			repositories = append(repositories, repository)
		}
	}
	return repositories, nil
}

func (r *repositoryDAOSQLite3Impl) GetRepositories(ctx context.Context, userID string, device string) ([]*Repository, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  TYPE,
  FILE,
  USE_TO_WRITE,
  IS_EXECUTE_IDF_WHEN_RELOAD,
  IS_WATCH_TARGET_FOR_UPDATE_REP,
  IS_ENABLE
FROM REPOSITORY
WHERE USER_ID = ? AND DEVICE = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get repositories sql: %w", err)
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

	repositories := []*Repository{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			repository := &Repository{}
			err = rows.Scan(
				&repository.ID,
				&repository.UserID,
				&repository.Device,
				&repository.Type,
				&repository.File,
				&repository.UseToWrite,
				&repository.IsExecuteIDFWhenReload,
				&repository.IsWatchTargetForUpdateRep,
				&repository.IsEnable,
			)
			if err != nil {
				err = fmt.Errorf("error at scan repository: %w", err)
				return nil, err
			}
			base := filepath.Base(repository.File)
			ext := filepath.Ext(base)
			withoutExt := base[:len(base)-len(ext)]
			repository.RepName = withoutExt

			repositories = append(repositories, repository)
		}
	}
	return repositories, nil
}

func (r *repositoryDAOSQLite3Impl) DeleteWriteRepositories(ctx context.Context, userID string, repositories []*Repository) (bool, error) {
	r.m.Lock()
	defer r.m.Unlock()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at update cache: %w", "error", err)
			}
		}
	}()

	sql := `
DELETE FROM REPOSITORY
WHERE USER_ID = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete repository sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		userID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		return false, err
	}

	insertSQL := `
INSERT INTO REPOSITORY (
  ID,
  USER_ID,
  DEVICE,
  TYPE,
  FILE,
  USE_TO_WRITE,
  IS_EXECUTE_IDF_WHEN_RELOAD,
  IS_WATCH_TARGET_FOR_UPDATE_REP,
  IS_ENABLE 
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

	insertStmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		err = fmt.Errorf("error at add repositories sql: %w", err)
		return false, err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for _, repository := range repositories {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", insertSQL)

		queryArgs := []interface{}{
			repository.ID,
			repository.UserID,
			repository.Device,
			repository.Type,
			repository.File,
			repository.UseToWrite,
			repository.IsExecuteIDFWhenReload,
			repository.IsWatchTargetForUpdateRep,
			repository.IsEnable,
		}
		slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", insertSQL, queryArgs)
		_, err = insertStmt.ExecContext(ctx, queryArgs...)

		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			return false, err
		}
	}

	err = r.checkUseToWriteRepositoryCount(ctx, tx, userID)
	if err != nil {
		return false, err
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		return false, err
	}
	isCommitted = true
	return true, nil
}

func (r *repositoryDAOSQLite3Impl) AddRepository(ctx context.Context, repository *Repository) (bool, error) {
	r.m.Lock()
	defer r.m.Unlock()
	sql := `
INSERT INTO REPOSITORY (
  ID,
  USER_ID,
  DEVICE,
  TYPE,
  FILE,
  USE_TO_WRITE,
  IS_EXECUTE_IDF_WHEN_RELOAD,
  IS_WATCH_TARGET_FOR_UPDATE_REP,
  IS_ENABLE 
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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin update repository: %w", err)
		return false, err
	}
	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at update cache: %w", "error", err)
			}
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add repository sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		repository.ID,
		repository.UserID,
		repository.Device,
		repository.Type,
		repository.File,
		repository.UseToWrite,
		repository.IsExecuteIDFWhenReload,
		repository.IsWatchTargetForUpdateRep,
		repository.IsEnable,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		return false, err
	}
	err = r.checkUseToWriteRepositoryCount(ctx, tx, repository.UserID)
	if err != nil {
		return false, err
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at add repository commit: %w", err)
		return false, err
	}
	isCommitted = true
	return true, nil
}

func (r *repositoryDAOSQLite3Impl) AddRepositories(ctx context.Context, repositories []*Repository) (bool, error) {
	r.m.Lock()
	defer r.m.Unlock()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at update cache: %w", "error", err)
			}
		}
	}()

	sql := `
INSERT INTO REPOSITORY (
  ID,
  USER_ID,
  DEVICE,
  TYPE,
  FILE,
  USE_TO_WRITE,
  IS_EXECUTE_IDF_WHEN_RELOAD,
  IS_WATCH_TARGET_FOR_UPDATE_REP,
  IS_ENABLE 
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

	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add repositories sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for _, repository := range repositories {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
		queryArgs := []interface{}{
			repository.ID,
			repository.UserID,
			repository.Device,
			repository.Type,
			repository.File,
			repository.UseToWrite,
			repository.IsExecuteIDFWhenReload,
			repository.IsWatchTargetForUpdateRep,
			repository.IsEnable,
		}
		slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
		_, err = stmt.ExecContext(ctx, queryArgs...)

		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			return false, err
		}
	}

	for _, repository := range repositories {
		err = r.checkUseToWriteRepositoryCount(ctx, tx, repository.UserID)
		if err != nil {
			return false, err
		}
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		return false, err
	}
	isCommitted = true
	return true, nil
}

func (r *repositoryDAOSQLite3Impl) UpdateRepository(ctx context.Context, repository *Repository) (bool, error) {
	r.m.Lock()
	defer r.m.Unlock()
	sql := `
UPDATE REPOSITORY SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  DEVICE_NAME = ?,
  PARENT_FOLDER_ID = ?,
  SEQ = ?,
  CHECK_WHEN_INITED = ?
WHERE ID = ?
`
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin update repository: %w", err)
		return false, err
	}
	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at update cache: %w", "error", err)
			}
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update repository sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		repository.ID,
		repository.UserID,
		repository.Device,
		repository.Type,
		repository.File,
		repository.UseToWrite,
		repository.IsExecuteIDFWhenReload,
		repository.IsEnable,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		return false, err
	}
	err = r.checkUseToWriteRepositoryCount(ctx, tx, repository.UserID)
	if err != nil {
		return false, err
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at add repository commit: %w", err)
		return false, err
	}
	isCommitted = true
	return true, nil
}

func (r *repositoryDAOSQLite3Impl) DeleteRepository(ctx context.Context, id string) (bool, error) {
	r.m.Lock()
	defer r.m.Unlock()
	sql := `
DELETE FROM REPOSITORY
WHERE ID = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete repository sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		id,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repositoryDAOSQLite3Impl) DeleteAllRepositoriesByUser(ctx context.Context, userID string, device string) (bool, error) {
	r.m.Lock()
	defer r.m.Unlock()
	sql := `
DELETE FROM REPOSITORY
WHERE USER_ID = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete repository sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		userID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repositoryDAOSQLite3Impl) Close(ctx context.Context) error {
	r.m.Lock()
	defer r.m.Unlock()
	return r.db.Close()
}

func (r *repositoryDAOSQLite3Impl) checkUseToWriteRepositoryCount(ctx context.Context, tx *sql.Tx, userID string) error {
	selectDeviceSQL := `
SELECT DEVICE FROM REPOSITORY WHERE USER_ID = ? GROUP BY DEVICE
`
	selectDeviceQueryArgs := []interface{}{
		userID,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", selectDeviceSQL)
	stmt, err := tx.PrepareContext(ctx, selectDeviceSQL)
	if err != nil {
		return err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", selectDeviceSQL, selectDeviceQueryArgs)
	rows, err := stmt.QueryContext(ctx, selectDeviceQueryArgs...)
	if err != nil {
		return err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	selectSQL := `
WITH TYPE_AND_DEVICE AS (SELECT ? AS USER_ID, ? AS DEVICE)
SELECT TYPE, COUNT FROM (
SELECT 'directory' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'directory'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
UNION
SELECT 'gpslog' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'gpslog'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
UNION
SELECT 'kmemo' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'kmemo'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
UNION
SELECT 'kc' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'kc'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
UNION
SELECT 'lantana' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'lantana'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
UNION
SELECT 'mi' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'mi'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
UNION
SELECT 'nlog' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'nlog'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
UNION
SELECT 'notification' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'notification'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
UNION
SELECT 'rekyou' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'rekyou'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
UNION
SELECT 'tag' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'tag'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
UNION
SELECT 'text' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'text'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
UNION
SELECT 'timeis' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'timeis'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
UNION
SELECT 'urlog' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE USE_TO_WRITE = TRUE
AND TYPE = 'urlog'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
AND DEVICE = (SELECT DEVICE FROM TYPE_AND_DEVICE)
AND IS_ENABLE = TRUE
)
GROUP BY TYPE, DEVICE
`
	selectStmt, err := tx.PrepareContext(ctx, selectSQL)
	if err != nil {
		err = fmt.Errorf("error at get use to write repository count sql: %w", err)
		return err
	}
	defer func() {
		err := selectStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	devices := []string{}
	for rows.Next() {
		device := ""
		err := rows.Scan(
			&device,
		)
		if err != nil {
			return err
		}
		devices = append(devices, device)
	}

	for _, targetDevice := range devices {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", selectSQL)

		queryArgs := []interface{}{
			userID,
			targetDevice,
		}
		slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", selectSQL, queryArgs)
		rows, err := selectStmt.QueryContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			return err
		}
		defer func() {
			err := rows.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()

		for rows.Next() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				repType := ""
				count := 0
				err := rows.Scan(
					&repType,
					&count,
				)
				if err != nil {
					slog.Log(ctx, gkill_log.Error, "error", "error", err)
					// err = fmt.Errorf("error at get use to write repository count: %w", err)
					err = fmt.Errorf("書き込み先Rep1つに対してがプロファイルに対して1つとなるようにしてください。対象：「%s」「%s」「%d」", targetDevice, repType, count)
					return err
				}

				if count != 1 {
					// err = fmt.Errorf("error at check use to write repository count")
					// err = fmt.Errorf("rep type %s use to write rep count is %d: %w", repType, count, err)
					err = fmt.Errorf("書き込み先Rep1つに対してがプロファイルに対して1つとなるようにしてください。対象：「%s」「%s」「%d」", targetDevice, repType, count)
					return err
				}
			}
		}
	}
	return nil
}
