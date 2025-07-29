package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type dnoteDataDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewDnoteDataDAOSQLite3Impl(ctx context.Context, filename string) (DnoteDataDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=1&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "DNOTE_DATA" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  DNOTE_JSON_DATA NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create DNOTE_DATA table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create DNOTE_DATA table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_DNOTE_DATA ON DNOTE_DATA (USER_ID);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create DNOTE_DATA index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create DNOTE_DATA index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create DNOTE_DATA table to %s: %w", filename, err)
		return nil, err
	}

	return &dnoteDataDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (r *dnoteDataDAOSQLite3Impl) GetAllDnoteDatas(ctx context.Context) ([]*DnoteData, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  DNOTE_JSON_DATA
FROM DNOTE_DATA
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all dnote datas sql: %w", err)
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

	dnoteDatas := []*DnoteData{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			dnoteData := &DnoteData{}
			err = rows.Scan(
				&dnoteData.ID,
				&dnoteData.UserID,
				&dnoteData.Device,
				&dnoteData.DnoteJSONData,
			)
			if err != nil {
				err = fmt.Errorf("error at scan dnote data: %w", err)
				return nil, err
			}
			dnoteDatas = append(dnoteDatas, dnoteData)
		}
	}
	return dnoteDatas, nil
}

func (r *dnoteDataDAOSQLite3Impl) GetDnoteData(ctx context.Context, userID string, device string) ([]*DnoteData, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  DNOTE_JSON_DATA
FROM DNOTE_DATA
WHERE USER_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all dnote datas sql: %w", err)
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

	dnoteDatas := []*DnoteData{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			dnoteData := &DnoteData{}
			err = rows.Scan(
				&dnoteData.ID,
				&dnoteData.UserID,
				&dnoteData.Device,
				&dnoteData.DnoteJSONData,
			)
			if err != nil {
				err = fmt.Errorf("error at scan dnote data: %w", err)
				return nil, err
			}
			dnoteDatas = append(dnoteDatas, dnoteData)
		}
	}
	return dnoteDatas, nil
}

func (r *dnoteDataDAOSQLite3Impl) AddDnoteData(ctx context.Context, dnoteData *DnoteData) (bool, error) {
	sql := `
INSERT INTO DNOTE_DATA (
  ID,
  USER_ID,
  DEVICE,
  DNOTE_JSON_DATA
) VALUES (
  ?,
  ?,
  ?,
  ?
)
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add dnote data sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		dnoteData.ID,
		dnoteData.UserID,
		dnoteData.Device,
		dnoteData.DnoteJSONData,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *dnoteDataDAOSQLite3Impl) UpdateDnoteData(ctx context.Context, dnoteData *DnoteData) (bool, error) {
	sql := `
UPDATE DNOTE_DATA SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  DNOTE_DATA = ?
WHERE ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update dnote data sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		dnoteData.ID,
		dnoteData.UserID,
		dnoteData.Device,
		dnoteData.DnoteJSONData,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (r *dnoteDataDAOSQLite3Impl) DeleteDnoteData(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE FROM DNOTE_DATA
WHERE ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete dnote data sql: %w", err)
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

func (r *dnoteDataDAOSQLite3Impl) DeleteUsersDnoteData(ctx context.Context, userID string) (bool, error) {

	sql := `
DELETE FROM DNOTE_DATA
WHERE USER_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete dnote data sql: %w", err)
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

func (r *dnoteDataDAOSQLite3Impl) Close(ctx context.Context) error {
	return r.db.Close()
}
