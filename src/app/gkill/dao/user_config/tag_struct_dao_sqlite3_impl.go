package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type tagStructDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewTagStructDAOSQLite3Impl(ctx context.Context, filename string) (TagStructDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
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
  IS_FORCE_HIDE NOT NULL
);`
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TAG_STRUCT table statement %s: %w", filename, err)
		return nil, err
	}

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
  IS_FORCE_HIDE
FROM TAG_STRUCT
`
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all tag struct sql: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}

	tagStructs := []*TagStruct{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tagStruct := &TagStruct{}
			err = rows.Scan(
				tagStruct.ID,
				tagStruct.UserID,
				tagStruct.Device,
				tagStruct.TagName,
				tagStruct.ParentFolderID,
				tagStruct.Seq,
				tagStruct.CheckWhenInited,
				tagStruct.IsForceHide,
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
  IS_FORCE_HIDE
FROM TAG_STRUCT
WHERE USER_ID = ? DEVICE = ?
`
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get tag struct sql: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, userID, device)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}

	tagStructs := []*TagStruct{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tagStruct := &TagStruct{}
			err = rows.Scan(
				tagStruct.ID,
				tagStruct.UserID,
				tagStruct.Device,
				tagStruct.TagName,
				tagStruct.ParentFolderID,
				tagStruct.Seq,
				tagStruct.CheckWhenInited,
				tagStruct.IsForceHide,
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
  IS_FORCE_HIDE
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
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add tag struct sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
		tagStruct.ID,
		tagStruct.UserID,
		tagStruct.Device,
		tagStruct.TagName,
		tagStruct.ParentFolderID,
		tagStruct.Seq,
		tagStruct.CheckWhenInited,
		tagStruct.IsForceHide,
	)
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
  IS_FORCE_HIDE
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
			err = fmt.Errorf("error at add tag struct sql: %w", err)
			return false, err
		}

		_, err = stmt.ExecContext(ctx,
			tagStruct.ID,
			tagStruct.UserID,
			tagStruct.Device,
			tagStruct.TagName,
			tagStruct.ParentFolderID,
			tagStruct.Seq,
			tagStruct.CheckWhenInited,
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
  IS_FORCE_HIDE
WHERE ID = ?
`
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update tag struct sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
		tagStruct.ID,
		tagStruct.UserID,
		tagStruct.Device,
		tagStruct.TagName,
		tagStruct.ParentFolderID,
		tagStruct.Seq,
		tagStruct.CheckWhenInited,
		tagStruct.IsForceHide,
		tagStruct.ID,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (t *tagStructDAOSQLite3Impl) DeleteTagStruct(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE TAG_STRUCT
WHERE ID = ?
`
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete tag struct sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (t *tagStructDAOSQLite3Impl) DeleteUsersTagStructs(ctx context.Context, userID string) (bool, error) {
	sql := `
DELETE TAG_STRUCT
WHERE USER_ID = ?
`
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete tag struct sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx, userID)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (t *tagStructDAOSQLite3Impl) Close(ctx context.Context) error {
	return t.db.Close()
}
