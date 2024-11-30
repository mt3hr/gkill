package reps

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	sqllib "database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type timeIsRepositorySQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewTimeIsRepositorySQLite3Impl(ctx context.Context, filename string) (TimeIsRepository, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

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
	log.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

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
  ? AS REP_NAME,
  'timeis_end' AS DATA_TYPE
FROM TIMEIS 
`

	sqlWhereFilterEndTimeIs := ""
	if query.IncludeEndTimeIs != nil && *query.IncludeEndTimeIs {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start', 'timeis_end') AND END_TIME IS NOT NULL"
	} else {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start') AND END_TIME IS NOT NULL"
	}

	sqlWhereForStart, err := t.whereSQLGenerator(true, query)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd, err := t.whereSQLGenerator(false, query)
	if err != nil {
		return nil, err
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s AND %s", sqlStartTimeIs, sqlWhereForStart, sqlEndTimeIs, sqlWhereForEnd, sqlWhereFilterEndTimeIs)

	log.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyous sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at TIMEIS: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName, repName)
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

func (t *timeIsRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
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
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`
	log.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, repName, id)
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
	sqlWhereFilterEndTimeIs := ""
	if query.IncludeEndTimeIs != nil && *query.IncludeEndTimeIs {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start', 'timeis_end')"
	} else {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start')"
	}

	sqlWhereForStart, err := t.whereSQLGenerator(true, query)
	if err != nil {
		return nil, err
	}

	sql := fmt.Sprintf("%s WHERE %s AND %s", sqlStartTimeIs, sqlWhereForStart, sqlWhereFilterEndTimeIs)

	log.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find time is sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at TIMEIS: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName)
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

func (t *timeIsRepositorySQLite3Impl) GetTimeIs(ctx context.Context, id string) (*TimeIs, error) {
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
  ? AS REP_NAME
FROM TIMEIS 
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`

	log.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get time is histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at TIMEIS: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName, id)
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
	log.Printf("sql: %s", sql)
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
	_, err = stmt.ExecContext(ctx,
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
	)
	if err != nil {
		err = fmt.Errorf("error at insert in to timeis %s: %w", timeis.ID, err)
		return err
	}
	return nil
}

func (t *timeIsRepositorySQLite3Impl) whereSQLGenerator(forStartTime bool, query *find.FindQuery) (string, error) {
	sqlWhere := ""
	whereCounter := 0

	// 削除済みであるかどうかのSQL追記
	if query.IsDeleted != nil && *query.IsDeleted {
		if whereCounter != 0 {
			sqlWhere += " AND "
		}
		sqlWhere += "IS_DELETED = TRUE"
		whereCounter++
	} else {
		if whereCounter != 0 {
			sqlWhere += " AND "
		}
		sqlWhere += "IS_DELETED = FALSE"
		whereCounter++
	}

	// id検索である場合のSQL追記
	ids := []string{}
	if query.IDs != nil {
		ids = *query.IDs
	}
	if query.UseIDs != nil && *query.UseIDs {
		if whereCounter != 0 {
			sqlWhere += " AND "
		}
		sqlWhere += " ID IN ("
		for i, id := range ids {
			sqlWhere += fmt.Sprintf("'%s'", id)
			if i != len(ids)-1 {
				sqlWhere += ", "
			}
		}
		sqlWhere += ")"
		whereCounter++
	}

	// 日付範囲指定ありの場合
	if query.UseCalendar != nil && *query.UseCalendar {
		// 開始日時を指定するSQLを追記
		columnName := ""
		if forStartTime {
			columnName = "START_TIME"
		} else {
			columnName = "END_TIME"
		}
		if query.CalendarStartDate != nil {
			startDate := query.CalendarStartDate
			if whereCounter != 0 {
				sqlWhere += " AND "
			}
			sqlWhere += sqlite3impl.EscapeSQLite(fmt.Sprintf("datetime("+columnName+", 'localtime') >= datetime('%s', 'localtime')", startDate.Format(sqlite3impl.TimeLayout)))
			whereCounter++
		}

		// 終了日時を指定するSQLを追記
		if query.CalendarEndDate != nil {
			endDate := query.CalendarEndDate
			if whereCounter != 0 {
				sqlWhere += " AND "
			}
			sqlWhere += sqlite3impl.EscapeSQLite(fmt.Sprintf("datetime("+columnName+", 'localtime') <= datetime('%s', 'localtime')", endDate.Format(sqlite3impl.TimeLayout)))
			whereCounter++
		}
	}

	words := []string{}
	if query.Words != nil {
		words = *query.Words
	}
	notWords := []string{}
	if query.NotWords != nil {
		notWords = *query.NotWords
	}

	// ワードand検索である場合のSQL追記
	if query.UseWords != nil && *query.UseWords {
		if len(words) != 0 {
			if whereCounter != 0 {
				sqlWhere += " AND "
			}
		}

		if query.WordsAnd != nil && *query.WordsAnd {
			for i, word := range words {
				if i == 0 {
					sqlWhere += " ( "
				}
				if whereCounter != 0 {
					sqlWhere += " AND "
				}
				sqlWhere += sqlite3impl.EscapeSQLite("TITLE LIKE '%" + word + "%'")
				if i == len(words)-1 {
					sqlWhere += " ) "
				}
				whereCounter++
			}
		} else {
			// ワードor検索である場合のSQL追記
			for i, word := range words {
				if i == 0 {
					sqlWhere += " ( "
				}
				if whereCounter != 0 {
					sqlWhere += " AND "
				}
				sqlWhere += sqlite3impl.EscapeSQLite("TITLE LIKE '%" + word + "%'")
				if i == len(words)-1 {
					sqlWhere += " ) "
				}
				whereCounter++
			}
		}

		if len(notWords) != 0 {
			if whereCounter != 0 {
				sqlWhere += " AND "
			}
		}

		// notワードを除外するSQLを追記
		for i, notWord := range notWords {
			if i == 0 {
				sqlWhere += " ( "
			}
			if whereCounter != 0 {
				sqlWhere += " AND "
			}
			sqlWhere += sqlite3impl.EscapeSQLite("TITLE NOT LIKE '%" + notWord + "%'")
			if i == len(words)-1 {
				sqlWhere += " ) "
			}
			whereCounter++
		}
	}
	// plaingの場合
	if query.UsePlaing != nil && *query.UsePlaing {
		plaingTime := query.PlaingTime
		if whereCounter != 0 {
			sqlWhere += " AND "
		}
		sqlWhere += fmt.Sprintf("(datetime('"+plaingTime.Format(sqlite3impl.TimeLayout)+"', 'localtime') BETWEEN datetime(START_TIME, 'localtime') AND datetime(START_TIME, 'localtime'))", plaingTime.Format(sqlite3impl.TimeLayout))
	}
	// UPDATE_TIMEが一番上のものだけを抽出
	sqlWhere += `
GROUP BY ID
HAVING MAX(datetime(UPDATE_TIME, 'localtime'))
`
	return sqlWhere, nil
}
