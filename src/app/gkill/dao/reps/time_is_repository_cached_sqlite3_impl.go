package reps

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	sqllib "database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type timeIsRepositoryCachedSQLite3Impl struct {
	dbName    string
	timeisRep TimeIsRepository
	cachedDB  *sqllib.DB
	m         *sync.RWMutex
}

func NewTimeIsRepositoryCachedSQLite3Impl(ctx context.Context, timeisRep TimeIsRepository, cacheDB *sqllib.DB, m *sync.RWMutex, dbName string) (TimeIsRepository, error) {
	if m == nil {
		m = &sync.RWMutex{}
	}
	var err error

	sql := `
CREATE TABLE IF NOT EXISTS "` + dbName + `" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TITLE NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  REP_NAME NOT NULL,
  START_TIME_UNIX NOT NULL,
  END_TIME_UNIX,
  CREATE_TIME_UNIX NOT NULL,
  UPDATE_TIME_UNIX NOT NULL
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS table statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create TIMEIS table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `_UNIX" ON "` + dbName + `"(ID, UPDATE_TIME_UNIX);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create timeis index unix statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create timeis index unix to %s: %w", dbName, err)
		return nil, err
	}

	return &timeIsRepositoryCachedSQLite3Impl{
		timeisRep: timeisRep,
		dbName:    dbName,
		cachedDB:  cacheDB,
		m:         m,
	}, nil
}
func (t *timeIsRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
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
	t.m.RLock()
	defer t.m.RUnlock()

	sqlStartTimeIs := `
SELECT 
  IS_DELETED,
  ID,
  START_TIME_UNIX AS RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  'timeis_start' AS DATA_TYPE
FROM ` + t.dbName + `
`
	sqlEndTimeIs := `
SELECT 
  IS_DELETED,
  ID,
  END_TIME_UNIX AS RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  'timeis_end' AS DATA_TYPE
FROM ` + t.dbName + `
`

	sqlWhereFilterEndTimeIs := ""
	if query.IncludeEndTimeIs != nil && *query.IncludeEndTimeIs {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start', 'timeis_end') AND END_TIME_UNIX IS NOT NULL"
	} else {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start')"
	}

	queryArgsForStart := []interface{}{}

	tableName := t.dbName
	tableNameAlias := t.dbName
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	if query.OnlyLatestData != nil {
		onlyLatestData = *query.OnlyLatestData
	} else {
		onlyLatestData = false
	}
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		onlyLatestData = true
	}
	queryArgsForPlaingStart := []interface{}{}
	sqlWhereFilterPlaingTimeisStart := ""
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisStart += " AND ((? >= START_TIME_UNIX) AND (? <= END_TIME_UNIX OR END_TIME_UNIX IS NULL)) "
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Unix())
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Unix())
		whereCounter++
		whereCounter++
	}

	tableName = t.dbName
	tableNameAlias = t.dbName
	queryArgsForEnd := []interface{}{}
	whereCounter = 0
	onlyLatestData = true
	relatedTimeColumnName = "RELATED_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	queryArgsForPlaingEnd := []interface{}{}
	sqlWhereFilterPlaingTimeisEnd := ""
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisEnd += " AND ((? >= START_TIME_UNIX) AND (? <= END_TIME_UNIX OR END_TIME_UNIX IS NULL)) "
		queryArgsForPlaingEnd = append(queryArgsForPlaingEnd, (*query.PlaingTime).Unix())
		queryArgsForPlaingEnd = append(queryArgsForPlaingEnd, (*query.PlaingTime).Unix())
		whereCounter++
		whereCounter++
	}

	orderby := " ORDER BY END_TIME_UNIX DESC "
	sql := fmt.Sprintf("%s WHERE %s %s UNION %s WHERE %s %s AND %s %s", sqlStartTimeIs, sqlWhereForStart, sqlWhereFilterPlaingTimeisStart, sqlEndTimeIs, sqlWhereForEnd, sqlWhereFilterPlaingTimeisEnd, sqlWhereFilterEndTimeIs, orderby)

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyous sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql, "params", queryArgsForStart, "params", queryArgsForPlaingStart, "params", queryArgsForEnd, "params", queryArgsForPlaingEnd)
	rows, err := stmt.QueryContext(ctx, append(queryArgsForStart, append(queryArgsForPlaingStart, append(queryArgsForEnd, queryArgsForPlaingEnd...)...)...)...)
	if err != nil {
		err = fmt.Errorf("error at select from TIMEIS: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	kyous := map[string][]Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := Kyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(
				&kyou.IsDeleted,
				&kyou.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&kyou.CreateApp,
				&kyou.CreateDevice,
				&kyou.CreateUser,
				&updateTimeUnix,
				&kyou.UpdateApp,
				&kyou.UpdateDevice,
				&kyou.UpdateUser,
				&kyou.RepName,
				&kyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan kyou: %w", err)
				return nil, err
			}

			kyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			kyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			kyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()

			if _, exist := kyous[kyou.ID]; !exist {
				kyous[kyou.ID] = []Kyou{}
			}
			kyous[kyou.ID] = append(kyous[kyou.ID], kyou)
		}
	}
	return kyous, nil
}

func (t *timeIsRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at TIMEIS: %w", err)
		return nil, err
	}

	// startのみ
	sql := `
SELECT 
  IS_DELETED,
  ID,
  START_TIME_UNIX AS RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  'timeis_start' AS DATA_TYPE
FROM ` + t.dbName + `
WHERE 
`
	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs:         &trueValue,
		IDs:            &ids,
		OnlyLatestData: new(updateTime == nil),
		UseUpdateTime:  new(updateTime != nil),
		UpdateTime:     updateTime,
	}

	tableName := t.dbName
	tableNameAlias := t.dbName
	queryArgs := []interface{}{}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
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
		err = fmt.Errorf("error at select from TIMEIS %s: %w", id, err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	kyous := []Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := Kyou{}
			kyou.RepName = repName
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(
				&kyou.IsDeleted,
				&kyou.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&kyou.CreateApp,
				&kyou.CreateDevice,
				&kyou.CreateUser,
				&updateTimeUnix,
				&kyou.UpdateApp,
				&kyou.UpdateDevice,
				&kyou.UpdateUser,
				&kyou.RepName,
				&kyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan kyou: %w", err)
				return nil, err
			}

			kyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			kyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			kyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			kyous = append(kyous, kyou)
		}
	}
	if len(kyous) == 0 {
		return nil, nil
	}
	return &kyous[0], nil
}

func (t *timeIsRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at TIMEIS: %w", err)
		return nil, err
	}

	// startのみ
	sql := `
SELECT 
  IS_DELETED,
  ID,
  START_TIME_UNIX AS RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  'timeis_start' AS DATA_TYPE
FROM ` + t.dbName + `
WHERE 
`
	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	tableName := t.dbName
	tableNameAlias := t.dbName
	queryArgs := []interface{}{}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
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
		err = fmt.Errorf("error at select from TIMEIS %s: %w", id, err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	kyous := []Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := Kyou{}
			kyou.RepName = repName
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(
				&kyou.IsDeleted,
				&kyou.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&kyou.CreateApp,
				&kyou.CreateDevice,
				&kyou.CreateUser,
				&updateTimeUnix,
				&kyou.UpdateApp,
				&kyou.UpdateDevice,
				&kyou.UpdateUser,
				&kyou.RepName,
				&kyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan kyou: %w", err)
				return nil, err
			}

			kyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			kyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			kyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (t *timeIsRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return t.timeisRep.GetPath(ctx, id)
}

func (t *timeIsRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	trueValue := true
	falseValue := false
	query := &find.FindQuery{
		UpdateCache:    &trueValue,
		OnlyLatestData: &falseValue,
	}

	allTimeiss, err := t.timeisRep.FindTimeIs(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all timeis at update cache: %w", err)
		return err
	}

	t.m.Lock()
	defer t.m.Unlock()

	tx, err := t.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add timeis: %w", err)
		return err
	}

	sql := `DELETE FROM ` + t.dbName
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS table statement %s: %w", "memory", err)
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
		err = fmt.Errorf("error at delete TIMEIS table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + t.dbName + `(
  IS_DELETED,
  ID,
  TITLE,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  START_TIME_UNIX,
  END_TIME_UNIX,
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
		err = fmt.Errorf("error at add timeis sql: %w", err)
		return err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for _, timeis := range allTimeiss {
		select {
		case <-ctx.Done():
			tx.Rollback()
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			var endTimeUnix interface{}
			if timeis.EndTime == nil {
				endTimeUnix = nil
			} else {
				endTimeUnix = timeis.EndTime.Unix()
			}

			queryArgs := []interface{}{
				timeis.IsDeleted,
				timeis.ID,
				timeis.Title,
				timeis.CreateApp,
				timeis.CreateDevice,
				timeis.CreateUser,
				timeis.UpdateApp,
				timeis.UpdateDevice,
				timeis.UpdateUser,
				timeis.RepName,
				timeis.StartTime.Unix(),
				endTimeUnix,
				timeis.CreateTime.Unix(),
				timeis.UpdateTime.Unix(),
			}
			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to timeis %s: %w", timeis.ID, err)
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
		err = fmt.Errorf("error at commit transaction for add timeiss: %w", err)
		return err
	}
	return nil
}

func (t *timeIsRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return t.timeisRep.GetRepName(ctx)
}

func (t *timeIsRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	t.m.Lock()
	defer t.m.Unlock()
	err := t.timeisRep.Close(ctx)
	if err != nil {
		return err
	}
	if gkill_options.CacheTimeIsReps == nil || !*gkill_options.CacheTimeIsReps {
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

func (t *timeIsRepositoryCachedSQLite3Impl) FindTimeIs(ctx context.Context, query *find.FindQuery) ([]TimeIs, error) {
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
	t.m.RLock()
	defer t.m.RUnlock()

	sqlStartTimeIs := `
SELECT 
  IS_DELETED,
  ID,
  TITLE,
  START_TIME_UNIX,
  END_TIME_UNIX,
  START_TIME_UNIX AS RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  'timeis_start' AS DATA_TYPE
FROM ` + t.dbName + `
`
	sqlEndTimeIs := `
SELECT 
  IS_DELETED,
  ID,
  TITLE,
  START_TIME_UNIX,
  END_TIME_UNIX,
  END_TIME_UNIX AS RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  'timeis_end' AS DATA_TYPE
FROM ` + t.dbName + `
`

	sqlWhereFilterEndTimeIs := ""
	if query.IncludeEndTimeIs != nil && *query.IncludeEndTimeIs {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start', 'timeis_end') AND END_TIME_UNIX IS NOT NULL"
	} else {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start')"
	}

	queryArgsForStart := []interface{}{}
	tableName := t.dbName
	tableNameAlias := t.dbName
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	if query.OnlyLatestData != nil {
		onlyLatestData = *query.OnlyLatestData
	} else {
		onlyLatestData = false
	}
	queryArgsForPlaingStart := []interface{}{}
	sqlWhereFilterPlaingTimeisStart := ""
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisStart += " AND ((? >= START_TIME_UNIX) AND (? <= END_TIME_UNIX OR END_TIME_UNIX IS NULL)) "
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Unix())
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Unix())
		whereCounter++
		whereCounter++
	}

	queryArgsForEnd := []interface{}{}
	tableName = t.dbName
	tableNameAlias = t.dbName
	whereCounter = 0
	onlyLatestData = true
	relatedTimeColumnName = "RELATED_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	queryArgsForPlaingEnd := []interface{}{}
	sqlWhereFilterPlaingTimeisEnd := ""
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisEnd += " AND ((? >= START_TIME_UNIX) AND (? <= END_TIME_UNIX OR END_TIME_UNIX IS NULL)) "
		queryArgsForPlaingEnd = append(queryArgsForPlaingEnd, (*query.PlaingTime).Unix())
		queryArgsForPlaingEnd = append(queryArgsForPlaingEnd, (*query.PlaingTime).Unix())
		whereCounter++
		whereCounter++
	}

	sql := fmt.Sprintf("%s WHERE %s %s UNION %s WHERE %s %s AND %s", sqlStartTimeIs, sqlWhereForStart, sqlWhereFilterPlaingTimeisStart, sqlEndTimeIs, sqlWhereForEnd, sqlWhereFilterPlaingTimeisEnd, sqlWhereFilterEndTimeIs)

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyous sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql, "params", queryArgsForStart, "params", queryArgsForPlaingStart, "params", queryArgsForEnd, "params", queryArgsForPlaingEnd)
	rows, err := stmt.QueryContext(ctx, append(queryArgsForStart, append(queryArgsForPlaingStart, append(queryArgsForEnd, queryArgsForPlaingEnd...)...)...)...)
	if err != nil {
		err = fmt.Errorf("error at select from TIMEIS: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	timeiss := []TimeIs{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			timeis := TimeIs{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			startTimeUnix, endTimeUnix := int64(0), sqllib.NullInt64{}

			err = rows.Scan(
				&timeis.IsDeleted,
				&timeis.ID,
				&timeis.Title,
				&startTimeUnix,
				&endTimeUnix,
				&relatedTimeUnix,
				&createTimeUnix,
				&timeis.CreateApp,
				&timeis.CreateDevice,
				&timeis.CreateUser,
				&updateTimeUnix,
				&timeis.UpdateApp,
				&timeis.UpdateDevice,
				&timeis.UpdateUser,
				&timeis.RepName,
				&timeis.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan timeis: %w", err)
				return nil, err
			}

			timeis.CreateTime = time.Unix(createTimeUnix, 0).Local()
			timeis.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			timeis.StartTime = time.Unix(startTimeUnix, 0).Local()
			if endTimeUnix.Valid {
				parsedEndTime := time.Unix(endTimeUnix.Int64, 0).Local()
				timeis.EndTime = &parsedEndTime
			}
			timeiss = append(timeiss, timeis)
		}
	}
	return timeiss, nil
}

func (t *timeIsRepositoryCachedSQLite3Impl) GetTimeIs(ctx context.Context, id string, updateTime *time.Time) (*TimeIs, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TITLE,
  START_TIME_UNIX,
  END_TIME_UNIX,
  CREATE_TIME_UNIX AS RELATED_TIME_UNIX,
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
	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs:         &trueValue,
		IDs:            &ids,
		OnlyLatestData: new(updateTime == nil),
		UseUpdateTime:  new(updateTime != nil),
		UpdateTime:     updateTime,
	}

	dataType := "timeis"

	queryArgs := []interface{}{
		dataType,
	}

	tableName := t.dbName
	tableNameAlias := t.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	queryArgsForPlaingStart := []interface{}{}
	sqlWhereFilterPlaingTimeisStart := ""
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisStart += " AND ((? >= START_TIME_UNIX) AND (? <= END_TIME_UNIX OR END_TIME_UNIX IS NULL)) "
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Unix())
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Unix())
		whereCounter++
		whereCounter++
	}

	sql += commonWhereSQL + sqlWhereFilterPlaingTimeisStart
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get time is histories sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, append(queryArgsForPlaingStart, queryArgs...))
	rows, err := stmt.QueryContext(ctx, append(queryArgsForPlaingStart, queryArgs...)...)

	if err != nil {
		err = fmt.Errorf("error at select from TIMEIS: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	timeiss := []TimeIs{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			timeis := TimeIs{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			startTimeUnix, endTimeUnix := int64(0), sqllib.NullInt64{}

			err = rows.Scan(
				&timeis.IsDeleted,
				&timeis.ID,
				&timeis.Title,
				&startTimeUnix,
				&endTimeUnix,
				&relatedTimeUnix,
				&createTimeUnix,
				&timeis.CreateApp,
				&timeis.CreateDevice,
				&timeis.CreateUser,
				&updateTimeUnix,
				&timeis.UpdateApp,
				&timeis.UpdateDevice,
				&timeis.UpdateUser,
				&timeis.RepName,
				&timeis.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan timeis: %w", err)
				return nil, err
			}

			timeis.CreateTime = time.Unix(createTimeUnix, 0).Local()
			timeis.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			timeis.StartTime = time.Unix(startTimeUnix, 0).Local()
			if endTimeUnix.Valid {
				parsedEndTime := time.Unix(endTimeUnix.Int64, 0).Local()
				timeis.EndTime = &parsedEndTime
			}
			timeiss = append(timeiss, timeis)
		}
	}
	if len(timeiss) == 0 {
		return nil, nil
	}
	return &timeiss[0], nil
}

func (t *timeIsRepositoryCachedSQLite3Impl) GetTimeIsHistories(ctx context.Context, id string) ([]TimeIs, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TITLE,
  START_TIME_UNIX,
  END_TIME_UNIX,
  CREATE_TIME_UNIX AS RELATED_TIME_UNIX,
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
	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	dataType := "timeis"

	queryArgs := []interface{}{
		dataType,
	}

	tableName := t.dbName
	tableNameAlias := t.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	queryArgsForPlaingStart := []interface{}{}
	sqlWhereFilterPlaingTimeisStart := ""
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisStart += " AND ((? >= START_TIME_UNIX) AND (? <= END_TIME_UNIX OR END_TIME_UNIX IS NULL)) "
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Unix())
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Unix())
		whereCounter++
		whereCounter++
	}

	sql += commonWhereSQL + sqlWhereFilterPlaingTimeisStart
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get time is histories sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, append(queryArgsForPlaingStart, queryArgs...))
	rows, err := stmt.QueryContext(ctx, append(queryArgsForPlaingStart, queryArgs...)...)

	if err != nil {
		err = fmt.Errorf("error at select from TIMEIS: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	timeiss := []TimeIs{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			timeis := TimeIs{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			startTimeUnix, endTimeUnix := int64(0), sqllib.NullInt64{}

			err = rows.Scan(
				&timeis.IsDeleted,
				&timeis.ID,
				&timeis.Title,
				&startTimeUnix,
				&endTimeUnix,
				&relatedTimeUnix,
				&createTimeUnix,
				&timeis.CreateApp,
				&timeis.CreateDevice,
				&timeis.CreateUser,
				&updateTimeUnix,
				&timeis.UpdateApp,
				&timeis.UpdateDevice,
				&timeis.UpdateUser,
				&timeis.RepName,
				&timeis.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan timeis: %w", err)
				return nil, err
			}

			timeis.CreateTime = time.Unix(createTimeUnix, 0).Local()
			timeis.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			timeis.StartTime = time.Unix(startTimeUnix, 0).Local()
			if endTimeUnix.Valid {
				parsedEndTime := time.Unix(endTimeUnix.Int64, 0).Local()
				timeis.EndTime = &parsedEndTime
			}
			timeiss = append(timeiss, timeis)
		}
	}
	return timeiss, nil
}

func (t *timeIsRepositoryCachedSQLite3Impl) AddTimeIsInfo(ctx context.Context, timeis TimeIs) error {
	t.m.Lock()
	defer t.m.Unlock()
	sql := `
INSERT INTO ` + t.dbName + `(
  IS_DELETED,
  ID,
  TITLE,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  START_TIME_UNIX,
  END_TIME_UNIX,
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
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add timeis sql %s: %w", timeis.ID, err)
		return err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	var endTimeUnix interface{}
	if timeis.EndTime == nil {
		endTimeUnix = nil
	} else {
		endTimeUnix = timeis.EndTime.Unix()
	}

	queryArgs := []interface{}{
		timeis.IsDeleted,
		timeis.ID,
		timeis.Title,
		timeis.CreateApp,
		timeis.CreateDevice,
		timeis.CreateUser,
		timeis.UpdateApp,
		timeis.UpdateDevice,
		timeis.UpdateUser,
		timeis.RepName,
		timeis.StartTime.Unix(),
		endTimeUnix,
		timeis.CreateTime.Unix(),
		timeis.UpdateTime.Unix(),
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to timeis %s: %w", timeis.ID, err)
		return err
	}
	return nil
}

func (t *timeIsRepositoryCachedSQLite3Impl) UnWrapTyped() ([]TimeIsRepository, error) {
	return t.timeisRep.UnWrapTyped()
}

func (t *timeIsRepositoryCachedSQLite3Impl) UnWrap() ([]Repository, error) {
	return t.timeisRep.UnWrap()
}
