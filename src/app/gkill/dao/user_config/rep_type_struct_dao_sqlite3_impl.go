package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type repTypeStructDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewRepTypeStructDAOSQLite3Impl(ctx context.Context, filename string) (RepTypeStructDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "REP_TYPE_STRUCT" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  REP_TYPE_NAME NOT NULL,
  PARENT_FOLDER_ID,
  SEQ NOT NULL,
  CHECK_WHEN_INITED NOT NULL
);`
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create REP_TYPE_STRUCT table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REP_TYPE_STRUCT table to %s: %w", filename, err)
		return nil, err
	}

	return &repTypeStructDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (r *repTypeStructDAOSQLite3Impl) GetAllRepTypeStructs(ctx context.Context) ([]*RepTypeStruct, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  REP_TYPE_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED
FROM REP_TYPE
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all rep type struct sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	repTypeStructs := []*RepTypeStruct{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			repTypeStruct := &RepTypeStruct{}
			err = rows.Scan(
				&repTypeStruct.ID,
				&repTypeStruct.UserID,
				&repTypeStruct.Device,
				&repTypeStruct.RepTypeName,
				&repTypeStruct.ParentFolderID,
				&repTypeStruct.Seq,
				&repTypeStruct.CheckWhenInited,
			)
			repTypeStructs = append(repTypeStructs, repTypeStruct)
		}
	}
	return repTypeStructs, nil
}

func (r *repTypeStructDAOSQLite3Impl) GetRepTypeStructs(ctx context.Context, userID string, device string) ([]*RepTypeStruct, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  REP_TYPE_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED
FROM REP_TYPE_STRUCT
WHERE USER_ID = ? DEVICE = ?
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get rep type struct sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, userID, device)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	repTypeStructs := []*RepTypeStruct{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			repTypeStruct := &RepTypeStruct{}
			err = rows.Scan(
				&repTypeStruct.ID,
				&repTypeStruct.UserID,
				&repTypeStruct.Device,
				&repTypeStruct.RepTypeName,
				&repTypeStruct.ParentFolderID,
				&repTypeStruct.Seq,
				&repTypeStruct.CheckWhenInited,
			)
			repTypeStructs = append(repTypeStructs, repTypeStruct)
		}
	}
	return repTypeStructs, nil
}

func (r *repTypeStructDAOSQLite3Impl) AddRepTypeStruct(ctx context.Context, repTypeStruct *RepTypeStruct) (bool, error) {
	sql := `
INSERT INTO REP_TYPE_STRUCT (
  ID,
  USER_ID,
  DEVICE,
  REP_TYPE_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED
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
		err = fmt.Errorf("error at add rep type struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		repTypeStruct.ID,
		repTypeStruct.UserID,
		repTypeStruct.Device,
		repTypeStruct.RepTypeName,
		repTypeStruct.ParentFolderID,
		repTypeStruct.Seq,
		repTypeStruct.CheckWhenInited,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repTypeStructDAOSQLite3Impl) AddRepTypeStructs(ctx context.Context, repTypeStructs []*RepTypeStruct) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	for _, repTypeStruct := range repTypeStructs {
		sql := `
INSERT INTO REP_TYPE_STRUCT (
  ID,
  USER_ID,
  DEVICE,
  REP_TYPE_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED
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
			err = fmt.Errorf("error at add rep type struct sql: %w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx,
			repTypeStruct.ID,
			repTypeStruct.UserID,
			repTypeStruct.Device,
			repTypeStruct.RepTypeName,
			repTypeStruct.ParentFolderID,
			repTypeStruct.Seq,
			repTypeStruct.CheckWhenInited,
		)
		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
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

func (r *repTypeStructDAOSQLite3Impl) UpdateRepTypeStruct(ctx context.Context, repTypeStruct *RepTypeStruct) (bool, error) {
	sql := `
UPDATE REP_TYPE_STRUCT SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  REP_TYPE_NAME = ?,
  PARENT_FOLDER_ID = ?,
  SEQ = ?,
  CHECK_WHEN_INITED = ?
WHERE ID = ?
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update rep type struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		repTypeStruct.ID,
		repTypeStruct.UserID,
		repTypeStruct.Device,
		repTypeStruct.RepTypeName,
		repTypeStruct.ParentFolderID,
		repTypeStruct.Seq,
		repTypeStruct.CheckWhenInited,
		repTypeStruct.ID,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repTypeStructDAOSQLite3Impl) DeleteRepTypeStruct(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE REP_TYPE_STRUCT
WHERE ID = ?
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete rep type struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repTypeStructDAOSQLite3Impl) DeleteUsersRepTypeStructs(ctx context.Context, userID string) (bool, error) {
	sql := `
DELETE REP_TYPE_STRUCT
WHERE USER_ID = ?
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete rep type struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, userID)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repTypeStructDAOSQLite3Impl) Close(ctx context.Context) error {
	return r.db.Close()
}
