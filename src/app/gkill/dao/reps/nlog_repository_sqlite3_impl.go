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

type nlogRepositorySQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewNlogRepositorySQLite3Impl(ctx context.Context, filename string) (NlogRepository, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "NLOG" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  SHOP NOT NULL,
  TITLE NOT NULL,
  AMOUNT NOT NULL,
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
		err = fmt.Errorf("error at create NLOG table statement %s: %w", filename, err)
		return nil, err
	}

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NLOG table to %s: %w", filename, err)
		return nil, err
	}

	return &nlogRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (n *nlogRepositorySQLite3Impl) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	var err error

	// jsonからパースする
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at NLOG %s: %w", queryJSON, err)
		return nil, err
	}

	// update_cacheであればキャッシュを更新する
	if queryMap["update_cache"] == fmt.Sprintf("%t", true) {
		err = n.UpdateCache(ctx)
		if err != nil {
			repName, _ := n.GetRepName(ctx)
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
FROM NLOG 
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
				sql += sqlite3impl.EscapeSQLite("SHOP LIKE '%" + word + "%'")
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
				sql += sqlite3impl.EscapeSQLite("SHOP LIKE '%" + word + "%'")
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
			sql += sqlite3impl.EscapeSQLite("SHOP LIKE '%" + notWord + "%'")
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

	stmt, err := n.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}

	repName, err := n.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at NLOG: %w", err)
		return nil, err
	}

	dataType := "nlog"
	rows, err := stmt.QueryContext(ctx, repName, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from NLOG %s: %w", err)
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
				err = fmt.Errorf("error at parse related time %s in NLOG: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in NLOG: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in NLOG: %w", updateTimeStr, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (n *nlogRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := n.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from NLOG%s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(kyouHistories) == 0 {
		return nil, nil
	}

	return kyouHistories[0], nil
}

func (n *nlogRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	repName, err := n.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at NLOG: %w", err)
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
FROM NLOG
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`
	stmt, err := n.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}

	dataType := "nlog"
	rows, err := stmt.QueryContext(ctx, repName, id, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from NLOG %s: %w", id, err)
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
				err = fmt.Errorf("error at parse related time %s at %s in NLOG: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in NLOG: %w", createTimeStr, id, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in NLOG: %w", updateTimeStr, id, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (n *nlogRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return filepath.Abs(n.filename)
}

func (n *nlogRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (n *nlogRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := n.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path nlog rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (n *nlogRepositorySQLite3Impl) Close(ctx context.Context) error {
	return n.db.Close()
}

func (n *nlogRepositorySQLite3Impl) FindNlog(ctx context.Context, queryJSON string) ([]*Nlog, error) {
	var err error

	// jsonからパースする
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at nlog %s: %w", queryJSON, err)
		return nil, err
	}

	// update_cacheであればキャッシュを更新する
	if queryMap["update_cache"] == fmt.Sprintf("%t", true) {
		err = n.UpdateCache(ctx)
		if err != nil {
			repName, _ := n.GetRepName(ctx)
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
  SHOP,
  TITLE,
  AMOUNT,
  ? AS REP_NAME
FROM NLOG
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

	stmt, err := n.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql: %w", err)
		return nil, err
	}

	repName, err := n.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at NLOG: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName)
	if err != nil {
		err = fmt.Errorf("error at select from NLOG %s: %w", err)
		return nil, err
	}

	nlogs := []*Nlog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			nlog := &Nlog{}
			nlog.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(nlog.IsDeleted,
				nlog.ID,
				relatedTimeStr,
				createTimeStr,
				nlog.CreateApp,
				nlog.CreateDevice,
				nlog.CreateUser,
				updateTimeStr,
				nlog.UpdateApp,
				nlog.UpdateDevice,
				nlog.UpdateUser,
				nlog.Shop,
				nlog.Title,
				nlog.Amount,
				nlog.RepName,
			)

			nlog.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in NLOG: %w", relatedTimeStr, err)
				return nil, err
			}
			nlog.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in NLOG: %w", createTimeStr, err)
				return nil, err
			}
			nlog.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in NLOG: %w", updateTimeStr, err)
				return nil, err
			}
			nlogs = append(nlogs, nlog)
		}
	}
	return nlogs, nil
}

func (n *nlogRepositorySQLite3Impl) GetNlog(ctx context.Context, id string) (*Nlog, error) {
	// 最新のデータを返す
	nlogHistories, err := n.GetNlogHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get nlog histories from NLOG%s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(nlogHistories) == 0 {
		return nil, nil
	}

	return nlogHistories[0], nil
}

func (n *nlogRepositorySQLite3Impl) GetNlogHistories(ctx context.Context, id string) ([]*Nlog, error) {
	repName, err := n.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at nlog: %w", err)
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
  SHOP,
  TITLE,
  AMOUNT
  ? AS REP_NAME
FROM NLOG 
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`
	stmt, err := n.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get nlog histories sql %s: %w", id, err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, repName, id)
	if err != nil {
		err = fmt.Errorf("error at query ")
		return nil, err
	}

	nlogs := []*Nlog{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			nlog := &Nlog{}
			nlog.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(nlog.IsDeleted,
				nlog.ID,
				relatedTimeStr,
				createTimeStr,
				nlog.CreateApp,
				nlog.CreateDevice,
				nlog.CreateUser,
				updateTimeStr,
				nlog.UpdateApp,
				nlog.UpdateDevice,
				nlog.UpdateUser,
				nlog.Shop,
				nlog.Title,
				nlog.Amount,
				nlog.RepName,
			)

			nlog.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s at %s in NLOG: %w", relatedTimeStr, id, err)
				return nil, err
			}
			nlog.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in NLOG: %w", createTimeStr, id, err)
				return nil, err
			}
			nlog.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in NLOG: %w", updateTimeStr, id, err)
				return nil, err
			}
			nlogs = append(nlogs, nlog)
		}
	}
	return nlogs, nil
}

func (n *nlogRepositorySQLite3Impl) AddNlogInfo(ctx context.Context, nlog *Nlog) error {
	sql := `
INSERT INTO NLOG
  IS_DELETED,
  ID,
  SHOP,
  TITLE,
  AMOUNT,
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
  ?
)`
	stmt, err := n.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add nlog sql %s: %w", nlog.ID, err)
		return err
	}

	_, err = stmt.ExecContext(ctx,
		nlog.IsDeleted,
		nlog.ID,
		nlog.Shop,
		nlog.Title,
		nlog.Amount,
		nlog.RelatedTime.Format(sqlite3impl.TimeLayout),
		nlog.CreateTime.Format(sqlite3impl.TimeLayout),
		nlog.CreateApp,
		nlog.CreateDevice,
		nlog.CreateUser,
		nlog.UpdateTime.Format(sqlite3impl.TimeLayout),
		nlog.UpdateApp,
		nlog.UpdateDevice,
		nlog.UpdateUser,
	)
	if err != nil {
		err = fmt.Errorf("error at insert in to NLOG %s: %w", nlog.ID, err)
		return err
	}
	return nil
}
