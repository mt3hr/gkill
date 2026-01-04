package reps

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type reKyouTempRepositorySQLite3Impl reKyouRepositorySQLite3Impl

func NewReKyouTempRepositorySQLite3Impl(ctx context.Context, db *sql.DB) (ReKyouTempRepository, error) {
	filename := "rekyou_temp"
	sql := `
CREATE TABLE IF NOT EXISTS "REKYOU" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
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
		err = fmt.Errorf("error at create REKYOU table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_REKYOU ON REKYOU (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create REKYOU table to %s: %w", filename, err)
		return nil, err
	}

	return &reKyouTempRepositorySQLite3Impl{
		filename:          filename,
		db:                db,
		m:                 &sync.Mutex{},
		gkillRepositories: &GkillRepositories{},
	}, nil
}
func (r *reKyouTempRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	impl := reKyouRepositorySQLite3Impl(*r)
	return impl.FindKyous(ctx, query)
}

func (r *reKyouTempRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	impl := reKyouRepositorySQLite3Impl(*r)
	return impl.GetKyou(ctx, id, updateTime)
}

func (r *reKyouTempRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	impl := reKyouRepositorySQLite3Impl(*r)
	return impl.GetKyouHistories(ctx, id)
}

func (r *reKyouTempRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("GetPath is not implemented for reKyouTempRepositorySQLite3Impl")
}

func (r *reKyouTempRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	impl := reKyouRepositorySQLite3Impl(*r)
	return impl.UpdateCache(ctx)
}

func (r *reKyouTempRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "rekyou_temp", nil
}

func (r *reKyouTempRepositorySQLite3Impl) Close(ctx context.Context) error {
	impl := reKyouRepositorySQLite3Impl(*r)
	return impl.Close(ctx)
}

func (r *reKyouTempRepositorySQLite3Impl) FindReKyou(ctx context.Context, query *find.FindQuery) ([]*ReKyou, error) {
	impl := reKyouRepositorySQLite3Impl(*r)
	return impl.FindReKyou(ctx, query)
}

func (r *reKyouTempRepositorySQLite3Impl) GetReKyou(ctx context.Context, id string, updateTime *time.Time) (*ReKyou, error) {
	impl := reKyouRepositorySQLite3Impl(*r)
	return impl.GetReKyou(ctx, id, updateTime)
}

func (r *reKyouTempRepositorySQLite3Impl) GetReKyouHistories(ctx context.Context, id string) ([]*ReKyou, error) {
	impl := reKyouRepositorySQLite3Impl(*r)
	return impl.GetReKyouHistories(ctx, id)
}

func (r *reKyouTempRepositorySQLite3Impl) AddReKyouInfo(ctx context.Context, rekyou *ReKyou, txID string, userID string, device string) error {
	sql := `
INSERT INTO REKYOU (
  IS_DELETED,
  ID,
  TARGET_ID,
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
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add rekyou sql %s: %w", rekyou.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		rekyou.IsDeleted,
		rekyou.ID,
		rekyou.TargetID,
		rekyou.RelatedTime.Format(sqlite3impl.TimeLayout),
		rekyou.CreateTime.Format(sqlite3impl.TimeLayout),
		rekyou.CreateApp,
		rekyou.CreateDevice,
		rekyou.CreateUser,
		rekyou.UpdateTime.Format(sqlite3impl.TimeLayout),
		rekyou.UpdateApp,
		rekyou.UpdateDevice,
		rekyou.UpdateUser,
		userID,
		device,
		txID,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to REKYOU %s: %w", rekyou.ID, err)
		return err
	}
	return nil
}

func (r *reKyouTempRepositorySQLite3Impl) GetReKyousAllLatest(ctx context.Context) ([]*ReKyou, error) {
	impl := reKyouRepositorySQLite3Impl(*r)
	return impl.GetReKyousAllLatest(ctx)
}

func (r *reKyouTempRepositorySQLite3Impl) GetRepositoriesWithoutReKyouRep(ctx context.Context) (*GkillRepositories, error) {
	impl := reKyouRepositorySQLite3Impl(*r)
	return impl.GetRepositoriesWithoutReKyouRep(ctx)
}

func (r *reKyouTempRepositorySQLite3Impl) GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error) {
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
FROM REKYOU
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`

	repName, err := r.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at rekyou temp: %w", err)
		return nil, err
	}

	dataType := "rekyou"
	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyous by TXID sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from rekyou temp: %w", err)
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
				err = fmt.Errorf("error at scan from rekyou temp: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in rekyou temp: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in rekyou temp: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in rekyou temp: %w", updateTimeStr, err)
				return nil, err
			}

			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (r *reKyouTempRepositorySQLite3Impl) GetReKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*ReKyou, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM REKYOU
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	repName, err := r.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at rekyou: %w", err)
		return nil, err
	}

	dataType := "rekyou"

	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get rekyou by tx id sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from REKYOU: %w", err)
		return nil, err
	}
	defer rows.Close()

	reKyous := []*ReKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			reKyou := &ReKyou{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(&reKyou.IsDeleted,
				&reKyou.ID,
				&reKyou.TargetID,
				&relatedTimeStr,
				&createTimeStr,
				&reKyou.CreateApp,
				&reKyou.CreateDevice,
				&reKyou.CreateUser,
				&updateTimeStr,
				&reKyou.UpdateApp,
				&reKyou.UpdateDevice,
				&reKyou.UpdateUser,
				&reKyou.RepName,
				&reKyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from REKYOU %s: %w", txID, err)
				return nil, err
			}

			reKyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in REKYOU: %w", relatedTimeStr, err)
				return nil, err
			}
			reKyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in REKYOU: %w", createTimeStr, err)
				return nil, err
			}
			reKyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in REKYOU: %w", updateTimeStr, err)
				return nil, err
			}
			reKyous = append(reKyous, reKyou)
		}
	}
	return reKyous, nil
}

func (r *reKyouTempRepositorySQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	sql := `
DELETE FROM REKYOU 
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete temp rekyou kyou by TXID sql: %w", err)
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
		err = fmt.Errorf("error at delete temp rekyou kyou by TXID sql: %w", err)
		return err
	}
	return nil
}

func (r *reKyouTempRepositorySQLite3Impl) UnWrapTyped() ([]ReKyouTempRepository, error) {
	return []ReKyouTempRepository{r}, nil
}

func (r *reKyouTempRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{r}, nil
}
