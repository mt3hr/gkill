package reps

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
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
)

type idfKyouRepositorySQLite3Impl struct {
	repositoriesRef *GkillRepositories
	idDBFile        string
	contentDir      string
	rootAddress     string
	r               *mux.Router

	autoIDF   *bool
	idfIgnore *[]string

	db *sql.DB
	m  *sync.Mutex
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
func NewIDFDirRep(ctx context.Context, dir, dbFilename string, r *mux.Router, autoIDF *bool, idfIgnore *[]string, repositoriesRef *GkillRepositories) (IDFKyouRepository, error) {
	filename := dbFilename

	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
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

	log.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create IDF table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create IDF table to %s: %w", filename, err)
		return nil, err
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
	}

	r.PathPrefix(rep.rootAddress).
		Handler(http.StripPrefix(rep.rootAddress, http.FileServer(http.Dir(dir))))

	return rep, nil
}

func (i *idfKyouRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error) {
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

	whereCounter := 0
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	// UPDATE_TIMEが一番上のものだけを抽出
	sql += `
GROUP BY ID
HAVING MAX(datetime(UPDATE_TIME, 'localtime'))
`

	log.Printf("sql: %s", sql)
	stmt, err := i.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyou sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at idf : %w", err)
		return nil, err
	}

	dataType := "idf"
	rows, err := stmt.QueryContext(ctx, repName, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from idf %s: %w", err)
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
			targetRepName, targetFile := "", ""

			err = rows.Scan(&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&targetFile,
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
			idf.FileName = filepath.Base(targetFile)

			// 対象IDFRepsからファイルURLを取得
			var targetRep Repository
			for _, rep := range i.repositoriesRef.Reps {
				repName, err := rep.GetRepName(ctx)
				if err != nil {
					err = fmt.Errorf("error at get rep name: %w", err)
					return nil, err
				}
				if repName == targetRepName {
					idf.FileURL = fmt.Sprintf("/files/%s/%s", repName, filepath.Base(idf.FileName))
					targetRep = rep
				}
			}

			// 画像であるか判定
			idf.IsImage = false
			ext := strings.ToLower(filepath.Ext(idf.FileName))
			switch ext {
			case ".jpg",
				".jpeg",
				".jfif",
				".png",
				".gif",
				".bmp":
				idf.IsImage = true
			}

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
			fileContentText := ""
			filename, err := targetRep.GetPath(ctx, idf.ID)
			if err != nil {
				err = fmt.Errorf("error at get path %s: %w", idf.ID, err)
				return nil, err
			}
			fileContentText += filename
			switch filepath.Ext(idf.FileName) {
			case ".md":
				fallthrough
			case ".txt":
				file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
				if err != nil {
					err = fmt.Errorf("error at open file %s: %w", filename, err)
					return nil, err
				}
				defer file.Close()
				b, err := io.ReadAll(file)
				if err != nil {
					err = fmt.Errorf("error at read all file content %s: %w", filename, err)
					return nil, err
				}
				fileContentText += strings.ToLower(string(b))
			}

			words := []string{}
			notWords := []string{}
			if query.Words != nil {
				words = *query.Words
			}
			if query.NotWords != nil {
				notWords = *query.NotWords
			}

			// ワードand検索である場合の判定
			match := true
			if query.UseWords != nil && *query.UseWords {
				match = false
				// ワードを解析

				if query.WordsAnd != nil && *query.WordsAnd {
					for _, word := range words {
						match = strings.Contains(fmt.Sprintf("%s", fileContentText), word)
						if !match {
							break
						}
					}
					if !match {
						break
					}
				} else {
					// ワードor検索である場合の判定
					for _, word := range words {
						match = strings.Contains(fmt.Sprintf("%s", fileContentText), word)
						if match {
							break
						}
					}
					if match {
						break
					}
				}

				// notワードを除外する場合の判定
				for _, notWord := range notWords {
					match = strings.Contains(fmt.Sprintf("%s", fileContentText), notWord)
					if match {
						break
					}
				}
				if match {
					break
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

				kyous = append(kyous, kyou)
			}
		}
	}
	sort.Slice(kyous, func(i, j int) bool {
		return kyous[i].RelatedTime.After(kyous[j].RelatedTime)
	})
	return kyous, nil
}

func (i *idfKyouRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
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

	return kyouHistories[0], nil
}

func (i *idfKyouRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	var err error
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
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`

	log.Printf("sql: %s", sql)
	stmt, err := i.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at idf : %w", err)
		return nil, err
	}

	dataType := "idf"
	rows, err := stmt.QueryContext(ctx, repName, dataType, id)
	if err != nil {
		err = fmt.Errorf("error at select from idf %s: %w", err)
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
			targetRepName, targetFile := "", ""

			err = rows.Scan(&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&targetFile,
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
			idf.FileName = filepath.Base(targetFile)

			for _, rep := range i.repositoriesRef.Reps {
				repName, err := rep.GetRepName(ctx)
				if err != nil {
					err = fmt.Errorf("error at get rep name: %w", err)
					return nil, err
				}
				if repName == targetRepName {
					idf.FileURL = fmt.Sprintf("/files/%s/%s", repName, filepath.Base(idf.FileName))
				}
			}

			// 画像であるか判定
			idf.IsImage = false
			ext := strings.ToLower(filepath.Ext(idf.FileName))
			switch ext {
			case ".jpg",
				".jpeg",
				".jfif",
				".png",
				".gif",
				".bmp":
				idf.IsImage = true
			}

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

			kyous = append(kyous, kyou)
		}
	}
	sort.Slice(kyous, func(i, j int) bool {
		return kyous[i].UpdateTime.After(kyous[j].UpdateTime)
	})
	return kyous, nil
}

func (i *idfKyouRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if id == "" {
		return i.contentDir, nil
	}
	var err error
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
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`
	log.Printf("sql: %s", sql)
	stmt, err := i.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return "", err
	}
	defer stmt.Close()

	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at idf : %w", err)
		return "", err
	}

	dataType := "idf"
	rows, err := stmt.QueryContext(ctx, repName, dataType, id)
	if err != nil {
		err = fmt.Errorf("error at select from idf %s: %w", err)
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
			targetRepName, targetFile := "", ""

			err = rows.Scan(&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&targetFile,
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
			idf.FileName = filepath.Base(targetFile)

			for _, rep := range i.repositoriesRef.Reps {
				repName, err := rep.GetRepName(ctx)
				if err != nil {
					err = fmt.Errorf("error at get rep name: %w", err)
					return "", err
				}
				if repName == targetRepName {
					idf.FileURL = fmt.Sprintf("/files/%s/%s", repName, filepath.Base(idf.FileName))
				}
			}

			// 画像であるか判定
			idf.IsImage = false
			ext := strings.ToLower(filepath.Ext(idf.FileName))
			switch ext {
			case ".jpg",
				".jpeg",
				".jfif",
				".png",
				".gif",
				".bmp":
				idf.IsImage = true
			}

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
	sort.Slice(idfKyous, func(i, j int) bool {
		return idfKyous[i].UpdateTime.After(idfKyous[j].UpdateTime)
	})
	if len(idfKyous) == 0 {
		repName, _ := i.GetRepName(ctx)
		err := fmt.Errorf("not found %s in %s", id, repName)
		return "", err
	}

	filename := filepath.Join(i.contentDir, idfKyous[0].FileName)
	return filename, nil
}

func (i *idfKyouRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (i *idfKyouRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := i.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path idf rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	return base, nil
}

func (i *idfKyouRepositorySQLite3Impl) Close(ctx context.Context) error {
	return i.db.Close()
}

func (i *idfKyouRepositorySQLite3Impl) FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]*IDFKyou, error) {
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

	whereCounter := 0
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	// UPDATE_TIMEが一番上のものだけを抽出
	sql += `
GROUP BY ID
HAVING MAX(datetime(UPDATE_TIME, 'localtime'))
`

	log.Printf("sql: %s", sql)
	stmt, err := i.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at find kyou sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at idf : %w", err)
		return nil, err
	}

	dataType := "idf"
	rows, err := stmt.QueryContext(ctx, repName, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from idf %s: %w", err)
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
			targetRepName, targetFile := "", ""

			err = rows.Scan(&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&targetFile,
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
			idf.FileName = filepath.Base(targetFile)

			// 対象IDFRepsからファイルURLを取得
			var targetRep Repository
			for _, rep := range i.repositoriesRef.Reps {
				repName, err := rep.GetRepName(ctx)
				if err != nil {
					err = fmt.Errorf("error at get rep name: %w", err)
					return nil, err
				}
				if repName == targetRepName {
					idf.FileURL = fmt.Sprintf("/files/%s/%s", repName, filepath.Base(idf.FileName))
					targetRep = rep
				}
			}

			// 画像であるか判定
			idf.IsImage = false
			ext := strings.ToLower(filepath.Ext(idf.FileName))
			switch ext {
			case ".jpg",
				".jpeg",
				".jfif",
				".png",
				".gif",
				".bmp":
				idf.IsImage = true
			}

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
			fileContentText := ""
			filename, err := targetRep.GetPath(ctx, idf.ID)
			if err != nil {
				err = fmt.Errorf("error at get path %s: %w", idf.ID, err)
				return nil, err
			}
			fileContentText += filename
			switch filepath.Ext(idf.FileName) {
			case ".md":
				fallthrough
			case ".txt":
				file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
				if err != nil {
					err = fmt.Errorf("error at open file %s: %w", filename, err)
					return nil, err
				}
				defer file.Close()
				b, err := io.ReadAll(file)
				if err != nil {
					err = fmt.Errorf("error at read all file content %s: %w", filename, err)
					return nil, err
				}
				fileContentText += strings.ToLower(string(b))
			}

			// ワードand検索である場合の判定
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
				match = false
				// ワードを解析
				if query.WordsAnd != nil && *query.WordsAnd {
					for _, word := range words {
						match = strings.Contains(fmt.Sprintf("%s", fileContentText), word)
						if !match {
							break
						}
					}
					if !match {
						break
					}
				} else {
					// ワードor検索である場合の判定
					for _, word := range words {
						match = strings.Contains(fmt.Sprintf("%s", fileContentText), word)
						if match {
							break
						}
					}
					if match {
						break
					}
				}

				// notワードを除外する場合の判定
				for _, notWord := range notWords {
					match = strings.Contains(fmt.Sprintf("%s", fileContentText), notWord)
					if match {
						break
					}
				}
				if match {
					break
				}
			}
			if match {
				idfKyous = append(idfKyous, idf)
			}
		}
	}
	sort.Slice(idfKyous, func(i, j int) bool {
		return idfKyous[i].RelatedTime.After(idfKyous[j].RelatedTime)
	})
	return idfKyous, nil
}

func (i *idfKyouRepositorySQLite3Impl) GetIDFKyou(ctx context.Context, id string) (*IDFKyou, error) {
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

	return idfHistories[0], nil
}

func (i *idfKyouRepositorySQLite3Impl) GetIDFKyouHistories(ctx context.Context, id string) ([]*IDFKyou, error) {
	var err error
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
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`

	log.Printf("sql: %s", sql)
	stmt, err := i.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get idf histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at idf : %w", err)
		return nil, err
	}

	dataType := "idf"
	rows, err := stmt.QueryContext(ctx, repName, dataType, id)
	if err != nil {
		err = fmt.Errorf("error at select from idf %s: %w", err)
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
			targetRepName, targetFile := "", ""

			err = rows.Scan(&idf.IsDeleted,
				&idf.ID,
				&targetRepName,
				&targetFile,
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
			idf.FileName = filepath.Base(targetFile)

			for _, rep := range i.repositoriesRef.Reps {
				repName, err := rep.GetRepName(ctx)
				if err != nil {
					err = fmt.Errorf("error at get rep name: %w", err)
					return nil, err
				}
				if repName == targetRepName {
					idf.FileURL = fmt.Sprintf("/files/%s/%s", repName, filepath.Base(idf.FileName))
				}
			}

			// 画像であるか判定
			idf.IsImage = false
			ext := strings.ToLower(filepath.Ext(idf.FileName))
			switch ext {
			case ".jpg",
				".jpeg",
				".jfif",
				".png",
				".gif",
				".bmp":
				idf.IsImage = true
			}

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
	sort.Slice(idfKyous, func(i, j int) bool {
		return idfKyous[i].UpdateTime.After(idfKyous[j].UpdateTime)
	})
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

	// 対象内のファイルfullPath

	existFileInfos := []fileinfo{}
	err = filepath.WalkDir(contentDirAbs, fs.WalkDirFunc(func(path string, d os.DirEntry, err error) error {
		for _, ignore := range *i.idfIgnore {
			if strings.Contains(filepath.Base(path), ignore) {
				return nil
			}
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		info.ModTime()
		existFileInfos = append(existFileInfos, fileinfo{
			Filename: path,
			Lastmod:  info.ModTime(),
		})
		return nil
	}))
	if err != nil {
		err = fmt.Errorf("error at walk dir at %s: %w", contentDirAbs, err)
		return err
	}

	// すでにidfされているもののfullPath
	idfExistFileNames := map[string]struct{}{}
	for _, idfKyou := range allIDFKyous {
		idfFilename := filepath.Join(contentDirAbs, idfKyou.FileName)
		idfExistFileNames[idfFilename] = struct{}{}
	}

	// まだidfされていないやつをリストアップする
	idfTargetList := []string{}
	for _, existFileInfo := range existFileInfos {
		_, existIDFRecord := idfExistFileNames[existFileInfo.Filename]
		if existIDFRecord {
			continue
		}
		idfTargetList = append(idfTargetList, existFileInfo.Filename)
	}

	// 対象をidfする
	idfKyous := []*IDFKyou{}
	now := time.Now()
	repName, err := i.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name: %w", err)
		return err
	}

	for _, idfTargetFileName := range idfTargetList {
		lastMod := now
		for _, existFileInfo := range existFileInfos {
			if existFileInfo.Filename == idfTargetFileName {
				lastMod = existFileInfo.Lastmod
			}
		}

		trimedFileName := strings.TrimLeft(idfTargetFileName, contentDirAbs)
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
			idf.FileName = trimedFileName
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
			idf.FileName = trimedFileName
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
	sql := `
INSERT INTO IDF (
  IS_DELETED,
  ID,
  TARGET_REP_NAME,
  TARGET_FILE,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
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
	log.Printf("sql: %s", sql)
	stmt, err := i.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add idf sql %s: %w", idfKyou.ID, err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		idfKyou.IsDeleted,
		idfKyou.ID,
		idfKyou.RepName,
		idfKyou.FileName,
		idfKyou.RelatedTime.Format(sqlite3impl.TimeLayout),
		idfKyou.CreateTime.Format(sqlite3impl.TimeLayout),
		idfKyou.CreateApp,
		idfKyou.CreateDevice,
		idfKyou.CreateUser,
		idfKyou.UpdateTime.Format(sqlite3impl.TimeLayout),
		idfKyou.UpdateApp,
		idfKyou.UpdateDevice,
		idfKyou.UpdateUser,
	)
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
	if len(str) != len(DUIDLayout)+37*len("_") {
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
