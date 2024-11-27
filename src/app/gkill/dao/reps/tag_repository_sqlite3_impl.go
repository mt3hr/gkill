package reps

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type tagRepositorySQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewTagRepositorySQLite3Impl(ctx context.Context, filename string) (TagRepository, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "TAG" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
  TAG NOT NULL,
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
		err = fmt.Errorf("error at create TAG table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG table to %s: %w", filename, err)
		return nil, err
	}

	return &tagRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (t *tagRepositorySQLite3Impl) FindTags(ctx context.Context, query *find.FindQuery) ([]*Tag, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = t.UpdateCache(ctx)
		if err != nil {
			repName, _ := t.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TAG,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM TAG
WHERE
`

	whereCounter := 0
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	words := []string{}
	if query.Words != nil {
		words = *query.Words
	}
	notWords := []string{}
	if query.NotWords != nil {
		notWords = *query.NotWords
	}

	// ワードand検索である場合のSQL追記
	if query.UseWords != nil && *query.UseWords {
		if whereCounter != 0 {
			sql += " AND "
		}

		if query.WordsAnd != nil && *query.WordsAnd {
			for i, word := range words {
				if i == 0 {
					sql += " ( "
				}
				if whereCounter != 0 {
					sql += " AND "
				}
				sql += sqlite3impl.EscapeSQLite("TAG LIKE '" + word + "'")
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
				sql += sqlite3impl.EscapeSQLite("TAG LIKE '" + word + "'")
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
			sql += sqlite3impl.EscapeSQLite("TAG NOT LIKE '" + notWord + "'")
			if i == len(words)-1 {
				sql += " ) "
			}
			whereCounter++
		}

		// id検索である場合のSQL追記
		if query.UseIDs != nil && *query.UseIDs {
			ids := []string{}
			if query.IDs != nil {
				ids = *query.IDs
			}
			if whereCounter != 0 {
				sql += " AND "
			}
			sql += "ID IN ("
			for i, id := range ids {
				sql += fmt.Sprintf("'%s'", id)
				if i != len(ids)-1 {
					sql += ", "
				}
			}
			sql += ")"
		}

	}
	// UPDATE_TIMEが一番上のものだけを抽出
	sql += `
GROUP BY ID
HAVING MAX(datetime(UPDATE_TIME, 'localtime'))
`
	sql += `;`

	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get tag histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at tag: %w", err)
		return nil, err
	}

	dataType := "tag"
	rows, err := stmt.QueryContext(ctx, repName, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from TAG %s: %w", err)
		return nil, err
	}
	defer rows.Close()

	tags := []*Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := &Tag{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeStr,
				&createTimeStr,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeStr,
				&tag.UpdateApp,
				&tag.UpdateDevice,
				&tag.UpdateUser,
				&tag.RepName,
				&dataType,
			)

			tag.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TAG: %w", relatedTimeStr, err)
				return nil, err
			}
			tag.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TAG: %w", createTimeStr, err)
				return nil, err
			}
			tag.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TAG: %w", updateTimeStr, err)
				return nil, err
			}
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) Close(ctx context.Context) error {
	return t.db.Close()
}

func (t *tagRepositorySQLite3Impl) GetTag(ctx context.Context, id string) (*Tag, error) {
	// 最新のデータを返す
	tagHistories, err := t.GetTagHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get tag histories from TAG %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(tagHistories) == 0 {
		return nil, nil
	}

	return tagHistories[0], nil
}

func (t *tagRepositorySQLite3Impl) GetTagsByTagName(ctx context.Context, tagname string) ([]*Tag, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TAG,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM TAG
WHERE TAG LIKE ?
`

	// UPDATE_TIMEが一番上のものだけを抽出
	sql += `
GROUP BY ID
HAVING MAX(datetime(UPDATE_TIME, 'localtime'))
`
	sql += `;`

	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get tag histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at tag: %w", err)
		return nil, err
	}

	dataType := "tag"
	rows, err := stmt.QueryContext(ctx, repName, dataType, tagname)
	if err != nil {
		err = fmt.Errorf("error at select from TAG %s: %w", err)
		return nil, err
	}
	defer rows.Close()

	tags := []*Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := &Tag{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeStr,
				&createTimeStr,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeStr,
				&tag.UpdateApp,
				&tag.UpdateDevice,
				&tag.UpdateUser,
				&tag.RepName,
				&dataType,
			)

			tag.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TAG: %w", relatedTimeStr, err)
				return nil, err
			}
			tag.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TAG: %w", createTimeStr, err)
				return nil, err
			}
			tag.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TAG: %w", updateTimeStr, err)
				return nil, err
			}
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) GetTagsByTargetID(ctx context.Context, target_id string) ([]*Tag, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TAG,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM TAG
WHERE TARGET_ID LIKE ?
`

	// UPDATE_TIMEが一番上のものだけを抽出
	sql += `
GROUP BY ID
HAVING MAX(datetime(UPDATE_TIME, 'localtime'))
`
	sql += `;`

	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get tag histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at tag: %w", err)
		return nil, err
	}

	dataType := "tag"
	rows, err := stmt.QueryContext(ctx, repName, dataType, target_id)
	if err != nil {
		err = fmt.Errorf("error at select from TAG %s: %w", err)
		return nil, err
	}
	defer rows.Close()

	tags := []*Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := &Tag{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeStr,
				&createTimeStr,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeStr,
				&tag.UpdateApp,
				&tag.UpdateDevice,
				&tag.UpdateUser,
				&tag.RepName,
				&dataType,
			)

			tag.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TAG: %w", relatedTimeStr, err)
				return nil, err
			}
			tag.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TAG: %w", createTimeStr, err)
				return nil, err
			}
			tag.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TAG: %w", updateTimeStr, err)
				return nil, err
			}
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (t *tagRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return filepath.Abs(t.filename)
}

func (t *tagRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := t.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path tag rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (t *tagRepositorySQLite3Impl) GetTagHistories(ctx context.Context, id string) ([]*Tag, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TAG,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM TAG
WHERE ID LIKE ?
`

	sql += `;`

	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get tag histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at tag: %w", err)
		return nil, err
	}

	dataType := "tag"
	rows, err := stmt.QueryContext(ctx, repName, dataType, id)
	if err != nil {
		err = fmt.Errorf("error at select from TAG %s: %w", err)
		return nil, err
	}
	defer rows.Close()

	tags := []*Tag{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tag := &Tag{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&tag.IsDeleted,
				&tag.ID,
				&tag.TargetID,
				&tag.Tag,
				&relatedTimeStr,
				&createTimeStr,
				&tag.CreateApp,
				&tag.CreateDevice,
				&tag.CreateUser,
				&updateTimeStr,
				&tag.UpdateApp,
				&tag.UpdateDevice,
				&tag.UpdateUser,
				&tag.RepName,
				&dataType,
			)

			tag.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TAG: %w", relatedTimeStr, err)
				return nil, err
			}
			tag.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TAG: %w", createTimeStr, err)
				return nil, err
			}
			tag.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TAG: %w", updateTimeStr, err)
				return nil, err
			}
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (t *tagRepositorySQLite3Impl) AddTagInfo(ctx context.Context, tag *Tag) error {
	sql := `
INSERT INTO TAG (
  IS_DELETED,
  ID,
  TAG,
  TARGET_ID,
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
)`
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add tag sql %s: %w", tag.ID, err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		tag.IsDeleted,
		tag.ID,
		tag.Tag,
		tag.TargetID,
		tag.RelatedTime.Format(sqlite3impl.TimeLayout),
		tag.CreateTime.Format(sqlite3impl.TimeLayout),
		tag.CreateApp,
		tag.CreateDevice,
		tag.CreateUser,
		tag.UpdateTime.Format(sqlite3impl.TimeLayout),
		tag.UpdateApp,
		tag.UpdateDevice,
		tag.UpdateUser,
	)
	if err != nil {
		err = fmt.Errorf("error at insert in to TAG %s: %w", tag.ID, err)
		return err
	}
	return nil
}

func (t *tagRepositorySQLite3Impl) GetAllTagNames(ctx context.Context) ([]string, error) {
	var err error

	sql := `
SELECT 
  DISTINCT TAG
FROM TAG
WHERE IS_DELETED = FALSE

`
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all tag names sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at select all tag names from TAG %s: %w", err)
		return nil, err
	}
	defer rows.Close()

	tagNames := []string{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tagName := ""
			err = rows.Scan(
				&tagName,
			)
			if err != nil {
				err = fmt.Errorf("error at read rows at get all tag names: %w", err)
				return nil, err
			}

			tagNames = append(tagNames, tagName)
		}
	}
	return tagNames, nil
}
