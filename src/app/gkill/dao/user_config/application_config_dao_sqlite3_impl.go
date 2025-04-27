package user_config

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type applicationConfigDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewApplicationConfigDAOSQLite3Impl(ctx context.Context, filename string) (ApplicationConfigDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=2&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "APPLICATION_CONFIG" (
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  USE_DARK_THEME NOT NULL,
  GOOGLE_MAP_API_KEY NOT NULL,
  RYKV_IMAGE_LIST_COLUMN_NUMBER NOT NULL,
  RYKV_HOT_RELOAD NOT NULL,
  MI_DEFAULT_BOARD NOT NULL,
  RYKV_DEFAULT_PERIOD NOT NULL,
  MI_DEFAULT_PERIOD NOT NULL,
  PRIMARY KEY(USER_ID, DEVICE)
);`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create APPLICATION_CONFIG table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
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
  USE_DARK_THEME,
  GOOGLE_MAP_API_KEY,
  RYKV_IMAGE_LIST_COLUMN_NUMBER,
  RYKV_HOT_RELOAD,
  MI_DEFAULT_BOARD,
  RYKV_DEFAULT_PERIOD,
  MI_DEFAULT_PERIOD
FROM APPLICATION_CONFIG
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all application configs sql: %w", err)
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

	applicationConfigs := []*ApplicationConfig{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			applicationConfig := &ApplicationConfig{}
			rykvDefaultPeriod := -1
			miDefaultPeriod := -1

			err = rows.Scan(
				&applicationConfig.UserID,
				&applicationConfig.Device,
				&applicationConfig.UseDarkTheme,
				&applicationConfig.GoogleMapAPIKey,
				&applicationConfig.RykvImageListColumnNumber,
				&applicationConfig.RykvHotReload,
				&applicationConfig.MiDefaultBoard,
				&rykvDefaultPeriod,
				&miDefaultPeriod,
			)

			applicationConfig.RykvDefaultPeriod = json.Number(strconv.Itoa(rykvDefaultPeriod))
			applicationConfig.MiDefaultPeriod = json.Number(strconv.Itoa(miDefaultPeriod))

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
  USE_DARK_THEME,
  GOOGLE_MAP_API_KEY,
  RYKV_IMAGE_LIST_COLUMN_NUMBER,
  RYKV_HOT_RELOAD,
  MI_DEFAULT_BOARD,
  RYKV_DEFAULT_PERIOD,
  MI_DEFAULT_PERIOD
FROM APPLICATION_CONFIG
WHERE USER_ID = ? AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get application config sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
		device,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	applicationConfigs := []*ApplicationConfig{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			applicationConfig := &ApplicationConfig{}

			rykvDefaultPeriod := -1
			miDefaultPeriod := -1

			err = rows.Scan(
				&applicationConfig.UserID,
				&applicationConfig.Device,
				&applicationConfig.UseDarkTheme,
				&applicationConfig.GoogleMapAPIKey,
				&applicationConfig.RykvImageListColumnNumber,
				&applicationConfig.RykvHotReload,
				&applicationConfig.MiDefaultBoard,
				&rykvDefaultPeriod,
				&miDefaultPeriod,
			)

			applicationConfig.RykvDefaultPeriod = json.Number(strconv.Itoa(rykvDefaultPeriod))
			applicationConfig.MiDefaultPeriod = json.Number(strconv.Itoa(miDefaultPeriod))

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
  USE_DARK_THEME,
  GOOGLE_MAP_API_KEY,
  RYKV_IMAGE_LIST_COLUMN_NUMBER,
  RYKV_HOT_RELOAD,
  MI_DEFAULT_BOARD,
  RYKV_DEFAULT_PERIOD,
  MI_DEFAULT_PERIOD
) VALUES (
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
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add application config sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		applicationConfig.UserID,
		applicationConfig.Device,
		applicationConfig.UseDarkTheme,
		applicationConfig.GoogleMapAPIKey,
		applicationConfig.RykvImageListColumnNumber,
		applicationConfig.RykvHotReload,
		applicationConfig.MiDefaultBoard,
		applicationConfig.RykvDefaultPeriod.String(),
		applicationConfig.MiDefaultPeriod.String(),
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

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
  USE_DARK_THEME = ?,
  GOOGLE_MAP_API_KEY = ?,
  RYKV_IMAGE_LIST_COLUMN_NUMBER = ?,
  RYKV_HOT_RELOAD = ?,
  MI_DEFAULT_BOARD = ?,
  RYKV_DEFAULT_PERIOD = ?,
  MI_DEFAULT_PERIOD = ?
WHERE USER_ID = ? AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update application config sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		applicationConfig.UserID,
		applicationConfig.Device,
		applicationConfig.UseDarkTheme,
		applicationConfig.GoogleMapAPIKey,
		applicationConfig.RykvImageListColumnNumber,
		applicationConfig.RykvHotReload,
		applicationConfig.MiDefaultBoard,
		applicationConfig.RykvDefaultPeriod.String(),
		applicationConfig.MiDefaultPeriod.String(),
		applicationConfig.UserID,
		applicationConfig.Device,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (a *applicationConfigDAOSQLite3Impl) DeleteApplicationConfig(ctx context.Context, userID string, device string) (bool, error) {
	sql := `
DELETE FROM APPLICATION_CONFIG 
WHERE USER_ID = ? AND DEVICE = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete application config sql: %w", err)
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

func (a *applicationConfigDAOSQLite3Impl) Close(ctx context.Context) error {
	return a.db.Close()
}
