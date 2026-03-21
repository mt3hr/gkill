package reps

import (
	"context"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	sqllib "database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

type tagRepositoryCachedSQLite3Impl struct {
	tagRep                TagRepository
	cachedDB              *sqllib.DB
	dbName                string
	getTagsByTargetIDSQL  string
	getTagsByTargetIDStmt *sqllib.Stmt
	addTagInfoSQL         string
	addTagInfoStmt        *sqllib.Stmt
	getAllTagNamesSQL     string
	getAllTagNamesStmt    *sqllib.Stmt
	m                     *sync.RWMutex
}

func NewTagRepositoryCachedSQLite3Impl(ctx context.Context, tagRep TagRepository, cacheDB *sqllib.DB, m *sync.RWMutex, dbName string) (TagRepository, error) {
	if m == nil {
		m = &sync.RWMutex{}
	}
	var err error
	sql := `
CREATE TABLE IF NOT EXISTS ` + sqlite3impl.QuoteIdent(dbName) + ` (
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TAG table statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create TAG table to %s: %w", dbName, err)
		return nil, err
	}

	indexTargetIDUnixSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_TARGET_ID_UNIX") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(TARGET_ID, UPDATE_TIME_UNIX DESC);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexTargetIDUnixSQL)
	indexTargetIDUnixStmt, err := cacheDB.PrepareContext(ctx, indexTargetIDUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG_TARGET_ID_UNIX index statement %s: %w", dbName, err)
		return nil, err
	}
	defer func() {
		err := indexTargetIDUnixStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexTargetIDUnixSQL)
	_, err = indexTargetIDUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_TARGET_ID_UNIX index to %s: %w", dbName, err)
		return nil, err
	}

	indexIDUpdateTimeUnixSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_ID_UPDATE_TIME_UNIX") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(ID, UPDATE_TIME_UNIX);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexIDUpdateTimeUnixSQL)
	indexIDUpdateTimeUnixStmt, err := cacheDB.PrepareContext(ctx, indexIDUpdateTimeUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME_UNIX index statement %s: %w", dbName, err)
		return nil, err
	}
	defer func() {
		err := indexIDUpdateTimeUnixStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexIDUpdateTimeUnixSQL)
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
FROM ` + sqlite3impl.QuoteIdent(dbName) + ` AS TAG1
WHERE TAG1.TARGET_ID = ?
  AND NOT EXISTS (
    SELECT 1
    FROM ` + sqlite3impl.QuoteIdent(dbName) + ` AS TAG2
    WHERE TAG2.ID = TAG1.ID
      AND TAG2.UPDATE_TIME_UNIX > TAG1.UPDATE_TIME_UNIX
  )
ORDER BY TAG1.UPDATE_TIME_UNIX DESC
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", getTagsByTargetIDSQL)
	getTagsByTargetIDStmt, err := cacheDB.PrepareContext(ctx, getTagsByTargetIDSQL)
	if err != nil {
		err = fmt.Errorf("error at get target id sql: %w", err)
		return nil, err
	}

	addTagInfoSQL := `
INSERT INTO ` + sqlite3impl.QuoteIdent(dbName) + ` (
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", addTagInfoSQL)
	addTagInfoStmt, err := cacheDB.PrepareContext(ctx, addTagInfoSQL)
	if err != nil {
		err = fmt.Errorf("error at add tag info sql: %w", err)
		return nil, err
	}

	getAllTagNamesTableName := sqlite3impl.QuoteIdent(dbName)
	getAllTagNamesSQL := `
SELECT
  DISTINCT TAG
FROM ` + getAllTagNamesTableName + `
` + fmt.Sprintf(" WHERE UPDATE_TIME_UNIX = ( SELECT MAX(UPDATE_TIME_UNIX) FROM %s AS INNER_TABLE WHERE ID = %s.ID )", getAllTagNamesTableName, getAllTagNamesTableName) + " GROUP BY TAG "
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", getAllTagNamesSQL)
	getAllTagNamesStmt, err := cacheDB.PrepareContext(ctx, getAllTagNamesSQL)
	if err != nil {
		err = fmt.Errorf("error at get all tag names sql: %w", err)
		return nil, err
	}

	cachedTagrepository := &tagRepositoryCachedSQLite3Impl{
		tagRep:                tagRep,
		dbName:                dbName,
		cachedDB:              cacheDB,
		getTagsByTargetIDSQL:  getTagsByTargetIDSQL,
		getTagsByTargetIDStmt: getTagsByTargetIDStmt,
		addTagInfoSQL:         addTagInfoSQL,
		addTagInfoStmt:        addTagInfoStmt,
		getAllTagNamesSQL:     getAllTagNamesSQL,
		getAllTagNamesStmt:    getAllTagNamesStmt,
		m:                     m,
	}
	return cachedTagrepository, nil
}
func (t *tagRepositoryCachedSQLite3Impl) FindTags(ctx context.Context, query *find.FindQuery) ([]Tag, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = t.UpdateCache(ctx)
		if err != nil {
			repName, _ := t.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}
	t.m.RLock()
	defer t.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(t.dbName) + `
WHERE
`
	dataType := "tag"
	queryArgs := []interface{}{
		dataType,
	}

	tableName := sqlite3impl.QuoteIdent(t.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(t.dbName)
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TAG"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := false
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find tags sql: %w", err)
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
		err = fmt.Errorf("error at select from TAG: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	tags := []Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := Tag{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return tags, nil
}

func (t *tagRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	t.m.Lock()
	defer t.m.Unlock()
	if t.addTagInfoStmt != nil {
		t.addTagInfoStmt.Close()
	}
	if t.getAllTagNamesStmt != nil {
		t.getAllTagNamesStmt.Close()
	}
	if t.getTagsByTargetIDStmt != nil {
		t.getTagsByTargetIDStmt.Close()
	}
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
		_, err = t.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+sqlite3impl.QuoteIdent(t.dbName))
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(t.dbName) + `
WHERE
`

	dataType := "tag"

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs:         true,
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UseUpdateTime:  updateTime != nil,
		UpdateTime:     updateTime,
	}
	queryArgs := []interface{}{
		dataType,
	}

	tableName := sqlite3impl.QuoteIdent(t.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(t.dbName)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get tag histories sql: %w", err)
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
		err = fmt.Errorf("error at select from TAG: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	tags := []Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := Tag{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(tags) == 0 {
		return nil, nil
	}
	return &tags[0], nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetTagsByTagName(ctx context.Context, tagname string) ([]Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(t.dbName) + `
WHERE
`

	dataType := "tag"

	words := []string{tagname}

	query := &find.FindQuery{
		UseWords: true,
		Words:    words,
	}
	queryArgs := []interface{}{
		dataType,
	}

	tableName := sqlite3impl.QuoteIdent(t.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(t.dbName)
	whereCounter := 0
	var onlyLatestData bool
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get tag by name sql %s: %w", tagname, err)
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
		err = fmt.Errorf("error at select from TAG: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	tags := []Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := Tag{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return tags, nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetTagsByTargetID(ctx context.Context, target_id string) ([]Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error
	dataType := "tag"

	queryArgs := []interface{}{
		dataType,
		target_id,
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", t.getTagsByTargetIDSQL, queryArgs)
	rows, err := t.getTagsByTargetIDStmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from TAG: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	tags := []Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := Tag{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return tags, nil
}

func (t *tagRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	err := t.tagRep.UpdateCache(ctx)
	if err != nil {
		return fmt.Errorf("error at update underlying tag rep cache: %w", err)
	}

	// 下層リポジトリに変更がなければフルリビルドをスキップ
	if !t.tagRep.LastUpdateCacheChanged() {
		return nil
	}

	query := &find.FindQuery{
		UpdateCache:        false,
		OnlyLatestData:     false,
		IncludeDeletedData: true,
	}

	allTags, err := t.tagRep.FindTags(ctx, query)
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

	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at update cache: %w", "error", err)
			}
		}
	}()

	sql := `DELETE FROM ` + sqlite3impl.QuoteIdent(t.dbName)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TAG table statement %s: %w", "memory", err)
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
		err = fmt.Errorf("error at delete TAG table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + sqlite3impl.QuoteIdent(t.dbName) + ` (
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add tag sql %s: %w", sql, err)
		return err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for _, tag := range allTags {
		select {
		case <-ctx.Done():
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
			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to TAG %s: %w", tag.ID, err)
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
		err = fmt.Errorf("error at commit transaction for add tags: %w", err)
		return err
	}
	isCommitted = true
	return nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return t.tagRep.GetPath(ctx, id)
}

func (t *tagRepositoryCachedSQLite3Impl) LastUpdateCacheChanged() bool {
	return true
}

func (t *tagRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return t.tagRep.GetRepName(ctx)
}

func (t *tagRepositoryCachedSQLite3Impl) GetTagHistories(ctx context.Context, id string) ([]Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(t.dbName) + `
WHERE
`

	dataType := "tag"

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: true,
		IDs:    ids,
	}
	queryArgs := []interface{}{
		dataType,
	}

	tableName := sqlite3impl.QuoteIdent(t.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(t.dbName)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get tag histories sql: %w", err)
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
		err = fmt.Errorf("error at select from TAG: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	tags := []Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := Tag{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return tags, nil
}

func (t *tagRepositoryCachedSQLite3Impl) AddTagInfo(ctx context.Context, tag Tag) error {
	t.m.Lock()
	defer t.m.Unlock()
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", t.addTagInfoSQL, queryArgs)
	_, err := t.addTagInfoStmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to TAG %s: %w", tag.ID, err)
		return err
	}
	return nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetAllTagNames(ctx context.Context) ([]string, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", t.getAllTagNamesSQL)
	rows, err := t.getAllTagNamesStmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at select all tag names from TAG: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	tagNames := []string{}
	for tagName := range tagNamesMap {
		tagNames = append(tagNames, tagName)
	}
	return tagNames, nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetAllTags(ctx context.Context) ([]Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(t.dbName) + `
WHERE
`

	dataType := "tag"

	query := &find.FindQuery{}
	queryArgs := []interface{}{
		dataType,
	}

	tableName := sqlite3impl.QuoteIdent(t.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(t.dbName)
	whereCounter := 0
	var onlyLatestData bool
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all tags sql: %w", err)
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
		err = fmt.Errorf("error at select from TAG: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	tags := []Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := Tag{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return tags, nil
}

func (t *tagRepositoryCachedSQLite3Impl) UnWrapTyped() ([]TagRepository, error) {
	return []TagRepository{t.tagRep}, nil
}

func (t *tagRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	repName, err := t.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	sql := `
SELECT IS_DELETED, ID AS TARGET_ID, TARGET_ID AS TARGET_ID_IN_DATA,
       ? AS LATEST_DATA_REPOSITORY_NAME, UPDATE_TIME_UNIX AS DATA_UPDATE_TIME_UNIX
FROM ` + sqlite3impl.QuoteIdent(t.dbName) + ` AS T
WHERE T.UPDATE_TIME_UNIX = (SELECT MAX(UPDATE_TIME_UNIX) FROM ` + sqlite3impl.QuoteIdent(t.dbName) + ` AS INNER_TABLE WHERE INNER_TABLE.ID = T.ID)
`
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
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
		addr.TargetIDInData = targetIDInData
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, addr)
	}
	return latestDataRepositoryAddresses, nil
}
