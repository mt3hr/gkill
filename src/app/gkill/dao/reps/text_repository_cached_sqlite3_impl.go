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

type textRepositoryCachedSQLite3Impl struct {
	dbName                 string
	textRep                TextRepository
	cachedDB               *sql.DB
	getTextsByTargetIDSQL  string
	getTextsByTargetIDStmt *sql.Stmt
	m                      *sync.Mutex
}

func NewTextRepositoryCachedSQLite3Impl(ctx context.Context, textRep TextRepository, cacheDB *sql.DB, m *sync.Mutex, dbName string) (TextRepository, error) {
	if m == nil {
		m = &sync.Mutex{}
	}
	var err error

	sql := `
CREATE TABLE IF NOT EXISTS "` + dbName + `" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
  TEXT NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_DEVICE NOT NULL,
  CREATE_USER NOT NULL,
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
		err = fmt.Errorf("error at create TEXT table statement %s: %w", dbName, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `" ON "` + dbName + `"(ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	gkill_log.TraceSQL.Printf("sql: %s", indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT index statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexUnixStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexUnixSQL)
	_, err = indexUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT index to %s: %w", dbName, err)
		return nil, err
	}

	indexTargetIDUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `_TARGET_ID" ON "` + dbName + `"(TARGET_ID, UPDATE_TIME_UNIX DESC);`
	gkill_log.TraceSQL.Printf("sql: %s", indexTargetIDUnixSQL)
	indexTargetIDUnixStmt, err := cacheDB.PrepareContext(ctx, indexTargetIDUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_TARGET_ID index statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexTargetIDUnixStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexTargetIDUnixSQL)
	_, err = indexTargetIDUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_TARGET_ID index to %s: %w", dbName, err)
		return nil, err
	}

	indexIDUpdateTimeUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `_ID_UPDATE_TIME_UNIX" ON "` + dbName + `"(ID, UPDATE_TIME_UNIX);`
	gkill_log.TraceSQL.Printf("sql: %s", indexIDUpdateTimeUnixSQL)
	indexIDUpdateTimeUnixStmt, err := cacheDB.PrepareContext(ctx, indexIDUpdateTimeUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_ID_UPDATE_TIME_UNIX index statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexIDUpdateTimeUnixStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexIDUpdateTimeUnixSQL)
	_, err = indexIDUpdateTimeUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_ID_UPDATE_TIME_UNIX index to %s: %w", dbName, err)
		return nil, err
	}

	getTextsByTargetIDSQL := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TEXT,
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
FROM ` + dbName + ` AS TEXT1
WHERE TEXT1.TARGET_ID = ?
  AND NOT EXISTS (
    SELECT 1
    FROM ` + dbName + ` AS TEXT2
    WHERE TEXT2.ID = TEXT1.ID
      AND TEXT2.UPDATE_TIME_UNIX > TEXT1.UPDATE_TIME_UNIX
  )
ORDER BY TEXT1.UPDATE_TIME_UNIX DESC
`
	gkill_log.TraceSQL.Printf("sql: %s", getTextsByTargetIDSQL)
	getTextsByTargetIDStmt, err := cacheDB.PrepareContext(ctx, getTextsByTargetIDSQL)
	if err != nil {
		err = fmt.Errorf("error at get get target id sql: %w", err)
		return nil, err
	}

	return &textRepositoryCachedSQLite3Impl{
		dbName:                 dbName,
		textRep:                textRep,
		cachedDB:               cacheDB,
		getTextsByTargetIDSQL:  getTextsByTargetIDSQL,
		getTextsByTargetIDStmt: getTextsByTargetIDStmt,
		m:                      m,
	}, nil
}
func (t *textRepositoryCachedSQLite3Impl) FindTexts(ctx context.Context, query *find.FindQuery) ([]*Text, error) {
	t.m.Lock()
	defer t.m.Unlock()
	var err error

	if query.UseWords != nil && *query.UseWords {
		if query.Words != nil && len(*query.Words) == 0 {
			return []*Text{}, nil
		}
	}

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = t.UpdateCache(ctx)
		if err != nil {
			repName, _ := t.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TEXT,
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
FROM ` + t.dbName + `
WHERE 
`

	dataType := "text"
	queryArgs := []interface{}{
		dataType,
	}

	tableName := t.dbName
	tableNameAlias := t.dbName
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TEXT"}
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
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get TEXT histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from TEXT: %w", err)
		return nil, err
	}
	defer rows.Close()

	texts := []*Text{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			text := &Text{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			dataType := ""

			err = rows.Scan(&text.IsDeleted,
				&text.ID,
				&text.TargetID,
				&text.Text,
				&relatedTimeUnix,
				&createTimeUnix,
				&text.CreateApp,
				&text.CreateDevice,
				&text.CreateUser,
				&updateTimeUnix,
				&text.UpdateApp,
				&text.UpdateDevice,
				&text.UpdateUser,
				&text.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan TEXT: %w", err)
				return nil, err
			}

			text.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			text.CreateTime = time.Unix(createTimeUnix, 0).Local()
			text.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			texts = append(texts, text)
		}
	}
	return texts, nil
}

func (t *textRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	err := t.textRep.Close(ctx)
	if err != nil {
		return err
	}
	if gkill_options.CacheTextReps == nil || !*gkill_options.CacheTextReps {
		err = t.cachedDB.Close()
		if err != nil {
			return err
		}
	} else {
		_, err = t.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+t.dbName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *textRepositoryCachedSQLite3Impl) GetText(ctx context.Context, id string, updateTime *time.Time) (*Text, error) {
	// 最新のデータを返す
	textHistories, err := t.GetTextHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get text histories from TEXT %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(textHistories) == 0 {
		return nil, nil
	}

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range textHistories {
			if kyou.UpdateTime.Unix() == updateTime.Unix() {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return textHistories[0], nil
}

func (t *textRepositoryCachedSQLite3Impl) GetTextsByTargetID(ctx context.Context, target_id string) ([]*Text, error) {
	var err error
	dataType := "text"

	queryArgs := []interface{}{
		dataType,
		target_id,
	}

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", t.getTextsByTargetIDSQL, queryArgs)
	rows, err := t.getTextsByTargetIDStmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from TEXT: %w", err)
		return nil, err
	}
	defer rows.Close()

	texts := []*Text{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			text := &Text{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			dataType := ""

			err = rows.Scan(&text.IsDeleted,
				&text.ID,
				&text.TargetID,
				&text.Text,
				&relatedTimeUnix,
				&createTimeUnix,
				&text.CreateApp,
				&text.CreateDevice,
				&text.CreateUser,
				&updateTimeUnix,
				&text.UpdateApp,
				&text.UpdateDevice,
				&text.UpdateUser,
				&text.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan TEXT: %w", err)
				return nil, err
			}

			text.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			text.CreateTime = time.Unix(createTimeUnix, 0).Local()
			text.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			texts = append(texts, text)
		}
	}
	return texts, nil
}

func (t *textRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	trueValue := true
	falseValue := false
	query := &find.FindQuery{
		UpdateCache:    &trueValue,
		OnlyLatestData: &falseValue,
	}

	allTexts, err := t.textRep.FindTexts(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all texts at update cache: %w", err)
		return err
	}

	t.m.Lock()
	defer t.m.Unlock()

	tx, err := t.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add texts: %w", err)
		return err
	}

	sql := `DELETE FROM ` + t.dbName
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TEXT table statement %s: %w", "memory", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at delete TEXT table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + t.dbName + ` (
  IS_DELETED,
  ID,
  TEXT,
  TARGET_ID,
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
  ?
)`

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add text sql: %w", err)
		return err
	}
	defer insertStmt.Close()

	for _, text := range allTexts {
		select {
		case <-ctx.Done():
			tx.Rollback()
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			queryArgs := []interface{}{
				text.IsDeleted,
				text.ID,
				text.Text,
				text.TargetID,
				text.CreateApp,
				text.CreateDevice,
				text.CreateUser,
				text.UpdateApp,
				text.UpdateDevice,
				text.UpdateUser,
				text.RepName,
				text.RelatedTime.Unix(),
				text.CreateTime.Unix(),
				text.UpdateTime.Unix(),
			}
			gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to TEXT %s: %w", text.ID, err)
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
		err = fmt.Errorf("error at commit transaction for add texts: %w", err)
		return err
	}
	return nil
}

func (t *textRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return t.textRep.GetPath(ctx, id)
}

func (t *textRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return t.textRep.GetRepName(ctx)
}

func (t *textRepositoryCachedSQLite3Impl) GetTextHistories(ctx context.Context, id string) ([]*Text, error) {
	t.m.Lock()
	defer t.m.Unlock()
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TEXT,
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
FROM ` + t.dbName + `
WHERE 
`

	dataType := "text"

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}
	queryArgs := []interface{}{
		dataType,
	}

	tableName := t.dbName
	tableNameAlias := t.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "UPDATE_TIME_UNIX"
	findWordTargetColumns := []string{"TEXT"}
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
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get text histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from TEXT: %w", err)
		return nil, err
	}
	defer rows.Close()

	texts := []*Text{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			text := &Text{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			dataType := ""

			err = rows.Scan(&text.IsDeleted,
				&text.ID,
				&text.TargetID,
				&text.Text,
				&relatedTimeUnix,
				&createTimeUnix,
				&text.CreateApp,
				&text.CreateDevice,
				&text.CreateUser,
				&updateTimeUnix,
				&text.UpdateApp,
				&text.UpdateDevice,
				&text.UpdateUser,
				&text.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan TEXT: %w", err)
				return nil, err
			}

			text.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			text.CreateTime = time.Unix(createTimeUnix, 0).Local()
			text.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			texts = append(texts, text)
		}
	}
	return texts, nil
}
func (t *textRepositoryCachedSQLite3Impl) AddTextInfo(ctx context.Context, text *Text) error {
	t.m.Lock()
	defer t.m.Unlock()
	sql := `
INSERT INTO ` + t.dbName + ` (
  IS_DELETED,
  ID,
  TEXT,
  TARGET_ID,
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
  ?
)`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add text sql %s: %w", text.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		text.IsDeleted,
		text.ID,
		text.Text,
		text.TargetID,
		text.CreateApp,
		text.CreateDevice,
		text.CreateUser,
		text.UpdateApp,
		text.UpdateDevice,
		text.UpdateUser,
		text.RepName,
		text.RelatedTime.Unix(),
		text.CreateTime.Unix(),
		text.UpdateTime.Unix(),
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to TEXT %s: %w", text.ID, err)
		return err
	}
	return nil
}

func (m *textRepositoryCachedSQLite3Impl) UnWrapTyped() ([]TextRepository, error) {
	return []TextRepository{m.textRep}, nil
}

func (t *textRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error) {
	latestData, err := t.textRep.GetLatestDataRepositoryAddress(ctx, updateCache)
	if err != nil {
		return nil, err
	}
	if updateCache {
		err = t.UpdateCache(ctx)
		if err != nil {
			return nil, err
		}
	}
	return latestData, nil
}
