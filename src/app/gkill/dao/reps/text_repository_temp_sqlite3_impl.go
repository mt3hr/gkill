package reps

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type textTempRepositorySQLite3Impl textRepositorySQLite3Impl

func NewTextTempRepositorySQLite3Impl(ctx context.Context, db *sql.DB) (TextTempRepository, error) {
	filename := "text_temp"
	sql := `
CREATE TABLE IF NOT EXISTS "TEXT" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
  TEXT NOT NULL,
  RELATED_TIME NOT NULL,
  CREATE_TIME NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_TIME NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  TX_ID NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TEXT table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_TEXT ON TEXT (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create TEXT index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TEXT index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create TEXT table to %s: %w", filename, err)
		return nil, err
	}

	return &textTempRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}

func (t *textTempRepositorySQLite3Impl) FindTexts(ctx context.Context, query *find.FindQuery) ([]*Text, error) {
	impl := textRepositorySQLite3Impl(*t)
	return impl.FindTexts(ctx, query)
}

func (t *textTempRepositorySQLite3Impl) Close(ctx context.Context) error {
	impl := textRepositorySQLite3Impl(*t)
	return impl.Close(ctx)
}

func (t *textTempRepositorySQLite3Impl) GetText(ctx context.Context, id string, updateTime *time.Time) (*Text, error) {
	impl := textRepositorySQLite3Impl(*t)
	return impl.GetText(ctx, id, updateTime)
}

func (t *textTempRepositorySQLite3Impl) GetTextsByTargetID(ctx context.Context, target_id string) ([]*Text, error) {
	impl := textRepositorySQLite3Impl(*t)
	return impl.GetTextsByTargetID(ctx, target_id)
}

func (t *textTempRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	impl := textRepositorySQLite3Impl(*t)
	return impl.UpdateCache(ctx)
}

func (t *textTempRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("text_temp repository does not support GetPath")
}

func (t *textTempRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "text_temp", nil
}

func (t *textTempRepositorySQLite3Impl) GetTextHistories(ctx context.Context, id string) ([]*Text, error) {
	impl := textRepositorySQLite3Impl(*t)
	return impl.GetTextHistories(ctx, id)
}

func (t *textTempRepositorySQLite3Impl) AddTextInfo(ctx context.Context, text *Text, txID string, userID string, device string) error {
	sql := `
INSERT INTO TEXT (
  IS_DELETED,
  ID,
  TEXT,
  TARGET_ID,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  USER_ID,
  DEVICE,
  TX_ID
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
  ?,
  ?
)`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add text sql %s: %w", text.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		text.IsDeleted,
		text.ID,
		text.Text,
		text.TargetID,
		text.RelatedTime.Format(sqlite3impl.TimeLayout),
		text.CreateTime.Format(sqlite3impl.TimeLayout),
		text.CreateApp,
		text.CreateDevice,
		text.CreateUser,
		text.UpdateTime.Format(sqlite3impl.TimeLayout),
		text.UpdateApp,
		text.UpdateDevice,
		text.UpdateUser,
		userID,
		device,
		txID,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to TEXT %s: %w", text.ID, err)
		return err
	}
	return nil
}

func (t *textTempRepositorySQLite3Impl) GetTextsByTXID(ctx context.Context, txID string, userID string, device string) ([]*Text, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  TEXT,
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
FROM TEXT
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at text: %w", err)
		return nil, err
	}
	dataType := "text"

	queryArgs := []interface{}{
		repName,
		dataType,
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get text by tx id sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from TEXT: %w", err)
		return nil, err
	}
	defer rows.Close()

	texts := []*Text{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			text := &Text{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(&text.IsDeleted,
				&text.ID,
				&text.TargetID,
				&text.Text,
				&relatedTimeStr,
				&createTimeStr,
				&text.CreateApp,
				&text.CreateDevice,
				&text.CreateUser,
				&updateTimeStr,
				&text.UpdateApp,
				&text.UpdateDevice,
				&text.UpdateUser,
				&text.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan TEXT: %w", err)
				return nil, err
			}

			text.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in TEXT: %w", relatedTimeStr, err)
				return nil, err
			}
			text.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in TEXT: %w", createTimeStr, err)
				return nil, err
			}
			text.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in TEXT: %w", updateTimeStr, err)
				return nil, err
			}
			texts = append(texts, text)
		}
	}
	return texts, nil
}

func (t *textTempRepositorySQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	sql := `
DELETE FROM TEXT
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete temp text by TXID sql: %w", err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		txID,
		userID,
		device,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at delete temp text by TXID sql: %w", err)
		return err
	}
	return nil
}
