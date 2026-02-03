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

type notificationTempRepositorySQLite3Impl notificationRepositorySQLite3Impl

func NewNotificationTempRepositorySQLite3Impl(ctx context.Context, db *sql.DB, m *sync.Mutex) (NotificationTempRepository, error) {
	filename := "notification_temp"
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
  UPDATE_USER NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  TX_ID NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_NOTIFICATION ON NOTIFICATION (ID, UPDATE_TIME);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION index to %s: %w", filename, err)
		return nil, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table to %s: %w", filename, err)
		return nil, err
	}

	return &notificationTempRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        m,
	}, nil
}
func (t *notificationTempRepositorySQLite3Impl) FindNotifications(ctx context.Context, query *find.FindQuery) ([]*Notification, error) {
	impl := notificationRepositorySQLite3Impl(*t)
	return impl.FindNotifications(ctx, query)
}

func (t *notificationTempRepositorySQLite3Impl) Close(ctx context.Context) error {
	impl := notificationRepositorySQLite3Impl(*t)
	return impl.Close(ctx)
}

func (t *notificationTempRepositorySQLite3Impl) GetNotification(ctx context.Context, id string, updateTime *time.Time) (*Notification, error) {
	impl := notificationRepositorySQLite3Impl(*t)
	return impl.GetNotification(ctx, id, updateTime)
}

func (t *notificationTempRepositorySQLite3Impl) GetNotificationsByTargetID(ctx context.Context, target_id string) ([]*Notification, error) {
	impl := notificationRepositorySQLite3Impl(*t)
	return impl.GetNotificationsByTargetID(ctx, target_id)
}

func (t *notificationTempRepositorySQLite3Impl) GetNotificationsBetweenNotificationTime(ctx context.Context, startTime time.Time, endTime time.Time) ([]*Notification, error) {
	impl := notificationRepositorySQLite3Impl(*t)
	return impl.GetNotificationsBetweenNotificationTime(ctx, startTime, endTime)
}

func (t *notificationTempRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	impl := notificationRepositorySQLite3Impl(*t)
	return impl.UpdateCache(ctx)
}

func (t *notificationTempRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (t *notificationTempRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "notification_temp", nil
}

func (t *notificationTempRepositorySQLite3Impl) GetNotificationHistories(ctx context.Context, id string) ([]*Notification, error) {
	impl := notificationRepositorySQLite3Impl(*t)
	return impl.GetNotificationHistories(ctx, id)
}

func (t *notificationTempRepositorySQLite3Impl) AddNotificationInfo(ctx context.Context, notification *Notification, txID string, userID string, device string) error {
	t.m.Lock()
	defer t.m.Unlock()
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
  ?,
  ?
)`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
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
		userID,
		device,
		txID,
	}
	gkill_log.TraceSQL.Printf("sql: %s params: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at insert in to NOTIFICATION %s: %w", notification.ID, err)
		return err
	}
	return nil
}

func (t *notificationTempRepositorySQLite3Impl) GetNotificationsByTXID(ctx context.Context, txID string, userID string, device string) ([]*Notification, error) {
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
  ? AS REP_NAME,
  ? AS DATA_TYPE
FROM NOTIFICATION
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
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
		txID,
		userID,
		device,
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get notification by tx id sql: %w", err)
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

func (t *notificationTempRepositorySQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	sql := `
DELETE FROM NOTIFICATION
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := t.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete temp notification by TXID sql: %w", err)
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
		err = fmt.Errorf("error at delete temp notification by TXID sql: %w", err)
		return err
	}
	return nil
}

func (t *notificationTempRepositorySQLite3Impl) UnWrapTyped() ([]NotificationTempRepository, error) {
	return []NotificationTempRepository{t}, nil
}
