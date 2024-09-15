package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type repStructDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewRepStructDAOSQLite3Impl(ctx context.Context, filename string) (RepStructDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "REP_STRUCT" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  REP_NAME NOT NULL,
  PARENT_FOLDER_ID,
  SEQ NOT NULL,
  CHECK_WHEN_INITED NOT NULL,
  IGNORE_CHECK_REP_RYKV NOT NULL
);`
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create REP_STRUCT table statement %s: %w", filename, err)
		return nil, err
	}

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REP_STRUCT table to %s: %w", filename, err)
		return nil, err
	}

	return &repStructDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (r *repStructDAOSQLite3Impl) GetAllRepStructs(ctx context.Context) ([]*RepStruct, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  REP_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED,
  IGNORE_CHECK_REP_RYKV
FROM REP_STRUCT
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all rep struct sql: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}

	repStructs := []*RepStruct{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			repStruct := &RepStruct{}
			err = rows.Scan(
				repStruct.ID,
				repStruct.UserID,
				repStruct.Device,
				repStruct.RepName,
				repStruct.ParentFolderID,
				repStruct.Seq,
				repStruct.CheckWhenInited,
				repStruct.IgnoreCheckRepRykv,
			)
			repStructs = append(repStructs, repStruct)
		}
	}
	return repStructs, nil
}

func (r *repStructDAOSQLite3Impl) GetRepStructs(ctx context.Context, userID string, device string) ([]*RepStruct, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  REP_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED,
  IGNORE_CHECK_REP_RYKV
FROM REP_STRUCT
WHERE USER_ID = ? DEVICE = ?
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get rep struct sql: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, userID, device)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}

	repStructs := []*RepStruct{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			repStruct := &RepStruct{}
			err = rows.Scan(
				repStruct.ID,
				repStruct.UserID,
				repStruct.Device,
				repStruct.RepName,
				repStruct.ParentFolderID,
				repStruct.Seq,
				repStruct.CheckWhenInited,
				repStruct.IgnoreCheckRepRykv,
			)
			repStructs = append(repStructs, repStruct)
		}
	}
	return repStructs, nil
}

func (r *repStructDAOSQLite3Impl) AddRepStruct(ctx context.Context, repStruct *RepStruct) (bool, error) {
	sql := `
INSERT INTO REP_STRUCT (
  ID,
  USER_ID,
  DEVICE,
  REP_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED,
  IGNORE_CHECK_REP_RYKV
)
VALUES (
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
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add rep struct sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
		repStruct.ID,
		repStruct.UserID,
		repStruct.Device,
		repStruct.RepName,
		repStruct.ParentFolderID,
		repStruct.Seq,
		repStruct.CheckWhenInited,
		repStruct.IgnoreCheckRepRykv,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repStructDAOSQLite3Impl) AddRepStructs(ctx context.Context, repStructs []*RepStruct) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	for _, repStruct := range repStructs {
		sql := `
INSERT INTO REP_STRUCT (
  ID,
  USER_ID,
  DEVICE,
  REP_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED,
  IGNORE_CHECK_REP_RYKV
)
VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
);
`
		stmt, err := tx.PrepareContext(ctx, sql)
		if err != nil {
			err = fmt.Errorf("error at add rep struct sql: %w", err)
			return false, err
		}

		_, err = stmt.ExecContext(ctx,
			repStruct.ID,
			repStruct.UserID,
			repStruct.Device,
			repStruct.RepName,
			repStruct.ParentFolderID,
			repStruct.Seq,
			repStruct.CheckWhenInited,
			repStruct.IgnoreCheckRepRykv,
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

func (r *repStructDAOSQLite3Impl) UpdateRepStruct(ctx context.Context, repStruct *RepStruct) (bool, error) {
	sql := `
UPDATE REP_STRUCT SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  REP_NAME = ?,
  PARENT_FOLDER_ID = ?,
  SEQ = ?,
  CHECK_WHEN_INITED = ?,
  IGNORE_CHECK_REP_RYKV = ?
WHERE ID = ?
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update rep struct sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
		repStruct.ID,
		repStruct.UserID,
		repStruct.Device,
		repStruct.RepName,
		repStruct.ParentFolderID,
		repStruct.Seq,
		repStruct.CheckWhenInited,
		repStruct.IgnoreCheckRepRykv,
		repStruct.ID,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repStructDAOSQLite3Impl) DeleteRepStruct(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE REP_STRUCT
WHERE ID = ?
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete rep struct sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repStructDAOSQLite3Impl) DeleteUsersRepStructs(ctx context.Context, userID string) (bool, error) {

	sql := `
DELETE REP_STRUCT
WHERE USER_ID = ?
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete rep struct sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx, userID)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repStructDAOSQLite3Impl) Close(ctx context.Context) error {
	return r.db.Close()
}
