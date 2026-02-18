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

type miRepositoryCachedSQLite3Impl struct {
	dbName   string
	miRep    MiRepository
	cachedDB *sqllib.DB
	m        *sync.RWMutex
}

func NewMiRepositoryCachedSQLite3Impl(ctx context.Context, miRep MiRepository, cacheDB *sqllib.DB, m *sync.RWMutex, dbName string) (MiRepository, error) {
	if m == nil {
		m = &sync.RWMutex{}
	}
	var err error

	sql := `
CREATE TABLE IF NOT EXISTS "` + dbName + `" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TITLE NOT NULL,
  IS_CHECKED NOT NULL,
  BOARD_NAME NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  REP_NAME NOT NULL,
  LIMIT_TIME_UNIX,
  ESTIMATE_START_TIME_UNIX,
  ESTIMATE_END_TIME_UNIX,
  CREATE_TIME_UNIX NOT NULL,
  UPDATE_TIME_UNIX NOT NULL
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create MI table statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create MI table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `_UNIX" ON "` + dbName + `"(ID, UPDATE_TIME_UNIX);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create mi index unix statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create mi index unix to %s: %w", dbName, err)
		return nil, err
	}

	return &miRepositoryCachedSQLite3Impl{
		dbName:   dbName,
		miRep:    miRep,
		cachedDB: cacheDB,
		m:        m,
	}, nil
}
func (m *miRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	var err error
	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = m.UpdateCache(ctx)
		if err != nil {
			repName, _ := m.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	m.m.RLock()
	defer m.m.RUnlock()

	sqlCreateMi := `
		SELECT
		  IS_DELETED,
		  ID,
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
		  'mi_create' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlCheckMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  UPDATE_TIME_UNIX AS RELATED_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_check' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlLimitMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  LIMIT_TIME_UNIX AS RELATED_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_limit' AS DATA_TYPE
		FROM ` + m.dbName + `
		`
	sqlStartMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  ESTIMATE_START_TIME_UNIX AS RELATED_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_start' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlEndMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  ESTIMATE_END_TIME_UNIX AS RELATED_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_end' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	// 検索対象のデータ抽出用WHERE
	filterWhereCounter := 0
	sqlWhereFilterEndMi := ""
	sqlWhereFilterEndMi += "DATA_TYPE IN ("

	if query.IncludeCreateMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_create'"
	}
	if query.IncludeCheckMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_check'"
	}
	if query.IncludeLimitMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_limit'"
	}
	if query.IncludeStartMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_start'"
	}
	if query.IncludeEndMi {
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

	tableName := m.dbName
	tableNameAlias := m.dbName
	queryArgsForCreate := []interface{}{}
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "CREATE_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false

	onlyLatestData = query.OnlyLatestData
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME_UNIX IS NOT NULL AND " + sqlWhereForCreate
	if query.UseMiBoardName {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForCheck := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "RELATED_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.UseMiBoardName {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForLimit := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "LIMIT_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME_UNIX IS NOT NULL AND " + sqlWhereForLimit
	if query.UseMiBoardName {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForStart := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "ESTIMATE_START_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME_UNIX IS NOT NULL AND " + sqlWhereForStart
	if query.UseMiBoardName {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForEnd := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "ESTIMATE_END_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME_UNIX IS NOT NULL AND " + sqlWhereForEnd
	if query.UseMiBoardName {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, query.MiBoardName)
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s AND %s", sqlCreateMi, sqlWhereForCreate, sqlCheckMi, sqlWhereForCheck, sqlLimitMi, sqlWhereForLimit, sqlStartMi, sqlWhereForStart, sqlEndMi, sqlWhereForEnd, sqlWhereFilterEndMi)

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := m.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find kyous sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from MI: %w", err)
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
				err = fmt.Errorf("error at scan mi: %w", err)
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

func (m *miRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var err error

	query := &find.FindQuery{
		UseIDs:          true,
		IDs:             []string{id},
		IncludeCreateMi: true,
		IncludeStartMi:  true,
		IncludeCheckMi:  true,
		OnlyLatestData:  updateTime == nil,
		UseUpdateTime:   updateTime != nil,
		UpdateTime:      updateTime,
	}

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = m.UpdateCache(ctx)
		if err != nil {
			repName, _ := m.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	m.m.RLock()
	defer m.m.RUnlock()

	sqlCreateMi := `
		SELECT
		  IS_DELETED,
		  ID,
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
		  'mi_create' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlCheckMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  UPDATE_TIME_UNIX AS RELATED_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_check' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlLimitMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  LIMIT_TIME_UNIX AS RELATED_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_limit' AS DATA_TYPE
		FROM ` + m.dbName + `
		`
	sqlStartMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  ESTIMATE_START_TIME_UNIX AS RELATED_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_start' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlEndMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  ESTIMATE_END_TIME_UNIX AS RELATED_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_end' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	// 検索対象のデータ抽出用WHERE
	filterWhereCounter := 0
	sqlWhereFilterEndMi := ""
	sqlWhereFilterEndMi += "DATA_TYPE IN ("

	if query.IncludeCreateMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_create'"
	}
	if query.IncludeCheckMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_check'"
	}
	if query.IncludeLimitMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_limit'"
	}
	if query.IncludeStartMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_start'"
	}
	if query.IncludeEndMi {
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

	tableName := m.dbName
	tableNameAlias := m.dbName
	queryArgsForCreate := []interface{}{}
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "CREATE_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false

	onlyLatestData = query.OnlyLatestData
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME_UNIX IS NOT NULL AND " + sqlWhereForCreate
	if query.UseMiBoardName {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForCheck := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "RELATED_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.UseMiBoardName {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForLimit := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "LIMIT_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME_UNIX IS NOT NULL AND " + sqlWhereForLimit
	if query.UseMiBoardName {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForStart := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "ESTIMATE_START_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME_UNIX IS NOT NULL AND " + sqlWhereForStart
	if query.UseMiBoardName {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForEnd := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "ESTIMATE_END_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME_UNIX IS NOT NULL AND " + sqlWhereForEnd
	if query.UseMiBoardName {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, query.MiBoardName)
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s AND %s", sqlCreateMi, sqlWhereForCreate, sqlCheckMi, sqlWhereForCheck, sqlLimitMi, sqlWhereForLimit, sqlStartMi, sqlWhereForStart, sqlEndMi, sqlWhereForEnd, sqlWhereFilterEndMi)

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := m.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find kyous sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from MI: %w", err)
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
				err = fmt.Errorf("error at scan mi: %w", err)
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

func (m *miRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	var err error

	query := &find.FindQuery{
		UseIDs:          true,
		IDs:             []string{id},
		IncludeCreateMi: true,
		IncludeStartMi:  true,
		IncludeCheckMi:  true,
	}

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = m.UpdateCache(ctx)
		if err != nil {
			repName, _ := m.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	m.m.RLock()
	defer m.m.RUnlock()

	sqlCreateMi := `
		SELECT
		  IS_DELETED,
		  ID,
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
		  'mi_create' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlCheckMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  UPDATE_TIME_UNIX AS RELATED_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_check' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlLimitMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  LIMIT_TIME_UNIX AS RELATED_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_limit' AS DATA_TYPE
		FROM ` + m.dbName + `
		`
	sqlStartMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  ESTIMATE_START_TIME_UNIX AS RELATED_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_start' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlEndMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  ESTIMATE_END_TIME_UNIX AS RELATED_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_end' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	// 検索対象のデータ抽出用WHERE
	filterWhereCounter := 0
	sqlWhereFilterEndMi := ""
	sqlWhereFilterEndMi += "DATA_TYPE IN ("

	if query.IncludeCreateMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_create'"
	}
	if query.IncludeCheckMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_check'"
	}
	if query.IncludeLimitMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_limit'"
	}
	if query.IncludeStartMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_start'"
	}
	if query.IncludeEndMi {
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

	tableName := m.dbName
	tableNameAlias := m.dbName
	queryArgsForCreate := []interface{}{}
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "CREATE_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false

	onlyLatestData = query.OnlyLatestData
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME_UNIX IS NOT NULL AND " + sqlWhereForCreate
	if query.UseMiBoardName {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForCheck := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "RELATED_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.UseMiBoardName {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForLimit := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "LIMIT_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME_UNIX IS NOT NULL AND " + sqlWhereForLimit
	if query.UseMiBoardName {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForStart := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "ESTIMATE_START_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME_UNIX IS NOT NULL AND " + sqlWhereForStart
	if query.UseMiBoardName {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForEnd := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "ESTIMATE_END_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME_UNIX IS NOT NULL AND " + sqlWhereForEnd
	if query.UseMiBoardName {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, query.MiBoardName)
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s AND %s", sqlCreateMi, sqlWhereForCreate, sqlCheckMi, sqlWhereForCheck, sqlLimitMi, sqlWhereForLimit, sqlStartMi, sqlWhereForStart, sqlEndMi, sqlWhereForEnd, sqlWhereFilterEndMi)

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := m.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find kyous sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from MI: %w", err)
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
				err = fmt.Errorf("error at scan mi: %w", err)
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

func (m *miRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return m.miRep.GetPath(ctx, id)
}

func (m *miRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {

	query := &find.FindQuery{
		UpdateCache:    true,
		OnlyLatestData: false,
	}

	allMis, err := m.miRep.FindMi(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all mis at update cache: %w", err)
		return err
	}

	m.m.Lock()
	defer m.m.Unlock()

	tx, err := m.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add mi: %w", err)
		return err
	}

	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at update cache: %w", "error", err)
			}
		}
	}()

	sql := `DELETE FROM ` + m.dbName
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create MI table statement %s: %w", "memory", err)
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
		err = fmt.Errorf("error at delete MI table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + m.dbName + ` (
  IS_DELETED,
  ID,
  TITLE,
  IS_CHECKED,
  BOARD_NAME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  LIMIT_TIME_UNIX,
  ESTIMATE_START_TIME_UNIX,
  ESTIMATE_END_TIME_UNIX,
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
  ?,
  ?,
  ?,
  ?
)`

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add mi sql: %w", err)
		return err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for _, mi := range allMis {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			var limitTimeUnix interface{}
			if mi.LimitTime == nil {
				limitTimeUnix = nil
			} else {
				limitTimeUnix = mi.LimitTime.Unix()
			}
			var startTimeUnix interface{}
			if mi.EstimateStartTime == nil {
				startTimeUnix = nil
			} else {
				startTimeUnix = mi.EstimateStartTime.Unix()
			}
			var endTimeUnix interface{}
			if mi.EstimateEndTime == nil {
				endTimeUnix = nil
			} else {
				endTimeUnix = mi.EstimateEndTime.Unix()
			}

			queryArgs := []interface{}{
				mi.IsDeleted,
				mi.ID,
				mi.Title,
				mi.IsChecked,
				mi.BoardName,
				mi.CreateApp,
				mi.CreateDevice,
				mi.CreateUser,
				mi.UpdateApp,
				mi.UpdateDevice,
				mi.UpdateUser,
				mi.RepName,
				limitTimeUnix,
				startTimeUnix,
				endTimeUnix,
				mi.CreateTime.Unix(),
				mi.UpdateTime.Unix(),
			}
			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to mi %s: %w", mi.ID, err)
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit transaction for add timeiss: %w", err)
		return err
	}
	isCommitted = true
	return nil
}

func (m *miRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return m.miRep.GetRepName(ctx)
}

func (m *miRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	m.m.Lock()
	defer m.m.Unlock()
	err := m.miRep.Close(ctx)
	if err != nil {
		return err
	}
	if gkill_options.CacheMiReps == nil || !*gkill_options.CacheMiReps {
		err = m.cachedDB.Close()
		if err != nil {
			return err
		}
	} else {
		_, err = m.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+m.dbName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *miRepositoryCachedSQLite3Impl) FindMi(ctx context.Context, query *find.FindQuery) ([]Mi, error) {
	var err error
	if query.UpdateCache {
		err = m.UpdateCache(ctx)
		if err != nil {
			repName, _ := m.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	m.m.RLock()
	defer m.m.RUnlock()

	sqlCreateMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_create' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlCheckMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_check' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlLimitMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_limit' AS DATA_TYPE
		FROM ` + m.dbName + `
		`
	sqlStartMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_start' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlEndMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_end' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	// 検索対象のデータ抽出用WHERE
	filterWhereCounter := 0
	sqlWhereFilterEndMi := ""
	sqlWhereFilterEndMi += "DATA_TYPE IN ("

	if query.IncludeCreateMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_create'"
	}
	if query.IncludeCheckMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_check'"
	}
	if query.IncludeLimitMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_limit'"
	}
	if query.IncludeStartMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_start'"
	}
	if query.IncludeEndMi {
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

	tableName := m.dbName
	tableNameAlias := m.dbName
	queryArgsForCreate := []interface{}{}
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "CREATE_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false

	onlyLatestData = query.OnlyLatestData
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME_UNIX IS NOT NULL AND " + sqlWhereForCreate
	if query.UseMiBoardName {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForCheck := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "CREATE_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.UseMiBoardName {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForLimit := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "LIMIT_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME_UNIX IS NOT NULL AND " + sqlWhereForLimit
	if query.UseMiBoardName {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForStart := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "ESTIMATE_START_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME_UNIX IS NOT NULL AND " + sqlWhereForStart
	if query.UseMiBoardName {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForEnd := []interface{}{}
	whereCounter = 0
	relatedTimeColumnName = "ESTIMATE_END_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME_UNIX IS NOT NULL AND " + sqlWhereForEnd
	if query.UseMiBoardName {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, query.MiBoardName)
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s AND %s", sqlCreateMi, sqlWhereForCreate, sqlCheckMi, sqlWhereForCheck, sqlLimitMi, sqlWhereForLimit, sqlStartMi, sqlWhereForStart, sqlEndMi, sqlWhereForEnd, sqlWhereFilterEndMi)

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := m.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find kyous sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from MI: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	mis := []Mi{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			mi := Mi{}
			createTimeUnix, updateTimeUnix := int64(0), int64(0)
			limitTime, estimateStartTime, estimateEndTime := sqllib.NullInt64{}, sqllib.NullInt64{}, sqllib.NullInt64{}

			err = rows.Scan(
				&mi.IsDeleted,
				&mi.ID,
				&mi.Title,
				&mi.IsChecked,
				&mi.BoardName,
				&limitTime,
				&estimateStartTime,
				&estimateEndTime,
				&createTimeUnix,
				&mi.CreateApp,
				&mi.CreateDevice,
				&mi.CreateUser,
				&updateTimeUnix,
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

			mi.CreateTime = time.Unix(createTimeUnix, 0).Local()
			mi.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			if limitTime.Valid {
				parsedLimitTime := time.Unix(limitTime.Int64, 0)
				mi.LimitTime = &parsedLimitTime
			}
			if estimateStartTime.Valid {
				parsedEstimateStartTime := time.Unix(estimateStartTime.Int64, 0).Local()
				mi.EstimateStartTime = &parsedEstimateStartTime
			}
			if estimateEndTime.Valid {
				parsedEstimateEndTime := time.Unix(estimateEndTime.Int64, 0).Local()
				mi.EstimateEndTime = &parsedEstimateEndTime
			}
			mis = append(mis, mi)
		}
	}
	return mis, nil
}

func (m *miRepositoryCachedSQLite3Impl) GetMi(ctx context.Context, id string, updateTime *time.Time) (*Mi, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	var err error

	query := &find.FindQuery{
		UseIDs:          true,
		IDs:             []string{id},
		IncludeCreateMi: true,
		IncludeStartMi:  true,
		IncludeCheckMi:  true,
		OnlyLatestData:  updateTime == nil,
		UseUpdateTime:   updateTime != nil,
		UpdateTime:      updateTime,
	}

	sqlCreateMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_create' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlCheckMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_check' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlLimitMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_limit' AS DATA_TYPE
		FROM ` + m.dbName + `
		`
	sqlStartMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_start' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlEndMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_end' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	// 検索対象のデータ抽出用WHERE
	filterWhereCounter := 0
	sqlWhereFilterEndMi := ""
	sqlWhereFilterEndMi += "DATA_TYPE IN ("

	if query.IncludeCreateMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_create'"
	}
	if query.IncludeCheckMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_check'"
	}
	if query.IncludeLimitMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_limit'"
	}
	if query.IncludeStartMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_start'"
	}
	if query.IncludeEndMi {
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

	tableName := m.dbName
	tableNameAlias := m.dbName
	queryArgsForCreate := []interface{}{}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "CREATE_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME_UNIX IS NOT NULL AND " + sqlWhereForCreate
	if query.UseMiBoardName {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForCheck := []interface{}{}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "CREATE_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.UseMiBoardName {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForLimit := []interface{}{}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "LIMIT_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME_UNIX IS NOT NULL AND " + sqlWhereForLimit
	if query.UseMiBoardName {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForStart := []interface{}{}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_START_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME_UNIX IS NOT NULL AND " + sqlWhereForStart
	if query.UseMiBoardName {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForEnd := []interface{}{}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_END_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME_UNIX IS NOT NULL AND " + sqlWhereForEnd
	if query.UseMiBoardName {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, query.MiBoardName)
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s AND %s", sqlCreateMi, sqlWhereForCreate, sqlCheckMi, sqlWhereForCheck, sqlLimitMi, sqlWhereForLimit, sqlStartMi, sqlWhereForStart, sqlEndMi, sqlWhereForEnd, sqlWhereFilterEndMi)

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := m.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find kyous sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from MI: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	mis := []Mi{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			mi := Mi{}
			createTimeUnix, updateTimeUnix := int64(0), int64(0)
			limitTime, estimateStartTime, estimateEndTime := sqllib.NullInt64{}, sqllib.NullInt64{}, sqllib.NullInt64{}

			err = rows.Scan(
				&mi.IsDeleted,
				&mi.ID,
				&mi.Title,
				&mi.IsChecked,
				&mi.BoardName,
				&limitTime,
				&estimateStartTime,
				&estimateEndTime,
				&createTimeUnix,
				&mi.CreateApp,
				&mi.CreateDevice,
				&mi.CreateUser,
				&updateTimeUnix,
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

			mi.CreateTime = time.Unix(createTimeUnix, 0).Local()
			mi.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			if limitTime.Valid {
				parsedLimitTime := time.Unix(limitTime.Int64, 0)
				mi.LimitTime = &parsedLimitTime
			}
			if estimateStartTime.Valid {
				parsedEstimateStartTime := time.Unix(estimateStartTime.Int64, 0).Local()
				mi.EstimateStartTime = &parsedEstimateStartTime
			}
			if estimateEndTime.Valid {
				parsedEstimateEndTime := time.Unix(estimateEndTime.Int64, 0).Local()
				mi.EstimateEndTime = &parsedEstimateEndTime
			}
			mis = append(mis, mi)
		}
	}
	if len(mis) == 0 {
		return nil, nil
	}
	return &mis[0], nil

}

func (m *miRepositoryCachedSQLite3Impl) GetMiHistories(ctx context.Context, id string) ([]Mi, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	var err error

	query := &find.FindQuery{
		UseIDs:          true,
		IDs:             []string{id},
		IncludeCreateMi: true,
		IncludeStartMi:  true,
		IncludeCheckMi:  true,
		OnlyLatestData:  false,
	}

	sqlCreateMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_create' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlCheckMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_check' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlLimitMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_limit' AS DATA_TYPE
		FROM ` + m.dbName + `
		`
	sqlStartMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_start' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	sqlEndMi := `
		SELECT
		  IS_DELETED,
		  ID,
		  TITLE,
  	      IS_CHECKED,
          BOARD_NAME,
          LIMIT_TIME_UNIX,
          ESTIMATE_START_TIME_UNIX,
          ESTIMATE_END_TIME_UNIX,
		  CREATE_TIME_UNIX,
		  CREATE_APP,
		  CREATE_DEVICE,
		  CREATE_USER,
		  UPDATE_TIME_UNIX,
		  UPDATE_APP,
		  UPDATE_DEVICE,
		  UPDATE_USER,
		  REP_NAME,
		  'mi_end' AS DATA_TYPE
		FROM ` + m.dbName + `
		`

	// 検索対象のデータ抽出用WHERE
	filterWhereCounter := 0
	sqlWhereFilterEndMi := ""
	sqlWhereFilterEndMi += "DATA_TYPE IN ("

	if query.IncludeCreateMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_create'"
	}
	if query.IncludeCheckMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_check'"
	}
	if query.IncludeLimitMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_limit'"
	}
	if query.IncludeStartMi {
		if filterWhereCounter != 0 {
			sqlWhereFilterEndMi += ", "
		}
		filterWhereCounter++
		sqlWhereFilterEndMi += "'mi_start'"
	}
	if query.IncludeEndMi {
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

	tableName := m.dbName
	tableNameAlias := m.dbName
	queryArgsForCreate := []interface{}{}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "CREATE_TIME_UNIX"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME_UNIX IS NOT NULL AND " + sqlWhereForCreate
	if query.UseMiBoardName {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForCheck := []interface{}{}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "CREATE_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.UseMiBoardName {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForLimit := []interface{}{}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "LIMIT_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME_UNIX IS NOT NULL AND " + sqlWhereForLimit
	if query.UseMiBoardName {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForStart := []interface{}{}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_START_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME_UNIX IS NOT NULL AND " + sqlWhereForStart
	if query.UseMiBoardName {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, query.MiBoardName)
	}

	tableName = m.dbName
	tableNameAlias = m.dbName
	queryArgsForEnd := []interface{}{}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_END_TIME_UNIX"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	ignoreCase = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME_UNIX IS NOT NULL AND " + sqlWhereForEnd
	if query.UseMiBoardName {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, query.MiBoardName)
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s AND %s", sqlCreateMi, sqlWhereForCreate, sqlCheckMi, sqlWhereForCheck, sqlLimitMi, sqlWhereForLimit, sqlStartMi, sqlWhereForStart, sqlEndMi, sqlWhereForEnd, sqlWhereFilterEndMi)

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := m.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get find kyous sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from MI: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	mis := []Mi{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			mi := Mi{}
			createTimeUnix, updateTimeUnix := int64(0), int64(0)
			limitTime, estimateStartTime, estimateEndTime := sqllib.NullInt64{}, sqllib.NullInt64{}, sqllib.NullInt64{}

			err = rows.Scan(
				&mi.IsDeleted,
				&mi.ID,
				&mi.Title,
				&mi.IsChecked,
				&mi.BoardName,
				&limitTime,
				&estimateStartTime,
				&estimateEndTime,
				&createTimeUnix,
				&mi.CreateApp,
				&mi.CreateDevice,
				&mi.CreateUser,
				&updateTimeUnix,
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

			mi.CreateTime = time.Unix(createTimeUnix, 0).Local()
			mi.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			if limitTime.Valid {
				parsedLimitTime := time.Unix(limitTime.Int64, 0)
				mi.LimitTime = &parsedLimitTime
			}
			if estimateStartTime.Valid {
				parsedEstimateStartTime := time.Unix(estimateStartTime.Int64, 0).Local()
				mi.EstimateStartTime = &parsedEstimateStartTime
			}
			if estimateEndTime.Valid {
				parsedEstimateEndTime := time.Unix(estimateEndTime.Int64, 0).Local()
				mi.EstimateEndTime = &parsedEstimateEndTime
			}
			mis = append(mis, mi)
		}
	}
	return mis, nil

}

func (m *miRepositoryCachedSQLite3Impl) AddMiInfo(ctx context.Context, mi Mi) error {
	m.m.Lock()
	defer m.m.Unlock()
	sql := `
INSERT INTO ` + m.dbName + ` (
  IS_DELETED,
  ID,
  TITLE,
  IS_CHECKED,
  BOARD_NAME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  LIMIT_TIME_UNIX,
  ESTIMATE_START_TIME_UNIX,
  ESTIMATE_END_TIME_UNIX,
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
  ?,
  ?,
  ?,
  ?
)`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := m.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add mi sql %s: %w", mi.ID, err)
		return err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	var limitTimeUnix interface{}
	if mi.LimitTime == nil {
		limitTimeUnix = nil
	} else {
		limitTimeUnix = mi.LimitTime.Unix()
	}
	var startTimeUnix interface{}
	if mi.EstimateStartTime == nil {
		startTimeUnix = nil
	} else {
		startTimeUnix = mi.EstimateStartTime.Unix()
	}
	var endTimeUnix interface{}
	if mi.EstimateEndTime == nil {
		endTimeUnix = nil
	} else {
		endTimeUnix = mi.EstimateEndTime.Unix()
	}

	queryArgs := []interface{}{
		mi.IsDeleted,
		mi.ID,
		mi.Title,
		mi.IsChecked,
		mi.BoardName,
		mi.CreateApp,
		mi.CreateDevice,
		mi.CreateUser,
		mi.UpdateApp,
		mi.UpdateDevice,
		mi.UpdateUser,
		mi.RepName,
		limitTimeUnix,
		startTimeUnix,
		endTimeUnix,
		mi.CreateTime.Unix(),
		mi.UpdateTime.Unix(),
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to mi %s: %w", mi.ID, err)
		return err
	}
	return nil
}

func (m *miRepositoryCachedSQLite3Impl) GetBoardNames(ctx context.Context) ([]string, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	var err error

	sql := `
SELECT 
  DISTINCT BOARD_NAME
FROM ` + m.dbName + `
`
	tableName := m.dbName
	tableNameAlias := m.dbName
	sql += fmt.Sprintf(" WHERE UPDATE_TIME_UNIX = ( SELECT MAX(UPDATE_TIME_UNIX) FROM %s AS INNER_TABLE WHERE ID = %s.ID )", tableName, tableNameAlias)
	sql += " GROUP BY BOARD_NAME"

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := m.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get board names sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at select board names from MI: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

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

func (m *miRepositoryCachedSQLite3Impl) UnWrapTyped() ([]MiRepository, error) {
	return m.miRep.UnWrapTyped()
}

func (m *miRepositoryCachedSQLite3Impl) UnWrap() ([]Repository, error) {
	return m.miRep.UnWrap()
}
