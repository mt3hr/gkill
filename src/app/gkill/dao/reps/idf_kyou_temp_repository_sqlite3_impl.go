package reps

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type idfKyouRepositoryTempSQLite3Impl idfKyouRepositorySQLite3Impl

func NewIDFKyouTempRepositorySQLite3Impl(ctx context.Context, db *sql.DB) (IDFKyouTempRepository, error) {
	filename := "temp_db"

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
  UPDATE_USER NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  TX_ID NOT NULL
)
`

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create IDF table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create IDF table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_IDF ON IDF (ID, RELATED_TIME, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create IDF index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create IDF index to %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create IDF table to %s: %w", filename, err)
		return nil, err
	}

	rep := &idfKyouRepositoryTempSQLite3Impl{
		repositoriesRef: &GkillRepositories{},
		// idDBFile:        dbFilename,
		// contentDir:      dir,
		// rootAddress: "/files/" + filepath.Base(dir) + "/",
		// r:         r,
		// autoIDF:   autoIDF,
		// idfIgnore: idfIgnore,
		db: db,
		m:  &sync.Mutex{},
	}

	//r.PathPrefix(rep.rootAddress).Handler(http.StripPrefix(rep.rootAddress, http.FileServer(http.Dir(dir))))

	return rep, nil
}

func (i *idfKyouRepositoryTempSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	impl := idfKyouRepositorySQLite3Impl(*i)
	return impl.FindKyous(ctx, query)
}

func (i *idfKyouRepositoryTempSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	impl := idfKyouRepositorySQLite3Impl(*i)
	return impl.GetKyou(ctx, id, updateTime)
}

func (i *idfKyouRepositoryTempSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	impl := idfKyouRepositorySQLite3Impl(*i)
	return impl.GetKyouHistories(ctx, id)
}

func (i *idfKyouRepositoryTempSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("not implemented yet, use GetKyouHistories instead")
}

func (i *idfKyouRepositoryTempSQLite3Impl) UpdateCache(ctx context.Context) error {
	impl := idfKyouRepositorySQLite3Impl(*i)
	return impl.UpdateCache(ctx)
}

func (i *idfKyouRepositoryTempSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "IDF_TEMP", nil
}

func (i *idfKyouRepositoryTempSQLite3Impl) Close(ctx context.Context) error {
	impl := idfKyouRepositorySQLite3Impl(*i)
	return impl.Close(ctx)
}

func (i *idfKyouRepositoryTempSQLite3Impl) FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]*IDFKyou, error) {
	impl := idfKyouRepositorySQLite3Impl(*i)
	return impl.FindIDFKyou(ctx, query)
}

func (i *idfKyouRepositoryTempSQLite3Impl) GetIDFKyou(ctx context.Context, id string, updateTime *time.Time) (*IDFKyou, error) {
	impl := idfKyouRepositorySQLite3Impl(*i)
	return impl.GetIDFKyou(ctx, id, updateTime)
}

func (i *idfKyouRepositoryTempSQLite3Impl) GetIDFKyouHistories(ctx context.Context, id string) ([]*IDFKyou, error) {
	impl := idfKyouRepositorySQLite3Impl(*i)
	return impl.GetIDFKyouHistories(ctx, id)
}

func (i *idfKyouRepositoryTempSQLite3Impl) IDF(ctx context.Context) error {
	return fmt.Errorf("not implemented yet, use IDF method of idfKyouRepositorySQLite3Impl instead")
}

func (i *idfKyouRepositoryTempSQLite3Impl) AddIDFKyouInfo(ctx context.Context, idfKyou *IDFKyou, txID string, userID string, device string) error {
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
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := i.db.PrepareContext(ctx, sql)
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
		userID,
		device,
		txID,
	}

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to idf %s: %w", idfKyou.ID, err)
		return err
	}
	return nil
}

func (i *idfKyouRepositoryTempSQLite3Impl) HandleFileServe(w http.ResponseWriter, r *http.Request) {
	panic("not implemented yet, use HandleFileServe method of idfKyouRepositorySQLite3Impl instead")
}

func (i *idfKyouRepositoryTempSQLite3Impl) GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error) {
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
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
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
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := i.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyous by TXID sql: %w", err)
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

			idf.FileURL = fmt.Sprintf("/files/%s/%s", targetRepName, filepath.Base(idf.TargetFile))

			// 画像であるか判定
			impl := idfKyouRepositorySQLite3Impl(*i)
			idf.IsImage = impl.isImage(idf.TargetFile)
			idf.IsVideo = impl.isVideo(idf.TargetFile)
			idf.IsAudio = impl.isAudio(idf.TargetFile)

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

			kyous = append(kyous, kyou)
		}
	}
	sort.Slice(kyous, func(i, j int) bool {
		return kyous[i].UpdateTime.After(kyous[j].UpdateTime)
	})
	return kyous, nil
}

func (i *idfKyouRepositoryTempSQLite3Impl) GetIDFKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*IDFKyou, error) {
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
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
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
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := i.db.PrepareContext(ctx, sql)
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

			idf.FileURL = fmt.Sprintf("/files/%s/%s", targetRepName, filepath.Base(idf.TargetFile))

			// 画像であるか判定
			impl := idfKyouRepositorySQLite3Impl(*i)
			idf.IsImage = impl.isImage(idf.TargetFile)
			idf.IsVideo = impl.isVideo(idf.TargetFile)
			idf.IsAudio = impl.isAudio(idf.TargetFile)

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

func (i *idfKyouRepositoryTempSQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	sql := `
DELETE FROM IDF
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := i.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete temp idf kyou by TXID sql: %w", err)
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
		err = fmt.Errorf("error at delete temp idf kyou by TXID sql: %w", err)
		return err
	}
	return nil
}
