package reps

import (
	"context"
	"database/sql"
	sqllib "database/sql"
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

type lantanaRepositoryCachedSQLite3Impl struct {
	dbName     string
	lantanaRep LantanaRepository
	cachedDB   *sqllib.DB
	m          *sync.Mutex
}

func NewLantanaRepositoryCachedSQLite3Impl(ctx context.Context, lantanaRep LantanaRepository, cacheDB *sql.DB, m *sync.Mutex, dbName string) (LantanaRepository, error) {
	if m == nil {
		m = &sync.Mutex{}
	}
	var err error
	sql := `
CREATE TABLE IF NOT EXISTS "` + dbName + `" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  MOOD NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  REP_NAME TEXT NOT NULL,
  RELATED_TIME_UNIX NOT NULL,
  CREATE_TIME_UNIX NOT NULL,
  UPDATE_TIME_UNIX NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create LANTANA table statement %s: %w", dbName, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create LANTANA table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `_UNIX" ON "` + dbName + `"(ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	gkill_log.TraceSQL.Printf("sql: %s", indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create lantana index unix statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexUnixStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexUnixSQL)
	_, err = indexUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create lantana index unix to %s: %w", dbName, err)
		return nil, err
	}

	return &lantanaRepositoryCachedSQLite3Impl{
		dbName:     dbName,
		lantanaRep: lantanaRep,
		cachedDB:   cacheDB,
		m:          m,
	}, nil
}
func (l *lantanaRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	l.m.Lock()
	defer l.m.Unlock()
	var err error
	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = l.UpdateCache(ctx)
		if err != nil {
			repName, _ := l.GetRepName(ctx)
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
FROM ` + l.dbName + `
WHERE
`
	dataType := "lantana"

	queryArgs := []interface{}{
		dataType,
	}

	tableName := l.dbName
	tableNameAlias := l.dbName
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
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
	stmt, err := l.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from LANTANA: %w", err)
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
				err = fmt.Errorf("error at scan from LANTANA: %w", err)
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

func (l *lantanaRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := l.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from LANTANA %s: %w", id, err)
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

func (l *lantanaRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	l.m.Lock()
	defer l.m.Unlock()
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
FROM ` + l.dbName + `
WHERE 
`

	dataType := "lantana"

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	queryArgs := []interface{}{
		dataType,
	}

	tableName := l.dbName
	tableNameAlias := l.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from LANTANA %s: %w", id, err)
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
				err = fmt.Errorf("error at scan from LANTANA %s: %w", id, err)
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

func (l *lantanaRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return l.lantanaRep.GetPath(ctx, id)
}

func (l *lantanaRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	trueValue := true
	falseValue := false
	query := &find.FindQuery{
		UpdateCache:    &trueValue,
		OnlyLatestData: &falseValue,
	}

	allLantanas, err := l.lantanaRep.FindLantana(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all lantanas at update cache: %w", err)
		return err
	}

	l.m.Lock()
	defer l.m.Unlock()

	tx, err := l.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add lantana: %w", err)
		return err
	}

	sql := `DELETE FROM ` + l.dbName
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create LANTANA table statement %s: %w", "memory", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at delete LANTANA table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + l.dbName + ` (
  IS_DELETED,
  ID,
  MOOD,
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
  ?
)`

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add lantana sql: %w", err)
		return err
	}
	defer insertStmt.Close()

	for _, lantana := range allLantanas {
		select {
		case <-ctx.Done():
			tx.Rollback()
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			queryArgs := []interface{}{
				lantana.IsDeleted,
				lantana.ID,
				lantana.Mood,
				lantana.CreateApp,
				lantana.CreateDevice,
				lantana.CreateUser,
				lantana.UpdateApp,
				lantana.UpdateDevice,
				lantana.UpdateUser,
				lantana.RepName,
				lantana.RelatedTime.Unix(),
				lantana.CreateTime.Unix(),
				lantana.UpdateTime.Unix(),
			}
			gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to LANTANA %s: %w", lantana.ID, err)
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
		err = fmt.Errorf("error at commit transaction for add timeiss: %w", err)
		return err
	}

	return nil
}

func (l *lantanaRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return l.lantanaRep.GetRepName(ctx)
}

func (l *lantanaRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	err := l.lantanaRep.Close(ctx)
	if err != nil {
		return err
	}
	if gkill_options.CacheLantanaReps == nil || !*gkill_options.CacheLantanaReps {
		err = l.cachedDB.Close()
		if err != nil {
			return err
		}
	} else {
		_, err = l.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+l.dbName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *lantanaRepositoryCachedSQLite3Impl) FindLantana(ctx context.Context, query *find.FindQuery) ([]*Lantana, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = l.UpdateCache(ctx)
		if err != nil {
			repName, _ := l.GetRepName(ctx)
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
  MOOD,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + l.dbName + `
WHERE
`
	dataType := "lantana"

	queryArgs := []interface{}{
		dataType,
	}

	tableName := l.dbName
	tableNameAlias := l.dbName
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
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
	stmt, err := l.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from LANTANA: %w", err)
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
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&lantana.IsDeleted,
				&lantana.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&lantana.CreateApp,
				&lantana.CreateDevice,
				&lantana.CreateUser,
				&updateTimeUnix,
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

			lantana.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			lantana.CreateTime = time.Unix(createTimeUnix, 0).Local()
			lantana.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			lantanas = append(lantanas, lantana)
		}
	}
	return lantanas, nil
}

func (l *lantanaRepositoryCachedSQLite3Impl) GetLantana(ctx context.Context, id string, updateTime *time.Time) (*Lantana, error) {
	// 最新のデータを返す
	lantanaHistories, err := l.GetLantanaHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get lantana histories from LANTANA %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(lantanaHistories) == 0 {
		return nil, nil
	}

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range lantanaHistories {
			if kyou.UpdateTime.Unix() == updateTime.Unix() {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return lantanaHistories[0], nil
}

func (l *lantanaRepositoryCachedSQLite3Impl) GetLantanaHistories(ctx context.Context, id string) ([]*Lantana, error) {
	l.m.Lock()
	defer l.m.Unlock()
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
  MOOD,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + l.dbName + `
WHERE
`

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	dataType := "lantana"

	queryArgs := []interface{}{
		dataType,
	}

	tableName := l.dbName
	tableNameAlias := l.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get lantana histories sql %s: %w", id, err)
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
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&lantana.IsDeleted,
				&lantana.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&lantana.CreateApp,
				&lantana.CreateDevice,
				&lantana.CreateUser,
				&updateTimeUnix,
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

			lantana.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			lantana.CreateTime = time.Unix(createTimeUnix, 0).Local()
			lantana.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			lantanas = append(lantanas, lantana)
		}
	}
	return lantanas, nil
}

func (l *lantanaRepositoryCachedSQLite3Impl) AddLantanaInfo(ctx context.Context, lantana *Lantana) error {
	l.m.Lock()
	defer l.m.Unlock()
	sql := `
INSERT INTO ` + l.dbName + ` (
  IS_DELETED,
  ID,
  MOOD,
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
  ?
)`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add lantana sql %s: %w", lantana.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		lantana.IsDeleted,
		lantana.ID,
		lantana.Mood,
		lantana.CreateApp,
		lantana.CreateDevice,
		lantana.CreateUser,
		lantana.UpdateApp,
		lantana.UpdateDevice,
		lantana.UpdateUser,
		lantana.RepName,
		lantana.RelatedTime.Unix(),
		lantana.CreateTime.Unix(),
		lantana.UpdateTime.Unix(),
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to LANTANA %s: %w", lantana.ID, err)
		return err
	}
	return nil
}

func (l *lantanaRepositoryCachedSQLite3Impl) UnWrapTyped() ([]LantanaRepository, error) {
	return l.lantanaRep.UnWrapTyped()
}

func (l *lantanaRepositoryCachedSQLite3Impl) UnWrap() ([]Repository, error) {
	return l.lantanaRep.UnWrap()
}

func (l *lantanaRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error) {
	latestData, err := l.lantanaRep.GetLatestDataRepositoryAddress(ctx, updateCache)
	if err != nil {
		return nil, err
	}
	if updateCache {
		err = l.UpdateCache(ctx)
		if err != nil {
			return nil, err
		}
	}
	return latestData, nil
}
