package reps

import (
	"context"
	sqllib "database/sql"
	"fmt"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	_ "modernc.org/sqlite"
)

type notificationRepositoryCachedSQLite3Impl struct {
	dbName                  string
	notificationRep         NotificationRepository
	cachedDB                *sqllib.DB
	m                       *sync.RWMutex
	addNotificationInfoSQL  string
	addNotificationInfoStmt *sqllib.Stmt
}

func NewNotificationRepositoryCachedSQLite3Impl(ctx context.Context, notificationRep NotificationRepository, cacheDB *sqllib.DB, m *sync.RWMutex, dbName string) (NotificationRepository, error) {
	if m == nil {
		m = &sync.RWMutex{}
	}
	var err error
	sql := `
CREATE TABLE IF NOT EXISTS ` + sqlite3impl.QuoteIdent(dbName) + ` (
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
	gkill_log.LogSQL(ctx, sql)
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

	gkill_log.LogSQL(ctx, sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create NOTIFICATION table to %s: %w", dbName, err)
		return nil, err
	}

	indexUnixSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_UNIX") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(ID, UPDATE_TIME_UNIX);`
	gkill_log.LogSQL(ctx, indexUnixSQL)
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

	gkill_log.LogSQL(ctx, indexUnixSQL)
	_, err = indexUnixStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create notification index unix to %s: %w", dbName, err)
		return nil, err
	}

	// 対象IDでの取得(GetNotificationsByTargetID)は Kyou 1件ごとに呼ばれるのに、
	// TARGET_ID に索引が無いので全表走査になっていた。
	// SQLが出すのは `((TARGET_ID) = ? OR (ID) = ?)` なので、TARGET_ID 側に索引があれば
	// SQLiteの2索引OR最適化が使える(ID側は既に上の索引で引ける)。
	// Tag / Text は最初からこの形を持っている(tag_repository_cached_sqlite3_impl.go)。
	indexTargetIDSQL := `CREATE INDEX IF NOT EXISTS ` + sqlite3impl.QuoteIdent("INDEX_"+dbName+"_TARGET_ID_UNIX") + ` ON ` + sqlite3impl.QuoteIdent(dbName) + `(TARGET_ID, UPDATE_TIME_UNIX DESC);`
	gkill_log.LogIndexSQL(ctx, indexTargetIDSQL)
	if _, err := cacheDB.ExecContext(ctx, indexTargetIDSQL); err != nil {
		return nil, fmt.Errorf("error at create notification target id index to %s: %w", dbName, err)
	}

	// 通知一覧は NOTIFICATION_TIME_UNIX で絞る/並べる
	if err := sqlite3impl.EnsureUnixColumnIndex(ctx, cacheDB, dbName, "NOTIFICATION_TIME_UNIX"); err != nil {
		return nil, err
	}

	addNotificationInfoSQL := `
INSERT INTO ` + sqlite3impl.QuoteIdent(dbName) + ` (
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
	gkill_log.LogSQL(ctx, addNotificationInfoSQL)
	addNotificationInfoStmt, err := cacheDB.PrepareContext(ctx, addNotificationInfoSQL)
	if err != nil {
		err = fmt.Errorf("error at add notification info sql: %w", err)
		return nil, err
	}

	return &notificationRepositoryCachedSQLite3Impl{
		dbName:                  dbName,
		cachedDB:                cacheDB,
		notificationRep:         notificationRep,
		m:                       m,
		addNotificationInfoSQL:  addNotificationInfoSQL,
		addNotificationInfoStmt: addNotificationInfoStmt,
	}, nil
}
func (n *notificationRepositoryCachedSQLite3Impl) FindNotifications(ctx context.Context, query *find.FindQuery) ([]Notification, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = n.UpdateCache(ctx)
		if err != nil {
			repName, _ := n.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}

	}
	n.m.RLock()
	defer n.m.RUnlock()

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
FROM ` + sqlite3impl.QuoteIdent(n.dbName) + `
WHERE 
`

	dataType := "notification"
	queryArgs := []any{
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

	onlyLatestData = query.OnlyLatestData
	commonWhereSQL, err := sqlite3impl.GenerateFindSQLCommon(query, tableName, tableNameAlias, &whereCounter, onlyLatestData, relatedTimeColumnName, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return nil, err
	}
	sql += commonWhereSQL

	gkill_log.LogSQL(ctx, sql)
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

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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

	notifications := []Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := Notification{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return notifications, nil
}

func (n *notificationRepositoryCachedSQLite3Impl) Close(ctx context.Context) error {
	n.m.Lock()
	defer n.m.Unlock()
	if n.addNotificationInfoStmt != nil {
		n.addNotificationInfoStmt.Close()
	}
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
		_, err = n.cachedDB.ExecContext(ctx, "DROP TABLE IF EXISTS "+sqlite3impl.QuoteIdent(n.dbName))
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *notificationRepositoryCachedSQLite3Impl) GetNotification(ctx context.Context, id string, updateTime *time.Time) (*Notification, error) {
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
FROM ` + sqlite3impl.QuoteIdent(n.dbName) + `
WHERE 
`

	// キャッシュ側のSELECTは REP_NAME を列からそのまま読むので、
	// プレースホルダは `? AS DATA_TYPE` の1つだけ。
	// ここに repName を積むとバインドが1つずれて、
	// DATA_TYPEにrepNameが、WHEREのIDにdataTypeが入り、必ず0件になる。
	dataType := "notification"

	ids := []string{id}
	query := &find.FindQuery{
		IDs:            ids,
		OnlyLatestData: updateTime == nil,
		UpdateTime:     updateTime,
	}
	queryArgs := []any{
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

	gkill_log.LogSQL(ctx, sql)
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

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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

	notifications := []Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := Notification{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(notifications) == 0 {
		return nil, nil
	}
	// ORDER BYを付けていないので、UpdateTimeが最新の行を明示的に選ぶ
	latestNotification := slices.MaxFunc(notifications, func(a, b Notification) int {
		return a.UpdateTime.Compare(b.UpdateTime)
	})
	return &latestNotification, nil
}

func (n *notificationRepositoryCachedSQLite3Impl) GetNotificationsByTargetID(ctx context.Context, target_id string) ([]Notification, error) {
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
FROM ` + sqlite3impl.QuoteIdent(n.dbName) + `
WHERE 
`

	dataType := "notification"

	targetIDs := []string{target_id}
	query := &find.FindQuery{
		Words:    targetIDs,
	}
	queryArgs := []any{
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

	gkill_log.LogSQL(ctx, sql)
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

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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

	notifications := []Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := Notification{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return notifications, nil
}

func (n *notificationRepositoryCachedSQLite3Impl) GetNotificationsBetweenNotificationTime(ctx context.Context, startTime time.Time, endTime time.Time) ([]Notification, error) {
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
FROM ` + sqlite3impl.QuoteIdent(n.dbName) + `
WHERE 
`
	sql += " (NOTIFICATION_TIME_UNIX BETWEEN ? AND ?) "

	dataType := "notification"

	query := &find.FindQuery{}
	queryArgs := []any{
		dataType,
		startTime.Unix(),
		endTime.Unix(),
	}

	tableName := n.dbName
	tableNameAlias := n.dbName
	whereCounter := 1
	// 通知スケジューラは各通知の最新版だけを見る前提(IsDeleted/IsNotificatedの判定は最新版で行う)。
	// 以前は全版が返っており、削除済み・時刻変更前の旧版で通知が飛ぶ余地があった
	onlyLatestData := true
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

	gkill_log.LogSQL(ctx, sql)
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

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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

	notifications := []Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := Notification{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return notifications, nil
}

func (n *notificationRepositoryCachedSQLite3Impl) UpdateCache(ctx context.Context) error {
	err := n.notificationRep.UpdateCache(ctx)
	if err != nil {
		return fmt.Errorf("error at update underlying notification rep cache: %w", err)
	}

	// 下層リポジトリに変更がなければフルリビルドをスキップ
	if !n.notificationRep.LastUpdateCacheChanged() {
		return nil
	}

	query := &find.FindQuery{
		UpdateCache:        false,
		OnlyLatestData:     false,
		IncludeDeletedData: true,
	}

	allNotifications, err := n.notificationRep.FindNotifications(ctx, query)
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

	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at update cache", "error", fmt.Sprintf("%q", err))
			}
		}
	}()

	sql := `DELETE FROM ` + sqlite3impl.QuoteIdent(n.dbName)
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
INSERT INTO ` + sqlite3impl.QuoteIdent(n.dbName) + ` (
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

	gkill_log.LogSQL(ctx, sql)
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
			err = ctx.Err()
			return err
		default:
		}
		err = func() error {
			queryArgs := []any{
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
			gkill_log.LogSQLParams(ctx, sql, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at insert in to NOTIFICATION %s: %w", notification.ID, err)
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit transaction for add notifications: %w", err)
		return err
	}
	isCommitted = true
	// ここまで来て初めて「取り込み済み」とみなす。
	// 途中で失敗した場合は基準を進めないので、次回も再構築される。
	commitCacheRebuildIfSupported(n.notificationRep)
	return nil
}

func (n *notificationRepositoryCachedSQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return n.notificationRep.GetPath(ctx, id)
}

func (n *notificationRepositoryCachedSQLite3Impl) LastUpdateCacheChanged() bool {
	return true
}

func (n *notificationRepositoryCachedSQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return n.notificationRep.GetRepName(ctx)
}

func (n *notificationRepositoryCachedSQLite3Impl) GetNotificationHistories(ctx context.Context, id string) ([]Notification, error) {
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
FROM ` + sqlite3impl.QuoteIdent(n.dbName) + `
WHERE 
`

	// プレースホルダは `? AS DATA_TYPE` の1つだけ。GetNotification と同じ理由で
	// repName を積んではいけない（積むとバインドがずれて必ず0件になる）。
	dataType := "notification"

	ids := []string{id}
	query := &find.FindQuery{
		IDs:    ids,
	}
	queryArgs := []any{
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

	gkill_log.LogSQL(ctx, sql)
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

	gkill_log.LogSQLParams(ctx, sql, queryArgs)
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

	notifications := []Notification{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			notification := Notification{}
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
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return notifications, nil
}
func (n *notificationRepositoryCachedSQLite3Impl) AddNotificationInfo(ctx context.Context, notification Notification) error {
	n.m.Lock()
	defer n.m.Unlock()
	queryArgs := []any{
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
	gkill_log.LogSQLParams(ctx, n.addNotificationInfoSQL, queryArgs)
	_, err := n.addNotificationInfoStmt.ExecContext(ctx, queryArgs...)

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

func (n *notificationRepositoryCachedSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	repName, err := n.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	sql := `
SELECT IS_DELETED, ID AS TARGET_ID, TARGET_ID AS TARGET_ID_IN_DATA,
       ? AS LATEST_DATA_REPOSITORY_NAME, UPDATE_TIME_UNIX AS DATA_UPDATE_TIME_UNIX
FROM ` + sqlite3impl.QuoteIdent(n.dbName) + ` AS T
WHERE T.UPDATE_TIME_UNIX = (SELECT MAX(UPDATE_TIME_UNIX) FROM ` + sqlite3impl.QuoteIdent(n.dbName) + ` AS INNER_TABLE WHERE INNER_TABLE.ID = T.ID)
`
	stmt, err := n.cachedDB.PrepareContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, repName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	latestDataRepositoryAddresses := []gkill_cache.LatestDataRepositoryAddress{}
	for rows.Next() {
		addr := gkill_cache.LatestDataRepositoryAddress{}
		var isDeletedInt int
		var dataUpdateTimeUnix int64
		var targetIDInData *string
		err := rows.Scan(&isDeletedInt, &addr.TargetID, &targetIDInData, &addr.LatestDataRepositoryName, &dataUpdateTimeUnix)
		if err != nil {
			return nil, err
		}
		addr.IsDeleted = isDeletedInt != 0
		addr.DataUpdateTime = time.Unix(dataUpdateTimeUnix, 0)
		addr.TargetIDInData = targetIDInData
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, addr)
	}
	return latestDataRepositoryAddresses, nil
}
