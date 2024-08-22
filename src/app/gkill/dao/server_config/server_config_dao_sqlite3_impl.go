// ˅
package server_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// ˄

type serverConfigDAOSQLite3Impl struct {
	// ˅
	filename string
	db       *sql.DB
	m        *sync.Mutex
	// ˄
}

// ˅
func NewServerConfigDAOSQLite3Impl(ctx context.Context, filename string) (ServerConfigDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "SERVER_CONFIG" (
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

	return &serverConfigDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (s *serverConfigDAOSQLite3Impl) GetAllServerConfigs(ctx context.Context) ([]*ServerConfig, error) {
	sql := `
SELECT 
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
			serverConfigs = append(serverConfigs, serverConfig)
		}
	}
	return serverConfigs, nil
}

func (s *serverConfigDAOSQLite3Impl) GetServerConfig(ctx context.Context, device string) (*ServerConfig, error) {
	sql := `
SELECT 
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
  ?
)
`
	stmt, err := s.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add server config sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
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

func (s *serverConfigDAOSQLite3Impl) UpdateServerConfig(ctx context.Context, serverConfig *ServerConfig) (bool, error) {
	sql := `
UPDATE SERVER_CONFIG SET
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
	stmt, err := s.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update server config sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
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

// ˄
