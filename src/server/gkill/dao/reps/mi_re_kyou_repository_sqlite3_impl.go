package reps

import (
	"context"
	sqllib "database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
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

// miReKyouTableName はMiReKyou実体テーブルの名前です。
const miReKyouTableName = "MIREKYOU"

type miReKyouRepositorySQLite3Impl struct {
	filename          string
	db                *sqllib.DB
	m                 *sync.RWMutex
	gkillRepositories *GkillRepositories
	fullConnect       bool
}

func NewMiReKyouRepositorySQLite3Impl(ctx context.Context, filename string, fullConnect bool, reps *GkillRepositories) (MiReKyouRepository, error) {
	db, err := sqlite3impl.GetSQLiteDBConnection(ctx, filename)
	if err != nil {
		return nil, err
	}
	if gkill_options.Optimize {
		err = sqlite3impl.DeleteAllIndex(db)
		if err != nil {
			err = fmt.Errorf("error at delete all index %w", err)
			return nil, err
		}
	}

	sql := `CREATE TABLE IF NOT EXISTS "` + miReKyouTableName + `" (` + miReKyouColumns + `);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create MIREKYOU table statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MIREKYOU table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_MIREKYOU ON ` + miReKyouTableName + ` (ID, UPDATE_TIME);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create MIREKYOU index statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := indexStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MIREKYOU index to %s: %w", filename, err)
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

	return &miReKyouRepositorySQLite3Impl{
		filename:          filename,
		db:                db,
		m:                 &sync.RWMutex{},
		gkillRepositories: reps,
		fullConnect:       fullConnect,
	}, nil
}

// getDBConnection は接続方式に応じたDBコネクションと、後始末用の関数を返します。
func (m *miReKyouRepositorySQLite3Impl) getDBConnection(ctx context.Context) (*sqllib.DB, func(), error) {
	if m.fullConnect {
		return m.db, func() {}, nil
	}
	db, err := sqlite3impl.GetSQLiteDBConnection(ctx, m.filename)
	if err != nil {
		return nil, nil, err
	}
	return db, func() {
		err := db.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}, nil
}

func (m *miReKyouRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	db, closeDB, err := m.getDBConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer closeDB()

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = m.UpdateCache(ctx)
		if err != nil {
			repName, _ := m.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

	// ターゲット解決はリポジトリ横断で行うためロックの外で先に済ませる
	repsWithoutMiReKyou, err := m.GetRepositoriesWithoutMiReKyouRep(ctx)
	if err != nil {
		return nil, err
	}
	targetFilter, err := newMiReKyouTargetFilter(ctx, repsWithoutMiReKyou, query)
	if err != nil {
		return nil, err
	}

	m.m.RLock()
	defer m.m.RUnlock()

	repName, err := m.getRepNameWithoutLock(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MIREKYOU: %w", err)
		return nil, err
	}

	// FindKyousは常に5射影すべてを対象にする(Miと同じ)
	sql, queryArgs, err := buildMiReKyouKyouSQL(query, repName, miReKyouTableName, false, nil)
	if err != nil {
		return nil, err
	}

	kyousList, err := m.queryKyous(ctx, db, sql, queryArgs, repName)
	if err != nil {
		return nil, err
	}

	targetIDMap, err := m.getTargetIDMap(ctx, db)
	if err != nil {
		return nil, err
	}

	kyous := map[string][]Kyou{}
	for _, kyou := range kyousList {
		// ターゲットが存在しない、またはワード検索にヒットしないものは除外する
		if !targetFilter.isMatch(targetIDMap[kyou.ID]) {
			continue
		}
		if _, exist := kyous[kyou.ID]; !exist {
			kyous[kyou.ID] = []Kyou{}
		}
		kyous[kyou.ID] = append(kyous[kyou.ID], kyou)
	}
	return kyous, nil
}

// queryKyous はKyou取得SQLを実行して結果を返します。
func (m *miReKyouRepositorySQLite3Impl) queryKyous(ctx context.Context, db *sqllib.DB, sql string, queryArgs []any, repName string) ([]Kyou, error) {
	if sql == "" {
		return []Kyou{}, nil
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get mirekyou kyou sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from MIREKYOU: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	return scanMiReKyouKyous(ctx, rows, repName)
}

// queryMiReKyous はMiReKyou取得SQLを実行して結果を返します。
func (m *miReKyouRepositorySQLite3Impl) queryMiReKyous(ctx context.Context, db *sqllib.DB, sql string, queryArgs []any, repName string) ([]MiReKyou, error) {
	if sql == "" {
		return []MiReKyou{}, nil
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find mirekyou sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from MIREKYOU: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	return scanMiReKyous(ctx, rows, repName)
}

// getTargetIDMap はMiReKyouのIDからTARGET_IDを引くマップを作ります。
func (m *miReKyouRepositorySQLite3Impl) getTargetIDMap(ctx context.Context, db *sqllib.DB) (map[string]string, error) {
	sql := `SELECT ID, TARGET_ID FROM ` + miReKyouTableName
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get target id map sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at select target id from MIREKYOU: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	targetIDMap := map[string]string{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			id, targetID := "", ""
			err = rows.Scan(&id, &targetID)
			if err != nil {
				err = fmt.Errorf("error at scan target id in MIREKYOU: %w", err)
				return nil, err
			}
			targetIDMap[id] = targetID
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return targetIDMap, nil
}

func (m *miReKyouRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	// updateTime未指定なら最新版のみを対象にする
	onlyLatest := updateTime == nil
	kyous, err := m.getKyousByID(ctx, id, updateTime, &onlyLatest)
	if err != nil {
		return nil, err
	}
	if len(kyous) == 0 {
		return nil, nil
	}
	return &kyous[0], nil
}

func (m *miReKyouRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	// 履歴なのですべての版を返す
	onlyLatest := false
	return m.getKyousByID(ctx, id, nil, &onlyLatest)
}

// getKyousByID はIDに一致するKyouを取得します。
func (m *miReKyouRepositorySQLite3Impl) getKyousByID(ctx context.Context, id string, updateTime *time.Time, onlyLatest *bool) ([]Kyou, error) {
	db, closeDB, err := m.getDBConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer closeDB()

	m.m.RLock()
	defer m.m.RUnlock()

	repName, err := m.getRepNameWithoutLock(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MIREKYOU: %w", err)
		return nil, err
	}

	query := &find.FindQuery{
		UseIDs:        true,
		IDs:           []string{id},
		UseUpdateTime: updateTime != nil,
		UpdateTime:    updateTime,
	}

	sql, queryArgs, err := buildMiReKyouKyouSQL(query, repName, miReKyouTableName, false, onlyLatest)
	if err != nil {
		return nil, err
	}
	return m.queryKyous(ctx, db, sql, queryArgs, repName)
}

func (m *miReKyouRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if id == "" {
		return m.filename, nil
	}
	return filepath.Abs(m.filename)
}

func (m *miReKyouRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (m *miReKyouRepositorySQLite3Impl) LastUpdateCacheChanged() bool {
	return true
}

func (m *miReKyouRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.getRepNameWithoutLock(ctx)
}

// getRepNameWithoutLock はロックを取らずにリポジトリ名を返します。
// ロック済みの箇所から呼ぶためのものです。
func (m *miReKyouRepositorySQLite3Impl) getRepNameWithoutLock(ctx context.Context) (string, error) {
	path, err := m.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path mirekyou rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (m *miReKyouRepositorySQLite3Impl) Close(ctx context.Context) error {
	m.m.Lock()
	defer m.m.Unlock()
	if m.fullConnect {
		return m.db.Close()
	}
	return nil
}

func (m *miReKyouRepositorySQLite3Impl) FindMiReKyou(ctx context.Context, query *find.FindQuery) ([]MiReKyou, error) {
	db, closeDB, err := m.getDBConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer closeDB()

	if query.UpdateCache {
		err = m.UpdateCache(ctx)
		if err != nil {
			repName, _ := m.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

	// ターゲット解決はリポジトリ横断で行うためロックの外で先に済ませる
	repsWithoutMiReKyou, err := m.GetRepositoriesWithoutMiReKyouRep(ctx)
	if err != nil {
		return nil, err
	}
	targetFilter, err := newMiReKyouTargetFilter(ctx, repsWithoutMiReKyou, query)
	if err != nil {
		return nil, err
	}

	m.m.RLock()
	defer m.m.RUnlock()

	repName, err := m.getRepNameWithoutLock(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MIREKYOU: %w", err)
		return nil, err
	}

	sql, queryArgs, err := buildMiReKyouSQL(query, repName, miReKyouTableName, true, nil)
	if err != nil {
		return nil, err
	}
	mirekyous, err := m.queryMiReKyous(ctx, db, sql, queryArgs, repName)
	if err != nil {
		return nil, err
	}

	filteredMiReKyous := []MiReKyou{}
	for _, mirekyou := range mirekyous {
		// ターゲットが存在しない、またはワード検索にヒットしないものは除外する
		if !targetFilter.isMatch(mirekyou.TargetID) {
			continue
		}
		filteredMiReKyous = append(filteredMiReKyous, mirekyou)
	}
	return filteredMiReKyous, nil
}

func (m *miReKyouRepositorySQLite3Impl) GetMiReKyou(ctx context.Context, id string, updateTime *time.Time) (*MiReKyou, error) {
	// updateTime未指定なら最新版のみを対象にする
	mirekyous, err := m.getMiReKyousByID(ctx, id, updateTime, updateTime == nil)
	if err != nil {
		return nil, err
	}
	if len(mirekyous) == 0 {
		return nil, nil
	}
	return &mirekyous[0], nil
}

func (m *miReKyouRepositorySQLite3Impl) GetMiReKyouHistories(ctx context.Context, id string) ([]MiReKyou, error) {
	// 履歴なのですべての版を返す
	return m.getMiReKyousByID(ctx, id, nil, false)
}

// getMiReKyousByID はIDに一致するMiReKyouを取得します。
func (m *miReKyouRepositorySQLite3Impl) getMiReKyousByID(ctx context.Context, id string, updateTime *time.Time, onlyLatest bool) ([]MiReKyou, error) {
	db, closeDB, err := m.getDBConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer closeDB()

	m.m.RLock()
	defer m.m.RUnlock()

	repName, err := m.getRepNameWithoutLock(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MIREKYOU: %w", err)
		return nil, err
	}

	query := &find.FindQuery{
		UseIDs:        true,
		IDs:           []string{id},
		UseUpdateTime: updateTime != nil,
		UpdateTime:    updateTime,
	}

	sql, queryArgs, err := buildMiReKyouSingleProjectionSQL(query, repName, miReKyouTableName, onlyLatest)
	if err != nil {
		return nil, err
	}
	return m.queryMiReKyous(ctx, db, sql, queryArgs, repName)
}

func (m *miReKyouRepositorySQLite3Impl) AddMiReKyouInfo(ctx context.Context, mirekyou MiReKyou) error {
	if strings.TrimSpace(mirekyou.TargetID) == "" {
		return fmt.Errorf("mirekyou target id must not be empty")
	}

	db, closeDB, err := m.getDBConnection(ctx)
	if err != nil {
		return err
	}
	defer closeDB()

	m.m.Lock()
	defer m.m.Unlock()

	sql := `INSERT INTO ` + miReKyouTableName + ` (` + miReKyouInsertColumnNames + `) VALUES (` + miReKyouInsertPlaceHolders + `)`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add mirekyou sql %s: %w", mirekyou.ID, err)
		return err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := miReKyouInsertArgs(mirekyou)
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to mirekyou %s: %w", mirekyou.ID, err)
		return err
	}
	return nil
}

// GetMiReKyousAllLatest はターゲット解決を行わない生のMiReKyou(ID毎の最新)を返します。
// ターゲットの存在確認やワード検索はFindMiReKyou/FindKyous側で行います。
func (m *miReKyouRepositorySQLite3Impl) GetMiReKyousAllLatest(ctx context.Context) ([]MiReKyou, error) {
	db, closeDB, err := m.getDBConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer closeDB()

	m.m.RLock()
	defer m.m.RUnlock()

	repName, err := m.getRepNameWithoutLock(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at MIREKYOU: %w", err)
		return nil, err
	}

	sql, queryArgs, err := buildMiReKyouSingleProjectionSQL(&find.FindQuery{}, repName, miReKyouTableName, true)
	if err != nil {
		return nil, err
	}
	return m.queryMiReKyous(ctx, db, sql, queryArgs, repName)
}

func (m *miReKyouRepositorySQLite3Impl) GetBoardNames(ctx context.Context) ([]string, error) {
	db, closeDB, err := m.getDBConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer closeDB()

	m.m.RLock()
	defer m.m.RUnlock()

	return queryMiReKyouBoardNames(ctx, db, miReKyouTableName)
}

func (m *miReKyouRepositorySQLite3Impl) GetRepositoriesWithoutMiReKyouRep(ctx context.Context) (*GkillRepositories, error) {
	mirekyouReps := MiReKyouRepositories{
		MiReKyouRepositories: nil,
		GkillRepositories:    m.gkillRepositories,
	}
	return mirekyouReps.GetRepositoriesWithoutMiReKyouRep(ctx)
}

func (m *miReKyouRepositorySQLite3Impl) UnWrapTyped() ([]MiReKyouRepository, error) {
	return []MiReKyouRepository{m}, nil
}

func (m *miReKyouRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{m}, nil
}

func (m *miReKyouRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	db, closeDB, err := m.getDBConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer closeDB()

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

	repName, err := m.getRepNameWithoutLock(ctx)
	if err != nil {
		return nil, err
	}

	return queryMiReKyouLatestDataRepositoryAddress(ctx, db, miReKyouTableName, repName)
}

// queryMiReKyouBoardNames は板名の一覧を取得します。
func queryMiReKyouBoardNames(ctx context.Context, db *sqllib.DB, tableName string) ([]string, error) {
	sql := `SELECT DISTINCT BOARD_NAME FROM ` + tableName
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
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

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at select board names from MIREKYOU: %w", err)
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
				err = fmt.Errorf("error at scan rows at get board names in MIREKYOU: %w", err)
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

// queryMiReKyouLatestDataRepositoryAddress は最新データ位置キャッシュ用の情報を取得します。
// ReKyouと同様、TARGET_ID_IN_DATAにリポスト対象のIDを入れます。
func queryMiReKyouLatestDataRepositoryAddress(ctx context.Context, db *sqllib.DB, tableName string, repName string) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	sql := `
SELECT IS_DELETED, ID AS TARGET_ID, TARGET_ID AS TARGET_ID_IN_DATA,
       ? AS LATEST_DATA_REPOSITORY_NAME, UPDATE_TIME AS DATA_UPDATE_TIME
FROM ` + tableName
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	rows, err := stmt.QueryContext(ctx, repName)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	latestDataRepositoryAddressMap := map[string]gkill_cache.LatestDataRepositoryAddress{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			addr := gkill_cache.LatestDataRepositoryAddress{}
			var isDeletedStr string
			var dataUpdateTimeStr string
			var targetIDInData *string
			err := rows.Scan(&isDeletedStr, &addr.TargetID, &targetIDInData, &addr.LatestDataRepositoryName, &dataUpdateTimeStr)
			if err != nil {
				return nil, err
			}
			addr.IsDeleted = isDeletedStr == "TRUE"
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
