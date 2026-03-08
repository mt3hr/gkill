package reps

import (
	"context"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	sqllib "database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

type urlogRepositoryCachedSQLite3Impl struct {
	dbName           string
	urlogRep         URLogRepository
	cachedDB         *sqllib.DB
	addURLogInfoSQL  string
	addURLogInfoStmt *sqllib.Stmt
	m                *sync.RWMutex
}

func NewURLogRepositoryCachedSQLite3Impl(ctx context.Context, urlogRepository URLogRepository, cacheDB *sqllib.DB, m *sync.RWMutex, dbName string) (URLogRepository, error) {
	if m == nil {
		m = &sync.RWMutex{}
	}
	var err error

	sql := `
CREATE TABLE IF NOT EXISTS ` + sqlite3impl.QuoteIdent(dbName) + ` (
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table statement %s: %w", dbName, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table statement %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_UNIX") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create urlog index unix statement %s: %w", dbName, err)
		return nil, err
	}
	defer func() {
		err := indexUnixStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexUnixSQL)
	_, err = indexUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create urlog index unix to %s: %w", dbName, err)
		return nil, err
	}

	addURLogInfoSQL := `
INSERT INTO ` + sqlite3impl.QuoteIdent(dbName) + ` (
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", addURLogInfoSQL)
	addURLogInfoStmt, err := cacheDB.PrepareContext(ctx, addURLogInfoSQL)
	if err != nil {
		err = fmt.Errorf("error at add urlog info sql: %w", err)
		return nil, err
	}

	return &urlogRepositoryCachedSQLite3Impl{
		dbName:           dbName,
		urlogRep:         urlogRepository,
		cachedDB:         cacheDB,
		addURLogInfoSQL:  addURLogInfoSQL,
		addURLogInfoStmt: addURLogInfoStmt,
		m:                m,
	}, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = u.UpdateCache(ctx)
		if err != nil {
			repName, _ := u.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}
	u.m.RLock()
	defer u.m.RUnlock()

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
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + `
WHERE
`

	dataType := "urlog"

	tableName := sqlite3impl.QuoteIdent(u.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(u.dbName)
	queryArgs := []interface{}{
		dataType,
	}
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from URLOG: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	kyous := map[string][]Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := Kyou{}
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
				kyous[kyou.ID] = []Kyou{}
			}
			kyous[kyou.ID] = append(kyous[kyou.ID], kyou)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return kyous, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + `
WHERE
`
	dataType := "urlog"

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs:         true,
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UseUpdateTime:  updateTime != nil,
		UpdateTime:     updateTime,
	}

	tableName := sqlite3impl.QuoteIdent(u.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(u.dbName)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from URLOG %s: %w", id, err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	kyous := []Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := Kyou{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(kyous) == 0 {
		return nil, nil
	}
	return &kyous[0], nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + `
WHERE
`
	dataType := "urlog"

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: true,
		IDs:    ids,
	}

	tableName := sqlite3impl.QuoteIdent(u.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(u.dbName)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from URLOG %s: %w", id, err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	kyous := []Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := Kyou{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return kyous, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return u.urlogRep.GetPath(ctx, id)
}

func (u *urlogRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {

	query := &find.FindQuery{
		UpdateCache:    true,
		OnlyLatestData: false,
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

	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at update cache: %w", "error", err)
			}
		}
	}()

	sql := `DELETE FROM ` + sqlite3impl.QuoteIdent(u.dbName)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table statement %s: %w", "memory", err)
		return err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at delete URLOG table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + sqlite3impl.QuoteIdent(u.dbName) + ` (
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add urlog sql: %w", err)
		return err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for _, urlog := range allURLogs {
		select {
		case <-ctx.Done():
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
			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
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
	isCommitted = true
	return nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return u.urlogRep.GetRepName(ctx)
}

func (u *urlogRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	u.m.Lock()
	defer u.m.Unlock()
	if u.addURLogInfoStmt != nil {
		u.addURLogInfoStmt.Close()
	}
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
		_, err = u.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+sqlite3impl.QuoteIdent(u.dbName))
		if err != nil {
			return err
		}
	}
	return nil
}

func (u *urlogRepositoryCachedSQLite3Impl) FindURLog(ctx context.Context, query *find.FindQuery) ([]URLog, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = u.UpdateCache(ctx)
		if err != nil {
			repName, _ := u.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}
	u.m.RLock()
	defer u.m.RUnlock()

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
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + `
WHERE
`

	dataType := "urlog"

	tableName := sqlite3impl.QuoteIdent(u.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(u.dbName)
	queryArgs := []interface{}{
		dataType,
	}
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from URLOG: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	urlogs := []URLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			urlog := URLog{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return urlogs, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetURLog(ctx context.Context, id string, updateTime *time.Time) (*URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + `
WHERE
`

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs:         true,
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UseUpdateTime:  updateTime != nil,
		UpdateTime:     updateTime,
	}
	dataType := "urlog"

	tableName := sqlite3impl.QuoteIdent(u.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(u.dbName)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get urlog histories sql %s: %w", id, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	urlogs := []URLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			urlog := URLog{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(urlogs) == 0 {
		return nil, nil
	}
	return &urlogs[0], nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetURLogHistories(ctx context.Context, id string) ([]URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + `
WHERE
`

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: true,
		IDs:    ids,
	}
	dataType := "urlog"

	tableName := sqlite3impl.QuoteIdent(u.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(u.dbName)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get urlog histories sql %s: %w", id, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	urlogs := []URLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			urlog := URLog{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return urlogs, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) AddURLogInfo(ctx context.Context, urlog URLog) error {
	u.m.Lock()
	defer u.m.Unlock()
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", u.addURLogInfoSQL, queryArgs)
	_, err := u.addURLogInfoStmt.ExecContext(ctx, queryArgs...)

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

func (u *urlogRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	repName, err := u.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	sql := `
SELECT IS_DELETED, ID AS TARGET_ID, NULL AS TARGET_ID_IN_DATA,
       ? AS LATEST_DATA_REPOSITORY_NAME, UPDATE_TIME_UNIX AS DATA_UPDATE_TIME_UNIX
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + ` AS T
WHERE T.UPDATE_TIME_UNIX = (SELECT MAX(UPDATE_TIME_UNIX) FROM ` + sqlite3impl.QuoteIdent(u.dbName) + ` AS INNER_TABLE WHERE INNER_TABLE.ID = T.ID)
`
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, repName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	latestDataRepositoryAddresses := []gkill_cache.LatestDataRepositoryAddress{}
	for rows.Next() {
		addr := gkill_cache.LatestDataRepositoryAddress{}
		var isDeletedInt int
		var dataUpdateTimeUnix int64
		var targetIDInData *string
		err := rows.Scan(&isDeletedInt, &addr.TargetID, &targetIDInData, &addr.LatestDataRepositoryName, &dataUpdateTimeUnix)
		if err != nil {
			return nil, err
		}
		addr.IsDeleted = isDeletedInt != 0
		addr.DataUpdateTime = time.Unix(dataUpdateTimeUnix, 0)
		if targetIDInData != nil {
			addr.TargetID = *targetIDInData
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, addr)
	}
	return latestDataRepositoryAddresses, nil
}
