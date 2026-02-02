package reps

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type nlogTempRepositorySQLite3Impl nlogRepositorySQLite3Impl

func NewNlogTempRepositorySQLite3Impl(ctx context.Context, db *sql.DB, m *sync.Mutex) (NlogTempRepository, error) {
	filename := "nlog_temp"

	sql := `
CREATE TABLE IF NOT EXISTS "NLOG" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  SHOP NOT NULL,
  TITLE NOT NULL,
  AMOUNT NOT NULL,
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
		err = fmt.Errorf("error at create NLOG table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NLOG table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_NLOG ON NLOG (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create NLOG index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NLOG index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create NLOG table to %s: %w", filename, err)
		return nil, err
	}

	return &nlogTempRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        m,
	}, nil
}
func (n *nlogTempRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	impl := nlogRepositorySQLite3Impl(*n)
	return impl.FindKyous(ctx, query)
}

func (n *nlogTempRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	impl := nlogRepositorySQLite3Impl(*n)
	return impl.GetKyou(ctx, id, updateTime)
}

func (n *nlogTempRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	impl := nlogRepositorySQLite3Impl(*n)
	return impl.GetKyouHistories(ctx, id)
}

func (n *nlogTempRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("GetPath is not implemented for nlog_temp")
}

func (n *nlogTempRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	impl := nlogRepositorySQLite3Impl(*n)
	return impl.UpdateCache(ctx)
}

func (n *nlogTempRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "nlog_temp", nil
}

func (n *nlogTempRepositorySQLite3Impl) Close(ctx context.Context) error {
	impl := nlogRepositorySQLite3Impl(*n)
	return impl.Close(ctx)
}

func (n *nlogTempRepositorySQLite3Impl) FindNlog(ctx context.Context, query *find.FindQuery) ([]*Nlog, error) {
	impl := nlogRepositorySQLite3Impl(*n)
	return impl.FindNlog(ctx, query)
}

func (n *nlogTempRepositorySQLite3Impl) GetNlog(ctx context.Context, id string, updateTime *time.Time) (*Nlog, error) {
	impl := nlogRepositorySQLite3Impl(*n)
	return impl.GetNlog(ctx, id, updateTime)
}

func (n *nlogTempRepositorySQLite3Impl) GetNlogHistories(ctx context.Context, id string) ([]*Nlog, error) {
	impl := nlogRepositorySQLite3Impl(*n)
	return impl.GetNlogHistories(ctx, id)
}

func (n *nlogTempRepositorySQLite3Impl) AddNlogInfo(ctx context.Context, nlog *Nlog, txID string, userID string, device string) error {
	n.m.Lock()
	defer n.m.Unlock()
	sql := `
INSERT INTO NLOG (
  IS_DELETED,
  ID,
  SHOP,
  TITLE,
  AMOUNT,
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
  ?,
  ?
)`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := n.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add nlog sql %s: %w", nlog.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		nlog.IsDeleted,
		nlog.ID,
		nlog.Shop,
		nlog.Title,
		nlog.Amount.String(),
		nlog.RelatedTime.Format(sqlite3impl.TimeLayout),
		nlog.CreateTime.Format(sqlite3impl.TimeLayout),
		nlog.CreateApp,
		nlog.CreateDevice,
		nlog.CreateUser,
		nlog.UpdateTime.Format(sqlite3impl.TimeLayout),
		nlog.UpdateApp,
		nlog.UpdateDevice,
		nlog.UpdateUser,
		userID,
		device,
		txID,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to NLOG %s: %w", nlog.ID, err)
		return err
	}
	return nil
}

func (n *nlogTempRepositorySQLite3Impl) GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error) {
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
FROM NLOG
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`

	repName, err := n.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at nlog temp: %w", err)
		return nil, err
	}

	dataType := "nlog"
	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := n.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyous by TXID sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from nlog temp: %w", err)
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
				err = fmt.Errorf("error at scan from nlog temp: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in nlog temp: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in nlog temp: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in nlog temp: %w", updateTimeStr, err)
				return nil, err
			}

			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (n *nlogTempRepositorySQLite3Impl) GetNlogsByTXID(ctx context.Context, txID string, userID string, device string) ([]*Nlog, error) {
	repName, err := n.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at nlog: %w", err)
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
  SHOP,
  TITLE,
  AMOUNT,
  ? AS REP_NAME,
  ? AS DATA_TYHPE
FROM NLOG 
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	dataType := "nlog"
	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := n.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get nlog by tx id sql %s: %w", txID, err)
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

	nlogs := []*Nlog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			nlog := &Nlog{}
			nlog.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			amount := 0

			err = rows.Scan(&nlog.IsDeleted,
				&nlog.ID,
				&relatedTimeStr,
				&createTimeStr,
				&nlog.CreateApp,
				&nlog.CreateDevice,
				&nlog.CreateUser,
				&updateTimeStr,
				&nlog.UpdateApp,
				&nlog.UpdateDevice,
				&nlog.UpdateUser,
				&nlog.Shop,
				&nlog.Title,
				&amount,
				&nlog.RepName,
				&nlog.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan NLOG %s: %w", txID, err)
				return nil, err
			}

			nlog.Amount = json.Number(strconv.Itoa(amount))

			nlog.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in NLOG: %w", relatedTimeStr, txID, err)
				return nil, err
			}
			nlog.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in NLOG: %w", createTimeStr, txID, err)
				return nil, err
			}
			nlog.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in NLOG: %w", updateTimeStr, txID, err)
				return nil, err
			}
			nlogs = append(nlogs, nlog)
		}
	}
	return nlogs, nil
}

func (n *nlogTempRepositorySQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	sql := `
DELETE FROM NLOG
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := n.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete temp nlog kyou by TXID sql: %w", err)
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
		err = fmt.Errorf("error at delete temp nlog kyou by TXID sql: %w", err)
		return err
	}
	return nil
}

func (n *nlogTempRepositorySQLite3Impl) UnWrapTyped() ([]NlogTempRepository, error) {
	return []NlogTempRepository{n}, nil
}

func (n *nlogTempRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{n}, nil
}

func (n *nlogTempRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error) {
	return nil, fmt.Errorf("not implements GetLatestDataRepositoryAddress at temp rep")
}
