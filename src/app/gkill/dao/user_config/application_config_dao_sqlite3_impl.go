package user_config

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=1&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
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

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_APPLICATION_CONFIG ON APPLICATION_CONFIG (USER_ID, DEVICE, KEY);`
	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create APPLICATION_CONFIG index statement %s: %w", filename, err)
		return nil, err
	}
	defer indexStmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create APPLICATION_CONFIG index to %s: %w", filename, err)
		return nil, err
	}

	return &applicationConfigDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
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

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all application configs sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
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
	defer rows.Close()

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
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := a.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get application config sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

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

	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add application config sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer insertStmt.Close()

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
		gkill_log.TraceSQL.Printf("sql: %s", sql)
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
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
		_, err = insertStmt.ExecContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at add application config sql: %w", err)
			err = fmt.Errorf("error at query :%w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}

	return true, nil
}

func (a *applicationConfigDAOSQLite3Impl) AddDefaultApplicationConfig(ctx context.Context, userID string, device string) (bool, error) {
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

	insertStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add application config sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer insertStmt.Close()

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
		gkill_log.TraceSQL.Printf("sql: %s", sql)
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
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
		_, err = insertStmt.ExecContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at add application config sql: %w", err)
			err = fmt.Errorf("error at query :%w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}

	return true, nil
}

func (a *applicationConfigDAOSQLite3Impl) UpdateApplicationConfig(ctx context.Context, applicationConfig *ApplicationConfig) (bool, error) {
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
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer checkExistStmt.Close()

	insertStmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		err = fmt.Errorf("error at add application config sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer insertStmt.Close()

	updateStmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer updateStmt.Close()

	// レコード自体が存在しなかったらいれる
	for key, value := range updateValuesMap {
		gkill_log.TraceSQL.Printf("sql: %s", sql)

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
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", checkExistSQL, queryArgs)
		row := checkExistStmt.QueryRowContext(ctx, queryArgs...)
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
			err = fmt.Errorf("error at scan:%w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		if recordCount == 0 {
			gkill_log.TraceSQL.Printf("sql: %s", insertSQL)
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
			gkill_log.TraceSQL.Printf("sql: %s query: %#v", insertSQL, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)

			if err != nil {
				err = fmt.Errorf("error at add application config sql: %w", err)
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
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", sql, queryArgs)
		_, err = updateStmt.ExecContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
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
		userID,
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
