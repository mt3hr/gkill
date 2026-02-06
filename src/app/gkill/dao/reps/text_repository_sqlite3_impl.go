package reps

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type textRepositorySQLite3Impl struct {
	filename    string
	db          *sql.DB
	m           *sync.Mutex
	fullConnect bool
}

func NewTextRepositorySQLite3Impl(ctx context.Context, filename string, fullConnect bool) (TextRepository, error) {
	db, err := sqlite3impl.GetSQLiteDBConnection(ctx, filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	if gkill_options.Optimize {
		err = sqlite3impl.DeleteAllIndex(db)
		if err != nil {
			err = fmt.Errorf("error at delete all index %w", err)
			return nil, err
		}
	}

	sql := `
CREATE TABLE IF NOT EXISTS "TEXT" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
  TEXT NOT NULL,
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TEXT table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_TEXT ON TEXT(ID, RELATED_TIME, UPDATE_TIME);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT index statement %s: %w", "TEXT", err)
		return nil, err
	}
	defer indexStmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT index to %s: %w", "TEXT", err)
		return nil, err
	}

	indexTargetIDSQL := `CREATE INDEX IF NOT EXISTS INDEX_TEXT_TARGET_ID ON TEXT(TARGET_ID, UPDATE_TIME DESC);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexTargetIDSQL)
	indexTargetIDStmt, err := db.PrepareContext(ctx, indexTargetIDSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_TARGET_ID index statement %s: %w", "TEXT", err)
		return nil, err
	}
	defer indexTargetIDStmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexTargetIDSQL)
	_, err = indexTargetIDStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_TARGET_ID index to %s: %w", "TEXT", err)
		return nil, err
	}

	indexIDUpdateTimeSQL := `CREATE INDEX IF NOT EXISTS INDEX_TEXT_ID_UPDATE_TIME ON ` + "TEXT" + `(ID, UPDATE_TIME);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexIDUpdateTimeSQL)
	indexIDUpdateTimeStmt, err := db.PrepareContext(ctx, indexIDUpdateTimeSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_ID_UPDATE_TIME index statement %s: %w", "TEXT", err)
		return nil, err
	}
	defer indexIDUpdateTimeStmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexIDUpdateTimeSQL)
	_, err = indexIDUpdateTimeStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_ID_UPDATE_TIME index to %s: %w", "TEXT", err)
		return nil, err
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = indexIDUpdateTimeStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT_ID_UPDATE_TIME table to %s: %w", "TEXT", err)
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

	return &textRepositorySQLite3Impl{
		filename:    filename,
		db:          db,
		m:           &sync.Mutex{},
		fullConnect: fullConnect,
	}, nil
}
func (t *textRepositorySQLite3Impl) FindTexts(ctx context.Context, query *find.FindQuery) ([]*Text, error) {
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
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM TEXT
WHERE 
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at text: %w", err)
		return nil, err
	}
	dataType := "text"
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "TEXT"
	tableNameAlias := "TEXT"
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "UPDATE_TIME"
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get TEXT histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
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

func (t *textRepositorySQLite3Impl) Close(ctx context.Context) error {
	if t.fullConnect {
		return t.db.Close()
	}
	return nil
}

func (t *textRepositorySQLite3Impl) GetText(ctx context.Context, id string, updateTime *time.Time) (*Text, error) {
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

func (t *textRepositorySQLite3Impl) GetTextsByTargetID(ctx context.Context, target_id string) ([]*Text, error) {
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
  TEXT,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM TEXT
WHERE 
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at text: %w", err)
		return nil, err
	}

	dataType := "text"

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

	tableName := "TEXT"
	tableNameAlias := "TEXT"
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "UPDATE_TIME"
	findWordTargetColumns := []string{"TARGET_ID"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := false
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get text histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
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

func (t *textRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (t *textRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if id == "" {
		return t.filename, nil
	}
	return filepath.Abs(t.filename)
}

func (t *textRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := t.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path text rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (t *textRepositorySQLite3Impl) GetTextHistories(ctx context.Context, id string) ([]*Text, error) {
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
  TEXT,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM TEXT
WHERE 
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at text: %w", err)
		return nil, err
	}
	dataType := "text"

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

	tableName := "TEXT"
	tableNameAlias := "TEXT"
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "UPDATE_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get text histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
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
func (t *textRepositorySQLite3Impl) AddTextInfo(ctx context.Context, text *Text) error {
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
INSERT INTO TEXT (
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
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
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to TEXT %s: %w", text.ID, err)
		return err
	}
	return nil
}

func (t *textRepositorySQLite3Impl) UnWrapTyped() ([]TextRepository, error) {
	return []TextRepository{t}, nil
}
