package reps

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	sqllib "database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type timeIsRepositorySQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewTimeIsRepositorySQLite3Impl(ctx context.Context, filename string) (TimeIsRepository, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=60000&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)

	sql := `
CREATE TABLE IF NOT EXISTS "TIMEIS" (
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
  UPDATE_USER NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS table to %s: %w", filename, err)
		return nil, err
	}

	return &timeIsRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (t *timeIsRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error) {
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
  ? AS REP_NAME,
  'timeis_start' AS DATA_TYPE
FROM TIMEIS 
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at TIMEIS: %w", err)
		return nil, err
	}

	queryArgsForStart := []interface{}{
		repName,
	}

	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	appendGroupBy := true
	findWordUseLike := true
	queryArgsForPlaingStart := []interface{}{}
	sqlWhereFilterPlaingTimeisStart := ""
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendGroupBy, appendOrderBy, &queryArgsForStart)
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

	sql := fmt.Sprintf("%s WHERE %s %s", sqlStartTimeIs, sqlWhereForStart, sqlWhereFilterPlaingTimeisStart)
	sql += " ORDER BY RELATED_TIME DESC"

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyous sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, append(queryArgsForStart, queryArgsForPlaingStart...))
	rows, err := stmt.QueryContext(ctx, append(queryArgsForStart, queryArgsForPlaingStart...)...)
	if err != nil {
		err = fmt.Errorf("error at select from TIMEIS%s: %w", err)
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
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (t *timeIsRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
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

func (t *timeIsRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
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
  ? AS REP_NAME,
  'timeis_start' AS DATA_TYPE
FROM TIMEIS 
WHERE 
`
	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	queryArgs := []interface{}{
		repName,
	}

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	appendGroupBy := false
	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendGroupBy, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}
	commonWhereSQL += " ORDER BY datetime(UPDATE_TIME, 'localtime') DESC "
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
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

func (t *timeIsRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return filepath.Abs(t.filename)
}

func (t *timeIsRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (t *timeIsRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := t.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path timeis rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (t *timeIsRepositorySQLite3Impl) Close(ctx context.Context) error {
	return t.db.Close()
}

func (t *timeIsRepositorySQLite3Impl) FindTimeIs(ctx context.Context, query *find.FindQuery) ([]*TimeIs, error) {
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
  ? AS REP_NAME,
  'timeis_start' AS DATA_TYPE
FROM TIMEIS 
`
	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at TIMEIS: %w", err)
		return nil, err
	}

	queryArgs := []interface{}{
		repName,
	}

	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	appendGroupBy := true
	findWordUseLike := true
	queryArgsForPlaingStart := []interface{}{}
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendGroupBy, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}
	sqlWhereFilterPlaingTimeisStart := ""
	if query.UsePlaing != nil && *query.UsePlaing && query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisStart += " AND ((datetime(?, 'localtime') >= datetime(START_TIME, 'localtime')) AND (datetime(?, 'localtime') <= datetime(END_TIME, 'localtime') OR END_TIME IS NULL)) "
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Format(sqlite3impl.TimeLayout))
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (*query.PlaingTime).Format(sqlite3impl.TimeLayout))
		whereCounter++
		whereCounter++
	}

	sql := fmt.Sprintf("%s WHERE %s %s", sqlStartTimeIs, sqlWhereForStart, sqlWhereFilterPlaingTimeisStart)

	sql += " ORDER BY RELATED_TIME DESC"
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find time is sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, append(queryArgs, queryArgsForPlaingStart...))
	rows, err := stmt.QueryContext(ctx, append(queryArgs, queryArgsForPlaingStart...)...)
	if err != nil {
		err = fmt.Errorf("error at select from TIMEIS%s: %w", err)
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
			timeis.RepName = repName
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

func (t *timeIsRepositorySQLite3Impl) GetTimeIs(ctx context.Context, id string, updateTime *time.Time) (*TimeIs, error) {
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

func (t *timeIsRepositorySQLite3Impl) GetTimeIsHistories(ctx context.Context, id string) ([]*TimeIs, error) {
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
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM TIMEIS 
WHERE
`
	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at TIMEIS: %w", err)
		return nil, err
	}

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	dataType := "timeis"

	queryArgs := []interface{}{
		repName,
		dataType,
	}

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	appendGroupBy := false

	queryArgsForPlaingStart := []interface{}{}
	sqlWhereFilterPlaingTimeisStart := ""
	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendGroupBy, appendOrderBy, &queryArgs)
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
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get time is histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, append(queryArgsForPlaingStart, queryArgs...))
	rows, err := stmt.QueryContext(ctx, append(queryArgsForPlaingStart, queryArgs...)...)

	if err != nil {
		err = fmt.Errorf("error at select from TIMEIS%s: %w", err)
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
			timeis.RepName = repName
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

func (t *timeIsRepositorySQLite3Impl) AddTimeIsInfo(ctx context.Context, timeis *TimeIs) error {
	sql := `
INSERT INTO TIMEIS (
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
	stmt, err := t.db.PrepareContext(ctx, sql)
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
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to timeis %s: %w", timeis.ID, err)
		return err
	}
	return nil
}
