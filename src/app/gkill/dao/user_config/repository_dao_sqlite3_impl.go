// ˅
package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

// ˄

type repositoryDAOSQLite3Impl struct {
	// ˅
	filename string
	db       *sql.DB
	m        *sync.Mutex
	// ˄
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
  FILE NOT NULL,
  USE_TO_WRITE NOT NULL,
  IS_EXECUTE_IDF_WHEN_RELOAD NOT NULL,
  IS_ENABLE NOT NULL
);`
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create REPOSITORY table statement %s: %w", filename, err)
		return nil, err
	}

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

// ˅
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
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all repositories sql: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}

	repositories := []*Repository{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			repository := &Repository{}
			err = rows.Scan(
				repository.ID,
				repository.UserID,
				repository.Device,
				repository.Type,
				repository.File,
				repository.UseToWrite,
				repository.IsExecuteIDFWhenReload,
				repository.IsEnable,
			)
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
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get repositories sql: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, userID, device)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}

	repositories := []*Repository{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			repository := &Repository{}
			err = rows.Scan(
				repository.ID,
				repository.UserID,
				repository.Device,
				repository.Type,
				repository.File,
				repository.UseToWrite,
				repository.IsExecuteIDFWhenReload,
				repository.IsEnable,
			)
			repositories = append(repositories, repository)
		}
	}
	return repositories, nil
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
)
VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add repository sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
		repository.ID,
		repository.UserID,
		repository.Device,
		repository.Type,
		repository.File,
		repository.UseToWrite,
		repository.IsExecuteIDFWhenReload,
		repository.IsEnable,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
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
)
VALUES (
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

		_, err = stmt.ExecContext(ctx,
			repository.ID,
			repository.UserID,
			repository.Device,
			repository.Type,
			repository.File,
			repository.UseToWrite,
			repository.IsExecuteIDFWhenReload,
			repository.IsEnable,
		)
		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			return false, err
		}
	}
	err = tx.Commit()
	if err != nil {
		fmt.Errorf("error at commit: %w", err)
		return false, err
	}

	return true, nil
}

func (r *repositoryDAOSQLite3Impl) UpdateRepository(ctx context.Context, repository *Repository) (bool, error) {
	sql := `
UPDATE DEVICE_STRUCT SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  DEVICE_NAME = ?,
  PARENT_FOLDER_ID = ?,
  SEQ = ?,
  CHECK_WHEN_INITED = ?
WHERE ID = ?
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update repository sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx, ctx,
		repository.ID,
		repository.UserID,
		repository.Device,
		repository.Type,
		repository.File,
		repository.UseToWrite,
		repository.IsExecuteIDFWhenReload,
		repository.IsEnable,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repositoryDAOSQLite3Impl) DeleteRepository(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE REPOSITORY
WHERE ID = ?
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete repository sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repositoryDAOSQLite3Impl) DeleteAllRepositoriesByUser(ctx context.Context, userID string, device string) (bool, error) {
	sql := `
DELETE REPOSITORY
WHERE USER_ID = ?
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete repository sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx, userID)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

// ˄
