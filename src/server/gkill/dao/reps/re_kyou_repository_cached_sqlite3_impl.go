package reps

import (
	"context"
	sqllib "database/sql"
	"errors"
	"fmt"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	_ "modernc.org/sqlite"
)

type reKyouRepositoryCachedSQLite3Impl struct {
	dbName            string
	rekyouRep         ReKyouRepository
	cachedDB          *sqllib.DB
	m                 *sync.RWMutex
	gkillRepositories *GkillRepositories
	addReKyouInfoSQL  string
	addReKyouInfoStmt *sqllib.Stmt
}

func NewReKyouRepositoryCachedSQLite3Impl(ctx context.Context, rekyouRep ReKyouRepository, gkillRepositories *GkillRepositories, cacheDB *sqllib.DB, m *sync.RWMutex, dbName string) (ReKyouRepository, error) {
	if m == nil {
		m = &sync.RWMutex{}
	}
	var err error
	sql := `
CREATE TABLE IF NOT EXISTS ` + sqlite3impl.QuoteIdent(dbName) + ` (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
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
	gkill_log.LogSQL(ctx, sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU table statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create REKYOU table to %s: %w", dbName, err)
		return nil, err
	}

	gkill_log.LogSQL(ctx, sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_UNIX") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	gkill_log.LogSQL(ctx, indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create rekyou index unix statement %s: %w", dbName, err)
		return nil, err
	}
	defer func() {
		err := indexUnixStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gkill_log.LogSQL(ctx, indexUnixSQL)
	_, err = indexUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create rekyou index unix to %s: %w", dbName, err)
		return nil, err
	}

	// 既存索引は先頭が ID なので時刻範囲にもORDER BYにも使えない。
	// 時刻列を先頭にした索引を別途張る。
	if err := sqlite3impl.EnsureUnixColumnIndex(ctx, cacheDB, dbName, "RELATED_TIME_UNIX"); err != nil {
		return nil, err
	}

	addReKyouInfoSQL := `
INSERT INTO ` + sqlite3impl.QuoteIdent(dbName) + ` (
  IS_DELETED,
  ID,
  TARGET_ID,
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
  ?
)`
	gkill_log.LogSQL(ctx, addReKyouInfoSQL)
	addReKyouInfoStmt, err := cacheDB.PrepareContext(ctx, addReKyouInfoSQL)
	if err != nil {
		err = fmt.Errorf("error at add rekyou info sql: %w", err)
		return nil, err
	}

	return &reKyouRepositoryCachedSQLite3Impl{
		dbName:            dbName,
		rekyouRep:         rekyouRep,
		cachedDB:          cacheDB,
		m:                 m,
		gkillRepositories: gkillRepositories,
		addReKyouInfoSQL:  addReKyouInfoSQL,
		addReKyouInfoStmt: addReKyouInfoStmt,
	}, nil
}
func (r *reKyouRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	// ここでr.mを取ってはいけない。DBへは自分でロックを取るヘルパー
	// (GetReKyousAllLatest・LatestDataRepositoryAddressDAO・FindKyousSequential)越しにしか触れないため、
	// 保持したまま呼ぶと同一goroutineの再帰RLockになり、UpdateCacheのLock待ちと交差した時点でデッドロックする
	matchKyous := map[string][]Kyou{}

	// 未削除ReKyouを抽出
	notDeletedAllReKyous := []ReKyou{}
	allReKyous, err := r.GetReKyousAllLatest(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rekyous all latest :%w", err)
		return nil, err
	}
	for _, rekyou := range allReKyous {
		if !rekyou.IsDeleted {
			notDeletedAllReKyous = append(notDeletedAllReKyous, rekyou)
		}
	}

	repsWithoutRekyou, err := r.GetRepositoriesWithoutReKyouRep(ctx)
	if err != nil {
		err = fmt.Errorf("error at get without rekyou: %w", err)
		return nil, err
	}

	// リポジトリ群を辿れない場合はターゲット解決を行わずすべて通す（MiReKyouのallowAllと同じ扱い）
	allowAllTargets := repsWithoutRekyou == nil

	if !allowAllTargets {
		if err := repsWithoutRekyou.EnsureLatestDataRepositoryAddresses(ctx); err != nil {
			err = fmt.Errorf("error at get all latest data repository addresses: %w", err)
			return nil, err
		}
	}

	// ワードフィルタ: ワード指定が非nilの場合、Targetに対してワード検索を実行しマッチしたIDを収集する
	wordMatchTargetIDs := map[string]bool{}
	useWordFilter := !allowAllTargets && isWordFilterEnabled(query)
	if useWordFilter {
		wordMatchTargetIDs, err = resolveReKyouWordMatchTargetIDs(ctx, repsWithoutRekyou.collectTargetDataRepositories(), repsWithoutRekyou.collectMiReKyouRepositories(), query)
		if err != nil {
			return nil, err
		}
	}

	// 最新版アドレスはループ前に1回だけロックを取ってまとめて引く。
	// 1件ごとに GetLatestDataRepositoryAddress を呼ぶと、遅延初期化のプロセス共有mutexと
	// 読み取りロックをReKyouの件数ぶん取り直すことになる。
	// このループの中からアドレス表を触る他のメソッドを呼んではいけない(再帰RLock)。
	var latestDataAddressReader LatestDataRepositoryAddressReader
	if !allowAllTargets {
		reader, releaseLatestDataAddressRead := repsWithoutRekyou.BeginLatestDataRepositoryAddressRead()
		defer releaseLatestDataAddressRead()
		latestDataAddressReader = reader
	}

	// ID指定はループの外で集合にしておく。1件ごとに query.IDs を線形走査すると、
	// ワード一致経路(find_filter.findKyous)が数千IDを渡してくるので
	// 実質 O(ReKyou数 × ID数) になる。
	// nil=ID指定なし(全通し) / 非nilの空=0件指定、という FindQuery の意味論はそのまま。
	var queryIDSet map[string]struct{}
	if query.IDs != nil {
		queryIDSet = make(map[string]struct{}, len(query.IDs))
		for _, id := range query.IDs {
			queryIDSet[id] = struct{}{}
		}
	}

	for _, rekyou := range notDeletedAllReKyous {
		// allowAllTargetsのとき repsWithoutRekyou はnilなので触ってはいけない
		existInRep := allowAllTargets
		if !allowAllTargets {
			latestDataRepositoryAddress, existAddress := latestDataAddressReader.Get(rekyou.TargetID)
			existInRep = existAddress && !latestDataRepositoryAddress.IsDeleted
		}

		matchID := true
		if queryIDSet != nil {
			_, matchID = queryIDSet[rekyou.ID]
		}
		if !matchID {
			continue
		}

		// ワードフィルタが有効な場合、TargetIDがマッチしなければスキップ
		if useWordFilter {
			if !wordMatchTargetIDs[rekyou.TargetID] {
				continue
			}
		}

		if existInRep {
			kyou := Kyou{}
			kyou.IsDeleted = rekyou.IsDeleted
			kyou.ID = rekyou.ID
			kyou.RepName = rekyou.RepName
			kyou.RelatedTime = rekyou.RelatedTime
			kyou.DataType = rekyou.DataType
			kyou.CreateTime = rekyou.CreateTime
			kyou.CreateApp = rekyou.CreateApp
			kyou.CreateDevice = rekyou.CreateDevice
			kyou.CreateUser = rekyou.CreateUser
			kyou.UpdateTime = rekyou.UpdateTime
			kyou.UpdateApp = rekyou.UpdateApp
			kyou.UpdateUser = rekyou.UpdateUser
			kyou.UpdateDevice = rekyou.UpdateDevice

			// 空スライスの事前確保はしない。存在しないキーへのappendはnilスライスに対して働くので
			// 結果は同じで、レコード1件につき1回の無駄な確保が消える(repositories.goと同じ)。
			matchKyous[kyou.ID] = append(matchKyous[kyou.ID], kyou)
		}
	}
	return matchKyous, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	r.m.RLock()
	defer r.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(r.dbName) + `
WHERE 
`
	dataType := "rekyou"

	ids := []string{id}
	query := &find.FindQuery{
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UpdateTime:     updateTime,
	}
	queryArgs := []any{
		dataType,
	}

	tableName := r.dbName
	tableNameAlias := r.dbName
	whereCounter := 0
	// GenerateFindSQLCommon は query.OnlyLatestData を読まず、この引数しか見ない。
	// false のままだと updateTime 未指定のときに **そのIDの全バージョンを無順序・無制限に読み**、
	// 下の kyous[0] が格納順の先頭(多くの場合いちばん古い版)を返してしまう。
	// 版の数だけ走査するので遅くもある。Tag / Text では既に同じ修正が入っている。
	//
	// 値は上の query リテラルで `OnlyLatestData: updateTime == nil` として組み立てた
	// ものをそのまま使う。かつては同じ式をここでもう一度書いたうえで
	// `onlyLatestData = query.OnlyLatestData` で上書きしていた（値は同じなので
	// 挙動は変わらないが、最初の代入が死んでいて、この説明が捨てられる行に
	// 付いている状態だった）。
	onlyLatestData := query.OnlyLatestData
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false

	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
	stmt, err := r.cachedDB.PrepareContext(ctx, sql)
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
		err = fmt.Errorf("error at select from REKYOU %s: %w", id, err)
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
				err = fmt.Errorf("error at scan from REKYOU %s: %w", id, err)
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
	// 最新版に絞ってもrepをまたいだ同一版が複数返りうるので、UpdateTimeが最大のものを選ぶ。
	// 格納順の先頭を返すと、どれが返るかがSQLiteの都合で決まってしまう。
	latestKyou := slices.MaxFunc(kyous, func(a Kyou, b Kyou) int {
		return a.UpdateTime.Compare(b.UpdateTime)
	})
	return &latestKyou, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	r.m.RLock()
	defer r.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(r.dbName) + `
WHERE 
`
	dataType := "rekyou"

	ids := []string{id}
	query := &find.FindQuery{
		IDs: ids,
	}
	queryArgs := []any{
		dataType,
	}

	tableName := r.dbName
	tableNameAlias := r.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
	stmt, err := r.cachedDB.PrepareContext(ctx, sql)
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
		err = fmt.Errorf("error at select from REKYOU %s: %w", id, err)
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
				err = fmt.Errorf("error at scan from REKYOU %s: %w", id, err)
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

func (r *reKyouRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return r.dbName, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	if r.rekyouRep == nil {
		return fmt.Errorf("underlying rekyou rep is nil")
	}
	if hasRecursiveReKyouRepReference(r.rekyouRep, r, map[*ReKyouRepositories]struct{}{}) {
		return fmt.Errorf("detected recursive rekyou cache reference")
	}

	err := r.rekyouRep.UpdateCache(ctx)
	if err != nil {
		return fmt.Errorf("error at update underlying rekyou rep cache: %w", err)
	}

	// 下層リポジトリに変更がなければフルリビルドをスキップ
	if !r.rekyouRep.LastUpdateCacheChanged() {
		return nil
	}

	query := &find.FindQuery{
		UpdateCache:    false,
		OnlyLatestData: false,
	}

	allReKyous, err := r.rekyouRep.FindReKyou(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all rekyou at update cache: %w", err)
		return err
	}

	r.m.Lock()
	defer r.m.Unlock()

	tx, err := r.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add rekyou: %w", err)
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

	sql := `DELETE FROM ` + sqlite3impl.QuoteIdent(r.dbName)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU table statement %s: %w", "memory", err)
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
		err = fmt.Errorf("error at delete REKYOU table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + sqlite3impl.QuoteIdent(r.dbName) + ` (
  IS_DELETED,
  ID,
  TARGET_ID,
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
  ?
)`

	gkill_log.LogSQL(ctx, sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add rekyou sql: %w", err)
		return err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for _, rekyou := range allReKyous {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			queryArgs := []any{
				rekyou.IsDeleted,
				rekyou.ID,
				rekyou.TargetID,
				rekyou.CreateApp,
				rekyou.CreateDevice,
				rekyou.CreateUser,
				rekyou.UpdateApp,
				rekyou.UpdateDevice,
				rekyou.UpdateUser,
				rekyou.RepName,
				rekyou.RelatedTime.Unix(),
				rekyou.CreateTime.Unix(),
				rekyou.UpdateTime.Unix(),
			}
			gkill_log.LogSQLParams(ctx, sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to REKYOU %s: %w", rekyou.ID, err)
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
		err = fmt.Errorf("error at commit transaction for add rekyou: %w", err)
		return err
	}
	isCommitted = true
	return nil
}

func hasRecursiveReKyouRepReference(rep ReKyouRepository, self *reKyouRepositoryCachedSQLite3Impl, visited map[*ReKyouRepositories]struct{}) bool {
	switch typed := rep.(type) {
	case *reKyouRepositoryCachedSQLite3Impl:
		return typed == self
	case *ReKyouRepositories:
		if _, exist := visited[typed]; exist {
			return false
		}
		visited[typed] = struct{}{}
		for _, nested := range typed.ReKyouRepositories {
			if hasRecursiveReKyouRepReference(nested, self, visited) {
				return true
			}
		}
	}
	return false
}

func (r *reKyouRepositoryCachedSQLite3Impl) LastUpdateCacheChanged() bool {
	return true
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return r.rekyouRep.GetRepName(ctx)
}

func (r *reKyouRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	r.m.Lock()
	defer r.m.Unlock()
	errs := []error{}
	if r.addReKyouInfoStmt != nil {
		if err := r.addReKyouInfoStmt.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.rekyouRep != nil {
		if err := r.rekyouRep.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if gkill_options.CacheReKyouReps != nil && *gkill_options.CacheReKyouReps {
		if _, err := r.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+sqlite3impl.QuoteIdent(r.dbName)); err != nil {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}
	if err := r.cachedDB.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (r *reKyouRepositoryCachedSQLite3Impl) FindReKyou(ctx context.Context, query *find.FindQuery) ([]ReKyou, error) {
	// FindKyousと同じく、自分でロックを取るヘルパーしか呼ばないためr.mを取ってはいけない(再帰RLockデッドロック防止)
	matchReKyous := []ReKyou{}

	// 未削除ReKyouを抽出
	notDeletedAllReKyous := []ReKyou{}
	allReKyous, err := r.GetReKyousAllLatest(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rekyous all latest :%w", err)
		return nil, err
	}
	for _, rekyou := range allReKyous {
		if !rekyou.IsDeleted {
			notDeletedAllReKyous = append(notDeletedAllReKyous, rekyou)
		}
	}

	repsWithoutRekyou, err := r.GetRepositoriesWithoutReKyouRep(ctx)
	if err != nil {
		err = fmt.Errorf("error at get without rekyou: %w", err)
		return nil, err
	}

	// リポジトリ群を辿れない場合はターゲット解決を行わずすべて通す（MiReKyouのallowAllと同じ扱い）
	allowAllTargets := repsWithoutRekyou == nil

	if !allowAllTargets {
		if err := repsWithoutRekyou.EnsureLatestDataRepositoryAddresses(ctx); err != nil {
			err = fmt.Errorf("error at get all latest data repository addresses: %w", err)
			return nil, err
		}
	}

	for _, rekyou := range notDeletedAllReKyous {
		if rekyou.IsDeleted {
			continue
		}
		if !allowAllTargets {
			if _, ok := repsWithoutRekyou.GetLatestDataRepositoryAddress(rekyou.TargetID); !ok {
				continue
			}
		}
		matchReKyous = append(matchReKyous, rekyou)
	}
	return matchReKyous, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetReKyou(ctx context.Context, id string, updateTime *time.Time) (*ReKyou, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(r.dbName) + `
WHERE  
`
	dataType := "rekyou"

	ids := []string{id}
	query := &find.FindQuery{
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UpdateTime:     updateTime,
	}
	queryArgs := []any{
		dataType,
	}

	tableName := r.dbName
	tableNameAlias := r.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
	stmt, err := r.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get rekyou histories sql: %w", err)
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
		err = fmt.Errorf("error at select from REKYOU: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	reKyous := []ReKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			reKyou := ReKyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&reKyou.IsDeleted,
				&reKyou.ID,
				&reKyou.TargetID,
				&relatedTimeUnix,
				&createTimeUnix,
				&reKyou.CreateApp,
				&reKyou.CreateDevice,
				&reKyou.CreateUser,
				&updateTimeUnix,
				&reKyou.UpdateApp,
				&reKyou.UpdateDevice,
				&reKyou.UpdateUser,
				&reKyou.RepName,
				&reKyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from REKYOU %s: %w", id, err)
				return nil, err
			}

			reKyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			reKyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			reKyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			reKyous = append(reKyous, reKyou)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(reKyous) == 0 {
		return nil, nil
	}
	return &reKyous[0], nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetReKyouHistories(ctx context.Context, id string) ([]ReKyou, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(r.dbName) + `
WHERE  
`
	dataType := "rekyou"

	ids := []string{id}
	query := &find.FindQuery{
		IDs: ids,
	}
	queryArgs := []any{
		dataType,
	}

	tableName := r.dbName
	tableNameAlias := r.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
	stmt, err := r.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get rekyou histories sql: %w", err)
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
		err = fmt.Errorf("error at select from REKYOU: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	reKyous := []ReKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			reKyou := ReKyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&reKyou.IsDeleted,
				&reKyou.ID,
				&reKyou.TargetID,
				&relatedTimeUnix,
				&createTimeUnix,
				&reKyou.CreateApp,
				&reKyou.CreateDevice,
				&reKyou.CreateUser,
				&updateTimeUnix,
				&reKyou.UpdateApp,
				&reKyou.UpdateDevice,
				&reKyou.UpdateUser,
				&reKyou.RepName,
				&reKyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from REKYOU %s: %w", id, err)
				return nil, err
			}

			reKyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			reKyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			reKyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			reKyous = append(reKyous, reKyou)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return reKyous, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) AddReKyouInfo(ctx context.Context, rekyou ReKyou) error {
	r.m.Lock()
	defer r.m.Unlock()
	if rekyou.RepName == "" {
		if r.gkillRepositories != nil && r.gkillRepositories.WriteReKyouRep != nil {
			repName, err := r.gkillRepositories.WriteReKyouRep.GetRepName(ctx)
			if err == nil && repName != "" {
				rekyou.RepName = repName
			}
		}
		if rekyou.RepName == "" && r.rekyouRep != nil {
			repImpls, err := r.rekyouRep.UnWrapTyped()
			if err == nil && len(repImpls) == 1 {
				repName, err := repImpls[0].GetRepName(ctx)
				if err == nil && repName != "" {
					rekyou.RepName = repName
				}
			}
		}
	}
	queryArgs := []any{
		rekyou.IsDeleted,
		rekyou.ID,
		rekyou.TargetID,
		rekyou.CreateApp,
		rekyou.CreateDevice,
		rekyou.CreateUser,
		rekyou.UpdateApp,
		rekyou.UpdateDevice,
		rekyou.UpdateUser,
		rekyou.RepName,
		rekyou.RelatedTime.Unix(),
		rekyou.CreateTime.Unix(),
		rekyou.UpdateTime.Unix(),
	}
	gkill_log.LogSQLParams(ctx, r.addReKyouInfoSQL, queryArgs)
	_, err := r.addReKyouInfoStmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to REKYOU %s: %w", rekyou.ID, err)
		return err
	}
	return nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetReKyousAllLatest(ctx context.Context) ([]ReKyou, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(r.dbName) + `
WHERE 
`

	dataType := "rekyou"

	queryArgs := []any{
		dataType,
	}
	query := &find.FindQuery{}

	tableName := r.dbName
	tableNameAlias := r.dbName
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
	stmt, err := r.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all rekyous sql: %w", err)
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
		err = fmt.Errorf("error at select from REKYOU: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	reKyous := []ReKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			reKyou := ReKyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&reKyou.IsDeleted,
				&reKyou.ID,
				&reKyou.TargetID,
				&relatedTimeUnix,
				&createTimeUnix,
				&reKyou.CreateApp,
				&reKyou.CreateDevice,
				&reKyou.CreateUser,
				&updateTimeUnix,
				&reKyou.UpdateApp,
				&reKyou.UpdateDevice,
				&reKyou.UpdateUser,
				&reKyou.RepName,
				&reKyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from REKYOU: %w", err)
				return nil, err
			}

			reKyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			reKyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			reKyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			reKyous = append(reKyous, reKyou)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return reKyous, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetRepositoriesWithoutReKyouRep(ctx context.Context) (*GkillRepositories, error) {
	if r.gkillRepositories == nil {
		// リポジトリ群を辿れない場合はターゲット解決を行わない。呼び出し側で判定する
		return nil, nil
	}
	return cloneRepositoriesWithoutReKyou(r.gkillRepositories, r.gkillRepositories.collectNonReKyouRepositories()), nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) UnWrapTyped() ([]ReKyouRepository, error) {
	if r.rekyouRep == nil {
		return []ReKyouRepository{r}, nil
	}
	return r.rekyouRep.UnWrapTyped()
}

func (r *reKyouRepositoryCachedSQLite3Impl) UnWrap() ([]Repository, error) {
	if r.rekyouRep == nil {
		return []Repository{r}, nil
	}
	return r.rekyouRep.UnWrap()
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	sql := `
SELECT IS_DELETED, ID AS TARGET_ID, NULL AS TARGET_ID_IN_DATA,
       REP_NAME AS LATEST_DATA_REPOSITORY_NAME, UPDATE_TIME_UNIX AS DATA_UPDATE_TIME_UNIX
FROM ` + sqlite3impl.QuoteIdent(r.dbName) + ` AS T
WHERE T.UPDATE_TIME_UNIX = (SELECT MAX(UPDATE_TIME_UNIX) FROM ` + sqlite3impl.QuoteIdent(r.dbName) + ` AS INNER_TABLE WHERE INNER_TABLE.ID = T.ID)
`
	stmt, err := r.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	latestDataRepositoryAddresses := []gkill_cache.LatestDataRepositoryAddress{}
	for rows.Next() {
		addr := gkill_cache.LatestDataRepositoryAddress{}
		var isDeletedInt int
		var dataUpdateTimeUnix int64
		var targetIDInData *string
		err := rows.Scan(&isDeletedInt, &addr.TargetID, &targetIDInData, &addr.LatestDataRepositoryName, &dataUpdateTimeUnix)
		if err != nil {
			return nil, err
		}
		addr.IsDeleted = isDeletedInt != 0
		addr.DataUpdateTime = time.Unix(dataUpdateTimeUnix, 0)
		if targetIDInData != nil {
			addr.TargetID = *targetIDInData
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, addr)
	}
	return latestDataRepositoryAddresses, nil
}
