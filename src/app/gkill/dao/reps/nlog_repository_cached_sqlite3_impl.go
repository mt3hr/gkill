package reps

import (
	"context"
	"database/sql"
	sqllib "database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type nlogRepositoryCachedSQLite3Impl struct {
	dbName   string
	nlogRep  NlogRepository
	cachedDB *sqllib.DB
	m        *sync.Mutex
}

func NewNlogRepositoryCachedSQLite3Impl(ctx context.Context, nlogRep NlogRepository, cacheDB *sql.DB, m *sync.Mutex, dbName string) (NlogRepository, error) {
	if m == nil {
		m = &sync.Mutex{}
	}
	var err error
	sql := `
CREATE TABLE IF NOT EXISTS "` + dbName + `" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  SHOP NOT NULL,
  TITLE NOT NULL,
  AMOUNT NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  REP_NAME NOT NULL,
  RELATED_TIME_UNIX NOT NULL,
  CREATE_TIME_UNIX NOT NULL,
  UPDATE_TIME_UNIX NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create NLOG table statement %s: %w", dbName, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NLOG table to %s: %w", dbName, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NLOG table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `_UNIX" ON "` + dbName + `"(ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	gkill_log.TraceSQL.Printf("sql: %s", indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create nlog index unix statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexUnixStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexUnixSQL)
	_, err = indexUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create nlog index unix to %s: %w", dbName, err)
		return nil, err
	}

	return &nlogRepositoryCachedSQLite3Impl{
		dbName:   dbName,
		nlogRep:  nlogRep,
		cachedDB: cacheDB,
		m:        m,
	}, nil
}
func (n *nlogRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	n.m.Lock()
	n.m.Unlock()
	var err error
	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = n.UpdateCache(ctx)
		if err != nil {
			repName, _ := n.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}

	sql := `
SELECT 
  IS_DELETED,
  ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + n.dbName + `
WHERE
`

	dataType := "nlog"

	queryArgs := []interface{}{
		dataType,
	}

	tableName := n.dbName
	tableNameAlias := n.dbName
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE", "SHOP"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := true
	if query.OnlyLatestData != nil {
		onlyLatestData = *query.OnlyLatestData
	} else {
		onlyLatestData = false
	}
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)

	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := n.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from NLOG: %w", err)
		return nil, err
	}
	defer rows.Close()

	kyous := map[string][]*Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := &Kyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&kyou.IsDeleted,
				&kyou.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&kyou.CreateApp,
				&kyou.CreateDevice,
				&kyou.CreateUser,
				&updateTimeUnix,
				&kyou.UpdateApp,
				&kyou.UpdateDevice,
				&kyou.UpdateUser,
				&kyou.RepName,
				&kyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan NLOG: %w", err)
				return nil, err
			}

			kyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			kyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			kyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			if _, exist := kyous[kyou.ID]; !exist {
				kyous[kyou.ID] = []*Kyou{}
			}
			kyous[kyou.ID] = append(kyous[kyou.ID], kyou)
		}
	}
	return kyous, nil
}

func (n *nlogRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := n.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from NLOG%s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(kyouHistories) == 0 {
		return nil, nil
	}

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range kyouHistories {
			if kyou.UpdateTime.Unix() == updateTime.Unix() {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return kyouHistories[0], nil
}

func (n *nlogRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	n.m.Lock()
	n.m.Unlock()
	sql := `
SELECT 
  IS_DELETED,
  ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + n.dbName + `
WHERE 
`

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	dataType := "nlog"
	queryArgs := []interface{}{
		dataType,
	}

	tableName := n.dbName
	tableNameAlias := n.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE", "SHOP"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := n.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from NLOG %s: %w", id, err)
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
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&kyou.IsDeleted,
				&kyou.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&kyou.CreateApp,
				&kyou.CreateDevice,
				&kyou.CreateUser,
				&updateTimeUnix,
				&kyou.UpdateApp,
				&kyou.UpdateDevice,
				&kyou.UpdateUser,
				&kyou.RepName,
				&kyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan NLOG %s: %w", id, err)
				return nil, err
			}

			kyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			kyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			kyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (n *nlogRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return n.nlogRep.GetPath(ctx, id)
}

func (n *nlogRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	trueValue := true
	falseValue := false
	query := &find.FindQuery{
		UpdateCache:    &trueValue,
		OnlyLatestData: &falseValue,
	}

	allNlogs, err := n.nlogRep.FindNlog(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all nlog at update cache: %w", err)
		return err
	}

	n.m.Lock()
	defer n.m.Unlock()

	tx, err := n.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add nlog: %w", err)
		return err
	}

	sql := `DELETE FROM ` + n.dbName
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create NLOG table statement %s: %w", "memory", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at delete NLOG table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + n.dbName + ` (
  IS_DELETED,
  ID,
  SHOP,
  TITLE,
  AMOUNT,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  UPDATE_TIME_UNIX
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
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add nlog sql: %w", err)
		return err
	}
	defer insertStmt.Close()

	for _, nlog := range allNlogs {
		select {
		case <-ctx.Done():
			tx.Rollback()
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			queryArgs := []interface{}{
				nlog.IsDeleted,
				nlog.ID,
				nlog.Shop,
				nlog.Title,
				nlog.Amount.String(),
				nlog.CreateApp,
				nlog.CreateDevice,
				nlog.CreateUser,
				nlog.UpdateApp,
				nlog.UpdateDevice,
				nlog.UpdateUser,
				nlog.RepName,
				nlog.RelatedTime.Unix(),
				nlog.CreateTime.Unix(),
				nlog.UpdateTime.Unix(),
			}
			gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to NLOG %s: %w", nlog.ID, err)
				return err
			}
			return nil
		}()
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit transaction for add nlogs: %w", err)
		return err
	}
	return nil
}

func (n *nlogRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return n.nlogRep.GetRepName(ctx)
}

func (n *nlogRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	err := n.nlogRep.Close(ctx)
	if err != nil {
		return err
	}
	if gkill_options.CacheNlogReps == nil || !*gkill_options.CacheNlogReps {
		err = n.cachedDB.Close()
		if err != nil {
			return err
		}
	} else {
		_, err = n.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+n.dbName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *nlogRepositoryCachedSQLite3Impl) FindNlog(ctx context.Context, query *find.FindQuery) ([]*Nlog, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = n.UpdateCache(ctx)
		if err != nil {
			repName, _ := n.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

	dataType := "nlog"

	sql := `
SELECT 
  IS_DELETED,
  ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  SHOP,
  TITLE,
  AMOUNT,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + n.dbName + `
WHERE
`

	queryArgs := []interface{}{
		dataType,
	}
	tableName := n.dbName
	tableNameAlias := n.dbName
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE", "SHOP"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := n.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from NLOG: %w", err)
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
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			amount := 0

			err = rows.Scan(&nlog.IsDeleted,
				&nlog.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&nlog.CreateApp,
				&nlog.CreateDevice,
				&nlog.CreateUser,
				&updateTimeUnix,
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
				err = fmt.Errorf("error at scan NLOG: %w", err)
				return nil, err
			}

			nlog.Amount = json.Number(strconv.Itoa(amount))

			nlog.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			nlog.CreateTime = time.Unix(createTimeUnix, 0).Local()
			nlog.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			nlogs = append(nlogs, nlog)
		}
	}
	return nlogs, nil
}

func (n *nlogRepositoryCachedSQLite3Impl) GetNlog(ctx context.Context, id string, updateTime *time.Time) (*Nlog, error) {
	// 最新のデータを返す
	nlogHistories, err := n.GetNlogHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get nlog histories from NLOG%s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(nlogHistories) == 0 {
		return nil, nil
	}

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range nlogHistories {
			if kyou.UpdateTime.Unix() == updateTime.Unix() {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return nlogHistories[0], nil
}

func (n *nlogRepositoryCachedSQLite3Impl) GetNlogHistories(ctx context.Context, id string) ([]*Nlog, error) {
	n.m.Lock()
	n.m.Unlock()
	sql := `
SELECT 
  IS_DELETED,
  ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  SHOP,
  TITLE,
  AMOUNT,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + n.dbName + `
WHERE 
`
	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	dataType := "nlog"
	queryArgs := []interface{}{
		dataType,
	}
	tableName := n.dbName
	tableNameAlias := n.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE", "SHOP"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)

	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := n.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get nlog histories sql %s: %w", id, err)
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
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			amount := 0

			err = rows.Scan(&nlog.IsDeleted,
				&nlog.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&nlog.CreateApp,
				&nlog.CreateDevice,
				&nlog.CreateUser,
				&updateTimeUnix,
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
				err = fmt.Errorf("error at scan NLOG %s: %w", id, err)
				return nil, err
			}

			nlog.Amount = json.Number(strconv.Itoa(amount))

			nlog.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			nlog.CreateTime = time.Unix(createTimeUnix, 0).Local()
			nlog.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			nlogs = append(nlogs, nlog)
		}
	}
	return nlogs, nil
}

func (n *nlogRepositoryCachedSQLite3Impl) AddNlogInfo(ctx context.Context, nlog *Nlog) error {
	sql := `
INSERT INTO ` + n.dbName + ` (
  IS_DELETED,
  ID,
  SHOP,
  TITLE,
  AMOUNT,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  UPDATE_TIME_UNIX 
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
	stmt, err := n.cachedDB.PrepareContext(ctx, sql)
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
		nlog.CreateApp,
		nlog.CreateDevice,
		nlog.CreateUser,
		nlog.UpdateApp,
		nlog.UpdateDevice,
		nlog.UpdateUser,
		nlog.RepName,
		nlog.RelatedTime.Unix(),
		nlog.CreateTime.Unix(),
		nlog.UpdateTime.Unix(),
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to NLOG %s: %w", nlog.ID, err)
		return err
	}
	return nil
}

func (n *nlogRepositoryCachedSQLite3Impl) UnWrapTyped() ([]NlogRepository, error) {
	return n.nlogRep.UnWrapTyped()
}

func (n *nlogRepositoryCachedSQLite3Impl) UnWrap() ([]Repository, error) {
	return n.nlogRep.UnWrap()
}
