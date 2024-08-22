// ˅
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

// ˄

type lantanaRepositorySQLite3Impl struct {
	// ˅
	filename string
	db       *sql.DB
	m        *sync.Mutex
	// ˄
}

// ˅
func NewLantanaRepositorySQLite3Impl(ctx context.Context, filename string) (LantanaRepository, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "LANTANA" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  MOOD NOT NULL,
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
		err = fmt.Errorf("error at create LANTANA table statement %s: %w", filename, err)
		return nil, err
	}

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create LANTANA table to %s: %w", filename, err)
		return nil, err
	}

	return &lantanaRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (l *lantanaRepositorySQLite3Impl) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	var err error

	// jsonからパースする
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at lantana %s: %w", queryJSON, err)
		return nil, err
	}

	// update_cacheであればキャッシュを更新する
	if queryMap["update_cache"] == fmt.Sprintf("%t", true) {
		err = l.UpdateCache(ctx)
		if err != nil {
			repName, _ := l.GetRepName(ctx)
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
FROM  LANTANA
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
				sql += sqlite3impl.EscapeSQLite("CONTENT LIKE '%" + word + "%'")
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
				sql += sqlite3impl.EscapeSQLite("CONTENT LIKE '%" + word + "%'")
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
			sql += sqlite3impl.EscapeSQLite("CONTENT NOT LIKE '%" + notWord + "%'")
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

	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}

	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
		return nil, err
	}

	dataType := "lantana"
	rows, err := stmt.QueryContext(ctx, repName, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from LANTANA%s: %w", err)
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
				err = fmt.Errorf("error at parse related time %s in LANTANA: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in LANTANA: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in LANTANA: %w", updateTimeStr, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (l *lantanaRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := l.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from LANTANA %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(kyouHistories) == 0 {
		return nil, nil
	}

	return kyouHistories[0], nil
}

func (l *lantanaRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
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
FROM  LANTANA
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}

	dataType := "lantana"
	rows, err := stmt.QueryContext(ctx, repName, id, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from LANTANA %s: %w", id, err)
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
				err = fmt.Errorf("error at parse related time %s at %s in LANTANA: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in LANTANA: %w", createTimeStr, id, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in LANTANA: %w", updateTimeStr, id, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (l *lantanaRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return filepath.Abs(l.filename)
}

func (l *lantanaRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (l *lantanaRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := l.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path lantana rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (l *lantanaRepositorySQLite3Impl) Close(ctx context.Context) error {
	return l.db.Close()
}

func (l *lantanaRepositorySQLite3Impl) FindLantana(ctx context.Context, queryJSON string) ([]*Lantana, error) {
	var err error

	// jsonからパースする
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at lantana %s: %w", queryJSON, err)
		return nil, err
	}

	// update_cacheであればキャッシュを更新する
	if queryMap["update_cache"] == fmt.Sprintf("%t", true) {
		err = l.UpdateCache(ctx)
		if err != nil {
			repName, _ := l.GetRepName(ctx)
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
  MOOD,
  ? AS REP_NAME
FROM LANTANA
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
				sql += sqlite3impl.EscapeSQLite("CONTENT LIKE '%" + word + "%'")
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
				sql += sqlite3impl.EscapeSQLite("CONTENT LIKE '%" + word + "%'")
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
			sql += sqlite3impl.EscapeSQLite(fmt.Sprintf("CONTENT NOT LIKE '%s'", notWord))
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

	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}

	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName)
	if err != nil {
		err = fmt.Errorf("error at select from LANTANA %s: %w", err)
		return nil, err
	}

	lantanas := []*Lantana{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			lantana := &Lantana{}
			lantana.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(lantana.IsDeleted,
				lantana.ID,
				relatedTimeStr,
				createTimeStr,
				lantana.CreateApp,
				lantana.CreateDevice,
				lantana.CreateUser,
				updateTimeStr,
				lantana.UpdateApp,
				lantana.UpdateDevice,
				lantana.UpdateUser,
				lantana.Mood,
				lantana.RepName,
			)

			lantana.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in LANTANA: %w", relatedTimeStr, err)
				return nil, err
			}
			lantana.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in LANTANA: %w", createTimeStr, err)
				return nil, err
			}
			lantana.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in LANTANA: %w", updateTimeStr, err)
				return nil, err
			}
			lantanas = append(lantanas, lantana)
		}
	}
	return lantanas, nil
}

func (l *lantanaRepositorySQLite3Impl) GetLantana(ctx context.Context, id string) (*Lantana, error) {
	// 最新のデータを返す
	lantanaHistories, err := l.GetLantanaHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get lantana histories from LANTANA %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(lantanaHistories) == 0 {
		return nil, nil
	}

	return lantanaHistories[0], nil
}

func (l *lantanaRepositorySQLite3Impl) GetLantanaHistories(ctx context.Context, id string) ([]*Lantana, error) {
	repName, err := l.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at lantana: %w", err)
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
  MOOD,
  ? AS REP_NAME
FROM LANTANA
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get lantana histories sql %s: %w", id, err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName, id)
	if err != nil {
		err = fmt.Errorf("error at query ")
		return nil, err
	}

	lantanas := []*Lantana{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			lantana := &Lantana{}
			lantana.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(lantana.IsDeleted,
				lantana.ID,
				relatedTimeStr,
				createTimeStr,
				lantana.CreateApp,
				lantana.CreateDevice,
				lantana.CreateUser,
				updateTimeStr,
				lantana.UpdateApp,
				lantana.UpdateDevice,
				lantana.UpdateUser,
				lantana.Mood,
				lantana.RepName,
			)

			lantana.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in LANTANA: %w", relatedTimeStr, id, err)
				return nil, err
			}
			lantana.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in LANTANA: %w", createTimeStr, id, err)
				return nil, err
			}
			lantana.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in LANTANA: %w", updateTimeStr, id, err)
				return nil, err
			}
			lantanas = append(lantanas, lantana)
		}
	}
	return lantanas, nil
}

func (l *lantanaRepositorySQLite3Impl) AddLantanaInfo(ctx context.Context, lantana *Lantana) error {
	sql := `
INSERT INTO LANTANA
  IS_DELETED,
  ID,
  CONTENT,
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
  ?
)`
	stmt, err := l.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add lantana sql %s: %w", lantana.ID, err)
		return err
	}

	_, err = stmt.ExecContext(ctx,
		lantana.IsDeleted,
		lantana.ID,
		lantana.Mood,
		lantana.RelatedTime.Format(sqlite3impl.TimeLayout),
		lantana.CreateTime.Format(sqlite3impl.TimeLayout),
		lantana.CreateApp,
		lantana.CreateDevice,
		lantana.CreateUser,
		lantana.UpdateTime.Format(sqlite3impl.TimeLayout),
		lantana.UpdateApp,
		lantana.UpdateDevice,
		lantana.UpdateUser,
	)
	if err != nil {
		err = fmt.Errorf("error at insert in to LANTANA %s: %w", lantana.ID, err)
		return err
	}
	return nil
}

// ˄
