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

type kmemoRepositorySQLite3Impl struct {
	// ˅
	filename string
	db       *sql.DB
	m        *sync.Mutex
	// ˄
}

// ˅
func NewKmemoRepositorySQLite3Impl(ctx context.Context, filename string) (KmemoRepository, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "KMEMO" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  CONTENT NOT NULL,
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
		err = fmt.Errorf("error at create KMEMO table statement %s: %w", filename, err)
		return nil, err
	}

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create KMEMO table to %s: %w", filename, err)
		return nil, err
	}

	return &kmemoRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}

func (k *kmemoRepositorySQLite3Impl) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	var err error

	// jsonからパースする
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at kmemo %s: %w", queryJSON, err)
		return nil, err
	}

	// update_cacheであればキャッシュを更新する
	if queryMap["update_cache"] == fmt.Sprintf("%t", true) {
		err = k.UpdateCache(ctx)
		if err != nil {
			repName, _ := k.GetRepName(ctx)
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
FROM KMEMO
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

	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}

	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at kmemo: %w", err)
		return nil, err
	}

	dataType := "kmemo"
	rows, err := stmt.QueryContext(ctx, repName, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from KMEMO %s: %w", err)
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
				err = fmt.Errorf("error at parse related time %s in KMEMO: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in KMEMO: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in KMEMO: %w", updateTimeStr, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (k *kmemoRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := k.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from KMEMO %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(kyouHistories) == 0 {
		return nil, nil
	}

	return kyouHistories[0], nil
}

func (k *kmemoRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at kmemo: %w", err)
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
FROM KMEMO
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}

	dataType := "kmemo"
	rows, err := stmt.QueryContext(ctx, repName, id, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from KMEMO %s: %w", id, err)
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
				err = fmt.Errorf("error at parse related time %s at %s in KMEMO: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in KMEMO: %w", createTimeStr, id, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in KMEMO: %w", updateTimeStr, id, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (k *kmemoRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return filepath.Abs(k.filename)
}

func (k *kmemoRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (k *kmemoRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := k.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path kmemo rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (k *kmemoRepositorySQLite3Impl) Close(ctx context.Context) error {
	return k.db.Close()
}

func (k *kmemoRepositorySQLite3Impl) FindKmemo(ctx context.Context, queryJSON string) ([]*Kmemo, error) {
	var err error

	// jsonからパースする
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at kmemo %s: %w", queryJSON, err)
		return nil, err
	}

	// update_cacheであればキャッシュを更新する
	if queryMap["update_cache"] == fmt.Sprintf("%t", true) {
		err = k.UpdateCache(ctx)
		if err != nil {
			repName, _ := k.GetRepName(ctx)
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
  CONTENT,
  ? AS REP_NAME
FROM KMEMO
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

	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}

	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at kmemo: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName)
	if err != nil {
		err = fmt.Errorf("error at select from KMEMO %s: %w", err)
		return nil, err
	}

	kmemos := []*Kmemo{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kmemo := &Kmemo{}
			kmemo.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(kmemo.IsDeleted,
				kmemo.ID,
				relatedTimeStr,
				createTimeStr,
				kmemo.CreateApp,
				kmemo.CreateDevice,
				kmemo.CreateUser,
				updateTimeStr,
				kmemo.UpdateApp,
				kmemo.UpdateDevice,
				kmemo.UpdateUser,
				kmemo.Content,
				kmemo.RepName,
			)

			kmemo.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in KMEMO: %w", relatedTimeStr, err)
				return nil, err
			}
			kmemo.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in KMEMO: %w", createTimeStr, err)
				return nil, err
			}
			kmemo.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in KMEMO: %w", updateTimeStr, err)
				return nil, err
			}
			kmemos = append(kmemos, kmemo)
		}
	}
	return kmemos, nil
}

func (k *kmemoRepositorySQLite3Impl) GetKmemo(ctx context.Context, id string) (*Kmemo, error) {
	// 最新のデータを返す
	kmemoHistories, err := k.GetKmemoHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kmemo histories from KMEMO %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(kmemoHistories) == 0 {
		return nil, nil
	}

	return kmemoHistories[0], nil
}

func (k *kmemoRepositorySQLite3Impl) GetKmemoHistories(ctx context.Context, id string) ([]*Kmemo, error) {
	repName, err := k.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at kmemo: %w", err)
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
  CONTENT,
  ? AS REP_NAME
FROM KMEMO
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kmemo histories sql %s: %w", id, err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName, id)
	if err != nil {
		err = fmt.Errorf("error at query ")
		return nil, err
	}

	kmemos := []*Kmemo{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kmemo := &Kmemo{}
			kmemo.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(kmemo.IsDeleted,
				kmemo.ID,
				relatedTimeStr,
				createTimeStr,
				kmemo.CreateApp,
				kmemo.CreateDevice,
				kmemo.CreateUser,
				updateTimeStr,
				kmemo.UpdateApp,
				kmemo.UpdateDevice,
				kmemo.UpdateUser,
				kmemo.Content,
				kmemo.RepName,
			)

			kmemo.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in KMEMO: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kmemo.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in KMEMO: %w", createTimeStr, id, err)
				return nil, err
			}
			kmemo.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in KMEMO: %w", updateTimeStr, id, err)
				return nil, err
			}
			kmemos = append(kmemos, kmemo)
		}
	}
	return kmemos, nil
}

func (k *kmemoRepositorySQLite3Impl) AddKmemoInfo(ctx context.Context, kmemo *Kmemo) error {
	sql := `
INSERT INTO KMEMO
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
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add kmemo sql %s: %w", kmemo.ID, err)
		return err
	}

	_, err = stmt.ExecContext(ctx,
		kmemo.IsDeleted,
		kmemo.ID,
		kmemo.Content,
		kmemo.RelatedTime.Format(sqlite3impl.TimeLayout),
		kmemo.CreateTime.Format(sqlite3impl.TimeLayout),
		kmemo.CreateApp,
		kmemo.CreateDevice,
		kmemo.CreateUser,
		kmemo.UpdateTime.Format(sqlite3impl.TimeLayout),
		kmemo.UpdateApp,
		kmemo.UpdateDevice,
		kmemo.UpdateUser,
	)
	if err != nil {
		err = fmt.Errorf("error at insert in to KMEMO %s: %w", kmemo.ID, err)
		return err
	}
	return nil
}

// ˄
