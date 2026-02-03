package reps

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type kcRepositorySQLite3Impl struct {
	filename    string
	db          *sql.DB
	m           *sync.Mutex
	fullConnect bool
}

func NewKCRepositorySQLite3Impl(ctx context.Context, filename string, fullConnect bool) (KCRepository, error) {
	db, err := sqlite3impl.GetSQLiteDBConnection(ctx, filename)
	if err != nil {
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "kc" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TITLE NOT NULL,
  NUM_VALUE NOT NULL,
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
		err = fmt.Errorf("error at create kc table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create kc table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_kc ON kc (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create kc index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create kc index to %s: %w", filename, err)
		return nil, err
	}

	dbName := "kc"
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

	return &kcRepositorySQLite3Impl{
		filename:    filename,
		db:          db,
		m:           &sync.Mutex{},
		fullConnect: fullConnect,
	}, nil
}

func (k *kcRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	var err error
	var db *sql.DB
	if k.fullConnect {
		db = k.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, k.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = k.UpdateCache(ctx)
		if err != nil {
			repName, _ := k.GetRepName(ctx)
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
FROM kc
WHERE
`

	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at kc: %w", err)
		return nil, err
	}

	dataType := "kc"
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "KC"
	tableNameAlias := "KC"
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
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
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from kc: %w", err)
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
			if err != nil {
				err = fmt.Errorf("error at scan kc: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in kc: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in kc: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in kc: %w", updateTimeStr, err)
				return nil, err
			}

			if _, exist := kyous[kyou.ID]; !exist {
				kyous[kyou.ID] = []*Kyou{}
			}
			kyous[kyou.ID] = append(kyous[kyou.ID], kyou)
		}
	}
	return kyous, nil
}

func (k *kcRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := k.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from kc %s: %w", id, err)
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

func (k *kcRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	var err error
	var db *sql.DB
	if k.fullConnect {
		db = k.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, k.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at kc: %w", err)
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
FROM kc
WHERE 
`
	dataType := "kc"

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

	tableName := "KC"
	tableNameAlias := "KC"
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
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
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from kc %s: %w", id, err)
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
			if err != nil {
				err = fmt.Errorf("error at scan kc %s: %w", id, err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in kc: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in kc: %w", createTimeStr, id, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in kc: %w", updateTimeStr, id, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (k *kcRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if id == "" {
		return k.filename, nil
	}
	return filepath.Abs(k.filename)
}

func (k *kcRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (k *kcRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := k.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path kc rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (k *kcRepositorySQLite3Impl) Close(ctx context.Context) error {
	if k.fullConnect {
		return k.db.Close()
	}
	return nil
}

func (k *kcRepositorySQLite3Impl) FindKC(ctx context.Context, query *find.FindQuery) ([]*KC, error) {
	var err error
	var db *sql.DB
	if k.fullConnect {
		db = k.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, k.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = k.UpdateCache(ctx)
		if err != nil {
			repName, _ := k.GetRepName(ctx)
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
  TITLE,
  NUM_VALUE,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM kc
WHERE
`

	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at kc: %w", err)
		return nil, err
	}
	dataType := "kc"

	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "KC"
	tableNameAlias := "KC"
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
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
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from kc: %w", err)
		return nil, err
	}
	defer rows.Close()

	kcs := []*KC{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kc := &KC{}
			kc.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			numValueStr := ""

			err = rows.Scan(&kc.IsDeleted,
				&kc.ID,
				&relatedTimeStr,
				&createTimeStr,
				&kc.CreateApp,
				&kc.CreateDevice,
				&kc.CreateUser,
				&updateTimeStr,
				&kc.UpdateApp,
				&kc.UpdateDevice,
				&kc.UpdateUser,
				&kc.Title,
				&numValueStr,
				&kc.RepName,
				&kc.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan kc: %w", err)
				return nil, err
			}
			numValue := strings.ReplaceAll(numValueStr, ",", "")
			kc.NumValue = json.Number(numValue)

			kc.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in kc: %w", relatedTimeStr, err)
				return nil, err
			}
			kc.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in kc: %w", createTimeStr, err)
				return nil, err
			}
			kc.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in kc: %w", updateTimeStr, err)
				return nil, err
			}
			kcs = append(kcs, kc)
		}
	}
	return kcs, nil
}

func (k *kcRepositorySQLite3Impl) GetKC(ctx context.Context, id string, updateTime *time.Time) (*KC, error) {
	// 最新のデータを返す
	kcHistories, err := k.GetKCHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kc histories from kc %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(kcHistories) == 0 {
		return nil, nil
	}

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range kcHistories {
			if kyou.UpdateTime.Unix() == updateTime.Unix() {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return kcHistories[0], nil
}

func (k *kcRepositorySQLite3Impl) GetKCHistories(ctx context.Context, id string) ([]*KC, error) {
	var err error
	var db *sql.DB
	if k.fullConnect {
		db = k.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, k.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at kc: %w", err)
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
  TITLE,
  NUM_VALUE,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM kc
WHERE 
`
	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}
	dataType := "kc"

	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "KC"
	tableNameAlias := "KC"
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
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
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kc histories sql %s: %w", id, err)
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

	kcs := []*KC{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kc := &KC{}
			kc.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			numValueStr := ""

			err = rows.Scan(&kc.IsDeleted,
				&kc.ID,
				&relatedTimeStr,
				&createTimeStr,
				&kc.CreateApp,
				&kc.CreateDevice,
				&kc.CreateUser,
				&updateTimeStr,
				&kc.UpdateApp,
				&kc.UpdateDevice,
				&kc.UpdateUser,
				&kc.Title,
				&numValueStr,
				&kc.RepName,
				&kc.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan kc %s: %w", id, err)
				return nil, err
			}
			numValue := strings.ReplaceAll(numValueStr, ",", "")
			kc.NumValue = json.Number(numValue)

			kc.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in kc: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kc.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in kc: %w", createTimeStr, id, err)
				return nil, err
			}
			kc.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in kc: %w", updateTimeStr, id, err)
				return nil, err
			}
			kcs = append(kcs, kc)
		}
	}
	return kcs, nil
}

func (k *kcRepositorySQLite3Impl) AddKCInfo(ctx context.Context, kc *KC) error {
	k.m.Lock()
	defer k.m.Unlock()
	var err error
	var db *sql.DB
	if k.fullConnect {
		db = k.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, k.filename)
		if err != nil {
			return err
		}
		defer db.Close()
	}

	sql := `
INSERT INTO kc (
  IS_DELETED,
  ID,
  TITLE,
  NUM_VALUE,
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
		err = fmt.Errorf("error at add kc sql %s: %w", kc.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		kc.IsDeleted,
		kc.ID,
		kc.Title,
		kc.NumValue.String(),
		kc.RelatedTime.Format(sqlite3impl.TimeLayout),
		kc.CreateTime.Format(sqlite3impl.TimeLayout),
		kc.CreateApp,
		kc.CreateDevice,
		kc.CreateUser,
		kc.UpdateTime.Format(sqlite3impl.TimeLayout),
		kc.UpdateApp,
		kc.UpdateDevice,
		kc.UpdateUser,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to kc %s: %w", kc.ID, err)
		return err
	}
	return nil
}

func (k *kcRepositorySQLite3Impl) UnWrapTyped() ([]KCRepository, error) {
	return []KCRepository{k}, nil
}

func (k *kcRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{k}, nil
}

func (k *kcRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error) {
	dbName := "kc"
	var err error
	var db *sql.DB
	if k.fullConnect {
		db = k.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, k.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

	// update_cacheであればキャッシュを更新する
	if updateCache {
		err = k.UpdateCache(ctx)
		if err != nil {
			repName, _ := k.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

	sql := fmt.Sprintf(`
SELECT
  tbl.IS_DELETED,
  tbl.ID AS TARGET_ID,
  NULL AS TARGET_ID_IN_DATA,
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

	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at %s : %w", k.filename, err)
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
