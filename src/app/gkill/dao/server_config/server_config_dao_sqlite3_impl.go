package server_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
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
  USER_DATA_DIRECTORY NOT NULL
);`
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create SERVER_CONFIG table statement %s: %w", filename, err)
		return nil, err
	}

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create SERVER_CONFIG table to %s: %w", filename, err)
		return nil, err
	}

	serverConfigDao := &serverConfigDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}

	err = serverConfigDao.insertInitData(ctx)
	if err != nil {
		err = fmt.Errorf("error at insert init data to server config %s: %w", filename, err)
		return nil, err
	}
	return serverConfigDao, nil
}

func (s *serverConfigDAOSQLite3Impl) insertInitData(ctx context.Context) error {
	serverConfigs, err := s.GetAllServerConfigs(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all server configs: %w", err)
		return err
	}
	// データが有れば何もしない
	if len(serverConfigs) != 0 {
		return nil
	}
	// データがなかったら初期データを作ってあげる
	serverConfig := &ServerConfig{
		EnableThisDevice:     true,
		Device:               "gkill",
		IsLocalOnlyAccess:    false,
		Address:              ":9999",
		EnableTLS:            false,
		TLSCertFile:          "",
		TLSKeyFile:           "",
		OpenDirectoryCommand: "explorer /select,$filename",
		OpenFileCommand:      "rundll32 url.dll,FileProtocolHandler $filename",
		URLogTimeout:         1 * time.Minute,
		URLogUserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36",
		UploadSizeLimitMonth: -1,
		UserDataDirectory:    "$HOME/gkill/datas",
	}
	_, err = s.AddServerConfig(ctx, serverConfig)
	if err != nil {
		err = fmt.Errorf("error at add init data to server config db: %w", err)
		return err
	}
	return nil
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
  USER_DATA_DIRECTORY
FROM SERVER_CONFIG
`
	stmt, err := s.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all server configs sql: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}

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
  USER_DATA_DIRECTORY
FROM SERVER_CONFIG
WHERE DEVICE = ?
`
	stmt, err := s.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get server config sql: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, device)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}

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
  USER_DATA_DIRECTORY
)
VALUES (
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
	stmt, err := s.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add server config sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
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
	)
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
  USER_DATA_DIRECTORY = ?
WHERE DEVICE = ?
`
		stmt, err := tx.PrepareContext(ctx, sql)
		if err != nil {
			err = fmt.Errorf("error at update server config sql: %w", err)
			return false, err
		}

		_, err = stmt.ExecContext(ctx,
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
			serverConfig.Device,
		)
		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			return false, err
		}
	}
	// 有効なDEVICEが存在しなければエラーで戻す
	checkEnableDeviceCountSQL := `
SELECT COUNT(*) 
FROM SERVER_CONFIG
WHERE ENABLE_THIS_DEVICE = ?
`
	checkEnableDeviceStmt, err := tx.PrepareContext(ctx, checkEnableDeviceCountSQL)
	if err != nil {
		err = fmt.Errorf("error at check enable device server config sql: %w", err)
		return false, err
	}
	defer checkEnableDeviceStmt.Close()

	enableDeviceCount := 0
	rows, err := checkEnableDeviceStmt.QueryContext(ctx, true)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	for rows.Next() {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			enableThisDevice := false
			err = rows.Scan(
				&enableThisDevice,
			)
			if enableThisDevice {
				enableDeviceCount++
			}
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
  USER_DATA_DIRECTORY = ?
WHERE DEVICE = ?
`
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update server config sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
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
		serverConfig.Device,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	// 有効なDEVICEが存在しなければエラーで戻す
	checkEnableDeviceCountSQL := `
SELECT COUNT(*) 
FROM SERVER_CONFIG
WHERE ENABLE_THIS_DEVICE = ?
`
	checkEnableDeviceStmt, err := tx.PrepareContext(ctx, checkEnableDeviceCountSQL)
	if err != nil {
		err = fmt.Errorf("error at check enable device server config sql: %w", err)
		return false, err
	}
	defer checkEnableDeviceStmt.Close()

	rows, err := checkEnableDeviceStmt.QueryContext(ctx, true)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	enableDeviceCount := 0
	for rows.Next() {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			enableThisDevice := false
			err = rows.Scan(
				&enableThisDevice,
			)
			if enableThisDevice {
				enableDeviceCount++
			}
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
DELETE SERVER_CONFIG 
WHERE DEVICE = ?
`
	stmt, err := s.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete server config sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx, device)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (s *serverConfigDAOSQLite3Impl) Close(ctx context.Context) error {
	return s.db.Close()
}
