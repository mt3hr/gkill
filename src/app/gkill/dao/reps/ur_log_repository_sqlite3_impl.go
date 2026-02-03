package reps

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type urlogRepositorySQLite3Impl struct {
	filename    string
	db          *sql.DB
	m           *sync.Mutex
	fullConnect bool
}

func NewURLogRepositorySQLite3Impl(ctx context.Context, filename string, fullConnect bool) (URLogRepository, error) {
	db, err := sqlite3impl.GetSQLiteDBConnection(ctx, filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "URLOG" (
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
  UPDATE_USER NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_URLOG ON URLOG (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create URLOG index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create URLOG index to %s: %w", filename, err)
		return nil, err
	}

	dbName := "URLOG"
	latestIndexSQL := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS INDEX_FOR_LATEST_DATA_REPOSITORY_ADDRESS ON %s(ID, UPDATE_TIME);`, dbName)
	gkill_log.TraceSQL.Printf("sql: %s", latestIndexSQL)
	latestIndexStmt, err := db.PrepareContext(ctx, latestIndexSQL)
	if err != nil {
		err = fmt.Errorf("error at create index for latest data repository address at %s index statement %s: %w", dbName, filename, err)
		return nil, err
	}
	defer latestIndexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", latestIndexSQL)
	_, err = latestIndexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create %s index for latest data repository address to %s: %w", dbName, filename, err)
		return nil, err
	}

	if !fullConnect {
		err = db.Close()
		if err != nil {
			return nil, err
		}
		db = nil
	}

	return &urlogRepositorySQLite3Impl{
		filename:    filename,
		db:          db,
		m:           &sync.Mutex{},
		fullConnect: fullConnect,
	}, nil
}

func (u *urlogRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	var err error
	var db *sql.DB
	if u.fullConnect {
		db = u.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, u.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

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
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM URLOG
WHERE
`

	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at urlog: %w", err)
		return nil, err
	}
	dataType := "urlog"

	tableName := "URLOG"
	tableNameAlias := "URLOG"
	queryArgs := []interface{}{
		repName,
		dataType,
	}
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
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
			kyou.RepName = repName
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

func (u *urlogRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
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

func (u *urlogRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	var err error
	var db *sql.DB
	if u.fullConnect {
		db = u.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, u.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

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
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM URLOG 
WHERE 
`
	dataType := "urlog"

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	tableName := "URLOG"
	tableNameAlias := "URLOG"
	queryArgs := []interface{}{
		repName,
		dataType,
	}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
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
			kyou.RepName = repName
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

func (u *urlogRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if id == "" {
		return u.filename, nil
	}
	return filepath.Abs(u.filename)
}

func (u *urlogRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (u *urlogRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := u.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path urlog rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil

}

func (u *urlogRepositorySQLite3Impl) Close(ctx context.Context) error {
	if u.fullConnect {
		return u.db.Close()
	}
	return nil
}

func (u *urlogRepositorySQLite3Impl) FindURLog(ctx context.Context, query *find.FindQuery) ([]*URLog, error) {
	var err error
	var db *sql.DB
	if u.fullConnect {
		db = u.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, u.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

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
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM URLOG
WHERE
`

	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at URLOG: %w", err)
		return nil, err
	}
	dataType := "urlog"

	tableName := "URLOG"
	tableNameAlias := "URLOG"
	queryArgs := []interface{}{
		repName,
		dataType,
	}
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
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

func (u *urlogRepositorySQLite3Impl) GetURLog(ctx context.Context, id string, updateTime *time.Time) (*URLog, error) {
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

func (u *urlogRepositorySQLite3Impl) GetURLogHistories(ctx context.Context, id string) ([]*URLog, error) {
	var err error
	var db *sql.DB
	if u.fullConnect {
		db = u.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, u.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

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
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM URLOG
WHERE
`

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}
	dataType := "urlog"

	tableName := "URLOG"
	tableNameAlias := "URLOG"
	queryArgs := []interface{}{
		repName,
		dataType,
	}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
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

func (u *urlogRepositorySQLite3Impl) AddURLogInfo(ctx context.Context, urlog *URLog) error {
	u.m.Lock()
	defer u.m.Unlock()
	var err error
	var db *sql.DB
	if u.fullConnect {
		db = u.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, u.filename)
		if err != nil {
			return err
		}
		defer db.Close()
	}

	sql := `
INSERT INTO URLOG (
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
  UPDATE_USER
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
	stmt, err := db.PrepareContext(ctx, sql)
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
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to URLog %s: %w", urlog.ID, err)
		return err
	}
	return nil
}

func (u *urlogRepositorySQLite3Impl) UnWrapTyped() ([]URLogRepository, error) {
	return []URLogRepository{u}, nil
}

func (u *urlogRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{u}, nil
}

func (u *urlogRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error) {
	dbName := "URLOG"
	var err error
	var db *sql.DB
	if u.fullConnect {
		db = u.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, u.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	// update_cacheであればキャッシュを更新する
	if updateCache {
		err = u.UpdateCache(ctx)
		if err != nil {
			repName, _ := u.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

	sql := fmt.Sprintf(`
SELECT
  tbl.IS_DELETED,
  tbl.ID AS TARGET_ID,
  NULL AS TARGET_ID_IN_DATA,
  ? AS LATEST_DATA_REPOSITORY_NAME,
  tbl.UPDATE_TIME AS DATA_UPDATE_TIME
FROM %s tbl
INNER JOIN (
  SELECT ID, MAX(UPDATE_TIME) AS UPDATE_TIME
  FROM %s
  GROUP BY ID
) joined
ON joined.ID = tbl.ID AND joined.UPDATE_TIME = tbl.UPDATE_TIME;
`,
		dbName, dbName)

	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at %s : %w", u.filename, err)
		return nil, err
	}

	queryArgs := []interface{}{
		repName,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get latest data repository address sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from latest data repository address at %s: %w", repName, err)
		return nil, err
	}
	defer rows.Close()

	latestDataRepositoryAddresses := []*gkill_cache.LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			latestDataRepositoryAddress := &gkill_cache.LatestDataRepositoryAddress{}
			dataUpdateTimeStr := ""

			err = rows.Scan(
				&latestDataRepositoryAddress.IsDeleted,
				&latestDataRepositoryAddress.TargetID,
				&latestDataRepositoryAddress.TargetIDInData,
				&latestDataRepositoryAddress.LatestDataRepositoryName,
				&dataUpdateTimeStr,
			)
			if err != nil {
				err = fmt.Errorf("error at scan latest data repository address at %s: %w", repName, err)
				return nil, err
			}
			latestDataRepositoryAddress.DataUpdateTime, err = time.Parse(sqlite3impl.TimeLayout, dataUpdateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse data update time %s in %s: %w", dataUpdateTimeStr, repName, err)
				return nil, err
			}

			latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
		}
	}
	return latestDataRepositoryAddresses, nil
}
