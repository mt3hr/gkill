package reps

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type idfKyouRepositorySQLite3Impl struct {
	repositoriesRef *GkillRepositories
	idDBFile        string
	contentDir      string
	rootAddress     string
	r               *mux.Router

	autoIDF   bool
	idfIgnore *[]string

	db          *sql.DB
	m           *sync.Mutex
	fullConnect bool

	fileServer     http.Handler
	thumbServer    http.Handler
	thumbGenerator ThumbGenerator
}

const DUIDLayout = "20060102T150405-0700"

type fileinfo struct {
	Filename string
	Lastmod  time.Time
}

// NewIDFDirRep .
// id.dbと関連づいたディレクトリによるrykv.Repの実装
// dir: ディレクトリ
// idBDFile: ディレクトリと関連付けられたid.DBファイル。（通常は dir/.kyou/id.db を指定する）
// r: ファイルサーバーをハンドルするrouter。 /files/filepath.Base(dir)/ でハンドルされる
// autoIDF: trueにするとGetAllKyous()が呼び出されるたびにidfする
// idfIgnore: autoIDFが有効なとき、idfの対象にしないファイル名パターン
// idfRecurse: autoIDFが有効なとき、サブディレクトリなどに対してもidfをする場合はtrueを指定する
func NewIDFDirRep(ctx context.Context, dir, dbFilename string, fullConnect bool, r *mux.Router, autoIDF bool, idfIgnore *[]string, repositoriesRef *GkillRepositories) (IDFKyouRepository, error) {
	filename := dbFilename

	db, err := sqlite3impl.GetSQLiteDBConnection(ctx, filename)
	if err != nil {
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "IDF" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_REP_NAME,
  TARGET_FILE NOT NULL,
  RELATED_TIME NOT NULL,
  CREATE_TIME NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_TIME NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL 
)
`

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create IDF table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create IDF table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_IDF ON IDF (ID, RELATED_TIME, UPDATE_TIME);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create IDF index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create IDF index to %s: %w", filename, err)
		return nil, err
	}

	dbName := "IDF"
	latestIndexSQL := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS INDEX_FOR_LATEST_DATA_REPOSITORY_ADDRESS ON %s(ID, UPDATE_TIME);`, dbName)
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s", latestIndexSQL)
	latestIndexStmt, err := db.PrepareContext(ctx, latestIndexSQL)
	if err != nil {
		err = fmt.Errorf("error at create index for latest data repository address at %s index statement %s: %w", dbName, filename, err)
		return nil, err
	}
	defer latestIndexStmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s", latestIndexSQL)
	_, err = latestIndexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create %s index for latest data repository address to %s: %w", dbName, filename, err)
		return nil, err
	}

	if !fullConnect {
		err = db.Close()
		if err != nil {
			return nil, err
		}
		db = nil
	}

	rep := &idfKyouRepositorySQLite3Impl{
		repositoriesRef: repositoriesRef,
		idDBFile:        dbFilename,
		contentDir:      dir,
		rootAddress:     "/files/" + filepath.Base(dir) + "/",
		r:               r,
		autoIDF:         autoIDF,
		idfIgnore:       idfIgnore,
		db:              db,
		m:               &sync.Mutex{},
		fullConnect:     fullConnect,
	}

	fs := http.FileServer(http.Dir(dir))
	rep.fileServer = fs
	rep.thumbServer = NewThumbFileServer(dir, fs)
	rep.thumbGenerator = rep.thumbServer.(ThumbGenerator)

	return rep, nil
}

func (i *idfKyouRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	var err error
	var db *sql.DB
	if i.fullConnect {
		db = i.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, i.idDBFile)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

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
FROM IDF
WHERE
`
	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at idf : %w", err)
		return nil, err
	}

	dataType := "idf"
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "IDF"
	tableNameAlias := "IDF"
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TARGET_FILE"}
	ignoreFindWord := true
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := true
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyou sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
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
			idf.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			targetRepName := ""

			err = rows.Scan(
				&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeStr,
				&createTimeStr,
				&idf.CreateApp,
				&idf.CreateDevice,
				&idf.CreateUser,
				&updateTimeStr,
				&idf.UpdateApp,
				&idf.UpdateDevice,
				&idf.UpdateUser,
				&idf.RepName,
				&idf.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from idf: %w", err)
				return nil, err
			}
			if targetRepName == "" || targetRepName == "." {
				targetRepName = repName
			}

			idf.ContentPath, err = i.GetPath(ctx, idf.ID)
			if err != nil {
				err = fmt.Errorf("error at get path %s: %w", idf.ID, err)
				return nil, err
			}

			// 対象IDFRepsからファイルURLを取得
			idf.FileURL = fmt.Sprintf("/files/%s/%s", targetRepName, filepath.Base(idf.TargetFile))

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)
			idf.IsAudio = isAudio(idf.TargetFile)

			idf.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in idf: %w", relatedTimeStr, err)
				return nil, err
			}
			idf.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in idf: %w", createTimeStr, err)
				return nil, err
			}
			idf.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in idf: %w", updateTimeStr, err)
				return nil, err
			}

			// 判定OKであれば追加する
			// ファイルの内容を取得する
			var targetRep Repository
			if targetRepName == "" || targetRepName == repName {
				targetRep = i
			} else {
				for _, rep := range i.repositoriesRef.Reps {
					repName, err := rep.GetRepName(ctx)
					if err != nil {
						err = fmt.Errorf("error at get rep name: %w", err)
						return nil, err
					}
					if repName == targetRepName {
						targetRep = rep
					}
				}
			}
			fileContentText := ""
			filename, err := targetRep.GetPath(ctx, idf.ID)
			if err != nil {
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
				kyou.IsVideo = idf.IsVideo

				if _, exist := kyous[kyou.ID]; !exist {
					kyous[kyou.ID] = []*Kyou{}
				}
				kyous[kyou.ID] = append(kyous[kyou.ID], kyou)
			}
		}
	}
	return kyous, nil
}

func (i *idfKyouRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
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

func (i *idfKyouRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	var err error
	var db *sql.DB
	if i.fullConnect {
		db = i.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, i.idDBFile)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}
	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
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
FROM IDF
WHERE
`

	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at idf : %w", err)
		return nil, err
	}

	dataType := "idf"
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	tableName := "IDF"
	tableNameAlias := "IDF"
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TARGET_FILE"}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		err = fmt.Errorf("error at generate find sql common: %w", err)
		return nil, err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
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
			idf.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			targetRepName := ""

			err = rows.Scan(
				&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeStr,
				&createTimeStr,
				&idf.CreateApp,
				&idf.CreateDevice,
				&idf.CreateUser,
				&updateTimeStr,
				&idf.UpdateApp,
				&idf.UpdateDevice,
				&idf.UpdateUser,
				&idf.RepName,
				&idf.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from idf: %w", err)
				return nil, err
			}

			idf.ContentPath, err = i.GetPath(ctx, idf.ID)
			if err != nil {
				err = fmt.Errorf("error at get path %s: %w", idf.ID, err)
				return nil, err
			}

			idf.FileURL = fmt.Sprintf("/files/%s/%s", targetRepName, filepath.Base(idf.TargetFile))

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)
			idf.IsAudio = isAudio(idf.TargetFile)

			idf.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in idf: %w", relatedTimeStr, err)
				return nil, err
			}
			idf.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in idf: %w", createTimeStr, err)
				return nil, err
			}
			idf.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in idf: %w", updateTimeStr, err)
				return nil, err
			}

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
			kyou.IsVideo = idf.IsVideo

			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (i *idfKyouRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if id == "" {
		return i.idDBFile, nil
	}
	var err error
	var db *sql.DB
	if i.fullConnect {
		db = i.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, i.idDBFile)
		if err != nil {
			return "", err
		}
		defer db.Close()
	}
	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
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
FROM IDF
WHERE
`
	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at idf : %w", err)
		return "", err
	}

	dataType := "idf"

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "IDF"
	tableNameAlias := "IDF"
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"ID"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := false
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return "", err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return "", err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from idf: %w", err)
		return "", err
	}
	defer rows.Close()

	idfKyous := []*IDFKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			idf := &IDFKyou{}
			idf.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			targetRepName := ""

			err = rows.Scan(&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeStr,
				&createTimeStr,
				&idf.CreateApp,
				&idf.CreateDevice,
				&idf.CreateUser,
				&updateTimeStr,
				&idf.UpdateApp,
				&idf.UpdateDevice,
				&idf.UpdateUser,
				&idf.RepName,
				&idf.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from idf: %w", err)
				_ = err
				return "", nil
			}

			idf.FileURL = fmt.Sprintf("/files/%s/%s", targetRepName, filepath.Base(idf.TargetFile))

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)
			idf.IsAudio = isAudio(idf.TargetFile)

			idf.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in idf: %w", relatedTimeStr, err)
				return "", err
			}
			idf.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in idf: %w", createTimeStr, err)
				return "", err
			}
			idf.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in idf: %w", updateTimeStr, err)
				return "", err
			}

			idfKyous = append(idfKyous, idf)
		}
	}

	if len(idfKyous) == 0 {
		repName, _ := i.GetRepName(ctx)
		err := fmt.Errorf("not found %s in %s", id, repName)
		return "", err
	}
	sort.Slice(idfKyous, func(i, j int) bool {
		return idfKyous[i].UpdateTime.After(idfKyous[j].UpdateTime)
	})
	filename := filepath.Join(i.contentDir, idfKyous[0].TargetFile)
	return filename, nil
}

func (i *idfKyouRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	if i.autoIDF {
		err := i.IDF(ctx)
		if err != nil {
			repName, _ := i.GetRepName(ctx)
			err = fmt.Errorf("error at idf %s: %w", repName, err)
			return err
		}
	}
	return nil
}

func (i *idfKyouRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return filepath.Base(i.contentDir), nil
}

func (i *idfKyouRepositorySQLite3Impl) Close(ctx context.Context) error {
	if i.fullConnect {
		return i.db.Close()
	}
	return nil
}

func (i *idfKyouRepositorySQLite3Impl) FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]*IDFKyou, error) {
	var err error
	var db *sql.DB
	if i.fullConnect {
		db = i.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, i.idDBFile)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

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
FROM IDF
WHERE
`
	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at idf : %w", err)
		return nil, err
	}

	dataType := "idf"
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "IDF"
	tableNameAlias := "IDF"
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"TARGET_FILE"}
	ignoreFindWord := true
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := true
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyou sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
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
			idf.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			targetRepName := ""

			err = rows.Scan(
				&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeStr,
				&createTimeStr,
				&idf.CreateApp,
				&idf.CreateDevice,
				&idf.CreateUser,
				&updateTimeStr,
				&idf.UpdateApp,
				&idf.UpdateDevice,
				&idf.UpdateUser,
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

			idf.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in idf: %w", relatedTimeStr, err)
				return nil, err
			}
			idf.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in idf: %w", createTimeStr, err)
				return nil, err
			}
			idf.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in idf: %w", updateTimeStr, err)
				return nil, err
			}

			// 判定OKであれば追加する
			// ファイルの内容を取得する
			if idf.RepName != repName && idf.RepName != "" {
				continue
			}

			idf.ContentPath, err = i.GetPath(ctx, idf.ID)
			if err != nil {
				err = fmt.Errorf("error at get path %s: %w", idf.ID, err)
				return nil, err
			}

			fileContentText := ""

			filename, err := i.GetPath(ctx, idf.ID)
			if err != nil {
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

func (i *idfKyouRepositorySQLite3Impl) GetIDFKyou(ctx context.Context, id string, updateTime *time.Time) (*IDFKyou, error) {
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

func (i *idfKyouRepositorySQLite3Impl) GetIDFKyouHistories(ctx context.Context, id string) ([]*IDFKyou, error) {
	var err error
	var db *sql.DB
	if i.fullConnect {
		db = i.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, i.idDBFile)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}
	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
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
FROM IDF
WHERE 
`

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}

	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at idf : %w", err)
		return nil, err
	}

	dataType := "idf"
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "IDF"
	tableNameAlias := "IDF"
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{"ID"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := false
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get idf histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
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
			idf.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			targetRepName := ""

			err = rows.Scan(
				&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&idf.TargetFile,
				&relatedTimeStr,
				&createTimeStr,
				&idf.CreateApp,
				&idf.CreateDevice,
				&idf.CreateUser,
				&updateTimeStr,
				&idf.UpdateApp,
				&idf.UpdateDevice,
				&idf.UpdateUser,
				&idf.RepName,
				&idf.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from idf: %w", err)
				return nil, err
			}

			idf.ContentPath, err = i.GetPath(ctx, idf.ID)
			if err != nil {
				err = fmt.Errorf("error at get path %s: %w", idf.ID, err)
				return nil, err
			}

			idf.FileURL = fmt.Sprintf("/files/%s/%s", targetRepName, filepath.Base(idf.TargetFile))

			// 画像であるか判定
			idf.IsImage = isImage(idf.TargetFile)
			idf.IsVideo = isVideo(idf.TargetFile)
			idf.IsAudio = isAudio(idf.TargetFile)

			idf.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in idf: %w", relatedTimeStr, err)
				return nil, err
			}
			idf.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in idf: %w", createTimeStr, err)
				return nil, err
			}
			idf.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in idf: %w", updateTimeStr, err)
				return nil, err
			}

			idfKyous = append(idfKyous, idf)
		}
	}
	return idfKyous, nil
}

func (i *idfKyouRepositorySQLite3Impl) IDF(ctx context.Context) error {
	allIDFKyous, err := i.FindIDFKyou(ctx, &find.FindQuery{})
	if err != nil {
		err = fmt.Errorf("error at find idf kyou: %w", err)
		return err
	}

	contentDirAbs, err := filepath.Abs(i.contentDir)
	if err != nil {
		err = fmt.Errorf("error at get abs path %s: %w", i.contentDir, err)
		return err
	}
	contentDirAbs = filepath.Clean(contentDirAbs)
	contentDirAbs = filepath.ToSlash(contentDirAbs)
	// 対象内のファイルfullPath
	existFileInfos := map[string]*fileinfo{}
	err = filepath.WalkDir(contentDirAbs, fs.WalkDirFunc(func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		path = filepath.ToSlash(path)
		path = strings.TrimPrefix(path, contentDirAbs+"/")
		for _, ignore := range *i.idfIgnore {
			if filepath.Base(path) == ignore {
				return nil
			}
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		info.ModTime()
		existFileInfos[path] = &fileinfo{
			Filename: path,
			Lastmod:  info.ModTime(),
		}
		return nil
	}))
	if err != nil {
		err = fmt.Errorf("error at walk dir at %s: %w", contentDirAbs, err)
		return err
	}

	// まだidfされていないやつをリストアップする
	existing := map[string]struct{}{}
	for _, idf := range allIDFKyous {
		existing[idf.TargetFile] = struct{}{}
	}

	idfTargetList := map[string]struct{}{}
	for path := range existFileInfos {
		if _, ok := existing[path]; !ok {
			idfTargetList[path] = struct{}{}
		}
	}

	// 対象をidfする
	idfKyous := []*IDFKyou{}
	now := time.Now()
	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name: %w", err)
		return err
	}

	for idfTargetFileName := range idfTargetList {
		lastMod := now
		for _, existFileInfo := range existFileInfos {
			if existFileInfo.Filename == idfTargetFileName {
				lastMod = existFileInfo.Lastmod
			}
		}

		trimedFileName := filepath.Clean(idfTargetFileName)
		trimedFileName = filepath.ToSlash(trimedFileName)
		trimedFileName = strings.TrimPrefix(trimedFileName, contentDirAbs)
		trimedFileName = strings.TrimPrefix(trimedFileName, "/")
		if trimedFileName == "" {
			continue
		}
		if i.isDUID(trimedFileName) {
			id, time, err := i.parseDUID(trimedFileName)
			if err != nil {
				err = fmt.Errorf("error at parseDUID %s:%w", trimedFileName, err)
				return err
			}

			idf := &IDFKyou{}
			idf.IsDeleted = false
			idf.ID = id.String()
			idf.RepName = repName
			idf.RelatedTime = time
			idf.DataType = "idf"
			idf.CreateTime = now
			idf.CreateApp = "idf"
			idf.CreateDevice = ""
			idf.CreateUser = "idf"
			idf.UpdateTime = now
			idf.UpdateApp = "idf"
			idf.UpdateUser = ""
			idf.UpdateDevice = "idf"
			idf.TargetFile = trimedFileName
			idfKyous = append(idfKyous, idf)
		} else {
			idf := &IDFKyou{}
			idf.IsDeleted = false
			idf.ID = sqlite3impl.GenerateNewID()
			idf.RepName = repName
			idf.RelatedTime = lastMod
			idf.DataType = "idf"
			idf.CreateTime = now
			idf.CreateApp = "idf"
			idf.CreateDevice = ""
			idf.CreateUser = "idf"
			idf.UpdateTime = now
			idf.UpdateApp = "idf"
			idf.UpdateUser = ""
			idf.UpdateDevice = "idf"
			idf.TargetFile = trimedFileName
			idfKyous = append(idfKyous, idf)
		}
	}

	for _, idf := range idfKyous {
		err := i.AddIDFKyouInfo(ctx, idf)
		if err != nil {
			err = fmt.Errorf("error at add idf kyou info %s: %w", idf.ID, err)
			return err
		}
	}
	return nil
}

func (i *idfKyouRepositorySQLite3Impl) AddIDFKyouInfo(ctx context.Context, idfKyou *IDFKyou) error {
	var err error
	var db *sql.DB
	if i.fullConnect {
		db = i.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, i.idDBFile)
		if err != nil {
			return err
		}
		defer db.Close()
	}
	i.m.Lock()
	defer i.m.Unlock()
	sql := `
INSERT INTO IDF (
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
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
  ?,
  ?
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
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
		idfKyou.RelatedTime.Format(sqlite3impl.TimeLayout),
		idfKyou.CreateTime.Format(sqlite3impl.TimeLayout),
		idfKyou.CreateApp,
		idfKyou.CreateDevice,
		idfKyou.CreateUser,
		idfKyou.UpdateTime.Format(sqlite3impl.TimeLayout),
		idfKyou.UpdateApp,
		idfKyou.UpdateDevice,
		idfKyou.UpdateUser,
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to idf %s: %w", idfKyou.ID, err)
		return err
	}
	return nil
}

func (i *idfKyouRepositorySQLite3Impl) isDUID(filename string) bool {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	_, _, err := i.parseDUID(withoutExt)
	return err == nil
}

func (i *idfKyouRepositorySQLite3Impl) parseDUID(str string) (id uuid.UUID, t time.Time, err error) {
	if len(str) != len(DUIDLayout)+36*len("_") {
		err := fmt.Errorf("%s is not duid", str)
		return uuid.UUID{}, time.Time{}, err
	}
	timestr := str[:len(DUIDLayout)]
	idstr := str[len("_")+len(DUIDLayout):]

	id, err = uuid.Parse(idstr)
	if err != nil {
		err = fmt.Errorf("failed to parse uuid %s: %w", idstr, err)
		return uuid.UUID{}, time.Time{}, err
	}
	t, err = time.Parse(DUIDLayout, timestr)
	if err != nil {
		err = fmt.Errorf("failed to parse time %s: %w", timestr, err)
		return uuid.UUID{}, time.Time{}, err
	}
	return id, t, nil
}

func (i *idfKyouRepositorySQLite3Impl) HandleFileServe(w http.ResponseWriter, r *http.Request) {
	if i.thumbServer != nil {
		i.thumbServer.ServeHTTP(w, r)
		return
	}
	http.FileServer(http.Dir(i.contentDir)).ServeHTTP(w, r)
}

func isImage(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".apng",
		".avif",
		".gif",
		".jpg",
		".jpeg",
		".jfif",
		".pjpeg",
		".pjp",
		".png",
		".svg",
		".webp",
		".bmp",
		".ico",
		".cur",
		".tif",
		".tiff",
		".tga",
		".dds",
		".heif",
		".heic",
		".jpe",
		".jif",
		".jfi",
		".jp2",
		".j2k",
		".jpf",
		".jpx",
		".jpm",
		".mj2",
		".xpm",
		".wbmp",
		".xbm",
		".pcx",
		".pnm",
		".pgm",
		".pbm",
		".ppm",
		".pam",
		".pfm":
		return true
	}
	return false
}

func isVideo(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4",
		".webm",
		".avi",
		".mov",
		".mpg",
		".mkv",
		".mwv",
		".flv",
		".asf",
		".f4v",
		".m4v",
		".3gp",
		".3g2",
		".3gp2",
		".3gpp",
		".ogv",
		".ogm",
		".ts",
		".vob",
		".rm",
		".rmvb",
		".wmv",
		".mks",
		".mk3d":
		return true
	}
	return false
}

func isAudio(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case
		".mp3",
		".aac",
		".m4a",
		".m4b",
		".m4p",
		".m4r",
		".ogg",
		".oga",
		".spx",
		".opus",
		".flac",
		".wav",
		".weba",
		".mka",
		".wma":
		return true
	}
	return false
}

func (i *idfKyouRepositorySQLite3Impl) GenerateThumbCache(ctx context.Context) error {
	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at generate thumb cache get rep name: %w", err)
		return err
	}

	query := &find.FindQuery{}
	idfKyous, err := i.FindIDFKyou(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at generate thumb cache at %s: %w", repName, err)
	}

	for _, idfKyou := range idfKyous {
		if !idfKyou.IsImage {
			continue
		}

		rel := filepath.ToSlash(idfKyou.TargetFile)
		rel = strings.TrimPrefix(rel, "/")

		url := &url.URL{
			Scheme: "http",
			Host:   "localhost:9999",
			Path:   "/" + rel,
		}
		query := url.Query()
		query.Set("thumb", "400x400")
		url.RawQuery = query.Encode()

		err = i.thumbGenerator.GenerateThumbCache(ctx, url.String())
		if err != nil {
			err = fmt.Errorf("error at generate thumb cache %s: %w", url.String(), err)
			slog.Log(ctx, gkill_log.Error, "error", err)
			continue
		}
	}
	return nil
}

func (i *idfKyouRepositorySQLite3Impl) ClearThumbCache() error {
	dir := filepath.Clean(os.ExpandEnv(i.contentDir))
	cacheDir := os.ExpandEnv(filepath.Join(gkill_options.CacheDir, "thumb_cache", filepath.Base(dir)))

	os.RemoveAll(cacheDir)
	return nil
}

func (i *idfKyouRepositorySQLite3Impl) UnWrapTyped() ([]IDFKyouRepository, error) {
	return []IDFKyouRepository{i}, nil
}

func (i *idfKyouRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{i}, nil
}
