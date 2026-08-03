package reps

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

const CURRENT_SCHEMA_VERSION_TAG_REPOISITORY_SQLITE3IMPL_DAO = "1.0.0"

type tagRepositorySQLite3Impl struct {
	// キャッシュrepのフルリビルドを、実DBファイルが変わったときだけに絞るための判定用。
	// temp repが構造体変換でこの型をコピーするため、必ずポインタで持つこと。
	cacheChange *dbFileChangeDetector

	filename    string
	db          *sql.DB
	m           *sync.RWMutex
	fullConnect bool
}

func NewTagRepositorySQLite3Impl(ctx context.Context, filename string, fullConnect bool) (TagRepository, error) {
	db, err := sqlite3impl.GetSQLiteDBConnection(ctx, filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	if isOld, oldVerDAO, err := checkAndResolveDataSchemaTagRepositorySQLite3Impl(ctx, db); err != nil {
		return nil, err
	} else if isOld {
		if oldVerDAO != nil {
			return oldVerDAO, nil
		} else {
			err = fmt.Errorf("error at load database schema %s", filename)
			return nil, err
		}
	}

	if gkill_options.Optimize {
		err = sqlite3impl.DeleteAllIndex(db)
		if err != nil {
			err = fmt.Errorf("error at delete all index %w", err)
			return nil, err
		}
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TAG table statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_TAG ON TAG (ID, RELATED_TIME, UPDATE_TIME);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG index statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := indexStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG index to %s: %w", filename, err)
		return nil, err
	}

	indexTargetIDSQL := `CREATE INDEX IF NOT EXISTS INDEX_TAG_TARGET_ID ON TAG (TARGET_ID, UPDATE_TIME DESC);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", indexTargetIDSQL))
	indexTargetIDStmt, err := db.PrepareContext(ctx, indexTargetIDSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG_TARGET_ID index statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := indexTargetIDStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", indexTargetIDSQL))
	_, err = indexTargetIDStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_TARGET_ID index to %s: %w", filename, err)
		return nil, err
	}

	indexIDUpdateTimeSQL := `CREATE INDEX IF NOT EXISTS INDEX_TAG_ID_UPDATE_TIME ON TAG (ID, UPDATE_TIME);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", indexIDUpdateTimeSQL))
	indexIDUpdateTimeStmt, err := db.PrepareContext(ctx, indexIDUpdateTimeSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME index statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := indexIDUpdateTimeStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", indexIDUpdateTimeSQL))
	_, err = indexIDUpdateTimeStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME index to %s: %w", filename, err)
		return nil, err
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	_, err = indexIDUpdateTimeStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME table to %s: %w", filename, err)
		return nil, err
	}

	if gkill_options.Optimize {
		err = sqlite3impl.Optimize(db)
		if err != nil {
			err = fmt.Errorf("error at optimize db %w", err)
			return nil, err
		}
	}

	if !fullConnect {
		err = db.Close()
		if err != nil {
			return nil, err
		}
		db = nil
	}

	return &tagRepositorySQLite3Impl{
		cacheChange: &dbFileChangeDetector{},
		filename:    filename,
		db:          db,
		m:           &sync.RWMutex{},
		fullConnect: fullConnect,
	}, nil
}
func (t *tagRepositorySQLite3Impl) FindTags(ctx context.Context, query *find.FindQuery) ([]Tag, error) {
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
	}

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
	queryArgs := []any{
		repName,
		dataType,
	}

	tableName := "TAG"
	tableNameAlias := "TAG"
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "UPDATE_TIME"
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) Close(ctx context.Context) error {
	t.m.Lock()
	defer t.m.Unlock()
	if t.fullConnect {
		return t.db.Close()
	}
	return nil
}

func (t *tagRepositorySQLite3Impl) GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
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

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs:         true,
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UseUpdateTime:  updateTime != nil,
		UpdateTime:     updateTime,
	}
	queryArgs := []any{
		repName,
		dataType,
	}

	tableName := "TAG"
	tableNameAlias := "TAG"
	whereCounter := 0
	// GenerateFindSQLCommonはquery.OnlyLatestDataを見ないので、ローカル変数側でも指定する。
	// UpdateTime未指定なら編集前のバージョンを返さないよう最新版のみを対象にする
	onlyLatestData := updateTime == nil
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(tags) == 0 {
		return nil, nil
	}
	// ORDER BYを付けていないので、UpdateTimeが最新の行を明示的に選ぶ
	latestTag := slices.MaxFunc(tags, func(a, b Tag) int {
		return a.UpdateTime.Compare(b.UpdateTime)
	})
	return &latestTag, nil
}

func (t *tagRepositorySQLite3Impl) GetTagsByTagName(ctx context.Context, tagname string) ([]Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
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

	words := []string{tagname}

	query := &find.FindQuery{
		UseWords: true,
		Words:    words,
	}
	queryArgs := []any{
		repName,
		dataType,
	}

	tableName := "TAG"
	tableNameAlias := "TAG"
	whereCounter := 0
	// 編集前のタグ名でヒットしないよう、IDごとの最新版のみを対象にする
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) GetTagsByTargetID(ctx context.Context, target_id string) ([]Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
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

	targetIDs := []string{target_id}
	query := &find.FindQuery{
		UseWords: true,
		Words:    targetIDs,
	}
	queryArgs := []any{
		repName,
		dataType,
	}

	tableName := "TAG"
	tableNameAlias := "TAG"
	whereCounter := 0
	// キャッシュ版(NOT EXISTSアンチジョイン)と揃えて、IDごとの最新版のみを対象にする
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get target id sql %s: %w", target_id, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	// 自身は実DBを直接見るのでキャッシュは持たないが、
	// 上位のキャッシュrepが「作り直す必要があるか」を判断できるよう、
	// ここでファイルの更新有無だけ観測しておく。
	t.cacheChange.refresh(t.filename)
	return nil
}

func (t *tagRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if id == "" {
		return t.filename, nil
	}
	return filepath.Abs(t.filename)
}

func (t *tagRepositorySQLite3Impl) LastUpdateCacheChanged() bool {
	return t.cacheChange.lastChanged()
}

// CommitCacheRebuild は上位のキャッシュrepが再構築に成功したときに呼ばれます。
func (t *tagRepositorySQLite3Impl) CommitCacheRebuild() {
	t.cacheChange.commit()
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

func (t *tagRepositorySQLite3Impl) GetTagHistories(ctx context.Context, id string) ([]Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
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

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: true,
		IDs:    ids,
	}
	queryArgs := []any{
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) AddTagInfo(ctx context.Context, tag Tag) error {
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
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
	}
	if strings.TrimSpace(tag.Tag) == "" {
		return fmt.Errorf("tag must not be empty")
	}
	if strings.TrimSpace(tag.TargetID) == "" {
		return fmt.Errorf("tag target_id must not be empty")
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add tag sql %s: %w", tag.ID, err)
		return err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []any{
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to TAG %s: %w", tag.ID, err)
		return err
	}
	return nil
}

func (t *tagRepositorySQLite3Impl) GetAllTagNames(ctx context.Context) ([]string, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
	}

	// 編集前のタグ名・削除済みのタグ名を返さないよう、IDごとの最新版かつ未削除の行のみを見る。
	// このrep単体でのスコープ判定なので、最新版が別repにあるケースは集約側(TagRepositories)が解決する
	sql := `
SELECT
  DISTINCT TAG
FROM TAG
WHERE IS_DELETED = 0
  AND UPDATE_TIME = ( SELECT UPDATE_TIME FROM TAG AS INNER_TABLE WHERE INNER_TABLE.ID = TAG.ID ORDER BY datetime(INNER_TABLE.UPDATE_TIME) DESC LIMIT 1 )

`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all tag names sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	rows, err := stmt.QueryContext(ctx)
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return tagNames, nil
}

func (t *tagRepositorySQLite3Impl) GetAllTags(ctx context.Context) ([]Tag, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
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
	queryArgs := []any{
		repName,
		dataType,
	}

	tableName := "TAG"
	tableNameAlias := "TAG"
	whereCounter := 0
	// 編集前のタグ名を混ぜないよう、IDごとの最新版のみを対象にする。
	// IS_DELETEDはrep跨ぎで最新版を決めたあとに判定するのでここでは落とさない
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) UnWrapTyped() ([]TagRepository, error) {
	return []TagRepository{t}, nil
}

func (t *tagRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
	}

	if updateCache {
		err = t.UpdateCache(ctx)
		if err != nil {
			repName, _ := t.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	t.m.RLock()
	defer t.m.RUnlock()

	repName, err := t.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	sql := `
SELECT IS_DELETED, ID AS TARGET_ID, TARGET_ID AS TARGET_ID_IN_DATA,
       ? AS LATEST_DATA_REPOSITORY_NAME, UPDATE_TIME AS DATA_UPDATE_TIME
FROM TAG
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, repName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	latestDataRepositoryAddressMap := map[string]gkill_cache.LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			addr := gkill_cache.LatestDataRepositoryAddress{}
			var isDeletedStr string
			var dataUpdateTimeStr string
			var targetIDInData *string
			err := rows.Scan(&isDeletedStr, &addr.TargetID, &targetIDInData, &addr.LatestDataRepositoryName, &dataUpdateTimeStr)
			if err != nil {
				return nil, err
			}
			addr.IsDeleted = isDeletedStr == "TRUE"
			addr.TargetIDInData = targetIDInData
			addr.DataUpdateTime, err = time.Parse(sqlite3impl.TimeLayout, dataUpdateTimeStr)
			if err != nil {
				return nil, err
			}
			if existing, exist := latestDataRepositoryAddressMap[addr.TargetID]; !exist || existing.DataUpdateTime.Before(addr.DataUpdateTime) {
				latestDataRepositoryAddressMap[addr.TargetID] = addr
			}
		}
	}
	latestDataRepositoryAddresses := make([]gkill_cache.LatestDataRepositoryAddress, 0, len(latestDataRepositoryAddressMap))
	for _, addr := range latestDataRepositoryAddressMap {
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, addr)
	}
	return latestDataRepositoryAddresses, nil
}

func checkAndResolveDataSchemaTagRepositorySQLite3Impl(ctx context.Context, db *sql.DB) (isOld bool, oldVerDAO TagRepository, err error) {
	schemaVersionKey := "SCHEMA_VERSION_TAG"
	currentSchemaVersion := CURRENT_SCHEMA_VERSION_TAG_REPOISITORY_SQLITE3IMPL_DAO

	// テーブルとインデックスがなければ作る
	createTableSQL := `
CREATE TABLE IF NOT EXISTS GKILL_META_INFO (
  KEY NOT NULL,
  VALUE,
  PRIMARY KEY(KEY)
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", createTableSQL))
	stmt, err := db.PrepareContext(ctx, createTableSQL)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info table statement: %w", err)
		return false, nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", createTableSQL))
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info table: %w", err)
		return false, nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_GKILL_META_INFO ON GKILL_META_INFO (KEY);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info index statement: %w", err)
		return false, nil, err
	}
	defer func() {
		err := indexStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info index: %w", err)
		return false, nil, err
	}

	// スキーマのージョンを取得する
	selectSchemaVersionSQL := `
SELECT 
  VALUE
FROM GKILL_META_INFO
WHERE KEY = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", selectSchemaVersionSQL))
	selectSchemaVersionStmt, err := db.PrepareContext(ctx, selectSchemaVersionSQL)
	if err != nil {
		err = fmt.Errorf("error at get schema version sql: %w", err)
		return false, nil, err
	}
	defer func() {
		err := selectSchemaVersionStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	dbSchemaVersion := ""
	queryArgs := []any{schemaVersionKey}
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", selectSchemaVersionSQL), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	err = selectSchemaVersionStmt.QueryRowContext(ctx, queryArgs...).Scan(&dbSchemaVersion)
	if err != nil {
		// データがなかったら今のバージョンをいれる
		if errors.Is(err, sql.ErrNoRows) {
			insertCurrentVersionSQL := `
INSERT INTO GKILL_META_INFO(KEY, VALUE)
VALUES(?, ?)`
			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", insertCurrentVersionSQL))
			insertCurrentVersionStmt, err := db.PrepareContext(ctx, insertCurrentVersionSQL)
			if err != nil {
				err = fmt.Errorf("error at get schema version sql: %w", err)
				err = fmt.Errorf("error at insert schema version sql: %w", err)
				return false, nil, err
			}
			defer func() {
				err := insertCurrentVersionStmt.Close()
				if err != nil {
					slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
				}
			}()
			queryArgs := []any{schemaVersionKey, currentSchemaVersion}
			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", insertCurrentVersionSQL), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
			_, err = insertCurrentVersionStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at get schema version sql: %w", err)
				err = fmt.Errorf("error at query :%w", err)
				return false, nil, err
			}

			queryArgs = []any{schemaVersionKey}
			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", selectSchemaVersionSQL), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
			err = selectSchemaVersionStmt.QueryRowContext(ctx, queryArgs...).Scan(&dbSchemaVersion)
			if err != nil {
				err = fmt.Errorf("error at get schema version sql: %w", err)
				return false, nil, err
			}
		} else {
			err = fmt.Errorf("error at query :%w", err)
			return false, nil, err
		}
	}

	// ここから 過去バージョンのスキーマだった場合の対応
	if currentSchemaVersion != dbSchemaVersion {
		switch dbSchemaVersion {
		case "1.0.0":
			// 過去のDAOを作って返す or 最新のDAOに変換して返す
		}
		err = fmt.Errorf("invalid db schema version %s", dbSchemaVersion)
		return true, nil, err
	}
	// ここまで 過去バージョンのスキーマだった場合の対応

	return false, nil, nil
}
