package mi_share_info

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type miShareInfoDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewMiShareInfoDAOSQLite3Impl(ctx context.Context, filename string) (MiShareInfoDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "MI_SHARE_INFO" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  SHARE_TITLE NOT NULL,
  IS_SHARE_DETAIL NOT NULL,
  SHARE_ID NOT NULL,
  FIND_QUERY_JSON NOT NULL
);`
	log.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create MI_SHARE_INFO table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	log.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MI_SHARE_INFO table to %s: %w", filename, err)
		return nil, err
	}

	return &miShareInfoDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (m *miShareInfoDAOSQLite3Impl) GetAllMiShareInfos(ctx context.Context) ([]*MiShareInfo, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  SHARE_TITLE,
  IS_SHARE_DETAIL,
  SHARE_ID,
  FIND_QUERY_JSON
FROM MI_SHARE_INFO
`
	log.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all mi share infos sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	log.Printf("sql: %s", sql)
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	miShareInfos := []*MiShareInfo{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			miShareInfo := &MiShareInfo{}
			err = rows.Scan(
				&miShareInfo.ID,
				&miShareInfo.UserID,
				&miShareInfo.Device,
				&miShareInfo.ShareTitle,
				&miShareInfo.IsShareDetail,
				&miShareInfo.ShareID,
				&miShareInfo.FindQueryJSON,
			)
			miShareInfos = append(miShareInfos, miShareInfo)
		}
	}
	return miShareInfos, nil
}

func (m *miShareInfoDAOSQLite3Impl) GetMiShareInfos(ctx context.Context, userID string, device string) ([]*MiShareInfo, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  SHARE_TITLE,
  IS_SHARE_DETAIL,
  SHARE_ID,
  FIND_QUERY_JSON
FROM MI_SHARE_INFO
WHERE USER_ID = ? AND DEVICE = ?
`
	log.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get mi share infos sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
		device,
	}
	log.Printf("%#v", queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	miShareInfos := []*MiShareInfo{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			miShareInfo := &MiShareInfo{}
			err = rows.Scan(
				&miShareInfo.ID,
				&miShareInfo.UserID,
				&miShareInfo.Device,
				&miShareInfo.ShareTitle,
				&miShareInfo.IsShareDetail,
				&miShareInfo.ShareID,
				&miShareInfo.FindQueryJSON,
			)
			miShareInfos = append(miShareInfos, miShareInfo)
		}
	}
	return miShareInfos, nil
}

func (m *miShareInfoDAOSQLite3Impl) GetMiShareInfo(ctx context.Context, sharedID string) (*MiShareInfo, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  SHARE_TITLE,
  IS_SHARE_DETAIL,
  SHARE_ID,
  FIND_QUERY_JSON
FROM MI_SHARE_INFO
WHERE SHARED_ID = ?
`
	log.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get mi share infos sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		sharedID,
	}
	log.Printf("%#v", queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	miShareInfos := []*MiShareInfo{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			miShareInfo := &MiShareInfo{}
			err = rows.Scan(
				&miShareInfo.ID,
				&miShareInfo.UserID,
				&miShareInfo.Device,
				&miShareInfo.ShareTitle,
				&miShareInfo.IsShareDetail,
				&miShareInfo.ShareID,
				&miShareInfo.FindQueryJSON,
			)
			miShareInfos = append(miShareInfos, miShareInfo)
		}
	}
	if len(miShareInfos) == 0 {
		return nil, nil
	}
	return miShareInfos[0], nil
}

func (m *miShareInfoDAOSQLite3Impl) AddMiShareInfo(ctx context.Context, miShareInfo *MiShareInfo) (bool, error) {
	sql := `
INSERT INTO MI_SHARE_INFO (
  ID,
  USER_ID,
  DEVICE,
  SHARE_TITLE,
  IS_SHARE_DETAIL,
  SHARE_ID,
  FIND_QUERY_JSON
) VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)
`
	log.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add mi share info sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		miShareInfo.ID,
		miShareInfo.UserID,
		miShareInfo.Device,
		miShareInfo.ShareTitle,
		miShareInfo.IsShareDetail,
		miShareInfo.ShareID,
		miShareInfo.FindQueryJSON,
	}
	log.Printf("%#v", queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (m *miShareInfoDAOSQLite3Impl) UpdateMiShareInfo(ctx context.Context, miShareInfo *MiShareInfo) (bool, error) {
	sql := `
UPDATE MI_SHARE_INFO SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  SHARE_TITLE = ?,
  IS_SHARE_DETAIL = ?,
  SHARE_ID = ?,
  FIND_QUERY_JSON = ?
WHERE ID = ?
`
	log.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update mi share info sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		miShareInfo.ID,
		miShareInfo.UserID,
		miShareInfo.Device,
		miShareInfo.ShareTitle,
		miShareInfo.IsShareDetail,
		miShareInfo.ShareID,
		miShareInfo.FindQueryJSON,
		miShareInfo.ID,
	}
	log.Printf("%#v", queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (m *miShareInfoDAOSQLite3Impl) DeleteMiShareInfo(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE FROM MI_SHARE_INFO
WHERE ID = ?
`
	log.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete mi share info sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		id,
	}
	log.Printf("%#v", queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (m *miShareInfoDAOSQLite3Impl) Close(ctx context.Context) error {
	return m.db.Close()
}
