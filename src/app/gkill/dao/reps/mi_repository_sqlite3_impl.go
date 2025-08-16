package reps

import (
	"context"
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

type miRepositorySQLite3Impl struct {
	filename string
	db       *sqllib.DB
	m        *sync.Mutex
}

func NewMiRepositorySQLite3Impl(ctx context.Context, filename string) (MiRepository, error) {
	var err error
	db, err := sqllib.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=1&_journal=DELETE")
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
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create MI table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MI table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_MI ON MI (ID, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create MI index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MI index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
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
func (m *miRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
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

	repName, err := m.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MI: %w", err)
		return nil, err
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
		  'mi_create' AS DATA_TYPE
		FROM MI
		`

	sqlCheckMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  UPDATE_TIME AS RELATED_TIME,
		  CREATE_TIME,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  ? AS REP_NAME,
		  'mi_check' AS DATA_TYPE
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

	if query.IncludeCreateMi != nil && *query.IncludeCreateMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_create'"
	}
	if query.IncludeCheckMi != nil && *query.IncludeCheckMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_check'"
	}
	if query.IncludeLimitMi != nil && *query.IncludeLimitMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_limit'"
	}
	if query.IncludeStartMi != nil && *query.IncludeStartMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_start'"
	}
	if query.IncludeEndMi != nil && *query.IncludeEndMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_end'"
	}
	sqlWhereFilterEndMi += ")"
	if filterWhereCounter == 0 {
		sqlWhereFilterEndMi = " 1 = 0 " // false
	}

	queryArgsForCreate := []interface{}{
		repName,
	}
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "CREATE_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false

	findWordUseLike := true
	ignoreCase := true
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME IS NOT NULL AND " + sqlWhereForCreate
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, *query.MiBoardName)
	}

	queryArgsForCheck := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = true
	relatedTimeColumnName = "RELATED_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, *query.MiBoardName)
	}

	queryArgsForLimit := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = true
	relatedTimeColumnName = "LIMIT_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME IS NOT NULL AND " + sqlWhereForLimit
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, *query.MiBoardName)
	}

	queryArgsForStart := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = true
	relatedTimeColumnName = "ESTIMATE_START_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME IS NOT NULL AND " + sqlWhereForStart
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, *query.MiBoardName)
	}

	queryArgsForEnd := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = true
	relatedTimeColumnName = "ESTIMATE_END_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME IS NOT NULL AND " + sqlWhereForEnd
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, *query.MiBoardName)
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s AND %s", sqlCreateMi, sqlWhereForCreate, sqlCheckMi, sqlWhereForCheck, sqlLimitMi, sqlWhereForLimit, sqlStartMi, sqlWhereForStart, sqlEndMi, sqlWhereForEnd, sqlWhereFilterEndMi)

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find kyous sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from MI: %w", err)
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
			if _, exist := kyous[kyou.ID]; !exist {
				kyous[kyou.ID] = []*Kyou{}
			}
			kyous[kyou.ID] = append(kyous[kyou.ID], kyou)
		}
	}
	return kyous, nil
}

func (m *miRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := m.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from MI %s: %w", id, err)
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

func (m *miRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	var err error

	trueValue := true
	query := &find.FindQuery{
		UseIDs:          &trueValue,
		IDs:             &[]string{id},
		IncludeCreateMi: &trueValue,
		IncludeStartMi:  &trueValue,
		IncludeCheckMi:  &trueValue,
	}

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = m.UpdateCache(ctx)
		if err != nil {
			repName, _ := m.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

	repName, err := m.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MI: %w", err)
		return nil, err
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
		  'mi_create' AS DATA_TYPE
		FROM MI
		`

	sqlCheckMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  UPDATE_TIME AS RELATED_TIME,
		  CREATE_TIME,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  ? AS REP_NAME,
		  'mi_check' AS DATA_TYPE
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

	if query.IncludeCreateMi != nil && *query.IncludeCreateMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_create'"
	}
	if query.IncludeCheckMi != nil && *query.IncludeCheckMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_check'"
	}
	if query.IncludeLimitMi != nil && *query.IncludeLimitMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_limit'"
	}
	if query.IncludeStartMi != nil && *query.IncludeStartMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_start'"
	}
	if query.IncludeEndMi != nil && *query.IncludeEndMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_end'"
	}
	sqlWhereFilterEndMi += ")"
	if filterWhereCounter == 0 {
		sqlWhereFilterEndMi = " 1 = 0 " // false
	}

	queryArgsForCreate := []interface{}{
		repName,
	}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "CREATE_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false

	findWordUseLike := true
	ignoreCase := true
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME IS NOT NULL AND " + sqlWhereForCreate
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, *query.MiBoardName)
	}

	queryArgsForCheck := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "RELATED_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, *query.MiBoardName)
	}

	queryArgsForLimit := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "LIMIT_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME IS NOT NULL AND " + sqlWhereForLimit
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, *query.MiBoardName)
	}

	queryArgsForStart := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_START_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME IS NOT NULL AND " + sqlWhereForStart
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, *query.MiBoardName)
	}

	queryArgsForEnd := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_END_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME IS NOT NULL AND " + sqlWhereForEnd
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, *query.MiBoardName)
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s AND %s", sqlCreateMi, sqlWhereForCreate, sqlCheckMi, sqlWhereForCheck, sqlLimitMi, sqlWhereForLimit, sqlStartMi, sqlWhereForStart, sqlEndMi, sqlWhereForEnd, sqlWhereFilterEndMi)

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find kyous sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

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

	repName, err := m.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MI: %w", err)
		return nil, err
	}

	sqlCreateMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME,
          ESTIMATE_START_TIME,
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

	sqlCheckMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME,
          ESTIMATE_START_TIME,
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
		  'mi_check' AS DATA_TYPE
		FROM MI
		`

	sqlLimitMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME,
          ESTIMATE_START_TIME,
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
		  'mi_limit' AS DATA_TYPE
		FROM MI
		`
	sqlStartMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME,
          ESTIMATE_START_TIME,
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
		  'mi_start' AS DATA_TYPE
		FROM MI
		`

	sqlEndMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME,
          ESTIMATE_START_TIME,
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
		  'mi_end' AS DATA_TYPE
		FROM MI
		`

	// 検索対象のデータ抽出用WHERE
	filterWhereCounter := 0
	sqlWhereFilterEndMi := ""
	sqlWhereFilterEndMi += "DATA_TYPE IN ("

	if query.IncludeCreateMi != nil && *query.IncludeCreateMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_create'"
	}
	if query.IncludeCheckMi != nil && *query.IncludeCheckMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_check'"
	}
	if query.IncludeLimitMi != nil && *query.IncludeLimitMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_limit'"
	}
	if query.IncludeStartMi != nil && *query.IncludeStartMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_start'"
	}
	if query.IncludeEndMi != nil && *query.IncludeEndMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_end'"
	}
	sqlWhereFilterEndMi += ")"
	if filterWhereCounter == 0 {
		sqlWhereFilterEndMi = " 1 = 0 " // false
	}

	queryArgsForCreate := []interface{}{
		repName,
	}
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "CREATE_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false

	findWordUseLike := true
	ignoreCase := true
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME IS NOT NULL AND " + sqlWhereForCreate
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, *query.MiBoardName)
	}

	queryArgsForCheck := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = true
	relatedTimeColumnName = "CREATE_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, *query.MiBoardName)
	}

	queryArgsForLimit := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = true
	relatedTimeColumnName = "LIMIT_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME IS NOT NULL AND " + sqlWhereForLimit
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, *query.MiBoardName)
	}

	queryArgsForStart := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = true
	relatedTimeColumnName = "ESTIMATE_START_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME IS NOT NULL AND " + sqlWhereForStart
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, *query.MiBoardName)
	}

	queryArgsForEnd := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = true
	relatedTimeColumnName = "ESTIMATE_END_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME IS NOT NULL AND " + sqlWhereForEnd
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, *query.MiBoardName)
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s AND %s", sqlCreateMi, sqlWhereForCreate, sqlCheckMi, sqlWhereForCheck, sqlLimitMi, sqlWhereForLimit, sqlStartMi, sqlWhereForStart, sqlEndMi, sqlWhereForEnd, sqlWhereFilterEndMi)

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find kyous sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from MI: %w", err)
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
			limitTime, estimateStartTime, estimateEndTime := sqllib.NullString{}, sqllib.NullString{}, sqllib.NullString{}

			err = rows.Scan(
				&mi.IsDeleted,
				&mi.ID,
				&mi.Title,
				&mi.IsChecked,
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
				&mi.DataType,
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

func (m *miRepositorySQLite3Impl) GetMi(ctx context.Context, id string, updateTime *time.Time) (*Mi, error) {
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

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range miHistories {
			if kyou.UpdateTime.Format(sqlite3impl.TimeLayout) == updateTime.Format(sqlite3impl.TimeLayout) {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return miHistories[0], nil
}

func (m *miRepositorySQLite3Impl) GetMiHistories(ctx context.Context, id string) ([]*Mi, error) {
	var err error

	trueValue := true
	query := &find.FindQuery{
		UseIDs:          &trueValue,
		IDs:             &[]string{id},
		IncludeCreateMi: &trueValue,
		IncludeStartMi:  &trueValue,
		IncludeCheckMi:  &trueValue,
	}

	repName, err := m.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MI: %w", err)
		return nil, err
	}

	sqlCreateMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME,
          ESTIMATE_START_TIME,
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

	sqlCheckMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME,
          ESTIMATE_START_TIME,
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
		  'mi_check' AS DATA_TYPE
		FROM MI
		`

	sqlLimitMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME,
          ESTIMATE_START_TIME,
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
		  'mi_limit' AS DATA_TYPE
		FROM MI
		`
	sqlStartMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME,
          ESTIMATE_START_TIME,
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
		  'mi_start' AS DATA_TYPE
		FROM MI
		`

	sqlEndMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME,
          ESTIMATE_START_TIME,
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
		  'mi_end' AS DATA_TYPE
		FROM MI
		`

	// 検索対象のデータ抽出用WHERE
	filterWhereCounter := 0
	sqlWhereFilterEndMi := ""
	sqlWhereFilterEndMi += "DATA_TYPE IN ("

	if query.IncludeCreateMi != nil && *query.IncludeCreateMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_create'"
	}
	if query.IncludeCheckMi != nil && *query.IncludeCheckMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_check'"
	}
	if query.IncludeLimitMi != nil && *query.IncludeLimitMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_limit'"
	}
	if query.IncludeStartMi != nil && *query.IncludeStartMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_start'"
	}
	if query.IncludeEndMi != nil && *query.IncludeEndMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_end'"
	}
	sqlWhereFilterEndMi += ")"
	if filterWhereCounter == 0 {
		sqlWhereFilterEndMi = " 1 = 0 " // false
	}

	queryArgsForCreate := []interface{}{
		repName,
	}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "CREATE_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false

	findWordUseLike := true
	ignoreCase := true
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME IS NOT NULL AND " + sqlWhereForCreate
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, *query.MiBoardName)
	}

	queryArgsForCheck := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "CREATE_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, *query.MiBoardName)
	}

	queryArgsForLimit := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "LIMIT_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME IS NOT NULL AND " + sqlWhereForLimit
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, *query.MiBoardName)
	}

	queryArgsForStart := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_START_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME IS NOT NULL AND " + sqlWhereForStart
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, *query.MiBoardName)
	}

	queryArgsForEnd := []interface{}{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_END_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME IS NOT NULL AND " + sqlWhereForEnd
	if query.UseMiBoardName != nil && query.MiBoardName != nil && *query.UseMiBoardName {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, *query.MiBoardName)
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s AND %s", sqlCreateMi, sqlWhereForCreate, sqlCheckMi, sqlWhereForCheck, sqlLimitMi, sqlWhereForLimit, sqlStartMi, sqlWhereForStart, sqlEndMi, sqlWhereForEnd, sqlWhereFilterEndMi)

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find kyous sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from MI: %w", err)
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
			limitTime, estimateStartTime, estimateEndTime := sqllib.NullString{}, sqllib.NullString{}, sqllib.NullString{}

			err = rows.Scan(
				&mi.IsDeleted,
				&mi.ID,
				&mi.Title,
				&mi.IsChecked,
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
				&mi.DataType,
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
  ?
)`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add mi sql %s: %w", mi.ID, err)
		return err
	}
	defer stmt.Close()

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

	queryArgs := []interface{}{
		mi.IsDeleted,
		mi.ID,
		mi.Title,
		mi.IsChecked,
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
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to mi %s: %w", mi.ID, err)
		return err
	}
	return nil
}

func (m *miRepositorySQLite3Impl) GetBoardNames(ctx context.Context) ([]string, error) {
	var err error

	sql := `
SELECT 
  DISTINCT BOARD_NAME
FROM MI
WHERE
`
	query := &find.FindQuery{}
	queryArgs := []interface{}{}

	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "UPDATE_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false

	findWordUseLike := true
	ignoreCase := true
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql = fmt.Sprintf("%s %s", sql, sqlWhereForCreate)
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get board names sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
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
