package reps

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type urlogRepositorySQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewURLogRepositorySQLite3Impl(ctx context.Context, filename string) (URLogRepository, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "URLOG" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  URL NOT NULL,
  TITLE NOT NULL,
  DESCRIPTION NOT NULL,
  FAVICON_IMAGE NOT NULL,
  THUMBNAIL_IMAGE NOT NULL,
  RELATED_TIME NOT NULL,
  CREATE_TIME NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_TIME NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL
);`
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table statement %s: %w", filename, err)
		return nil, err
	}

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create URLOG table to %s: %w", filename, err)
		return nil, err
	}

	return &urlogRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}

func (u *urlogRepositorySQLite3Impl) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	var err error

	// jsonからパースする
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at URLOG%s: %w", queryJSON, err)
		return nil, err
	}

	// update_cacheであればキャッシュを更新する
	if queryMap["update_cache"] == fmt.Sprintf("%t", true) {
		err = u.UpdateCache(ctx)
		if err != nil {
			repName, _ := u.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}

	sql := `
SELECT 
  IS_DELETED,
  ID,
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
FROM URLOG
WHERE
`

	whereCounter := 0
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(queryJSON, &whereCounter)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	// ワードand検索である場合のSQL追記
	if queryMap["use_word"] == fmt.Sprintf("%t", true) {
		// ワードを解析
		words := []string{}
		err = json.Unmarshal([]byte(queryMap["words"]), &words)
		if err != nil {
			err = fmt.Errorf("error at parse query word %s: %w", queryMap["words"], err)
			return nil, err
		}
		notWords := []string{}
		err = json.Unmarshal([]byte(queryMap["not_words"]), &words)
		if err != nil {
			err = fmt.Errorf("error at parse query not word %s: %w", queryMap["not_words"], err)
			return nil, err
		}

		if whereCounter != 0 {
			sql += " AND "
		}

		if queryMap["words_and"] == fmt.Sprintf("%t", true) {
			for i, word := range words {
				if i == 0 {
					sql += " ( "
				}
				if whereCounter != 0 {
					sql += " AND "
				}
				sql += sqlite3impl.EscapeSQLite("TITLE LIKE '%" + word + "%'")
				sql += sqlite3impl.EscapeSQLite(" OR ")
				sql += sqlite3impl.EscapeSQLite("DESCRIPTION LIKE '%" + word + "%'")
				sql += sqlite3impl.EscapeSQLite(" OR ")
				sql += sqlite3impl.EscapeSQLite("URL LIKE '%" + word + "%'")
				if i == len(words)-1 {
					sql += " ) "
				}
				whereCounter++
			}
		} else {
			// ワードor検索である場合のSQL追記
			for i, word := range words {
				if i == 0 {
					sql += " ( "
				}
				if whereCounter != 0 {
					sql += " AND "
				}
				sql += sqlite3impl.EscapeSQLite("TITLE LIKE '%" + word + "%'")
				sql += sqlite3impl.EscapeSQLite(" OR ")
				sql += sqlite3impl.EscapeSQLite("DESCRIPTION LIKE '%" + word + "%'")
				sql += sqlite3impl.EscapeSQLite(" OR ")
				sql += sqlite3impl.EscapeSQLite("URL LIKE '%" + word + "%'")
				if i == len(words)-1 {
					sql += " ) "
				}
				whereCounter++
			}
		}

		if whereCounter != 0 {
			sql += " AND "
		}

		// notワードを除外するSQLを追記
		for i, notWord := range notWords {
			if i == 0 {
				sql += " ( "
			}
			if whereCounter != 0 {
				sql += " AND "
			}
			sql += sqlite3impl.EscapeSQLite("TITLE NOT LIKE '%" + notWord + "%'")
			sql += sqlite3impl.EscapeSQLite(" AND ")
			sql += sqlite3impl.EscapeSQLite("DESCRIPTION NOT LIKE '%" + notWord + "%'")
			sql += sqlite3impl.EscapeSQLite(" AND ")
			sql += sqlite3impl.EscapeSQLite("URL LIKE '%" + notWord + "%'")
			if i == len(words)-1 {
				sql += " ) "
			}
			whereCounter++
		}
	}
	// UPDATE_TIMEが一番上のものだけを抽出
	sql += `
GROUP BY ID
HAVING MAX(datetime(UPDATE_TIME, 'localtime'))
`
	sql += `;`

	stmt, err := u.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}

	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at URLOG: %w", err)
		return nil, err
	}

	dataType := "urlog"
	rows, err := stmt.QueryContext(ctx, repName, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from URLOG%s: %w", err)
		return nil, err
	}

	kyous := []*Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := &Kyou{}
			kyou.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(kyou.IsDeleted,
				kyou.ID,
				relatedTimeStr,
				createTimeStr,
				kyou.CreateApp,
				kyou.CreateDevice,
				kyou.CreateUser,
				updateTimeStr,
				kyou.UpdateApp,
				kyou.UpdateDevice,
				kyou.UpdateUser,
				kyou.RepName,
				kyou.DataType,
			)

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in URLOG: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in URLOG: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in URLOG: %w", updateTimeStr, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (u *urlogRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := u.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from URLOG %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(kyouHistories) == 0 {
		return nil, nil
	}

	return kyouHistories[0], nil
}

func (u *urlogRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at URLOG: %w", err)
		return nil, err
	}

	sql := `
SELECT 
  IS_DELETED,
  ID,
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
FROM URLOG 
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`
	stmt, err := u.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}

	dataType := "urlog"
	rows, err := stmt.QueryContext(ctx, repName, id, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from URLOG %s: %w", id, err)
		return nil, err
	}

	kyous := []*Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := &Kyou{}
			kyou.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(kyou.IsDeleted,
				kyou.ID,
				relatedTimeStr,
				createTimeStr,
				kyou.CreateApp,
				kyou.CreateDevice,
				kyou.CreateUser,
				updateTimeStr,
				kyou.UpdateApp,
				kyou.UpdateDevice,
				kyou.UpdateUser,
				kyou.RepName,
			)

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in URLOG: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in URLOG: %w", createTimeStr, id, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in URLOG: %w", updateTimeStr, id, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (u *urlogRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return filepath.Abs(u.filename)
}

func (u *urlogRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (u *urlogRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := u.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path urlog rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil

}

func (u *urlogRepositorySQLite3Impl) Close(ctx context.Context) error {
	return u.db.Close()
}

func (u *urlogRepositorySQLite3Impl) FindURLog(ctx context.Context, queryJSON string) ([]*URLog, error) {
	var err error

	// jsonからパースする
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at URLOG  %s: %w", queryJSON, err)
		return nil, err
	}

	// update_cacheであればキャッシュを更新する
	if queryMap["update_cache"] == fmt.Sprintf("%t", true) {
		err = u.UpdateCache(ctx)
		if err != nil {
			repName, _ := u.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}

	sql := `
SELECT 
  IS_DELETED,
  ID,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
  THUMBNAIL_IMAGE,
  ? AS REP_NAME
FROM URLOG
WHERE
`

	whereCounter := 0
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(queryJSON, &whereCounter)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	// ワードand検索である場合のSQL追記
	if queryMap["use_word"] == fmt.Sprintf("%t", true) {
		// ワードを解析
		words := []string{}
		err = json.Unmarshal([]byte(queryMap["words"]), &words)
		if err != nil {
			err = fmt.Errorf("error at parse query word %s: %w", queryMap["words"], err)
			return nil, err
		}
		notWords := []string{}
		err = json.Unmarshal([]byte(queryMap["not_words"]), &words)
		if err != nil {
			err = fmt.Errorf("error at parse query not word %s: %w", queryMap["not_words"], err)
			return nil, err
		}

		if whereCounter != 0 {
			sql += " AND "
		}

		if queryMap["words_and"] == fmt.Sprintf("%t", true) {
			for i, word := range words {
				if i == 0 {
					sql += " ( "
				}
				if whereCounter != 0 {
					sql += " AND "
				}
				sql += sqlite3impl.EscapeSQLite("TITLE LIKE '%" + word + "%'")
				sql += sqlite3impl.EscapeSQLite(" OR ")
				sql += sqlite3impl.EscapeSQLite("DESCRIPTION LIKE '%" + word + "%'")
				sql += sqlite3impl.EscapeSQLite(" OR ")
				sql += sqlite3impl.EscapeSQLite("URL LIKE '%" + word + "%'")
				if i == len(words)-1 {
					sql += " ) "
				}
				whereCounter++
			}
		} else {
			// ワードor検索である場合のSQL追記
			for i, word := range words {
				if i == 0 {
					sql += " ( "
				}
				if whereCounter != 0 {
					sql += " AND "
				}
				sql += sqlite3impl.EscapeSQLite("TITLE LIKE '%" + word + "%'")
				sql += sqlite3impl.EscapeSQLite(" OR ")
				sql += sqlite3impl.EscapeSQLite("DESCRIPTION LIKE '%" + word + "%'")
				sql += sqlite3impl.EscapeSQLite(" OR ")
				sql += sqlite3impl.EscapeSQLite("URL LIKE '%" + word + "%'")
				if i == len(words)-1 {
					sql += " ) "
				}
				whereCounter++
			}
		}

		if whereCounter != 0 {
			sql += " AND "
		}

		// notワードを除外するSQLを追記
		for i, notWord := range notWords {
			if i == 0 {
				sql += " ( "
			}
			if whereCounter != 0 {
				sql += " AND "
			}
			sql += sqlite3impl.EscapeSQLite("TITLE NOT LIKE '%" + notWord + "%'")
			sql += sqlite3impl.EscapeSQLite(" AND ")
			sql += sqlite3impl.EscapeSQLite("DESCRIPTION NOT LIKE '%" + notWord + "%'")
			if i == len(words)-1 {
				sql += " ) "
			}
			whereCounter++
		}
	}

	// UPDATE_TIMEが一番上のものだけを抽出
	sql += `
GROUP BY ID
HAVING MAX(datetime(UPDATE_TIME, 'localtime'))
`
	sql += `;`

	stmt, err := u.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}

	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at URLOG: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName)
	if err != nil {
		err = fmt.Errorf("error at select from URLOG %s: %w", err)
		return nil, err
	}

	urlogs := []*URLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			urlog := &URLog{}
			urlog.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(urlog.IsDeleted,
				urlog.ID,
				relatedTimeStr,
				createTimeStr,
				urlog.CreateApp,
				urlog.CreateDevice,
				urlog.CreateUser,
				updateTimeStr,
				urlog.UpdateApp,
				urlog.UpdateDevice,
				urlog.UpdateUser,
				urlog.URL,
				urlog.Title,
				urlog.Description,
				urlog.FaviconImage,
				urlog.ThumbnailImage,
				urlog.RepName,
			)

			urlog.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in URLOG: %w", relatedTimeStr, err)
				return nil, err
			}
			urlog.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in URLOG: %w", createTimeStr, err)
				return nil, err
			}
			urlog.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in URLOG: %w", updateTimeStr, err)
				return nil, err
			}
			urlogs = append(urlogs, urlog)
		}
	}
	return urlogs, nil
}

func (u *urlogRepositorySQLite3Impl) GetURLog(ctx context.Context, id string) (*URLog, error) {
	// 最新のデータを返す
	urlogHistories, err := u.GetURLogHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get urlog histories from URLog %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(urlogHistories) == 0 {
		return nil, nil
	}

	return urlogHistories[0], nil
}

func (u *urlogRepositorySQLite3Impl) GetURLogHistories(ctx context.Context, id string) ([]*URLog, error) {
	repName, err := u.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at URLOG: %w", err)
		return nil, err
	}

	sql := `
SELECT 
  IS_DELETED,
  ID,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
  THUMBNAIL_IMAGE,
  ? AS REP_NAME
FROM URLOG
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`
	stmt, err := u.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kmemo histories sql %s: %w", id, err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName, id)
	if err != nil {
		err = fmt.Errorf("error at query ")
		return nil, err
	}

	urlogs := []*URLog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			urlog := &URLog{}
			urlog.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(urlog.IsDeleted,
				urlog.ID,
				relatedTimeStr,
				createTimeStr,
				urlog.CreateApp,
				urlog.CreateDevice,
				urlog.CreateUser,
				updateTimeStr,
				urlog.UpdateApp,
				urlog.UpdateDevice,
				urlog.UpdateUser,
				urlog.URL,
				urlog.Title,
				urlog.Description,
				urlog.FaviconImage,
				urlog.ThumbnailImage,
				urlog.RepName,
			)

			urlog.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in KMEMO: %w", relatedTimeStr, id, err)
				return nil, err
			}
			urlog.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in KMEMO: %w", createTimeStr, id, err)
				return nil, err
			}
			urlog.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in KMEMO: %w", updateTimeStr, id, err)
				return nil, err
			}
			urlogs = append(urlogs, urlog)
		}
	}
	return urlogs, nil
}

func (u *urlogRepositorySQLite3Impl) AddURLogInfo(ctx context.Context, urlog *URLog) error {
	sql := `
INSERT INTO URLOG
  IS_DELETED,
  ID,
  TITLE,
  DESCRIPTION,
  FAVICON_IMAGE,
  THUMBNAIL_IMAGE,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER
VASLUES(
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
	stmt, err := u.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add urlog sql %s: %w", urlog.ID, err)
		return err
	}

	_, err = stmt.ExecContext(ctx,
		urlog.IsDeleted,
		urlog.ID,
		urlog.Title,
		urlog.Description,
		urlog.FaviconImage,
		urlog.ThumbnailImage,
		urlog.RelatedTime.Format(sqlite3impl.TimeLayout),
		urlog.CreateTime.Format(sqlite3impl.TimeLayout),
		urlog.CreateApp,
		urlog.CreateDevice,
		urlog.CreateUser,
		urlog.UpdateTime.Format(sqlite3impl.TimeLayout),
		urlog.UpdateApp,
		urlog.UpdateDevice,
		urlog.UpdateUser,
	)
	if err != nil {
		err = fmt.Errorf("error at insert in to URLog %s: %w", urlog.ID, err)
		return err
	}
	return nil
}
