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

type miRepositorySQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewMiRepositorySQLite3Impl(ctx context.Context, filename string) (MiRepository, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "MI" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TITLE NOT NULL,
  IS_CHECKED NOT NULL,
  CHECKED_TIME,
  BOARD_NAME NOT NULL,
  LIMIT_TIME,
  ESTIMATE_START_TIME,
  ESTIMATE_END_TIME,
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
		err = fmt.Errorf("error at create MI table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MI table to %s: %w", filename, err)
		return nil, err
	}

	return &miRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (m *miRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error) {
	var err error
	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = m.UpdateCache(ctx)
		if err != nil {
			repName, _ := m.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

	sqlCreateMi := `
		SELECT
		  IS_DELETED,
		  ID,
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
		  'mi_limit' AS DATA_TYPE
		FROM MI
		`

	sqlCheckMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  CHECKED_TIME AS RELATED_TIME,
		  CREATE_TIME,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  ? AS REP_NAME,
		  'mi_limit' AS DATA_TYPE
		FROM MI
		`

	sqlLimitMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  LIMIT_TIME AS RELATED_TIME,
		  CREATE_TIME,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  ? AS REP_NAME,
		  'mi_limit' AS DATA_TYPE
		FROM MI
		`
	sqlStartMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  ESTIMATE_START_TIME AS RELATED_TIME,
		  CREATE_TIME,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  ? AS REP_NAME,
		  'mi_start' AS DATA_TYPE
		FROM MI
		`

	sqlEndMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  ESTIMATE_END_TIME AS RELATED_TIME,
		  CREATE_TIME,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  ? AS REP_NAME,
		  'mi_end' AS DATA_TYPE
		FROM MI
		`

	// 検索対象のデータ抽出用WHERE
	filterWhereCounter := 0
	sqlWhereFilterEndMi := ""
	sqlWhereFilterEndMi += "DATA_TYPE IN ("
	if query.IncludeCheckMi != nil && *query.IncludeCheckMi {
		if filterWhereCounter == 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_check'"
	}
	if query.IncludeLimitMi != nil && *query.IncludeLimitMi {
		if filterWhereCounter == 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_limit'"
	}
	if query.IncludeStartMi != nil && *query.IncludeStartMi {
		if filterWhereCounter == 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_start'"
	}
	if query.IncludeEndMi != nil && *query.IncludeEndMi {
		if filterWhereCounter == 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_end'"
	}
	sqlWhereFilterEndMi += ")"
	if filterWhereCounter == 0 {
		sqlWhereFilterEndMi = ""
	}

	sqlWhereForCreate, err := m.whereSQLGenerator("CREATE_TIME", query)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck, err := m.whereSQLGenerator("CHECKED_TIME", query)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck += " AND CHECKED_TIME IS NOT NULL "
	sqlWhereForLimit, err := m.whereSQLGenerator("LIMIT_TIME", query)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit += " AND LIMIT_TIME IS NOT NULL "
	sqlWhereForStart, err := m.whereSQLGenerator("ESTIMATE_START_TIME", query)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart += " AND ESTIMATE_START_TIME IS NOT NULL "
	sqlWhereForEnd, err := m.whereSQLGenerator("ESTIMATE_END_TIME", query)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd += " AND ESTIMATE_END_TIME IS NOT NULL "
	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s %s", sqlCreateMi, sqlWhereForCreate, sqlCheckMi, sqlWhereForCheck, sqlLimitMi, sqlWhereForLimit, sqlStartMi, sqlWhereForStart, sqlEndMi, sqlWhereForEnd, sqlWhereFilterEndMi)

	log.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := m.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MI: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName, repName, repName, repName, repName)
	if err != nil {
		err = fmt.Errorf("error at select from MI: %w", err)
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
				err = fmt.Errorf("error at scan mi: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in MI: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in MI: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in MI: %w", updateTimeStr, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (m *miRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := m.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from KMEMO %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(kyouHistories) == 0 {
		return nil, nil
	}

	return kyouHistories[0], nil
}

func (m *miRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	repName, err := m.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MI: %w", err)
		return nil, err
	}

	// startのみ
	sql := `
SELECT 
  IS_DELETED,
  ID,
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
  'mi_create' AS DATA_TYPE
FROM MI 
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`
	log.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, repName, id)
	if err != nil {
		err = fmt.Errorf("error at select from MI %s: %w", id, err)
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
				err = fmt.Errorf("error at scan mi: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in MI: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in MI: %w", createTimeStr, id, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in MI: %w", updateTimeStr, id, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (m *miRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return filepath.Abs(m.filename)
}

func (m *miRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (m *miRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := m.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path mi rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (m *miRepositorySQLite3Impl) Close(ctx context.Context) error {
	return m.db.Close()
}

func (m *miRepositorySQLite3Impl) FindMi(ctx context.Context, query *find.FindQuery) ([]*Mi, error) {
	var err error
	if query.UpdateCache != nil && *query.UpdateCache {
		err = m.UpdateCache(ctx)
		if err != nil {
			repName, _ := m.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}

	sqlCreateMi := `
SELECT 
  IS_DELETED,
  ID,
  TITLE,
  IS_CHECKED,
  CHECKED_TIME,
  BOARD_NAME,
  LIMIT_TIME,
  ESTIMATE_END_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  'mi_create' AS DATA_TYPE
FROM MI 
`
	sqlWhereFilterCreateMi := "DATA_TYPE IN ('mi_create')"

	sqlWhereForCreate, err := m.whereSQLGenerator("CREATE_TIME", query)
	if err != nil {
		return nil, err
	}
	sql := fmt.Sprintf("SELECT * FROM (SELECT * FROM (%s) WHERE %s) WHERE %s", sqlCreateMi, sqlWhereForCreate, sqlWhereFilterCreateMi)

	log.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := m.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MI: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName)
	if err != nil {
		err = fmt.Errorf("error at select from MI %s: %w", err)
		return nil, err
	}
	defer rows.Close()

	mis := []*Mi{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			mi := &Mi{}
			mi.RepName = repName
			createTimeStr, updateTimeStr := "", ""
			checkedTime, limitTime, estimateStartTime, estimateEndTime := sqllib.NullString{}, sqllib.NullString{}, sqllib.NullString{}, sqllib.NullString{}

			err = rows.Scan(
				&mi.IsDeleted,
				&mi.ID,
				&mi.Title,
				&mi.IsChecked,
				&checkedTime,
				&mi.BoardName,
				&limitTime,
				&estimateStartTime,
				&estimateEndTime,
				&createTimeStr,
				&mi.CreateApp,
				&mi.CreateDevice,
				&mi.CreateUser,
				&updateTimeStr,
				&mi.UpdateApp,
				&mi.UpdateDevice,
				&mi.UpdateUser,
				&mi.RepName,
			)
			if err != nil {
				err = fmt.Errorf("error at scan mi: %w", err)
				return nil, err
			}

			mi.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in MI: %w", createTimeStr, err)
				return nil, err
			}
			mi.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in MI: %w", updateTimeStr, err)
				return nil, err
			}
			if checkedTime.Valid {
				parsedCheckedTime, _ := time.Parse(sqlite3impl.TimeLayout, checkedTime.String)
				mi.CheckedTime = &parsedCheckedTime
			}
			if limitTime.Valid {
				parsedLimitTime, _ := time.Parse(sqlite3impl.TimeLayout, limitTime.String)
				mi.LimitTime = &parsedLimitTime
			}
			if estimateStartTime.Valid {
				parsedEstimateStartTime, _ := time.Parse(sqlite3impl.TimeLayout, estimateStartTime.String)
				mi.EstimateStartTime = &parsedEstimateStartTime
			}
			if estimateEndTime.Valid {
				parsedEstimateEndTime, _ := time.Parse(sqlite3impl.TimeLayout, estimateEndTime.String)
				mi.EstimateEndTime = &parsedEstimateEndTime
			}
			mis = append(mis, mi)
		}
	}
	return mis, nil
}

func (m *miRepositorySQLite3Impl) GetMi(ctx context.Context, id string) (*Mi, error) {
	// 最新のデータを返す
	miHistories, err := m.GetMiHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get mi histories from MI %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(miHistories) == 0 {
		return nil, nil
	}

	return miHistories[0], nil
}

func (m *miRepositorySQLite3Impl) GetMiHistories(ctx context.Context, id string) ([]*Mi, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TITLE,
  IS_CHECKED,
  CHECKED_TIME,
  BOARD_NAME,
  LIMIT_TIME,
  ESTIMATE_START_TIME,
  ESTIMATE_END_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME
FROM MI
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`

	log.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get mi histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := m.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MI: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName, id)
	if err != nil {
		err = fmt.Errorf("error at select from MI %s: %w", err)
		return nil, err
	}
	defer rows.Close()

	mis := []*Mi{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			mi := &Mi{}
			mi.RepName = repName
			createTimeStr, updateTimeStr := "", ""
			checkedTime, limitTime, estimateStartTime, estimateEndTime := sqllib.NullString{}, sqllib.NullString{}, sqllib.NullString{}, sqllib.NullString{}

			err = rows.Scan(
				&mi.IsDeleted,
				&mi.ID,
				&mi.Title,
				&mi.IsChecked,
				&checkedTime,
				&mi.BoardName,
				&limitTime,
				&estimateStartTime,
				&estimateEndTime,
				&createTimeStr,
				&mi.CreateApp,
				&mi.CreateDevice,
				&mi.CreateUser,
				&updateTimeStr,
				&mi.UpdateApp,
				&mi.UpdateDevice,
				&mi.UpdateUser,
				&mi.RepName,
			)
			if err != nil {
				err = fmt.Errorf("error at scan mi: %w", err)
				return nil, err
			}

			mi.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in MI: %w", createTimeStr, err)
				return nil, err
			}
			mi.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in MI: %w", updateTimeStr, err)
				return nil, err
			}
			if checkedTime.Valid {
				parsedCheckedTime, _ := time.Parse(sqlite3impl.TimeLayout, checkedTime.String)
				mi.CheckedTime = &parsedCheckedTime
			}
			if limitTime.Valid {
				parsedLimitTime, _ := time.Parse(sqlite3impl.TimeLayout, limitTime.String)
				mi.LimitTime = &parsedLimitTime
			}
			if estimateStartTime.Valid {
				parsedEstimateStartTime, _ := time.Parse(sqlite3impl.TimeLayout, estimateStartTime.String)
				mi.EstimateStartTime = &parsedEstimateStartTime
			}
			if estimateEndTime.Valid {
				parsedEstimateEndTime, _ := time.Parse(sqlite3impl.TimeLayout, estimateEndTime.String)
				mi.EstimateEndTime = &parsedEstimateEndTime
			}
			mis = append(mis, mi)
		}
	}
	return mis, nil

}

func (m *miRepositorySQLite3Impl) AddMiInfo(ctx context.Context, mi *Mi) error {
	sql := `
INSERT INTO MI (
  IS_DELETED,
  ID,
  TITLE,
  IS_CHECKED,
  CHECKED_TIME,
  BOARD_NAME,
  LIMIT_TIME,
  ESTIMATE_START_TIME,
  ESTIMATE_END_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
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
  ?,
  ?,
  ?,
  ?,
  ?
)`
	log.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add mi sql %s: %w", mi.ID, err)
		return err
	}
	defer stmt.Close()

	var checkedTimeStr interface{}
	if mi.CheckedTime == nil {
		checkedTimeStr = nil
	} else {
		checkedTimeStr = mi.CheckedTime.Format(sqlite3impl.TimeLayout)
	}
	var limitTimeStr interface{}
	if mi.LimitTime == nil {
		limitTimeStr = nil
	} else {
		limitTimeStr = mi.LimitTime.Format(sqlite3impl.TimeLayout)
	}
	var startTimeStr interface{}
	if mi.EstimateStartTime == nil {
		startTimeStr = nil
	} else {
		startTimeStr = mi.EstimateStartTime.Format(sqlite3impl.TimeLayout)
	}
	var endTimeStr interface{}
	if mi.EstimateEndTime == nil {
		endTimeStr = nil
	} else {
		endTimeStr = mi.EstimateEndTime.Format(sqlite3impl.TimeLayout)
	}
	_, err = stmt.ExecContext(ctx,
		mi.IsDeleted,
		mi.ID,
		mi.Title,
		mi.IsChecked,
		checkedTimeStr,
		mi.BoardName,
		limitTimeStr,
		startTimeStr,
		endTimeStr,
		mi.CreateTime.Format(sqlite3impl.TimeLayout),
		mi.CreateApp,
		mi.CreateDevice,
		mi.CreateUser,
		mi.UpdateTime.Format(sqlite3impl.TimeLayout),
		mi.UpdateApp,
		mi.UpdateDevice,
		mi.UpdateUser,
	)
	if err != nil {
		err = fmt.Errorf("error at insert in to mi %s: %w", mi.ID, err)
		return err
	}
	return nil
}

func (m *miRepositorySQLite3Impl) whereSQLGenerator(timeColmunName string, query *find.FindQuery) (string, error) {
	whereCounter := 0
	sqlWhere := ""

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
			whereCounter++
		}
		sqlWhere += ")"
	}

	// 日付範囲指定ありの場合
	if query.UseCalendar != nil && *query.UseCalendar {
		// 開始日時を指定するSQLを追記
		columnName := timeColmunName
		if query.CalendarStartDate != nil {
			startDate := *query.CalendarStartDate
			if whereCounter != 0 {
				sqlWhere += " AND "
			}
			sqlWhere += sqlite3impl.EscapeSQLite(fmt.Sprintf("datetime("+columnName+", 'localtime') >= datetime('%s', 'localtime')", startDate.Format(sqlite3impl.TimeLayout)))
			whereCounter++
		}

		// 終了日時を指定するSQLを追記
		if query.UseCalendar != nil && *query.UseCalendar {
			endDate := *query.CalendarEndDate
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

		if whereCounter != 0 {
			sqlWhere += " AND "
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
	// UPDATE_TIMEが一番上のものだけを抽出
	sqlWhere += `
		GROUP BY ID
		HAVING MAX(datetime(UPDATE_TIME, 'localtime'))
		`
	return sqlWhere, nil
}

func (m *miRepositorySQLite3Impl) GetBoardNames(ctx context.Context) ([]string, error) {
	sql := `
SELECT 
DISTINCT BOARD_NAME
FROM MI 
WHERE IS_DELETED = FALSE
`
	log.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get board names sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at select board names from MI: %w", err)
		return nil, err
	}
	defer rows.Close()

	boardNames := []string{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			boardName := ""
			err = rows.Scan(&boardName)
			if err != nil {
				err = fmt.Errorf("error at scan rows at get board names in MI: %w", err)
				return nil, err
			}
			boardNames = append(boardNames, boardName)
		}
	}
	return boardNames, nil
}
