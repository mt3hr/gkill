package user_config

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

const CURRENT_SCHEMA_VERSION_APPLICATION_CONFIG_DAO = "1.0.0"

type applicationConfigDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.RWMutex
}

func NewApplicationConfigDAOSQLite3Impl(ctx context.Context, filename string) (ApplicationConfigDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=1&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	if isOld, oldVerDAO, err := checkAndResolveDataSchemaApplicationConfigDAO(ctx, db); err != nil {
		return nil, err
	} else if isOld {
		if oldVerDAO != nil {
			return oldVerDAO, nil
		} else {
			err = fmt.Errorf("error at load database schema %s", filename)
			return nil, err
		}
	}

	if gkill_options.Optimize {
		err = sqlite3impl.DeleteAllIndex(db)
		if err != nil {
			err = fmt.Errorf("error at delete all index %w", err)
			return nil, err
		}
	}

	sql := `
CREATE TABLE IF NOT EXISTS APPLICATION_CONFIG (
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  KEY NOT NULL,
  VALUE,
  PRIMARY KEY(USER_ID, DEVICE, KEY)
);
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create APPLICATION_CONFIG table statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		err = fmt.Errorf("error at create APPLICATION_CONFIG table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_APPLICATION_CONFIG ON APPLICATION_CONFIG (USER_ID, DEVICE, KEY);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create APPLICATION_CONFIG index statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := indexStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create APPLICATION_CONFIG index to %s: %w", filename, err)
		return nil, err
	}

	if gkill_options.Optimize {
		err = sqlite3impl.Optimize(db)
		if err != nil {
			err = fmt.Errorf("error at optimize db %w", err)
			return nil, err
		}
	}

	return &applicationConfigDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.RWMutex{},
	}, nil
}

func GetDefaultApplicationConfig(userID string, device string) *ApplicationConfig {
	return &ApplicationConfig{
		UserID:                    userID,
		Device:                    device,
		UseDarkTheme:              (applicationConfigDefaultValue["USE_DARK_THEME"]).(bool),
		GoogleMapAPIKey:           (applicationConfigDefaultValue["GOOGLE_MAP_API_KEY"]).(string),
		RykvImageListColumnNumber: (applicationConfigDefaultValue["RYKV_IMAGE_LIST_COLUMN_NUMBER"]).(json.Number),
		RykvHotReload:             (applicationConfigDefaultValue["RYKV_HOT_RELOAD"]).(bool),
		MiDefaultBoard:            (applicationConfigDefaultValue["MI_DEFAULT_BOARD"]).(string),
		RykvDefaultPeriod:         (applicationConfigDefaultValue["RYKV_DEFAULT_PERIOD"]).(json.Number),
		MiDefaultPeriod:           (applicationConfigDefaultValue["MI_DEFAULT_PERIOD"]).(json.Number),
		IsShowShareFooter:         (applicationConfigDefaultValue["IS_SHOW_SHARE_FOOTER"]).(bool),
		DefaultPage:               (applicationConfigDefaultValue["DEFAULT_PAGE"]).(string),
		ShowTagsInList:            (applicationConfigDefaultValue["SHOW_TAGS_IN_LIST"]).(bool),
		RyuuJSONData:              (applicationConfigDefaultValue["RYUU_JSON_DATA"]).(*json.RawMessage),
		TagStruct:                 (applicationConfigDefaultValue["TAG_STRUCT"]).(*json.RawMessage),
		RepStruct:                 (applicationConfigDefaultValue["REP_STRUCT"]).(*json.RawMessage),
		RepTypeStruct:             (applicationConfigDefaultValue["REP_TYPE_STRUCT"]).(*json.RawMessage),
		DeviceStruct:              (applicationConfigDefaultValue["DEVICE_STRUCT"]).(*json.RawMessage),
		MiBoardStruct:             (applicationConfigDefaultValue["MI_BOARD_STRUCT"]).(*json.RawMessage),
		KFTLTemplate:              (applicationConfigDefaultValue["KFTL_TEMPLATE_STRUCT"]).(*json.RawMessage),
		DnoteJSONData:             (applicationConfigDefaultValue["DNOTE_JSON_DATA"]).(*json.RawMessage),
	}
}

var nullJSONStr = json.RawMessage("null")
var emptyArrayJSONStr = json.RawMessage("[]")
var applicationConfigDefaultValue = map[string]interface{}{
	"USER_ID":                       "",
	"DEVICE":                        "",
	"USE_DARK_THEME":                false,
	"GOOGLE_MAP_API_KEY":            "",
	"RYKV_IMAGE_LIST_COLUMN_NUMBER": json.Number("3"),
	"RYKV_HOT_RELOAD":               true,
	"RYKV_DEFAULT_PERIOD":           json.Number("-1"),
	"MI_DEFAULT_BOARD":              "Inbox",
	"MI_DEFAULT_PERIOD":             json.Number("-1"),
	"IS_SHOW_SHARE_FOOTER":          false,
	"DEFAULT_PAGE":                  "rykv",
	"SHOW_TAGS_IN_LIST":             true,
	"RYUU_JSON_DATA":                &emptyArrayJSONStr,
	"TAG_STRUCT":                    &nullJSONStr,
	"REP_STRUCT":                    &nullJSONStr,
	"REP_TYPE_STRUCT":               &nullJSONStr,
	"DEVICE_STRUCT":                 &nullJSONStr,
	"MI_BOARD_STRUCT":               &nullJSONStr,
	"KFTL_TEMPLATE_STRUCT":          &nullJSONStr,
	"DNOTE_JSON_DATA":               &nullJSONStr,
}

var ignoreDeviceNameConfigKey = []string{
	"RYUU_JSON_DATA",
	"TAG_STRUCT",
	"REP_STRUCT",
	"REP_TYPE_STRUCT",
	"DEVICE_STRUCT",
	"MI_BOARD_STRUCT",
	"KFTL_TEMPLATE_STRUCT",
	"DNOTE_JSON_DATA",
}

func (a *applicationConfigDAOSQLite3Impl) GetAllApplicationConfigs(ctx context.Context) ([]*ApplicationConfig, error) {
	a.m.RLock()
	defer a.m.RUnlock()
	sql := `
SELECT 
  /* USER_ID */
  USER_ID AS USER_ID,
  /* DEVICE */ 
  DEVICE AS DEVICE,
  /* USE_DARK_THEME */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'USE_DARK_THEME'
  ) AS USE_DARK_THEME,
  /* GOOGLE_MAP_API_KEY */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'GOOGLE_MAP_API_KEY'
  ) AS GOOGLE_MAP_API_KEY,
  /* RYKV_IMAGE_LIST_COLUMN_NUMBER */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'RYKV_IMAGE_LIST_COLUMN_NUMBER'
  ) AS RYKV_IMAGE_LIST_COLUMN_NUMBER,
  /* RYKV_HOT_RELOAD */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'RYKV_HOT_RELOAD'
  ) AS RYKV_HOT_RELOAD,
  /* MI_DEFAULT_BOARD */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'MI_DEFAULT_BOARD'
  ) AS MI_DEFAULT_BOARD,
  /* RYKV_DEFAULT_PERIOD */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'RYKV_DEFAULT_PERIOD'
  ) AS RYKV_DEFAULT_PERIOD,
  /* MI_DEFAULT_PERIOD */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'MI_DEFAULT_PERIOD'
  ) AS MI_DEFAULT_PERIOD,
  /* IS_SHOW_SHARE_FOOTER */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'IS_SHOW_SHARE_FOOTER'
  ) AS IS_SHOW_SHARE_FOOTER,
  /* DEFAULT_PAGE */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'DEFAULT_PAGE'
  ) AS DEFAULT_PAGE,
  /* SHOW_TAGS_IN_LIST */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'SHOW_TAGS_IN_LIST'
  ) AS SHOW_TAGS_IN_LIST,
  /* RYUU_JSON_DATA */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'RYUU_JSON_DATA'
  ) AS RYUU_JSON_DATA,
  /* TAG_STRUCT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'TAG_STRUCT'
  ) AS TAG_STRUCT,
  /* REP_STRUCT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'REP_STRUCT'
  ) AS REP_STRUCT,
  /* REP_TYPE_STRUCT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'REP_TYPE_STRUCT'
  ) AS REP_TYPE_STRUCT,
  /* DEVICE_STRUCT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'DEVICE_STRUCT'
  ) AS DEVICE_STRUCT,
  /* MI_BOARD_STRUCT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'MI_BOARD_STRUCT'
  ) AS MI_BOARD_STRUCT,
  /* KFTL_TEMPLATE_STRUCT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'KFTL_TEMPLATE_STRUCT'
  ) AS KFTL_TEMPLATE_STRUCT,
  /* DNOTE_JSON_DATA */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'DNOTE_JSON_DATA'
  ) AS DNOTE_JSON_DATA
FROM APPLICATION_CONFIG AS GROUPED_APPLICATION_CONFIG
GROUP BY USER_ID, DEVICE
`

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all application configs sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	rows, err := stmt.QueryContext(ctx,
		applicationConfigDefaultValue["USE_DARK_THEME"],
		applicationConfigDefaultValue["GOOGLE_MAP_API_KEY"],
		applicationConfigDefaultValue["RYKV_IMAGE_LIST_COLUMN_NUMBER"],
		applicationConfigDefaultValue["RYKV_HOT_RELOAD"],
		applicationConfigDefaultValue["MI_DEFAULT_BOARD"],
		applicationConfigDefaultValue["RYKV_DEFAULT_PERIOD"],
		applicationConfigDefaultValue["MI_DEFAULT_PERIOD"],
		applicationConfigDefaultValue["IS_SHOW_SHARE_FOOTER"],
		applicationConfigDefaultValue["DEFAULT_PAGE"],
		applicationConfigDefaultValue["SHOW_TAGS_IN_LIST"],
		applicationConfigDefaultValue["RYUU_JSON_DATA"],
		applicationConfigDefaultValue["TAG_STRUCT"],
		applicationConfigDefaultValue["REP_STRUCT"],
		applicationConfigDefaultValue["REP_TYPE_STRUCT"],
		applicationConfigDefaultValue["DEVICE_STRUCT"],
		applicationConfigDefaultValue["MI_BOARD_STRUCT"],
		applicationConfigDefaultValue["KFTL_TEMPLATE_STRUCT"],
		applicationConfigDefaultValue["DNOTE_JSON_DATA"],
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	applicationConfigs := []*ApplicationConfig{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			applicationConfig := &ApplicationConfig{}
			ryuuJSONData := ""
			tagStrcut := ""
			repStruct := ""
			repTypeStruct := ""
			deviceStruct := ""
			miBoardStruct := ""
			kftlTemplateStruct := ""
			dnoteJsonData := ""

			err = rows.Scan(
				&applicationConfig.UserID,
				&applicationConfig.Device,
				&applicationConfig.UseDarkTheme,
				&applicationConfig.GoogleMapAPIKey,
				&applicationConfig.RykvImageListColumnNumber,
				&applicationConfig.RykvHotReload,
				&applicationConfig.MiDefaultBoard,
				&applicationConfig.RykvDefaultPeriod,
				&applicationConfig.MiDefaultPeriod,
				&applicationConfig.IsShowShareFooter,
				&applicationConfig.DefaultPage,
				&applicationConfig.ShowTagsInList,
				&ryuuJSONData,
				&tagStrcut,
				&repStruct,
				&repTypeStruct,
				&deviceStruct,
				&miBoardStruct,
				&kftlTemplateStruct,
				&dnoteJsonData,
			)
			if err != nil {
				return nil, err
			}

			if ryuuJSONData != "" {
				r := json.RawMessage(ryuuJSONData)
				applicationConfig.RyuuJSONData = &r
			}
			if tagStrcut != "" {
				t := json.RawMessage(tagStrcut)
				applicationConfig.TagStruct = &t
			}
			if repStruct != "" {
				r := json.RawMessage(repStruct)
				applicationConfig.RepStruct = &r
			}
			if repTypeStruct != "" {
				r := json.RawMessage(repTypeStruct)
				applicationConfig.RepTypeStruct = &r
			}
			if deviceStruct != "" {
				d := json.RawMessage(deviceStruct)
				applicationConfig.DeviceStruct = &d
			}
			if miBoardStruct != "" {
				m := json.RawMessage(miBoardStruct)
				applicationConfig.MiBoardStruct = &m
			}
			if kftlTemplateStruct != "" {
				k := json.RawMessage(kftlTemplateStruct)
				applicationConfig.KFTLTemplate = &k
			}
			if dnoteJsonData != "" {
				d := json.RawMessage(dnoteJsonData)
				applicationConfig.DnoteJSONData = &d
			}

			applicationConfigs = append(applicationConfigs, applicationConfig)
		}
	}
	return applicationConfigs, nil
}

func (a *applicationConfigDAOSQLite3Impl) GetApplicationConfig(ctx context.Context, userID string, device string) (*ApplicationConfig, error) {
	a.m.RLock()
	defer a.m.RUnlock()
	sql := `
SELECT 
  /* USER_ID */
  USER_ID AS USER_ID,
  /* DEVICE */ 
  DEVICE AS DEVICE,
  /* USE_DARK_THEME */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'USE_DARK_THEME'
  ) AS USE_DARK_THEME,
  /* GOOGLE_MAP_API_KEY */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'GOOGLE_MAP_API_KEY'
  ) AS GOOGLE_MAP_API_KEY,
  /* RYKV_IMAGE_LIST_COLUMN_NUMBER */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'RYKV_IMAGE_LIST_COLUMN_NUMBER'
  ) AS RYKV_IMAGE_LIST_COLUMN_NUMBER,
  /* RYKV_HOT_RELOAD */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'RYKV_HOT_RELOAD'
  ) AS RYKV_HOT_RELOAD,
  /* MI_DEFAULT_BOARD */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'MI_DEFAULT_BOARD'
  ) AS MI_DEFAULT_BOARD,
  /* RYKV_DEFAULT_PERIOD */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'RYKV_DEFAULT_PERIOD'
  ) AS RYKV_DEFAULT_PERIOD,
  /* MI_DEFAULT_PERIOD */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'MI_DEFAULT_PERIOD'
  ) AS MI_DEFAULT_PERIOD,
  /* IS_SHOW_SHARE_FOOTER */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'IS_SHOW_SHARE_FOOTER'
  ) AS IS_SHOW_SHARE_FOOTER,
  /* DEFAULT_PAGE */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'DEFAULT_PAGE'
  ) AS DEFAULT_PAGE,
  /* SHOW_TAGS_IN_LIST */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = GROUPED_APPLICATION_CONFIG.DEVICE
	AND KEY = 'SHOW_TAGS_IN_LIST'
  ) AS SHOW_TAGS_IN_LIST,
  /* RYUU_JSON_DATA */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'RYUU_JSON_DATA'
  ) AS RYUU_JSON_DATA,
  /* TAG_STRUCT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'TAG_STRUCT'
  ) AS TAG_STRUCT,
  /* REP_STRUCT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'REP_STRUCT'
  ) AS REP_STRUCT,
  /* REP_TYPE_STRUCT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'REP_TYPE_STRUCT'
  ) AS REP_TYPE_STRUCT,
  /* DEVICE_STRUCT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'DEVICE_STRUCT'
  ) AS DEVICE_STRUCT,
  /* MI_BOARD_STRUCT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'MI_BOARD_STRUCT'
  ) AS MI_BOARD_STRUCT,
  /* KFTL_TEMPLATE_STRUCT */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'KFTL_TEMPLATE_STRUCT'
  ) AS KFTL_TEMPLATE_STRUCT,
  /* DNOTE_JSON_DATA */ (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL AND COUNT(VALUE) = 1 
		THEN VALUE
		ELSE ?
	  END
	FROM APPLICATION_CONFIG
	WHERE USER_ID = GROUPED_APPLICATION_CONFIG.USER_ID
	AND DEVICE = 'ALL'
	AND KEY = 'DNOTE_JSON_DATA'
  ) AS DNOTE_JSON_DATA
FROM APPLICATION_CONFIG AS GROUPED_APPLICATION_CONFIG
GROUP BY USER_ID, DEVICE
HAVING USER_ID = ? AND DEVICE = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get application config sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		applicationConfigDefaultValue["USE_DARK_THEME"],
		applicationConfigDefaultValue["GOOGLE_MAP_API_KEY"],
		applicationConfigDefaultValue["RYKV_IMAGE_LIST_COLUMN_NUMBER"],
		applicationConfigDefaultValue["RYKV_HOT_RELOAD"],
		applicationConfigDefaultValue["MI_DEFAULT_BOARD"],
		applicationConfigDefaultValue["RYKV_DEFAULT_PERIOD"],
		applicationConfigDefaultValue["MI_DEFAULT_PERIOD"],
		applicationConfigDefaultValue["IS_SHOW_SHARE_FOOTER"],
		applicationConfigDefaultValue["DEFAULT_PAGE"],
		applicationConfigDefaultValue["SHOW_TAGS_IN_LIST"],
		applicationConfigDefaultValue["RYUU_JSON_DATA"],
		applicationConfigDefaultValue["TAG_STRUCT"],
		applicationConfigDefaultValue["REP_STRUCT"],
		applicationConfigDefaultValue["REP_TYPE_STRUCT"],
		applicationConfigDefaultValue["DEVICE_STRUCT"],
		applicationConfigDefaultValue["MI_BOARD_STRUCT"],
		applicationConfigDefaultValue["KFTL_TEMPLATE_STRUCT"],
		applicationConfigDefaultValue["DNOTE_JSON_DATA"],

		userID,
		device,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	applicationConfigs := []*ApplicationConfig{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			applicationConfig := &ApplicationConfig{}
			ryuuJSONData := ""
			tagStrcut := ""
			repStruct := ""
			repTypeStruct := ""
			deviceStruct := ""
			miBoardStruct := ""
			kftlTemplateStruct := ""
			dnoteJsonData := ""

			err = rows.Scan(
				&applicationConfig.UserID,
				&applicationConfig.Device,
				&applicationConfig.UseDarkTheme,
				&applicationConfig.GoogleMapAPIKey,
				&applicationConfig.RykvImageListColumnNumber,
				&applicationConfig.RykvHotReload,
				&applicationConfig.MiDefaultBoard,
				&applicationConfig.RykvDefaultPeriod,
				&applicationConfig.MiDefaultPeriod,
				&applicationConfig.IsShowShareFooter,
				&applicationConfig.DefaultPage,
				&applicationConfig.ShowTagsInList,
				&ryuuJSONData,
				&tagStrcut,
				&repStruct,
				&repTypeStruct,
				&deviceStruct,
				&miBoardStruct,
				&kftlTemplateStruct,
				&dnoteJsonData,
			)

			if ryuuJSONData != "" {
				r := json.RawMessage(ryuuJSONData)
				applicationConfig.RyuuJSONData = &r
			}
			if tagStrcut != "" {
				t := json.RawMessage(tagStrcut)
				applicationConfig.TagStruct = &t
			}
			if repStruct != "" {
				r := json.RawMessage(repStruct)
				applicationConfig.RepStruct = &r
			}
			if repTypeStruct != "" {
				r := json.RawMessage(repTypeStruct)
				applicationConfig.RepTypeStruct = &r
			}
			if deviceStruct != "" {
				d := json.RawMessage(deviceStruct)
				applicationConfig.DeviceStruct = &d
			}
			if miBoardStruct != "" {
				m := json.RawMessage(miBoardStruct)
				applicationConfig.MiBoardStruct = &m
			}
			if kftlTemplateStruct != "" {
				k := json.RawMessage(kftlTemplateStruct)
				applicationConfig.KFTLTemplate = &k
			}
			if dnoteJsonData != "" {
				d := json.RawMessage(dnoteJsonData)
				applicationConfig.DnoteJSONData = &d
			}

			applicationConfigs = append(applicationConfigs, applicationConfig)
		}
	}
	if len(applicationConfigs) == 0 {
		// なかったらデフォ値を返す。
		application_config := &ApplicationConfig{
			UserID:                    userID,
			Device:                    device,
			UseDarkTheme:              (applicationConfigDefaultValue["USE_DARK_THEME"]).(bool),
			GoogleMapAPIKey:           (applicationConfigDefaultValue["GOOGLE_MAP_API_KEY"]).(string),
			RykvImageListColumnNumber: (applicationConfigDefaultValue["RYKV_IMAGE_LIST_COLUMN_NUMBER"]).(json.Number),
			RykvHotReload:             (applicationConfigDefaultValue["RYKV_HOT_RELOAD"]).(bool),
			MiDefaultBoard:            (applicationConfigDefaultValue["MI_DEFAULT_BOARD"]).(string),
			RykvDefaultPeriod:         (applicationConfigDefaultValue["RYKV_DEFAULT_PERIOD"]).(json.Number),
			MiDefaultPeriod:           (applicationConfigDefaultValue["MI_DEFAULT_PERIOD"]).(json.Number),
			IsShowShareFooter:         (applicationConfigDefaultValue["IS_SHOW_SHARE_FOOTER"]).(bool),
			DefaultPage:               (applicationConfigDefaultValue["DEFAULT_PAGE"]).(string),
			ShowTagsInList:            (applicationConfigDefaultValue["SHOW_TAGS_IN_LIST"]).(bool),
			RyuuJSONData:              (applicationConfigDefaultValue["RYUU_JSON_DATA"]).(*json.RawMessage),
			TagStruct:                 (applicationConfigDefaultValue["TAG_STRUCT"]).(*json.RawMessage),
			RepStruct:                 (applicationConfigDefaultValue["REP_STRUCT"]).(*json.RawMessage),
			RepTypeStruct:             (applicationConfigDefaultValue["REP_TYPE_STRUCT"]).(*json.RawMessage),
			DeviceStruct:              (applicationConfigDefaultValue["DEVICE_STRUCT"]).(*json.RawMessage),
			MiBoardStruct:             (applicationConfigDefaultValue["MI_BOARD_STRUCT"]).(*json.RawMessage),
			KFTLTemplate:              (applicationConfigDefaultValue["KFTL_TEMPLATE_STRUCT"]).(*json.RawMessage),
			DnoteJSONData:             (applicationConfigDefaultValue["DNOTE_JSON_DATA"]).(*json.RawMessage),
		}
		return application_config, nil
	} else if len(applicationConfigs) == 1 {
		return applicationConfigs[0], nil
	}
	return nil, fmt.Errorf("複数のアプリケーションコンフィグが見つかりました。: %w", err)
}

func (a *applicationConfigDAOSQLite3Impl) AddApplicationConfig(ctx context.Context, applicationConfig *ApplicationConfig) (bool, error) {
	a.m.Lock()
	defer a.m.Unlock()
	sql := `
INSERT INTO APPLICATION_CONFIG (
  USER_ID,
  DEVICE,
  KEY,
  VALUE
) VALUES (
  ?,
  ?,
  ?,
  ?
)
`

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at update cache: %w", "error", err)
			}
		}
	}()

	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add application config sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	insertValuesMap := map[string]interface{}{
		"USE_DARK_THEME":                applicationConfig.UseDarkTheme,
		"GOOGLE_MAP_API_KEY":            applicationConfig.GoogleMapAPIKey,
		"RYKV_IMAGE_LIST_COLUMN_NUMBER": applicationConfig.RykvImageListColumnNumber,
		"RYKV_HOT_RELOAD":               applicationConfig.RykvHotReload,
		"MI_DEFAULT_BOARD":              applicationConfig.MiDefaultBoard,
		"RYKV_DEFAULT_PERIOD":           applicationConfig.RykvDefaultPeriod,
		"MI_DEFAULT_PERIOD":             applicationConfig.MiDefaultPeriod,
		"IS_SHOW_SHARE_FOOTER":          applicationConfig.IsShowShareFooter,
		"DEFAULT_PAGE":                  applicationConfig.DefaultPage,
		"SHOW_TAGS_IN_LIST":             applicationConfig.ShowTagsInList,
		"RYUU_JSON_DATA":                applicationConfig.RyuuJSONData,
		"TAG_STRUCT":                    applicationConfig.TagStruct,
		"REP_STRUCT":                    applicationConfig.RepStruct,
		"REP_TYPE_STRUCT":               applicationConfig.RepTypeStruct,
		"DEVICE_STRUCT":                 applicationConfig.DeviceStruct,
		"MI_BOARD_STRUCT":               applicationConfig.MiBoardStruct,
		"KFTL_TEMPLATE_STRUCT":          applicationConfig.KFTLTemplate,
		"DNOTE_JSON_DATA":               applicationConfig.DnoteJSONData,
	}
	for key, value := range insertValuesMap {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
		device := applicationConfig.Device
		isIgnoreDeviceNameKey := false
		for _, ignoreDeviceNameKey := range ignoreDeviceNameConfigKey {
			if key == ignoreDeviceNameKey {
				isIgnoreDeviceNameKey = true
				break
			}
		}
		if isIgnoreDeviceNameKey {
			device = "ALL"
		}
		queryArgs := []interface{}{
			applicationConfig.UserID,
			device,
			key,
			value,
		}
		slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
		_, err = insertStmt.ExecContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at add application config sql: %w", err)
			err = fmt.Errorf("error at query :%w", err)
			return false, err
		}
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		return false, err
	}
	isCommitted = true
	return true, nil
}

func (a *applicationConfigDAOSQLite3Impl) AddDefaultApplicationConfig(ctx context.Context, userID string, device string) (bool, error) {
	a.m.Lock()
	defer a.m.Unlock()
	sql := `
INSERT INTO APPLICATION_CONFIG (
  USER_ID,
  DEVICE,
  KEY,
  VALUE
) VALUES (
  ?,
  ?,
  ?,
  ?
)
`

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at update cache: %w", "error", err)
			}
		}
	}()

	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add application config sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	insertValuesMap := map[string]interface{}{
		"USE_DARK_THEME":                applicationConfigDefaultValue["USE_DARK_THEME"],
		"GOOGLE_MAP_API_KEY":            applicationConfigDefaultValue["GOOGLE_MAP_API_KEY"],
		"RYKV_IMAGE_LIST_COLUMN_NUMBER": applicationConfigDefaultValue["RYKV_IMAGE_LIST_COLUMN_NUMBER"],
		"RYKV_HOT_RELOAD":               applicationConfigDefaultValue["RYKV_HOT_RELOAD"],
		"RYKV_DEFAULT_PERIOD":           applicationConfigDefaultValue["RYKV_DEFAULT_PERIOD"],
		"MI_DEFAULT_BOARD":              applicationConfigDefaultValue["MI_DEFAULT_BOARD"],
		"MI_DEFAULT_PERIOD":             applicationConfigDefaultValue["MI_DEFAULT_PERIOD"],
		"IS_SHOW_SHARE_FOOTER":          applicationConfigDefaultValue["IS_SHOW_SHARE_FOOTER"],
		"DEFAULT_PAGE":                  applicationConfigDefaultValue["DEFAULT_PAGE"],
		"SHOW_TAGS_IN_LIST":             applicationConfigDefaultValue["SHOW_TAGS_IN_LIST"],
		"RYUU_JSON_DATA":                applicationConfigDefaultValue["RYUU_JSON_DATA"],
		"TAG_STRUCT":                    applicationConfigDefaultValue["TAG_STRUCT"],
		"REP_STRUCT":                    applicationConfigDefaultValue["REP_STRUCT"],
		"REP_TYPE_STRUCT":               applicationConfigDefaultValue["REP_TYPE_STRUCT"],
		"DEVICE_STRUCT":                 applicationConfigDefaultValue["DEVICE_STRUCT"],
		"MI_BOARD_STRUCT":               applicationConfigDefaultValue["MI_BOARD_STRUCT"],
		"KFTL_TEMPLATE_STRUCT":          applicationConfigDefaultValue["KFTL_TEMPLATE_STRUCT"],
		"DNOTE_JSON_DATA":               applicationConfigDefaultValue["DNOTE_JSON_DATA"],
	}
	for key, value := range insertValuesMap {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
		device := device
		isIgnoreDeviceNameKey := false
		for _, ignoreDeviceNameKey := range ignoreDeviceNameConfigKey {
			if key == ignoreDeviceNameKey {
				isIgnoreDeviceNameKey = true
				break
			}
		}
		if isIgnoreDeviceNameKey {
			device = "ALL"
		}
		queryArgs := []interface{}{
			userID,
			device,
			key,
			value,
		}
		slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
		_, err = insertStmt.ExecContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at add application config sql: %w", err)
			err = fmt.Errorf("error at query :%w", err)
			return false, err
		}
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		return false, err
	}
	isCommitted = true
	return true, nil
}

func (a *applicationConfigDAOSQLite3Impl) UpdateApplicationConfig(ctx context.Context, applicationConfig *ApplicationConfig) (bool, error) {
	a.m.Lock()
	defer a.m.Unlock()
	sql := `
UPDATE APPLICATION_CONFIG SET
VALUE = ?
WHERE USER_ID = ?
AND DEVICE = ?
AND KEY = ?
`
	checkExistSQL := `
SELECT COUNT(*)
FROM APPLICATION_CONFIG
WHERE USER_ID = ?
AND DEVICE = ?
AND KEY = ?
`
	insertSQL := `
INSERT INTO APPLICATION_CONFIG (
  USER_ID,
  DEVICE,
  KEY,
  VALUE
) VALUES (
  ?,
  ?,
  ?,
  ?
)
`

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	isCommitted := false
	defer func() {
		if !isCommitted {
			err := tx.Rollback()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback at update cache: %w", "error", err)
			}
		}
	}()

	updateValuesMap := map[string]interface{}{
		"USE_DARK_THEME":                applicationConfig.UseDarkTheme,
		"GOOGLE_MAP_API_KEY":            applicationConfig.GoogleMapAPIKey,
		"RYKV_IMAGE_LIST_COLUMN_NUMBER": applicationConfig.RykvImageListColumnNumber,
		"RYKV_HOT_RELOAD":               applicationConfig.RykvHotReload,
		"MI_DEFAULT_BOARD":              applicationConfig.MiDefaultBoard,
		"RYKV_DEFAULT_PERIOD":           applicationConfig.RykvDefaultPeriod,
		"MI_DEFAULT_PERIOD":             applicationConfig.MiDefaultPeriod,
		"IS_SHOW_SHARE_FOOTER":          applicationConfig.IsShowShareFooter,
		"DEFAULT_PAGE":                  applicationConfig.DefaultPage,
		"SHOW_TAGS_IN_LIST":             applicationConfig.ShowTagsInList,
		"RYUU_JSON_DATA":                applicationConfig.RyuuJSONData,
		"TAG_STRUCT":                    applicationConfig.TagStruct,
		"REP_STRUCT":                    applicationConfig.RepStruct,
		"REP_TYPE_STRUCT":               applicationConfig.RepTypeStruct,
		"DEVICE_STRUCT":                 applicationConfig.DeviceStruct,
		"MI_BOARD_STRUCT":               applicationConfig.MiBoardStruct,
		"KFTL_TEMPLATE_STRUCT":          applicationConfig.KFTLTemplate,
		"DNOTE_JSON_DATA":               applicationConfig.DnoteJSONData,
	}

	checkExistStmt, err := tx.PrepareContext(ctx, checkExistSQL)
	if err != nil {
		err = fmt.Errorf("error at pre get application config sql: %w", err)
		return false, err
	}
	defer func() {
		err := checkExistStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	insertStmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		err = fmt.Errorf("error at add application config sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	updateStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	defer func() {
		err := updateStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	// レコード自体が存在しなかったらいれる
	for key, value := range updateValuesMap {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)

		device := applicationConfig.Device
		isIgnoreDeviceNameKey := false
		for _, ignoreDeviceNameKey := range ignoreDeviceNameConfigKey {
			if key == ignoreDeviceNameKey {
				isIgnoreDeviceNameKey = true
				break
			}
		}
		if isIgnoreDeviceNameKey {
			device = "ALL"
		}
		queryArgs := []interface{}{
			applicationConfig.UserID,
			device,
			key,
		}
		slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", checkExistSQL, queryArgs)
		row := checkExistStmt.QueryRowContext(ctx, queryArgs...)
		err = row.Err()
		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			return false, err
		}

		recordCount := 0
		err = row.Scan(&recordCount)
		if err != nil {
			err = fmt.Errorf("error at scan:%w", err)
			return false, err
		}
		if recordCount == 0 {
			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", insertSQL)
			device := applicationConfig.Device
			isIgnoreDeviceNameKey := false
			for _, ignoreDeviceNameKey := range ignoreDeviceNameConfigKey {
				if key == ignoreDeviceNameKey {
					isIgnoreDeviceNameKey = true
					break
				}
			}
			if isIgnoreDeviceNameKey {
				device = "ALL"
			}
			queryArgs := []interface{}{
				applicationConfig.UserID,
				device,
				key,
				value,
			}
			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", insertSQL, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)

			if err != nil {
				err = fmt.Errorf("error at add application config sql: %w", err)
				err = fmt.Errorf("error at query :%w", err)
				return false, err
			}
		}
	}

	// 更新する
	for key, value := range updateValuesMap {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
		device := applicationConfig.Device
		isIgnoreDeviceNameKey := false
		for _, ignoreDeviceNameKey := range ignoreDeviceNameConfigKey {
			if key == ignoreDeviceNameKey {
				isIgnoreDeviceNameKey = true
				break
			}
		}
		if isIgnoreDeviceNameKey {
			device = "ALL"
		}
		queryArgs := []interface{}{
			value,
			applicationConfig.UserID,
			device,
			key,
		}
		slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
		_, err = updateStmt.ExecContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			return false, err
		}
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		return false, err
	}
	isCommitted = true
	return true, nil
}

func (a *applicationConfigDAOSQLite3Impl) DeleteApplicationConfig(ctx context.Context, userID string, device string) (bool, error) {
	a.m.Lock()
	defer a.m.Unlock()
	sql := `
DELETE FROM APPLICATION_CONFIG 
WHERE USER_ID = ? AND DEVICE = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete application config sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []interface{}{
		userID,
		device,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (a *applicationConfigDAOSQLite3Impl) Close(ctx context.Context) error {
	a.m.Lock()
	defer a.m.Unlock()
	return a.db.Close()
}

func checkAndResolveDataSchemaApplicationConfigDAO(ctx context.Context, db *sql.DB) (isOld bool, oldVerDAO ApplicationConfigDAO, err error) {
	schemaVersionKey := "SCHEMA_VERSION_APPLICATION_CONFIG"
	currentSchemaVersion := CURRENT_SCHEMA_VERSION_APPLICATION_CONFIG_DAO

	// テーブルとインデックスがなければ作る
	createTableSQL := `
CREATE TABLE IF NOT EXISTS GKILL_META_INFO (
  KEY NOT NULL,
  VALUE,
  PRIMARY KEY(KEY)
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", createTableSQL)
	stmt, err := db.PrepareContext(ctx, createTableSQL)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info table statement: %w", err)
		return false, nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", createTableSQL)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info table: %w", err)
		return false, nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_GKILL_META_INFO ON GKILL_META_INFO (KEY);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info index statement: %w", err)
		return false, nil, err
	}
	defer func() {
		err := indexStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info index: %w", err)
		return false, nil, err
	}

	// スキーマのージョンを取得する
	selectSchemaVersionSQL := `
SELECT 
  VALUE
FROM GKILL_META_INFO
WHERE KEY = ?
`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", selectSchemaVersionSQL)
	selectSchemaVersionStmt, err := db.PrepareContext(ctx, selectSchemaVersionSQL)
	if err != nil {
		err = fmt.Errorf("error at get schema version sql: %w", err)
		return false, nil, err
	}
	defer func() {
		err := selectSchemaVersionStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	dbSchemaVersion := ""
	queryArgs := []interface{}{schemaVersionKey}
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", selectSchemaVersionSQL, "query", queryArgs)
	err = selectSchemaVersionStmt.QueryRowContext(ctx, queryArgs...).Scan(&dbSchemaVersion)
	if err != nil {
		// データがなかったら今のバージョンをいれる
		if errors.Is(err, sql.ErrNoRows) {
			insertCurrentVersionSQL := `
INSERT INTO GKILL_META_INFO(KEY, VALUE)
VALUES(?, ?)`
			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", insertCurrentVersionSQL)
			insertCurrentVersionStmt, err := db.PrepareContext(ctx, insertCurrentVersionSQL)
			if err != nil {
				err = fmt.Errorf("error at get schema version sql: %w", err)
				err = fmt.Errorf("error at insert schema version sql: %w", err)
				return false, nil, err
			}
			defer func() {
				err := insertCurrentVersionStmt.Close()
				if err != nil {
					slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
				}
			}()
			queryArgs := []interface{}{schemaVersionKey, currentSchemaVersion}
			slog.Log(ctx, gkill_log.TraceSQL, "sql: %s query: %#v", insertCurrentVersionSQL, queryArgs)
			_, err = insertCurrentVersionStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at get schema version sql: %w", err)
				err = fmt.Errorf("error at query :%w", err)
				return false, nil, err
			}

			queryArgs = []interface{}{schemaVersionKey}
			slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", selectSchemaVersionSQL, "query", queryArgs)
			err = selectSchemaVersionStmt.QueryRowContext(ctx, queryArgs...).Scan(&dbSchemaVersion)
			if err != nil {
				err = fmt.Errorf("error at get schema version sql: %w", err)
				return false, nil, err
			}
		} else {
			err = fmt.Errorf("error at query :%w", err)
			return false, nil, err
		}
	}

	// ここから 過去バージョンのスキーマだった場合の対応
	if currentSchemaVersion != dbSchemaVersion {
		switch dbSchemaVersion {
		case "1.0.0":
			// 過去のDAOを作って返す or 最新のDAOに変換して返す
		}
		err = fmt.Errorf("invalid db schema version %s", dbSchemaVersion)
		return true, nil, err
	}
	// ここまで 過去バージョンのスキーマだった場合の対応

	return false, nil, nil
}
