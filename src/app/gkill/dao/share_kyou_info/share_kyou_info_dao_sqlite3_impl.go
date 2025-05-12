package share_kyou_info

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type shareKyouInfoDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewShareKyouInfoDAOSQLite3Impl(ctx context.Context, filename string) (ShareKyouInfoDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=1&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "SHARE_KYOU_INFO" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  SHARE_TITLE NOT NULL,
  IS_SHARE_DETAIL NOT NULL,
  SHARE_ID NOT NULL,
  FIND_QUERY_JSON NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create SHARE_KYOU_INFO table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create SHARE_KYOU_INFO table to %s: %w", filename, err)
		return nil, err
	}

	return &shareKyouInfoDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (m *shareKyouInfoDAOSQLite3Impl) GetAllKyouShareInfos(ctx context.Context) ([]*ShareKyouInfo, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  SHARE_TITLE,
  IS_SHARE_DETAIL,
  SHARE_ID,
  FIND_QUERY_JSON
FROM SHARE_KYOU_INFO
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all kyou share infos sql: %w", err)
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

	kyouShareInfos := []*ShareKyouInfo{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyouShareInfo := &ShareKyouInfo{}
			err = rows.Scan(
				&kyouShareInfo.ID,
				&kyouShareInfo.UserID,
				&kyouShareInfo.Device,
				&kyouShareInfo.ShareTitle,
				&kyouShareInfo.IsShareDetail,
				&kyouShareInfo.ShareID,
				&kyouShareInfo.FindQueryJSON,
			)
			err = fmt.Errorf("error at scan kyou share info: %w", err)
			if err != nil {
				return nil, err
			}
			kyouShareInfos = append(kyouShareInfos, kyouShareInfo)
		}
	}
	return kyouShareInfos, nil
}

func (m *shareKyouInfoDAOSQLite3Impl) GetKyouShareInfos(ctx context.Context, userID string, device string) ([]*ShareKyouInfo, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  SHARE_TITLE,
  IS_SHARE_DETAIL,
  SHARE_ID,
  FIND_QUERY_JSON
FROM SHARE_KYOU_INFO
WHERE USER_ID = ? AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get kyou share infos sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
		device,
	}
	gkill_log.TraceSQL.Printf("%#v", queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	kyouShareInfos := []*ShareKyouInfo{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyouShareInfo := &ShareKyouInfo{}
			err = rows.Scan(
				&kyouShareInfo.ID,
				&kyouShareInfo.UserID,
				&kyouShareInfo.Device,
				&kyouShareInfo.ShareTitle,
				&kyouShareInfo.IsShareDetail,
				&kyouShareInfo.ShareID,
				&kyouShareInfo.FindQueryJSON,
			)
			if err != nil {
				err = fmt.Errorf("error at scan kyou share info: %w", err)
				return nil, err
			}
			kyouShareInfos = append(kyouShareInfos, kyouShareInfo)
		}
	}
	return kyouShareInfos, nil
}

func (m *shareKyouInfoDAOSQLite3Impl) GetKyouShareInfo(ctx context.Context, sharedID string) (*ShareKyouInfo, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  SHARE_TITLE,
  IS_SHARE_DETAIL,
  SHARE_ID,
  FIND_QUERY_JSON
FROM SHARE_KYOU_INFO
WHERE SHARE_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get kyou share infos sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		sharedID,
	}
	gkill_log.TraceSQL.Printf("%#v", queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	kyouShareInfos := []*ShareKyouInfo{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyouShareInfo := &ShareKyouInfo{}
			err = rows.Scan(
				&kyouShareInfo.ID,
				&kyouShareInfo.UserID,
				&kyouShareInfo.Device,
				&kyouShareInfo.ShareTitle,
				&kyouShareInfo.IsShareDetail,
				&kyouShareInfo.ShareID,
				&kyouShareInfo.FindQueryJSON,
			)
			if err != nil {
				err = fmt.Errorf("error at scan kyou share info: %w", err)
				return nil, err
			}
			kyouShareInfos = append(kyouShareInfos, kyouShareInfo)
		}
	}
	if len(kyouShareInfos) == 0 {
		return nil, nil
	}
	return kyouShareInfos[0], nil
}

func (m *shareKyouInfoDAOSQLite3Impl) AddKyouShareInfo(ctx context.Context, kyouShareInfo *ShareKyouInfo) (bool, error) {
	sql := `
INSERT INTO SHARE_KYOU_INFO (
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
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add kyou share info sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		kyouShareInfo.ID,
		kyouShareInfo.UserID,
		kyouShareInfo.Device,
		kyouShareInfo.ShareTitle,
		kyouShareInfo.IsShareDetail,
		kyouShareInfo.ShareID,
		kyouShareInfo.FindQueryJSON,
	}
	gkill_log.TraceSQL.Printf("%#v", queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (m *shareKyouInfoDAOSQLite3Impl) UpdateKyouShareInfo(ctx context.Context, kyouShareInfo *ShareKyouInfo) (bool, error) {
	sql := `
UPDATE SHARE_KYOU_INFO SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  SHARE_TITLE = ?,
  IS_SHARE_DETAIL = ?,
  SHARE_ID = ?,
  FIND_QUERY_JSON = ?
WHERE SHARE_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update kyou share info sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		kyouShareInfo.ID,
		kyouShareInfo.UserID,
		kyouShareInfo.Device,
		kyouShareInfo.ShareTitle,
		kyouShareInfo.IsShareDetail,
		kyouShareInfo.ShareID,
		kyouShareInfo.FindQueryJSON,
		kyouShareInfo.ShareID,
	}
	gkill_log.TraceSQL.Printf("%#v", queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (m *shareKyouInfoDAOSQLite3Impl) DeleteKyouShareInfo(ctx context.Context, shareID string) (bool, error) {
	sql := `
DELETE FROM SHARE_KYOU_INFO
WHERE SHARE_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete kyou share info sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		shareID,
	}
	gkill_log.TraceSQL.Printf("%#v", queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (m *shareKyouInfoDAOSQLite3Impl) Close(ctx context.Context) error {
	return m.db.Close()
}
