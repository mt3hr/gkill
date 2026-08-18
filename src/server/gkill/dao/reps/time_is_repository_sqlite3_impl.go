package reps

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"path/filepath"
	"strings"
	"sync"
	"time"

	sqllib "database/sql"

	_ "modernc.org/sqlite"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

const CURRENT_SCHEMA_VERSION_TIMEIS_REPOSITORY_SQLITE3IMPL_DAO = "1.0.0"

type timeIsRepositorySQLite3Impl struct {
	// キャッシュrepのフルリビルドを、実DBファイルが変わったときだけに絞るための判定用。
	// temp repが構造体変換でこの型をコピーするため、必ずポインタで持つこと。
	cacheChange *dbFileChangeDetector

	filename    string
	db          *sqllib.DB
	m           *sync.RWMutex
	fullConnect bool
}

func NewTimeIsRepositorySQLite3Impl(ctx context.Context, filename string, fullConnect bool) (TimeIsRepository, error) {
	db, err := sqlite3impl.GetSQLiteDBConnection(ctx, filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	if isOld, oldVerDAO, err := checkAndResolveDataSchemaTimeIsRepositorySQLite3Impl(ctx, db); err != nil {
		return nil, err
	} else if isOld {
		if oldVerDAO != nil {
			return oldVerDAO, nil
		} else {
			err = fmt.Errorf("error at load database schema %s", filename)
			return nil, err
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
	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS table statement %s: %w", filename, err)
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
		err = fmt.Errorf("error at create TIMEIS table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_TIMEIS ON TIMEIS (ID, UPDATE_TIME);`
	gkill_log.LogIndexSQL(ctx, indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create TIMEIS index statement %s: %w", filename, err)
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
		err = fmt.Errorf("error at create TIMEIS index to %s: %w", filename, err)
		return nil, err
	}

	// 時刻範囲検索と並び替えのための式インデックス。
	// GenerateFindSQLCommon が unixepoch(列) で比較・整列するので、
	// 式そのものに索引を張らないと全走査になる。
	if err := sqlite3impl.EnsureUnixepochIndex(ctx, db, "TIMEIS", "START_TIME", "END_TIME"); err != nil {
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

	return &timeIsRepositorySQLite3Impl{
		cacheChange: &dbFileChangeDetector{},
		filename:    filename,
		db:          db,
		m:           &sync.RWMutex{},
		fullConnect: fullConnect,
	}, nil
}
func (t *timeIsRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	var err error
	var db *sqllib.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
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
	if query.IncludeEndTimeIs {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start', 'timeis_end') AND END_TIME IS NOT NULL"
	} else {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start')"
	}

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at TIMEIS: %w", err)
		return nil, err
	}

	tableName := "TIMEIS"
	tableNameAlias := "TIMEIS"
	queryArgsForStart := []any{
		repName,
	}
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	queryArgsForPlaingStart := []any{}
	sqlWhereFilterPlaingTimeisStart := ""
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	if query.PlaingTime != nil {
		onlyLatestData = true
	}
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	if query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisStart += " AND ((datetime(?, 'localtime') >= datetime(START_TIME, 'localtime')) AND (datetime(?, 'localtime') <= datetime(END_TIME, 'localtime') OR END_TIME IS NULL)) "
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (query.PlaingTime).Format(sqlite3impl.TimeLayout))
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (query.PlaingTime).Format(sqlite3impl.TimeLayout))
		whereCounter++
		whereCounter++
	}

	tableName = "TIMEIS"
	tableNameAlias = "TIMEIS"
	queryArgsForEnd := []any{
		repName,
	}
	whereCounter = 0
	relatedTimeColumnName = "RELATED_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	queryArgsForPlaingEnd := []any{}
	sqlWhereFilterPlaingTimeisEnd := ""
	ignoreCase = true

	onlyLatestData = query.OnlyLatestData
	if query.PlaingTime != nil {
		onlyLatestData = true
	}
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	if query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisEnd += " AND ((datetime(?, 'localtime') >= datetime(START_TIME, 'localtime')) AND (datetime(?, 'localtime') <= datetime(END_TIME, 'localtime') OR END_TIME IS NULL)) "
		queryArgsForPlaingEnd = append(queryArgsForPlaingEnd, (query.PlaingTime).Format(sqlite3impl.TimeLayout))
		queryArgsForPlaingEnd = append(queryArgsForPlaingEnd, (query.PlaingTime).Format(sqlite3impl.TimeLayout))
		whereCounter++
		whereCounter++
	}

	// 各射影は DATA_TYPE に異なる文字列リテラルを出すので、腕をまたいで同一行になることが原理的に無い。
	// UNIONの重複排除は必ず空振りするのに、SQLiteは結果全体ぶんの一時Btreeを作る
	// (modernc.org/sqlite は SQLITE_TEMP_STORE=1 なので、それがディスクの一時ファイルに落ちる)。
	// 腕の中の完全重複も、消費側が必ずIDキーのマップへ入れる(find_filter の kyouEntryKey / filterMiForMi)ので
	// 観測可能な差は出ない。
	sql := fmt.Sprintf("%s WHERE %s %s UNION ALL %s WHERE %s %s AND %s", sqlStartTimeIs, sqlWhereForStart, sqlWhereFilterPlaingTimeisStart, sqlEndTimeIs, sqlWhereForEnd, sqlWhereFilterPlaingTimeisEnd, sqlWhereFilterEndTimeIs)

	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
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

	if gkill_log.TraceSQLEnabled(ctx) {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params_start", fmt.Sprintf("%q", fmt.Sprint(queryArgsForStart)), "params_plaing_start", fmt.Sprintf("%q", fmt.Sprint(queryArgsForPlaingStart)), "params_end", fmt.Sprintf("%q", fmt.Sprint(queryArgsForEnd)), "params_plaing_end", fmt.Sprintf("%q", fmt.Sprint(queryArgsForPlaingEnd)))
	}
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

func (t *timeIsRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error
	var db *sqllib.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
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

	ids := []string{id}
	query := &find.FindQuery{
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UpdateTime:     updateTime,
	}

	queryArgs := []any{
		repName,
	}
	tableName := "TIMEIS"
	tableNameAlias := "TIMEIS"
	whereCounter := 0
	// GenerateFindSQLCommon は query.OnlyLatestData を読まず、この引数しか見ない。
	// false のままだと updateTime 未指定のときに **そのIDの全バージョンを無順序・無制限に読み**、
	// 下の kyous[0] が格納順の先頭(多くの場合いちばん古い版)を返してしまう。
	// 版の数だけ走査するので遅くもある。Tag / Text では既に同じ修正が入っている。
	onlyLatestData := updateTime == nil
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

	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
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

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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

func (t *timeIsRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error
	var db *sqllib.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
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

	ids := []string{id}
	query := &find.FindQuery{
		IDs: ids,
	}

	queryArgs := []any{
		repName,
	}
	tableName := "TIMEIS"
	tableNameAlias := "TIMEIS"
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

	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
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

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return kyous, nil
}

func (t *timeIsRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if id == "" {
		return t.filename, nil
	}
	return filepath.Abs(t.filename)
}

func (t *timeIsRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	// 自身は実DBを直接見るのでキャッシュは持たないが、
	// 上位のキャッシュrepが「作り直す必要があるか」を判断できるよう、
	// ここでファイルの更新有無だけ観測しておく。
	t.cacheChange.refresh(t.filename)
	return nil
}

func (t *timeIsRepositorySQLite3Impl) LastUpdateCacheChanged() bool {
	return t.cacheChange.lastChanged()
}

// CommitCacheRebuild は上位のキャッシュrepが再構築に成功したときに呼ばれます。
func (t *timeIsRepositorySQLite3Impl) CommitCacheRebuild() {
	t.cacheChange.commit()
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
	t.m.Lock()
	defer t.m.Unlock()
	if t.fullConnect {
		return t.db.Close()
	}
	return nil
}

func (t *timeIsRepositorySQLite3Impl) FindTimeIs(ctx context.Context, query *find.FindQuery) ([]TimeIs, error) {
	var err error
	var db *sqllib.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
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
  ? AS REP_NAME,
  'timeis_end' AS DATA_TYPE
FROM TIMEIS 
`

	sqlWhereFilterEndTimeIs := ""
	if query.IncludeEndTimeIs {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start', 'timeis_end') AND END_TIME IS NOT NULL"
	} else {
		sqlWhereFilterEndTimeIs = "DATA_TYPE IN ('timeis_start')"
	}

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at TIMEIS: %w", err)
		return nil, err
	}

	tableName := "TIMEIS"
	tableNameAlias := "TIMEIS"
	queryArgsForStart := []any{
		repName,
	}
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	queryArgsForPlaingStart := []any{}
	sqlWhereFilterPlaingTimeisStart := ""
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	if query.PlaingTime != nil {
		onlyLatestData = true
	}
	sqlWhereForStart, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForStart)
	if err != nil {
		return nil, err
	}
	if query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisStart += " AND ((datetime(?, 'localtime') >= datetime(START_TIME, 'localtime')) AND (datetime(?, 'localtime') <= datetime(END_TIME, 'localtime') OR END_TIME IS NULL)) "
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (query.PlaingTime).Format(sqlite3impl.TimeLayout))
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (query.PlaingTime).Format(sqlite3impl.TimeLayout))
		whereCounter++
		whereCounter++
	}

	tableName = "TIMEIS"
	tableNameAlias = "TIMEIS"
	queryArgsForEnd := []any{
		repName,
	}
	whereCounter = 0
	// start分岐と同じくquery依存にする(以前はend分岐だけtrue固定で非対称だった)
	onlyLatestData = query.OnlyLatestData
	if query.PlaingTime != nil {
		onlyLatestData = true
	}
	relatedTimeColumnName = "RELATED_TIME"
	findWordTargetColumns = []string{"TITLE"}
	ignoreFindWord = false
	appendOrderBy = false
	findWordUseLike = true
	queryArgsForPlaingEnd := []any{}
	sqlWhereFilterPlaingTimeisEnd := ""
	ignoreCase = true
	sqlWhereForEnd, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgsForEnd)
	if err != nil {
		return nil, err
	}
	if query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisEnd += " AND ((datetime(?, 'localtime') >= datetime(START_TIME, 'localtime')) AND (datetime(?, 'localtime') <= datetime(END_TIME, 'localtime') OR END_TIME IS NULL)) "
		queryArgsForPlaingEnd = append(queryArgsForPlaingEnd, (query.PlaingTime).Format(sqlite3impl.TimeLayout))
		queryArgsForPlaingEnd = append(queryArgsForPlaingEnd, (query.PlaingTime).Format(sqlite3impl.TimeLayout))
		whereCounter++
		whereCounter++
	}

	// 各射影は DATA_TYPE に異なる文字列リテラルを出すので、腕をまたいで同一行になることが原理的に無い。
	// UNIONの重複排除は必ず空振りするのに、SQLiteは結果全体ぶんの一時Btreeを作る
	// (modernc.org/sqlite は SQLITE_TEMP_STORE=1 なので、それがディスクの一時ファイルに落ちる)。
	// 腕の中の完全重複も、消費側が必ずIDキーのマップへ入れる(find_filter の kyouEntryKey / filterMiForMi)ので
	// 観測可能な差は出ない。
	sql := fmt.Sprintf("%s WHERE %s %s UNION ALL %s WHERE %s %s AND %s", sqlStartTimeIs, sqlWhereForStart, sqlWhereFilterPlaingTimeisStart, sqlEndTimeIs, sqlWhereForEnd, sqlWhereFilterPlaingTimeisEnd, sqlWhereFilterEndTimeIs)

	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find timeis sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	if gkill_log.TraceSQLEnabled(ctx) {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params_start", fmt.Sprintf("%q", fmt.Sprint(queryArgsForStart)), "params_plaing_start", fmt.Sprintf("%q", fmt.Sprint(queryArgsForPlaingStart)), "params_end", fmt.Sprintf("%q", fmt.Sprint(queryArgsForEnd)), "params_plaing_end", fmt.Sprintf("%q", fmt.Sprint(queryArgsForPlaingEnd)))
	}
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
				err = fmt.Errorf("error at parse start time %s in TIMEIS: %w", startTimeStr, err)
				return nil, err
			}
			if endTime.Valid {
				parsedEndTime, _ := time.Parse(sqlite3impl.TimeLayout, endTime.String)
				timeis.EndTime = &parsedEndTime
			}
			timeiss = append(timeiss, timeis)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return timeiss, nil
}

func (t *timeIsRepositorySQLite3Impl) GetTimeIs(ctx context.Context, id string, updateTime *time.Time) (*TimeIs, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error
	var db *sqllib.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
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

	ids := []string{id}
	query := &find.FindQuery{
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UpdateTime:     updateTime,
	}

	dataType := "timeis"

	tableName := "TIMEIS"
	tableNameAlias := "TIMEIS"
	queryArgs := []any{
		repName,
		dataType,
	}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	queryArgsForPlaingStart := []any{}
	sqlWhereFilterPlaingTimeisStart := ""
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	if query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisStart += " AND ((datetime(?, 'localtime') >= datetime(START_TIME, 'localtime')) AND (datetime(?, 'localtime') <= datetime(END_TIME, 'localtime') OR END_TIME IS NULL)) "
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (query.PlaingTime).Format(sqlite3impl.TimeLayout))
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (query.PlaingTime).Format(sqlite3impl.TimeLayout))
		whereCounter++
		whereCounter++
	}

	sql += commonWhereSQL + sqlWhereFilterPlaingTimeisStart
	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get time is sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	if gkill_log.TraceSQLEnabled(ctx) {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(append(queryArgsForPlaingStart, queryArgs...))))
	}
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
				err = fmt.Errorf("error at parse start time %s in TIMEIS: %w", startTimeStr, err)
				return nil, err
			}
			if endTime.Valid {
				parsedEndTime, _ := time.Parse(sqlite3impl.TimeLayout, endTime.String)
				timeis.EndTime = &parsedEndTime
			}
			timeiss = append(timeiss, timeis)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(timeiss) == 0 {
		return nil, nil
	}
	return &timeiss[0], nil
}

func (t *timeIsRepositorySQLite3Impl) GetTimeIsHistories(ctx context.Context, id string) ([]TimeIs, error) {
	t.m.RLock()
	defer t.m.RUnlock()
	var err error
	var db *sqllib.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
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

	ids := []string{id}
	query := &find.FindQuery{
		IDs: ids,
	}

	dataType := "timeis"

	tableName := "TIMEIS"
	tableNameAlias := "TIMEIS"
	queryArgs := []any{
		repName,
		dataType,
	}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TITLE"}
	ignoreFindWord := false
	appendOrderBy := false
	queryArgsForPlaingStart := []any{}
	sqlWhereFilterPlaingTimeisStart := ""
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	if query.PlaingTime != nil {
		sqlWhereFilterPlaingTimeisStart += " AND ((datetime(?, 'localtime') >= datetime(START_TIME, 'localtime')) AND (datetime(?, 'localtime') <= datetime(END_TIME, 'localtime') OR END_TIME IS NULL)) "
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (query.PlaingTime).Format(sqlite3impl.TimeLayout))
		queryArgsForPlaingStart = append(queryArgsForPlaingStart, (query.PlaingTime).Format(sqlite3impl.TimeLayout))
		whereCounter++
		whereCounter++
	}

	sql += commonWhereSQL + sqlWhereFilterPlaingTimeisStart
	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
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

	if gkill_log.TraceSQLEnabled(ctx) {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(append(queryArgsForPlaingStart, queryArgs...))))
	}
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
				err = fmt.Errorf("error at parse start time %s in TIMEIS: %w", startTimeStr, err)
				return nil, err
			}
			if endTime.Valid {
				parsedEndTime, _ := time.Parse(sqlite3impl.TimeLayout, endTime.String)
				timeis.EndTime = &parsedEndTime
			}
			timeiss = append(timeiss, timeis)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return timeiss, nil
}

func (t *timeIsRepositorySQLite3Impl) AddTimeIsInfo(ctx context.Context, timeis TimeIs) error {
	t.m.Lock()
	defer t.m.Unlock()
	var err error
	var db *sqllib.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
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

	if strings.TrimSpace(timeis.Title) == "" {
		return fmt.Errorf("timeis title must not be empty")
	}

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
	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
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

	var endTimeStr any
	if timeis.EndTime == nil {
		endTimeStr = nil
	} else {
		endTimeStr = timeis.EndTime.Format(sqlite3impl.TimeLayout)
	}

	queryArgs := []any{
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
	gkill_log.LogSQLParams(ctx, sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to timeis %s: %w", timeis.ID, err)
		return err
	}
	return nil
}

func (t *timeIsRepositorySQLite3Impl) UnWrapTyped() ([]TimeIsRepository, error) {
	return []TimeIsRepository{t}, nil
}

func (t *timeIsRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{t}, nil
}

func (t *timeIsRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	var err error
	var db *sqllib.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
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
		err = t.UpdateCache(ctx)
		if err != nil {
			repName, _ := t.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	t.m.RLock()
	defer t.m.RUnlock()

	repName, err := t.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	sql := `
SELECT IS_DELETED, ID AS TARGET_ID, NULL AS TARGET_ID_IN_DATA,
       ? AS LATEST_DATA_REPOSITORY_NAME, UPDATE_TIME AS DATA_UPDATE_TIME
FROM TIMEIS
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
	latestDataRepositoryAddresses := make([]gkill_cache.LatestDataRepositoryAddress, 0, len(latestDataRepositoryAddressMap))
	for _, addr := range latestDataRepositoryAddressMap {
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, addr)
	}
	return latestDataRepositoryAddresses, nil
}

func checkAndResolveDataSchemaTimeIsRepositorySQLite3Impl(ctx context.Context, db *sqllib.DB) (isOld bool, oldVerDAO TimeIsRepository, err error) {
	schemaVersionKey := "SCHEMA_VERSION_TIMEIS"
	currentSchemaVersion := CURRENT_SCHEMA_VERSION_TIMEIS_REPOSITORY_SQLITE3IMPL_DAO

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
				err = fmt.Errorf("error at get schema version sql: %w", err)
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
				err = fmt.Errorf("error at get schema version sql: %w", err)
				err = fmt.Errorf("error at query :%w", err)
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
