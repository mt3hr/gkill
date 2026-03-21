package reps

import (
	"context"
	sqllib "database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

type gitCommitLogRepositoryCachedSQLite3Impl struct {
	dbName          string
	gitRep          GitCommitLogRepository
	cachedDB        *sqllib.DB
	m               *sync.RWMutex
	ownDB           bool // trueの場合、永続ファイルDBを自前で管理する
	backgroundUpdate bool // trueの場合、初回フルリビルドをバックグラウンドで実行する
	lastUpdateCacheChanged bool
}

func NewGitRepCachedSQLite3Impl(ctx context.Context, gitRep GitCommitLogRepository, cacheDB *sqllib.DB, m *sync.RWMutex, dbName string) (GitCommitLogRepository, error) {
	if m == nil {
		m = &sync.RWMutex{}
	}
	var err error
	sql := `
CREATE TABLE IF NOT EXISTS ` + sqlite3impl.QuoteIdent(dbName) + ` (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  COMMIT_MESSAGE NOT NULL,
  ADDITION NOT NULL,
  DELETION NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  REP_NAME NOT NULL,
  RELATED_TIME_UNIX NOT NULL,
  CREATE_TIME_UNIX NOT NULL,
  UPDATE_TIME_UNIX NOT NULL
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create git commit log table statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create git commit log table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_UNIX") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create git commit log index unix statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create git commit log git commit log index unix to %s: %w", dbName, err)
		return nil, err
	}

	return &gitCommitLogRepositoryCachedSQLite3Impl{
		dbName:   dbName,
		gitRep:   gitRep,
		cachedDB: cacheDB,
		m:        m,
	}, nil
}

// NewGitRepCachedSQLite3ImplPersistent Phase 1: GitCommitLog専用の永続ファイルベースSQLite DBを使うコンストラクタ
func NewGitRepCachedSQLite3ImplPersistent(ctx context.Context, gitRep GitCommitLogRepository, cacheDBPath string, dbName string, backgroundUpdate bool) (GitCommitLogRepository, error) {
	m := &sync.RWMutex{}

	db, err := sqllib.Open("sqlite", cacheDBPath+"?_txlock=immediate&_pragma=busy_timeout(6000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, fmt.Errorf("error at open persistent git commit log cache db: %w", err)
	}

	// キャッシュテーブル作成
	createSQL := `
CREATE TABLE IF NOT EXISTS ` + sqlite3impl.QuoteIdent(dbName) + ` (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  COMMIT_MESSAGE NOT NULL,
  ADDITION NOT NULL,
  DELETION NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  REP_NAME NOT NULL,
  RELATED_TIME_UNIX NOT NULL,
  CREATE_TIME_UNIX NOT NULL,
  UPDATE_TIME_UNIX NOT NULL
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", createSQL)
	_, err = db.ExecContext(ctx, createSQL)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("error at create git commit log cache table: %w", err)
	}

	// インデックス作成
	indexSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_UNIX") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexSQL)
	_, err = db.ExecContext(ctx, indexSQL)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("error at create git commit log cache index: %w", err)
	}

	// REF_HASHESテーブル作成（ref hashの永続化用）
	refHashesSQL := `
CREATE TABLE IF NOT EXISTS ` + sqlite3impl.QuoteIdent(dbName+"_REF_HASHES") + ` (
  REP_NAME TEXT NOT NULL,
  REF_NAME TEXT NOT NULL,
  REF_HASH TEXT NOT NULL,
  PRIMARY KEY (REP_NAME, REF_NAME)
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", refHashesSQL)
	_, err = db.ExecContext(ctx, refHashesSQL)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("error at create ref hashes table: %w", err)
	}

	return &gitCommitLogRepositoryCachedSQLite3Impl{
		dbName:           dbName,
		gitRep:           gitRep,
		cachedDB:         db,
		m:                m,
		ownDB:            true,
		backgroundUpdate: backgroundUpdate,
	}, nil
}

func (g *gitCommitLogRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	var err error
	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = g.UpdateCache(ctx)
		if err != nil {
			repName, _ := g.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	g.m.RLock()
	defer g.m.RUnlock()

	sql := `
SELECT
  IS_DELETED,
  ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(g.dbName) + `
WHERE
`

	dataType := "git_commit_log"
	queryArgs := []interface{}{
		dataType,
	}

	tableName := g.dbName
	tableNameAlias := g.dbName
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"COMMIT_MESSAGE"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := g.cachedDB.PrepareContext(ctx, sql)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from git commit log: %w", err)
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

			err = rows.Scan(&kyou.IsDeleted,
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
				err = fmt.Errorf("error at scan git commit log: %w", err)
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return kyous, nil
}

func (g *gitCommitLogRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	g.m.RLock()
	defer g.m.RUnlock()
	sql := `
SELECT
  IS_DELETED,
  ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(g.dbName) + `
WHERE
`
	dataType := "git_commit_log"

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs:         true,
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UseUpdateTime:  updateTime != nil,
		UpdateTime:     updateTime,
	}
	queryArgs := []interface{}{
		dataType,
	}

	tableName := g.dbName
	tableNameAlias := g.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"COMMIT_MESSAGE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := g.cachedDB.PrepareContext(ctx, sql)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from git commit log %s: %w", id, err)
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

			err = rows.Scan(&kyou.IsDeleted,
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
				err = fmt.Errorf("error at scan git commit log %s: %w", id, err)
				return nil, err
			}

			kyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			kyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			kyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
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
	return &kyous[0], nil
}

func (g *gitCommitLogRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	g.m.RLock()
	defer g.m.RUnlock()
	sql := `
SELECT
  IS_DELETED,
  ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(g.dbName) + `
WHERE
`
	dataType := "git_commit_log"

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: true,
		IDs:    ids,
	}
	queryArgs := []interface{}{
		dataType,
	}

	tableName := g.dbName
	tableNameAlias := g.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"COMMIT_MESSAGE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := g.cachedDB.PrepareContext(ctx, sql)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from git commit log %s: %w", id, err)
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

			err = rows.Scan(&kyou.IsDeleted,
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
				err = fmt.Errorf("error at scan git commit log %s: %w", id, err)
				return nil, err
			}

			kyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			kyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			kyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			kyous = append(kyous, kyou)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return kyous, nil
}

func (g *gitCommitLogRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return g.gitRep.GetPath(ctx, id)
}

// UpdateCache Phase 1+2: 永続キャッシュ＋差分更新
func (g *gitCommitLogRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	// Step 1: 下層リポジトリのref追跡を更新
	err := g.gitRep.UpdateCache(ctx)
	if err != nil {
		return fmt.Errorf("error at update underlying git commit log rep cache: %w", err)
	}

	// Step 2: 下層リポジトリに変更がなければスキップ
	if !g.gitRep.LastUpdateCacheChanged() {
		g.lastUpdateCacheChanged = false
		return nil
	}

	// Step 3: Phase 1 永続DB — 永続化されたref hashesと現在のref hashesを比較
	// 再起動後でも下層のLastUpdateCacheChangedは常にtrueを返すが、
	// 永続化されたref hashesが一致すればリビルドをスキップできる
	if g.ownDB {
		currentRefHashes := g.getCurrentRefHashes(ctx)
		persistedRefHashes, err := g.loadPersistedRefHashes(ctx)
		if err == nil && refHashesEqual(currentRefHashes, persistedRefHashes) {
			g.lastUpdateCacheChanged = false
			return nil
		}
	}

	// Step 4: Phase 2 差分更新 — キャッシュ済みIDと現在のIDを比較
	cachedIDs, err := g.getCachedCommitIDs(ctx)
	if err != nil {
		return fmt.Errorf("error at get cached commit ids: %w", err)
	}

	// 現在のコミットIDを高速に取得（StatsContextなし、FindKyousを使用）
	currentKyous, err := g.gitRep.FindKyous(ctx, &find.FindQuery{UpdateCache: false, OnlyLatestData: false})
	if err != nil {
		return fmt.Errorf("error at get current commit ids: %w", err)
	}
	currentIDs := make(map[string]bool, len(currentKyous))
	for id := range currentKyous {
		currentIDs[id] = true
	}

	// 差分計算: 新規コミット = 現在 - キャッシュ済み、削除コミット = キャッシュ済み - 現在
	var newIDs []string
	for id := range currentIDs {
		if !cachedIDs[id] {
			newIDs = append(newIDs, id)
		}
	}
	var deletedIDs []string
	for id := range cachedIDs {
		if !currentIDs[id] {
			deletedIDs = append(deletedIDs, id)
		}
	}

	// データ変更がなければref hashesだけ更新
	if len(newIDs) == 0 && len(deletedIDs) == 0 {
		g.lastUpdateCacheChanged = false
		if g.ownDB {
			g.saveRefHashes(ctx, g.getCurrentRefHashes(ctx))
		}
		return nil
	}

	// Phase 4: 初回フルリビルド（キャッシュ空）の場合はバックグラウンドで実行
	if g.backgroundUpdate && len(cachedIDs) == 0 && len(newIDs) > 0 {
		slog.Log(ctx, gkill_log.Info, "git commit log cache build starting in background", "numCommits", len(newIDs))
		currentRefHashes := g.getCurrentRefHashes(ctx)
		go func() {
			bgCtx := context.Background()
			err := g.doIncrementalUpdate(bgCtx, newIDs, deletedIDs, currentRefHashes)
			if err != nil {
				slog.Log(bgCtx, gkill_log.Warn, "error at background git commit log cache build", "error", err)
			} else {
				slog.Log(bgCtx, gkill_log.Info, "git commit log cache build completed", "numCommits", len(newIDs))
			}
		}()
		g.lastUpdateCacheChanged = true
		return nil
	}

	// 差分更新を実行
	currentRefHashes := g.getCurrentRefHashes(ctx)
	err = g.doIncrementalUpdate(ctx, newIDs, deletedIDs, currentRefHashes)
	if err != nil {
		return err
	}
	g.lastUpdateCacheChanged = true
	return nil
}

// doIncrementalUpdate 新規コミットのINSERTと削除コミットのDELETEを実行する
func (g *gitCommitLogRepositoryCachedSQLite3Impl) doIncrementalUpdate(ctx context.Context, newIDs []string, deletedIDs []string, currentRefHashes map[string]map[string]string) error {
	// 新規コミットのGitCommitLogを取得（StatsContext付き、Phase 3で並列実行）
	var newLogs []GitCommitLog
	if len(newIDs) > 0 {
		var err error
		newLogs, err = g.gitRep.FindGitCommitLogByIDs(ctx, newIDs)
		if err != nil {
			return fmt.Errorf("error at find git commit log by ids for incremental update: %w", err)
		}
	}

	g.m.Lock()
	defer g.m.Unlock()

	tx, err := g.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error at begin transaction for incremental update: %w", err)
	}
	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at incremental update", "error", err)
			}
		}
	}()

	// 削除コミットをDELETE
	if len(deletedIDs) > 0 {
		deleteSQL := `DELETE FROM ` + sqlite3impl.QuoteIdent(g.dbName) + ` WHERE ID = ?`
		deleteStmt, err := tx.PrepareContext(ctx, deleteSQL)
		if err != nil {
			return fmt.Errorf("error at prepare delete statement: %w", err)
		}
		defer deleteStmt.Close()

		for _, id := range deletedIDs {
			_, err = deleteStmt.ExecContext(ctx, id)
			if err != nil {
				return fmt.Errorf("error at delete git commit log %s: %w", id, err)
			}
		}
	}

	// 新規コミットをINSERT
	if len(newLogs) > 0 {
		insertSQL := `
INSERT INTO ` + sqlite3impl.QuoteIdent(g.dbName) + ` (
  IS_DELETED,
  ID,
  COMMIT_MESSAGE,
  ADDITION,
  DELETION,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  RELATED_TIME_UNIX,
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
  ?
)`
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", insertSQL)
		insertStmt, err := tx.PrepareContext(ctx, insertSQL)
		if err != nil {
			return fmt.Errorf("error at prepare insert statement: %w", err)
		}
		defer insertStmt.Close()

		for _, gitCommitLog := range newLogs {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			queryArgs := []interface{}{
				gitCommitLog.IsDeleted,
				gitCommitLog.ID,
				gitCommitLog.CommitMessage,
				gitCommitLog.Addition,
				gitCommitLog.Deletion,
				gitCommitLog.CreateApp,
				gitCommitLog.CreateDevice,
				gitCommitLog.CreateUser,
				gitCommitLog.UpdateApp,
				gitCommitLog.UpdateDevice,
				gitCommitLog.UpdateUser,
				gitCommitLog.RepName,
				gitCommitLog.RelatedTime.Unix(),
				gitCommitLog.CreateTime.Unix(),
				gitCommitLog.UpdateTime.Unix(),
			}
			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", insertSQL, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				return fmt.Errorf("error at insert git commit log %s: %w", gitCommitLog.ID, err)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error at commit transaction for incremental update: %w", err)
	}
	isCommitted = true

	// ref hashesを永続化
	if g.ownDB {
		g.saveRefHashes(ctx, currentRefHashes)
	}

	return nil
}

// getCurrentRefHashes 下層のlocal implから現在のref hashesを取得する
func (g *gitCommitLogRepositoryCachedSQLite3Impl) getCurrentRefHashes(ctx context.Context) map[string]map[string]string {
	result := map[string]map[string]string{}
	unwrapped, err := g.gitRep.UnWrapTyped()
	if err != nil {
		return result
	}
	for _, rep := range unwrapped {
		if localImpl, ok := rep.(*gitCommitLogRepositoryLocalImpl); ok {
			repName, _ := localImpl.GetRepName(ctx)
			if localImpl.lastHeadHashes != nil {
				result[repName] = localImpl.lastHeadHashes
			}
		}
	}
	return result
}

// loadPersistedRefHashes 永続DBからref hashesを読み込む
func (g *gitCommitLogRepositoryCachedSQLite3Impl) loadPersistedRefHashes(ctx context.Context) (map[string]map[string]string, error) {
	result := map[string]map[string]string{}
	tableName := sqlite3impl.QuoteIdent(g.dbName + "_REF_HASHES")

	rows, err := g.cachedDB.QueryContext(ctx, "SELECT REP_NAME, REF_NAME, REF_HASH FROM "+tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var repName, refName, refHash string
		if err := rows.Scan(&repName, &refName, &refHash); err != nil {
			return nil, err
		}
		if result[repName] == nil {
			result[repName] = map[string]string{}
		}
		result[repName][refName] = refHash
	}
	return result, rows.Err()
}

// saveRefHashes ref hashesを永続DBに保存する
func (g *gitCommitLogRepositoryCachedSQLite3Impl) saveRefHashes(ctx context.Context, refHashes map[string]map[string]string) {
	tableName := sqlite3impl.QuoteIdent(g.dbName + "_REF_HASHES")

	tx, err := g.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		slog.Log(ctx, gkill_log.Warn, "error at begin transaction for save ref hashes", "error", err)
		return
	}
	isCommitted := false
	defer func() {
		if !isCommitted {
			tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, "DELETE FROM "+tableName)
	if err != nil {
		slog.Log(ctx, gkill_log.Warn, "error at delete ref hashes", "error", err)
		return
	}

	insertSQL := "INSERT INTO " + tableName + " (REP_NAME, REF_NAME, REF_HASH) VALUES (?, ?, ?)"
	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		slog.Log(ctx, gkill_log.Warn, "error at prepare insert ref hashes", "error", err)
		return
	}
	defer stmt.Close()

	for repName, refs := range refHashes {
		for refName, refHash := range refs {
			_, err = stmt.ExecContext(ctx, repName, refName, refHash)
			if err != nil {
				slog.Log(ctx, gkill_log.Warn, "error at insert ref hash", "error", err)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Log(ctx, gkill_log.Warn, "error at commit ref hashes", "error", err)
		return
	}
	isCommitted = true
}

// getCachedCommitIDs キャッシュDBから全コミットIDを取得する
func (g *gitCommitLogRepositoryCachedSQLite3Impl) getCachedCommitIDs(ctx context.Context) (map[string]bool, error) {
	g.m.RLock()
	defer g.m.RUnlock()

	rows, err := g.cachedDB.QueryContext(ctx, "SELECT ID FROM "+sqlite3impl.QuoteIdent(g.dbName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

// refHashesEqual 2つのref hashesマップが同一かどうかを判定する
func refHashesEqual(a, b map[string]map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for repName, aRefs := range a {
		bRefs, ok := b[repName]
		if !ok {
			return false
		}
		if len(aRefs) != len(bRefs) {
			return false
		}
		for refName, aHash := range aRefs {
			if bRefs[refName] != aHash {
				return false
			}
		}
	}
	return true
}

func (g *gitCommitLogRepositoryCachedSQLite3Impl) LastUpdateCacheChanged() bool {
	return g.lastUpdateCacheChanged
}

func (g *gitCommitLogRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return g.gitRep.GetRepName(ctx)
}

func (g *gitCommitLogRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	g.m.Lock()
	defer g.m.Unlock()
	err := g.gitRep.Close(ctx)
	if err != nil {
		return err
	}
	// 永続DBの場合はDBをクローズするだけ（テーブルは永続化する）
	if g.ownDB {
		return g.cachedDB.Close()
	}
	if gkill_options.CacheGitCommitLogReps == nil || !*gkill_options.CacheGitCommitLogReps {
		err = g.cachedDB.Close()
		if err != nil {
			return err
		}
	} else {
		_, err = g.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+sqlite3impl.QuoteIdent(g.dbName))
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *gitCommitLogRepositoryCachedSQLite3Impl) FindGitCommitLog(ctx context.Context, query *find.FindQuery) ([]GitCommitLog, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = g.UpdateCache(ctx)
		if err != nil {
			repName, _ := g.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}
	g.m.RLock()
	defer g.m.RUnlock()

	sql := `
SELECT
  IS_DELETED,
  ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  COMMIT_MESSAGE,
  ADDITION,
  DELETION,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(g.dbName) + `
WHERE
`

	dataType := "git_commit_log"

	queryArgs := []interface{}{
		dataType,
	}

	tableName := g.dbName
	tableNameAlias := g.dbName
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"COMMIT_MESSAGE"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := g.cachedDB.PrepareContext(ctx, sql)
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from git commit log: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gitCommitLogs := []GitCommitLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			gitCommitLog := GitCommitLog{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&gitCommitLog.IsDeleted,
				&gitCommitLog.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&gitCommitLog.CreateApp,
				&gitCommitLog.CreateDevice,
				&gitCommitLog.CreateUser,
				&updateTimeUnix,
				&gitCommitLog.UpdateApp,
				&gitCommitLog.UpdateDevice,
				&gitCommitLog.UpdateUser,
				&gitCommitLog.CommitMessage,
				&gitCommitLog.Addition,
				&gitCommitLog.Deletion,
				&gitCommitLog.RepName,
				&gitCommitLog.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan git commit log: %w", err)
				return nil, err
			}

			gitCommitLog.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			gitCommitLog.CreateTime = time.Unix(createTimeUnix, 0).Local()
			gitCommitLog.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			gitCommitLogs = append(gitCommitLogs, gitCommitLog)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return gitCommitLogs, nil
}

// FindGitCommitLogByIDs キャッシュDBから指定IDのGitCommitLogを取得する
func (g *gitCommitLogRepositoryCachedSQLite3Impl) FindGitCommitLogByIDs(ctx context.Context, ids []string) ([]GitCommitLog, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	g.m.RLock()
	defer g.m.RUnlock()

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	sql := `
SELECT
  IS_DELETED,
  ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  COMMIT_MESSAGE,
  ADDITION,
  DELETION,
  REP_NAME,
  'git_commit_log' AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(g.dbName) + `
WHERE ID IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := g.cachedDB.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("error at find git commit log by ids: %w", err)
	}
	defer rows.Close()

	var gitCommitLogs []GitCommitLog
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			gitCommitLog := GitCommitLog{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&gitCommitLog.IsDeleted,
				&gitCommitLog.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&gitCommitLog.CreateApp,
				&gitCommitLog.CreateDevice,
				&gitCommitLog.CreateUser,
				&updateTimeUnix,
				&gitCommitLog.UpdateApp,
				&gitCommitLog.UpdateDevice,
				&gitCommitLog.UpdateUser,
				&gitCommitLog.CommitMessage,
				&gitCommitLog.Addition,
				&gitCommitLog.Deletion,
				&gitCommitLog.RepName,
				&gitCommitLog.DataType,
			)
			if err != nil {
				return nil, fmt.Errorf("error at scan git commit log by ids: %w", err)
			}

			gitCommitLog.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			gitCommitLog.CreateTime = time.Unix(createTimeUnix, 0).Local()
			gitCommitLog.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			gitCommitLogs = append(gitCommitLogs, gitCommitLog)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error at iterate rows: %w", err)
	}
	return gitCommitLogs, nil
}

func (g *gitCommitLogRepositoryCachedSQLite3Impl) GetGitCommitLog(ctx context.Context, id string, updateTime *time.Time) (*GitCommitLog, error) {
	g.m.RLock()
	defer g.m.RUnlock()
	sql := `
SELECT
  IS_DELETED,
  ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  COMMIT_MESSAGE,
  ADDITION,
  DELETION,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(g.dbName) + `
WHERE
`

	ids := []string{id}
	query := &find.FindQuery{
		UseIDs:         true,
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UseUpdateTime:  updateTime != nil,
		UpdateTime:     updateTime,
	}
	dataType := "git_commit_log"

	queryArgs := []interface{}{
		dataType,
	}

	tableName := g.dbName
	tableNameAlias := g.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"COMMIT_MESSAGE"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := g.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get git_commit_log histories sql %s: %w", id, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
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

	gitCommitLog := []GitCommitLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			gitCommitLoig := GitCommitLog{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&gitCommitLoig.IsDeleted,
				&gitCommitLoig.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&gitCommitLoig.CreateApp,
				&gitCommitLoig.CreateDevice,
				&gitCommitLoig.CreateUser,
				&updateTimeUnix,
				&gitCommitLoig.UpdateApp,
				&gitCommitLoig.UpdateDevice,
				&gitCommitLoig.UpdateUser,
				&gitCommitLoig.CommitMessage,
				&gitCommitLoig.Addition,
				&gitCommitLoig.Deletion,
				&gitCommitLoig.RepName,
				&gitCommitLoig.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan git_commit_log %s: %w", id, err)
				return nil, err
			}

			gitCommitLoig.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			gitCommitLoig.CreateTime = time.Unix(createTimeUnix, 0).Local()
			gitCommitLoig.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			gitCommitLog = append(gitCommitLog, gitCommitLoig)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(gitCommitLog) == 0 {
		return nil, nil
	}
	return &gitCommitLog[0], nil
}

func (g *gitCommitLogRepositoryCachedSQLite3Impl) UnWrapTyped() ([]GitCommitLogRepository, error) {
	return g.gitRep.UnWrapTyped()
}

func (g *gitCommitLogRepositoryCachedSQLite3Impl) UnWrap() ([]Repository, error) {
	return g.gitRep.UnWrap()
}

func (g *gitCommitLogRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	return g.gitRep.GetLatestDataRepositoryAddress(ctx, updateCache)
}
