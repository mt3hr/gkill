package server_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type serverConfigDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewServerConfigDAOSQLite3Impl(ctx context.Context, filename string) (ServerConfigDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
VACUUM;
CREATE TABLE IF NOT EXISTS "SERVER_CONFIG" (
  ENABLE_THIS_DEVICE NOT NULL,
  DEVICE PRIMARY KEY NOT NULL,
  IS_LOCAL_ONLY_ACCESS NOT NULL,
  ADDRESS NOT NULL,
  ENABLE_TLS NOT NULL,
  TLS_CERT_FILE NOT NULL,
  TLS_KEY_FILE NOT NULL,
  OPEN_DIRECTORY_COMMAND,
  OPEN_FILE_COMMAND,
  URLOG_TIMEOUT NOT NULL,
  URLOG_USERAGENT NOT NULL,
  UPLOAD_SIZE_LIMIT_MONTH,
  USER_DATA_DIRECTORY NOT NULL,
  GKILL_NOTIFICATION_PUBLIC_KEY NOT NULL,
  GKILL_NOTIFICATION_PRIVATE_KEY NOT NULL
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create SERVER_CONFIG table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create SERVER_CONFIG table to %s: %w", filename, err)
		return nil, err
	}

	return &serverConfigDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}

func (s *serverConfigDAOSQLite3Impl) GetAllServerConfigs(ctx context.Context) ([]*ServerConfig, error) {
	sql := `
SELECT 
  ENABLE_THIS_DEVICE,
  DEVICE,
  IS_LOCAL_ONLY_ACCESS,
  ADDRESS,
  ENABLE_TLS,
  TLS_CERT_FILE,
  TLS_KEY_FILE,
  OPEN_DIRECTORY_COMMAND,
  OPEN_FILE_COMMAND,
  URLOG_TIMEOUT,
  URLOG_USERAGENT,
  UPLOAD_SIZE_LIMIT_MONTH,
  USER_DATA_DIRECTORY,
  GKILL_NOTIFICATION_PUBLIC_KEY,
  GKILL_NOTIFICATION_PRIVATE_KEY
FROM SERVER_CONFIG
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := s.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all server configs sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	serverConfigs := []*ServerConfig{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			serverConfig := &ServerConfig{}
			err = rows.Scan(
				&serverConfig.EnableThisDevice,
				&serverConfig.Device,
				&serverConfig.IsLocalOnlyAccess,
				&serverConfig.Address,
				&serverConfig.EnableTLS,
				&serverConfig.TLSCertFile,
				&serverConfig.TLSKeyFile,
				&serverConfig.OpenDirectoryCommand,
				&serverConfig.OpenFileCommand,
				&serverConfig.URLogTimeout,
				&serverConfig.URLogUserAgent,
				&serverConfig.UploadSizeLimitMonth,
				&serverConfig.UserDataDirectory,
				&serverConfig.GkillNotificationPublicKey,
				&serverConfig.GkillNotificationPrivateKey,
			)
			serverConfigs = append(serverConfigs, serverConfig)
		}
	}
	return serverConfigs, nil
}

func (s *serverConfigDAOSQLite3Impl) GetServerConfig(ctx context.Context, device string) (*ServerConfig, error) {
	sql := `
SELECT 
  ENABLE_THIS_DEVICE,
  DEVICE,
  IS_LOCAL_ONLY_ACCESS,
  ADDRESS,
  ENABLE_TLS,
  TLS_CERT_FILE,
  TLS_KEY_FILE,
  OPEN_DIRECTORY_COMMAND,
  OPEN_FILE_COMMAND,
  URLOG_TIMEOUT,
  URLOG_USERAGENT,
  UPLOAD_SIZE_LIMIT_MONTH,
  USER_DATA_DIRECTORY,
  GKILL_NOTIFICATION_PUBLIC_KEY,
  GKILL_NOTIFICATION_PRIVATE_KEY
FROM SERVER_CONFIG
WHERE DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := s.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get server config sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		device,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	serverConfigs := []*ServerConfig{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			serverConfig := &ServerConfig{}
			err = rows.Scan(
				&serverConfig.EnableThisDevice,
				&serverConfig.Device,
				&serverConfig.IsLocalOnlyAccess,
				&serverConfig.Address,
				&serverConfig.EnableTLS,
				&serverConfig.TLSCertFile,
				&serverConfig.TLSKeyFile,
				&serverConfig.OpenDirectoryCommand,
				&serverConfig.OpenFileCommand,
				&serverConfig.URLogTimeout,
				&serverConfig.URLogUserAgent,
				&serverConfig.UploadSizeLimitMonth,
				&serverConfig.UserDataDirectory,
				&serverConfig.GkillNotificationPublicKey,
				&serverConfig.GkillNotificationPrivateKey,
			)
			serverConfigs = append(serverConfigs, serverConfig)
		}
	}
	if len(serverConfigs) == 0 {
		return nil, nil
	} else if len(serverConfigs) == 1 {
		return serverConfigs[0], nil
	}
	return nil, fmt.Errorf("複数のサーバコンフィグが見つかりました。%s: %w", err)
}

func (s *serverConfigDAOSQLite3Impl) AddServerConfig(ctx context.Context, serverConfig *ServerConfig) (bool, error) {
	sql := `
INSERT INTO SERVER_CONFIG (
  ENABLE_THIS_DEVICE,
  DEVICE,
  IS_LOCAL_ONLY_ACCESS,
  ADDRESS,
  ENABLE_TLS,
  TLS_CERT_FILE,
  TLS_KEY_FILE,
  OPEN_DIRECTORY_COMMAND,
  OPEN_FILE_COMMAND,
  URLOG_TIMEOUT,
  URLOG_USERAGENT,
  UPLOAD_SIZE_LIMIT_MONTH,
  USER_DATA_DIRECTORY,
  GKILL_NOTIFICATION_PUBLIC_KEY,
  GKILL_NOTIFICATION_PRIVATE_KEY
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
)
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := s.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add server config sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		serverConfig.EnableThisDevice,
		serverConfig.Device,
		serverConfig.IsLocalOnlyAccess,
		serverConfig.Address,
		serverConfig.EnableTLS,
		serverConfig.TLSCertFile,
		serverConfig.TLSKeyFile,
		serverConfig.OpenDirectoryCommand,
		serverConfig.OpenFileCommand,
		serverConfig.URLogTimeout,
		serverConfig.URLogUserAgent,
		serverConfig.UploadSizeLimitMonth,
		serverConfig.UserDataDirectory,
		serverConfig.GkillNotificationPublicKey,
		serverConfig.GkillNotificationPrivateKey,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (s *serverConfigDAOSQLite3Impl) UpdateServerConfigs(ctx context.Context, serverConfigs []*ServerConfig) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	for _, serverConfig := range serverConfigs {
		sql := `
UPDATE SERVER_CONFIG SET
  ENABLE_THIS_DEVICE = ?,
  DEVICE = ?,
  IS_LOCAL_ONLY_ACCESS = ?,
  ADDRESS = ?,
  ENABLE_TLS = ?,
  TLS_CERT_FILE = ?,
  TLS_KEY_FILE = ?,
  OPEN_DIRECTORY_COMMAND = ?,
  OPEN_FILE_COMMAND = ?,
  URLOG_TIMEOUT = ?,
  URLOG_USERAGENT = ?,
  UPLOAD_SIZE_LIMIT_MONTH = ?,
  USER_DATA_DIRECTORY = ?,
  GKILL_NOTIFICATION_PUBLIC_KEY = ?,
  GKILL_NOTIFICATION_PRIVATE_KEY = ?
WHERE DEVICE = ?
`
		gkill_log.TraceSQL.Printf("sql: %s", sql)
		stmt, err := tx.PrepareContext(ctx, sql)
		if err != nil {
			err = fmt.Errorf("error at update server config sql: %w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

		queryArgs := []interface{}{
			serverConfig.EnableThisDevice,
			serverConfig.Device,
			serverConfig.IsLocalOnlyAccess,
			serverConfig.Address,
			serverConfig.EnableTLS,
			serverConfig.TLSCertFile,
			serverConfig.TLSKeyFile,
			serverConfig.OpenDirectoryCommand,
			serverConfig.OpenFileCommand,
			serverConfig.URLogTimeout,
			serverConfig.URLogUserAgent,
			serverConfig.UploadSizeLimitMonth,
			serverConfig.UserDataDirectory,
			serverConfig.GkillNotificationPublicKey,
			serverConfig.GkillNotificationPrivateKey,
			serverConfig.Device,
		}
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
		_, err = stmt.ExecContext(ctx, queryArgs...)

		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
	}
	// 有効なDEVICEが存在しなければエラーで戻す
	checkEnableDeviceCountSQL := `
SELECT COUNT(*) AS COUNT
FROM SERVER_CONFIG
WHERE ENABLE_THIS_DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", checkEnableDeviceCountSQL)
	checkEnableDeviceStmt, err := tx.PrepareContext(ctx, checkEnableDeviceCountSQL)
	if err != nil {
		err = fmt.Errorf("error at check enable device server config sql: %w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer checkEnableDeviceStmt.Close()

	enableDeviceCount := 0
	queryArgs := []interface{}{
		true,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", checkEnableDeviceCountSQL, queryArgs)
	rows, err := checkEnableDeviceStmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer rows.Close()

	enableDeviceCount = 0
	for rows.Next() {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			enableCount := 0
			err = rows.Scan(
				&enableCount,
			)
			enableDeviceCount += enableCount
		}
	}
	if enableDeviceCount != 1 {
		errAtRollBack := tx.Rollback()
		err := fmt.Errorf("enable device count is not 1.")
		if errAtRollBack != nil {
			err = fmt.Errorf("%w: %w", err, errAtRollBack)
			fmt.Errorf("error at commit: %w", err)
		}
		return false, err
	}
	err = tx.Commit()
	if err != nil {
		fmt.Errorf("error at commit: %w", err)
		errAtRollBack := tx.Rollback()
		if errAtRollBack != nil {
			err = fmt.Errorf("%w: %w", err, errAtRollBack)
			fmt.Errorf("error at commit: %w", err)
			return false, err
		}
		return false, err
	}
	return true, nil
}

func (s *serverConfigDAOSQLite3Impl) UpdateServerConfig(ctx context.Context, serverConfig *ServerConfig) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		fmt.Errorf("error at begin: %w", err)
		return false, err
	}

	sql := `
UPDATE SERVER_CONFIG SET
  ENABLE_THIS_DEVICE = ?,
  DEVICE = ?,
  IS_LOCAL_ONLY_ACCESS = ?,
  ADDRESS = ?,
  ENABLE_TLS = ?,
  TLS_CERT_FILE = ?,
  TLS_KEY_FILE = ?,
  OPEN_DIRECTORY_COMMAND = ?,
  OPEN_FILE_COMMAND = ?,
  URLOG_TIMEOUT = ?,
  URLOG_USERAGENT = ?,
  UPLOAD_SIZE_LIMIT_MONTH = ?,
  USER_DATA_DIRECTORY = ?,
  GKILL_NOTIFICATION_PUBLIC_KEY = ?,
  GKILL_NOTIFICATION_PRIVATE_KEY = ?
WHERE DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update server config sql: %w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		serverConfig.EnableThisDevice,
		serverConfig.Device,
		serverConfig.IsLocalOnlyAccess,
		serverConfig.Address,
		serverConfig.EnableTLS,
		serverConfig.TLSCertFile,
		serverConfig.TLSKeyFile,
		serverConfig.OpenDirectoryCommand,
		serverConfig.OpenFileCommand,
		serverConfig.URLogTimeout,
		serverConfig.URLogUserAgent,
		serverConfig.UploadSizeLimitMonth,
		serverConfig.UserDataDirectory,
		serverConfig.GkillNotificationPublicKey,
		serverConfig.GkillNotificationPrivateKey,
		serverConfig.Device,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer rows.Close()

	// 有効なDEVICEが存在しなければエラーで戻す
	checkEnableDeviceCountSQL := `
SELECT COUNT(*) 
FROM SERVER_CONFIG
WHERE ENABLE_THIS_DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	checkEnableDeviceStmt, err := tx.PrepareContext(ctx, checkEnableDeviceCountSQL)
	if err != nil {
		err = fmt.Errorf("error at check enable device server config sql: %w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer checkEnableDeviceStmt.Close()

	queryArgs = []interface{}{
		true,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err = stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer rows.Close()

	enableDeviceCount := 0
	for rows.Next() {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			enableCount := 0
			err = rows.Scan(
				&enableCount,
			)
			enableDeviceCount += enableCount
		}
	}
	if enableDeviceCount != 1 {
		errAtRollBack := tx.Rollback()
		err := fmt.Errorf("enable device count is not 1.")
		if errAtRollBack != nil {
			err = fmt.Errorf("%w: %w", err, errAtRollBack)
			fmt.Errorf("error at commit: %w", err)
		}
		return false, err
	}
	err = tx.Commit()
	if err != nil {
		errAtRollBack := tx.Rollback()
		if errAtRollBack != nil {
			err = fmt.Errorf("%w: %w", err, errAtRollBack)
			fmt.Errorf("error at commit: %w", err)
			return false, err
		}

		fmt.Errorf("error at commit: %w", err)
		return false, err
	}
	return true, nil
}

func (s *serverConfigDAOSQLite3Impl) DeleteServerConfig(ctx context.Context, device string) (bool, error) {
	sql := `
DELETE FROM SERVER_CONFIG 
WHERE DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := s.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete server config sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		device,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (s *serverConfigDAOSQLite3Impl) DeleteWriteServerConfigs(ctx context.Context, serverConfigs []*ServerConfig) (bool, error) {
	s.m.Lock()
	defer s.m.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		fmt.Errorf("error at begin: %w", err)
		return false, err
	}

	// 全レコード削除
	deleteSQL := `
DELETE FROM SERVER_CONFIG 
`
	queryArgs := []interface{}{}
	gkill_log.TraceSQL.Printf("sql: %s", deleteSQL)

	stmt, err := tx.PrepareContext(ctx, deleteSQL)
	if err != nil {
		err = fmt.Errorf("error at delete server config sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s query: %#v", deleteSQL, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	// 渡された値を登録
	insertSQL := `
INSERT INTO SERVER_CONFIG (
  ENABLE_THIS_DEVICE,
  DEVICE,
  IS_LOCAL_ONLY_ACCESS,
  ADDRESS,
  ENABLE_TLS,
  TLS_CERT_FILE,
  TLS_KEY_FILE,
  OPEN_DIRECTORY_COMMAND,
  OPEN_FILE_COMMAND,
  URLOG_TIMEOUT,
  URLOG_USERAGENT,
  UPLOAD_SIZE_LIMIT_MONTH,
  USER_DATA_DIRECTORY,
  GKILL_NOTIFICATION_PUBLIC_KEY,
  GKILL_NOTIFICATION_PRIVATE_KEY
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
)
`

	for _, serverConfig := range serverConfigs {
		queryArgs := []interface{}{
			serverConfig.EnableThisDevice,
			serverConfig.Device,
			serverConfig.IsLocalOnlyAccess,
			serverConfig.Address,
			serverConfig.EnableTLS,
			serverConfig.TLSCertFile,
			serverConfig.TLSKeyFile,
			serverConfig.OpenDirectoryCommand,
			serverConfig.OpenFileCommand,
			serverConfig.URLogTimeout,
			serverConfig.URLogUserAgent,
			serverConfig.UploadSizeLimitMonth,
			serverConfig.UserDataDirectory,
			serverConfig.GkillNotificationPublicKey,
			serverConfig.GkillNotificationPrivateKey,
		}
		gkill_log.TraceSQL.Printf("sql: %s", insertSQL)

		stmt, err := tx.PrepareContext(ctx, insertSQL)
		if err != nil {
			err = fmt.Errorf("error at add server config sql: %w", err)
			return false, err
		}
		defer stmt.Close()

		gkill_log.TraceSQL.Printf("sql: %s query: %#v", insertSQL, queryArgs)
		_, err = stmt.ExecContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			return false, err
		}
	}

	// 有効なDEVICEが存在しなければエラーで戻す
	checkEnableDeviceCountSQL := `
SELECT COUNT(*) AS COUNT
FROM SERVER_CONFIG
WHERE ENABLE_THIS_DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", checkEnableDeviceCountSQL)
	checkEnableDeviceStmt, err := tx.PrepareContext(ctx, checkEnableDeviceCountSQL)
	if err != nil {
		err = fmt.Errorf("error at check enable device server config sql: %w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer checkEnableDeviceStmt.Close()

	queryArgsCheckEnableDevice := []interface{}{
		true,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", checkEnableDeviceCountSQL, queryArgsCheckEnableDevice)
	rows, err := checkEnableDeviceStmt.QueryContext(ctx, queryArgsCheckEnableDevice...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer rows.Close()

	enableDeviceCount := 0
	for rows.Next() {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			enableCount := 0
			err = rows.Scan(
				&enableCount,
			)
			enableDeviceCount += enableCount
		}
	}
	if enableDeviceCount != 1 {
		errAtRollBack := tx.Rollback()
		err := fmt.Errorf("enable device count is not 1.")
		if errAtRollBack != nil {
			err = fmt.Errorf("%w: %w", err, errAtRollBack)
			fmt.Errorf("error at commit: %w", err)
		}
		return false, err
	}
	err = tx.Commit()
	if err != nil {
		fmt.Errorf("error at commit: %w", err)
		errAtRollBack := tx.Rollback()
		if errAtRollBack != nil {
			err = fmt.Errorf("%w: %w", err, errAtRollBack)
			fmt.Errorf("error at commit: %w", err)
			return false, err
		}
		return false, err
	}
	return true, nil
}

func (s *serverConfigDAOSQLite3Impl) Close(ctx context.Context) error {
	return s.db.Close()
}
