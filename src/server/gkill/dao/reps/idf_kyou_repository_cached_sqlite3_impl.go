package reps

import (
	"context"
	sqllib "database/sql"
	"fmt"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"log/slog"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	_ "modernc.org/sqlite"
)

type idfKyouRepositoryCachedSQLite3Impl struct {
	dbName             string
	idfRep             IDFKyouRepository
	cachedDB           *sqllib.DB
	m                  *sync.RWMutex
	addIDFKyouInfoSQL  string
	addIDFKyouInfoStmt *sqllib.Stmt
}

func NewIDFCachedRep(ctx context.Context, idfRep IDFKyouRepository, cacheDB *sqllib.DB, m *sync.RWMutex, dbName string) (IDFKyouRepository, error) {
	if m == nil {
		m = &sync.RWMutex{}
	}
	sql := `
CREATE TABLE IF NOT EXISTS ` + sqlite3impl.QuoteIdent(dbName) + ` (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_REP_NAME,
  TARGET_FILE NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  CONTENT_PATH NOT NULL,
  REP_NAME NOT NULL,
  RELATED_TIME_UNIX NOT NULL,
  CREATE_TIME_UNIX NOT NULL,
  UPDATE_TIME_UNIX NOT NULL
)
`

	gkill_log.LogSQL(ctx, sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create IDF table statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create IDF table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_UNIX") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + ` (ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	gkill_log.LogSQL(ctx, indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create IDF index unix statement %s: %w", dbName, err)
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
		err = fmt.Errorf("error at create IDF index unix to %s: %w", dbName, err)
		return nil, err
	}

	// 既存索引は先頭が ID なので時刻範囲にもORDER BYにも使えない。
	// 時刻列を先頭にした索引を別途張る。
	if err := sqlite3impl.EnsureUnixColumnIndex(ctx, cacheDB, dbName, "RELATED_TIME_UNIX"); err != nil {
		return nil, err
	}

	addIDFKyouInfoSQL := `
INSERT INTO ` + sqlite3impl.QuoteIdent(dbName) + ` (
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  CONTENT_PATH,
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
);`
	gkill_log.LogSQL(ctx, addIDFKyouInfoSQL)
	addIDFKyouInfoStmt, err := cacheDB.PrepareContext(ctx, addIDFKyouInfoSQL)
	if err != nil {
		err = fmt.Errorf("error at add idf kyou info sql: %w", err)
		return nil, err
	}

	rep := &idfKyouRepositoryCachedSQLite3Impl{
		dbName:             dbName,
		idfRep:             idfRep,
		cachedDB:           cacheDB,
		m:                  m,
		addIDFKyouInfoSQL:  addIDFKyouInfoSQL,
		addIDFKyouInfoStmt: addIDFKyouInfoStmt,
	}
	return rep, nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	var err error
	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = i.UpdateCache(ctx)
		if err != nil {
			repName, _ := i.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	i.m.RLock()
	defer i.m.RUnlock()

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  CONTENT_PATH,
  REP_NAME
FROM ` + sqlite3impl.QuoteIdent(i.dbName) + `
WHERE
`
	// DATA_TYPE はコンパイル時定数なので、SQLの射影に混ぜない。
	// `? AS DATA_TYPE` にすると、既知の値のために**1行ごとに文字列を確保**して
	// スキャンし直すことになる(実データでは56万行ぶん)。Go側で代入すれば済む。
	dataType := "idf"
	queryArgs := []any{}

	tableName := i.dbName
	tableNameAlias := i.dbName
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TARGET_FILE"}
	ignoreFindWord := true
	// 結果は map[string][]Kyou に収めるので、SQL側で並べても順序は捨てられる。
	// 最終的な並び順は find_filter の Go 側ソートで決まる。
	appendOrderBy := false
	findWordUseLike := true
	// 非キャッシュ側のFindKyousと揃える。
	// ignoreFindWordが真なのでキーワードのSQLは組み立てられず現状は効かないが、
	// 同じ検索に対して実装ごとに大文字小文字の扱いが違う状態を残さない。
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyou sql: %w", err)
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
		err = fmt.Errorf("error at select from idf: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	words := lowerFindWords(query.Words)
	notWords := lowerFindWords(query.NotWords)

	kyous := map[string][]Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			idf := IDFKyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			targetRepName := ""

			err = rows.Scan(
				&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeUnix,
				&createTimeUnix,
				&idf.CreateApp,
				&idf.CreateDevice,
				&idf.CreateUser,
				&updateTimeUnix,
				&idf.UpdateApp,
				&idf.UpdateDevice,
				&idf.UpdateUser,
				&idf.ContentPath,
				&idf.RepName,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from idf: %w", err)
				return nil, err
			}
			idf.DataType = dataType

			// targetRepNameが空の場合はRepNameにフォールバック
			if targetRepName == "" || targetRepName == "." {
				targetRepName = idf.RepName
			}

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)

			idf.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			idf.CreateTime = time.Unix(createTimeUnix, 0).Local()
			idf.UpdateTime = time.Unix(updateTimeUnix, 0).Local()

			// 判定OKであれば追加する
			filename := idf.ContentPath
			if filename == "" {
				// 接続されていないRepのIDがあったときは無視する
				continue
			}

			match := true
			if query.HasWordFilter() {
				match = matchFindWords(findWordTextOfIDFKyou(ctx, idf.TargetFile, filename), words, notWords, query.WordsAnd)
			}

			if match {
				kyou := Kyou{}
				kyou.IsDeleted = idf.IsDeleted
				kyou.ID = idf.ID
				kyou.RepName = idf.RepName
				kyou.RelatedTime = idf.RelatedTime
				kyou.DataType = idf.DataType
				kyou.CreateTime = idf.CreateTime
				kyou.CreateApp = idf.CreateApp
				kyou.CreateDevice = idf.CreateDevice
				kyou.CreateUser = idf.CreateUser
				kyou.UpdateTime = idf.UpdateTime
				kyou.UpdateApp = idf.UpdateApp
				kyou.UpdateUser = idf.UpdateUser
				kyou.UpdateDevice = idf.UpdateDevice
				kyou.IsImage = idf.IsImage
				kyou.IsVideo = idf.IsVideo

				// 空スライスの事前確保はしない。存在しないキーへのappendはnilスライスに対して働くので
				// 結果は同じで、レコード1件につき1回の無駄な確保(実データで56万回)が消える。
				// 同じ整理は dao/reps/repositories.go の集約側では既に済んでいる。
				kyous[kyou.ID] = append(kyous[kyou.ID], kyou)
			}
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return kyous, nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	var err error
	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  CONTENT_PATH,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(i.dbName) + `
WHERE
`

	dataType := "idf"
	queryArgs := []any{
		dataType,
	}

	ids := []string{id}
	query := &find.FindQuery{
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UpdateTime:     updateTime,
	}

	tableName := i.dbName
	tableNameAlias := i.dbName
	whereCounter := 0
	// GenerateFindSQLCommon は query.OnlyLatestData を読まず、この引数しか見ない。
	// false のままだと updateTime 未指定のときに **そのIDの全バージョンを無順序・無制限に読み**、
	// 下の kyous[0] が格納順の先頭(多くの場合いちばん古い版)を返してしまう。
	// 版の数だけ走査するので遅くもある。Tag / Text では既に同じ修正が入っている。
	onlyLatestData := updateTime == nil
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TARGET_FILE"}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		err = fmt.Errorf("error at generate find sql common: %w", err)
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
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
		err = fmt.Errorf("error at select from idf: %w", err)
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
			idf := IDFKyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			targetRepName := ""

			err = rows.Scan(
				&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeUnix,
				&createTimeUnix,
				&idf.CreateApp,
				&idf.CreateDevice,
				&idf.CreateUser,
				&updateTimeUnix,
				&idf.UpdateApp,
				&idf.UpdateDevice,
				&idf.UpdateUser,
				&idf.ContentPath,
				&idf.RepName,
				&idf.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from idf: %w", err)
				return nil, err
			}

			// targetRepNameが空の場合はRepNameにフォールバック
			if targetRepName == "" || targetRepName == "." {
				targetRepName = idf.RepName
			}

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)

			idf.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			idf.CreateTime = time.Unix(createTimeUnix, 0).Local()
			idf.UpdateTime = time.Unix(updateTimeUnix, 0).Local()

			kyou := Kyou{}
			kyou.IsDeleted = idf.IsDeleted
			kyou.ID = idf.ID
			kyou.RepName = idf.RepName
			kyou.RelatedTime = idf.RelatedTime
			kyou.DataType = idf.DataType
			kyou.CreateTime = idf.CreateTime
			kyou.CreateApp = idf.CreateApp
			kyou.CreateDevice = idf.CreateDevice
			kyou.CreateUser = idf.CreateUser
			kyou.UpdateTime = idf.UpdateTime
			kyou.UpdateApp = idf.UpdateApp
			kyou.UpdateUser = idf.UpdateUser
			kyou.UpdateDevice = idf.UpdateDevice
			kyou.IsImage = idf.IsImage
			kyou.IsVideo = idf.IsVideo

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

func (i *idfKyouRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	var err error
	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  CONTENT_PATH,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(i.dbName) + `
WHERE
`

	dataType := "idf"
	queryArgs := []any{
		dataType,
	}

	ids := []string{id}
	query := &find.FindQuery{
		IDs: ids,
	}

	tableName := i.dbName
	tableNameAlias := i.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TARGET_FILE"}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		err = fmt.Errorf("error at generate find sql common: %w", err)
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
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
		err = fmt.Errorf("error at select from idf: %w", err)
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
			idf := IDFKyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			targetRepName := ""

			err = rows.Scan(
				&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeUnix,
				&createTimeUnix,
				&idf.CreateApp,
				&idf.CreateDevice,
				&idf.CreateUser,
				&updateTimeUnix,
				&idf.UpdateApp,
				&idf.UpdateDevice,
				&idf.UpdateUser,
				&idf.ContentPath,
				&idf.RepName,
				&idf.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from idf: %w", err)
				return nil, err
			}

			// targetRepNameが空の場合はRepNameにフォールバック
			if targetRepName == "" || targetRepName == "." {
				targetRepName = idf.RepName
			}

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)

			idf.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			idf.CreateTime = time.Unix(createTimeUnix, 0).Local()
			idf.UpdateTime = time.Unix(updateTimeUnix, 0).Local()

			kyou := Kyou{}
			kyou.IsDeleted = idf.IsDeleted
			kyou.ID = idf.ID
			kyou.RepName = idf.RepName
			kyou.RelatedTime = idf.RelatedTime
			kyou.DataType = idf.DataType
			kyou.CreateTime = idf.CreateTime
			kyou.CreateApp = idf.CreateApp
			kyou.CreateDevice = idf.CreateDevice
			kyou.CreateUser = idf.CreateUser
			kyou.UpdateTime = idf.UpdateTime
			kyou.UpdateApp = idf.UpdateApp
			kyou.UpdateUser = idf.UpdateUser
			kyou.UpdateDevice = idf.UpdateDevice
			kyou.IsImage = idf.IsImage
			kyou.IsVideo = idf.IsVideo

			kyous = append(kyous, kyou)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return kyous, nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return i.idfRep.GetPath(ctx, id)
}

func (i *idfKyouRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	err := i.idfRep.UpdateCache(ctx)
	if err != nil {
		return fmt.Errorf("error at update underlying idf kyou rep cache: %w", err)
	}

	// 下層リポジトリに変更がなければフルリビルドをスキップ
	if !i.idfRep.LastUpdateCacheChanged() {
		return nil
	}

	query := &find.FindQuery{
		UpdateCache:    false,
		OnlyLatestData: false,
	}

	// フルリビルド対象のIDFKyouを一度にまとめて1つのスライスへ載せると、
	// ファイル数の多いユーザ（数十万件規模）ではそれだけで数GBに達する。
	// 下層が複数リポジトリの集合なら、リポジトリ単位に取得とINSERTを回して、
	// 同時にメモリへ載る件数を1リポジトリ分に抑える。
	idfRepsForFetch := []IDFKyouRepository{i.idfRep}
	if idfReps, ok := i.idfRep.(IDFKyouRepositories); ok && len(idfReps) != 0 {
		idfRepsForFetch = idfReps
	}

	i.m.Lock()
	defer i.m.Unlock()

	tx, err := i.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add idf kyou: %w", err)
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

	sql := `DELETE FROM ` + sqlite3impl.QuoteIdent(i.dbName)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create idf kyou table statement %s: %w", "memory", err)
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
		err = fmt.Errorf("error at delete idf kyou table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + sqlite3impl.QuoteIdent(i.dbName) + ` (
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  CONTENT_PATH,
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
);`
	gkill_log.LogSQL(ctx, sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add idf sql: %w", err)
		return err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for _, idfRep := range idfRepsForFetch {
		idfKyous, err := idfRep.FindIDFKyou(ctx, query)
		if err != nil {
			return fmt.Errorf("error at get all idf kyou at update cache: %w", err)
		}

		for _, idfKyou := range idfKyous {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return err
			default:
			}
			err = func() error {
				queryArgs := []any{
					idfKyou.IsDeleted,
					idfKyou.ID,
					idfKyou.RepName,
					idfKyou.TargetFile,
					idfKyou.CreateApp,
					// 列は CREATE_APP, CREATE_USER, CREATE_DEVICE の順（AddIDFKyouInfo と同じ）
					idfKyou.CreateUser,
					idfKyou.CreateDevice,
					idfKyou.UpdateApp,
					idfKyou.UpdateDevice,
					idfKyou.UpdateUser,
					idfKyou.ContentPath,
					idfKyou.RepName,
					idfKyou.RelatedTime.Unix(),
					idfKyou.CreateTime.Unix(),
					idfKyou.UpdateTime.Unix(),
				}

				gkill_log.LogSQLQuery(ctx, sql, queryArgs)
				_, err = insertStmt.ExecContext(ctx, queryArgs...)
				if err != nil {
					err = fmt.Errorf("error at insert in to idf %s: %w", idfKyou.ID, err)
					return err
				}
				return nil
			}()
			if err != nil {
				return err
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit transaction for add idf kyous: %w", err)
		return err
	}
	isCommitted = true

	// 取り込みに成功したので下層の基準を進める。
	// これを呼ばないと次回も「変更あり」のままで、フルリビルドが走り続ける。
	commitCacheRebuildIfSupported(i.idfRep)
	return nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) LastUpdateCacheChanged() bool {
	return true
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return i.idfRep.GetRepName(ctx)
}

func (i *idfKyouRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	if i.addIDFKyouInfoStmt != nil {
		i.addIDFKyouInfoStmt.Close()
	}
	err := i.idfRep.Close(ctx)
	if err != nil {
		return err
	}
	if gkill_options.CacheIDFKyouReps == nil || !*gkill_options.CacheIDFKyouReps {
		err = i.cachedDB.Close()
		if err != nil {
			return err
		}
	} else {
		_, err = i.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+sqlite3impl.QuoteIdent(i.dbName))
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]IDFKyou, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = i.UpdateCache(ctx)
		if err != nil {
			repName, _ := i.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	i.m.RLock()
	defer i.m.RUnlock()

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  CONTENT_PATH,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(i.dbName) + `
WHERE
`
	dataType := "idf"
	queryArgs := []any{
		dataType,
	}

	tableName := i.dbName
	tableNameAlias := i.dbName
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TARGET_FILE"}
	ignoreFindWord := true
	appendOrderBy := true
	findWordUseLike := true
	// 非キャッシュ側のFindIDFKyouと揃える。理由はFindKyousと同じ。
	ignoreCase := true

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)

	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyou sql: %w", err)
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
		err = fmt.Errorf("error at select from idf: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	words := lowerFindWords(query.Words)
	notWords := lowerFindWords(query.NotWords)

	idfKyous := []IDFKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			idf := IDFKyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			targetRepName := ""

			err = rows.Scan(
				&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeUnix,
				&createTimeUnix,
				&idf.CreateApp,
				&idf.CreateDevice,
				&idf.CreateUser,
				&updateTimeUnix,
				&idf.UpdateApp,
				&idf.UpdateDevice,
				&idf.UpdateUser,
				&idf.ContentPath,
				&idf.RepName,
				&idf.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from idf: %w", err)
				return nil, err
			}

			// targetRepNameが空の場合はRepNameにフォールバック
			if targetRepName == "" || targetRepName == "." {
				targetRepName = idf.RepName
			}
			// 対象IDFRepsからファイルURLを取得（targetRepName解決後に構築）
			idf.FileURL = buildIDFFileURL(targetRepName, idf.TargetFile)

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)
			idf.IsAudio = isAudio(idf.TargetFile)
			idf.IsZip = isZip(idf.TargetFile)

			idf.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			idf.CreateTime = time.Unix(createTimeUnix, 0).Local()
			idf.UpdateTime = time.Unix(updateTimeUnix, 0).Local()

			filename := idf.ContentPath
			if filename == "" {
				err = fmt.Errorf("error at get path %s: %w", idf.ID, err)
				return nil, err
			}

			match := true
			if query.HasWordFilter() {
				match = matchFindWords(findWordTextOfIDFKyou(ctx, idf.TargetFile, filename), words, notWords, query.WordsAnd)
			}

			if match {
				idfKyous = append(idfKyous, idf)
			}
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return idfKyous, nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetIDFKyou(ctx context.Context, id string, updateTime *time.Time) (*IDFKyou, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	var err error
	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  CONTENT_PATH,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(i.dbName) + `
WHERE
`

	ids := []string{id}
	query := &find.FindQuery{
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UpdateTime:     updateTime,
	}

	dataType := "idf"
	queryArgs := []any{
		dataType,
	}

	tableName := i.dbName
	tableNameAlias := i.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"ID"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := false
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get idf histories sql: %w", err)
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
		err = fmt.Errorf("error at select from idf: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	idfKyous := []IDFKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			idf := IDFKyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			targetRepName := ""

			err = rows.Scan(
				&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeUnix,
				&createTimeUnix,
				&idf.CreateApp,
				&idf.CreateDevice,
				&idf.CreateUser,
				&updateTimeUnix,
				&idf.UpdateApp,
				&idf.UpdateDevice,
				&idf.UpdateUser,
				&idf.ContentPath,
				&idf.RepName,
				&idf.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from idf: %w", err)
				return nil, err
			}

			// targetRepNameが空の場合はRepNameにフォールバック
			if targetRepName == "" || targetRepName == "." {
				targetRepName = idf.RepName
			}
			// 対象IDFRepsからファイルURLを取得（targetRepName解決後に構築）
			idf.FileURL = buildIDFFileURL(targetRepName, idf.TargetFile)

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)
			idf.IsAudio = isAudio(idf.TargetFile)
			idf.IsZip = isZip(idf.TargetFile)

			idf.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			idf.CreateTime = time.Unix(createTimeUnix, 0).Local()
			idf.UpdateTime = time.Unix(updateTimeUnix, 0).Local()

			idfKyous = append(idfKyous, idf)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(idfKyous) == 0 {
		return nil, nil
	}
	return &idfKyous[0], nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetIDFKyouByTargetFile(ctx context.Context, targetFile string) (*IDFKyou, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	var err error
	tableName := sqlite3impl.QuoteIdent(i.dbName)
	sql := `
SELECT
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  CONTENT_PATH,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + tableName + `
WHERE TARGET_FILE IN (?, ?)
  AND UPDATE_TIME_UNIX = ( SELECT MAX(UPDATE_TIME_UNIX) FROM ` + tableName + ` AS INNER_TABLE WHERE INNER_TABLE.ID = ` + tableName + `.ID )
ORDER BY UPDATE_TIME_UNIX DESC
`

	// TARGET_FILEはOS依存の区切り文字で格納されている可能性があるため両方で検索する
	slashPath := filepath.ToSlash(targetFile)
	backslashPath := strings.ReplaceAll(slashPath, "/", "\\")

	dataType := "idf"
	queryArgs := []any{
		dataType,
		slashPath,
		backslashPath,
	}

	gkill_log.LogSQL(ctx, sql)
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get idf kyou by target file sql: %w", err)
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
		err = fmt.Errorf("error at select from idf: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			idf := IDFKyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			targetRepName := ""

			err = rows.Scan(
				&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeUnix,
				&createTimeUnix,
				&idf.CreateApp,
				&idf.CreateDevice,
				&idf.CreateUser,
				&updateTimeUnix,
				&idf.UpdateApp,
				&idf.UpdateDevice,
				&idf.UpdateUser,
				&idf.ContentPath,
				&idf.RepName,
				&idf.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from idf: %w", err)
				return nil, err
			}

			// 削除済みの最新データは対象外
			if idf.IsDeleted {
				continue
			}

			// targetRepNameが空の場合はRepNameにフォールバック
			if targetRepName == "" || targetRepName == "." {
				targetRepName = idf.RepName
			}
			// 対象IDFRepsからファイルURLを取得（targetRepName解決後に構築）
			idf.FileURL = buildIDFFileURL(targetRepName, idf.TargetFile)

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)
			idf.IsAudio = isAudio(idf.TargetFile)
			idf.IsZip = isZip(idf.TargetFile)

			idf.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			idf.CreateTime = time.Unix(createTimeUnix, 0).Local()
			idf.UpdateTime = time.Unix(updateTimeUnix, 0).Local()

			return &idf, nil
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return nil, nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetIDFKyouHistories(ctx context.Context, id string) ([]IDFKyou, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	var err error
	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  CONTENT_PATH,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + sqlite3impl.QuoteIdent(i.dbName) + `
WHERE
`

	ids := []string{id}
	query := &find.FindQuery{
		IDs: ids,
	}

	dataType := "idf"
	queryArgs := []any{
		dataType,
	}

	tableName := i.dbName
	tableNameAlias := i.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"ID"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := false
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get idf histories sql: %w", err)
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
		err = fmt.Errorf("error at select from idf: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	idfKyous := []IDFKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			idf := IDFKyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			targetRepName := ""

			err = rows.Scan(
				&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeUnix,
				&createTimeUnix,
				&idf.CreateApp,
				&idf.CreateDevice,
				&idf.CreateUser,
				&updateTimeUnix,
				&idf.UpdateApp,
				&idf.UpdateDevice,
				&idf.UpdateUser,
				&idf.ContentPath,
				&idf.RepName,
				&idf.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from idf: %w", err)
				return nil, err
			}

			// targetRepNameが空の場合はRepNameにフォールバック
			if targetRepName == "" || targetRepName == "." {
				targetRepName = idf.RepName
			}
			// 対象IDFRepsからファイルURLを取得（targetRepName解決後に構築）
			idf.FileURL = buildIDFFileURL(targetRepName, idf.TargetFile)

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)
			idf.IsAudio = isAudio(idf.TargetFile)
			idf.IsZip = isZip(idf.TargetFile)

			idf.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			idf.CreateTime = time.Unix(createTimeUnix, 0).Local()
			idf.UpdateTime = time.Unix(updateTimeUnix, 0).Local()

			idfKyous = append(idfKyous, idf)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return idfKyous, nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) IDF(ctx context.Context) error {
	return i.idfRep.IDF(ctx)
}

func (i *idfKyouRepositoryCachedSQLite3Impl) AddIDFKyouInfo(ctx context.Context, idfKyou IDFKyou) error {
	i.m.Lock()
	defer i.m.Unlock()
	queryArgs := []any{
		idfKyou.IsDeleted,
		idfKyou.ID,
		idfKyou.RepName,
		idfKyou.TargetFile,
		idfKyou.CreateApp,
		// 列は CREATE_APP, CREATE_USER, CREATE_DEVICE の順。**引数もこの順に揃えること。**
		// 以前は CreateDevice, CreateUser の順に渡していて、作成ユーザと作成端末が
		// 入れ替わったままキャッシュへ入り、読み出しでも入れ替わって返っていた
		// （--cache_in_memory の既定は true なので通常の経路。実DBは無傷）
		idfKyou.CreateUser,
		idfKyou.CreateDevice,
		idfKyou.UpdateApp,
		idfKyou.UpdateDevice,
		idfKyou.UpdateUser,
		idfKyou.ContentPath,
		idfKyou.RepName,
		idfKyou.RelatedTime.Unix(),
		idfKyou.CreateTime.Unix(),
		idfKyou.UpdateTime.Unix(),
	}

	gkill_log.LogSQLQuery(ctx, i.addIDFKyouInfoSQL, queryArgs)
	_, err := i.addIDFKyouInfoStmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to idf %s: %w", idfKyou.ID, err)
		return err
	}
	return nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) HandleFileServe(w http.ResponseWriter, r *http.Request) {
	i.idfRep.HandleFileServe(w, r)
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GenerateThumbCache(ctx context.Context) error {
	return i.idfRep.GenerateThumbCache(ctx)
}

func (i *idfKyouRepositoryCachedSQLite3Impl) ClearThumbCache(userID string) error {
	return i.idfRep.ClearThumbCache(userID)
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GenerateVideoCache(ctx context.Context) error {
	return i.idfRep.GenerateVideoCache(ctx)
}

func (i *idfKyouRepositoryCachedSQLite3Impl) ClearVideoCache(userID string) error {
	return i.idfRep.ClearVideoCache(userID)
}

func (i *idfKyouRepositoryCachedSQLite3Impl) ClearZipCache(userID string) error {
	return i.idfRep.ClearZipCache(userID)
}

func (i *idfKyouRepositoryCachedSQLite3Impl) UnWrapTyped() ([]IDFKyouRepository, error) {
	return i.idfRep.UnWrapTyped()
}
func (i *idfKyouRepositoryCachedSQLite3Impl) UnWrap() ([]Repository, error) {
	return i.idfRep.UnWrap()
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	repName, err := i.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	sql := `
SELECT IS_DELETED, ID AS TARGET_ID, NULL AS TARGET_ID_IN_DATA,
       ? AS LATEST_DATA_REPOSITORY_NAME, UPDATE_TIME_UNIX AS DATA_UPDATE_TIME_UNIX
FROM ` + sqlite3impl.QuoteIdent(i.dbName) + ` AS T
WHERE T.UPDATE_TIME_UNIX = (SELECT MAX(UPDATE_TIME_UNIX) FROM ` + sqlite3impl.QuoteIdent(i.dbName) + ` AS INNER_TABLE WHERE INNER_TABLE.ID = T.ID)
`
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
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
	return latestDataRepositoryAddresses, nil
}
