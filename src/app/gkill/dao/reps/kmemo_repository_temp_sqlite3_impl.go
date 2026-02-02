package reps

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type kmemoTempRepositorySQLite3Impl kmemoRepositorySQLite3Impl

func NewKmemoTempRepositorySQLite3Impl(ctx context.Context, db *sql.DB, m *sync.Mutex) (KmemoTempRepository, error) {
	filename := "kmemo_temp"
	sql := `
CREATE TABLE IF NOT EXISTS "KMEMO" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  CONTENT NOT NULL,
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
		err = fmt.Errorf("error at create KMEMO table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create KMEMO table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_KMEMO ON KMEMO (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create KMEMO index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create KMEMO index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create KMEMO table to %s: %w", filename, err)
		return nil, err
	}

	return &kmemoTempRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        m,
	}, nil
}

func (k *kmemoTempRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	impl := kmemoRepositorySQLite3Impl(*k)
	return impl.FindKyous(ctx, query)
}

func (k *kmemoTempRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	impl := kmemoRepositorySQLite3Impl(*k)
	return impl.GetKyou(ctx, id, updateTime)
}

func (k *kmemoTempRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	impl := kmemoRepositorySQLite3Impl(*k)
	return impl.GetKyouHistories(ctx, id)
}

func (k *kmemoTempRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("kmemoTempRepositorySQLite3Impl does not support GetPath")
}

func (k *kmemoTempRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	impl := kmemoRepositorySQLite3Impl(*k)
	return impl.UpdateCache(ctx)
}

func (k *kmemoTempRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "KmemoTemp", nil
}

func (k *kmemoTempRepositorySQLite3Impl) Close(ctx context.Context) error {
	impl := kmemoRepositorySQLite3Impl(*k)
	return impl.Close(ctx)
}

func (k *kmemoTempRepositorySQLite3Impl) FindKmemo(ctx context.Context, query *find.FindQuery) ([]*Kmemo, error) {
	impl := kmemoRepositorySQLite3Impl(*k)
	return impl.FindKmemo(ctx, query)
}

func (k *kmemoTempRepositorySQLite3Impl) GetKmemo(ctx context.Context, id string, updateTime *time.Time) (*Kmemo, error) {
	impl := kmemoRepositorySQLite3Impl(*k)
	return impl.GetKmemo(ctx, id, updateTime)
}

func (k *kmemoTempRepositorySQLite3Impl) GetKmemoHistories(ctx context.Context, id string) ([]*Kmemo, error) {
	impl := kmemoRepositorySQLite3Impl(*k)
	return impl.GetKmemoHistories(ctx, id)
}

func (k *kmemoTempRepositorySQLite3Impl) AddKmemoInfo(ctx context.Context, kmemo *Kmemo, txID string, userID string, device string) error {
	k.m.Lock()
	defer k.m.Unlock()
	sql := `
INSERT INTO KMEMO (
  IS_DELETED,
  ID,
  CONTENT,
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
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add kmemo sql %s: %w", kmemo.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		kmemo.IsDeleted,
		kmemo.ID,
		kmemo.Content,
		kmemo.RelatedTime.Format(sqlite3impl.TimeLayout),
		kmemo.CreateTime.Format(sqlite3impl.TimeLayout),
		kmemo.CreateApp,
		kmemo.CreateDevice,
		kmemo.CreateUser,
		kmemo.UpdateTime.Format(sqlite3impl.TimeLayout),
		kmemo.UpdateApp,
		kmemo.UpdateDevice,
		kmemo.UpdateUser,
		userID,
		device,
		txID,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to KMEMO %s: %w", kmemo.ID, err)
		return err
	}
	return nil
}

func (k *kmemoTempRepositorySQLite3Impl) GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error) {
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
FROM KMEMO 
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`

	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at kmemo temp: %w", err)
		return nil, err
	}

	dataType := "kmemo"
	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyous by TXID sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from kmemo temp: %w", err)
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
				err = fmt.Errorf("error at scan from kmemo temp: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in kmemo temp: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in kmemo temp: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in kmemo temp: %w", updateTimeStr, err)
				return nil, err
			}

			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (k *kmemoTempRepositorySQLite3Impl) GetKmemosByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kmemo, error) {
	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at kmemo: %w", err)
		return nil, err
	}

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
  CONTENT,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM KMEMO
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	dataType := "kmemo"

	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kmemo by tx id sql %s: %w", txID, err)
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

	kmemos := []*Kmemo{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kmemo := &Kmemo{}
			kmemo.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(&kmemo.IsDeleted,
				&kmemo.ID,
				&relatedTimeStr,
				&createTimeStr,
				&kmemo.CreateApp,
				&kmemo.CreateDevice,
				&kmemo.CreateUser,
				&updateTimeStr,
				&kmemo.UpdateApp,
				&kmemo.UpdateDevice,
				&kmemo.UpdateUser,
				&kmemo.Content,
				&kmemo.RepName,
				&kmemo.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan kmemo %s: %w", txID, err)
				return nil, err
			}

			kmemo.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in KMEMO: %w", relatedTimeStr, txID, err)
				return nil, err
			}
			kmemo.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in KMEMO: %w", createTimeStr, txID, err)
				return nil, err
			}
			kmemo.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in KMEMO: %w", updateTimeStr, txID, err)
				return nil, err
			}
			kmemos = append(kmemos, kmemo)
		}
	}
	return kmemos, nil
}

func (k *kmemoTempRepositorySQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	sql := `
DELETE FROM KMEMO
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete temp kmemo kyou by TXID sql: %w", err)
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
		err = fmt.Errorf("error at delete temp kmemo kyou by TXID sql: %w", err)
		return err
	}
	return nil
}

func (k *kmemoTempRepositorySQLite3Impl) UnWrapTyped() ([]KmemoTempRepository, error) {
	return []KmemoTempRepository{k}, nil
}

func (k *kmemoTempRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{k}, nil
}

func (k *kmemoTempRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error) {
	return nil, fmt.Errorf("not implements GetLatestDataRepositoryAddress at temp rep")
}
