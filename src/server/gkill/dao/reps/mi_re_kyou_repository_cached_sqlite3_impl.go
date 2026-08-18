package reps

import (
	"context"
	sqllib "database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	_ "modernc.org/sqlite"
)

// miReKyouSequentialRepository は入れ子でのスレッドプール枯渇を避けるための
// 逐次版メソッドを持つ集約リポジトリです。
//
// キャッシュ実装は下層集約への委譲を必ずこのインタフェース経由で行ってください。
// threads.Goはプールのスロットを呼び出し元goroutineで同期取得するため、
// スロットを保持したまま集約の並列メソッドを呼ぶとプールが枯渇します。
type miReKyouSequentialRepository interface {
	GetKyouSequential(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)
	GetKyouHistoriesSequential(ctx context.Context, id string) ([]Kyou, error)
	GetPathSequential(ctx context.Context, id string) (string, error)
	UpdateCacheSequential(ctx context.Context) error
	GetMiReKyouSequential(ctx context.Context, id string, updateTime *time.Time) (*MiReKyou, error)
	GetMiReKyouHistoriesSequential(ctx context.Context, id string) ([]MiReKyou, error)
	GetMiReKyousAllLatestSequential(ctx context.Context) ([]MiReKyou, error)
}

// 集約が逐次版を持つことをコンパイル時に固定する。
// メソッド名を変えたり消したりしたらここでビルドが壊れる。
var _ miReKyouSequentialRepository = (*MiReKyouRepositories)(nil)

// miReKyouRepositoryCachedSQLite3Impl はMiReKyouをインメモリDBにキャッシュするリポジトリです。
// キャッシュテーブルは実体テーブルと同じ列名・同じ時刻フォーマットで作るため、
// 検索SQLはテーブル名を差し替えるだけで実体と共有できます。
// キャッシュにはID毎の最新版のみを持つため、履歴取得は下層リポジトリへ委譲します。
type miReKyouRepositoryCachedSQLite3Impl struct {
	dbName      string
	mirekyouRep MiReKyouRepository
	// sequentialRep は mirekyouRep が集約だった場合の逐次版インタフェースです。
	// 単一リポジトリやTX中の一時リポジトリを渡された場合はnilになり、
	// そのときは入れ子にならないので mirekyouRep をそのまま呼びます。
	sequentialRep     miReKyouSequentialRepository
	cachedDB          *sqllib.DB
	m                 *sync.RWMutex
	gkillRepositories *GkillRepositories
}

func NewMiReKyouRepositoryCachedSQLite3Impl(ctx context.Context, mirekyouRep MiReKyouRepository, gkillRepositories *GkillRepositories, cacheDB *sqllib.DB, m *sync.RWMutex, dbName string) (MiReKyouRepository, error) {
	if m == nil {
		m = &sync.RWMutex{}
	}

	sql := `CREATE TABLE IF NOT EXISTS ` + sqlite3impl.QuoteIdent(dbName) + ` (` + miReKyouColumns + `,
  REP_NAME NOT NULL
);`
	gkill_log.LogSQL(ctx, sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create MIREKYOU cache table statement %s: %w", dbName, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MIREKYOU cache table to %s: %w", dbName, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName) + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(ID, UPDATE_TIME);`
	gkill_log.LogIndexSQL(ctx, indexSQL)
	indexStmt, err := cacheDB.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create MIREKYOU cache index statement %s: %w", dbName, err)
		return nil, err
	}
	defer func() {
		err := indexStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MIREKYOU cache index to %s: %w", dbName, err)
		return nil, err
	}

	// このキャッシュ表だけ時刻が RFC3339 の文字列のままで、5つの射影は
	// `unixepoch(<列>)` で絞り・並べる(mi_re_kyou_sql.go)。式索引が無いと5回とも全表走査になる。
	// MI では既に同じ手当てが入っている(mi_repository_cached_sqlite3_impl.go)。
	//
	// 式は GenerateFindSQLCommon が出すものとバイト単位で一致していないと黙って効かなくなる
	// (sqlite3impl_util.go の EnsureUnixepochIndex の注記を参照)。
	if err := sqlite3impl.EnsureUnixepochIndex(ctx, cacheDB, dbName,
		"CREATE_TIME", "LIMIT_TIME", "ESTIMATE_START_TIME", "ESTIMATE_END_TIME"); err != nil {
		return nil, err
	}

	// 下層が集約なら逐次版を掴んでおく。以降の委譲は必ずこちらを使う。
	sequentialRep, _ := mirekyouRep.(miReKyouSequentialRepository)

	return &miReKyouRepositoryCachedSQLite3Impl{
		dbName:            dbName,
		mirekyouRep:       mirekyouRep,
		sequentialRep:     sequentialRep,
		cachedDB:          cacheDB,
		m:                 m,
		gkillRepositories: gkillRepositories,
	}, nil
}

// tableName はキャッシュテーブル名をクォートして返します。
func (m *miReKyouRepositoryCachedSQLite3Impl) tableName() string {
	return sqlite3impl.QuoteIdent(m.dbName)
}

func (m *miReKyouRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	if query.UpdateCache {
		err := m.UpdateCache(ctx)
		if err != nil {
			return nil, fmt.Errorf("error at update cache at find kyous: %w", err)
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

	repName, err := m.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	m.m.RLock()
	defer m.m.RUnlock()

	sql, queryArgs, err := buildMiReKyouKyouSQL(query, repName, m.tableName(), false, nil)
	if err != nil {
		return nil, err
	}
	kyousList, err := m.queryKyous(ctx, sql, queryArgs, repName)
	if err != nil {
		return nil, err
	}

	// ターゲット解決をしないなら ID→TARGET_ID の対応も要らない。
	// getTargetIDMapWithoutLock は WHERE の無い全表スキャンなので、
	// 使わない場合に引くと丸ごと無駄になる。
	targetIDMap := map[string]string{}
	if !targetFilter.allowAll {
		targetIDMap, err = m.getTargetIDMapWithoutLock(ctx)
		if err != nil {
			return nil, err
		}
	}

	// アドレス表の読み取りロックは行ごとではなく1回だけ取る
	releaseTargetAddressRead := targetFilter.beginTargetAddressRead()
	defer releaseTargetAddressRead()

	kyous := map[string][]Kyou{}
	for _, kyou := range kyousList {
		// ターゲットが存在しない、またはワード検索にヒットしないものは除外する
		if !targetFilter.isMatch(targetIDMap[kyou.ID]) {
			continue
		}
		// 空スライスの事前確保はしない。存在しないキーへのappendはnilスライスに対して働くので
		// 結果は同じで、レコード1件につき1回の無駄な確保(実データで56万回)が消える。
		// 同じ整理は dao/reps/repositories.go の集約側では既に済んでいる。
		kyous[kyou.ID] = append(kyous[kyou.ID], kyou)
	}
	return kyous, nil
}

// queryKyous はキャッシュテーブルへKyou取得SQLを実行します。
func (m *miReKyouRepositoryCachedSQLite3Impl) queryKyous(ctx context.Context, sql string, queryArgs []any, repName string) ([]Kyou, error) {
	if sql == "" {
		return []Kyou{}, nil
	}
	gkill_log.LogSQLParams(ctx, sql, queryArgs)
	rows, err := m.cachedDB.QueryContext(ctx, sql, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from mirekyou cache: %w", err)
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

// queryMiReKyous はキャッシュテーブルへMiReKyou取得SQLを実行します。
func (m *miReKyouRepositoryCachedSQLite3Impl) queryMiReKyous(ctx context.Context, sql string, queryArgs []any, repName string) ([]MiReKyou, error) {
	if sql == "" {
		return []MiReKyou{}, nil
	}
	gkill_log.LogSQLParams(ctx, sql, queryArgs)
	rows, err := m.cachedDB.QueryContext(ctx, sql, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from mirekyou cache: %w", err)
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

// getTargetIDMapWithoutLock はキャッシュからIDとTARGET_IDの対応を取得します。
func (m *miReKyouRepositoryCachedSQLite3Impl) getTargetIDMapWithoutLock(ctx context.Context) (map[string]string, error) {
	sql := `SELECT ID, TARGET_ID FROM ` + m.tableName()
	gkill_log.LogSQL(ctx, sql)
	rows, err := m.cachedDB.QueryContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at select target id from mirekyou cache: %w", err)
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
				err = fmt.Errorf("error at scan target id in mirekyou cache: %w", err)
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

// GetKyou はKyouを1件取得します。履歴を含む取得は下層リポジトリへ委譲します。
//
// 呼び出し元がthreads.Goのスロットを保持しているので、集約へは逐次版で逃がします。
// 並列版を呼ぶとスロットを保持したまま子のスロットを待つ入れ子になり、プールが枯渇します。
func (m *miReKyouRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	if m.sequentialRep != nil {
		return m.sequentialRep.GetKyouSequential(ctx, id, updateTime)
	}
	return m.mirekyouRep.GetKyou(ctx, id, updateTime)
}

// GetKyouHistories は履歴を取得します。キャッシュは最新版のみ持つため下層へ委譲します。
// 委譲が逐次版である理由はGetKyouと同じです。
func (m *miReKyouRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	if m.sequentialRep != nil {
		return m.sequentialRep.GetKyouHistoriesSequential(ctx, id)
	}
	return m.mirekyouRep.GetKyouHistories(ctx, id)
}

func (m *miReKyouRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if m.sequentialRep != nil {
		return m.sequentialRep.GetPathSequential(ctx, id)
	}
	return m.mirekyouRep.GetPath(ctx, id)
}

func (m *miReKyouRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	if m.mirekyouRep == nil {
		return fmt.Errorf("underlying mirekyou rep is nil")
	}
	if hasRecursiveMiReKyouRepReference(m.mirekyouRep, m, map[*MiReKyouRepositories]struct{}{}) {
		return fmt.Errorf("detected recursive mirekyou cache reference")
	}

	// 下層への委譲は逐次版で行う。UpdateCacheはthreads.Goのスロットを
	// 保持した状態から呼ばれうるため、並列版だと入れ子でプールが枯渇する。
	if m.sequentialRep != nil {
		err := m.sequentialRep.UpdateCacheSequential(ctx)
		if err != nil {
			return fmt.Errorf("error at update underlying mirekyou rep cache: %w", err)
		}
	} else {
		err := m.mirekyouRep.UpdateCache(ctx)
		if err != nil {
			return fmt.Errorf("error at update underlying mirekyou rep cache: %w", err)
		}
	}

	// 下層リポジトリに変更がなければフルリビルドをスキップ
	if !m.mirekyouRep.LastUpdateCacheChanged() {
		return nil
	}

	// ターゲット解決前の生データをキャッシュする。
	// ターゲットの存在確認は読み出しの都度行うため、対象が消えたMiReKyouも保持しておく。
	var allMiReKyous []MiReKyou
	var err error
	if m.sequentialRep != nil {
		allMiReKyous, err = m.sequentialRep.GetMiReKyousAllLatestSequential(ctx)
	} else {
		allMiReKyous, err = m.mirekyouRep.GetMiReKyousAllLatest(ctx)
	}
	if err != nil {
		err = fmt.Errorf("error at get all mirekyou at update cache: %w", err)
		return err
	}

	m.m.Lock()
	defer m.m.Unlock()

	tx, err := m.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add mirekyou: %w", err)
		return err
	}

	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at update cache", "error", fmt.Sprintf("%q", err))
			}
		}
	}()

	deleteSQL := `DELETE FROM ` + m.tableName()
	_, err = tx.ExecContext(ctx, deleteSQL)
	if err != nil {
		err = fmt.Errorf("error at delete mirekyou cache table: %w", err)
		return err
	}

	insertSQL := `INSERT INTO ` + m.tableName() + ` (` + miReKyouInsertColumnNames + `, REP_NAME) VALUES (` + miReKyouInsertPlaceHolders + `, ?)`
	gkill_log.LogSQL(ctx, insertSQL)
	insertStmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		err = fmt.Errorf("error at add mirekyou sql: %w", err)
		return err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for _, mirekyou := range allMiReKyous {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		queryArgs := append(miReKyouInsertArgs(mirekyou), mirekyou.RepName)
		_, err = insertStmt.ExecContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at insert in to mirekyou cache %s: %w", mirekyou.ID, err)
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit transaction for add mirekyou: %w", err)
		return err
	}
	isCommitted = true
	return nil
}

// hasRecursiveMiReKyouRepReference は自己参照によるキャッシュ更新の無限再帰を検出します。
func hasRecursiveMiReKyouRepReference(rep MiReKyouRepository, self *miReKyouRepositoryCachedSQLite3Impl, visited map[*MiReKyouRepositories]struct{}) bool {
	switch typed := rep.(type) {
	case *miReKyouRepositoryCachedSQLite3Impl:
		return typed == self
	case *MiReKyouRepositories:
		if _, exist := visited[typed]; exist {
			return false
		}
		visited[typed] = struct{}{}
		for _, nested := range typed.MiReKyouRepositories {
			if hasRecursiveMiReKyouRepReference(nested, self, visited) {
				return true
			}
		}
	}
	return false
}

func (m *miReKyouRepositoryCachedSQLite3Impl) LastUpdateCacheChanged() bool {
	return true
}

func (m *miReKyouRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return m.mirekyouRep.GetRepName(ctx)
}

func (m *miReKyouRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	m.m.Lock()
	defer m.m.Unlock()
	errs := []error{}
	if m.mirekyouRep != nil {
		if err := m.mirekyouRep.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if gkill_options.CacheMiReKyouReps != nil && *gkill_options.CacheMiReKyouReps {
		if _, err := m.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+m.tableName()); err != nil {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}
	if err := m.cachedDB.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (m *miReKyouRepositoryCachedSQLite3Impl) FindMiReKyou(ctx context.Context, query *find.FindQuery) ([]MiReKyou, error) {
	if query.UpdateCache {
		err := m.UpdateCache(ctx)
		if err != nil {
			return nil, fmt.Errorf("error at update cache at find mirekyou: %w", err)
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

	repName, err := m.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	m.m.RLock()
	defer m.m.RUnlock()

	sql, queryArgs, err := buildMiReKyouSQL(query, repName, m.tableName(), true, nil)
	if err != nil {
		return nil, err
	}
	mirekyous, err := m.queryMiReKyous(ctx, sql, queryArgs, repName)
	if err != nil {
		return nil, err
	}

	filteredMiReKyous := []MiReKyou{}
	for _, mirekyou := range mirekyous {
		if !targetFilter.isMatch(mirekyou.TargetID) {
			continue
		}
		filteredMiReKyous = append(filteredMiReKyous, mirekyou)
	}
	return filteredMiReKyous, nil
}

// GetMiReKyou はMiReKyouを1件取得します。履歴を含む取得は下層リポジトリへ委譲します。
// 委譲が逐次版である理由はGetKyouと同じです。
func (m *miReKyouRepositoryCachedSQLite3Impl) GetMiReKyou(ctx context.Context, id string, updateTime *time.Time) (*MiReKyou, error) {
	if m.sequentialRep != nil {
		return m.sequentialRep.GetMiReKyouSequential(ctx, id, updateTime)
	}
	return m.mirekyouRep.GetMiReKyou(ctx, id, updateTime)
}

// GetMiReKyouHistories は履歴を取得します。キャッシュは最新版のみ持つため下層へ委譲します。
// 委譲が逐次版である理由はGetKyouと同じです。
func (m *miReKyouRepositoryCachedSQLite3Impl) GetMiReKyouHistories(ctx context.Context, id string) ([]MiReKyou, error) {
	if m.sequentialRep != nil {
		return m.sequentialRep.GetMiReKyouHistoriesSequential(ctx, id)
	}
	return m.mirekyouRep.GetMiReKyouHistories(ctx, id)
}

func (m *miReKyouRepositoryCachedSQLite3Impl) AddMiReKyouInfo(ctx context.Context, mirekyou MiReKyou) error {
	repName, err := m.GetRepName(ctx)
	if err != nil {
		return err
	}
	if mirekyou.RepName == "" {
		mirekyou.RepName = repName
	}

	m.m.Lock()
	defer m.m.Unlock()

	tx, err := m.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add mirekyou: %w", err)
		return err
	}
	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at add mirekyou", "error", fmt.Sprintf("%q", err))
			}
		}
	}()

	// キャッシュはID毎の最新版のみを持つため、既存行を置き換える
	deleteSQL := `DELETE FROM ` + m.tableName() + ` WHERE ID = ?`
	_, err = tx.ExecContext(ctx, deleteSQL, mirekyou.ID)
	if err != nil {
		err = fmt.Errorf("error at delete old mirekyou cache %s: %w", mirekyou.ID, err)
		return err
	}

	insertSQL := `INSERT INTO ` + m.tableName() + ` (` + miReKyouInsertColumnNames + `, REP_NAME) VALUES (` + miReKyouInsertPlaceHolders + `, ?)`
	queryArgs := append(miReKyouInsertArgs(mirekyou), mirekyou.RepName)
	gkill_log.LogSQLParams(ctx, insertSQL, queryArgs)
	_, err = tx.ExecContext(ctx, insertSQL, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to mirekyou cache %s: %w", mirekyou.ID, err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit transaction for add mirekyou: %w", err)
		return err
	}
	isCommitted = true
	return nil
}

func (m *miReKyouRepositoryCachedSQLite3Impl) GetMiReKyousAllLatest(ctx context.Context) ([]MiReKyou, error) {
	repName, err := m.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	m.m.RLock()
	defer m.m.RUnlock()

	sql, queryArgs, err := buildMiReKyouSingleProjectionSQL(&find.FindQuery{}, repName, m.tableName(), true)
	if err != nil {
		return nil, err
	}
	return m.queryMiReKyous(ctx, sql, queryArgs, repName)
}

func (m *miReKyouRepositoryCachedSQLite3Impl) GetBoardNames(ctx context.Context) ([]string, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return queryMiReKyouBoardNames(ctx, m.cachedDB, m.tableName())
}

func (m *miReKyouRepositoryCachedSQLite3Impl) GetRepositoriesWithoutMiReKyouRep(ctx context.Context) (*GkillRepositories, error) {
	mirekyouReps := MiReKyouRepositories{
		MiReKyouRepositories: nil,
		GkillRepositories:    m.gkillRepositories,
	}
	return mirekyouReps.GetRepositoriesWithoutMiReKyouRep(ctx)
}

func (m *miReKyouRepositoryCachedSQLite3Impl) UnWrapTyped() ([]MiReKyouRepository, error) {
	return []MiReKyouRepository{m}, nil
}

func (m *miReKyouRepositoryCachedSQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{m}, nil
}

func (m *miReKyouRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	if updateCache {
		err := m.UpdateCache(ctx)
		if err != nil {
			return nil, fmt.Errorf("error at update cache at get latest data repository address: %w", err)
		}
	}

	repName, err := m.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	m.m.RLock()
	defer m.m.RUnlock()

	return queryMiReKyouLatestDataRepositoryAddress(ctx, m.cachedDB, m.tableName(), repName)
}
