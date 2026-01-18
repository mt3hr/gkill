package reps

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type kcTempRepositorySQLite3Impl kcRepositorySQLite3Impl

func NewKCTempRepositorySQLite3Impl(ctx context.Context, db *sql.DB, m *sync.Mutex) (KCTempRepository, error) {
	filename := "temp_db"

	sql := `
CREATE TABLE IF NOT EXISTS "kc" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TITLE NOT NULL,
  NUM_VALUE NOT NULL,
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
		err = fmt.Errorf("error at create kc table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create kc table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_kc ON kc (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create kc index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create kc index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create kc table to %s: %w", filename, err)
		return nil, err
	}

	return &kcTempRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        m,
	}, nil
}

func (k *kcTempRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	impl := kcRepositorySQLite3Impl(*k)
	return impl.FindKyous(ctx, query)
}

func (k *kcTempRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	impl := kcRepositorySQLite3Impl(*k)
	return impl.GetKyou(ctx, id, updateTime)
}

func (k *kcTempRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	impl := kcRepositorySQLite3Impl(*k)
	return impl.GetKyouHistories(ctx, id)
}

func (k *kcTempRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("GetPath is not implemented for kcTempRepositorySQLite3Impl")
}

func (k *kcTempRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	impl := kcRepositorySQLite3Impl(*k)
	return impl.UpdateCache(ctx)
}

func (k *kcTempRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "KC_TEMP", nil
}

func (k *kcTempRepositorySQLite3Impl) Close(ctx context.Context) error {
	impl := kcRepositorySQLite3Impl(*k)
	return impl.Close(ctx)
}

func (k *kcTempRepositorySQLite3Impl) FindKC(ctx context.Context, query *find.FindQuery) ([]*KC, error) {
	impl := kcRepositorySQLite3Impl(*k)
	return impl.FindKC(ctx, query)
}

func (k *kcTempRepositorySQLite3Impl) GetKC(ctx context.Context, id string, updateTime *time.Time) (*KC, error) {
	impl := kcRepositorySQLite3Impl(*k)
	return impl.GetKC(ctx, id, updateTime)
}

func (k *kcTempRepositorySQLite3Impl) GetKCHistories(ctx context.Context, id string) ([]*KC, error) {
	impl := kcRepositorySQLite3Impl(*k)
	return impl.GetKCHistories(ctx, id)
}

func (k *kcTempRepositorySQLite3Impl) AddKCInfo(ctx context.Context, kc *KC, txID string, userID string, device string) error {
	k.m.Lock()
	defer k.m.Unlock()
	sql := `
INSERT INTO kc (
  IS_DELETED,
  ID,
  TITLE,
  NUM_VALUE,
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
  ?,
  ?
)`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add kc sql %s: %w", kc.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		kc.IsDeleted,
		kc.ID,
		kc.Title,
		kc.NumValue.String(),
		kc.RelatedTime.Format(sqlite3impl.TimeLayout),
		kc.CreateTime.Format(sqlite3impl.TimeLayout),
		kc.CreateApp,
		kc.CreateDevice,
		kc.CreateUser,
		kc.UpdateTime.Format(sqlite3impl.TimeLayout),
		kc.UpdateApp,
		kc.UpdateDevice,
		kc.UpdateUser,
		userID,
		device,
		txID,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to kc %s: %w", kc.ID, err)
		return err
	}
	return nil
}

func (k *kcTempRepositorySQLite3Impl) GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error) {
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
FROM kc
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`

	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at kc temp: %w", err)
		return nil, err
	}

	dataType := "kc"
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
		err = fmt.Errorf("error at select from kc temp: %w", err)
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
				err = fmt.Errorf("error at scan from kc temp: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in kc temp: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in kc temp: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in kc temp: %w", updateTimeStr, err)
				return nil, err
			}

			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (k *kcTempRepositorySQLite3Impl) GetKCsByTXID(ctx context.Context, txID string, userID string, device string) ([]*KC, error) {
	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at kc: %w", err)
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
  TITLE,
  NUM_VALUE,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM kc
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	dataType := "kc"

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
		err = fmt.Errorf("error at get kcs by tx id sql %s: %w", txID, err)
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

	kcs := []*KC{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kc := &KC{}
			kc.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			numValueStr := ""

			err = rows.Scan(&kc.IsDeleted,
				&kc.ID,
				&relatedTimeStr,
				&createTimeStr,
				&kc.CreateApp,
				&kc.CreateDevice,
				&kc.CreateUser,
				&updateTimeStr,
				&kc.UpdateApp,
				&kc.UpdateDevice,
				&kc.UpdateUser,
				&kc.Title,
				&numValueStr,
				&kc.RepName,
				&kc.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan kc %s: %w", txID, err)
				return nil, err
			}
			numValue := strings.ReplaceAll(numValueStr, ",", "")
			kc.NumValue = json.Number(numValue)

			kc.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in kc: %w", relatedTimeStr, txID, err)
				return nil, err
			}
			kc.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in kc: %w", createTimeStr, txID, err)
				return nil, err
			}
			kc.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in kc: %w", updateTimeStr, txID, err)
				return nil, err
			}
			kcs = append(kcs, kc)
		}
	}
	return kcs, nil
}

func (k *kcTempRepositorySQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	sql := `
DELETE FROM kc
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete temp kc kyou by TXID sql: %w", err)
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
		err = fmt.Errorf("error at delete temp kc kyou by TXID sql: %w", err)
		return err
	}
	return nil
}

func (k *kcTempRepositorySQLite3Impl) UnWrapTyped() ([]KCTempRepository, error) {
	return []KCTempRepository{k}, nil
}

func (k *kcTempRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{k}, nil
}
