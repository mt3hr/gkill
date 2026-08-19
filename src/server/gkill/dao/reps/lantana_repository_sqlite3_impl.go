package reps

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	_ "modernc.org/sqlite"
)

const CURRENT_SCHEMA_VERSION_LANTANA_REPOSITORY_SQLITE3IMPL_DAO = "1.0.0"

type lantanaRepositorySQLite3Impl struct {
	// キャッシュrepのフルリビルドを、実DBファイルが変わったときだけに絞るための判定用。
	// temp repが構造体変換でこの型をコピーするため、必ずポインタで持つこと。
	cacheChange *dbFileChangeDetector

	filename    string
	db          *sql.DB
	m           *sync.RWMutex
	fullConnect bool
}

func NewLantanaRepositorySQLite3Impl(ctx context.Context, filename string, fullConnect bool) (LantanaRepository, error) {
	db, err := sqlite3impl.GetSQLiteDBConnection(ctx, filename)
	if err != nil {
		return nil, err
	}

	if isOld, oldVerDAO, err := checkAndResolveDataSchemaLantanaRepositorySQLite3Impl(ctx, db); err != nil {
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
CREATE TABLE IF NOT EXISTS "LANTANA" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  MOOD NOT NULL,
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
	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create LANTANA table statement %s: %w", filename, err)
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
		err = fmt.Errorf("error at create LANTANA table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_LANTANA ON LANTANA (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.LogIndexSQL(ctx, indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create LANTANA index statement %s: %w", filename, err)
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
		err = fmt.Errorf("error at create LANTANA index to %s: %w", filename, err)
		return nil, err
	}

	// 時刻範囲検索と並び替えのための式インデックス。
	// GenerateFindSQLCommon が unixepoch(列) で比較・整列するので、
	// 式そのものに索引を張らないと全走査になる。
	if err := sqlite3impl.EnsureUnixepochIndex(ctx, db, "LANTANA", "RELATED_TIME"); err != nil {
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

	return &lantanaRepositorySQLite3Impl{
		cacheChange: &dbFileChangeDetector{},
		filename:    filename,
		db:          db,
		m:           &sync.RWMutex{},
		fullConnect: fullConnect,
	}, nil
}

func (l *lantanaRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	var err error
	var db *sql.DB
	if l.fullConnect {
		db = l.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, l.filename)
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
		err = l.UpdateCache(ctx)
		if err != nil {
			repName, _ := l.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}
	l.m.RLock()
	defer l.m.RUnlock()

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
FROM LANTANA
WHERE
`
	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
		return nil, err
	}

	dataType := "lantana"

	queryArgs := []any{
		repName,
		dataType,
	}

	tableName := "LANTANA"
	tableNameAlias := "LANTANA"
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	// 結果は map[string][]Kyou に収めるので、SQL側で並べても順序は捨てられる。
	// 最終的な並び順は find_filter の Go 側ソートで決まる。
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

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
		err = fmt.Errorf("error at select from LANTANA: %w", err)
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
				err = fmt.Errorf("error at scan from LANTANA: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in LANTANA: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in LANTANA: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in LANTANA: %w", updateTimeStr, err)
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

func (l *lantanaRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	l.m.RLock()
	defer l.m.RUnlock()
	var err error
	var db *sql.DB
	if l.fullConnect {
		db = l.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, l.filename)
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

	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
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
FROM  LANTANA
WHERE 
`

	dataType := "lantana"

	ids := []string{id}
	query := &find.FindQuery{
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UpdateTime:     updateTime,
	}

	queryArgs := []any{
		repName,
		dataType,
	}

	tableName := "LANTANA"
	tableNameAlias := "LANTANA"
	whereCounter := 0
	// GenerateFindSQLCommon は query.OnlyLatestData を読まず、この引数しか見ない。
	// false のままだと updateTime 未指定のときに **そのIDの全バージョンを無順序・無制限に読み**、
	// 下の kyous[0] が格納順の先頭(多くの場合いちばん古い版)を返してしまう。
	// 版の数だけ走査するので遅くもある。Tag / Text では既に同じ修正が入っている。
	onlyLatestData := updateTime == nil
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
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
		err = fmt.Errorf("error at select from LANTANA %s: %w", id, err)
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
				err = fmt.Errorf("error at scan from LANTANA %s: %w", id, err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in LANTANA: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in LANTANA: %w", createTimeStr, id, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in LANTANA: %w", updateTimeStr, id, err)
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

func (l *lantanaRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	l.m.RLock()
	defer l.m.RUnlock()
	var err error
	var db *sql.DB
	if l.fullConnect {
		db = l.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, l.filename)
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

	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
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
FROM  LANTANA
WHERE 
`

	dataType := "lantana"

	ids := []string{id}
	query := &find.FindQuery{
		IDs: ids,
	}

	queryArgs := []any{
		repName,
		dataType,
	}

	tableName := "LANTANA"
	tableNameAlias := "LANTANA"
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
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
		err = fmt.Errorf("error at select from LANTANA %s: %w", id, err)
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
				err = fmt.Errorf("error at scan from LANTANA %s: %w", id, err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in LANTANA: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in LANTANA: %w", createTimeStr, id, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in LANTANA: %w", updateTimeStr, id, err)
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

func (l *lantanaRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if id == "" {
		return l.filename, nil
	}
	return filepath.Abs(l.filename)
}

func (l *lantanaRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	// 自身は実DBを直接見るのでキャッシュは持たないが、
	// 上位のキャッシュrepが「作り直す必要があるか」を判断できるよう、
	// ここでファイルの更新有無だけ観測しておく。
	l.cacheChange.refresh(l.filename)
	return nil
}

func (l *lantanaRepositorySQLite3Impl) LastUpdateCacheChanged() bool {
	return l.cacheChange.lastChanged()
}

// CommitCacheRebuild は上位のキャッシュrepが再構築に成功したときに呼ばれます。
func (l *lantanaRepositorySQLite3Impl) CommitCacheRebuild() {
	l.cacheChange.commit()
}

func (l *lantanaRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := l.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path lantana rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (l *lantanaRepositorySQLite3Impl) Close(ctx context.Context) error {
	l.m.Lock()
	defer l.m.Unlock()
	if l.fullConnect {
		return l.db.Close()
	}
	return nil
}

func (l *lantanaRepositorySQLite3Impl) FindLantana(ctx context.Context, query *find.FindQuery) ([]Lantana, error) {
	var err error
	var db *sql.DB
	if l.fullConnect {
		db = l.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, l.filename)
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
		err = l.UpdateCache(ctx)
		if err != nil {
			repName, _ := l.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}
	l.m.RLock()
	defer l.m.RUnlock()

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
  MOOD,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM LANTANA
WHERE
`

	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
		return nil, err
	}

	dataType := "lantana"

	queryArgs := []any{
		repName,
		dataType,
	}

	tableName := "LANTANA"
	tableNameAlias := "LANTANA"
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find lantana sql: %w", err)
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
		err = fmt.Errorf("error at select from LANTANA: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	lantanas := []Lantana{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			lantana := Lantana{}
			lantana.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(&lantana.IsDeleted,
				&lantana.ID,
				&relatedTimeStr,
				&createTimeStr,
				&lantana.CreateApp,
				&lantana.CreateDevice,
				&lantana.CreateUser,
				&updateTimeStr,
				&lantana.UpdateApp,
				&lantana.UpdateDevice,
				&lantana.UpdateUser,
				&lantana.Mood,
				&lantana.RepName,
				&lantana.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from LANTANA: %w", err)
				return nil, err
			}

			lantana.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in LANTANA: %w", relatedTimeStr, err)
				return nil, err
			}
			lantana.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in LANTANA: %w", createTimeStr, err)
				return nil, err
			}
			lantana.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in LANTANA: %w", updateTimeStr, err)
				return nil, err
			}
			lantanas = append(lantanas, lantana)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return lantanas, nil
}

func (l *lantanaRepositorySQLite3Impl) GetLantana(ctx context.Context, id string, updateTime *time.Time) (*Lantana, error) {
	l.m.RLock()
	defer l.m.RUnlock()
	var err error
	var db *sql.DB
	if l.fullConnect {
		db = l.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, l.filename)
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
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  MOOD,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM LANTANA
WHERE
`

	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
		return nil, err
	}

	ids := []string{id}
	query := &find.FindQuery{
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UpdateTime:     updateTime,
	}

	dataType := "lantana"

	queryArgs := []any{
		repName,
		dataType,
	}

	tableName := "LANTANA"
	tableNameAlias := "LANTANA"
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
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
		err = fmt.Errorf("error at get lantana sql %s: %w", id, err)
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
		err = fmt.Errorf("error at query: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	lantanas := []Lantana{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			lantana := Lantana{}
			lantana.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(&lantana.IsDeleted,
				&lantana.ID,
				&relatedTimeStr,
				&createTimeStr,
				&lantana.CreateApp,
				&lantana.CreateDevice,
				&lantana.CreateUser,
				&updateTimeStr,
				&lantana.UpdateApp,
				&lantana.UpdateDevice,
				&lantana.UpdateUser,
				&lantana.Mood,
				&lantana.RepName,
				&lantana.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from LANTANA: %w", err)
				return nil, err
			}

			lantana.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in LANTANA: %w", relatedTimeStr, id, err)
				return nil, err
			}
			lantana.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in LANTANA: %w", createTimeStr, id, err)
				return nil, err
			}
			lantana.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in LANTANA: %w", updateTimeStr, id, err)
				return nil, err
			}
			lantanas = append(lantanas, lantana)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(lantanas) == 0 {
		return nil, nil
	}
	return &lantanas[0], nil
}

func (l *lantanaRepositorySQLite3Impl) GetLantanaHistories(ctx context.Context, id string) ([]Lantana, error) {
	l.m.RLock()
	defer l.m.RUnlock()
	var err error
	var db *sql.DB
	if l.fullConnect {
		db = l.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, l.filename)
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
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  MOOD,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM LANTANA
WHERE
`

	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
		return nil, err
	}

	ids := []string{id}
	query := &find.FindQuery{
		IDs: ids,
	}

	dataType := "lantana"

	queryArgs := []any{
		repName,
		dataType,
	}

	tableName := "LANTANA"
	tableNameAlias := "LANTANA"
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
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
		err = fmt.Errorf("error at get lantana histories sql %s: %w", id, err)
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
		err = fmt.Errorf("error at query: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	lantanas := []Lantana{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			lantana := Lantana{}
			lantana.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(&lantana.IsDeleted,
				&lantana.ID,
				&relatedTimeStr,
				&createTimeStr,
				&lantana.CreateApp,
				&lantana.CreateDevice,
				&lantana.CreateUser,
				&updateTimeStr,
				&lantana.UpdateApp,
				&lantana.UpdateDevice,
				&lantana.UpdateUser,
				&lantana.Mood,
				&lantana.RepName,
				&lantana.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from LANTANA: %w", err)
				return nil, err
			}

			lantana.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in LANTANA: %w", relatedTimeStr, id, err)
				return nil, err
			}
			lantana.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in LANTANA: %w", createTimeStr, id, err)
				return nil, err
			}
			lantana.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in LANTANA: %w", updateTimeStr, id, err)
				return nil, err
			}
			lantanas = append(lantanas, lantana)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return lantanas, nil
}

func (l *lantanaRepositorySQLite3Impl) AddLantanaInfo(ctx context.Context, lantana Lantana) error {
	l.m.Lock()
	defer l.m.Unlock()
	var err error
	var db *sql.DB
	if l.fullConnect {
		db = l.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, l.filename)
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

	sql := `
INSERT INTO LANTANA (
  IS_DELETED,
  ID,
  MOOD,
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
  ?
)`
	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add lantana sql %s: %w", lantana.ID, err)
		return err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []any{
		lantana.IsDeleted,
		lantana.ID,
		lantana.Mood,
		lantana.RelatedTime.Format(sqlite3impl.TimeLayout),
		lantana.CreateTime.Format(sqlite3impl.TimeLayout),
		lantana.CreateApp,
		lantana.CreateDevice,
		lantana.CreateUser,
		lantana.UpdateTime.Format(sqlite3impl.TimeLayout),
		lantana.UpdateApp,
		lantana.UpdateDevice,
		lantana.UpdateUser,
	}
	gkill_log.LogSQLParams(ctx, sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to LANTANA %s: %w", lantana.ID, err)
		return err
	}
	return nil
}

func (l *lantanaRepositorySQLite3Impl) UnWrapTyped() ([]LantanaRepository, error) {
	return []LantanaRepository{l}, nil
}

func (l *lantanaRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{l}, nil
}

func (l *lantanaRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	var err error
	var db *sql.DB
	if l.fullConnect {
		db = l.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, l.filename)
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
		err = l.UpdateCache(ctx)
		if err != nil {
			repName, _ := l.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	l.m.RLock()
	defer l.m.RUnlock()

	repName, err := l.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	sql := `
SELECT IS_DELETED, ID AS TARGET_ID, NULL AS TARGET_ID_IN_DATA,
       ? AS LATEST_DATA_REPOSITORY_NAME, UPDATE_TIME AS DATA_UPDATE_TIME
FROM LANTANA
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

func checkAndResolveDataSchemaLantanaRepositorySQLite3Impl(ctx context.Context, db *sql.DB) (isOld bool, oldVerDAO LantanaRepository, err error) {
	schemaVersionKey := "SCHEMA_VERSION_LANTANA"
	currentSchemaVersion := CURRENT_SCHEMA_VERSION_LANTANA_REPOSITORY_SQLITE3IMPL_DAO

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
		if errors.Is(err, sql.ErrNoRows) {
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
