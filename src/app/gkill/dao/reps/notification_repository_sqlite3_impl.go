package reps

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type notificationRepositorySQLite3Impl struct {
	filename    string
	db          *sql.DB
	m           *sync.Mutex
	fullConnect bool
}

func NewNotificationRepositorySQLite3Impl(ctx context.Context, filename string, fullConnect bool) (NotificationRepository, error) {
	db, err := sqlite3impl.GetSQLiteDBConnection(ctx, filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "NOTIFICATION" (
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
  UPDATE_USER NOT NULL 
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_NOTIFICATION ON NOTIFICATION (ID, UPDATE_TIME);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION index to %s: %w", filename, err)
		return nil, err
	}

	if !fullConnect {
		err = db.Close()
		if err != nil {
			return nil, err
		}
		db = nil
	}

	return &notificationRepositorySQLite3Impl{
		filename:    filename,
		db:          db,
		m:           &sync.Mutex{},
		fullConnect: fullConnect,
	}, nil
}
func (t *notificationRepositorySQLite3Impl) FindNotifications(ctx context.Context, query *find.FindQuery) ([]*Notification, error) {
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
	}

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
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM NOTIFICATION
WHERE 
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at notification: %w", err)
		return nil, err
	}
	dataType := "notification"
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "NOTIFICATION"
	tableNameAlias := "NOTIFICATION"
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "UPDATE_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get NOTIFICATION histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
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

func (t *notificationRepositorySQLite3Impl) Close(ctx context.Context) error {
	if t.fullConnect {
		return t.db.Close()
	}
	return nil
}

func (t *notificationRepositorySQLite3Impl) GetNotification(ctx context.Context, id string, updateTime *time.Time) (*Notification, error) {
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

func (t *notificationRepositorySQLite3Impl) GetNotificationsByTargetID(ctx context.Context, target_id string) ([]*Notification, error) {
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
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
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM NOTIFICATION
WHERE 
`

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at notification: %w", err)
		return nil, err
	}

	dataType := "notification"

	trueValue := true
	targetIDs := []string{target_id}
	query := &find.FindQuery{
		UseWords: &trueValue,
		Words:    &targetIDs,
	}
	queryArgs := []interface{}{
		repName,
		dataType,
	}

	tableName := "NOTIFICATION"
	tableNameAlias := "NOTIFICATION"
	whereCounter := 0
	onlyLatestData := true
	relatedTimeColumnName := "UPDATE_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get notification histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
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

func (t *notificationRepositorySQLite3Impl) GetNotificationsBetweenNotificationTime(ctx context.Context, startTime time.Time, endTime time.Time) ([]*Notification, error) {
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
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
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM NOTIFICATION
WHERE 
`
	sql += " (datetime(NOTIFICATION_TIME, 'localtime') BETWEEN datetime(?, 'localtime') AND datetime(?, 'localtime')) "

	repName, err := t.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at text: %w", err)
		return nil, err
	}

	dataType := "notification"

	query := &find.FindQuery{}
	queryArgs := []interface{}{
		repName,
		dataType,
		startTime.Format(sqlite3impl.TimeLayout),
		endTime.Format(sqlite3impl.TimeLayout),
	}

	tableName := "NOTIFICATION"
	tableNameAlias := "NOTIFICATION"
	whereCounter := 1
	onlyLatestData := true
	relatedTimeColumnName := "NOTIFICATION_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get notification between notification time sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
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

func (t *notificationRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (t *notificationRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	if id == "" {
		return t.filename, nil
	}
	return filepath.Abs(t.filename)
}

func (t *notificationRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	path, err := t.GetPath(ctx, "")
	if err != nil {
		err = fmt.Errorf("error at get path notification rep: %w", err)
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt, nil
}

func (t *notificationRepositorySQLite3Impl) GetNotificationHistories(ctx context.Context, id string) ([]*Notification, error) {
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return nil, err
		}
		defer db.Close()
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
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM NOTIFICATION
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

	tableName := "NOTIFICATION"
	tableNameAlias := "NOTIFICATION"
	whereCounter := 0
	onlyLatestData := false
	relatedTimeColumnName := "UPDATE_TIME"
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
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get notification histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
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
func (t *notificationRepositorySQLite3Impl) AddNotificationInfo(ctx context.Context, notification *Notification) error {
	t.m.Lock()
	defer t.m.Unlock()
	var err error
	var db *sql.DB
	if t.fullConnect {
		db = t.db
	} else {
		db, err = sqlite3impl.GetSQLiteDBConnection(ctx, t.filename)
		if err != nil {
			return err
		}
		defer db.Close()
	}
	sql := `
INSERT INTO NOTIFICATION (
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
  ?,
  ?
)`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
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
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to NOTIFICATION %s: %w", notification.ID, err)
		return err
	}
	return nil
}

func (t *notificationRepositorySQLite3Impl) UnWrapTyped() ([]NotificationRepository, error) {
	return []NotificationRepository{t}, nil
}
