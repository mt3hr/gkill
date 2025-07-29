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

type urlogRepositoryCachedSQLite3Impl struct {
	dbName   string
	urlogRep URLogRepository
	cachedDB *sql.DB
	m        *sync.Mutex
}

func NewURLogRepositoryCachedSQLite3Impl(ctx context.Context, urlogRepository URLogRepository, cacheDB *sql.DB, m *sync.Mutex, dbName string) (URLogRepository, error) {
	if m == nil {
		m = &sync.Mutex{}
	}
	var err error

	sql := `
CREATE TABLE IF NOT EXISTS "` + dbName + `" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  URL NOT NULL,
  TITLE NOT NULL,
  DESCRIPTION NOT NULL,
  FAVICON_IMAGE NOT NULL,
  THUMBNAIL_IMAGE NOT NULL,
  RELATED_TIME NOT NULL,
  CREATE_TIME NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_TIME NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  REP_NAME NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table statement %s: %w", dbName, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table to %s: %w", dbName, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_` + dbName + ` ON ` + dbName + ` (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := cacheDB.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create URLOG index statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create URLOG index to %s: %w", dbName, err)
		return nil, err
	}

	if err != nil {
		err = fmt.Errorf("error at create URLOG table to %s: %w", dbName, err)
		return nil, err
	}

	return &urlogRepositoryCachedSQLite3Impl{
		dbName:   dbName,
		urlogRep: urlogRepository,
		cachedDB: cacheDB,
		m:        &sync.Mutex{},
	}, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = u.UpdateCache(ctx)
		if err != nil {
			repName, _ := u.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

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
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + u.dbName + `
WHERE
`

	dataType := "urlog"

	queryArgs := []interface{}{
		dataType,
	}

	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
	ignoreFindWord := false
	appendOrderBy := true

	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from URLOG: %w", err)
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
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(
				&kyou.IsDeleted,
				&kyou.ID,
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
				err = fmt.Errorf("error at scan from URLOG: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in URLOG: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in URLOG: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in URLOG: %w", updateTimeStr, err)
				return nil, err
			}
			if _, exist := kyous[kyou.ID]; !exist {
				kyous[kyou.ID] = []*Kyou{}
			}
			kyous[kyou.ID] = append(kyous[kyou.ID], kyou)
		}
	}
	return kyous, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := u.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from URLOG %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(kyouHistories) == 0 {
		return nil, nil
	}

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range kyouHistories {
			if kyou.UpdateTime.Format(sqlite3impl.TimeLayout) == updateTime.Format(sqlite3impl.TimeLayout) {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return kyouHistories[0], nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
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
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + u.dbName + `
WHERE 
`
	dataType := "urlog"

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	queryArgs := []interface{}{
		dataType,
	}

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
	ignoreFindWord := false
	appendOrderBy := false

	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}
	commonWhereSQL += " ORDER BY datetime(UPDATE_TIME, 'localtime') DESC "
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from URLOG %s: %w", id, err)
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
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(
				&kyou.IsDeleted,
				&kyou.ID,
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
				err = fmt.Errorf("error at scan from URLOG %s: %w", id, err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in URLOG: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in URLOG: %w", createTimeStr, id, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in URLOG: %w", updateTimeStr, id, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return u.urlogRep.GetPath(ctx, id)
}

func (u *urlogRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	u.m.Lock()
	defer u.m.Unlock()

	trueValue := true
	query := &find.FindQuery{
		UpdateCache: &trueValue,
	}

	allURLogs, err := u.urlogRep.FindURLog(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all urlogs at update cache: %w", err)
		return err
	}

	tx, err := u.cachedDB.Begin()
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add urlogs: %w", err)
		return err
	}

	sql := `DELETE FROM ` + u.dbName
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table statement %s: %w", "memory", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at delete URLOG table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + u.dbName + ` (
  IS_DELETED,
  ID,
  URL,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
  THUMBNAIL_IMAGE,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME
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
	for _, urlog := range allURLogs {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			gkill_log.TraceSQL.Printf("sql: %s", sql)
			insertStmt, err := tx.PrepareContext(ctx, sql)
			if err != nil {
				err = fmt.Errorf("error at add urlog sql: %w", err)
				return err
			}
			defer insertStmt.Close()

			queryArgs := []interface{}{
				urlog.IsDeleted,
				urlog.ID,
				urlog.URL,
				urlog.Title,
				urlog.Description,
				urlog.FaviconImage,
				urlog.ThumbnailImage,
				urlog.RelatedTime.Format(sqlite3impl.TimeLayout),
				urlog.CreateTime.Format(sqlite3impl.TimeLayout),
				urlog.CreateApp,
				urlog.CreateDevice,
				urlog.CreateUser,
				urlog.UpdateTime.Format(sqlite3impl.TimeLayout),
				urlog.UpdateApp,
				urlog.UpdateDevice,
				urlog.UpdateUser,
				urlog.RepName,
			}
			gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to URLog %s: %w", urlog.ID, err)
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit transaction for add urlogs: %w", err)
		return err
	}
	return nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return u.urlogRep.GetRepName(ctx)
}

func (u *urlogRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	_, err := u.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+u.dbName)
	return err
}

func (u *urlogRepositoryCachedSQLite3Impl) FindURLog(ctx context.Context, query *find.FindQuery) ([]*URLog, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = u.UpdateCache(ctx)
		if err != nil {
			repName, _ := u.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

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
  URL,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
  THUMBNAIL_IMAGE,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + u.dbName + `
WHERE
`

	dataType := "urlog"

	queryArgs := []interface{}{
		dataType,
	}

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
	ignoreFindWord := false
	appendOrderBy := true

	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from URLOG: %w", err)
		return nil, err
	}
	defer rows.Close()

	urlogs := []*URLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			urlog := &URLog{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(
				&urlog.IsDeleted,
				&urlog.ID,
				&relatedTimeStr,
				&createTimeStr,
				&urlog.CreateApp,
				&urlog.CreateDevice,
				&urlog.CreateUser,
				&updateTimeStr,
				&urlog.UpdateApp,
				&urlog.UpdateDevice,
				&urlog.UpdateUser,
				&urlog.URL,
				&urlog.Title,
				&urlog.Description,
				&urlog.FaviconImage,
				&urlog.ThumbnailImage,
				&urlog.RepName,
				&urlog.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from URLOG: %w", err)
				return nil, err
			}

			urlog.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in URLOG: %w", relatedTimeStr, err)
				return nil, err
			}
			urlog.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in URLOG: %w", createTimeStr, err)
				return nil, err
			}
			urlog.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in URLOG: %w", updateTimeStr, err)
				return nil, err
			}
			urlogs = append(urlogs, urlog)
		}
	}
	return urlogs, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetURLog(ctx context.Context, id string, updateTime *time.Time) (*URLog, error) {
	// 最新のデータを返す
	urlogHistories, err := u.GetURLogHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get urlog histories from URLog %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(urlogHistories) == 0 {
		return nil, nil
	}

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range urlogHistories {
			if kyou.UpdateTime.Format(sqlite3impl.TimeLayout) == updateTime.Format(sqlite3impl.TimeLayout) {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return urlogHistories[0], nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetURLogHistories(ctx context.Context, id string) ([]*URLog, error) {
	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at URLOG: %w", err)
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
  URL,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
  THUMBNAIL_IMAGE,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + u.dbName + `
WHERE
`

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}
	dataType := "urlog"

	queryArgs := []interface{}{
		dataType,
	}

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
	ignoreFindWord := false
	appendOrderBy := false

	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}
	commonWhereSQL += " ORDER BY datetime(UPDATE_TIME, 'localtime') DESC "
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get urlog histories sql %s: %w", id, err)
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

	urlogs := []*URLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			urlog := &URLog{}
			urlog.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(
				&urlog.IsDeleted,
				&urlog.ID,
				&relatedTimeStr,
				&createTimeStr,
				&urlog.CreateApp,
				&urlog.CreateDevice,
				&urlog.CreateUser,
				&updateTimeStr,
				&urlog.UpdateApp,
				&urlog.UpdateDevice,
				&urlog.UpdateUser,
				&urlog.URL,
				&urlog.Title,
				&urlog.Description,
				&urlog.FaviconImage,
				&urlog.ThumbnailImage,
				&urlog.RepName,
				&urlog.DataType,
			)

			if err != nil {
				err = fmt.Errorf("error at scan from URLOG %s: %w", id, err)
				return nil, err
			}

			urlog.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in URLog: %w", relatedTimeStr, id, err)
				return nil, err
			}
			urlog.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in URLog: %w", createTimeStr, id, err)
				return nil, err
			}
			urlog.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in URLog: %w", updateTimeStr, id, err)
				return nil, err
			}
			urlogs = append(urlogs, urlog)
		}
	}
	return urlogs, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) AddURLogInfo(ctx context.Context, urlog *URLog) error {
	sql := `
INSERT INTO ` + u.dbName + ` (
  IS_DELETED,
  ID,
  URL,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
  THUMBNAIL_IMAGE,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME
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
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add urlog sql %s: %w", urlog.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		urlog.IsDeleted,
		urlog.ID,
		urlog.URL,
		urlog.Title,
		urlog.Description,
		urlog.FaviconImage,
		urlog.ThumbnailImage,
		urlog.RelatedTime.Format(sqlite3impl.TimeLayout),
		urlog.CreateTime.Format(sqlite3impl.TimeLayout),
		urlog.CreateApp,
		urlog.CreateDevice,
		urlog.CreateUser,
		urlog.UpdateTime.Format(sqlite3impl.TimeLayout),
		urlog.UpdateApp,
		urlog.UpdateDevice,
		urlog.UpdateUser,
		urlog.RepName,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to URLog %s: %w", urlog.ID, err)
		return err
	}
	return nil
}
