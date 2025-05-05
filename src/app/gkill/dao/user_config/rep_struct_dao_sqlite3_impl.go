package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type repStructDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewRepStructDAOSQLite3Impl(ctx context.Context, filename string) (RepStructDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=1&_journal=DELETE")
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
  IGNORE_CHECK_REP_RYKV NOT NULL,
  IS_DIR NOT NULL,
  IS_OPEN_DEFAULT NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create REP_STRUCT table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REP_STRUCT table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_REP_STRUCT ON REP_STRUCT (USER_ID);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create REP_STRUCT index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REP_STRUCT index to %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
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
  IGNORE_CHECK_REP_RYKV,
  IS_DIR,
  IS_OPEN_DEFAULT
FROM REP_STRUCT
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all rep struct sql: %w", err)
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

	repStructs := []*RepStruct{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			repStruct := &RepStruct{}
			err = rows.Scan(
				&repStruct.ID,
				&repStruct.UserID,
				&repStruct.Device,
				&repStruct.RepName,
				&repStruct.ParentFolderID,
				&repStruct.Seq,
				&repStruct.CheckWhenInited,
				&repStruct.IgnoreCheckRepRykv,
				&repStruct.IsDir,
				&repStruct.IsOpenDefault,
			)
			if err != nil {
				err = fmt.Errorf("error at scan rep struct: %w", err)
				return nil, err
			}
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
  IGNORE_CHECK_REP_RYKV,
  IS_DIR,
  IS_OPEN_DEFAULT
FROM REP_STRUCT
WHERE USER_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get rep struct sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	repStructs := []*RepStruct{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			repStruct := &RepStruct{}
			err = rows.Scan(
				&repStruct.ID,
				&repStruct.UserID,
				&repStruct.Device,
				&repStruct.RepName,
				&repStruct.ParentFolderID,
				&repStruct.Seq,
				&repStruct.CheckWhenInited,
				&repStruct.IgnoreCheckRepRykv,
				&repStruct.IsDir,
				&repStruct.IsOpenDefault,
			)
			if err != nil {
				err = fmt.Errorf("error at scan rep struct: %w", err)
				return nil, err
			}
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
  IGNORE_CHECK_REP_RYKV,
  IS_DIR,
  IS_OPEN_DEFAULT
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
  ?
)
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add rep struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		repStruct.ID,
		repStruct.UserID,
		repStruct.Device,
		repStruct.RepName,
		repStruct.ParentFolderID,
		repStruct.Seq,
		repStruct.CheckWhenInited,
		repStruct.IgnoreCheckRepRykv,
		repStruct.IsDir,
		repStruct.IsOpenDefault,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repStructDAOSQLite3Impl) AddRepStructs(ctx context.Context, repStructs []*RepStruct) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
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
  IGNORE_CHECK_REP_RYKV,
  IS_DIR,
  IS_OPEN_DEFAULT
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
  ?
)
`
		gkill_log.TraceSQL.Printf("sql: %s", sql)
		stmt, err := tx.PrepareContext(ctx, sql)
		if err != nil {
			err = fmt.Errorf("error at add rep struct sql: %w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

		queryArgs := []interface{}{
			repStruct.ID,
			repStruct.UserID,
			repStruct.Device,
			repStruct.RepName,
			repStruct.ParentFolderID,
			repStruct.Seq,
			repStruct.CheckWhenInited,
			repStruct.IgnoreCheckRepRykv,
			repStruct.IsDir,
			repStruct.IsOpenDefault,
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

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
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
  IGNORE_CHECK_REP_RYKV = ?,
  IS_DIR = ?,
  IS_OPEN_DEFAULT = ?
WHERE ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update rep struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		repStruct.ID,
		repStruct.UserID,
		repStruct.Device,
		repStruct.RepName,
		repStruct.ParentFolderID,
		repStruct.Seq,
		repStruct.CheckWhenInited,
		repStruct.IgnoreCheckRepRykv,
		repStruct.ID,
		repStruct.IsDir,
		repStruct.IsOpenDefault,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repStructDAOSQLite3Impl) DeleteRepStruct(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE FROM REP_STRUCT
WHERE ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete rep struct sql: %w", err)
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

func (r *repStructDAOSQLite3Impl) DeleteUsersRepStructs(ctx context.Context, userID string) (bool, error) {

	sql := `
DELETE FROM REP_STRUCT
WHERE USER_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete rep struct sql: %w", err)
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

func (r *repStructDAOSQLite3Impl) Close(ctx context.Context) error {
	return r.db.Close()
}
