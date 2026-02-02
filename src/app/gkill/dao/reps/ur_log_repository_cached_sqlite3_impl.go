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
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
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
		err = fmt.Errorf("error at create URLOG table statement %s: %w", dbName, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table statement %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `_UNIX" ON "` + dbName + `"(ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	gkill_log.TraceSQL.Printf("sql: %s", indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create urlog index unix statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexUnixStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexUnixSQL)
	_, err = indexUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create urlog index unix to %s: %w", dbName, err)
		return nil, err
	}

	return &urlogRepositoryCachedSQLite3Impl{
		dbName:   dbName,
		urlogRep: urlogRepository,
		cachedDB: cacheDB,
		m:        m,
	}, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	u.m.Lock()
	u.m.Unlock()
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
FROM ` + u.dbName + `
WHERE
`

	dataType := "urlog"

	tableName := u.dbName
	tableNameAlias := u.dbName
	queryArgs := []interface{}{
		dataType,
	}
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
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
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(
				&kyou.IsDeleted,
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
				err = fmt.Errorf("error at scan from URLOG: %w", err)
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
			if kyou.UpdateTime.Unix() == updateTime.Unix() {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return kyouHistories[0], nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	u.m.Lock()
	u.m.Unlock()
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

	tableName := u.dbName
	tableNameAlias := u.dbName
	queryArgs := []interface{}{
		dataType,
	}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
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
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(
				&kyou.IsDeleted,
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
				err = fmt.Errorf("error at scan from URLOG %s: %w", id, err)
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

func (u *urlogRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return u.urlogRep.GetPath(ctx, id)
}

func (u *urlogRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	trueValue := true
	falseValue := false
	query := &find.FindQuery{
		UpdateCache:    &trueValue,
		OnlyLatestData: &falseValue,
	}

	allURLogs, err := u.urlogRep.FindURLog(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all urlogs at update cache: %w", err)
		return err
	}

	u.m.Lock()
	defer u.m.Unlock()

	tx, err := u.cachedDB.BeginTx(ctx, nil)
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
  ?,
  ?,
  ?
)`

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add urlog sql: %w", err)
		return err
	}
	defer insertStmt.Close()

	for _, urlog := range allURLogs {
		select {
		case <-ctx.Done():
			tx.Rollback()
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			queryArgs := []interface{}{
				urlog.IsDeleted,
				urlog.ID,
				urlog.URL,
				urlog.Title,
				urlog.Description,
				urlog.FaviconImage,
				urlog.ThumbnailImage,
				urlog.CreateApp,
				urlog.CreateDevice,
				urlog.CreateUser,
				urlog.UpdateApp,
				urlog.UpdateDevice,
				urlog.UpdateUser,
				urlog.RepName,
				urlog.RelatedTime.Unix(),
				urlog.CreateTime.Unix(),
				urlog.UpdateTime.Unix(),
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
			tx.Rollback()
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
	err := u.urlogRep.Close(ctx)
	if err != nil {
		return err
	}
	if gkill_options.CacheURLogReps == nil || !*gkill_options.CacheURLogReps {
		err = u.cachedDB.Close()
		if err != nil {
			return err
		}
	} else {
		_, err = u.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+u.dbName)
		if err != nil {
			return err
		}
	}
	return nil
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
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
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

	tableName := u.dbName
	tableNameAlias := u.dbName
	queryArgs := []interface{}{
		dataType,
	}
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
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
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(
				&urlog.IsDeleted,
				&urlog.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&urlog.CreateApp,
				&urlog.CreateDevice,
				&urlog.CreateUser,
				&updateTimeUnix,
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

			urlog.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			urlog.CreateTime = time.Unix(createTimeUnix, 0).Local()
			urlog.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
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
			if kyou.UpdateTime.Unix() == updateTime.Unix() {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return urlogHistories[0], nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetURLogHistories(ctx context.Context, id string) ([]*URLog, error) {
	u.m.Lock()
	u.m.Unlock()
	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at URLOG: %w", err)
		return nil, err
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

	tableName := u.dbName
	tableNameAlias := u.dbName
	queryArgs := []interface{}{
		dataType,
	}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
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
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(
				&urlog.IsDeleted,
				&urlog.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&urlog.CreateApp,
				&urlog.CreateDevice,
				&urlog.CreateUser,
				&updateTimeUnix,
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

			urlog.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			urlog.CreateTime = time.Unix(createTimeUnix, 0).Local()
			urlog.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			if err != nil {
				err = fmt.Errorf("error at scan from URLOG %s: %w", id, err)
				return nil, err
			}
			urlogs = append(urlogs, urlog)
		}
	}
	return urlogs, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) AddURLogInfo(ctx context.Context, urlog *URLog) error {
	u.m.Lock()
	defer u.m.Unlock()
	sql := `
INSERT INTO ` + u.dbName + ` (
  IS_DELETED,
  ID,
  URL,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
  THUMBNAIL_IMAGE,
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
		urlog.CreateApp,
		urlog.CreateDevice,
		urlog.CreateUser,
		urlog.UpdateApp,
		urlog.UpdateDevice,
		urlog.UpdateUser,
		urlog.RepName,
		urlog.RelatedTime.Unix(),
		urlog.CreateTime.Unix(),
		urlog.UpdateTime.Unix(),
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to URLog %s: %w", urlog.ID, err)
		return err
	}
	return nil
}

func (u *urlogRepositoryCachedSQLite3Impl) UnWrapTyped() ([]URLogRepository, error) {
	return u.urlogRep.UnWrapTyped()
}

func (u *urlogRepositoryCachedSQLite3Impl) UnWrap() ([]Repository, error) {
	return u.urlogRep.UnWrap()
}

func (u *urlogRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error) {
	defer u.UpdateCache(ctx)
	return u.urlogRep.GetLatestDataRepositoryAddress(ctx, updateCache)
}
