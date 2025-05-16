package server_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type serverConfigDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewServerConfigDAOSQLite3Impl(ctx context.Context, filename string) (ServerConfigDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=1&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "SERVER_CONFIG" (
  DEVICE NOT NULL,
  KEY NOT NULL,
  VALUE,
  PRIMARY KEY(DEVICE, KEY)
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

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_SERVER_CONFIG ON SERVER_CONFIG (DEVICE, KEY);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create SERVER_CONFIG index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create SERVER_CONFIG index to %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	return &serverConfigDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}

var serverConfigDefaultValue = map[string]interface{}{
	"DEVICE":                         "",
	"ENABLE_THIS_DEVICE":             false,
	"IS_LOCAL_ONLY_ACCESS":           true,
	"ADDRESS":                        ":9999",
	"ENABLE_TLS":                     false,
	"TLS_CERT_FILE":                  gkill_options.TLSCertFileDefault,
	"TLS_KEY_FILE":                   gkill_options.TLSKeyFileDefault,
	"OPEN_DIRECTORY_COMMAND":         "explorer /select,$filename",
	"OPEN_FILE_COMMAND":              "rundll32 url.dll,FileProtocolHandler $filename",
	"URLOG_TIMEOUT":                  1 * time.Minute,
	"URLOG_USERAGENT":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36",
	"UPLOAD_SIZE_LIMIT_MONTH":        -1,
	"USER_DATA_DIRECTORY":            gkill_options.DataDirectoryDefault,
	"GKILL_NOTIFICATION_PUBLIC_KEY":  "",
	"GKILL_NOTIFICATION_PRIVATE_KEY": "",
	"USE_GKILL_NOTIFICATION":         true,
	"GOOGLE_MAP_API_KEY":             "",
}

func (s *serverConfigDAOSQLite3Impl) GetAllServerConfigs(ctx context.Context) ([]*ServerConfig, error) {
	sql := fmt.Sprintf(`
SELECT 
  /* ENABLE_THIS_DEVICE */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE %v
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'ENABLE_THIS_DEVICE'
  ) AS ENABLE_THIS_DEVICE,
  /* DEVICE */
  DEVICE AS DEVICE,
  /* IS_LOCAL_ONLY_ACCESS */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE %v
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'IS_LOCAL_ONLY_ACCESS'
  ) AS IS_LOCAL_ONLY_ACCESS,
  /* ADDRESS */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'ADDRESS'
  ) AS ADDRESS,
  /* ENABLE_TLS */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE %v
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'ENABLE_TLS'
  ) AS ENABLE_TLS,
  /* TLS_CERT_FILE */(
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'TLS_CERT_FILE'
  ) AS TLS_CERT_FILE,
  /* TLS_KEY_FILE */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'TLS_KEY_FILE'
  ) AS TLS_KEY_FILE,
  /* OPEN_DIRECTORY_COMMAND */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'OPEN_DIRECTORY_COMMAND'
  ) AS OPEN_DIRECTORY_COMMAND,
  /* OPEN_FILE_COMMAND */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'OPEN_FILE_COMMAND'
  ) AS OPEN_FILE_COMMAND,
  /* URLOG_TIMEOUT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'URLOG_TIMEOUT'
  ) AS URLOG_TIMEOUT,
  /* URLOG_USERAGENT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'URLOG_USERAGENT'
  ) AS URLOG_USERAGENT,
  /* UPLOAD_SIZE_LIMIT_MONTH */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'UPLOAD_SIZE_LIMIT_MONTH'
  ) AS UPLOAD_SIZE_LIMIT_MONTH,
  /* USER_DATA_DIRECTORY */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'USER_DATA_DIRECTORY'
  ) AS USER_DATA_DIRECTORY,
  /* GKILL_NOTIFICATION_PUBLIC_KEY */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'GKILL_NOTIFICATION_PUBLIC_KEY'
  ) AS GKILL_NOTIFICATION_PUBLIC_KEY,
  /* GKILL_NOTIFICATION_PRIVATE_KEY */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'GKILL_NOTIFICATION_PRIVATE_KEY'
  ) AS GKILL_NOTIFICATION_PRIVATE_KEY,
  /* USE_GKILL_NOTIFICATION */ (
      SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE %v
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'USE_GKILL_NOTIFICATION'
  ) AS USE_GKILL_NOTIFICATION,
  /* GOOGLE_MAP_API_KEY */ (
      SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'GOOGLE_MAP_API_KEY'
  ) AS GOOGLE_MAP_API_KEY
FROM SERVER_CONFIG AS GROUPED_SERVER_CONFIG
GROUP BY DEVICE
`,
		serverConfigDefaultValue["ENABLE_THIS_DEVICE"],
		serverConfigDefaultValue["IS_LOCAL_ONLY_ACCESS"],
		serverConfigDefaultValue["ADDRESS"],
		serverConfigDefaultValue["ENABLE_TLS"],
		serverConfigDefaultValue["TLS_CERT_FILE"],
		serverConfigDefaultValue["TLS_KEY_FILE"],
		serverConfigDefaultValue["OPEN_DIRECTORY_COMMAND"],
		serverConfigDefaultValue["OPEN_FILE_COMMAND"],
		serverConfigDefaultValue["URLOG_TIMEOUT"],
		serverConfigDefaultValue["URLOG_USERAGENT"],
		serverConfigDefaultValue["UPLOAD_SIZE_LIMIT_MONTH"],
		serverConfigDefaultValue["USER_DATA_DIRECTORY"],
		serverConfigDefaultValue["GKILL_NOTIFICATION_PUBLIC_KEY"],
		serverConfigDefaultValue["GKILL_NOTIFICATION_PRIVATE_KEY"],
		serverConfigDefaultValue["USE_GKILL_NOTIFICATION"],
		serverConfigDefaultValue["GOOGLE_MAP_API_KEY"],
	)
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
				&serverConfig.UseGkillNotification,
				&serverConfig.GoogleMapAPIKey,
			)
			if err != nil {
				return nil, err
			}
			serverConfigs = append(serverConfigs, serverConfig)
		}
	}
	return serverConfigs, nil
}

func (s *serverConfigDAOSQLite3Impl) GetServerConfig(ctx context.Context, device string) (*ServerConfig, error) {
	sql := fmt.Sprintf(`
SELECT 
  /* ENABLE_THIS_DEVICE */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE %v
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'ENABLE_THIS_DEVICE'
  ) AS ENABLE_THIS_DEVICE,
  /* DEVICE */
  DEVICE AS DEVICE,
  /* IS_LOCAL_ONLY_ACCESS */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE %v
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'IS_LOCAL_ONLY_ACCESS'
  ) AS IS_LOCAL_ONLY_ACCESS,
  /* ADDRESS */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'ADDRESS'
  ) AS ADDRESS,
  /* ENABLE_TLS */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE %v
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'ENABLE_TLS'
  ) AS ENABLE_TLS,
  /* TLS_CERT_FILE */(
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'TLS_CERT_FILE'
  ) AS TLS_CERT_FILE,
  /* TLS_KEY_FILE */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'TLS_KEY_FILE'
  ) AS TLS_KEY_FILE,
  /* OPEN_DIRECTORY_COMMAND */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'OPEN_DIRECTORY_COMMAND'
  ) AS OPEN_DIRECTORY_COMMAND,
  /* OPEN_FILE_COMMAND */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'OPEN_FILE_COMMAND'
  ) AS OPEN_FILE_COMMAND,
  /* URLOG_TIMEOUT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'URLOG_TIMEOUT'
  ) AS URLOG_TIMEOUT,
  /* URLOG_USERAGENT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'URLOG_USERAGENT'
  ) AS URLOG_USERAGENT,
  /* UPLOAD_SIZE_LIMIT_MONTH */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'UPLOAD_SIZE_LIMIT_MONTH'
  ) AS UPLOAD_SIZE_LIMIT_MONTH,
  /* USER_DATA_DIRECTORY */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'USER_DATA_DIRECTORY'
  ) AS USER_DATA_DIRECTORY,
  /* GKILL_NOTIFICATION_PUBLIC_KEY */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'GKILL_NOTIFICATION_PUBLIC_KEY'
  ) AS GKILL_NOTIFICATION_PUBLIC_KEY,
  /* GKILL_NOTIFICATION_PRIVATE_KEY */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'GKILL_NOTIFICATION_PRIVATE_KEY'
  ) AS GKILL_NOTIFICATION_PRIVATE_KEY,
  /* USE_GKILL_NOTIFICATION */ (
      SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE %v
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'USE_GKILL_NOTIFICATION'
  ) AS USE_GKILL_NOTIFICATION,
  /* GOOGLE_MAP_API_KEY */ (
      SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SERVER_CONFIG
	WHERE DEVICE = GROUPED_SERVER_CONFIG.DEVICE
	AND KEY = 'GOOGLE_MAP_API_KEY'
  ) AS GOOGLE_MAP_API_KEY
FROM SERVER_CONFIG AS GROUPED_SERVER_CONFIG
GROUP BY DEVICE
HAVING DEVICE = ?
`,
		serverConfigDefaultValue["ENABLE_THIS_DEVICE"],
		serverConfigDefaultValue["IS_LOCAL_ONLY_ACCESS"],
		serverConfigDefaultValue["ADDRESS"],
		serverConfigDefaultValue["ENABLE_TLS"],
		serverConfigDefaultValue["TLS_CERT_FILE"],
		serverConfigDefaultValue["TLS_KEY_FILE"],
		serverConfigDefaultValue["OPEN_DIRECTORY_COMMAND"],
		serverConfigDefaultValue["OPEN_FILE_COMMAND"],
		serverConfigDefaultValue["URLOG_TIMEOUT"],
		serverConfigDefaultValue["URLOG_USERAGENT"],
		serverConfigDefaultValue["UPLOAD_SIZE_LIMIT_MONTH"],
		serverConfigDefaultValue["USER_DATA_DIRECTORY"],
		serverConfigDefaultValue["GKILL_NOTIFICATION_PUBLIC_KEY"],
		serverConfigDefaultValue["GKILL_NOTIFICATION_PRIVATE_KEY"],
		serverConfigDefaultValue["USE_GKILL_NOTIFICATION"],
		serverConfigDefaultValue["GOOGLE_MAP_API_KEY"],
	)
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
				&serverConfig.UseGkillNotification,
				&serverConfig.GoogleMapAPIKey,
			)
			serverConfigs = append(serverConfigs, serverConfig)
		}
	}
	if len(serverConfigs) == 0 {
		return nil, nil
	} else if len(serverConfigs) == 1 {
		return serverConfigs[0], nil
	}
	return nil, fmt.Errorf("複数のサーバコンフィグが見つかりました。: %w", err)
}

func (s *serverConfigDAOSQLite3Impl) AddServerConfig(ctx context.Context, serverConfig *ServerConfig) (bool, error) {
	sql := `
INSERT INTO SERVER_CONFIG (
  DEVICE,
  KEY,
  VALUE
) VALUES (
  ?,
  ?,
  ?
)
`
	tx, err := s.db.Begin()
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	insertValuesMap := map[string]interface{}{
		"ENABLE_THIS_DEVICE":             serverConfig.EnableThisDevice,
		"IS_LOCAL_ONLY_ACCESS":           serverConfig.IsLocalOnlyAccess,
		"ADDRESS":                        serverConfig.Address,
		"ENABLE_TLS":                     serverConfig.EnableTLS,
		"TLS_CERT_FILE":                  serverConfig.TLSCertFile,
		"TLS_KEY_FILE":                   serverConfig.TLSKeyFile,
		"OPEN_DIRECTORY_COMMAND":         serverConfig.OpenDirectoryCommand,
		"OPEN_FILE_COMMAND":              serverConfig.OpenFileCommand,
		"URLOG_TIMEOUT":                  serverConfig.URLogTimeout,
		"URLOG_USERAGENT":                serverConfig.URLogUserAgent,
		"UPLOAD_SIZE_LIMIT_MONTH":        serverConfig.UploadSizeLimitMonth,
		"USER_DATA_DIRECTORY":            serverConfig.UserDataDirectory,
		"GKILL_NOTIFICATION_PUBLIC_KEY":  serverConfig.GkillNotificationPublicKey,
		"GKILL_NOTIFICATION_PRIVATE_KEY": serverConfig.GkillNotificationPrivateKey,
		"USE_GKILL_NOTIFICATION":         serverConfig.UseGkillNotification,
		"GOOGLE_MAP_API_KEY":             serverConfig.GoogleMapAPIKey,
	}

	for key, value := range insertValuesMap {
		gkill_log.TraceSQL.Printf("sql: %s", sql)
		stmt, err := s.db.PrepareContext(ctx, sql)
		if err != nil {
			err = fmt.Errorf("error at add server config sql: %w", err)
			err = fmt.Errorf("error at query :%w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

		queryArgs := []interface{}{
			serverConfig.Device,
			key,
			value,
		}
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
		_, err = stmt.ExecContext(ctx, queryArgs...)

		if err != nil {
			err = fmt.Errorf("error at add server config sql: %w", err)
			err = fmt.Errorf("error at query :%w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
	}
	return true, nil
}

func (s *serverConfigDAOSQLite3Impl) UpdateServerConfigs(ctx context.Context, serverConfigs []*ServerConfig) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	for _, serverConfig := range serverConfigs {
		sql := `
UPDATE SERVER_CONFIG SET
VALUE = ?
WHERE DEVICE = ?
AND KEY = ?
`
		checkExistSQL := `
SELECT COUNT(*)
FROM SERVER_CONFIG
WHERE DEVICE = ?
AND KEY = ?
`
		insertSQL := `
INSERT INTO SERVER_CONFIG (
  DEVICE,
  KEY,
  VALUE
) VALUES (
  ?,
  ?,
  ?
)
`
		updateValuesMap := map[string]interface{}{
			"ENABLE_THIS_DEVICE":             serverConfig.EnableThisDevice,
			"IS_LOCAL_ONLY_ACCESS":           serverConfig.IsLocalOnlyAccess,
			"ADDRESS":                        serverConfig.Address,
			"ENABLE_TLS":                     serverConfig.EnableTLS,
			"TLS_CERT_FILE":                  serverConfig.TLSCertFile,
			"TLS_KEY_FILE":                   serverConfig.TLSKeyFile,
			"OPEN_DIRECTORY_COMMAND":         serverConfig.OpenDirectoryCommand,
			"OPEN_FILE_COMMAND":              serverConfig.OpenFileCommand,
			"URLOG_TIMEOUT":                  serverConfig.URLogTimeout,
			"URLOG_USERAGENT":                serverConfig.URLogUserAgent,
			"UPLOAD_SIZE_LIMIT_MONTH":        serverConfig.UploadSizeLimitMonth,
			"USER_DATA_DIRECTORY":            serverConfig.UserDataDirectory,
			"GKILL_NOTIFICATION_PUBLIC_KEY":  serverConfig.GkillNotificationPublicKey,
			"GKILL_NOTIFICATION_PRIVATE_KEY": serverConfig.GkillNotificationPrivateKey,
			"USE_GKILL_NOTIFICATION":         serverConfig.UseGkillNotification,
			"GOOGLE_MAP_API_KEY":             serverConfig.GoogleMapAPIKey,
		}

		// レコード自体が存在しなかったらいれる
		for key, value := range updateValuesMap {
			gkill_log.TraceSQL.Printf("sql: %s", sql)
			stmt, err := tx.PrepareContext(ctx, checkExistSQL)
			if err != nil {
				err = fmt.Errorf("error at pre get server config sql: %w", err)
				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					err = fmt.Errorf("%w: %w", err, rollbackErr)
				}
				return false, err
			}
			defer stmt.Close()

			queryArgs := []interface{}{
				serverConfig.Device,
				key,
			}
			gkill_log.TraceSQL.Printf("sql: %s query: %#v", checkExistSQL, queryArgs)
			row := stmt.QueryRowContext(ctx, queryArgs...)
			err = row.Err()
			if err != nil {
				if err != nil {
					err = fmt.Errorf("error at query :%w", err)
					rollbackErr := tx.Rollback()
					if rollbackErr != nil {
						err = fmt.Errorf("%w: %w", err, rollbackErr)
					}
					return false, err
				}
			}

			recordCount := 0
			err = row.Scan(&recordCount)
			if err != nil {
				err = fmt.Errorf("error at scan:%w", err)
				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					err = fmt.Errorf("%w: %w", err, rollbackErr)
				}
				return false, err
			}
			if recordCount == 0 {
				gkill_log.TraceSQL.Printf("sql: %s", insertSQL)
				stmt, err := tx.PrepareContext(ctx, insertSQL)
				if err != nil {
					err = fmt.Errorf("error at add server config sql: %w", err)
					err = fmt.Errorf("error at query :%w", err)
					rollbackErr := tx.Rollback()
					if rollbackErr != nil {
						err = fmt.Errorf("%w: %w", err, rollbackErr)
					}
					return false, err
				}
				defer stmt.Close()

				queryArgs := []interface{}{
					serverConfig.Device,
					key,
					value,
				}
				gkill_log.TraceSQL.Printf("sql: %s query: %#v", insertSQL, queryArgs)
				_, err = stmt.ExecContext(ctx, queryArgs...)

				if err != nil {
					err = fmt.Errorf("error at add server config sql: %w", err)
					err = fmt.Errorf("error at query :%w", err)
					rollbackErr := tx.Rollback()
					if rollbackErr != nil {
						err = fmt.Errorf("%w: %w", err, rollbackErr)
					}
					return false, err
				}
			}
		}

		// 更新する
		for key, value := range updateValuesMap {
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
				value,
				serverConfig.Device,
				key,
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
	}
	// 有効なDEVICEが存在しなければエラーで戻す
	checkEnableDeviceCountSQL := `
SELECT COUNT(*) AS COUNT
FROM SERVER_CONFIG
WHERE KEY = 'ENABLE_THIS_DEVICE'
AND VALUE = ?
GROUP BY DEVICE
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
			if err != nil {
				gkill_log.Debug.Println(err.Error())
				break
			}
			enableDeviceCount += enableCount
		}
	}
	if enableDeviceCount != 1 {
		errAtRollBack := tx.Rollback()
		err = fmt.Errorf("enable device count is not 1")
		if errAtRollBack != nil {
			err = fmt.Errorf("%w: %w", err, errAtRollBack)
			err = fmt.Errorf("error at commit: %w", err)
		}
		return false, err
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		errAtRollBack := tx.Rollback()
		if errAtRollBack != nil {
			err = fmt.Errorf("%w: %w", err, errAtRollBack)
			err = fmt.Errorf("error at commit: %w", err)
			return false, err
		}
		return false, err
	}
	return true, nil
}

func (s *serverConfigDAOSQLite3Impl) UpdateServerConfig(ctx context.Context, serverConfig *ServerConfig) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	sql := `
UPDATE SERVER_CONFIG SET
VALUE = ?
WHERE DEVICE = ?
AND KEY = ?
`
	checkExistSQL := `
SELECT COUNT(*)
FROM SERVER_CONFIG
WHERE DEVICE = ?
AND KEY = ?
`
	insertSQL := `
INSERT INTO SERVER_CONFIG (
  DEVICE,
  KEY,
  VALUE
) VALUES (
  ?,
  ?,
  ?
)
`
	updateValuesMap := map[string]interface{}{
		"ENABLE_THIS_DEVICE":             serverConfig.EnableThisDevice,
		"IS_LOCAL_ONLY_ACCESS":           serverConfig.IsLocalOnlyAccess,
		"ADDRESS":                        serverConfig.Address,
		"ENABLE_TLS":                     serverConfig.EnableTLS,
		"TLS_CERT_FILE":                  serverConfig.TLSCertFile,
		"TLS_KEY_FILE":                   serverConfig.TLSKeyFile,
		"OPEN_DIRECTORY_COMMAND":         serverConfig.OpenDirectoryCommand,
		"OPEN_FILE_COMMAND":              serverConfig.OpenFileCommand,
		"URLOG_TIMEOUT":                  serverConfig.URLogTimeout,
		"URLOG_USERAGENT":                serverConfig.URLogUserAgent,
		"UPLOAD_SIZE_LIMIT_MONTH":        serverConfig.UploadSizeLimitMonth,
		"USER_DATA_DIRECTORY":            serverConfig.UserDataDirectory,
		"GKILL_NOTIFICATION_PUBLIC_KEY":  serverConfig.GkillNotificationPublicKey,
		"GKILL_NOTIFICATION_PRIVATE_KEY": serverConfig.GkillNotificationPrivateKey,
		"USE_GKILL_NOTIFICATION":         serverConfig.UseGkillNotification,
		"GOOGLE_MAP_API_KEY":             serverConfig.GoogleMapAPIKey,
	}

	// レコード自体が存在しなかったらいれる
	for key, value := range updateValuesMap {
		gkill_log.TraceSQL.Printf("sql: %s", sql)
		stmt, err := tx.PrepareContext(ctx, checkExistSQL)
		if err != nil {
			err = fmt.Errorf("error at pre get server config sql: %w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

		queryArgs := []interface{}{
			serverConfig.Device,
			key,
		}
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", checkExistSQL, queryArgs)
		row := stmt.QueryRowContext(ctx, queryArgs...)
		err = row.Err()
		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}

		recordCount := 0
		err = row.Scan(&recordCount)
		if err != nil {
			if err != nil {
				err = fmt.Errorf("error at scan:%w", err)
				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					err = fmt.Errorf("%w: %w", err, rollbackErr)
				}
				return false, err
			}
		}
		if recordCount == 0 {
			gkill_log.TraceSQL.Printf("sql: %s", insertSQL)
			stmt, err := tx.PrepareContext(ctx, insertSQL)
			if err != nil {
				err = fmt.Errorf("error at add server config sql: %w", err)
				err = fmt.Errorf("error at query :%w", err)
				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					err = fmt.Errorf("%w: %w", err, rollbackErr)
				}
				return false, err
			}
			defer stmt.Close()

			queryArgs := []interface{}{
				serverConfig.Device,
				key,
				value,
			}
			gkill_log.TraceSQL.Printf("sql: %s query: %#v", insertSQL, queryArgs)
			_, err = stmt.ExecContext(ctx, queryArgs...)

			if err != nil {
				err = fmt.Errorf("error at add server config sql: %w", err)
				err = fmt.Errorf("error at query :%w", err)
				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					err = fmt.Errorf("%w: %w", err, rollbackErr)
				}
				return false, err
			}
		}
	}

	for key, value := range updateValuesMap {
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
			value,
			serverConfig.Device,
			key,
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
WHERE KEY = 'ENABLE_THIS_DEVICE'
AND VALUE = ?
GROUP BY DEVICE
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

	queryArgs := []interface{}{
		true,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
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
			if err != nil {
				gkill_log.Debug.Println(err.Error())
				break
			}
			enableDeviceCount += enableCount
		}
	}
	if enableDeviceCount != 1 {
		errAtRollBack := tx.Rollback()
		err = fmt.Errorf("enable device count is not 1")
		if errAtRollBack != nil {
			err = fmt.Errorf("%w: %w", err, errAtRollBack)
			err = fmt.Errorf("error at commit: %w", err)
		}
		return false, err
	}
	err = tx.Commit()
	if err != nil {
		errAtRollBack := tx.Rollback()
		if errAtRollBack != nil {
			err = fmt.Errorf("%w: %w", err, errAtRollBack)
			err = fmt.Errorf("error at commit: %w", err)
			return false, err
		}

		err = fmt.Errorf("error at commit: %w", err)
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
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}

	// 全レコード削除
	deleteSQL := `
DELETE FROM SERVER_CONFIG 
`
	gkill_log.TraceSQL.Printf("sql: %s", deleteSQL)

	stmt, err := tx.PrepareContext(ctx, deleteSQL)
	if err != nil {
		err = fmt.Errorf("error at delete server config sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", deleteSQL)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	// 渡された値を登録
	insertSQL := `
INSERT INTO SERVER_CONFIG (
  DEVICE,
  KEY,
  VALUE
) VALUES (
  ?,
  ?,
  ?
)
`

	for _, serverConfig := range serverConfigs {
		insertValuesMap := map[string]interface{}{
			"ENABLE_THIS_DEVICE":             serverConfig.EnableThisDevice,
			"IS_LOCAL_ONLY_ACCESS":           serverConfig.IsLocalOnlyAccess,
			"ADDRESS":                        serverConfig.Address,
			"ENABLE_TLS":                     serverConfig.EnableTLS,
			"TLS_CERT_FILE":                  serverConfig.TLSCertFile,
			"TLS_KEY_FILE":                   serverConfig.TLSKeyFile,
			"OPEN_DIRECTORY_COMMAND":         serverConfig.OpenDirectoryCommand,
			"OPEN_FILE_COMMAND":              serverConfig.OpenFileCommand,
			"URLOG_TIMEOUT":                  serverConfig.URLogTimeout,
			"URLOG_USERAGENT":                serverConfig.URLogUserAgent,
			"UPLOAD_SIZE_LIMIT_MONTH":        serverConfig.UploadSizeLimitMonth,
			"USER_DATA_DIRECTORY":            serverConfig.UserDataDirectory,
			"GKILL_NOTIFICATION_PUBLIC_KEY":  serverConfig.GkillNotificationPublicKey,
			"GKILL_NOTIFICATION_PRIVATE_KEY": serverConfig.GkillNotificationPrivateKey,
			"USE_GKILL_NOTIFICATION":         serverConfig.UseGkillNotification,
			"GOOGLE_MAP_API_KEY":             serverConfig.GoogleMapAPIKey,
		}

		for key, value := range insertValuesMap {
			queryArgs := []interface{}{
				serverConfig.Device,
				key,
				value,
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
	}

	// 有効なDEVICEが存在しなければエラーで戻す
	checkEnableDeviceCountSQL := `
SELECT COUNT(*) AS COUNT
FROM SERVER_CONFIG
WHERE KEY = 'ENABLE_THIS_DEVICE'
AND VALUE = ?
GROUP BY DEVICE
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
			if err != nil {
				err = fmt.Errorf("error at query :%w", err)
				return false, err
			}
			enableDeviceCount += enableCount
		}
	}
	if enableDeviceCount != 1 {
		errAtRollBack := tx.Rollback()
		err = fmt.Errorf("enable device count is not 1")
		if errAtRollBack != nil {
			err = fmt.Errorf("%w: %w", err, errAtRollBack)
			err = fmt.Errorf("error at commit: %w", err)
		}
		return false, err
	}
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		errAtRollBack := tx.Rollback()
		if errAtRollBack != nil {
			err = fmt.Errorf("%w: %w", err, errAtRollBack)
			err = fmt.Errorf("error at commit: %w", err)
			return false, err
		}
		return false, err
	}
	return true, nil
}

func (s *serverConfigDAOSQLite3Impl) Close(ctx context.Context) error {
	return s.db.Close()
}
