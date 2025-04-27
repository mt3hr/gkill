package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type tagStructDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewTagStructDAOSQLite3Impl(ctx context.Context, filename string) (TagStructDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=60000&_synchronous=1&_mutex=full&_locking_mode=EXCLUSIVE&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "TAG_STRUCT" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  TAG_NAME NOT NULL,
  PARENT_FOLDER_ID,
  SEQ NOT NULL,
  CHECK_WHEN_INITED NOT NULL,
  IS_FORCE_HIDE NOT NULL,
  IS_DIR NOT NULL,
  IS_OPEN_DEFAULT NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TAG_STRUCT table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_STRUCT table to %s: %w", filename, err)
		return nil, err
	}

	return &tagStructDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (t *tagStructDAOSQLite3Impl) GetAllTagStructs(ctx context.Context) ([]*TagStruct, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  TAG_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED,
  IS_FORCE_HIDE,
  IS_DIR,
  IS_OPEN_DEFAULT
FROM TAG_STRUCT
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all tag struct sql: %w", err)
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

	tagStructs := []*TagStruct{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tagStruct := &TagStruct{}
			err = rows.Scan(
				&tagStruct.ID,
				&tagStruct.UserID,
				&tagStruct.Device,
				&tagStruct.TagName,
				&tagStruct.ParentFolderID,
				&tagStruct.Seq,
				&tagStruct.CheckWhenInited,
				&tagStruct.IsForceHide,
				&tagStruct.IsDir,
				&tagStruct.IsOpenDefault,
			)
			tagStructs = append(tagStructs, tagStruct)
		}
	}
	return tagStructs, nil
}

func (t *tagStructDAOSQLite3Impl) GetTagStructs(ctx context.Context, userID string, device string) ([]*TagStruct, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  TAG_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED,
  IS_FORCE_HIDE,
  IS_DIR,
  IS_OPEN_DEFAULT
FROM TAG_STRUCT
WHERE USER_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get tag struct sql: %w", err)
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

	tagStructs := []*TagStruct{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tagStruct := &TagStruct{}
			err = rows.Scan(
				&tagStruct.ID,
				&tagStruct.UserID,
				&tagStruct.Device,
				&tagStruct.TagName,
				&tagStruct.ParentFolderID,
				&tagStruct.Seq,
				&tagStruct.CheckWhenInited,
				&tagStruct.IsForceHide,
				&tagStruct.IsDir,
				&tagStruct.IsOpenDefault,
			)
			tagStructs = append(tagStructs, tagStruct)
		}
	}
	return tagStructs, nil
}

func (t *tagStructDAOSQLite3Impl) AddTagStruct(ctx context.Context, tagStruct *TagStruct) (bool, error) {
	sql := `
INSERT INTO TAG_STRUCT (
  ID,
  USER_ID,
  DEVICE,
  TAG_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED,
  IS_FORCE_HIDE,
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
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add tag struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		tagStruct.ID,
		tagStruct.UserID,
		tagStruct.Device,
		tagStruct.TagName,
		tagStruct.ParentFolderID,
		tagStruct.Seq,
		tagStruct.CheckWhenInited,
		tagStruct.IsForceHide,
		tagStruct.IsDir,
		tagStruct.IsOpenDefault,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (t *tagStructDAOSQLite3Impl) AddTagStructs(ctx context.Context, tagStructs []*TagStruct) (bool, error) {
	tx, err := t.db.Begin()
	if err != nil {
		fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	for _, tagStruct := range tagStructs {
		sql := `
INSERT INTO TAG_STRUCT (
  ID,
  USER_ID,
  DEVICE,
  TAG_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED,
  IS_FORCE_HIDE,
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
			err = fmt.Errorf("error at add tag struct sql: %w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

		queryArgs := []interface{}{
			tagStruct.ID,
			tagStruct.UserID,
			tagStruct.Device,
			tagStruct.TagName,
			tagStruct.ParentFolderID,
			tagStruct.Seq,
			tagStruct.CheckWhenInited,
			tagStruct.IsForceHide,
			tagStruct.IsDir,
			tagStruct.IsOpenDefault,
		}
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
		_, err = stmt.ExecContext(ctx, queryArgs...)
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

func (t *tagStructDAOSQLite3Impl) UpdateTagStruct(ctx context.Context, tagStruct *TagStruct) (bool, error) {
	sql := `
UPDATE TAG_STRUCT SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  TAG_NAME = ?,
  PARENT_FOLDER_ID = ?,
  SEQ = ?,
  CHECK_WHEN_INITED = ?,
  IS_FORCE_HIDE = ?,
  IS_DIR = ?,
  IS_OPEN_DEFAULT = ?
WHERE ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update tag struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		tagStruct.ID,
		tagStruct.UserID,
		tagStruct.Device,
		tagStruct.TagName,
		tagStruct.ParentFolderID,
		tagStruct.Seq,
		tagStruct.CheckWhenInited,
		tagStruct.IsForceHide,
		tagStruct.IsDir,
		tagStruct.IsOpenDefault,
		tagStruct.ID,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (t *tagStructDAOSQLite3Impl) DeleteTagStruct(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE FROM TAG_STRUCT
WHERE ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete tag struct sql: %w", err)
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

func (t *tagStructDAOSQLite3Impl) DeleteUsersTagStructs(ctx context.Context, userID string) (bool, error) {
	sql := `
DELETE FROM TAG_STRUCT
WHERE USER_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete tag struct sql: %w", err)
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

func (t *tagStructDAOSQLite3Impl) Close(ctx context.Context) error {
	return t.db.Close()
}
