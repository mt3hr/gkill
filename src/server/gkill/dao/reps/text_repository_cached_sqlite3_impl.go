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

type textRepositoryCachedSQLite3Impl struct {
	dbName                 string
	textRep                TextRepository
	cachedDB               *sqllib.DB
	getTextsByTargetIDSQL  string
	getTextsByTargetIDStmt *sqllib.Stmt
	addTextInfoSQL         string
	addTextInfoStmt        *sqllib.Stmt
	m                      *sync.RWMutex
}

func NewTextRepositoryCachedSQLite3Impl(ctx context.Context, textRep TextRepository, cacheDB *sqllib.DB, m *sync.RWMutex, dbName string) (TextRepository, error) {
	if m == nil {
		m = &sync.RWMutex{}
	}
	var err error

	sql := `
CREATE TABLE IF NOT EXISTS ` + sqlite3impl.QuoteIdent(dbName) + ` (
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TEXT table statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create TEXT table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName) + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT index statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create TEXT index to %s: %w", dbName, err)
		return nil, err
	}

	indexTargetIDUnixSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_TARGET_ID") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(TARGET_ID, UPDATE_TIME_UNIX DESC);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexTargetIDUnixSQL)
	indexTargetIDUnixStmt, err := cacheDB.PrepareContext(ctx, indexTargetIDUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_TARGET_ID index statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create TEXT_TARGET_ID index to %s: %w", dbName, err)
		return nil, err
	}

	indexIDUpdateTimeUnixSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_ID_UPDATE_TIME_UNIX") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(ID, UPDATE_TIME_UNIX);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexIDUpdateTimeUnixSQL)
	indexIDUpdateTimeUnixStmt, err := cacheDB.PrepareContext(ctx, indexIDUpdateTimeUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_ID_UPDATE_TIME_UNIX index statement %s: %w", dbName, err)
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
FROM ` + sqlite3impl.QuoteIdent(dbName) + ` AS TEXT1
WHERE TEXT1.TARGET_ID = ?
  AND NOT EXISTS (
    SELECT 1
    FROM ` + sqlite3impl.QuoteIdent(dbName) + ` AS TEXT2
    WHERE TEXT2.ID = TEXT1.ID
      AND TEXT2.UPDATE_TIME_UNIX > TEXT1.UPDATE_TIME_UNIX
  )
ORDER BY TEXT1.UPDATE_TIME_UNIX DESC
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", getTextsByTargetIDSQL)
	getTextsByTargetIDStmt, err := cacheDB.PrepareContext(ctx, getTextsByTargetIDSQL)
	if err != nil {
		err = fmt.Errorf("error at get target id sql: %w", err)
		return nil, err
	}

	addTextInfoSQL := `
INSERT INTO ` + sqlite3impl.QuoteIdent(dbName) + ` (
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", addTextInfoSQL)
	addTextInfoStmt, err := cacheDB.PrepareContext(ctx, addTextInfoSQL)
	if err != nil {
		err = fmt.Errorf("error at add text info sql: %w", err)
		return nil, err
	}

	return &textRepositoryCachedSQLite3Impl{
		dbName:                 dbName,
		textRep:                textRep,
		cachedDB:               cacheDB,
		getTextsByTargetIDSQL:  getTextsByTargetIDSQL,
		getTextsByTargetIDStmt: getTextsByTargetIDStmt,
		addTextInfoSQL:         addTextInfoSQL,
		addTextInfoStmt:        addTextInfoStmt,
		m:                      m,
	}, nil
}
func (t *textRepositoryCachedSQLite3Impl) FindTexts(ctx context.Context, query *find.FindQuery) ([]Text, error) {
	var err error

	if query.UseWords {
		if len(query.Words) == 0 {
			return []Text{}, nil
		}
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
FROM ` + sqlite3impl.QuoteIdent(t.dbName) + `
WHERE
`

	dataType := "text"
	queryArgs := []any{
		dataType,
	}

	tableName := sqlite3impl.QuoteIdent(t.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(t.dbName)
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TEXT"}
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
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get TEXT histories sql: %w", err)
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
		err = fmt.Errorf("error at select from TEXT: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	texts := []Text{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			text := Text{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return texts, nil
}

func (t *textRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	t.m.Lock()
	defer t.m.Unlock()
	if t.addTextInfoStmt != nil {
		t.addTextInfoStmt.Close()
	}
	if t.getTextsByTargetIDStmt != nil {
		t.getTextsByTargetIDStmt.Close()
	}
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
		_, err = t.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+sqlite3impl.QuoteIdent(t.dbName))
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *textRepositoryCachedSQLite3Impl) GetText(ctx context.Context, id string, updateTime *time.Time) (*Text, error) {
	t.m.RLock()
	defer t.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(t.dbName) + `
WHERE
`

	dataType := "text"

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs:         true,
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UseUpdateTime:  updateTime != nil,
		UpdateTime:     updateTime,
	}
	queryArgs := []any{
		dataType,
	}

	tableName := sqlite3impl.QuoteIdent(t.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(t.dbName)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get text histories sql: %w", err)
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
		err = fmt.Errorf("error at select from TEXT: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	texts := []Text{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			text := Text{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(texts) == 0 {
		return nil, nil
	}
	return &texts[0], nil
}

func (t *textRepositoryCachedSQLite3Impl) GetTextsByTargetID(ctx context.Context, target_id string) ([]Text, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error
	dataType := "text"

	queryArgs := []any{
		dataType,
		target_id,
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", t.getTextsByTargetIDSQL, queryArgs)
	rows, err := t.getTextsByTargetIDStmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from TEXT: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	texts := []Text{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			text := Text{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return texts, nil
}

func (t *textRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	err := t.textRep.UpdateCache(ctx)
	if err != nil {
		return fmt.Errorf("error at update underlying text rep cache: %w", err)
	}

	// 下層リポジトリに変更がなければフルリビルドをスキップ
	if !t.textRep.LastUpdateCacheChanged() {
		return nil
	}

	query := &find.FindQuery{
		UpdateCache:        false,
		OnlyLatestData:     false,
		IncludeDeletedData: true,
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
		err = fmt.Errorf("error at create TEXT table statement %s: %w", "memory", err)
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
		err = fmt.Errorf("error at delete TEXT table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + sqlite3impl.QuoteIdent(t.dbName) + ` (
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add text sql: %w", err)
		return err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for _, text := range allTexts {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			queryArgs := []any{
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
			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to TEXT %s: %w", text.ID, err)
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
		err = fmt.Errorf("error at commit transaction for add texts: %w", err)
		return err
	}
	isCommitted = true
	return nil
}

func (t *textRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return t.textRep.GetPath(ctx, id)
}

func (t *textRepositoryCachedSQLite3Impl) LastUpdateCacheChanged() bool {
	return true
}

func (t *textRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return t.textRep.GetRepName(ctx)
}

func (t *textRepositoryCachedSQLite3Impl) GetTextHistories(ctx context.Context, id string) ([]Text, error) {
	t.m.RLock()
	defer t.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(t.dbName) + `
WHERE
`

	dataType := "text"

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: true,
		IDs:    ids,
	}
	queryArgs := []any{
		dataType,
	}

	tableName := sqlite3impl.QuoteIdent(t.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(t.dbName)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get text histories sql: %w", err)
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
		err = fmt.Errorf("error at select from TEXT: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	texts := []Text{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			text := Text{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return texts, nil
}
func (t *textRepositoryCachedSQLite3Impl) AddTextInfo(ctx context.Context, text Text) error {
	t.m.Lock()
	defer t.m.Unlock()
	queryArgs := []any{
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", t.addTextInfoSQL, queryArgs)
	_, err := t.addTextInfoStmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to TEXT %s: %w", text.ID, err)
		return err
	}
	return nil
}

func (m *textRepositoryCachedSQLite3Impl) UnWrapTyped() ([]TextRepository, error) {
	return []TextRepository{m.textRep}, nil
}

func (t *textRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
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
