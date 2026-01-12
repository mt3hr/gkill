package reps

import (
	"context"
	"database/sql"
	sqllib "database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type idfKyouRepositoryCachedSQLite3Impl struct {
	dbName   string
	idfRep   IDFKyouRepository
	cachedDB *sqllib.DB
	m        *sync.Mutex
}

func NewIDFCachedRep(ctx context.Context, idfRep IDFKyouRepository, cacheDB *sql.DB, m *sync.Mutex, dbName string) (IDFKyouRepository, error) {
	if m == nil {
		m = &sync.Mutex{}
	}
	sql := `
CREATE TABLE IF NOT EXISTS "` + dbName + `" (
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

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create IDF table statement %s: %w", dbName, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create IDF table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `_UNIX" ON "` + dbName + `" (ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	gkill_log.TraceSQL.Printf("sql: %s", indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create IDF index unix statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexUnixStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexUnixSQL)
	_, err = indexUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create IDF index unix to %s: %w", dbName, err)
		return nil, err
	}

	rep := &idfKyouRepositoryCachedSQLite3Impl{
		dbName:   dbName,
		idfRep:   idfRep,
		cachedDB: cacheDB,
		m:        m,
	}
	return rep, nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	i.m.Lock()
	i.m.Unlock()
	var err error
	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = i.UpdateCache(ctx)
		if err != nil {
			repName, _ := i.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

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
FROM ` + i.dbName + `
WHERE
`
	dataType := "idf"
	queryArgs := []interface{}{
		dataType,
	}

	tableName := i.dbName
	tableNameAlias := i.dbName
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TARGET_FILE"}
	ignoreFindWord := true
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := false
	if query.OnlyLatestData != nil {
		onlyLatestData = *query.OnlyLatestData
	} else {
		onlyLatestData = false
	}
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyou sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from idf: %w", err)
		return nil, err
	}
	defer rows.Close()

	kyous := map[string][]*Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			idf := &IDFKyou{}
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			targetRepName := ""

			err = rows.Scan(
				&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeUnix,
				&createTimeUnix,
				&idf.CreateDevice,
				&idf.CreateApp,
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

			// 対象IDFRepsからファイルURLを取得
			idf.FileURL = fmt.Sprintf("/files/%s/%s", targetRepName, filepath.Base(idf.TargetFile))

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)
			idf.IsAudio = isAudio(idf.TargetFile)

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
			if query.UseWords != nil && *query.UseWords {
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

			words := []string{}
			notWords := []string{}
			if query.Words != nil {
				words = *query.Words
				for i := range words {
					words[i] = strings.ToLower(words[i])
				}
			}
			if query.NotWords != nil {
				notWords = *query.NotWords
				for i := range notWords {
					notWords[i] = strings.ToLower(notWords[i])
				}
			}

			match := true
			if query.UseWords != nil && *query.UseWords {
				// ワードand検索である場合の判定
				if query.WordsAnd != nil && *query.WordsAnd {
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
				} else if query.WordsAnd != nil && !(*query.WordsAnd) {
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
				kyou := &Kyou{}
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

				if _, exist := kyous[kyou.ID]; !exist {
					kyous[kyou.ID] = []*Kyou{}
				}
				kyous[kyou.ID] = append(kyous[kyou.ID], kyou)
			}
		}
	}
	return kyous, nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := i.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from idf %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(kyouHistories) == 0 {
		return nil, nil
	}

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range kyouHistories {
			if kyou.UpdateTime.Unix() == updateTime.Unix() {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return kyouHistories[0], nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	i.m.Lock()
	i.m.Unlock()
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
FROM ` + i.dbName + `
WHERE
`

	dataType := "idf"
	queryArgs := []interface{}{
		dataType,
	}

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
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

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from idf: %w", err)
		return nil, err
	}
	defer rows.Close()

	kyous := []*Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			idf := &IDFKyou{}
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

			idf.FileURL = fmt.Sprintf("/files/%s/%s", targetRepName, filepath.Base(idf.TargetFile))

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)
			idf.IsAudio = isAudio(idf.TargetFile)

			idf.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			idf.CreateTime = time.Unix(createTimeUnix, 0).Local()
			idf.UpdateTime = time.Unix(updateTimeUnix, 0).Local()

			kyou := &Kyou{}
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

			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return i.idfRep.GetPath(ctx, id)
}

func (i *idfKyouRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	trueValue := true
	falseValue := false
	query := &find.FindQuery{
		UpdateCache:    &trueValue,
		OnlyLatestData: &falseValue,
	}

	allIDFKyous, err := i.idfRep.FindIDFKyou(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all idf kyou at update cache: %w", err)
		return err
	}

	i.m.Lock()
	defer i.m.Unlock()

	tx, err := i.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add idf kyou: %w", err)
		return err
	}

	sql := `DELETE FROM ` + i.dbName
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create idf kyou table statement %s: %w", "memory", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at delete idf kyou table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + i.dbName + ` (
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
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add idf sql: %w", err)
		return err
	}
	defer insertStmt.Close()

	for _, idfKyou := range allIDFKyous {
		select {
		case <-ctx.Done():
			tx.Rollback()
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			queryArgs := []interface{}{
				idfKyou.IsDeleted,
				idfKyou.ID,
				idfKyou.RepName,
				idfKyou.TargetFile,
				idfKyou.CreateApp,
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

			gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to idf %s: %w", idfKyou.ID, err)
				return err
			}
			return nil
		}()
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit transaction for add idf kyous: %w", err)
		return err
	}

	return nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return i.idfRep.GetRepName(ctx)
}

func (i *idfKyouRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
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
		_, err = i.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+i.dbName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]*IDFKyou, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = i.UpdateCache(ctx)
		if err != nil {
			repName, _ := i.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

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
FROM ` + i.dbName + `
WHERE
`
	dataType := "idf"
	queryArgs := []interface{}{
		dataType,
	}

	tableName := i.dbName
	tableNameAlias := i.dbName
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{"TARGET_FILE"}
	ignoreFindWord := true
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := false
	if query.OnlyLatestData != nil {
		onlyLatestData = *query.OnlyLatestData
	} else {
		onlyLatestData = false
	}
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)

	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyou sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from idf: %w", err)
		return nil, err
	}
	defer rows.Close()

	idfKyous := []*IDFKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			idf := &IDFKyou{}
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

			// 対象IDFRepsからファイルURLを取得
			idf.FileURL = fmt.Sprintf("/files/%s/%s", targetRepName, filepath.Base(idf.TargetFile))

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)
			idf.IsAudio = isAudio(idf.TargetFile)

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
			if query.UseWords != nil && *query.UseWords {
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

			words := []string{}
			notWords := []string{}
			if query.Words != nil {
				words = *query.Words
			}
			if query.NotWords != nil {
				notWords = *query.NotWords
			}

			match := true
			if query.UseWords != nil && *query.UseWords {
				// ワードand検索である場合の判定
				if query.WordsAnd != nil && *query.WordsAnd {
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
				} else if query.WordsAnd != nil && !(*query.WordsAnd) {
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
	return idfKyous, nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetIDFKyou(ctx context.Context, id string, updateTime *time.Time) (*IDFKyou, error) {
	// 最新のデータを返す
	idfHistories, err := i.GetIDFKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get idf kyou histories from IDF %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(idfHistories) == 0 {
		return nil, nil
	}

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range idfHistories {
			if kyou.UpdateTime.Unix() == updateTime.Unix() {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return idfHistories[0], nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) GetIDFKyouHistories(ctx context.Context, id string) ([]*IDFKyou, error) {
	i.m.Lock()
	i.m.Unlock()
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
FROM ` + i.dbName + `
WHERE 
`

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	dataType := "idf"
	queryArgs := []interface{}{
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

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get idf histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from idf: %w", err)
		return nil, err
	}
	defer rows.Close()

	idfKyous := []*IDFKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			idf := &IDFKyou{}
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

			idf.FileURL = fmt.Sprintf("/files/%s/%s", targetRepName, filepath.Base(idf.TargetFile))

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)
			idf.IsAudio = isAudio(idf.TargetFile)

			idf.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			idf.CreateTime = time.Unix(createTimeUnix, 0).Local()
			idf.UpdateTime = time.Unix(updateTimeUnix, 0).Local()

			idfKyous = append(idfKyous, idf)
		}
	}
	return idfKyous, nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) IDF(ctx context.Context) error {
	panic("not implemented")
}

func (i *idfKyouRepositoryCachedSQLite3Impl) AddIDFKyouInfo(ctx context.Context, idfKyou *IDFKyou) error {
	sql := `
INSERT INTO ` + i.dbName + ` (
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
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := i.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add idf sql %s: %w", idfKyou.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		idfKyou.IsDeleted,
		idfKyou.ID,
		idfKyou.RepName,
		idfKyou.TargetFile,
		idfKyou.CreateApp,
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

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to idf %s: %w", idfKyou.ID, err)
		return err
	}
	return nil
}

func (i *idfKyouRepositoryCachedSQLite3Impl) HandleFileServe(w http.ResponseWriter, r *http.Request) {
	i.idfRep.HandleFileServe(w, r)
}

func (i *idfKyouRepositoryCachedSQLite3Impl) UnWrapTyped() ([]IDFKyouRepository, error) {
	return i.idfRep.UnWrapTyped()
}
func (i *idfKyouRepositoryCachedSQLite3Impl) UnWrap() ([]Repository, error) {
	return i.idfRep.UnWrap()
}
