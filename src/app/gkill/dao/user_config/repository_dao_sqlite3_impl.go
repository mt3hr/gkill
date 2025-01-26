package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type repositoryDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewRepositoryDAOSQLite3Impl(ctx context.Context, filename string) (RepositoryDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
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
  IS_ENABLE NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create REPOSITORY table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REPOSITORY table to %s: %w", filename, err)
		return nil, err
	}

	return &repositoryDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}

func (r *repositoryDAOSQLite3Impl) GetAllRepositories(ctx context.Context) ([]*Repository, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  TYPE,
  FILE,
  USE_TO_WRITE,
  IS_EXECUTE_IDF_WHEN_RELOAD,
  IS_ENABLE
FROM REPOSITORY
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all repositories sql: %w", err)
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
				&repository.IsEnable,
			)
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
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  TYPE,
  FILE,
  USE_TO_WRITE,
  IS_EXECUTE_IDF_WHEN_RELOAD,
  IS_ENABLE
FROM REPOSITORY
WHERE USER_ID = ? AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get repositories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
		device,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

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
				&repository.IsEnable,
			)
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
	tx, err := r.db.Begin()
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}

	sql := `
DELETE FROM REPOSITORY
WHERE USER_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete repository sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		errAtRollback := tx.Rollback()
		err = fmt.Errorf("%w, %w", err, errAtRollback)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	for _, repository := range repositories {
		sql := `
INSERT INTO REPOSITORY (
  ID,
  USER_ID,
  DEVICE,
  TYPE,
  FILE,
  USE_TO_WRITE,
  IS_EXECUTE_IDF_WHEN_RELOAD,
  IS_ENABLE 
) VALUES (
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

		gkill_log.TraceSQL.Printf("sql: %s", sql)
		stmt, err := tx.PrepareContext(ctx, sql)
		if err != nil {
			err = fmt.Errorf("error at add repositories sql: %w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

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
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
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

	for _, repository := range repositories {
		err = r.checkUseToWriteRepositoryCount(ctx, tx, repository.UserID)
		if err != nil {
			errAtRollback := tx.Rollback()
			err = fmt.Errorf("%w, %w", err, errAtRollback)
			err = fmt.Errorf("error at query :%w", err)
			return false, err
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

func (r *repositoryDAOSQLite3Impl) AddRepository(ctx context.Context, repository *Repository) (bool, error) {
	sql := `
INSERT INTO REPOSITORY (
  ID,
  USER_ID,
  DEVICE,
  TYPE,
  FILE,
  USE_TO_WRITE,
  IS_EXECUTE_IDF_WHEN_RELOAD,
  IS_ENABLE 
) VALUES (
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
	tx, err := r.db.Begin()
	if err != nil {
		err = fmt.Errorf("error at begin update repository: %w", err)
		return false, err
	}
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add repository sql: %w", err)
		return false, err
	}
	defer stmt.Close()

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
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		errAtRollback := tx.Rollback()
		err = fmt.Errorf("%w, %w", err, errAtRollback)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	err = r.checkUseToWriteRepositoryCount(ctx, tx, repository.UserID)
	if err != nil {
		errAtRollback := tx.Rollback()
		err = fmt.Errorf("%w, %w", err, errAtRollback)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at add repository commit: %w", err)
		return false, err
	}
	return true, nil
}

func (r *repositoryDAOSQLite3Impl) AddRepositories(ctx context.Context, repositories []*Repository) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		fmt.Errorf("error at begin: %w", err)
		return false, err
	}

	for _, repository := range repositories {
		sql := `
INSERT INTO REPOSITORY (
  ID,
  USER_ID,
  DEVICE,
  TYPE,
  FILE,
  USE_TO_WRITE,
  IS_EXECUTE_IDF_WHEN_RELOAD,
  IS_ENABLE 
) VALUES (
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

		gkill_log.TraceSQL.Printf("sql: %s", sql)
		stmt, err := tx.PrepareContext(ctx, sql)
		if err != nil {
			err = fmt.Errorf("error at add repositories sql: %w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

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
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
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

	for _, repository := range repositories {
		err = r.checkUseToWriteRepositoryCount(ctx, tx, repository.UserID)
		if err != nil {
			errAtRollback := tx.Rollback()
			err = fmt.Errorf("%w, %w", err, errAtRollback)
			err = fmt.Errorf("error at query :%w", err)
			return false, err
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

func (r *repositoryDAOSQLite3Impl) UpdateRepository(ctx context.Context, repository *Repository) (bool, error) {
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
	tx, err := r.db.Begin()
	if err != nil {
		err = fmt.Errorf("error at begin update repository: %w", err)
		return false, err
	}
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update repository sql: %w", err)
		return false, err
	}
	defer stmt.Close()

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
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		errAtRollback := tx.Rollback()
		err = fmt.Errorf("%w, %w", err, errAtRollback)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	err = r.checkUseToWriteRepositoryCount(ctx, tx, repository.UserID)
	if err != nil {
		errAtRollback := tx.Rollback()
		err = fmt.Errorf("%w, %w", err, errAtRollback)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at add repository commit: %w", err)
		return false, err
	}
	return true, nil
}

func (r *repositoryDAOSQLite3Impl) DeleteRepository(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE FROM REPOSITORY
WHERE ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete repository sql: %w", err)
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

func (r *repositoryDAOSQLite3Impl) DeleteAllRepositoriesByUser(ctx context.Context, userID string, device string) (bool, error) {
	sql := `
DELETE FROM REPOSITORY
WHERE USER_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete repository sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repositoryDAOSQLite3Impl) Close(ctx context.Context) error {
	return r.db.Close()
}

func (r *repositoryDAOSQLite3Impl) checkUseToWriteRepositoryCount(ctx context.Context, tx *sql.Tx, userID string) error {
	sql := `
WITH TYPE_AND_DEVICE AS (SELECT ? AS USER_ID)
SELECT 'directory' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE REPOSITORY.USE_TO_WRITE = TRUE
AND TYPE = 'directory'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
UNION
SELECT 'gpslog' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE REPOSITORY.USE_TO_WRITE = TRUE
AND TYPE = 'gpslog'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
UNION
SELECT 'kmemo' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE REPOSITORY.USE_TO_WRITE = TRUE
AND TYPE = 'kmemo'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
UNION
SELECT 'lantana' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE REPOSITORY.USE_TO_WRITE = TRUE
AND TYPE = 'lantana'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
UNION
SELECT 'mi' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE REPOSITORY.USE_TO_WRITE = TRUE
AND TYPE = 'mi'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
UNION
SELECT 'nlog' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE REPOSITORY.USE_TO_WRITE = TRUE
AND TYPE = 'nlog'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
UNION
SELECT 'notification' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE REPOSITORY.USE_TO_WRITE = TRUE
AND TYPE = 'notification'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
UNION
SELECT 'rekyou' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE REPOSITORY.USE_TO_WRITE = TRUE
AND TYPE = 'rekyou'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
UNION
SELECT 'tag' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE REPOSITORY.USE_TO_WRITE = TRUE
AND TYPE = 'tag'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
UNION
SELECT 'text' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE REPOSITORY.USE_TO_WRITE = TRUE
AND TYPE = 'text'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
UNION
SELECT 'timeis' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE REPOSITORY.USE_TO_WRITE = TRUE
AND TYPE = 'timeis'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
UNION
SELECT 'urlog' AS TYPE, DEVICE, COUNT(*) AS COUNT
FROM REPOSITORY
WHERE REPOSITORY.USE_TO_WRITE = TRUE
AND TYPE = 'urlog'
AND USER_ID = (SELECT USER_ID FROM TYPE_AND_DEVICE)
GROUP BY TYPE, DEVICE
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get use to write repository count sql: %w", err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			repType := ""
			device := ""
			count := 0
			err := rows.Scan(
				&repType,
				&device,
				&count,
			)
			if err != nil {
				err = fmt.Errorf("error at get use to write repository count: %w", err)
				err = fmt.Errorf("書き込み先Rep1つに対してがプロファイルに対して1つとなるよ兎にしてください。対象：「%s」", repType)
				return err
			}

			if count >= 2 {
				// err = fmt.Errorf("error at check use to write repository count")
				// err = fmt.Errorf("rep type %s use to write rep count is %d: %w", repType, count, err)
				err = fmt.Errorf("書き込み先Rep1つに対してがプロファイルに対して1つとなるよ兎にしてください。対象：「%s」", repType)
				return err
			}
		}
	}
	return nil
}
