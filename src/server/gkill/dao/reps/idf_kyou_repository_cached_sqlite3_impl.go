package reps

import (
	"context"
	sqllib "database/sql"
	"fmt"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create IDF table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_UNIX") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + ` (ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", indexUnixSQL))
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", indexUnixSQL))
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", addIDFKyouInfoSQL))
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
	// 結果は map[string][]Kyou に収めるので、SQL側で並べても順序は捨てられる。
	// 最終的な並び順は find_filter の Go 側ソートで決まる。
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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

	words := []string{}
	notWords := []string{}
	// query は全repで共有されているので、スライスの中身を直接書き換えると
	// 並列に走っている他repの検索語まで小文字化して壊してしまう。必ず複製すること。
	if query.Words != nil {
		words = make([]string, len(query.Words))
		for i, word := range query.Words {
			words[i] = strings.ToLower(word)
		}
	}
	if query.NotWords != nil {
		notWords = make([]string, len(query.NotWords))
		for i, notWord := range query.NotWords {
			notWords[i] = strings.ToLower(notWord)
		}
	}

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

			// 判定OKであれば追加する
			// ファイルの内容を取得する
			fileContentText := ""
			filename := idf.ContentPath
			if filename == "" {
				// err = fmt.Errorf("error at get path %s: %w", idf.ID, err)
				// return nil, err

				// 接続されていないRepのIDがあったときは無視する
				continue
			}
			if query.UseWords {
				fileContentText += strings.ToLower(filename)
				switch filepath.Ext(idf.TargetFile) {
				case ".md":
					fallthrough
				case ".txt":
					file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
					if err != nil {
						err = fmt.Errorf("error at open file %s: %w", filename, err)
						return nil, err
					}
					b, err := io.ReadAll(file)
					file.Close()
					if err != nil {
						err = fmt.Errorf("error at read all file content %s: %w", filename, err)
						return nil, err
					}
					fileContentText += strings.ToLower(string(b))
				}
			}

			match := true
			if query.UseWords {
				// ワードand検索である場合の判定
				if query.WordsAnd {
					match = true
					for _, word := range words {
						match = strings.Contains(fileContentText, word)
						if !match {
							break
						}
					}
					if !match {
						continue
					}
				} else if !(query.WordsAnd) {
					// ワードor検索である場合の判定
					match = false
					for _, word := range words {
						match = strings.Contains(fileContentText, word)
						if match {
							break
						}
					}
				}
				// notワードを除外する場合の判定
				for _, notWord := range notWords {
					match = strings.Contains(fileContentText, notWord)
					if match {
						match = false
						break
					}
				}
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

				if _, exist := kyous[kyou.ID]; !exist {
					kyous[kyou.ID] = []Kyou{}
				}
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
		UseIDs:         true,
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UseUpdateTime:  updateTime != nil,
		UpdateTime:     updateTime,
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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
	return &kyous[0], nil
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
		UseIDs: true,
		IDs:    ids,
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
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
					idfKyou.CreateDevice,
					idfKyou.CreateUser,
					idfKyou.UpdateApp,
					idfKyou.UpdateDevice,
					idfKyou.UpdateUser,
					idfKyou.ContentPath,
					idfKyou.RepName,
					idfKyou.RelatedTime.Unix(),
					idfKyou.CreateTime.Unix(),
					idfKyou.UpdateTime.Unix(),
				}

				slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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
	ignoreCase := false

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)

	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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

	words := []string{}
	notWords := []string{}
	// query は全repで共有されているので、スライスの中身を直接書き換えると
	// 並列に走っている他repの検索語まで小文字化して壊してしまう。必ず複製すること。
	if query.Words != nil {
		words = make([]string, len(query.Words))
		for i, word := range query.Words {
			words[i] = strings.ToLower(word)
		}
	}
	if query.NotWords != nil {
		notWords = make([]string, len(query.NotWords))
		for i, notWord := range query.NotWords {
			notWords[i] = strings.ToLower(notWord)
		}
	}

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

			// ファイルの内容を取得する
			fileContentText := ""
			filename := idf.ContentPath
			if filename == "" {
				err = fmt.Errorf("error at get path %s: %w", idf.ID, err)
				return nil, err
			}
			if query.UseWords {
				fileContentText += filename
				switch filepath.Ext(idf.TargetFile) {
				case ".md":
					fallthrough
				case ".txt":
					file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
					if err != nil {
						err = fmt.Errorf("error at open file %s: %w", filename, err)
						return nil, err
					}
					b, err := io.ReadAll(file)
					file.Close()
					if err != nil {
						err = fmt.Errorf("error at read all file content %s: %w", filename, err)
						return nil, err
					}
					fileContentText += strings.ToLower(string(b))
				}
			}

			match := true
			if query.UseWords {
				// ワードand検索である場合の判定
				if query.WordsAnd {
					match = true
					for _, word := range words {
						match = strings.Contains(fileContentText, word)
						if !match {
							break
						}
					}
					if !match {
						continue
					}
				} else if !(query.WordsAnd) {
					// ワードor検索である場合の判定
					match = false
					for _, word := range words {
						match = strings.Contains(fileContentText, word)
						if match {
							break
						}
					}
				}
				// notワードを除外する場合の判定
				for _, notWord := range notWords {
					match = strings.Contains(fileContentText, notWord)
					if match {
						match = false
						break
					}
				}
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
		UseIDs:         true,
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UseUpdateTime:  updateTime != nil,
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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
		UseIDs: true,
		IDs:    ids,
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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
		idfKyou.CreateDevice,
		idfKyou.CreateUser,
		idfKyou.UpdateApp,
		idfKyou.UpdateDevice,
		idfKyou.UpdateUser,
		idfKyou.ContentPath,
		idfKyou.RepName,
		idfKyou.RelatedTime.Unix(),
		idfKyou.CreateTime.Unix(),
		idfKyou.UpdateTime.Unix(),
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", i.addIDFKyouInfoSQL), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
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
