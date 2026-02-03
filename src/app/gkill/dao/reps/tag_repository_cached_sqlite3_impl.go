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

type tagRepositoryCachedSQLite3Impl struct {
	tagRep                TagRepository
	cachedDB              *sql.DB
	dbName                string
	getTagsByTargetIDSQL  string
	getTagsByTargetIDStmt *sql.Stmt
	m                     *sync.Mutex
}

func NewTagRepositoryCachedSQLite3Impl(ctx context.Context, tagRep TagRepository, cacheDB *sql.DB, m *sync.Mutex, dbName string) (TagRepository, error) {
	if m == nil {
		m = &sync.Mutex{}
	}
	var err error
	sql := `
CREATE TABLE IF NOT EXISTS "` + dbName + `" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
  TAG NOT NULL,
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
		err = fmt.Errorf("error at create TAG table statement %s: %w", dbName, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG table to %s: %w", dbName, err)
		return nil, err
	}

	indexTargetIDUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `_TARGET_ID_UNIX" ON "` + dbName + `"(TARGET_ID, UPDATE_TIME_UNIX DESC);`
	gkill_log.TraceSQL.Printf("sql: %s", indexTargetIDUnixSQL)
	indexTargetIDUnixStmt, err := cacheDB.PrepareContext(ctx, indexTargetIDUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG_TARGET_ID_UNIX index statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexTargetIDUnixStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexTargetIDUnixSQL)
	_, err = indexTargetIDUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_TARGET_ID_UNIX index to %s: %w", dbName, err)
		return nil, err
	}

	indexIDUpdateTimeUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `_ID_UPDATE_TIME_UNIX" ON "` + dbName + `"(ID, UPDATE_TIME_UNIX);`
	gkill_log.TraceSQL.Printf("sql: %s", indexIDUpdateTimeUnixSQL)
	indexIDUpdateTimeUnixStmt, err := cacheDB.PrepareContext(ctx, indexIDUpdateTimeUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME_UNIX index statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexIDUpdateTimeUnixStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexIDUpdateTimeUnixSQL)
	_, err = indexIDUpdateTimeUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME_UNIX index to %s: %w", dbName, err)
		return nil, err
	}

	getTagsByTargetIDSQL := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TAG,
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
FROM ` + dbName + ` AS TAG1
WHERE TAG1.TARGET_ID = ?
  AND NOT EXISTS (
    SELECT 1
    FROM ` + dbName + ` AS TAG2
    WHERE TAG2.ID = TAG1.ID
      AND TAG2.UPDATE_TIME_UNIX > TAG1.UPDATE_TIME_UNIX
  )
ORDER BY TAG1.UPDATE_TIME_UNIX DESC
`
	gkill_log.TraceSQL.Printf("sql: %s", getTagsByTargetIDSQL)
	getTagsByTargetIDStmt, err := cacheDB.PrepareContext(ctx, getTagsByTargetIDSQL)
	if err != nil {
		err = fmt.Errorf("error at get get target id sql: %w", err)
		return nil, err
	}

	cachedTagrepository := &tagRepositoryCachedSQLite3Impl{
		tagRep:                tagRep,
		dbName:                dbName,
		cachedDB:              cacheDB,
		getTagsByTargetIDSQL:  getTagsByTargetIDSQL,
		getTagsByTargetIDStmt: getTagsByTargetIDStmt,
		m:                     m,
	}
	return cachedTagrepository, nil
}
func (t *tagRepositoryCachedSQLite3Impl) FindTags(ctx context.Context, query *find.FindQuery) ([]*Tag, error) {
	t.m.Lock()
	defer t.m.Unlock()
	var err error

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
  TAG,
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
	dataType := "tag"
	queryArgs := []interface{}{
		dataType,
	}

	tableName := t.dbName
	tableNameAlias := t.dbName
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TAG"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := false
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
		err = fmt.Errorf("error at get find tags sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from TAG: %w", err)
		return nil, err
	}
	defer rows.Close()

	tags := []*Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := &Tag{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeUnix,
				&createTimeUnix,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeUnix,
				&tag.UpdateApp,
				&tag.UpdateDevice,
				&tag.UpdateUser,
				&tag.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at read rows at find tags: %w", err)
				return nil, err
			}

			tag.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			tag.CreateTime = time.Unix(createTimeUnix, 0).Local()
			tag.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	err := t.tagRep.Close(ctx)
	if err != nil {
		return err
	}
	if gkill_options.CacheTagReps == nil || !*gkill_options.CacheTagReps {
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

func (t *tagRepositoryCachedSQLite3Impl) GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error) {
	// 最新のデータを返す
	tagHistories, err := t.GetTagHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get tag histories from TAG %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(tagHistories) == 0 {
		return nil, nil
	}

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range tagHistories {
			if kyou.UpdateTime.Unix() == updateTime.Unix() {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return tagHistories[0], nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetTagsByTagName(ctx context.Context, tagname string) ([]*Tag, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TAG,
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

	dataType := "tag"

	trueValue := true
	words := []string{tagname}

	query := &find.FindQuery{
		UseWords: &trueValue,
		Words:    &words,
	}
	queryArgs := []interface{}{
		dataType,
	}

	tableName := t.dbName
	tableNameAlias := t.dbName
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "UPDATE_TIME_UNIX"
	findWordTargetColumns := []string{"TAG"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := false
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get tag by name sql %s: %w", tagname, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from TAG: %w", err)
		return nil, err
	}
	defer rows.Close()

	tags := []*Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := &Tag{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeUnix,
				&createTimeUnix,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeUnix,
				&tag.UpdateApp,
				&tag.UpdateDevice,
				&tag.UpdateUser,
				&tag.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at read rows at get tag by name %s: %w", tagname, err)
				return nil, err
			}

			tag.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			tag.CreateTime = time.Unix(createTimeUnix, 0).Local()
			tag.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetTagsByTargetID(ctx context.Context, target_id string) ([]*Tag, error) {
	var err error
	dataType := "tag"

	queryArgs := []interface{}{
		dataType,
		target_id,
	}

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", t.getTagsByTargetIDSQL, queryArgs)
	rows, err := t.getTagsByTargetIDStmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from TAG: %w", err)
		return nil, err
	}
	defer rows.Close()

	tags := []*Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := &Tag{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeUnix,
				&createTimeUnix,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeUnix,
				&tag.UpdateApp,
				&tag.UpdateDevice,
				&tag.UpdateUser,
				&tag.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at read rows at get target id %s: %w", target_id, err)
				return nil, err
			}

			tag.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			tag.CreateTime = time.Unix(createTimeUnix, 0).Local()
			tag.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	allTags, err := t.tagRep.GetAllTags(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all tags at update cache: %w", err)
		return err
	}

	t.m.Lock()
	defer t.m.Unlock()

	tx, err := t.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add tags: %w", err)
		return err
	}

	sql := `DELETE FROM ` + t.dbName
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TAG table statement %s: %w", "memory", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at delete TAG table: %w", err)
		return err
	}

	sql = `
INSERT INTO "` + t.dbName + `" (
  IS_DELETED,
  ID,
  TAG,
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
		err = fmt.Errorf("error at add tag sql %s: %w", sql, err)
		return err
	}
	defer insertStmt.Close()

	for _, tag := range allTags {
		select {
		case <-ctx.Done():
			tx.Rollback()
			err = ctx.Err()
			return err
		default:
		}
		func() error {
			queryArgs := []interface{}{
				tag.IsDeleted,
				tag.ID,
				tag.Tag,
				tag.TargetID,
				tag.CreateApp,
				tag.CreateDevice,
				tag.CreateUser,
				tag.UpdateApp,
				tag.UpdateDevice,
				tag.UpdateUser,
				tag.RepName,
				tag.RelatedTime.Unix(),
				tag.CreateTime.Unix(),
				tag.UpdateTime.Unix(),
			}
			gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to TAG %s: %w", tag.ID, err)
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
		err = fmt.Errorf("error at commit transaction for add tags: %w", err)
		return err
	}
	return nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return t.tagRep.GetPath(ctx, id)
}

func (t *tagRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return t.tagRep.GetRepName(ctx)
}

func (t *tagRepositoryCachedSQLite3Impl) GetTagHistories(ctx context.Context, id string) ([]*Tag, error) {
	t.m.Lock()
	defer t.m.Unlock()
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TAG,
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

	dataType := "tag"

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
	findWordTargetColumns := []string{"TAG"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := false
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get tag histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from TAG: %w", err)
		return nil, err
	}
	defer rows.Close()

	tags := []*Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := &Tag{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeUnix,
				&createTimeUnix,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeUnix,
				&tag.UpdateApp,
				&tag.UpdateDevice,
				&tag.UpdateUser,
				&tag.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at read rows at get tag histories: %w", err)
				return nil, err
			}

			tag.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			tag.CreateTime = time.Unix(createTimeUnix, 0).Local()
			tag.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositoryCachedSQLite3Impl) AddTagInfo(ctx context.Context, tag *Tag) error {
	t.m.Lock()
	defer t.m.Unlock()
	sql := `
INSERT INTO ` + t.dbName + ` (
  IS_DELETED,
  ID,
  TAG,
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
		err = fmt.Errorf("error at add tag sql %s: %w", tag.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		tag.IsDeleted,
		tag.ID,
		tag.Tag,
		tag.TargetID,
		tag.CreateApp,
		tag.CreateDevice,
		tag.CreateUser,
		tag.UpdateApp,
		tag.UpdateDevice,
		tag.UpdateUser,
		tag.RepName,
		tag.RelatedTime.Unix(),
		tag.CreateTime.Unix(),
		tag.UpdateTime.Unix(),
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to TAG %s: %w", tag.ID, err)
		return err
	}
	return nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetAllTagNames(ctx context.Context) ([]string, error) {
	t.m.Lock()
	defer t.m.Unlock()
	var err error

	sql := `
SELECT 
  DISTINCT TAG
FROM ` + t.dbName + `
`
	tableName := t.dbName
	tableNameAlias := t.dbName
	sql += fmt.Sprintf(" WHERE UPDATE_TIME_UNIX = ( SELECT MAX(UPDATE_TIME_UNIX) FROM %s AS INNER_TABLE WHERE ID = %s.ID )", tableName, tableNameAlias)
	sql += fmt.Sprintf(" GROUP BY TAG ")

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all tag names sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at select all tag names from TAG: %w", err)
		return nil, err
	}
	defer rows.Close()

	tagNamesMap := map[string]struct{}{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tagName := ""
			err = rows.Scan(
				&tagName,
			)
			if err != nil {
				err = fmt.Errorf("error at read rows at get all tag names: %w", err)
				return nil, err
			}

			tagNamesMap[tagName] = struct{}{}
		}
	}
	tagNames := []string{}
	for tagName := range tagNamesMap {
		tagNames = append(tagNames, tagName)
	}
	return tagNames, nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetAllTags(ctx context.Context) ([]*Tag, error) {
	t.m.Lock()
	defer t.m.Unlock()
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TAG,
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

	dataType := "tag"

	query := &find.FindQuery{}
	queryArgs := []interface{}{
		dataType,
	}

	tableName := t.dbName
	tableNameAlias := t.dbName
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := true
	findWordUseLike := false
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all tags sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from TAG: %w", err)
		return nil, err
	}
	defer rows.Close()

	tags := []*Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := &Tag{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeUnix,
				&createTimeUnix,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeUnix,
				&tag.UpdateApp,
				&tag.UpdateDevice,
				&tag.UpdateUser,
				&tag.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at read rows at get all tags: %w", err)
				return nil, err
			}

			tag.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			tag.CreateTime = time.Unix(createTimeUnix, 0).Local()
			tag.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositoryCachedSQLite3Impl) UnWrapTyped() ([]TagRepository, error) {
	return []TagRepository{t.tagRep}, nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error) {
	latestData, err := t.tagRep.GetLatestDataRepositoryAddress(ctx, updateCache)
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
