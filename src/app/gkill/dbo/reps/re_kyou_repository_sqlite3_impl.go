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

	"github.com/mt3hr/gkill/src/app/gkill/dbo/sqlite3impl"
)

// ˄

type reKyouRepositorySQLite3Impl struct {
	// ˅
	filename string
	db       *sql.DB
	m        *sync.Mutex
	reps     *Repositories
	// ˄
}

// ˅
func NewReKyouRepositorySQLite3Impl(ctx context.Context, filename string, reps *Repositories) (ReKyouRepository, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "REKYOU" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
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
		err = fmt.Errorf("error at create REKYOU table statement %s: %w", filename, err)
		return nil, err
	}

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU table to %s: %w", filename, err)
		return nil, err
	}

	return &reKyouRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
		reps:     reps,
	}, nil
}
func (r *reKyouRepositorySQLite3Impl) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	matchKyous := []*Kyou{}

	// 未削除ReKyouを抽出
	notDeletedAllReKyous := []*ReKyou{}
	allReKyous, err := r.GetReKyousAllLatest(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rekyous all latest :%w", err)
		return nil, err
	}
	for _, rekyou := range allReKyous {
		if !rekyou.IsDeleted {
			notDeletedAllReKyous = append(notDeletedAllReKyous, rekyou)
		}
	}

	// ReKyou対象が検索ヒットすれば返す
	// 検索用クエリJSONを作成
	queryJSONForFindKyouTemplate := ""
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at rekyou %s: %w", queryJSON, err)
		return nil, err
	}
	queryMap["is_deleted"] = "false"
	queryMap["use_ids"] = "true"
	queryMap["ids"] = "[\"%s\"]"
	marshaledJSONb, err := json.Marshal(queryMap)
	if err != nil {
		err = fmt.Errorf("error at marshal json: %w", err)
		return nil, err
	}
	queryJSONForFindKyouTemplate = string(marshaledJSONb)

	reps, err := r.GetRepositories(ctx)
	if err != nil {
		err = fmt.Errorf("error at get repositories: %w", err)
		return nil, err
	}

	for _, rekyou := range notDeletedAllReKyous {
		kyous, err := reps.FindKyous(ctx, fmt.Sprintf(queryJSONForFindKyouTemplate, rekyou.TargetID))
		if err != nil {
			err = fmt.Errorf("error at find kyous: %w", err)
			return nil, err
		}
		// 存在すれば検索ヒットとする
		if len(kyous) != 0 {
			kyou := &Kyou{}
			kyou.IsDeleted = rekyou.IsDeleted
			kyou.ID = rekyou.ID
			kyou.RepName = rekyou.RepName
			kyou.RelatedTime = rekyou.RelatedTime
			kyou.DataType = rekyou.DataType
			kyou.CreateTime = rekyou.CreateTime
			kyou.CreateApp = rekyou.CreateApp
			kyou.CreateDevice = rekyou.CreateDevice
			kyou.CreateUser = rekyou.CreateUser
			kyou.UpdateTime = rekyou.UpdateTime
			kyou.UpdateApp = rekyou.UpdateApp
			kyou.UpdateUser = rekyou.UpdateUser
			kyou.UpdateDevice = rekyou.UpdateDevice
			matchKyous = append(matchKyous, kyou)
		}
	}
	return matchKyous, nil
}

func (r *reKyouRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	// 最新のデータを返す
	kyouHistories, err := r.GetKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories from REKYOU %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(kyouHistories) == 0 {
		return nil, nil
	}

	return kyouHistories[0], nil
}

func (r *reKyouRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	repName, err := r.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at rekyou: %w", err)
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
FROM REKYOU
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
`
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}

	dataType := "rekyou"
	rows, err := stmt.QueryContext(ctx, repName, id, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from REKYOU %s: %w", id, err)
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
				err = fmt.Errorf("error at parse related time %s at %s in REKYOU: %w", relatedTimeStr, id, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s at %s in REKYOU: %w", createTimeStr, id, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s at %s in REKYOU: %w", updateTimeStr, id, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (r *reKyouRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return filepath.Abs(r.filename)
}

func (r *reKyouRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (r *reKyouRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := r.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path rekyou rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (r *reKyouRepositorySQLite3Impl) Close(ctx context.Context) error {
	return r.db.Close()
}

func (r *reKyouRepositorySQLite3Impl) FindReKyou(ctx context.Context, queryJSON string) ([]*ReKyou, error) {
	matchReKyous := []*ReKyou{}

	// 未削除ReKyouを抽出
	notDeletedAllReKyous := []*ReKyou{}
	allReKyous, err := r.GetReKyousAllLatest(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rekyous all latest :%w", err)
		return nil, err
	}
	for _, rekyou := range allReKyous {
		if !rekyou.IsDeleted {
			notDeletedAllReKyous = append(notDeletedAllReKyous, rekyou)
		}
	}

	// ReKyou対象が検索ヒットすれば返す
	// 検索用クエリJSONを作成
	queryJSONForFindKyouTemplate := ""
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at rekyou %s: %w", queryJSON, err)
		return nil, err
	}
	queryMap["is_deleted"] = "false"
	queryMap["use_ids"] = "true"
	queryMap["ids"] = "[\"%s\"]"
	marshaledJSONb, err := json.Marshal(queryMap)
	if err != nil {
		err = fmt.Errorf("error at marshal json: %w", err)
		return nil, err
	}
	queryJSONForFindKyouTemplate = string(marshaledJSONb)

	reps, err := r.GetRepositories(ctx)
	if err != nil {
		err = fmt.Errorf("error at get repositories: %w", err)
		return nil, err
	}

	for _, rekyou := range notDeletedAllReKyous {
		kyous, err := reps.FindKyous(ctx, fmt.Sprintf(queryJSONForFindKyouTemplate, rekyou.TargetID))
		if err != nil {
			err = fmt.Errorf("error at find kyous: %w", err)
			return nil, err
		}
		// 存在すれば検索ヒットとする
		if len(kyous) != 0 {
			matchReKyous = append(matchReKyous, rekyou)
		}
	}
	return matchReKyous, nil
}

func (r *reKyouRepositorySQLite3Impl) GetReKyou(ctx context.Context, id string) (*ReKyou, error) {
	// 最新のデータを返す
	reKyouHistories, err := r.GetReKyouHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get rekyou histories from REKYOU %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(reKyouHistories) == 0 {
		return nil, nil
	}

	return reKyouHistories[0], nil
}

func (r *reKyouRepositorySQLite3Impl) GetReKyouHistories(ctx context.Context, id string) ([]*ReKyou, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM REKYOU
WHERE TARGET_ID LIKE ?
`

	sql += `;`

	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get rekyou histories sql: %w", err)
		return nil, err
	}

	repName, err := r.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at rekyou: %w", err)
		return nil, err
	}

	dataType := "rekyou"
	rows, err := stmt.QueryContext(ctx, repName, dataType, id)
	if err != nil {
		err = fmt.Errorf("error at select from REKYOU %s: %w", err)
		return nil, err
	}

	reKyous := []*ReKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			reKyou := &ReKyou{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			repName, dataType := "", ""

			err = rows.Scan(reKyou.IsDeleted,
				reKyou.ID,
				reKyou.TargetID,
				relatedTimeStr,
				createTimeStr,
				reKyou.CreateApp,
				reKyou.CreateDevice,
				reKyou.CreateUser,
				updateTimeStr,
				reKyou.UpdateApp,
				reKyou.UpdateDevice,
				reKyou.UpdateUser,
				repName,
				dataType,
			)

			reKyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in REKYOU: %w", relatedTimeStr, err)
				return nil, err
			}
			reKyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in REKYOU: %w", createTimeStr, err)
				return nil, err
			}
			reKyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in REKYOU: %w", updateTimeStr, err)
				return nil, err
			}
			reKyous = append(reKyous, reKyou)
		}
	}
	return reKyous, nil
}

func (r *reKyouRepositorySQLite3Impl) AddReKyouInfo(ctx context.Context, rekyou *ReKyou) error {
	// ˅
	sql := `
INSERT INTO REKYOU
  IS_DELETED,
  ID,
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
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add rekyou sql %s: %w", rekyou.ID, err)
		return err
	}

	_, err = stmt.ExecContext(ctx,
		rekyou.IsDeleted,
		rekyou.ID,
		rekyou.TargetID,
		rekyou.RelatedTime.Format(sqlite3impl.TimeLayout),
		rekyou.CreateTime.Format(sqlite3impl.TimeLayout),
		rekyou.CreateApp,
		rekyou.CreateDevice,
		rekyou.CreateUser,
		rekyou.UpdateTime.Format(sqlite3impl.TimeLayout),
		rekyou.UpdateApp,
		rekyou.UpdateDevice,
		rekyou.UpdateUser,
	)
	if err != nil {
		err = fmt.Errorf("error at insert in to REKYOU %s: %w", rekyou.ID, err)
		return err
	}
	return nil
	// ˄
}

func (r *reKyouRepositorySQLite3Impl) GetReKyousAllLatest(ctx context.Context) ([]*ReKyou, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM REKYOU
`

	// UPDATE_TIMEが一番上のものだけを抽出
	sql += `
GROUP BY ID
HAVING MAX(datetime(UPDATE_TIME, 'localtime'))
`

	sql += `;`

	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all rekyous sql: %w", err)
		return nil, err
	}

	repName, err := r.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at rekyou: %w", err)
		return nil, err
	}

	dataType := "rekyou"
	rows, err := stmt.QueryContext(ctx, repName, dataType)
	if err != nil {
		err = fmt.Errorf("error at select from REKYOU %s: %w", err)
		return nil, err
	}

	reKyous := []*ReKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			reKyou := &ReKyou{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""
			repName, dataType := "", ""

			err = rows.Scan(reKyou.IsDeleted,
				reKyou.ID,
				reKyou.TargetID,
				relatedTimeStr,
				createTimeStr,
				reKyou.CreateApp,
				reKyou.CreateDevice,
				reKyou.CreateUser,
				updateTimeStr,
				reKyou.UpdateApp,
				reKyou.UpdateDevice,
				reKyou.UpdateUser,
				repName,
				dataType,
			)

			reKyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in REKYOU: %w", relatedTimeStr, err)
				return nil, err
			}
			reKyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in REKYOU: %w", createTimeStr, err)
				return nil, err
			}
			reKyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in REKYOU: %w", updateTimeStr, err)
				return nil, err
			}
			reKyous = append(reKyous, reKyou)
		}
	}
	return reKyous, nil
}

func (r *reKyouRepositorySQLite3Impl) GetRepositories(ctx context.Context) (*Repositories, error) {
	return r.reps, nil
}

// ˄
