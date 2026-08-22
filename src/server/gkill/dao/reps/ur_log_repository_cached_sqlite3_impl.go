package reps

// サムネイルをキャッシュに載せない理由と、集約への丸投げを却下した理由:
// documents/adr/0016-exclude-urlog-thumbnail-from-cache.md

import (
	"context"
	sqllib "database/sql"
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

type urlogRepositoryCachedSQLite3Impl struct {
	dbName           string
	urlogRep         URLogRepository
	cachedDB         *sqllib.DB
	addURLogInfoSQL  string
	addURLogInfoStmt *sqllib.Stmt
	m                *sync.RWMutex

	// サムネイルを取り直すときに使う「rep名 -> rep」の索引。
	// 配下のrepは生成後に変わらないので一度だけ作る。
	repsByNameOnce sync.Once
	repsByName     map[string]URLogRepository
}

// ownerRepOf は、その版を持っているrepを REP_NAME から引きます。
//
// サムネイルはキャッシュ表に持っていないので実DBから読み直す必要があるが、
// u.urlogRep は配下rep全部の集約なので、そこへ丸ごと投げると
// 1件取るのにrep数ぶんのクエリが飛ぶ（実データでは17rep = 約13.8ms）。
// キャッシュ表の REP_NAME には個々のrep名が入っているので、
// それを使って持ち主のrepだけを引く。
func (u *urlogRepositoryCachedSQLite3Impl) ownerRepOf(ctx context.Context, repName string) (URLogRepository, bool) {
	u.repsByNameOnce.Do(func() {
		reps, err := u.urlogRep.UnWrapTyped()
		if err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at unwrap urlog reps", "error", fmt.Sprintf("%q", err))
			return
		}
		byName := make(map[string]URLogRepository, len(reps))
		for _, rep := range reps {
			name, err := rep.GetRepName(ctx)
			if err != nil {
				continue
			}
			if _, exist := byName[name]; !exist {
				byName[name] = rep
			}
		}
		u.repsByName = byName
	})
	rep, exist := u.repsByName[repName]
	return rep, exist
}

// fillThumbnailImages は、キャッシュに持っていないサムネイルを
// その版を持つrepからだけ読み直して埋めます。
//
// 持ち主のrepが見つからない場合や、その版が実DBに無い場合
// （キャッシュにしか無い行など）はサムネイルを空のままにする。
// 表示側は空文字なら noimage にフォールバックするので壊れない。
func (u *urlogRepositoryCachedSQLite3Impl) fillThumbnailImages(ctx context.Context, urlogs []URLog) {
	for i := range urlogs {
		rep, exist := u.ownerRepOf(ctx, urlogs[i].RepName)
		if !exist {
			continue
		}
		updateTime := urlogs[i].UpdateTime
		got, err := rep.GetURLog(ctx, urlogs[i].ID, &updateTime)
		if err != nil || got == nil {
			continue
		}
		urlogs[i].ThumbnailImage = got.ThumbnailImage
	}
}

func NewURLogRepositoryCachedSQLite3Impl(ctx context.Context, urlogRepository URLogRepository, cacheDB *sqllib.DB, m *sync.RWMutex, dbName string) (URLogRepository, error) {
	if m == nil {
		m = &sync.RWMutex{}
	}
	var err error

	// THUMBNAIL_IMAGE 列は意図的に持たせていない。
	//
	// このキャッシュ表は既定でインメモリDB(gkill_memory_db_<userID>)上に作られる。
	// URLogのTHUMBNAIL_IMAGEはbase64で1行あたり平均406KB・最大10MBあり、
	// 実データ227行の合計が90MBに達する。これを常時メモリに置きたくない。
	// (FAVICON_IMAGEは合計0.10MB・平均0.5KBなので持たせている)
	//
	// サムネイルが要る GetURLog / GetURLogHistories は、
	// REP_NAME からその版を持つrepを特定してそこだけ読み直す。
	sql := `
CREATE TABLE IF NOT EXISTS ` + sqlite3impl.QuoteIdent(dbName) + ` (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  URL NOT NULL,
  TITLE NOT NULL,
  DESCRIPTION NOT NULL,
  FAVICON_IMAGE NOT NULL,
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
		err = fmt.Errorf("error at create URLOG table statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create URLOG table statement %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_UNIX") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	gkill_log.LogSQL(ctx, indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create urlog index unix statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create urlog index unix to %s: %w", dbName, err)
		return nil, err
	}

	// 既存索引は先頭が ID なので時刻範囲にもORDER BYにも使えない。
	// 時刻列を先頭にした索引を別途張る。
	if err := sqlite3impl.EnsureUnixColumnIndex(ctx, cacheDB, dbName, "RELATED_TIME_UNIX"); err != nil {
		return nil, err
	}

	// THUMBNAIL_IMAGE はこのキャッシュ表に持たない（メモリに載せないため）。
	// 詳細は CREATE TABLE 側のコメントを参照。
	addURLogInfoSQL := `
INSERT INTO ` + sqlite3impl.QuoteIdent(dbName) + ` (
  IS_DELETED,
  ID,
  URL,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
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
  ?,
  ?
)`
	gkill_log.LogSQL(ctx, addURLogInfoSQL)
	addURLogInfoStmt, err := cacheDB.PrepareContext(ctx, addURLogInfoSQL)
	if err != nil {
		err = fmt.Errorf("error at add urlog info sql: %w", err)
		return nil, err
	}

	return &urlogRepositoryCachedSQLite3Impl{
		dbName:           dbName,
		urlogRep:         urlogRepository,
		cachedDB:         cacheDB,
		addURLogInfoSQL:  addURLogInfoSQL,
		addURLogInfoStmt: addURLogInfoStmt,
		m:                m,
	}, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = u.UpdateCache(ctx)
		if err != nil {
			repName, _ := u.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}
	u.m.RLock()
	defer u.m.RUnlock()

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
  REP_NAME
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + `
WHERE
`

	// DATA_TYPE はコンパイル時定数なので、SQLの射影に混ぜない。
	// `? AS DATA_TYPE` にすると、既知の値のために**1行ごとに文字列を確保**して
	// スキャンし直すことになる(実データでは56万行ぶん)。Go側で代入すれば済む。
	dataType := "urlog"

	tableName := sqlite3impl.QuoteIdent(u.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(u.dbName)
	queryArgs := []any{}
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
	ignoreFindWord := false
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
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
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

	gkill_log.LogSQLQuery(ctx, sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from URLOG: %w", err)
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
			)
			if err != nil {
				err = fmt.Errorf("error at scan from URLOG: %w", err)
				return nil, err
			}
			kyou.DataType = dataType

			kyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			kyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			kyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
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

func (u *urlogRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + `
WHERE
`
	dataType := "urlog"

	ids := []string{id}
	query := &find.FindQuery{
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UpdateTime:     updateTime,
	}

	tableName := sqlite3impl.QuoteIdent(u.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(u.dbName)
	queryArgs := []any{
		dataType,
	}
	whereCounter := 0
	// GenerateFindSQLCommon は query.OnlyLatestData を読まず、この引数しか見ない。
	// false のままだと updateTime 未指定のときに **そのIDの全バージョンを無順序・無制限に読み**、
	// 下の kyous[0] が格納順の先頭(多くの場合いちばん古い版)を返してしまう。
	// 版の数だけ走査するので遅くもある。Tag / Text では既に同じ修正が入っている。
	onlyLatestData := updateTime == nil
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
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
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
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
		err = fmt.Errorf("error at select from URLOG %s: %w", id, err)
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
				err = fmt.Errorf("error at scan from URLOG %s: %w", id, err)
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

func (u *urlogRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
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
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + `
WHERE
`
	dataType := "urlog"

	ids := []string{id}
	query := &find.FindQuery{
		IDs: ids,
	}

	tableName := sqlite3impl.QuoteIdent(u.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(u.dbName)
	queryArgs := []any{
		dataType,
	}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
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
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
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
		err = fmt.Errorf("error at select from URLOG %s: %w", id, err)
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
				err = fmt.Errorf("error at scan from URLOG %s: %w", id, err)
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

func (u *urlogRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return u.urlogRep.GetPath(ctx, id)
}

func (u *urlogRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {

	err := u.urlogRep.UpdateCache(ctx)
	if err != nil {
		return fmt.Errorf("error at update underlying urlog rep cache: %w", err)
	}

	// 下層リポジトリに変更がなければフルリビルドをスキップ
	if !u.urlogRep.LastUpdateCacheChanged() {
		return nil
	}

	// サムネイルはキャッシュ表に入れないので、下層から読む段階で外す。
	// これを付けないと、全件を []URLog に載せる時点で
	// Goのヒープに90MBが乗ってしまう（実データ227行の合計）。
	query := &find.FindQuery{
		UpdateCache:                false,
		OnlyLatestData:             false,
		ExcludeURLogThumbnailImage: true,
	}

	allURLogs, err := u.urlogRep.FindURLog(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all urlogs at update cache: %w", err)
		return err
	}

	u.m.Lock()
	defer u.m.Unlock()

	tx, err := u.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add urlogs: %w", err)
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

	sql := `DELETE FROM ` + sqlite3impl.QuoteIdent(u.dbName)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table statement %s: %w", "memory", err)
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
		err = fmt.Errorf("error at delete URLOG table: %w", err)
		return err
	}

	// THUMBNAIL_IMAGE はキャッシュ表に持たない（メモリに載せないため）
	sql = `
INSERT INTO ` + sqlite3impl.QuoteIdent(u.dbName) + ` (
  IS_DELETED,
  ID,
  URL,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
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
  ?,
  ?
)`

	gkill_log.LogSQL(ctx, sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add urlog sql: %w", err)
		return err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for _, urlog := range allURLogs {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			queryArgs := []any{
				urlog.IsDeleted,
				urlog.ID,
				urlog.URL,
				urlog.Title,
				urlog.Description,
				urlog.FaviconImage,
				// THUMBNAIL_IMAGE はキャッシュ表に持たない（メモリに載せないため）
				urlog.CreateApp,
				urlog.CreateDevice,
				urlog.CreateUser,
				urlog.UpdateApp,
				urlog.UpdateDevice,
				urlog.UpdateUser,
				urlog.RepName,
				urlog.RelatedTime.Unix(),
				urlog.CreateTime.Unix(),
				urlog.UpdateTime.Unix(),
			}
			gkill_log.LogSQLParams(ctx, sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to URLog %s: %w", urlog.ID, err)
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
		err = fmt.Errorf("error at commit transaction for add urlogs: %w", err)
		return err
	}
	isCommitted = true
	// ここまで来て初めて「取り込み済み」とみなす。
	// 途中で失敗した場合は基準を進めないので、次回も再構築される。
	commitCacheRebuildIfSupported(u.urlogRep)
	return nil
}

func (u *urlogRepositoryCachedSQLite3Impl) LastUpdateCacheChanged() bool {
	return true
}

func (u *urlogRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return u.urlogRep.GetRepName(ctx)
}

func (u *urlogRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	u.m.Lock()
	defer u.m.Unlock()
	if u.addURLogInfoStmt != nil {
		u.addURLogInfoStmt.Close()
	}
	err := u.urlogRep.Close(ctx)
	if err != nil {
		return err
	}
	if gkill_options.CacheURLogReps == nil || !*gkill_options.CacheURLogReps {
		err = u.cachedDB.Close()
		if err != nil {
			return err
		}
	} else {
		_, err = u.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+sqlite3impl.QuoteIdent(u.dbName))
		if err != nil {
			return err
		}
	}
	return nil
}

func (u *urlogRepositoryCachedSQLite3Impl) FindURLog(ctx context.Context, query *find.FindQuery) ([]URLog, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = u.UpdateCache(ctx)
		if err != nil {
			repName, _ := u.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}

	// サムネイルはキャッシュ表に持っていないので、要求されたら下層の実DBへ回す。
	// ここを通るのは共有ページ(handle_get_shared_kyous)だけで頻度が低いため、
	// 下層repの数だけクエリが飛ぶことを許容する。
	// 一件ずつ引く GetURLog / GetURLogHistories は表示のたびに呼ばれるので、
	// そちらは REP_NAME で持ち主のrepだけを引く方式にしてある。
	if !query.ExcludeURLogThumbnailImage {
		return u.urlogRep.FindURLog(ctx, query)
	}

	u.m.RLock()
	defer u.m.RUnlock()

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
  URL,
  TITLE,
  DESCRIPTION,
  ` + urlogImageColumnsSQL(query) + `,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + `
WHERE
`

	dataType := "urlog"

	tableName := sqlite3impl.QuoteIdent(u.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(u.dbName)
	queryArgs := []any{
		dataType,
	}
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
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

	gkill_log.LogSQL(ctx, sql)
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
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
		err = fmt.Errorf("error at select from URLOG: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	urlogs := []URLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			urlog := URLog{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(
				&urlog.IsDeleted,
				&urlog.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&urlog.CreateApp,
				&urlog.CreateDevice,
				&urlog.CreateUser,
				&updateTimeUnix,
				&urlog.UpdateApp,
				&urlog.UpdateDevice,
				&urlog.UpdateUser,
				&urlog.URL,
				&urlog.Title,
				&urlog.Description,
				&urlog.FaviconImage,
				&urlog.ThumbnailImage,
				&urlog.RepName,
				&urlog.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from URLOG: %w", err)
				return nil, err
			}

			urlog.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			urlog.CreateTime = time.Unix(createTimeUnix, 0).Local()
			urlog.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			urlogs = append(urlogs, urlog)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return urlogs, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetURLog(ctx context.Context, id string, updateTime *time.Time) (*URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at URLOG: %w", err)
		return nil, err
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
  URL,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
  '' AS THUMBNAIL_IMAGE,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + `
WHERE
`

	ids := []string{id}
	query := &find.FindQuery{
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UpdateTime:     updateTime,
	}
	dataType := "urlog"

	tableName := sqlite3impl.QuoteIdent(u.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(u.dbName)
	queryArgs := []any{
		dataType,
	}
	whereCounter := 0
	// GenerateFindSQLCommon は query.OnlyLatestData を読まず、この引数しか見ない（既定 false のままだと最古版を返す）。
	onlyLatestData := query.OnlyLatestData
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
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
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get urlog histories sql %s: %w", id, err)
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

	urlogs := []URLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			urlog := URLog{}
			urlog.RepName = repName
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(
				&urlog.IsDeleted,
				&urlog.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&urlog.CreateApp,
				&urlog.CreateDevice,
				&urlog.CreateUser,
				&updateTimeUnix,
				&urlog.UpdateApp,
				&urlog.UpdateDevice,
				&urlog.UpdateUser,
				&urlog.URL,
				&urlog.Title,
				&urlog.Description,
				&urlog.FaviconImage,
				&urlog.ThumbnailImage,
				&urlog.RepName,
				&urlog.DataType,
			)

			urlog.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			urlog.CreateTime = time.Unix(createTimeUnix, 0).Local()
			urlog.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			if err != nil {
				err = fmt.Errorf("error at scan from URLOG %s: %w", id, err)
				return nil, err
			}
			urlogs = append(urlogs, urlog)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(urlogs) == 0 {
		return nil, nil
	}
	// サムネイルはキャッシュに持っていないので、その版を持つrepから読み直す
	u.fillThumbnailImages(ctx, urlogs)
	// 最新版に絞ってもrepをまたいだ同一版が複数返りうるので、UpdateTimeが最大のものを選ぶ。
	// 格納順の先頭を返すと、どれが返るかがSQLiteの都合で決まってしまう。
	latestURLog := slices.MaxFunc(urlogs, func(a URLog, b URLog) int {
		return a.UpdateTime.Compare(b.UpdateTime)
	})
	return &latestURLog, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) GetURLogHistories(ctx context.Context, id string) ([]URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at URLOG: %w", err)
		return nil, err
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
  URL,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
  '' AS THUMBNAIL_IMAGE,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + `
WHERE
`

	ids := []string{id}
	query := &find.FindQuery{
		IDs: ids,
	}
	dataType := "urlog"

	tableName := sqlite3impl.QuoteIdent(u.dbName)
	tableNameAlias := sqlite3impl.QuoteIdent(u.dbName)
	queryArgs := []any{
		dataType,
	}
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"URL", "TITLE", "DESCRIPTION"}
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
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get urlog histories sql %s: %w", id, err)
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

	urlogs := []URLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			urlog := URLog{}
			urlog.RepName = repName
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(
				&urlog.IsDeleted,
				&urlog.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&urlog.CreateApp,
				&urlog.CreateDevice,
				&urlog.CreateUser,
				&updateTimeUnix,
				&urlog.UpdateApp,
				&urlog.UpdateDevice,
				&urlog.UpdateUser,
				&urlog.URL,
				&urlog.Title,
				&urlog.Description,
				&urlog.FaviconImage,
				&urlog.ThumbnailImage,
				&urlog.RepName,
				&urlog.DataType,
			)

			urlog.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			urlog.CreateTime = time.Unix(createTimeUnix, 0).Local()
			urlog.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			if err != nil {
				err = fmt.Errorf("error at scan from URLOG %s: %w", id, err)
				return nil, err
			}
			urlogs = append(urlogs, urlog)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	// サムネイルはキャッシュに持っていないので、その版を持つrepから読み直す。
	// フロントがURLogの画像を受け取るのはこの経路（/api/get_urlog）だけ。
	u.fillThumbnailImages(ctx, urlogs)
	return urlogs, nil
}

func (u *urlogRepositoryCachedSQLite3Impl) AddURLogInfo(ctx context.Context, urlog URLog) error {
	u.m.Lock()
	defer u.m.Unlock()
	queryArgs := []any{
		urlog.IsDeleted,
		urlog.ID,
		urlog.URL,
		urlog.Title,
		urlog.Description,
		urlog.FaviconImage,
		// THUMBNAIL_IMAGE はキャッシュ表に持たない（メモリに載せないため）
		urlog.CreateApp,
		urlog.CreateDevice,
		urlog.CreateUser,
		urlog.UpdateApp,
		urlog.UpdateDevice,
		urlog.UpdateUser,
		urlog.RepName,
		urlog.RelatedTime.Unix(),
		urlog.CreateTime.Unix(),
		urlog.UpdateTime.Unix(),
	}
	gkill_log.LogSQLParams(ctx, u.addURLogInfoSQL, queryArgs)
	_, err := u.addURLogInfoStmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to URLog %s: %w", urlog.ID, err)
		return err
	}
	return nil
}

func (u *urlogRepositoryCachedSQLite3Impl) UnWrapTyped() ([]URLogRepository, error) {
	return u.urlogRep.UnWrapTyped()
}

func (u *urlogRepositoryCachedSQLite3Impl) UnWrap() ([]Repository, error) {
	return u.urlogRep.UnWrap()
}

func (u *urlogRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	repName, err := u.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	sql := `
SELECT IS_DELETED, ID AS TARGET_ID, NULL AS TARGET_ID_IN_DATA,
       ? AS LATEST_DATA_REPOSITORY_NAME, UPDATE_TIME_UNIX AS DATA_UPDATE_TIME_UNIX
FROM ` + sqlite3impl.QuoteIdent(u.dbName) + ` AS T
WHERE T.UPDATE_TIME_UNIX = (SELECT MAX(UPDATE_TIME_UNIX) FROM ` + sqlite3impl.QuoteIdent(u.dbName) + ` AS INNER_TABLE WHERE INNER_TABLE.ID = T.ID)
`
	stmt, err := u.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, repName)
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return latestDataRepositoryAddresses, nil
}
