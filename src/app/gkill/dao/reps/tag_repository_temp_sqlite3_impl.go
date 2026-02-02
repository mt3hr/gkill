package reps

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type tagTempRepositorySQLite3Impl tagRepositorySQLite3Impl

func NewTagTempRepositorySQLite3Impl(ctx context.Context, db *sql.DB, m *sync.Mutex) (TagTempRepository, error) {
	filename := "tag_temp"
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
  UPDATE_USER NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  TX_ID NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create TAG table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_TAG ON TAG (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG index statement %s: %w", filename, err)
		return nil, err
	}

	indexTargetIDSQL := `CREATE INDEX IF NOT EXISTS INDEX_TAG_TARGET_ID ON TAG (TARGET_ID, UPDATE_TIME DESC);`
	gkill_log.TraceSQL.Printf("sql: %s", indexTargetIDSQL)
	indexTargetIDStmt, err := db.PrepareContext(ctx, indexTargetIDSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG_TARGET_ID index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexTargetIDStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexTargetIDSQL)
	_, err = indexTargetIDStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_TARGET_ID index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", indexTargetIDSQL)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME index statement %s: %w", filename, err)
		return nil, err
	}

	indexIDUpdateTimeSQL := `CREATE INDEX IF NOT EXISTS INDEX_TAG_ID_UPDATE_TIME ON TAG (ID, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexIDUpdateTimeSQL)
	indexIDUpdateTimeStmt, err := db.PrepareContext(ctx, indexIDUpdateTimeSQL)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexIDUpdateTimeStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexIDUpdateTimeSQL)
	_, err = indexIDUpdateTimeStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = indexIDUpdateTimeStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create TAG_ID_UPDATE_TIME table to %s: %w", filename, err)
		return nil, err
	}

	return &tagTempRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        m,
	}, nil
}
func (t *tagTempRepositorySQLite3Impl) FindTags(ctx context.Context, query *find.FindQuery) ([]*Tag, error) {
	impl := tagRepositorySQLite3Impl(*t)
	return impl.FindTags(ctx, query)
}

func (t *tagTempRepositorySQLite3Impl) Close(ctx context.Context) error {
	impl := tagRepositorySQLite3Impl(*t)
	return impl.Close(ctx)
}

func (t *tagTempRepositorySQLite3Impl) GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error) {
	impl := tagRepositorySQLite3Impl(*t)
	return impl.GetTag(ctx, id, updateTime)
}

func (t *tagTempRepositorySQLite3Impl) GetTagsByTagName(ctx context.Context, tagname string) ([]*Tag, error) {
	impl := tagRepositorySQLite3Impl(*t)
	return impl.GetTagsByTagName(ctx, tagname)
}

func (t *tagTempRepositorySQLite3Impl) GetTagsByTargetID(ctx context.Context, target_id string) ([]*Tag, error) {
	impl := tagRepositorySQLite3Impl(*t)
	return impl.GetTagsByTargetID(ctx, target_id)
}

func (t *tagTempRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	impl := tagRepositorySQLite3Impl(*t)
	return impl.UpdateCache(ctx)
}

func (t *tagTempRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("GetPath not implemented for tag_temp repository")
}

func (t *tagTempRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "tag_temp", nil
}

func (t *tagTempRepositorySQLite3Impl) GetTagHistories(ctx context.Context, id string) ([]*Tag, error) {
	impl := tagRepositorySQLite3Impl(*t)
	return impl.GetTagHistories(ctx, id)
}

func (t *tagTempRepositorySQLite3Impl) AddTagInfo(ctx context.Context, tag *Tag, txID string, userID string, device string) error {
	t.m.Lock()
	defer t.m.Unlock()
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
		err = fmt.Errorf("error at add tag sql %s: %w", tag.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
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
		userID,
		device,
		txID,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to TAG %s: %w", tag.ID, err)
		return err
	}
	return nil
}

func (t *tagTempRepositorySQLite3Impl) GetAllTagNames(ctx context.Context) ([]string, error) {
	impl := tagRepositorySQLite3Impl(*t)
	return impl.GetAllTagNames(ctx)
}

func (t *tagTempRepositorySQLite3Impl) GetAllTags(ctx context.Context) ([]*Tag, error) {
	impl := tagRepositorySQLite3Impl(*t)
	return impl.GetAllTags(ctx)
}

func (t *tagTempRepositorySQLite3Impl) GetTagsByTXID(ctx context.Context, txID string, userID string, device string) ([]*Tag, error) {
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
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM TAG
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at tag: %w", err)
		return nil, err
	}
	dataType := "tag"

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
		err = fmt.Errorf("error at get tag by tx id sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from TAG: %w", err)
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

			if err != nil {
				err = fmt.Errorf("error at read rows at get tag by tx id: %w", err)
				return nil, err
			}

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

func (t *tagTempRepositorySQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	sql := `
DELETE FROM TAG
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete temp tag by TXID sql: %w", err)
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
		err = fmt.Errorf("error at delete temp tag by TXID sql: %w", err)
		return err
	}
	return nil
}

func (t *tagTempRepositorySQLite3Impl) UnWrapTyped() ([]TagTempRepository, error) {
	return []TagTempRepository{t}, nil
}

func (t *tagTempRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error) {
	return nil, fmt.Errorf("not implements GetLatestDataRepositoryAddress at temp rep")
}
