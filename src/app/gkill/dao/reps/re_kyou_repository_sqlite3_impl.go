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
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type reKyouRepositorySQLite3Impl struct {
	filename          string
	db                *sql.DB
	m                 *sync.Mutex
	gkillRepositories *GkillRepositories
}

func NewReKyouRepositorySQLite3Impl(ctx context.Context, filename string, reps *GkillRepositories) (ReKyouRepository, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=60000&_synchronous=1&_mutex=full&_locking_mode=EXCLUSIVE&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

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
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU table to %s: %w", filename, err)
		return nil, err
	}

	return &reKyouRepositorySQLite3Impl{
		filename:          filename,
		db:                db,
		m:                 &sync.Mutex{},
		gkillRepositories: reps,
	}, nil
}
func (r *reKyouRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error) {
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

	repsWithoutRekyou, err := r.GetRepositoriesWithoutReKyouRep(ctx)
	if err != nil {
		err = fmt.Errorf("error at get without rekyou: %w", err)
		return nil, err
	}

	latestDataRepositoryAddresses, err := repsWithoutRekyou.LatestDataRepositoryAddressDAO.GetAllLatestDataRepositoryAddresses(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all latest data repository addresses: %w", err)
		return nil, err
	}

	for _, rekyou := range notDeletedAllReKyous {
		existInRep := false
		for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
			if latestDataRepositoryAddress.TargetID == rekyou.TargetID && !latestDataRepositoryAddress.IsDeleted {
				existInRep = true
				break
			}
		}

		matchID := false
		if query.UseIDs == nil || !*query.UseIDs {
			matchID = true
		} else if *query.UseIDs {
			if query.IDs != nil && len(*query.IDs) != 0 {
				for _, id := range *query.IDs {
					if id == rekyou.ID {
						matchID = true
						break
					}
				}
			}
		}
		if !matchID {
			continue
		}

		if existInRep {
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

func (r *reKyouRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
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

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range kyouHistories {
			if kyou.UpdateTime.Format(sqlite3impl.TimeLayout) == updateTime.Format(sqlite3impl.TimeLayout) {
				return kyou, nil
			}
		}
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
WHERE 
`
	dataType := "rekyou"

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false
	appendGroupBy := false
	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendGroupBy, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}
	commonWhereSQL += " ORDER BY datetime(UPDATE_TIME, 'localtime') DESC "
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from REKYOU %s: %w", id, err)
		return nil, err
	}
	defer rows.Close()

	kyous := []*Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := &Kyou{}
			kyou.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(&kyou.IsDeleted,
				&kyou.ID,
				&relatedTimeStr,
				&createTimeStr,
				&kyou.CreateApp,
				&kyou.CreateDevice,
				&kyou.CreateUser,
				&updateTimeStr,
				&kyou.UpdateApp,
				&kyou.UpdateDevice,
				&kyou.UpdateUser,
				&kyou.RepName,
				&kyou.DataType,
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

func (r *reKyouRepositorySQLite3Impl) FindReKyou(ctx context.Context, query *find.FindQuery) ([]*ReKyou, error) {
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

	repsWithoutRekyou, err := r.GetRepositoriesWithoutReKyouRep(ctx)
	if err != nil {
		err = fmt.Errorf("error at get without rekyou: %w", err)
		return nil, err
	}

	latestDataRepositoryAddresses, err := repsWithoutRekyou.LatestDataRepositoryAddressDAO.GetAllLatestDataRepositoryAddresses(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all latest data repository addresses: %w", err)
		return nil, err
	}

	for _, rekyou := range notDeletedAllReKyous {
		existInRep := false
		for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
			if latestDataRepositoryAddress.TargetID == rekyou.TargetID {
				existInRep = true
				break
			}
		}
		if existInRep {
			matchReKyous = append(matchReKyous, rekyou)
		}
	}
	return matchReKyous, nil
}

func (r *reKyouRepositorySQLite3Impl) GetReKyou(ctx context.Context, id string, updateTime *time.Time) (*ReKyou, error) {
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

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, kyou := range reKyouHistories {
			if kyou.UpdateTime.Format(sqlite3impl.TimeLayout) == updateTime.Format(sqlite3impl.TimeLayout) {
				return kyou, nil
			}
		}
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
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM REKYOU
WHERE  
`
	repName, err := r.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at rekyou: %w", err)
		return nil, err
	}

	dataType := "rekyou"

	trueValue := true
	ids := []string{id}
	query := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &ids,
	}
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false
	appendGroupBy := false
	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendGroupBy, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}
	commonWhereSQL += " ORDER BY datetime(UPDATE_TIME, 'localtime') DESC "
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get rekyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from REKYOU %s: %w", err)
		return nil, err
	}
	defer rows.Close()

	reKyous := []*ReKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			reKyou := &ReKyou{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(&reKyou.IsDeleted,
				&reKyou.ID,
				&reKyou.TargetID,
				&relatedTimeStr,
				&createTimeStr,
				&reKyou.CreateApp,
				&reKyou.CreateDevice,
				&reKyou.CreateUser,
				&updateTimeStr,
				&reKyou.UpdateApp,
				&reKyou.UpdateDevice,
				&reKyou.UpdateUser,
				&reKyou.RepName,
				&reKyou.DataType,
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
	sql := `
INSERT INTO REKYOU (
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
  ?
)`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add rekyou sql %s: %w", rekyou.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
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
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to REKYOU %s: %w", rekyou.ID, err)
		return err
	}
	return nil
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
  UPDATE_USER,
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM REKYOU
WHERE 
`

	repName, err := r.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at rekyou: %w", err)
		return nil, err
	}

	dataType := "rekyou"

	queryArgs := []interface{}{
		repName,
		dataType,
	}
	query := &find.FindQuery{}

	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "RELATED_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := true
	appendGroupBy := true
	findWordUseLike := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendGroupBy, appendOrderBy, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := r.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all rekyous sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from REKYOU %s: %w", err)
		return nil, err
	}
	defer rows.Close()

	reKyous := []*ReKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			reKyou := &ReKyou{}
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err = rows.Scan(&reKyou.IsDeleted,
				&reKyou.ID,
				&reKyou.TargetID,
				&relatedTimeStr,
				&createTimeStr,
				&reKyou.CreateApp,
				&reKyou.CreateDevice,
				&reKyou.CreateUser,
				&updateTimeStr,
				&reKyou.UpdateApp,
				&reKyou.UpdateDevice,
				&reKyou.UpdateUser,
				&reKyou.RepName,
				&reKyou.DataType,
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

func (r *reKyouRepositorySQLite3Impl) GetRepositoriesWithoutReKyouRep(ctx context.Context) (*GkillRepositories, error) {
	withoutRekyouReps := Repositories{}
	for _, rep := range r.gkillRepositories.Reps {
		repIsRekyouRep := false

		repPath, err := rep.GetPath(ctx, "")
		if err != nil {
			err = fmt.Errorf("error at get reps path: %w", err)
			return nil, err
		}

		for _, reKyouRep := range r.gkillRepositories.ReKyouReps.ReKyouRepositories {
			rekyouRepPath, err := reKyouRep.GetPath(ctx, "")
			if err != nil {
				err = fmt.Errorf("error at get rekyous reps path: %w", err)
				return nil, err
			}

			if filepath.ToSlash(repPath) == filepath.ToSlash(rekyouRepPath) {
				repIsRekyouRep = true
				break
			}
		}
		if repIsRekyouRep {
			continue
		}
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}

	withoutRekyouGkillRepsValue := *r.gkillRepositories
	withoutRekyouGkillRepsValue.Reps = withoutRekyouReps
	withoutRekyouGkillRepsValue.ReKyouReps.GkillRepositories = &withoutRekyouGkillRepsValue
	withoutRekyouGkillRepsValue.ReKyouReps.ReKyouRepositories = nil
	return &withoutRekyouGkillRepsValue, nil
}
