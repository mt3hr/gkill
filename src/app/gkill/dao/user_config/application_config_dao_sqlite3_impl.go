// ˅
package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// ˄

type applicationConfigDAOSQLite3Impl struct {
	// ˅
	filename string
	db       *sql.DB
	m        *sync.Mutex
	// ˄
}

// ˅
func NewApplicationConfigDAOSQLite3Impl(ctx context.Context, filename string) (ApplicationConfigDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "APPLICATION_CONFIG" (
  USER_ID PRIMARY KEY NOT NULL,
  DEVICE PRIMARY KEY NOT NULL,
  ENABLE_BROWSER_CACHE NOT NULL,
  GOOGLE_MAP_API_KEY NOT NULL,
  RYKV_IMAGE_LIST_COLUMN_NUMBER NOT NULL,
  RYKV_HOT_RELOAD NOT NULL,
  MI_DEFAULT_BOARD NOT NULL
);`
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create APPLICATION_CONFIG table statement %s: %w", filename, err)
		return nil, err
	}

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create APPLICATION_CONFIG table to %s: %w", filename, err)
		return nil, err
	}

	return &applicationConfigDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (a *applicationConfigDAOSQLite3Impl) GetAllApplicationConfigs(ctx context.Context) ([]*ApplicationConfig, error) {
	sql := `
SELECT 
  USER_ID,
  DEVICE,
  ENABLE_BROWSER_CACHE,
  GOOGLE_MAP_API_KEY,
  RYKV_IMAGE_LIST_COLUMN_NUMBER,
  RYKV_HOT_RELOAD,
  MI_DEFAULT_BOARD
FROM APPLICATION_CONFIG
`
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all application configs sql: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}

	applicationConfigs := []*ApplicationConfig{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			applicationConfig := &ApplicationConfig{}
			err = rows.Scan(
				applicationConfig.UserID,
				applicationConfig.Device,
				applicationConfig.EnableBrowserCache,
				applicationConfig.GoogleMapAPIKey,
				applicationConfig.RykvImageListColumnNumber,
				applicationConfig.RykvHotReload,
				applicationConfig.MiDefaultBoard,
			)
			applicationConfigs = append(applicationConfigs, applicationConfig)
		}
	}
	return applicationConfigs, nil
}

func (a *applicationConfigDAOSQLite3Impl) GetApplicationConfig(ctx context.Context, userID string, device string) (*ApplicationConfig, error) {
	sql := `
SELECT 
  USER_ID,
  DEVICE,
  ENABLE_BROWSER_CACHE,
  GOOGLE_MAP_API_KEY,
  RYKV_IMAGE_LIST_COLUMN_NUMBER,
  RYKV_HOT_RELOAD,
  MI_DEFAULT_BOARD
FROM APPLICATION_CONFIG
WHERE USER_ID = ? AND DEVICE = ?
`
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get application config sql: %w", err)
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, device)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}

	applicationConfigs := []*ApplicationConfig{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			applicationConfig := &ApplicationConfig{}
			err = rows.Scan(
				applicationConfig.UserID,
				applicationConfig.Device,
				applicationConfig.EnableBrowserCache,
				applicationConfig.GoogleMapAPIKey,
				applicationConfig.RykvImageListColumnNumber,
				applicationConfig.RykvHotReload,
				applicationConfig.MiDefaultBoard,
			)
			applicationConfigs = append(applicationConfigs, applicationConfig)
		}
	}
	if len(applicationConfigs) == 0 {
		return nil, nil
	} else if len(applicationConfigs) == 1 {
		return applicationConfigs[0], nil
	}
	return nil, fmt.Errorf("複数のアプリケーションコンフィグが見つかりました。%s: %w", err)
}

func (a *applicationConfigDAOSQLite3Impl) AddApplicationConfig(ctx context.Context, applicationConfig *ApplicationConfig) (bool, error) {
	sql := `
INSERT INTO APPLICATION_CONFIG (
  USER_ID,
  DEVICE,
  ENABLE_BROWSER_CACHE,
  GOOGLE_MAP_API_KEY,
  RYKV_IMAGE_LIST_COLUMN_NUMBER,
  RYKV_HOT_RELOAD,
  MI_DEFAULT_BOARD
)
VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)
`
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add application config sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
		applicationConfig.UserID,
		applicationConfig.Device,
		applicationConfig.EnableBrowserCache,
		applicationConfig.GoogleMapAPIKey,
		applicationConfig.RykvImageListColumnNumber,
		applicationConfig.RykvHotReload,
		applicationConfig.MiDefaultBoard,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (a *applicationConfigDAOSQLite3Impl) UpdateApplicationConfig(ctx context.Context, applicationConfig *ApplicationConfig) (bool, error) {
	sql := `
UPDATE APPLICATION_CONFIG SET
  USER_ID = ?,
  DEVICE = ?,
  ENABLE_BROWSER_CACHE = ?,
  GOOGLE_MAP_API_KEY = ?,
  RYKV_IMAGE_LIST_COLUMN_NUMBER = ?,
  RYKV_HOT_RELOAD = ?,
  MI_DEFAULT_BOARD = ?
WHERE USER_ID = ? AND DEVICE = ?
`
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update application config sql: %w", err)
		return false, err
	}

	_, err = stmt.ExecContext(ctx,
		applicationConfig.UserID,
		applicationConfig.Device,
		applicationConfig.EnableBrowserCache,
		applicationConfig.GoogleMapAPIKey,
		applicationConfig.RykvImageListColumnNumber,
		applicationConfig.RykvHotReload,
		applicationConfig.MiDefaultBoard,
		applicationConfig.UserID,
		applicationConfig.Device,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (a *applicationConfigDAOSQLite3Impl) DeleteApplicationConfig(ctx context.Context, userID string, device string) (bool, error) {
	sql := `
DELETE APPLICATION_CONFIG 
WHERE USER_ID = ? AND DEVICE = ?
`
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete application config sql: %w", err)
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
