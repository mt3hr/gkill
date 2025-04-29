package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type repTypeStructDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewRepTypeStructDAOSQLite3Impl(ctx context.Context, filename string) (RepTypeStructDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=2&_journal=DELETE")
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
  CHECK_WHEN_INITED NOT NULL,
  IS_DIR NOT NULL,
  IS_OPEN_DEFAULT NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create REP_TYPE_STRUCT table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REP_TYPE_STRUCT table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_REP_TYPE_STRUCT ON REP_TYPE_STRUCT (USER_ID);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create REP_TYPE_STRUCT index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REP_TYPE_STRUCT index to %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
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
  CHECK_WHEN_INITED,
  IS_DIR,
  IS_OPEN_DEFAULT
FROM REP_TYPE
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all rep type struct sql: %w", err)
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
				&repTypeStruct.IsDir,
				&repTypeStruct.IsOpenDefault,
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
  CHECK_WHEN_INITED,
  IS_DIR,
  IS_OPEN_DEFAULT
FROM REP_TYPE_STRUCT
WHERE USER_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get rep type struct sql: %w", err)
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
				&repTypeStruct.IsDir,
				&repTypeStruct.IsOpenDefault,
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
  CHECK_WHEN_INITED,
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
  ?
)
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add rep type struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		repTypeStruct.ID,
		repTypeStruct.UserID,
		repTypeStruct.Device,
		repTypeStruct.RepTypeName,
		repTypeStruct.ParentFolderID,
		repTypeStruct.Seq,
		repTypeStruct.CheckWhenInited,
		repTypeStruct.IsDir,
		repTypeStruct.IsOpenDefault,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

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
  CHECK_WHEN_INITED,
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
  ?
)
`
		gkill_log.TraceSQL.Printf("sql: %s", sql)
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

		queryArgs := []interface{}{
			repTypeStruct.ID,
			repTypeStruct.UserID,
			repTypeStruct.Device,
			repTypeStruct.RepTypeName,
			repTypeStruct.ParentFolderID,
			repTypeStruct.Seq,
			repTypeStruct.CheckWhenInited,
			repTypeStruct.IsDir,
			repTypeStruct.IsOpenDefault,
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
  CHECK_WHEN_INITED = ?,
  IS_DIR = ?,
  IS_OPEN_DEFAULT = ?
WHERE ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update rep type struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		repTypeStruct.ID,
		repTypeStruct.UserID,
		repTypeStruct.Device,
		repTypeStruct.RepTypeName,
		repTypeStruct.ParentFolderID,
		repTypeStruct.Seq,
		repTypeStruct.CheckWhenInited,
		repTypeStruct.IsDir,
		repTypeStruct.IsOpenDefault,
		repTypeStruct.ID,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *repTypeStructDAOSQLite3Impl) DeleteRepTypeStruct(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE FROM REP_TYPE_STRUCT
WHERE ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete rep type struct sql: %w", err)
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

func (r *repTypeStructDAOSQLite3Impl) DeleteUsersRepTypeStructs(ctx context.Context, userID string) (bool, error) {
	sql := `
DELETE FROM REP_TYPE_STRUCT
WHERE USER_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete rep type struct sql: %w", err)
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

func (r *repTypeStructDAOSQLite3Impl) Close(ctx context.Context) error {
	return r.db.Close()
}
