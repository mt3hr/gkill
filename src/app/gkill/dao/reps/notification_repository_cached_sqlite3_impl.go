package reps

import (
	"context"
	sqllib "database/sql"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type notificationRepositoryCachedSQLite3Impl struct {
	dbName          string
	notificationRep NotificationRepository
	cachedDB        *sqllib.DB
	m               *sync.RWMutex
}

func NewNotificationRepositoryCachedSQLite3Impl(ctx context.Context, notificationRep NotificationRepository, cacheDB *sqllib.DB, m *sync.RWMutex, dbName string) (NotificationRepository, error) {
	if m == nil {
		m = &sync.RWMutex{}
	}
	var err error
	sql := `
CREATE TABLE IF NOT EXISTS "` + dbName + `" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
  CONTENT NOT NULL,
  IS_NOTIFICATED NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  REP_NAME NOT NULL,
  NOTIFICATION_TIME_UNIX NOT NULL,
  CREATE_TIME_UNIX NOT NULL,
  UPDATE_TIME_UNIX NOT NULL
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table statement %s: %w", dbName, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS "INDEX_` + dbName + `_UNIX" ON "` + dbName + `"(ID, UPDATE_TIME_UNIX);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexUnixSQL)
	indexUnixStmt, err := cacheDB.PrepareContext(ctx, indexUnixSQL)
	if err != nil {
		err = fmt.Errorf("error at create notification index unix statement %s: %w", dbName, err)
		return nil, err
	}
	defer func() {
		err := indexUnixStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", indexUnixSQL)
	_, err = indexUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create notification index unix to %s: %w", dbName, err)
		return nil, err
	}

	return &notificationRepositoryCachedSQLite3Impl{
		dbName:          dbName,
		cachedDB:        cacheDB,
		notificationRep: notificationRep,
		m:               m,
	}, nil
}
func (n *notificationRepositoryCachedSQLite3Impl) FindNotifications(ctx context.Context, query *find.FindQuery) ([]*Notification, error) {
	n.m.RLock()
	defer n.m.RUnlock()
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
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
  TARGET_ID,
  NOTIFICATION_TIME_UNIX,
  CONTENT,
  IS_NOTIFICATED,
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
FROM ` + n.dbName + `
WHERE 
`

	dataType := "notification"
	queryArgs := []interface{}{
		dataType,
	}

	tableName := n.dbName
	tableNameAlias := n.dbName
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "UPDATE_TIME_UNIX"
	findWordTargetColumns := []string{"CONTENT"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := true
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
	stmt, err := n.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get NOTIFICATION histories sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from NOTIFICATION: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	notifications := []*Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := &Notification{}
			notificationTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			dataType := ""

			err = rows.Scan(
				&notification.IsDeleted,
				&notification.ID,
				&notification.TargetID,
				&notificationTimeUnix,
				&notification.Content,
				&notification.IsNotificated,
				&createTimeUnix,
				&notification.CreateApp,
				&notification.CreateDevice,
				&notification.CreateUser,
				&updateTimeUnix,
				&notification.UpdateApp,
				&notification.UpdateDevice,
				&notification.UpdateUser,
				&notification.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from NOTIFICATION: %w", err)
				return nil, err
			}

			notification.NotificationTime = time.Unix(notificationTimeUnix, 0).Local()
			notification.CreateTime = time.Unix(createTimeUnix, 0).Local()
			notification.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			notifications = append(notifications, notification)
		}
	}
	return notifications, nil
}

func (n *notificationRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	n.m.Lock()
	defer n.m.Unlock()
	err := n.notificationRep.Close(ctx)
	if err != nil {
		return err
	}
	if gkill_options.CacheNotificationReps == nil || !*gkill_options.CacheNotificationReps {
		err = n.cachedDB.Close()
		if err != nil {
			return err
		}
	} else {
		_, err = n.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+n.dbName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *notificationRepositoryCachedSQLite3Impl) GetNotification(ctx context.Context, id string, updateTime *time.Time) (*Notification, error) {
	n.m.RLock()
	defer n.m.RUnlock()
	// 最新のデータを返す
	notificationHistories, err := n.GetNotificationHistories(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get notification histories from NOTIFICATION %s: %w", id, err)
		return nil, err
	}

	// なければnilを返す
	if len(notificationHistories) == 0 {
		return nil, nil
	}

	// updateTimeが指定されていれば一致するものを返す
	if updateTime != nil {
		for _, notification := range notificationHistories {
			if notification.UpdateTime.Format(sqlite3impl.TimeLayout) == updateTime.Format(sqlite3impl.TimeLayout) {
				return notification, nil
			}
		}
		return nil, nil
	}

	return notificationHistories[0], nil
}

func (n *notificationRepositoryCachedSQLite3Impl) GetNotificationsByTargetID(ctx context.Context, target_id string) ([]*Notification, error) {
	n.m.RLock()
	defer n.m.RUnlock()
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  NOTIFICATION_TIME_UNIX,
  CONTENT,
  IS_NOTIFICATED,
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
FROM ` + n.dbName + `
WHERE 
`

	dataType := "notification"

	trueValue := true
	targetIDs := []string{target_id}
	query := &find.FindQuery{
		UseWords: &trueValue,
		Words:    &targetIDs,
	}
	queryArgs := []interface{}{
		dataType,
	}

	tableName := n.dbName
	tableNameAlias := n.dbName
	whereCounter := 0
	var onlyLatestData bool
	relatedTimeColumnName := "UPDATE_TIME_UNIX"
	findWordTargetColumns := []string{"TARGET_ID"}
	ignoreFindWord := false
	appendOrderBy := false
	findWordUseLike := false
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := n.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get notification histories sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from NOTIFICATION: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	notifications := []*Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := &Notification{}
			notificationTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			dataType := ""

			err = rows.Scan(
				&notification.IsDeleted,
				&notification.ID,
				&notification.TargetID,
				&notificationTimeUnix,
				&notification.Content,
				&notification.IsNotificated,
				&createTimeUnix,
				&notification.CreateApp,
				&notification.CreateDevice,
				&notification.CreateUser,
				&updateTimeUnix,
				&notification.UpdateApp,
				&notification.UpdateDevice,
				&notification.UpdateUser,
				&notification.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from NOTIFICATION: %w", err)
				return nil, err
			}

			notification.NotificationTime = time.Unix(notificationTimeUnix, 0).Local()
			notification.CreateTime = time.Unix(createTimeUnix, 0).Local()
			notification.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			notifications = append(notifications, notification)
		}
	}
	return notifications, nil
}

func (n *notificationRepositoryCachedSQLite3Impl) GetNotificationsBetweenNotificationTime(ctx context.Context, startTime time.Time, endTime time.Time) ([]*Notification, error) {
	n.m.RLock()
	defer n.m.RUnlock()
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  NOTIFICATION_TIME_UNIX,
  CONTENT,
  IS_NOTIFICATED,
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
FROM ` + n.dbName + `
WHERE 
`
	sql += " (NOTIFICATION_TIME_UNIX BETWEEN ? AND ?) "

	dataType := "notification"

	query := &find.FindQuery{}
	queryArgs := []interface{}{
		dataType,
		startTime.Unix(),
		endTime.Unix(),
	}

	tableName := n.dbName
	tableNameAlias := n.dbName
	whereCounter := 1
	var onlyLatestData bool
	relatedTimeColumnName := "NOTIFICATION_TIME_UNIX"
	findWordTargetColumns := []string{"CONTENT"}
	ignoreFindWord := false
	appendOrderBy := true
	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := n.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get notification between notification time sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from NOTIFICATION: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	notifications := []*Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := &Notification{}
			notificationTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)

			dataType := ""

			err = rows.Scan(
				&notification.IsDeleted,
				&notification.ID,
				&notification.TargetID,
				&notificationTimeUnix,
				&notification.Content,
				&notification.IsNotificated,
				&createTimeUnix,
				&notification.CreateApp,
				&notification.CreateDevice,
				&notification.CreateUser,
				&updateTimeUnix,
				&notification.UpdateApp,
				&notification.UpdateDevice,
				&notification.UpdateUser,
				&notification.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from NOTIFICATION: %w", err)
				return nil, err
			}

			notification.NotificationTime = time.Unix(notificationTimeUnix, 0).Local()
			notification.CreateTime = time.Unix(createTimeUnix, 0).Local()
			notification.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			notifications = append(notifications, notification)
		}
	}
	return notifications, nil
}

func (n *notificationRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	n.m.Lock()
	defer n.m.Unlock()
	allNotifications, err := n.notificationRep.GetNotificationsBetweenNotificationTime(ctx, time.Unix(0, 0), time.Unix(math.MaxInt64, 0))
	if err != nil {
		err = fmt.Errorf("error at get all notifications at update cache: %w", err)
		return err
	}

	n.m.Lock()
	defer n.m.Unlock()

	tx, err := n.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add notifications: %w", err)
		return err
	}

	sql := `DELETE FROM ` + n.dbName
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table statement %s: %w", "memory", err)
		return err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at delete NOTIFICATION table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + n.dbName + ` (
  IS_DELETED,
  ID,
  CONTENT,
  TARGET_ID,
  IS_NOTIFICATED,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  NOTIFICATION_TIME_UNIX,
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
  ?,
  ?,
  ?
)`

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add NOTIFICATION sql: %w", err)
		return err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	for _, notification := range allNotifications {
		select {
		case <-ctx.Done():
			tx.Rollback()
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			queryArgs := []interface{}{
				notification.IsDeleted,
				notification.ID,
				notification.Content,
				notification.TargetID,
				notification.IsNotificated,
				notification.CreateApp,
				notification.CreateDevice,
				notification.CreateUser,
				notification.UpdateApp,
				notification.UpdateDevice,
				notification.UpdateUser,
				notification.NotificationTime.Unix(),
				notification.CreateTime.Unix(),
				notification.UpdateTime.Unix(),
			}
			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to NOTIFICATION %s: %w", notification.ID, err)
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
		err = fmt.Errorf("error at commit transaction for add notifications: %w", err)
		return err
	}
	return nil
}

func (n *notificationRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return n.notificationRep.GetPath(ctx, id)
}

func (n *notificationRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return n.notificationRep.GetRepName(ctx)
}

func (n *notificationRepositoryCachedSQLite3Impl) GetNotificationHistories(ctx context.Context, id string) ([]*Notification, error) {
	n.m.RLock()
	defer n.m.RUnlock()
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  NOTIFICATION_TIME_UNIX,
  CONTENT,
  IS_NOTIFICATED,
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
FROM ` + n.dbName + `
WHERE 
`

	repName, err := n.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at notification: %w", err)
		return nil, err
	}
	dataType := "notification"

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

	tableName := n.dbName
	tableNameAlias := n.dbName
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "UPDATE_TIME_UNIX"
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
	stmt, err := n.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get notification histories sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from NOTIFICATION: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	notifications := []*Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := &Notification{}
			notificationTimeUnix, createTimeUnix, updateTimeUnix := int64(0), int64(0), int64(0)
			dataType := ""

			err = rows.Scan(
				&notification.IsDeleted,
				&notification.ID,
				&notification.TargetID,
				&notificationTimeUnix,
				&notification.Content,
				&notification.IsNotificated,
				&createTimeUnix,
				&notification.CreateApp,
				&notification.CreateDevice,
				&notification.CreateUser,
				&updateTimeUnix,
				&notification.UpdateApp,
				&notification.UpdateDevice,
				&notification.UpdateUser,
				&notification.RepName,
				&dataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan from NOTIFICATION: %w", err)
				return nil, err
			}

			notification.NotificationTime = time.Unix(notificationTimeUnix, 0).Local()
			notification.CreateTime = time.Unix(createTimeUnix, 0).Local()
			notification.UpdateTime = time.Unix(updateTimeUnix, 0).Local()
			notifications = append(notifications, notification)
		}
	}
	return notifications, nil
}
func (n *notificationRepositoryCachedSQLite3Impl) AddNotificationInfo(ctx context.Context, notification *Notification) error {
	n.m.Lock()
	defer n.m.Unlock()
	sql := `
INSERT INTO ` + n.dbName + ` (
  IS_DELETED,
  ID,
  CONTENT,
  TARGET_ID,
  IS_NOTIFICATED,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  NOTIFICATION_TIME_UNIX,
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
  ?,
  ?,
  ?
)`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := n.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add NOTIFICATION sql %s: %w", notification.ID, err)
		return err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		notification.IsDeleted,
		notification.ID,
		notification.Content,
		notification.TargetID,
		notification.IsNotificated,
		notification.CreateApp,
		notification.CreateDevice,
		notification.CreateUser,
		notification.UpdateApp,
		notification.UpdateDevice,
		notification.UpdateUser,
		notification.RepName,
		notification.NotificationTime.Unix(),
		notification.CreateTime.Unix(),
		notification.UpdateTime.Unix(),
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to NOTIFICATION %s: %w", notification.ID, err)
		return err
	}
	return nil
}

func (n *notificationRepositoryCachedSQLite3Impl) UnWrapTyped() ([]NotificationRepository, error) {
	unWraped, err := n.notificationRep.UnWrapTyped()
	if err != nil {
		return nil, err
	}
	return unWraped, nil
}
