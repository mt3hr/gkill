package reps

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
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
  RELATED_TIME NOT NULL,
  CREATE_TIME NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_DEVICE NOT NULL,
  CREATE_USER NOT NULL,
  UPDATE_TIME NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  REP_NAME NOT NULL
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

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT table to %s: %w", dbName, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_` + dbName + ` ON ` + dbName + `(ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := cacheDB.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT index statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT index to %s: %w", dbName, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT index statement %s: %w", dbName, err)
		return nil, err
	}

	indexTargetIDSQL := `CREATE INDEX IF NOT EXISTS INDEX_` + dbName + `_TARGET_ID ON ` + dbName + `(TARGET_ID, UPDATE_TIME DESC);`
	gkill_log.TraceSQL.Printf("sql: %s", indexTargetIDSQL)
	indexTargetIDStmt, err := cacheDB.PrepareContext(ctx, indexTargetIDSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_TARGET_ID index statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexTargetIDStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexTargetIDSQL)
	_, err = indexTargetIDStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_TARGET_ID index to %s: %w", dbName, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", indexTargetIDSQL)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_ID_UPDATE_TIME index statement %s: %w", dbName, err)
		return nil, err
	}

	indexIDUpdateTimeSQL := `CREATE INDEX IF NOT EXISTS INDEX_` + dbName + `_ID_UPDATE_TIME ON ` + dbName + `(ID, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexIDUpdateTimeSQL)
	indexIDUpdateTimeStmt, err := cacheDB.PrepareContext(ctx, indexIDUpdateTimeSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_ID_UPDATE_TIME index statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexIDUpdateTimeStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexIDUpdateTimeSQL)
	_, err = indexIDUpdateTimeStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_ID_UPDATE_TIME index to %s: %w", dbName, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = indexIDUpdateTimeStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_ID_UPDATE_TIME table to %s: %w", dbName, err)
		return nil, err
	}

	getTextsByTargetIDSQL := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TEXT,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
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
      AND TEXT2.UPDATE_TIME > TEXT1.UPDATE_TIME
  )
ORDER BY TEXT1.UPDATE_TIME DESC
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
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
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

	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "UPDATE_TIME"
	findWordTargetColumns := []string{"TEXT"}
	ignoreFindWord := false
	appendOrderBy := true

	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
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
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&text.IsDeleted,
				&text.ID,
				&text.TargetID,
				&text.Text,
				&relatedTimeStr,
				&createTimeStr,
				&text.CreateApp,
				&text.CreateDevice,
				&text.CreateUser,
				&updateTimeStr,
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

			text.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TEXT : %w", relatedTimeStr, err)
				return nil, err
			}
			text.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TEXT: %w", createTimeStr, err)
				return nil, err
			}
			text.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TEXT: %w", updateTimeStr, err)
				return nil, err
			}
			texts = append(texts, text)
		}
	}
	return texts, nil
}

func (t *textRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	t.getTextsByTargetIDStmt.Close()
	_, err := t.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+t.dbName)
	return err
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
			if kyou.UpdateTime.Format(sqlite3impl.TimeLayout) == updateTime.Format(sqlite3impl.TimeLayout) {
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
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&text.IsDeleted,
				&text.ID,
				&text.TargetID,
				&text.Text,
				&relatedTimeStr,
				&createTimeStr,
				&text.CreateApp,
				&text.CreateDevice,
				&text.CreateUser,
				&updateTimeStr,
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

			text.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TEXT: %w", relatedTimeStr, err)
				return nil, err
			}
			text.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TEXT: %w", createTimeStr, err)
				return nil, err
			}
			text.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TEXT: %w", updateTimeStr, err)
				return nil, err
			}
			texts = append(texts, text)
		}
	}
	return texts, nil
}

func (t *textRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	// t.m.Lock()
	// defer t.m.Unlock()

	trueValue := true
	query := &find.FindQuery{
		UpdateCache: &trueValue,
	}
	allTexts, err := t.textRep.FindTexts(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all texts at update cache: %w", err)
		return err
	}

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
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME
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
				text.RelatedTime.Format(sqlite3impl.TimeLayout),
				text.CreateTime.Format(sqlite3impl.TimeLayout),
				text.CreateApp,
				text.CreateDevice,
				text.CreateUser,
				text.UpdateTime.Format(sqlite3impl.TimeLayout),
				text.UpdateApp,
				text.UpdateDevice,
				text.UpdateUser,
				text.RepName,
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
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TEXT,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
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

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "UPDATE_TIME"
	findWordTargetColumns := []string{"TEXT"}
	ignoreFindWord := false
	appendOrderBy := false

	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	commonWhereSQL += " ORDER BY datetime(UPDATE_TIME, 'localtime') DESC "
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
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&text.IsDeleted,
				&text.ID,
				&text.TargetID,
				&text.Text,
				&relatedTimeStr,
				&createTimeStr,
				&text.CreateApp,
				&text.CreateDevice,
				&text.CreateUser,
				&updateTimeStr,
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

			text.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TEXT: %w", relatedTimeStr, err)
				return nil, err
			}
			text.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TEXT: %w", createTimeStr, err)
				return nil, err
			}
			text.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TEXT: %w", updateTimeStr, err)
				return nil, err
			}
			texts = append(texts, text)
		}
	}
	return texts, nil
}
func (t *textRepositoryCachedSQLite3Impl) AddTextInfo(ctx context.Context, text *Text) error {
	sql := `
INSERT INTO ` + t.dbName + ` (
  IS_DELETED,
  ID,
  TEXT,
  TARGET_ID,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME
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
		text.RelatedTime.Format(sqlite3impl.TimeLayout),
		text.CreateTime.Format(sqlite3impl.TimeLayout),
		text.CreateApp,
		text.CreateDevice,
		text.CreateUser,
		text.UpdateTime.Format(sqlite3impl.TimeLayout),
		text.UpdateApp,
		text.UpdateDevice,
		text.UpdateUser,
		text.RepName,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to TEXT %s: %w", text.ID, err)
		return err
	}
	return nil
}
