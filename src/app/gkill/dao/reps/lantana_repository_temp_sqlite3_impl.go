package reps

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type lantanaTempRepositorySQLite3Impl lantanaRepositorySQLite3Impl

func NewLantanaTempRepositorySQLite3Impl(ctx context.Context, db *sql.DB) (LantanaTempRepository, error) {
	filename := "lantana_temp"
	sql := `
CREATE TABLE IF NOT EXISTS "LANTANA" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  MOOD NOT NULL,
  RELATED_TIME NOT NULL,
  CREATE_TIME NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_TIME NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  TX_ID NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create LANTANA table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create LANTANA table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_LANTANA ON LANTANA (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create LANTANA index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create LANTANA index to %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create LANTANA table to %s: %w", filename, err)
		return nil, err
	}

	return &lantanaTempRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (l *lantanaTempRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	impl := lantanaRepositorySQLite3Impl(*l)
	return impl.FindKyous(ctx, query)
}

func (l *lantanaTempRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	impl := lantanaRepositorySQLite3Impl(*l)
	return impl.GetKyou(ctx, id, updateTime)
}

func (l *lantanaTempRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	impl := lantanaRepositorySQLite3Impl(*l)
	return impl.GetKyouHistories(ctx, id)
}

func (l *lantanaTempRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (l *lantanaTempRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	impl := lantanaRepositorySQLite3Impl(*l)
	return impl.UpdateCache(ctx)
}

func (l *lantanaTempRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "lantana_temp", nil
}

func (l *lantanaTempRepositorySQLite3Impl) Close(ctx context.Context) error {
	impl := lantanaRepositorySQLite3Impl(*l)
	return impl.Close(ctx)
}

func (l *lantanaTempRepositorySQLite3Impl) FindLantana(ctx context.Context, query *find.FindQuery) ([]*Lantana, error) {
	impl := lantanaRepositorySQLite3Impl(*l)
	return impl.FindLantana(ctx, query)
}

func (l *lantanaTempRepositorySQLite3Impl) GetLantana(ctx context.Context, id string, updateTime *time.Time) (*Lantana, error) {
	impl := lantanaRepositorySQLite3Impl(*l)
	return impl.GetLantana(ctx, id, updateTime)
}

func (l *lantanaTempRepositorySQLite3Impl) GetLantanaHistories(ctx context.Context, id string) ([]*Lantana, error) {
	impl := lantanaRepositorySQLite3Impl(*l)
	return impl.GetLantanaHistories(ctx, id)
}

func (l *lantanaTempRepositorySQLite3Impl) AddLantanaInfo(ctx context.Context, lantana *Lantana, txID string, userID string, device string) error {
	sql := `
INSERT INTO LANTANA (
  IS_DELETED,
  ID,
  MOOD,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  USER_ID,
  DEVICE,
  TX_ID
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
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add lantana sql %s: %w", lantana.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		lantana.IsDeleted,
		lantana.ID,
		lantana.Mood,
		lantana.RelatedTime.Format(sqlite3impl.TimeLayout),
		lantana.CreateTime.Format(sqlite3impl.TimeLayout),
		lantana.CreateApp,
		lantana.CreateDevice,
		lantana.CreateUser,
		lantana.UpdateTime.Format(sqlite3impl.TimeLayout),
		lantana.UpdateApp,
		lantana.UpdateDevice,
		lantana.UpdateUser,
		userID,
		device,
		txID,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to LANTANA %s: %w", lantana.ID, err)
		return err
	}
	return nil
}

func (l *lantanaTempRepositorySQLite3Impl) GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error) {
	var err error
	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM LANTANA
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`

	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana temp: %w", err)
		return nil, err
	}

	dataType := "lantana"
	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyous by TXID sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from lantana temp: %w", err)
		return nil, err
	}
	defer rows.Close()

	kyous := []*Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := &Kyou{}
			kyou.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			targetRepName := ""

			err = rows.Scan(
				&kyou.IsDeleted,
				&kyou.ID,
				&targetRepName,
				&relatedTimeStr,
				&createTimeStr,
				&kyou.CreateApp,
				&kyou.CreateDevice,
				&kyou.CreateUser,
				&updateTimeStr,
				&kyou.UpdateApp,
				&kyou.UpdateDevice,
				&kyou.UpdateUser,
				&kyou.RepName,
				&kyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from lantana temp: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in lantana temp: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in lantana temp: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in lantana temp: %w", updateTimeStr, err)
				return nil, err
			}

			kyous = append(kyous, kyou)
		}
	}
	sort.Slice(kyous, func(i, j int) bool {
		return kyous[i].UpdateTime.After(kyous[j].UpdateTime)
	})
	return kyous, nil
}

func (l *lantanaTempRepositorySQLite3Impl) GetLantanasByTXID(ctx context.Context, txID string, userID string, device string) ([]*Lantana, error) {
	sql := `
SELECT 
  IS_DELETED,
  ID,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  MOOD,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM LANTANA
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
		return nil, err
	}

	dataType := "lantana"

	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get lantana by txid sql %s: %w", txID, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query ")
		return nil, err
	}
	defer rows.Close()

	lantanas := []*Lantana{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			lantana := &Lantana{}
			lantana.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(&lantana.IsDeleted,
				&lantana.ID,
				&relatedTimeStr,
				&createTimeStr,
				&lantana.CreateApp,
				&lantana.CreateDevice,
				&lantana.CreateUser,
				&updateTimeStr,
				&lantana.UpdateApp,
				&lantana.UpdateDevice,
				&lantana.UpdateUser,
				&lantana.Mood,
				&lantana.RepName,
				&lantana.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from LANTANA: %w", err)
				return nil, err
			}

			lantana.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in LANTANA: %w", relatedTimeStr, txID, err)
				return nil, err
			}
			lantana.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in LANTANA: %w", createTimeStr, txID, err)
				return nil, err
			}
			lantana.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in LANTANA: %w", updateTimeStr, txID, err)
				return nil, err
			}
			lantanas = append(lantanas, lantana)
		}
	}
	return lantanas, nil
}

func (l *lantanaTempRepositorySQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	sql := `
DELETE FROM LANTANA
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete temp lantana kyou by TXID sql: %w", err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		txID,
		userID,
		device,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at delete temp lantana kyou by TXID sql: %w", err)
		return err
	}
	return nil
}
