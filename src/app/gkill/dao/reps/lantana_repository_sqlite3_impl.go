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
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type lantanaRepositorySQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewLantanaRepositorySQLite3Impl(ctx context.Context, filename string) (LantanaRepository, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_auto_vacuum=1&_timeout=60000&_journal=WAL&_cache_size=-50000&_mutex=full&_sync=1&_txlock=deferred")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "LANTANA" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  MOOD NOT NULL,
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
		err = fmt.Errorf("error at create LANTANA table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create LANTANA table to %s: %w", filename, err)
		return nil, err
	}

	return &lantanaRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (l *lantanaRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error) {
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
FROM LANTANA
WHERE
`
	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
		return nil, err
	}

	dataType := "lantana"

	queryArgs := []interface{}{
		repName,
		dataType,
	}
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := true
	appendGroupBy := true
	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendGroupBy, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from LANTANA%s: %w", err)
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

			err = rows.Scan(&kyou.IsDeleted,
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

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in LANTANA: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in LANTANA: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in LANTANA: %w", updateTimeStr, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (l *lantanaRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
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
			if kyou.UpdateTime.Format(sqlite3impl.TimeLayout) == updateTime.Format(sqlite3impl.TimeLayout) {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return kyouHistories[0], nil
}

func (l *lantanaRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
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
FROM  LANTANA
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
		repName,
		dataType,
	}

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := true
	appendGroupBy := false
	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendGroupBy, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
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
			kyou.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(&kyou.IsDeleted,
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

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in LANTANA: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in LANTANA: %w", createTimeStr, id, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in LANTANA: %w", updateTimeStr, id, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (l *lantanaRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return filepath.Abs(l.filename)
}

func (l *lantanaRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (l *lantanaRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := l.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path lantana rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (l *lantanaRepositorySQLite3Impl) Close(ctx context.Context) error {
	return l.db.Close()
}

func (l *lantanaRepositorySQLite3Impl) FindLantana(ctx context.Context, query *find.FindQuery) ([]*Lantana, error) {
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
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  MOOD,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM LANTANA
WHERE
`

	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
		return nil, err
	}

	dataType := "lantana"

	queryArgs := []interface{}{
		repName,
		dataType,
	}

	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := true
	appendGroupBy := true
	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendGroupBy, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from LANTANA %s: %w", err)
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
			lantana.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(&lantana.IsDeleted,
				&lantana.ID,
				&relatedTimeStr,
				&createTimeStr,
				&lantana.CreateApp,
				&lantana.CreateDevice,
				&lantana.CreateUser,
				&updateTimeStr,
				&lantana.UpdateApp,
				&lantana.UpdateDevice,
				&lantana.UpdateUser,
				&lantana.Mood,
				&lantana.RepName,
				&lantana.DataType,
			)

			lantana.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in LANTANA: %w", relatedTimeStr, err)
				return nil, err
			}
			lantana.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in LANTANA: %w", createTimeStr, err)
				return nil, err
			}
			lantana.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in LANTANA: %w", updateTimeStr, err)
				return nil, err
			}
			lantanas = append(lantanas, lantana)
		}
	}
	return lantanas, nil
}

func (l *lantanaRepositorySQLite3Impl) GetLantana(ctx context.Context, id string, updateTime *time.Time) (*Lantana, error) {
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
			if kyou.UpdateTime.Format(sqlite3impl.TimeLayout) == updateTime.Format(sqlite3impl.TimeLayout) {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return lantanaHistories[0], nil
}

func (l *lantanaRepositorySQLite3Impl) GetLantanaHistories(ctx context.Context, id string) ([]*Lantana, error) {
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
  MOOD,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM LANTANA
WHERE
`

	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
		return nil, err
	}

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	dataType := "lantana"

	queryArgs := []interface{}{
		repName,
		dataType,
	}

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := true
	appendGroupBy := false
	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendGroupBy, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
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
			lantana.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(&lantana.IsDeleted,
				&lantana.ID,
				&relatedTimeStr,
				&createTimeStr,
				&lantana.CreateApp,
				&lantana.CreateDevice,
				&lantana.CreateUser,
				&updateTimeStr,
				&lantana.UpdateApp,
				&lantana.UpdateDevice,
				&lantana.UpdateUser,
				&lantana.Mood,
				&lantana.RepName,
				&lantana.DataType,
			)

			lantana.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in LANTANA: %w", relatedTimeStr, id, err)
				return nil, err
			}
			lantana.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in LANTANA: %w", createTimeStr, id, err)
				return nil, err
			}
			lantana.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in LANTANA: %w", updateTimeStr, id, err)
				return nil, err
			}
			lantanas = append(lantanas, lantana)
		}
	}
	return lantanas, nil
}

func (l *lantanaRepositorySQLite3Impl) AddLantanaInfo(ctx context.Context, lantana *Lantana) error {
	sql := `
INSERT INTO LANTANA (
  IS_DELETED,
  ID,
  MOOD,
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
  ?
)`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add lantana sql %s: %w", lantana.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		lantana.IsDeleted,
		lantana.ID,
		lantana.Mood,
		lantana.RelatedTime.Format(sqlite3impl.TimeLayout),
		lantana.CreateTime.Format(sqlite3impl.TimeLayout),
		lantana.CreateApp,
		lantana.CreateDevice,
		lantana.CreateUser,
		lantana.UpdateTime.Format(sqlite3impl.TimeLayout),
		lantana.UpdateApp,
		lantana.UpdateDevice,
		lantana.UpdateUser,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to LANTANA %s: %w", lantana.ID, err)
		return err
	}
	return nil
}
