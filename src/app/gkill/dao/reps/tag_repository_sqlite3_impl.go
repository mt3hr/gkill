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

type tagRepositorySQLite3Impl struct {
	filename    string
	db          *sql.DB
	m           *sync.Mutex
	fullConnect bool
}

func NewTagRepositorySQLite3Impl(ctx context.Context, filename string, fullConnect bool) (TagRepository, error) {
	db, err := sqlite3impl.GetSQLiteDBConnection(ctx, filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "TAG" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
  TAG NOT NULL,
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
		err = fmt.Errorf("error at create TAG table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_TAG ON TAG (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG index to %s: %w", filename, err)
		return nil, err
	}

	indexTargetIDSQL := `CREATE INDEX IF NOT EXISTS INDEX_TAG_TARGET_ID ON TAG (TARGET_ID, UPDATE_TIME DESC);`
	gkill_log.TraceSQL.Printf("sql: %s", indexTargetIDSQL)
	indexTargetIDStmt, err := db.PrepareContext(ctx, indexTargetIDSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG_TARGET_ID index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexTargetIDStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexTargetIDSQL)
	_, err = indexTargetIDStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_TARGET_ID index to %s: %w", filename, err)
		return nil, err
	}

	indexIDUpdateTimeSQL := `CREATE INDEX IF NOT EXISTS INDEX_TAG_ID_UPDATE_TIME ON TAG (ID, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexIDUpdateTimeSQL)
	indexIDUpdateTimeStmt, err := db.PrepareContext(ctx, indexIDUpdateTimeSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexIDUpdateTimeStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexIDUpdateTimeSQL)
	_, err = indexIDUpdateTimeStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = indexIDUpdateTimeStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME table to %s: %w", filename, err)
		return nil, err
	}

	dbName := "TAG"
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

	return &tagRepositorySQLite3Impl{
		filename:    filename,
		db:          db,
		m:           &sync.Mutex{},
		fullConnect: fullConnect,
	}, nil
}
func (t *tagRepositorySQLite3Impl) FindTags(ctx context.Context, query *find.FindQuery) ([]*Tag, error) {
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
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
  TAG,
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
FROM TAG
WHERE
`
	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at tag: %w", err)
		return nil, err
	}
	dataType := "tag"
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "TAG"
	tableNameAlias := "TAG"
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "UPDATE_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
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
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeStr,
				&createTimeStr,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeStr,
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

			tag.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TAG: %w", relatedTimeStr, err)
				return nil, err
			}
			tag.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TAG: %w", createTimeStr, err)
				return nil, err
			}
			tag.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TAG: %w", updateTimeStr, err)
				return nil, err
			}
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) Close(ctx context.Context) error {
	if t.fullConnect {
		return t.db.Close()
	}
	return nil
}

func (t *tagRepositorySQLite3Impl) GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error) {
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

func (t *tagRepositorySQLite3Impl) GetTagsByTagName(ctx context.Context, tagname string) ([]*Tag, error) {
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TAG,
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
FROM TAG
WHERE 
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at tag: %w", err)
		return nil, err
	}
	dataType := "tag"

	trueValue := true
	words := []string{tagname}

	query := &find.FindQuery{
		UseWords: &trueValue,
		Words:    &words,
	}
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "TAG"
	tableNameAlias := "TAG"
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "UPDATE_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
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
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeStr,
				&createTimeStr,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeStr,
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

			tag.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TAG: %w", relatedTimeStr, err)
				return nil, err
			}
			tag.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TAG: %w", createTimeStr, err)
				return nil, err
			}
			tag.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TAG: %w", updateTimeStr, err)
				return nil, err
			}
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) GetTagsByTargetID(ctx context.Context, target_id string) ([]*Tag, error) {
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TAG,
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
FROM TAG
WHERE 
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at tag: %w", err)
		return nil, err
	}

	dataType := "tag"

	trueValue := true
	targetIDs := []string{target_id}
	query := &find.FindQuery{
		UseWords: &trueValue,
		Words:    &targetIDs,
	}
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "TAG"
	tableNameAlias := "TAG"
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "UPDATE_TIME"
	findWordTargetColumns := []string{"TARGET_ID"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := false
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get target id sql %s: %w", target_id, err)
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
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeStr,
				&createTimeStr,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeStr,
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

			tag.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TAG: %w", relatedTimeStr, err)
				return nil, err
			}
			tag.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TAG: %w", createTimeStr, err)
				return nil, err
			}
			tag.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TAG: %w", updateTimeStr, err)
				return nil, err
			}
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (t *tagRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if id == "" {
		return t.filename, nil
	}
	return filepath.Abs(t.filename)
}

func (t *tagRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := t.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path tag rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (t *tagRepositorySQLite3Impl) GetTagHistories(ctx context.Context, id string) ([]*Tag, error) {
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TAG,
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
FROM TAG
WHERE 
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at tag: %w", err)
		return nil, err
	}
	dataType := "tag"

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "TAG"
	tableNameAlias := "TAG"
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "UPDATE_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
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
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeStr,
				&createTimeStr,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeStr,
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

			tag.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TAG: %w", relatedTimeStr, err)
				return nil, err
			}
			tag.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TAG: %w", createTimeStr, err)
				return nil, err
			}
			tag.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TAG: %w", updateTimeStr, err)
				return nil, err
			}
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) AddTagInfo(ctx context.Context, tag *Tag) error {
	t.m.Lock()
	defer t.m.Unlock()
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return err
		}
		defer db.Close()
	}
	sql := `
INSERT INTO TAG (
  IS_DELETED,
  ID,
  TAG,
  TARGET_ID,
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
  ?
)`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
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
		tag.RelatedTime.Format(sqlite3impl.TimeLayout),
		tag.CreateTime.Format(sqlite3impl.TimeLayout),
		tag.CreateApp,
		tag.CreateDevice,
		tag.CreateUser,
		tag.UpdateTime.Format(sqlite3impl.TimeLayout),
		tag.UpdateApp,
		tag.UpdateDevice,
		tag.UpdateUser,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to TAG %s: %w", tag.ID, err)
		return err
	}
	return nil
}

func (t *tagRepositorySQLite3Impl) GetAllTagNames(ctx context.Context) ([]string, error) {
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	sql := `
SELECT 
  DISTINCT TAG
FROM TAG
WHERE IS_DELETED = FALSE

`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
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

	tagNames := []string{}
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

			tagNames = append(tagNames, tagName)
		}
	}
	return tagNames, nil
}

func (t *tagRepositorySQLite3Impl) GetAllTags(ctx context.Context) ([]*Tag, error) {
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TAG,
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
FROM TAG
WHERE 
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at tag: %w", err)
		return nil, err
	}
	dataType := "tag"

	query := &find.FindQuery{}
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "TAG"
	tableNameAlias := "TAG"
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "UPDATE_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
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
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeStr,
				&createTimeStr,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeStr,
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

			tag.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TAG: %w", relatedTimeStr, err)
				return nil, err
			}
			tag.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TAG: %w", createTimeStr, err)
				return nil, err
			}
			tag.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TAG: %w", updateTimeStr, err)
				return nil, err
			}
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) UnWrapTyped() ([]TagRepository, error) {
	return []TagRepository{t}, nil
}

func (t *tagRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error) {
	dbName := "TAG"
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	// update_cacheであればキャッシュを更新する
	if updateCache {
		err = t.UpdateCache(ctx)
		if err != nil {
			repName, _ := t.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

	sql := fmt.Sprintf(`
SELECT
  tbl.IS_DELETED,
  tbl.TARGET_ID AS TARGET_ID,
  tbl.TARGET_ID AS TARGET_ID_IN_DATA,
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

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at %s : %w", t.filename, err)
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
