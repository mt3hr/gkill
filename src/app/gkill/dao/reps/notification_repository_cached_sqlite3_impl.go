package reps

import (
	"context"
	"database/sql"
	sqllib "database/sql"
	"fmt"
	"math"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type notificationRepositoryCachedSQLite3Impl struct {
	dbName          string
	notificationRep NotificationRepository
	cachedDB        *sqllib.DB
	m               *sync.Mutex
}

func NewNotificationRepositoryCachedSQLite3Impl(ctx context.Context, notificationRep NotificationRepository, cacheDB *sql.DB, m *sync.Mutex, dbName string) (NotificationRepository, error) {
	if m == nil {
		m = &sync.Mutex{}
	}
	var err error
	sql := `
CREATE TABLE IF NOT EXISTS "` + dbName + `" (
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
  NOTIFICATION_TIME NOT NULL,
  CONTENT NOT NULL,
  IS_NOTIFICATED NOT NULL,
  CREATE_TIME NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_TIME NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL,
  REP_NAME NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := cacheDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table statement %s: %w", dbName, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table to %s: %w", dbName, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table to %s: %w", dbName, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_` + dbName + ` ON ` + dbName + ` (ID, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := cacheDB.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION index statement %s: %w", dbName, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION index to %s: %w", dbName, err)
		return nil, err
	}

	return &notificationRepositoryCachedSQLite3Impl{
		dbName:          dbName,
		cachedDB:        cacheDB,
		notificationRep: notificationRep,
		m:               m,
	}, nil
}
func (t *notificationRepositoryCachedSQLite3Impl) FindNotifications(ctx context.Context, query *find.FindQuery) ([]*Notification, error) {
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
  NOTIFICATION_TIME,
  CONTENT,
  IS_NOTIFICATED,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + t.dbName + `
WHERE 
`

	dataType := "notification"
	queryArgs := []interface{}{
		dataType,
	}

	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "UPDATE_TIME"
	findWordTargetColumns := []string{"CONTENT"}
	ignoreFindWord := false
	appendOrderBy := true

	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get NOTIFICATION histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from NOTIFICATION: %w", err)
		return nil, err
	}
	defer rows.Close()

	notifications := []*Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := &Notification{}
			createTimeStr, updateTimeStr, notificationTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(
				&notification.IsDeleted,
				&notification.ID,
				&notification.TargetID,
				&notificationTimeStr,
				&notification.Content,
				&notification.IsNotificated,
				&createTimeStr,
				&notification.CreateApp,
				&notification.CreateDevice,
				&notification.CreateUser,
				&updateTimeStr,
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

			notification.NotificationTime, err = time.Parse(sqlite3impl.TimeLayout, notificationTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse notification time %s in NOTIFICATION: %w", notificationTimeStr, err)
				return nil, err
			}
			notification.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in NOTIFICATION: %w", createTimeStr, err)
				return nil, err
			}
			notification.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in NOTIFICATION: %w", updateTimeStr, err)
				return nil, err
			}
			notifications = append(notifications, notification)
		}
	}
	return notifications, nil
}

func (t *notificationRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	_, err := t.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+t.dbName)
	return err
}

func (t *notificationRepositoryCachedSQLite3Impl) GetNotification(ctx context.Context, id string, updateTime *time.Time) (*Notification, error) {
	// 最新のデータを返す
	notificationHistories, err := t.GetNotificationHistories(ctx, id)
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

func (t *notificationRepositoryCachedSQLite3Impl) GetNotificationsByTargetID(ctx context.Context, target_id string) ([]*Notification, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  NOTIFICATION_TIME,
  CONTENT,
  IS_NOTIFICATED,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + t.dbName + `
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

	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "UPDATE_TIME"
	findWordTargetColumns := []string{"TARGET_ID"}
	ignoreFindWord := false
	appendOrderBy := false

	findWordUseLike := false
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	commonWhereSQL += " ORDER BY datetime(UPDATE_TIME, 'localtime') DESC "
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get notification histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from NOTIFICATION: %w", err)
		return nil, err
	}
	defer rows.Close()

	notifications := []*Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := &Notification{}
			createTimeStr, updateTimeStr, notificationTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(
				&notification.IsDeleted,
				&notification.ID,
				&notification.TargetID,
				&notificationTimeStr,
				&notification.Content,
				&notification.IsNotificated,
				&createTimeStr,
				&notification.CreateApp,
				&notification.CreateDevice,
				&notification.CreateUser,
				&updateTimeStr,
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

			notification.NotificationTime, err = time.Parse(sqlite3impl.TimeLayout, notificationTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse notification time %s in NOTIFICATION: %w", notificationTimeStr, err)
				return nil, err
			}
			notification.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in NOTIFICATION: %w", createTimeStr, err)
				return nil, err
			}
			notification.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in NOTIFICATION: %w", updateTimeStr, err)
				return nil, err
			}
			notifications = append(notifications, notification)
		}
	}
	return notifications, nil
}

func (t *notificationRepositoryCachedSQLite3Impl) GetNotificationsBetweenNotificationTime(ctx context.Context, startTime time.Time, endTime time.Time) ([]*Notification, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  NOTIFICATION_TIME,
  CONTENT,
  IS_NOTIFICATED,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + t.dbName + `
WHERE 
`
	sql += " (datetime(NOTIFICATION_TIME, 'localtime') BETWEEN datetime(?, 'localtime') AND datetime(?, 'localtime')) "

	dataType := "notification"

	query := &find.FindQuery{}
	queryArgs := []interface{}{
		dataType,
		startTime.Format(sqlite3impl.TimeLayout),
		endTime.Format(sqlite3impl.TimeLayout),
	}

	whereCounter := 1
	onlyLatestData := true
	relatedTimeColumnName := "NOTIFICATION_TIME"
	findWordTargetColumns := []string{"CONTENT"}
	ignoreFindWord := false
	appendOrderBy := true

	findWordUseLike := true
	ignoreCase := true
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}

	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get notification between notification time sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from NOTIFICATION: %w", err)
		return nil, err
	}
	defer rows.Close()

	notifications := []*Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := &Notification{}
			createTimeStr, updateTimeStr, notificationTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(
				&notification.IsDeleted,
				&notification.ID,
				&notification.TargetID,
				&notificationTimeStr,
				&notification.Content,
				&notification.IsNotificated,
				&createTimeStr,
				&notification.CreateApp,
				&notification.CreateDevice,
				&notification.CreateUser,
				&updateTimeStr,
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

			notification.NotificationTime, err = time.Parse(sqlite3impl.TimeLayout, notificationTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse notification time %s in NOTIFICATION: %w", notificationTimeStr, err)
				return nil, err
			}
			notification.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in NOTIFICATION: %w", createTimeStr, err)
				return nil, err
			}
			notification.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in NOTIFICATION: %w", updateTimeStr, err)
				return nil, err
			}
			notifications = append(notifications, notification)
		}
	}
	return notifications, nil
}

func (t *notificationRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	// t.m.Lock()
	// defer t.m.Unlock()

	allNotifications, err := t.notificationRep.GetNotificationsBetweenNotificationTime(ctx, time.Unix(0, 0), time.Unix(math.MaxInt64, 0))
	if err != nil {
		err = fmt.Errorf("error at get all notifications at update cache: %w", err)
		return err
	}

	tx, err := t.cachedDB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin transaction for add notifications: %w", err)
		return err
	}

	sql := `DELETE FROM ` + t.dbName
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table statement %s: %w", "memory", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at delete NOTIFICATION table: %w", err)
		return err
	}

	sql = `
INSERT INTO ` + t.dbName + ` (
  IS_DELETED,
  ID,
  NOTIFICATION_TIME,
  CONTENT,
  TARGET_ID,
  IS_NOTIFICATED,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME
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

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add NOTIFICATION sql: %w", err)
		return err
	}
	defer insertStmt.Close()
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
				notification.NotificationTime.Format(sqlite3impl.TimeLayout),
				notification.Content,
				notification.TargetID,
				notification.IsNotificated,
				notification.CreateTime.Format(sqlite3impl.TimeLayout),
				notification.CreateApp,
				notification.CreateDevice,
				notification.CreateUser,
				notification.UpdateTime.Format(sqlite3impl.TimeLayout),
				notification.UpdateApp,
				notification.UpdateDevice,
				notification.UpdateUser,
			}
			gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
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

func (t *notificationRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return t.notificationRep.GetPath(ctx, id)
}

func (t *notificationRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return t.notificationRep.GetRepName(ctx)
}

func (t *notificationRepositoryCachedSQLite3Impl) GetNotificationHistories(ctx context.Context, id string) ([]*Notification, error) {
	var err error

	sql := `
SELECT 
  IS_DELETED,
  ID,
  TARGET_ID,
  NOTIFICATION_TIME,
  CONTENT,
  IS_NOTIFICATED,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME,
  ? AS DATA_TYPE
FROM ` + t.dbName + `
WHERE 
`

	repName, err := t.GetRepName(ctx)
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

	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "UPDATE_TIME"
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false

	findWordUseLike := true
	ignoreCase := false
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	commonWhereSQL += " ORDER BY datetime(UPDATE_TIME, 'localtime') DESC "
	sql += commonWhereSQL

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get notification histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at select from NOTIFICATION: %w", err)
		return nil, err
	}
	defer rows.Close()

	notifications := []*Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := &Notification{}
			createTimeStr, updateTimeStr, notificationTimeStr := "", "", ""
			dataType := ""

			err = rows.Scan(
				&notification.IsDeleted,
				&notification.ID,
				&notification.TargetID,
				&notificationTimeStr,
				&notification.Content,
				&notification.IsNotificated,
				&createTimeStr,
				&notification.CreateApp,
				&notification.CreateDevice,
				&notification.CreateUser,
				&updateTimeStr,
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

			notification.NotificationTime, err = time.Parse(sqlite3impl.TimeLayout, notificationTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse notification time %s in NOTIFICATION: %w", notificationTimeStr, err)
				return nil, err
			}
			notification.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in NOTIFICATION: %w", createTimeStr, err)
				return nil, err
			}
			notification.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in NOTIFICATION: %w", updateTimeStr, err)
				return nil, err
			}
			notifications = append(notifications, notification)
		}
	}
	return notifications, nil
}
func (t *notificationRepositoryCachedSQLite3Impl) AddNotificationInfo(ctx context.Context, notification *Notification) error {
	sql := `
INSERT INTO ` + t.dbName + ` (
  IS_DELETED,
  ID,
  NOTIFICATION_TIME,
  CONTENT,
  TARGET_ID,
  IS_NOTIFICATED,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  REP_NAME
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
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add NOTIFICATION sql %s: %w", notification.ID, err)
		return err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		notification.IsDeleted,
		notification.ID,
		notification.NotificationTime.Format(sqlite3impl.TimeLayout),
		notification.Content,
		notification.TargetID,
		notification.IsNotificated,
		notification.CreateTime.Format(sqlite3impl.TimeLayout),
		notification.CreateApp,
		notification.CreateDevice,
		notification.CreateUser,
		notification.UpdateTime.Format(sqlite3impl.TimeLayout),
		notification.UpdateApp,
		notification.UpdateDevice,
		notification.UpdateUser,
		notification.RepName,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to NOTIFICATION %s: %w", notification.ID, err)
		return err
	}
	return nil
}
