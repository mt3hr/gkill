package reps

import (
	"context"
	sqllib "database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type reKyouRepositoryCachedSQLite3Impl struct {
	dbName            string
	rekyouRep         ReKyouRepository
	cachedDB          *sqllib.DB
	m                 *sync.Mutex
	gkillRepositories *GkillRepositories
}

func NewReKyouRepositoryCachedSQLite3Impl(ctx context.Context, rekyouRep ReKyouRepository, gkillRepositories *GkillRepositories, cacheDB *sqllib.DB, m *sync.Mutex, dbName string) (ReKyouRepository, error) {
	if m == nil {
		m = &sync.Mutex{}
	}
	var err error
	sql := `
CREATE TABLE IF NOT EXISTS "` + dbName + `" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  REP_NAME NOT NULL,
  RELATED_TIME_UNIX NOT NULL,
  CREATE_TIME_UNIX NOT NULL,
  UPDATE_TIME_UNIX NOT NULL
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU table statement %s: %w", dbName, err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU table to %s: %w", dbName, err)
		return nil, err
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `_UNIX" ON "` + dbName + `"(ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create rekyou index unix statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexUnixStmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexUnixSQL)
	_, err = indexUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create rekyou index unix to %s: %w", dbName, err)
		return nil, err
	}

	return &reKyouRepositoryCachedSQLite3Impl{
		dbName:            dbName,
		rekyouRep:         rekyouRep,
		cachedDB:          cacheDB,
		m:                 m,
		gkillRepositories: gkillRepositories,
	}, nil
}
func (r *reKyouRepositoryCachedSQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	r.m.Lock()
	defer r.m.Unlock()
	matchKyous := map[string][]*Kyou{}

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

			if _, exist := matchKyous[kyou.ID]; !exist {
				matchKyous[kyou.ID] = []*Kyou{}
			}
			matchKyous[kyou.ID] = append(matchKyous[kyou.ID], kyou)
		}
	}
	return matchKyous, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
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
			if kyou.UpdateTime.Unix() == updateTime.Unix() {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return kyouHistories[0], nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	r.m.Lock()
	defer r.m.Unlock()
	sql := `
SELECT 
  IS_DELETED,
  ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + r.dbName + `
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
		dataType,
	}

	tableName := r.dbName
	tableNameAlias := r.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false
	if query.OnlyLatestData != nil {
		onlyLatestData = *query.OnlyLatestData
	} else {
		onlyLatestData = false
	}
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := r.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories sql %s: %w", id, err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
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
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&kyou.IsDeleted,
				&kyou.ID,
				&relatedTimeUnix,
				&createTimeUnix,
				&kyou.CreateApp,
				&kyou.CreateDevice,
				&kyou.CreateUser,
				&updateTimeUnix,
				&kyou.UpdateApp,
				&kyou.UpdateDevice,
				&kyou.UpdateUser,
				&kyou.RepName,
				&kyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from REKYOU %s: %w", id, err)
				return nil, err
			}

			kyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			kyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			kyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			kyous = append(kyous, kyou)
		}
	}
	return kyous, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return r.dbName, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	trueValue := true
	falseValue := false
	query := &find.FindQuery{
		UpdateCache:    &trueValue,
		OnlyLatestData: &falseValue,
	}

	allReKyous, err := r.rekyouRep.FindReKyou(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get all rekyou at update cache: %w", err)
		return err
	}

	r.m.Lock()
	defer r.m.Unlock()

	tx, err := r.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add rekyou: %w", err)
		return err
	}

	sql := `DELETE FROM ` + r.dbName
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create REKYOU table statement %s: %w", "memory", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at delete REKYOU table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + r.dbName + ` (
  IS_DELETED,
  ID,
  TARGET_ID,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  UPDATE_TIME_UNIX
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

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add rekyou sql: %w", err)
		return err
	}
	defer insertStmt.Close()

	for _, rekyou := range allReKyous {
		select {
		case <-ctx.Done():
			tx.Rollback()
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			queryArgs := []interface{}{
				rekyou.IsDeleted,
				rekyou.ID,
				rekyou.TargetID,
				rekyou.CreateApp,
				rekyou.CreateDevice,
				rekyou.CreateUser,
				rekyou.UpdateApp,
				rekyou.UpdateDevice,
				rekyou.UpdateUser,
				rekyou.RepName,
				rekyou.RelatedTime.Unix(),
				rekyou.CreateTime.Unix(),
				rekyou.UpdateTime.Unix(),
			}
			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to REKYOU %s: %w", rekyou.ID, err)
				return err
			}
			return nil
		}()
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit transaction for add rekyou: %w", err)
		return err
	}
	return nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return r.rekyouRep.GetRepName(ctx)
}

func (r *reKyouRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	r.m.Lock()
	defer r.m.Unlock()

	if gkill_options.CacheReKyouReps != nil && *gkill_options.CacheReKyouReps {
		_, err := r.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+r.dbName)
		return err
	}
	return r.cachedDB.Close()
}

func (r *reKyouRepositoryCachedSQLite3Impl) FindReKyou(ctx context.Context, query *find.FindQuery) ([]*ReKyou, error) {
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

func (r *reKyouRepositoryCachedSQLite3Impl) GetReKyou(ctx context.Context, id string, updateTime *time.Time) (*ReKyou, error) {
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
			if kyou.UpdateTime.Unix() == updateTime.Unix() {
				return kyou, nil
			}
		}
		return nil, nil
	}

	return reKyouHistories[0], nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetReKyouHistories(ctx context.Context, id string) ([]*ReKyou, error) {
	r.m.Lock()
	defer r.m.Unlock()
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + r.dbName + `
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
		dataType,
	}

	tableName := r.dbName
	tableNameAlias := r.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := r.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get rekyou histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from REKYOU: %w", err)
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
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&reKyou.IsDeleted,
				&reKyou.ID,
				&reKyou.TargetID,
				&relatedTimeUnix,
				&createTimeUnix,
				&reKyou.CreateApp,
				&reKyou.CreateDevice,
				&reKyou.CreateUser,
				&updateTimeUnix,
				&reKyou.UpdateApp,
				&reKyou.UpdateDevice,
				&reKyou.UpdateUser,
				&reKyou.RepName,
				&reKyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from REKYOU %s: %w", id, err)
				return nil, err
			}

			reKyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			reKyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			reKyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			reKyous = append(reKyous, reKyou)
		}
	}
	return reKyous, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) AddReKyouInfo(ctx context.Context, rekyou *ReKyou) error {
	r.m.Lock()
	defer r.m.Unlock()
	sql := `
INSERT INTO ` + r.dbName + ` (
  IS_DELETED,
  ID,
  TARGET_ID,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  UPDATE_TIME_UNIX 
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
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := r.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add rekyou sql %s: %w", rekyou.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		rekyou.IsDeleted,
		rekyou.ID,
		rekyou.TargetID,
		rekyou.CreateApp,
		rekyou.CreateDevice,
		rekyou.CreateUser,
		rekyou.UpdateApp,
		rekyou.UpdateDevice,
		rekyou.UpdateUser,
		rekyou.RepName,
		rekyou.RelatedTime.Unix(),
		rekyou.CreateTime.Unix(),
		rekyou.UpdateTime.Unix(),
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to REKYOU %s: %w", rekyou.ID, err)
		return err
	}
	return nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetReKyousAllLatest(ctx context.Context) ([]*ReKyou, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  RELATED_TIME_UNIX,
  CREATE_TIME_UNIX,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME_UNIX,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + r.dbName + `
WHERE 
`

	dataType := "rekyou"

	queryArgs := []interface{}{
		dataType,
	}
	query := &find.FindQuery{}

	tableName := r.dbName
	tableNameAlias := r.dbName
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "RELATED_TIME_UNIX"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := r.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all rekyous sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from REKYOU: %w", err)
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
			relatedTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			err = rows.Scan(&reKyou.IsDeleted,
				&reKyou.ID,
				&reKyou.TargetID,
				&relatedTimeUnix,
				&createTimeUnix,
				&reKyou.CreateApp,
				&reKyou.CreateDevice,
				&reKyou.CreateUser,
				&updateTimeUnix,
				&reKyou.UpdateApp,
				&reKyou.UpdateDevice,
				&reKyou.UpdateUser,
				&reKyou.RepName,
				&reKyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from REKYOU: %w", err)
				return nil, err
			}

			reKyou.RelatedTime = time.Unix(relatedTimeUnix, 0).Local()
			reKyou.CreateTime = time.Unix(createTimeUnix, 0).Local()
			reKyou.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			reKyous = append(reKyous, reKyou)
		}
	}
	return reKyous, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) GetRepositoriesWithoutReKyouRep(ctx context.Context) (*GkillRepositories, error) {
	withoutRekyouReps := Repositories{}
	for _, rep := range r.gkillRepositories.KmemoReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.gkillRepositories.KCReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.gkillRepositories.URLogReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.gkillRepositories.NlogReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.gkillRepositories.TimeIsReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.gkillRepositories.MiReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.gkillRepositories.LantanaReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.gkillRepositories.IDFKyouReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.gkillRepositories.GitCommitLogReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}

	withoutRekyouGkillRepsValue := r.gkillRepositories
	withoutRekyouGkillRepsValue.Reps = withoutRekyouReps
	withoutRekyouGkillRepsValue.ReKyouReps.GkillRepositories = withoutRekyouGkillRepsValue
	withoutRekyouGkillRepsValue.ReKyouReps.ReKyouRepositories = nil
	return withoutRekyouGkillRepsValue, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) UnWrapTyped() ([]ReKyouRepository, error) {
	return []ReKyouRepository{r}, nil
}

func (r *reKyouRepositoryCachedSQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{r}, nil
}
