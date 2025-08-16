package reps

import (
	"context"
	"fmt"
	"sync"
	"time"

	"database/sql"
	sqllib "database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type timeIsRepositoryCachedSQLite3Impl struct {
	dbName    string
	timeisRep TimeIsRepository
	cachedDB  *sqllib.DB
	m         *sync.Mutex
}

func NewTimeIsRepositoryCachedSQLite3Impl(ctx context.Context, timeisRep TimeIsRepository, cacheDB *sql.DB, m *sync.Mutex, dbName string) (TimeIsRepository, error) {
	if m == nil {
		m = &sync.Mutex{}
	}
	var err error

	sql := `
CREATE TABLE IF NOT EXISTS "` + dbName + `" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TITLE NOT NULL,
  START_TIME NOT NULL,
  END_TIME,
  CREATE_TIME NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_TIME NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  REP_NAME NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS table statement %s: %w", dbName, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS table to %s: %w", dbName, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS table to %s: %w", dbName, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_` + dbName + ` ON ` + dbName + ` (ID, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := cacheDB.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS index statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS index to %s: %w", dbName, err)
		return nil, err
	}

	return &timeIsRepositoryCachedSQLite3Impl{
		timeisRep: timeisRep,
		dbName:    dbName,
		cachedDB:  cacheDB,
		m:         m,
	}, nil
}
func (t *timeIsRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
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

	sqlStartTimeIs := `
SELECT 
  IS_DELETED,
  ID,
  START_TIME AS RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
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
  END_TIME AS RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  'timeis_end' AS DATA_TYPE
FROM ` + t.dbName + `
`

	sqlWhereFilterEndTimeIs := ""
	if query.IncludeEndTimeIs != nil && *query.IncludeEndTimeIs {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start', 'timeis_end') AND END_TIME IS NOT NULL"
	} else {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start')"
	}

	queryArgsForStart := []interface{}{}

	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false

	findWordUseLike := true
	ignoreCase := true
	queryArgsForPlaingStart := []interface{}{}
	sqlWhereFilterPlaingTimeisStart := ""
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisStart += " AND ((datetime(?, 'localtime') >= datetime(START_TIME, 'localtime')) AND (datetime(?, 'localtime') <= datetime(END_TIME, 'localtime') OR END_TIME IS NULL)) "
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Format(sqlite3impl.TimeLayout))
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Format(sqlite3impl.TimeLayout))
		whereCounter++
		whereCounter++
	}

	queryArgsForEnd := []interface{}{}

	whereCounter = 0
	onlyLatestData = true
	relatedTimeColumnName = "RELATED_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	queryArgsForPlaingEnd := []interface{}{}
	sqlWhereFilterPlaingTimeisEnd := ""
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisEnd += " AND ((datetime(?, 'localtime') >= datetime(START_TIME, 'localtime')) AND (datetime(?, 'localtime') <= datetime(END_TIME, 'localtime') OR END_TIME IS NULL)) "
		queryArgsForPlaingEnd = append(queryArgsForPlaingEnd, (*query.PlaingTime).Format(sqlite3impl.TimeLayout))
		queryArgsForPlaingEnd = append(queryArgsForPlaingEnd, (*query.PlaingTime).Format(sqlite3impl.TimeLayout))
		whereCounter++
		whereCounter++
	}

	sql := fmt.Sprintf("%s WHERE %s %s UNION %s WHERE %s %s AND %s", sqlStartTimeIs, sqlWhereForStart, sqlWhereFilterPlaingTimeisStart, sqlEndTimeIs, sqlWhereForEnd, sqlWhereFilterPlaingTimeisEnd, sqlWhereFilterEndTimeIs)
	sql += " ORDER BY RELATED_TIME DESC"

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyous sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v %#v %#v %#v", sql, queryArgsForStart, queryArgsForPlaingStart, queryArgsForEnd, queryArgsForPlaingEnd)
	rows, err := stmt.QueryContext(ctx, append(queryArgsForStart, append(queryArgsForPlaingStart, append(queryArgsForEnd, queryArgsForPlaingEnd...)...)...)...)
	if err != nil {
		err = fmt.Errorf("error at select from TIMEIS: %w", err)
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
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(
				&kyou.IsDeleted,
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
				err = fmt.Errorf("error at scan kyou: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TIMEIS: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TIMEIS: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TIMEIS: %w", updateTimeStr, err)
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

func (t *timeIsRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := t.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from TIMEIS%s: %w", id, err)
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

func (t *timeIsRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
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
  START_TIME AS RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
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

	queryArgs := []interface{}{}

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
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
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}
	defer stmt.Close()

	sql += " ORDER BY RELATED_TIME DESC"
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from TIMEIS %s: %w", id, err)
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

			err = rows.Scan(
				&kyou.IsDeleted,
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
				err = fmt.Errorf("error at scan kyou: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in TIMEIS: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in TIMEIS: %w", createTimeStr, id, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in TIMEIS: %w", updateTimeStr, id, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (t *timeIsRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return t.timeisRep.GetPath(ctx, id)
}

func (t *timeIsRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	// t.m.Lock()
	// defer t.m.Unlock()

	trueValue := true
	query := &find.FindQuery{
		UpdateCache: &trueValue,
	}

	allTimeiss, err := t.timeisRep.FindTimeIs(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all timeis at update cache: %w", err)
		return err
	}

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
	defer stmt.Close()
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
  START_TIME,
  END_TIME,
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
		err = fmt.Errorf("error at add timeis sql: %w", err)
		return err
	}
	defer insertStmt.Close()

	for _, timeis := range allTimeiss {
		select {
		case <-ctx.Done():
			tx.Rollback()
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			var endTimeStr interface{}
			if timeis.EndTime == nil {
				endTimeStr = nil
			} else {
				endTimeStr = timeis.EndTime.Format(sqlite3impl.TimeLayout)
			}

			queryArgs := []interface{}{
				timeis.IsDeleted,
				timeis.ID,
				timeis.Title,
				timeis.StartTime.Format(sqlite3impl.TimeLayout),
				endTimeStr,
				timeis.CreateTime.Format(sqlite3impl.TimeLayout),
				timeis.CreateApp,
				timeis.CreateDevice,
				timeis.CreateUser,
				timeis.UpdateTime.Format(sqlite3impl.TimeLayout),
				timeis.UpdateApp,
				timeis.UpdateDevice,
				timeis.UpdateUser,
				timeis.RepName,
			}
			gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
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
	_, err := t.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+t.dbName)
	return err
}

func (t *timeIsRepositoryCachedSQLite3Impl) FindTimeIs(ctx context.Context, query *find.FindQuery) ([]*TimeIs, error) {
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

	sqlStartTimeIs := `
SELECT 
  IS_DELETED,
  ID,
  TITLE,
  START_TIME,
  END_TIME,
  START_TIME AS RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
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
  START_TIME,
  END_TIME,
  END_TIME AS RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  'timeis_end' AS DATA_TYPE
FROM ` + t.dbName + `
`

	sqlWhereFilterEndTimeIs := ""
	if query.IncludeEndTimeIs != nil && *query.IncludeEndTimeIs {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start', 'timeis_end') AND END_TIME IS NOT NULL"
	} else {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start')"
	}

	queryArgsForStart := []interface{}{}

	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false

	findWordUseLike := true
	ignoreCase := true
	queryArgsForPlaingStart := []interface{}{}
	sqlWhereFilterPlaingTimeisStart := ""
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisStart += " AND ((datetime(?, 'localtime') >= datetime(START_TIME, 'localtime')) AND (datetime(?, 'localtime') <= datetime(END_TIME, 'localtime') OR END_TIME IS NULL)) "
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Format(sqlite3impl.TimeLayout))
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Format(sqlite3impl.TimeLayout))
		whereCounter++
		whereCounter++
	}

	queryArgsForEnd := []interface{}{}

	whereCounter = 0
	onlyLatestData = true
	relatedTimeColumnName = "RELATED_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	queryArgsForPlaingEnd := []interface{}{}
	sqlWhereFilterPlaingTimeisEnd := ""
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisEnd += " AND ((datetime(?, 'localtime') >= datetime(START_TIME, 'localtime')) AND (datetime(?, 'localtime') <= datetime(END_TIME, 'localtime') OR END_TIME IS NULL)) "
		queryArgsForPlaingEnd = append(queryArgsForPlaingEnd, (*query.PlaingTime).Format(sqlite3impl.TimeLayout))
		queryArgsForPlaingEnd = append(queryArgsForPlaingEnd, (*query.PlaingTime).Format(sqlite3impl.TimeLayout))
		whereCounter++
		whereCounter++
	}

	sql := fmt.Sprintf("%s WHERE %s %s UNION %s WHERE %s %s AND %s", sqlStartTimeIs, sqlWhereForStart, sqlWhereFilterPlaingTimeisStart, sqlEndTimeIs, sqlWhereForEnd, sqlWhereFilterPlaingTimeisEnd, sqlWhereFilterEndTimeIs)
	sql += " ORDER BY RELATED_TIME DESC"

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyous sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v %#v %#v %#v", sql, queryArgsForStart, queryArgsForPlaingStart, queryArgsForEnd, queryArgsForPlaingEnd)
	rows, err := stmt.QueryContext(ctx, append(queryArgsForStart, append(queryArgsForPlaingStart, append(queryArgsForEnd, queryArgsForPlaingEnd...)...)...)...)
	if err != nil {
		err = fmt.Errorf("error at select from TIMEIS: %w", err)
		return nil, err
	}
	defer rows.Close()

	timeiss := []*TimeIs{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			timeis := &TimeIs{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			startTimeStr, endTime := "", sqllib.NullString{}

			err = rows.Scan(
				&timeis.IsDeleted,
				&timeis.ID,
				&timeis.Title,
				&startTimeStr,
				&endTime,
				&relatedTimeStr,
				&createTimeStr,
				&timeis.CreateApp,
				&timeis.CreateDevice,
				&timeis.CreateUser,
				&updateTimeStr,
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

			timeis.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TIMEIS: %w", createTimeStr, err)
				return nil, err
			}
			timeis.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TIMEIS: %w", updateTimeStr, err)
				return nil, err
			}
			timeis.StartTime, err = time.Parse(sqlite3impl.TimeLayout, startTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse start time %s in TIMEIS: %w", updateTimeStr, err)
				return nil, err
			}
			if endTime.Valid {
				parsedEndTime, _ := time.Parse(sqlite3impl.TimeLayout, endTime.String)
				timeis.EndTime = &parsedEndTime
			}
			timeiss = append(timeiss, timeis)
		}
	}
	return timeiss, nil
}

func (t *timeIsRepositoryCachedSQLite3Impl) GetTimeIs(ctx context.Context, id string, updateTime *time.Time) (*TimeIs, error) {
	// 最新のデータを返す
	timeisHistories, err := t.GetTimeIsHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get timeis histories from TIMEIS%s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(timeisHistories) == 0 {
		return nil, nil
	}

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range timeisHistories {
			if kyou.UpdateTime.Format(sqlite3impl.TimeLayout) == updateTime.Format(sqlite3impl.TimeLayout) {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return timeisHistories[0], nil
}

func (t *timeIsRepositoryCachedSQLite3Impl) GetTimeIsHistories(ctx context.Context, id string) ([]*TimeIs, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TITLE,
  START_TIME,
  END_TIME,
  CREATE_TIME AS RELATED_TIME,
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

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false

	queryArgsForPlaingStart := []interface{}{}
	sqlWhereFilterPlaingTimeisStart := ""
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisStart += " AND ((datetime(?, 'localtime') >= datetime(START_TIME, 'localtime')) AND (datetime(?, 'localtime') <= datetime(END_TIME, 'localtime') OR END_TIME IS NULL)) "
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Format(sqlite3impl.TimeLayout))
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Format(sqlite3impl.TimeLayout))
		whereCounter++
		whereCounter++
	}

	sql += commonWhereSQL + sqlWhereFilterPlaingTimeisStart
	sql += " ORDER BY datetime(UPDATE_TIME, 'localtime') DESC "
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get time is histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, append(queryArgsForPlaingStart, queryArgs...))
	rows, err := stmt.QueryContext(ctx, append(queryArgsForPlaingStart, queryArgs...)...)

	if err != nil {
		err = fmt.Errorf("error at select from TIMEIS: %w", err)
		return nil, err
	}
	defer rows.Close()

	timeiss := []*TimeIs{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			timeis := &TimeIs{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			startTimeStr, endTime := "", sqllib.NullString{}

			err = rows.Scan(
				&timeis.IsDeleted,
				&timeis.ID,
				&timeis.Title,
				&startTimeStr,
				&endTime,
				&relatedTimeStr,
				&createTimeStr,
				&timeis.CreateApp,
				&timeis.CreateDevice,
				&timeis.CreateUser,
				&updateTimeStr,
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

			timeis.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TIMEIS: %w", createTimeStr, err)
				return nil, err
			}
			timeis.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TIMEIS: %w", updateTimeStr, err)
				return nil, err
			}
			timeis.StartTime, err = time.Parse(sqlite3impl.TimeLayout, startTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse start time %s in TIMEIS: %w", updateTimeStr, err)
				return nil, err
			}
			if endTime.Valid {
				parsedEndTime, _ := time.Parse(sqlite3impl.TimeLayout, endTime.String)
				timeis.EndTime = &parsedEndTime
			}
			timeiss = append(timeiss, timeis)
		}
	}
	return timeiss, nil
}

func (t *timeIsRepositoryCachedSQLite3Impl) AddTimeIsInfo(ctx context.Context, timeis *TimeIs) error {
	sql := `
INSERT INTO ` + t.dbName + `(
  IS_DELETED,
  ID,
  TITLE,
  START_TIME,
  END_TIME,
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
		err = fmt.Errorf("error at add timeis sql %s: %w", timeis.ID, err)
		return err
	}
	defer stmt.Close()

	var endTimeStr interface{}
	if timeis.EndTime == nil {
		endTimeStr = nil
	} else {
		endTimeStr = timeis.EndTime.Format(sqlite3impl.TimeLayout)
	}

	queryArgs := []interface{}{
		timeis.IsDeleted,
		timeis.ID,
		timeis.Title,
		timeis.StartTime.Format(sqlite3impl.TimeLayout),
		endTimeStr,
		timeis.CreateTime.Format(sqlite3impl.TimeLayout),
		timeis.CreateApp,
		timeis.CreateDevice,
		timeis.CreateUser,
		timeis.UpdateTime.Format(sqlite3impl.TimeLayout),
		timeis.UpdateApp,
		timeis.UpdateDevice,
		timeis.UpdateUser,
		timeis.RepName,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to timeis %s: %w", timeis.ID, err)
		return err
	}
	return nil
}
