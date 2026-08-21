package reps

import (
	"context"
	sqllib "database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	_ "modernc.org/sqlite"
)

const CURRENT_SCHEMA_VERSION_MI_REPOSITORY_SQLITE3IMPL_DAO = "1.0.0"

type miRepositorySQLite3Impl struct {
	// キャッシュrepのフルリビルドを、実DBファイルが変わったときだけに絞るための判定用。
	// temp repが構造体変換でこの型をコピーするため、必ずポインタで持つこと。
	cacheChange *dbFileChangeDetector

	filename    string
	db          *sqllib.DB
	m           *sync.RWMutex
	fullConnect bool
}

func NewMiRepositorySQLite3Impl(ctx context.Context, filename string, fullConnect bool) (MiRepository, error) {
	db, err := sqlite3impl.GetSQLiteDBConnection(ctx, filename)
	if err != nil {
		return nil, err
	}
	if isOld, oldVerDAO, err := checkAndResolveDataSchemaMiRepositorySQLite3Impl(ctx, db); err != nil {
		return nil, err
	} else if isOld {
		if oldVerDAO != nil {
			return oldVerDAO, nil
		} else {
			return nil, fmt.Errorf("error at load database schema %s", filename)
		}
	}
	if gkill_options.Optimize {
		err = sqlite3impl.DeleteAllIndex(db)
		if err != nil {
			err = fmt.Errorf("error at delete all index %w", err)
			return nil, err
		}
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
	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create MI table statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gkill_log.LogSQL(ctx, sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MI table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_MI ON MI (ID, UPDATE_TIME);`
	gkill_log.LogIndexSQL(ctx, indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create MI index statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := indexStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gkill_log.LogIndexSQL(ctx, indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MI index to %s: %w", filename, err)
		return nil, err
	}

	// 時刻範囲検索と並び替えのための式インデックス。
	// GenerateFindSQLCommon が unixepoch(列) で比較・整列するので、
	// 式そのものに索引を張らないと全走査になる。
	//
	// MIの検索は5射影のUNIONで、射影ごとに別の時刻列を
	// relatedTimeColumnName として使う。CREATE_TIME だけ張っていたので
	// 残り3本のUNIONが全走査になっていた。
	if err := sqlite3impl.EnsureUnixepochIndex(ctx, db, "MI", "CREATE_TIME", "LIMIT_TIME", "ESTIMATE_START_TIME", "ESTIMATE_END_TIME"); err != nil {
		return nil, err
	}

	// mi板の絞り込みで使う。
	// IS_CHECKED は NOT NULL 列に対する IS NOT NULL としてしか使われず
	// 常に真＝選択性ゼロなので、単独索引は張らない。
	boardNameIndexSQL := `CREATE INDEX IF NOT EXISTS INDEX_MI_BOARD_NAME ON MI (BOARD_NAME);`
	gkill_log.LogIndexSQL(ctx, boardNameIndexSQL)
	if _, err := db.ExecContext(ctx, boardNameIndexSQL); err != nil {
		err = fmt.Errorf("error at create MI board name index to %s: %w", filename, err)
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

	return &miRepositorySQLite3Impl{
		cacheChange: &dbFileChangeDetector{},
		filename:    filename,
		db:          db,
		m:           &sync.RWMutex{},
		fullConnect: fullConnect,
	}, nil
}

func (m *miRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	var err error
	var db *sqllib.DB
	if m.fullConnect {
		db = m.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, m.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
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

	tableName := "MI"
	tableNameAlias := "MI"
	queryArgsForCreate := []any{
		repName,
	}
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "CREATE_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME IS NOT NULL AND " + sqlWhereForCreate
	if query.MiBoardName != nil {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForCheck := []any{
		repName,
	}
	whereCounter = 0
	// 以前はcheck/limit/start/end分岐だけtrue固定で、create分岐・cached実装と非対称だった
	onlyLatestData = query.OnlyLatestData
	relatedTimeColumnName = "RELATED_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.MiBoardName != nil {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForLimit := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = query.OnlyLatestData
	relatedTimeColumnName = "LIMIT_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME IS NOT NULL AND " + sqlWhereForLimit
	if query.MiBoardName != nil {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForStart := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = query.OnlyLatestData
	relatedTimeColumnName = "ESTIMATE_START_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME IS NOT NULL AND " + sqlWhereForStart
	if query.MiBoardName != nil {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForEnd := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = query.OnlyLatestData
	relatedTimeColumnName = "ESTIMATE_END_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME IS NOT NULL AND " + sqlWhereForEnd
	if query.MiBoardName != nil {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, *query.MiBoardName)
	}

	// ここは UNION のままにすること。
	// 射影ごとに DATA_TYPE のリテラルが違うので「腕をまたぐ重複は無い」のは正しいが、
	// **腕の中で**同一射影が複数行になることが実際にある。キャッシュ表の最新版判定は
	// UPDATE_TIME_UNIX(秒)なので、同じ秒に複数版があると全部が最新版として当たるため。
	// UNION ALL にすると cached だけ行数が増え、mi_find_kyous_parity_test.go が落ちる。
	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s",
		sqlCreateMi, sqlWhereForCreate,
		sqlCheckMi, sqlWhereForCheck,
		sqlLimitMi, sqlWhereForLimit,
		sqlStartMi, sqlWhereForStart,
		sqlEndMi, sqlWhereForEnd)
	queryArgs := []any{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)

	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get mi sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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
			// 空スライスの事前確保はしない。存在しないキーへのappendはnilスライスに対して働くので
			// 結果は同じで、レコード1件につき1回の無駄な確保(実データで56万回)が消える。
			// 同じ整理は dao/reps/repositories.go の集約側では既に済んでいる。
			kyous[kyou.ID] = append(kyous[kyou.ID], kyou)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return kyous, nil
}

func (m *miRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var err error
	var db *sqllib.DB
	if m.fullConnect {
		db = m.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, m.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
	}

	query := &find.FindQuery{
		IDs:             []string{id},
		IncludeCreateMi: true,
		IncludeStartMi:  true,
		IncludeCheckMi:  true,
		OnlyLatestData:  updateTime == nil,
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

	tableName := "MI"
	tableNameAlias := "MI"
	queryArgsForCreate := []any{
		repName,
	}
	whereCounter := 0
	// GenerateFindSQLCommon は query.OnlyLatestData を読まず、この引数しか見ない。
	// false のままだと updateTime 未指定のときに **そのIDの全バージョンを無順序・無制限に読み**、
	// 下の kyous[0] が格納順の先頭(多くの場合いちばん古い版)を返してしまう。
	// 版の数だけ走査するので遅くもある。Tag / Text では既に同じ修正が入っている。
	onlyLatestData := updateTime == nil
	relatedTimeColumnName := "CREATE_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME IS NOT NULL AND " + sqlWhereForCreate
	if query.MiBoardName != nil {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForCheck := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "RELATED_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.MiBoardName != nil {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForLimit := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "LIMIT_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME IS NOT NULL AND " + sqlWhereForLimit
	if query.MiBoardName != nil {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForStart := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_START_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME IS NOT NULL AND " + sqlWhereForStart
	if query.MiBoardName != nil {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForEnd := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_END_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME IS NOT NULL AND " + sqlWhereForEnd
	if query.MiBoardName != nil {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, *query.MiBoardName)
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s",
		sqlCreateMi, sqlWhereForCreate,
		sqlCheckMi, sqlWhereForCheck,
		sqlLimitMi, sqlWhereForLimit,
		sqlStartMi, sqlWhereForStart,
		sqlEndMi, sqlWhereForEnd)
	queryArgs := []any{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)

	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(kyous) == 0 {
		return nil, nil
	}
	// 最新版に絞ってもrepをまたいだ同一版が複数返りうるので、UpdateTimeが最大のものを選ぶ。
	// 格納順の先頭を返すと、どれが返るかがSQLiteの都合で決まってしまう。
	latestKyou := slices.MaxFunc(kyous, func(a Kyou, b Kyou) int {
		return a.UpdateTime.Compare(b.UpdateTime)
	})
	return &latestKyou, nil
}

func (m *miRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	var err error
	var db *sqllib.DB
	if m.fullConnect {
		db = m.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, m.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
	}

	query := &find.FindQuery{
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

	tableName := "MI"
	tableNameAlias := "MI"
	queryArgsForCreate := []any{
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
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME IS NOT NULL AND " + sqlWhereForCreate
	if query.MiBoardName != nil {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForCheck := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "RELATED_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.MiBoardName != nil {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForLimit := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "LIMIT_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME IS NOT NULL AND " + sqlWhereForLimit
	if query.MiBoardName != nil {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForStart := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_START_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME IS NOT NULL AND " + sqlWhereForStart
	if query.MiBoardName != nil {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForEnd := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_END_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME IS NOT NULL AND " + sqlWhereForEnd
	if query.MiBoardName != nil {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, *query.MiBoardName)
	}

	sql := fmt.Sprintf("%s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s UNION %s WHERE %s",
		sqlCreateMi, sqlWhereForCreate,
		sqlCheckMi, sqlWhereForCheck,
		sqlLimitMi, sqlWhereForLimit,
		sqlStartMi, sqlWhereForStart,
		sqlEndMi, sqlWhereForEnd)
	queryArgs := []any{}
	queryArgs = append(queryArgs, queryArgsForCreate...)
	queryArgs = append(queryArgs, queryArgsForCheck...)
	queryArgs = append(queryArgs, queryArgsForLimit...)
	queryArgs = append(queryArgs, queryArgsForStart...)
	queryArgs = append(queryArgs, queryArgsForEnd...)

	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return kyous, nil
}

func (m *miRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if id == "" {
		return m.filename, nil
	}
	return filepath.Abs(m.filename)
}

func (m *miRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	// 自身は実DBを直接見るのでキャッシュは持たないが、
	// 上位のキャッシュrepが「作り直す必要があるか」を判断できるよう、
	// ここでファイルの更新有無だけ観測しておく。
	m.cacheChange.refresh(m.filename)
	return nil
}

func (m *miRepositorySQLite3Impl) LastUpdateCacheChanged() bool {
	return m.cacheChange.lastChanged()
}

// CommitCacheRebuild は上位のキャッシュrepが再構築に成功したときに呼ばれます。
func (m *miRepositorySQLite3Impl) CommitCacheRebuild() {
	m.cacheChange.commit()
}

func (m *miRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	m.m.RLock()
	defer m.m.RUnlock()
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
	m.m.Lock()
	defer m.m.Unlock()
	if m.fullConnect {
		return m.db.Close()
	}
	return nil
}

func (m *miRepositorySQLite3Impl) FindMi(ctx context.Context, query *find.FindQuery) ([]Mi, error) {
	var err error
	var db *sqllib.DB
	if m.fullConnect {
		db = m.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, m.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
	}

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

	tableName := "MI"
	tableNameAlias := "MI"
	queryArgsForCreate := []any{
		repName,
	}
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "CREATE_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME IS NOT NULL AND " + sqlWhereForCreate
	if query.MiBoardName != nil {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForCheck := []any{
		repName,
	}
	whereCounter = 0
	// 以前はcheck/limit/start/end分岐だけtrue固定で、create分岐・cached実装と非対称だった
	onlyLatestData = query.OnlyLatestData
	relatedTimeColumnName = "UPDATE_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.MiBoardName != nil {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForLimit := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = query.OnlyLatestData
	relatedTimeColumnName = "LIMIT_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME IS NOT NULL AND " + sqlWhereForLimit
	if query.MiBoardName != nil {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForStart := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = query.OnlyLatestData
	relatedTimeColumnName = "ESTIMATE_START_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME IS NOT NULL AND " + sqlWhereForStart
	if query.MiBoardName != nil {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForEnd := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = query.OnlyLatestData
	relatedTimeColumnName = "ESTIMATE_END_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME IS NOT NULL AND " + sqlWhereForEnd
	if query.MiBoardName != nil {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, *query.MiBoardName)
	}

	sqlSegments := []string{}
	queryArgs := []any{}
	if query.IncludeCreateMi {
		sqlSegments = append(sqlSegments, sqlCreateMi+" WHERE "+sqlWhereForCreate)
		queryArgs = append(queryArgs, queryArgsForCreate...)
	}
	if query.IncludeCheckMi {
		sqlSegments = append(sqlSegments, sqlCheckMi+" WHERE "+sqlWhereForCheck)
		queryArgs = append(queryArgs, queryArgsForCheck...)
	}
	if query.IncludeLimitMi {
		sqlSegments = append(sqlSegments, sqlLimitMi+" WHERE "+sqlWhereForLimit)
		queryArgs = append(queryArgs, queryArgsForLimit...)
	}
	if query.IncludeStartMi {
		sqlSegments = append(sqlSegments, sqlStartMi+" WHERE "+sqlWhereForStart)
		queryArgs = append(queryArgs, queryArgsForStart...)
	}
	if query.IncludeEndMi {
		sqlSegments = append(sqlSegments, sqlEndMi+" WHERE "+sqlWhereForEnd)
		queryArgs = append(queryArgs, queryArgsForEnd...)
	}
	if len(sqlSegments) == 0 {
		return []Mi{}, nil
	}
	// ここは UNION のままにすること。
	// 射影ごとに DATA_TYPE のリテラルが違うので「腕をまたぐ重複は無い」のは正しいが、
	// **腕の中で**同一射影が複数行になることが実際にある。キャッシュ表の最新版判定は
	// UPDATE_TIME_UNIX(秒)なので、同じ秒に複数版があると全部が最新版として当たるため。
	// UNION ALL にすると cached だけ行数が増え、mi_find_kyous_parity_test.go が落ちる。
	sql := strings.Join(sqlSegments, " UNION ")

	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find mi sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return mis, nil
}

func (m *miRepositorySQLite3Impl) GetMi(ctx context.Context, id string, updateTime *time.Time) (*Mi, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	var err error
	var db *sqllib.DB
	if m.fullConnect {
		db = m.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, m.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
	}

	query := &find.FindQuery{
		IDs:             []string{id},
		IncludeCreateMi: true,
		IncludeStartMi:  true,
		IncludeCheckMi:  true,
		OnlyLatestData:  updateTime == nil,
		UpdateTime:      updateTime,
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

	tableName := "MI"
	tableNameAlias := "MI"
	queryArgsForCreate := []any{
		repName,
	}
	whereCounter := 0
	// GenerateFindSQLCommon は query.OnlyLatestData を読まず、この引数しか見ない（既定 false のままだと最古版を返す）。
	onlyLatestData := query.OnlyLatestData
	relatedTimeColumnName := "CREATE_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME IS NOT NULL AND " + sqlWhereForCreate
	if query.MiBoardName != nil {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForCheck := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = query.OnlyLatestData
	relatedTimeColumnName = "CREATE_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.MiBoardName != nil {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForLimit := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = query.OnlyLatestData
	relatedTimeColumnName = "LIMIT_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME IS NOT NULL AND " + sqlWhereForLimit
	if query.MiBoardName != nil {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForStart := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = query.OnlyLatestData
	relatedTimeColumnName = "ESTIMATE_START_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME IS NOT NULL AND " + sqlWhereForStart
	if query.MiBoardName != nil {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForEnd := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = query.OnlyLatestData
	relatedTimeColumnName = "ESTIMATE_END_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME IS NOT NULL AND " + sqlWhereForEnd
	if query.MiBoardName != nil {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, *query.MiBoardName)
	}

	sqlSegments := []string{}
	queryArgs := []any{}
	if query.IncludeCreateMi {
		sqlSegments = append(sqlSegments, sqlCreateMi+" WHERE "+sqlWhereForCreate)
		queryArgs = append(queryArgs, queryArgsForCreate...)
	}
	if query.IncludeCheckMi {
		sqlSegments = append(sqlSegments, sqlCheckMi+" WHERE "+sqlWhereForCheck)
		queryArgs = append(queryArgs, queryArgsForCheck...)
	}
	if query.IncludeLimitMi {
		sqlSegments = append(sqlSegments, sqlLimitMi+" WHERE "+sqlWhereForLimit)
		queryArgs = append(queryArgs, queryArgsForLimit...)
	}
	if query.IncludeStartMi {
		sqlSegments = append(sqlSegments, sqlStartMi+" WHERE "+sqlWhereForStart)
		queryArgs = append(queryArgs, queryArgsForStart...)
	}
	if query.IncludeEndMi {
		sqlSegments = append(sqlSegments, sqlEndMi+" WHERE "+sqlWhereForEnd)
		queryArgs = append(queryArgs, queryArgsForEnd...)
	}
	if len(sqlSegments) == 0 {
		return nil, nil
	}
	sql := strings.Join(sqlSegments, " UNION ")

	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get mi sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(mis) == 0 {
		return nil, nil
	}
	// 最新版に絞ってもrepをまたいだ同一版が複数返りうるので、UpdateTimeが最大のものを選ぶ。
	// 格納順の先頭を返すと、どれが返るかがSQLiteの都合で決まってしまう。
	latestMi := slices.MaxFunc(mis, func(a Mi, b Mi) int {
		return a.UpdateTime.Compare(b.UpdateTime)
	})
	return &latestMi, nil

}

func (m *miRepositorySQLite3Impl) GetMiHistories(ctx context.Context, id string) ([]Mi, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	var err error
	var db *sqllib.DB
	if m.fullConnect {
		db = m.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, m.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
	}

	query := &find.FindQuery{
		IDs:             []string{id},
		IncludeCreateMi: true,
		IncludeStartMi:  true,
		IncludeCheckMi:  true,
		OnlyLatestData:  false,
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

	tableName := "MI"
	tableNameAlias := "MI"
	queryArgsForCreate := []any{
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
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCreate)
	if err != nil {
		return nil, err
	}
	sqlWhereForCreate = "CREATE_TIME IS NOT NULL AND " + sqlWhereForCreate
	if query.MiBoardName != nil {
		sqlWhereForCreate += " AND "
		sqlWhereForCreate += " BOARD_NAME = ? "
		queryArgsForCreate = append(queryArgsForCreate, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForCheck := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "CREATE_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForCheck, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForCheck)
	if err != nil {
		return nil, err
	}
	sqlWhereForCheck = " IS_CHECKED IS NOT NULL AND " + sqlWhereForCheck
	if query.MiBoardName != nil {
		sqlWhereForCheck += " AND "
		sqlWhereForCheck += " BOARD_NAME = ? "
		queryArgsForCheck = append(queryArgsForCheck, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForLimit := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "LIMIT_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForLimit, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForLimit)
	if err != nil {
		return nil, err
	}
	sqlWhereForLimit = "LIMIT_TIME IS NOT NULL AND " + sqlWhereForLimit
	if query.MiBoardName != nil {
		sqlWhereForLimit += " AND "
		sqlWhereForLimit += " BOARD_NAME = ? "
		queryArgsForLimit = append(queryArgsForLimit, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForStart := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_START_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	sqlWhereForStart = "ESTIMATE_START_TIME IS NOT NULL AND " + sqlWhereForStart
	if query.MiBoardName != nil {
		sqlWhereForStart += " AND "
		sqlWhereForStart += " BOARD_NAME = ? "
		queryArgsForStart = append(queryArgsForStart, *query.MiBoardName)
	}

	tableName = "MI"
	tableNameAlias = "MI"
	queryArgsForEnd := []any{
		repName,
	}
	whereCounter = 0
	onlyLatestData = false
	relatedTimeColumnName = "ESTIMATE_END_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	sqlWhereForEnd = "ESTIMATE_END_TIME IS NOT NULL AND " + sqlWhereForEnd
	if query.MiBoardName != nil {
		sqlWhereForEnd += " AND "
		sqlWhereForEnd += " BOARD_NAME = ? "
		queryArgsForEnd = append(queryArgsForEnd, *query.MiBoardName)
	}

	sqlSegments := []string{}
	queryArgs := []any{}
	if query.IncludeCreateMi {
		sqlSegments = append(sqlSegments, sqlCreateMi+" WHERE "+sqlWhereForCreate)
		queryArgs = append(queryArgs, queryArgsForCreate...)
	}
	if query.IncludeCheckMi {
		sqlSegments = append(sqlSegments, sqlCheckMi+" WHERE "+sqlWhereForCheck)
		queryArgs = append(queryArgs, queryArgsForCheck...)
	}
	if query.IncludeLimitMi {
		sqlSegments = append(sqlSegments, sqlLimitMi+" WHERE "+sqlWhereForLimit)
		queryArgs = append(queryArgs, queryArgsForLimit...)
	}
	if query.IncludeStartMi {
		sqlSegments = append(sqlSegments, sqlStartMi+" WHERE "+sqlWhereForStart)
		queryArgs = append(queryArgs, queryArgsForStart...)
	}
	if query.IncludeEndMi {
		sqlSegments = append(sqlSegments, sqlEndMi+" WHERE "+sqlWhereForEnd)
		queryArgs = append(queryArgs, queryArgsForEnd...)
	}
	if len(sqlSegments) == 0 {
		return []Mi{}, nil
	}
	sql := strings.Join(sqlSegments, " UNION ")

	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get mi histories sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return mis, nil

}

func (m *miRepositorySQLite3Impl) AddMiInfo(ctx context.Context, mi Mi) error {
	m.m.Lock()
	defer m.m.Unlock()
	var err error
	var db *sqllib.DB
	if m.fullConnect {
		db = m.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, m.filename)
		if err != nil {
			return err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
	}

	if strings.TrimSpace(mi.Title) == "" {
		return fmt.Errorf("mi title must not be empty")
	}

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
	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
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

	var limitTimeStr any
	if mi.LimitTime == nil {
		limitTimeStr = nil
	} else {
		limitTimeStr = mi.LimitTime.Format(sqlite3impl.TimeLayout)
	}
	var startTimeStr any
	if mi.EstimateStartTime == nil {
		startTimeStr = nil
	} else {
		startTimeStr = mi.EstimateStartTime.Format(sqlite3impl.TimeLayout)
	}
	var endTimeStr any
	if mi.EstimateEndTime == nil {
		endTimeStr = nil
	} else {
		endTimeStr = mi.EstimateEndTime.Format(sqlite3impl.TimeLayout)
	}

	queryArgs := []any{
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
		mi.CreateUser,
		mi.CreateDevice,
		mi.UpdateTime.Format(sqlite3impl.TimeLayout),
		mi.UpdateApp,
		mi.UpdateDevice,
		mi.UpdateUser,
	}
	gkill_log.LogSQLParams(ctx, sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to mi %s: %w", mi.ID, err)
		return err
	}
	return nil
}

func (m *miRepositorySQLite3Impl) GetBoardNames(ctx context.Context) ([]string, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	var err error
	var db *sqllib.DB
	if m.fullConnect {
		db = m.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, m.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
	}

	// 板名は「最新版かつ未削除」のレコードからだけ集める。
	// 全履歴を見ると、板名を打ち間違えて直したあとも旧バージョンの名前が板一覧に出続ける。
	// whereCounterを1から始めているのは、下のIS_DELETED条件を自分で書いたぶんを数えているため
	// （0のままだとGenerateFindSQLCommonが区切りなしで " 0 = 0 " を続けて構文エラーになる）。
	sql := `
SELECT
  DISTINCT BOARD_NAME
FROM MI
WHERE
  IS_DELETED = 0
`
	query := &find.FindQuery{}

	tableName := "MI"
	tableNameAlias := "MI"
	queryArgs := []any{}
	whereCounter := 1
	onlyLatestData := true
	relatedTimeColumnName := "UPDATE_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	sqlWhereForCreate, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql = fmt.Sprintf("%s %s", sql, sqlWhereForCreate)
	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
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

	gkill_log.LogSQL(ctx, sql)
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return boardNames, nil
}

func (m *miRepositorySQLite3Impl) UnWrapTyped() ([]MiRepository, error) {
	return []MiRepository{m}, nil
}

func (m *miRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{m}, nil
}

func (m *miRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	var err error
	var db *sqllib.DB
	if m.fullConnect {
		db = m.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, m.filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := db.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
	}

	if updateCache {
		err = m.UpdateCache(ctx)
		if err != nil {
			repName, _ := m.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	m.m.RLock()
	defer m.m.RUnlock()

	repName, err := m.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	sql := `
SELECT IS_DELETED, ID AS TARGET_ID, NULL AS TARGET_ID_IN_DATA,
       ? AS LATEST_DATA_REPOSITORY_NAME, UPDATE_TIME AS DATA_UPDATE_TIME
FROM MI
`
	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, repName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	latestDataRepositoryAddressMap := map[string]gkill_cache.LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			addr := gkill_cache.LatestDataRepositoryAddress{}
			var dataUpdateTimeStr string
			var targetIDInData *string
			// IS_DELETEDはboolバインドでINTEGER(0/1)格納なので直接boolへScanする。
			// 以前は文字列に受けて "TRUE" と比較しており(実値は"0"/"1")、常にfalseになって
			// 削除済みターゲットを指すReKyou/MiReKyouが検索結果に残っていた
			err := rows.Scan(&addr.IsDeleted, &addr.TargetID, &targetIDInData, &addr.LatestDataRepositoryName, &dataUpdateTimeStr)
			if err != nil {
				return nil, err
			}
			addr.TargetIDInData = targetIDInData
			addr.DataUpdateTime, err = time.Parse(sqlite3impl.TimeLayout, dataUpdateTimeStr)
			if err != nil {
				return nil, err
			}
			if existing, exist := latestDataRepositoryAddressMap[addr.TargetID]; !exist || existing.DataUpdateTime.Before(addr.DataUpdateTime) {
				latestDataRepositoryAddressMap[addr.TargetID] = addr
			}
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	latestDataRepositoryAddresses := make([]gkill_cache.LatestDataRepositoryAddress, 0, len(latestDataRepositoryAddressMap))
	for _, addr := range latestDataRepositoryAddressMap {
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, addr)
	}
	return latestDataRepositoryAddresses, nil
}

func checkAndResolveDataSchemaMiRepositorySQLite3Impl(ctx context.Context, db *sqllib.DB) (isOld bool, oldVerDAO MiRepository, err error) {
	schemaVersionKey := "SCHEMA_VERSION_MI"
	currentSchemaVersion := CURRENT_SCHEMA_VERSION_MI_REPOSITORY_SQLITE3IMPL_DAO

	// テーブルとインデックスがなければ作る
	createTableSQL := `
CREATE TABLE IF NOT EXISTS GKILL_META_INFO (
  KEY NOT NULL,
  VALUE,
  PRIMARY KEY(KEY)
);`
	gkill_log.LogSQL(ctx, createTableSQL)
	stmt, err := db.PrepareContext(ctx, createTableSQL)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info table statement: %w", err)
		return false, nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gkill_log.LogSQL(ctx, createTableSQL)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info table: %w", err)
		return false, nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_GKILL_META_INFO ON GKILL_META_INFO (KEY);`
	gkill_log.LogIndexSQL(ctx, indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info index statement: %w", err)
		return false, nil, err
	}
	defer func() {
		err := indexStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gkill_log.LogIndexSQL(ctx, indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info index: %w", err)
		return false, nil, err
	}

	// スキーマのバージョンを取得する
	selectSchemaVersionSQL := `
SELECT 
  VALUE
FROM GKILL_META_INFO
WHERE KEY = ?
`
	gkill_log.LogSQL(ctx, selectSchemaVersionSQL)
	selectSchemaVersionStmt, err := db.PrepareContext(ctx, selectSchemaVersionSQL)
	if err != nil {
		err = fmt.Errorf("error at get schema version sql: %w", err)
		return false, nil, err
	}
	defer func() {
		err := selectSchemaVersionStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	dbSchemaVersion := ""
	queryArgs := []any{schemaVersionKey}
	gkill_log.LogSQLQuery(ctx, selectSchemaVersionSQL, queryArgs)
	err = selectSchemaVersionStmt.QueryRowContext(ctx, queryArgs...).Scan(&dbSchemaVersion)
	if err != nil {
		// データがなかったら今のバージョンをいれる
		if errors.Is(err, sqllib.ErrNoRows) {
			insertCurrentVersionSQL := `
INSERT INTO GKILL_META_INFO(KEY, VALUE)
VALUES(?, ?)`
			gkill_log.LogSQL(ctx, insertCurrentVersionSQL)
			insertCurrentVersionStmt, err := db.PrepareContext(ctx, insertCurrentVersionSQL)
			if err != nil {
				err = fmt.Errorf("error at insert schema version sql: %w", err)
				return false, nil, err
			}
			defer func() {
				err := insertCurrentVersionStmt.Close()
				if err != nil {
					slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
				}
			}()
			queryArgs := []any{schemaVersionKey, currentSchemaVersion}
			gkill_log.LogSQLQuery(ctx, insertCurrentVersionSQL, queryArgs)
			_, err = insertCurrentVersionStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert schema version: %w", err)
				return false, nil, err
			}

			queryArgs = []any{schemaVersionKey}
			gkill_log.LogSQLQuery(ctx, selectSchemaVersionSQL, queryArgs)
			err = selectSchemaVersionStmt.QueryRowContext(ctx, queryArgs...).Scan(&dbSchemaVersion)
			if err != nil {
				err = fmt.Errorf("error at get schema version sql: %w", err)
				return false, nil, err
			}
		} else {
			err = fmt.Errorf("error at query :%w", err)
			return false, nil, err
		}
	}

	// ここから 過去バージョンのスキーマだった場合の対応
	if currentSchemaVersion != dbSchemaVersion {
		switch dbSchemaVersion {
		case "1.0.0":
			// 過去のDAOを作って返す or 最新のDAOに変換して返す
		}
		err = fmt.Errorf("invalid db schema version %s", dbSchemaVersion)
		return true, nil, err
	}
	// ここまで 過去バージョンのスキーマだった場合の対応

	return false, nil, nil
}
