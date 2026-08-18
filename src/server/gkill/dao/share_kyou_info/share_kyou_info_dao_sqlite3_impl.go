package share_kyou_info

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	_ "modernc.org/sqlite"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

const CURRENT_SCHEMA_VERSION_SHARE_KYOU_INFO_DAO = "1.1.0"

type shareKyouInfoDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.RWMutex
}

func NewShareKyouInfoDAOSQLite3Impl(ctx context.Context, filename string) (ShareKyouInfoDAO, error) {
	var err error
	db, err := sql.Open("sqlite", "file:"+filename+"?_pragma=busy_timeout(6000)&_pragma=synchronous(NORMAL)&_pragma=journal_mode(DELETE)")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	if isOld, oldVerDAO, err := checkAndResolveDataSchemaShareKyouInfoDAO(ctx, db); err != nil {
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
CREATE TABLE IF NOT EXISTS "SHARE_KYOU_INFO" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  SHARE_TITLE NOT NULL,
  SHARE_ID NOT NULL,
  FIND_QUERY_JSON NOT NULL,
  VIEW_TYPE NOT NULL
);`
	gkill_log.LogSQL(ctx, sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create SHARE_KYOU_INFO table statement %s: %w", filename, err)
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
		err = fmt.Errorf("error at create SHARE_KYOU_INFO table to %s: %w", filename, err)
		return nil, err
	}

	sql = `
CREATE TABLE IF NOT EXISTS "SHARE_KYOU_INFO_OPTIONS" (
  SHARE_ID NOT NULL,
  KEY NOT NULL,
  VALUE NOT NULL,
  PRIMARY KEY (SHARE_ID, KEY)
);`
	gkill_log.LogSQL(ctx, sql)
	stmt, err = db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create SHARE_KYOU_INFO_OPTIONS table statement %s: %w", filename, err)
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
		err = fmt.Errorf("error at create SHARE_KYOU_INFO_OPTIONS table to %s: %w", filename, err)
		return nil, err
	}

	if gkill_options.Optimize {
		err = sqlite3impl.Optimize(db)
		if err != nil {
			err = fmt.Errorf("error at optimize db %w", err)
			return nil, err
		}
	}

	return &shareKyouInfoDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.RWMutex{},
	}, nil
}

var shareKyouInfoDefaultValue = map[string]any{
	"IS_SHARE_TIME_ONLY":      false,
	"IS_SHARE_WITH_TAGS":      false,
	"IS_SHARE_WITH_TEXTS":     false,
	"IS_SHARE_WITH_TIMEISS":   false,
	"IS_SHARE_WITH_LOCATIONS": false,
}

func (m *shareKyouInfoDAOSQLite3Impl) GetAllKyouShareInfos(ctx context.Context) ([]*ShareKyouInfo, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	sql := fmt.Sprintf(`
SELECT 
  ID,
  USER_ID,
  DEVICE,
  SHARE_TITLE,
  SHARE_ID,
  FIND_QUERY_JSON,
  VIEW_TYPE,
  /* IS_SHARE_TIME_ONLY */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_TIME_ONLY'
  ) AS IS_SHARE_TIME_ONLY,
  /* IS_SHARE_WITH_TAGS */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_WITH_TAGS'
  ) AS IS_SHARE_WITH_TAGS,
  /* IS_SHARE_WITH_TEXTS */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_WITH_TEXTS'
  ) AS IS_SHARE_WITH_TEXTS,
  /* IS_SHARE_WITH_TIMEISS */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_WITH_TIMEISS'
  ) AS IS_SHARE_WITH_TIMEISS,
  /* IS_SHARE_WITH_LOCATIONS */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_WITH_LOCATIONS'
  ) AS IS_SHARE_WITH_LOCATIONS
FROM SHARE_KYOU_INFO
`,
		shareKyouInfoDefaultValue["IS_SHARE_TIME_ONLY"],
		shareKyouInfoDefaultValue["IS_SHARE_WITH_TAGS"],
		shareKyouInfoDefaultValue["IS_SHARE_WITH_TEXTS"],
		shareKyouInfoDefaultValue["IS_SHARE_WITH_TIMEISS"],
		shareKyouInfoDefaultValue["IS_SHARE_WITH_LOCATIONS"],
	)

	gkill_log.LogSQL(ctx, sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get all kyou share infos sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	gkill_log.LogSQL(ctx, sql)
	rows, err := stmt.QueryContext(ctx)
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

	kyouShareInfos := []*ShareKyouInfo{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyouShareInfo := &ShareKyouInfo{}
			err = rows.Scan(
				&kyouShareInfo.ID,
				&kyouShareInfo.UserID,
				&kyouShareInfo.Device,
				&kyouShareInfo.ShareTitle,
				&kyouShareInfo.ShareID,
				&kyouShareInfo.FindQueryJSON,
				&kyouShareInfo.ViewType,
				&kyouShareInfo.IsShareTimeOnly,
				&kyouShareInfo.IsShareWithTags,
				&kyouShareInfo.IsShareWithTexts,
				&kyouShareInfo.IsShareWithTimeIss,
				&kyouShareInfo.IsShareWithLocations,
			)
			if err != nil {
				err = fmt.Errorf("error at scan kyou share info: %w", err)
				return nil, err
			}
			kyouShareInfos = append(kyouShareInfos, kyouShareInfo)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return kyouShareInfos, nil
}

func (m *shareKyouInfoDAOSQLite3Impl) GetKyouShareInfos(ctx context.Context, userID string, device string) ([]*ShareKyouInfo, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	sql := fmt.Sprintf(`
SELECT 
  ID,
  USER_ID,
  DEVICE,
  SHARE_TITLE,
  SHARE_ID,
  FIND_QUERY_JSON,
  VIEW_TYPE,
  /* IS_SHARE_TIME_ONLY */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_TIME_ONLY'
  ) AS IS_SHARE_TIME_ONLY,
  /* IS_SHARE_WITH_TAGS */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_WITH_TAGS'
  ) AS IS_SHARE_WITH_TAGS,
  /* IS_SHARE_WITH_TEXTS */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_WITH_TEXTS'
  ) AS IS_SHARE_WITH_TEXTS,
  /* IS_SHARE_WITH_TIMEISS */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_WITH_TIMEISS'
  ) AS IS_SHARE_WITH_TIMEISS,
  /* IS_SHARE_WITH_LOCATIONS */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_WITH_LOCATIONS'
  ) AS IS_SHARE_WITH_LOCATIONS
FROM SHARE_KYOU_INFO
WHERE USER_ID = ? AND DEVICE = ?
`,
		shareKyouInfoDefaultValue["IS_SHARE_TIME_ONLY"],
		shareKyouInfoDefaultValue["IS_SHARE_WITH_TAGS"],
		shareKyouInfoDefaultValue["IS_SHARE_WITH_TEXTS"],
		shareKyouInfoDefaultValue["IS_SHARE_WITH_TIMEISS"],
		shareKyouInfoDefaultValue["IS_SHARE_WITH_LOCATIONS"],
	)

	gkill_log.LogSQL(ctx, sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou share infos sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []any{
		userID,
		device,
	}
	if gkill_log.TraceSQLEnabled(ctx) {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "query args", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	}
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

	kyouShareInfos := []*ShareKyouInfo{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyouShareInfo := &ShareKyouInfo{}
			err = rows.Scan(
				&kyouShareInfo.ID,
				&kyouShareInfo.UserID,
				&kyouShareInfo.Device,
				&kyouShareInfo.ShareTitle,
				&kyouShareInfo.ShareID,
				&kyouShareInfo.FindQueryJSON,
				&kyouShareInfo.ViewType,
				&kyouShareInfo.IsShareTimeOnly,
				&kyouShareInfo.IsShareWithTags,
				&kyouShareInfo.IsShareWithTexts,
				&kyouShareInfo.IsShareWithTimeIss,
				&kyouShareInfo.IsShareWithLocations,
			)
			if err != nil {
				err = fmt.Errorf("error at scan kyou share info: %w", err)
				return nil, err
			}
			kyouShareInfos = append(kyouShareInfos, kyouShareInfo)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return kyouShareInfos, nil
}

func (m *shareKyouInfoDAOSQLite3Impl) GetKyouShareInfo(ctx context.Context, sharedID string) (*ShareKyouInfo, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	sql := fmt.Sprintf(`
SELECT 
  ID,
  USER_ID,
  DEVICE,
  SHARE_TITLE,
  SHARE_ID,
  FIND_QUERY_JSON,
  VIEW_TYPE,
  /* IS_SHARE_TIME_ONLY */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_TIME_ONLY'
  ) AS IS_SHARE_TIME_ONLY,
  /* IS_SHARE_WITH_TAGS */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_WITH_TAGS'
  ) AS IS_SHARE_WITH_TAGS,
  /* IS_SHARE_WITH_TEXTS */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_WITH_TEXTS'
  ) AS IS_SHARE_WITH_TEXTS,
  /* IS_SHARE_WITH_TIMEISS */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_WITH_TIMEISS'
  ) AS IS_SHARE_WITH_TIMEISS,
  /* IS_SHARE_WITH_LOCATIONS */
  (
    SELECT 
	  CASE 
	    WHEN VALUE IS NOT NULL 
		THEN VALUE
		ELSE '%v'
	  END
	FROM SHARE_KYOU_INFO_OPTIONS
	WHERE SHARE_KYOU_INFO.SHARE_ID = SHARE_KYOU_INFO_OPTIONS.SHARE_ID
	AND SHARE_KYOU_INFO_OPTIONS.KEY = 'IS_SHARE_WITH_LOCATIONS'
  ) AS IS_SHARE_WITH_LOCATIONS
FROM SHARE_KYOU_INFO
WHERE SHARE_ID = ?
`,
		shareKyouInfoDefaultValue["IS_SHARE_TIME_ONLY"],
		shareKyouInfoDefaultValue["IS_SHARE_WITH_TAGS"],
		shareKyouInfoDefaultValue["IS_SHARE_WITH_TEXTS"],
		shareKyouInfoDefaultValue["IS_SHARE_WITH_TIMEISS"],
		shareKyouInfoDefaultValue["IS_SHARE_WITH_LOCATIONS"],
	)

	gkill_log.LogSQL(ctx, sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get kyou share infos sql: %w", err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []any{
		sharedID,
	}
	if gkill_log.TraceSQLEnabled(ctx) {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "query args", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	}
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

	kyouShareInfos := []*ShareKyouInfo{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyouShareInfo := &ShareKyouInfo{}
			err = rows.Scan(
				&kyouShareInfo.ID,
				&kyouShareInfo.UserID,
				&kyouShareInfo.Device,
				&kyouShareInfo.ShareTitle,
				&kyouShareInfo.ShareID,
				&kyouShareInfo.FindQueryJSON,
				&kyouShareInfo.ViewType,
				&kyouShareInfo.IsShareTimeOnly,
				&kyouShareInfo.IsShareWithTags,
				&kyouShareInfo.IsShareWithTexts,
				&kyouShareInfo.IsShareWithTimeIss,
				&kyouShareInfo.IsShareWithLocations,
			)
			if err != nil {
				err = fmt.Errorf("error at scan kyou share info: %w", err)
				return nil, err
			}
			kyouShareInfos = append(kyouShareInfos, kyouShareInfo)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	if len(kyouShareInfos) == 0 {
		return nil, nil
	}
	return kyouShareInfos[0], nil
}

func (m *shareKyouInfoDAOSQLite3Impl) AddKyouShareInfo(ctx context.Context, kyouShareInfo *ShareKyouInfo) (bool, error) {
	m.m.Lock()
	defer m.m.Unlock()
	sql := `
INSERT INTO SHARE_KYOU_INFO (
  ID,
  USER_ID,
  DEVICE,
  SHARE_TITLE,
  SHARE_ID,
  FIND_QUERY_JSON,
  VIEW_TYPE
) VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)
`
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
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

	gkill_log.LogSQL(ctx, sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add kyou share info sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []any{
		kyouShareInfo.ID,
		kyouShareInfo.UserID,
		kyouShareInfo.Device,
		kyouShareInfo.ShareTitle,
		kyouShareInfo.ShareID,
		kyouShareInfo.FindQueryJSON,
		kyouShareInfo.ViewType,
	}
	if gkill_log.TraceSQLEnabled(ctx) {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", queryArgs))
	}
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	optionsSQL := `
INSERT INTO SHARE_KYOU_INFO_OPTIONS (
  SHARE_ID,
  KEY,
  VALUE
) VALUES (
 ?,
 ?,
 ?
)
`
	insertValuesMap := map[string]any{
		"IS_SHARE_TIME_ONLY":      kyouShareInfo.IsShareTimeOnly,
		"IS_SHARE_WITH_TAGS":      kyouShareInfo.IsShareWithTags,
		"IS_SHARE_WITH_TEXTS":     kyouShareInfo.IsShareWithTexts,
		"IS_SHARE_WITH_TIMEISS":   kyouShareInfo.IsShareWithTimeIss,
		"IS_SHARE_WITH_LOCATIONS": kyouShareInfo.IsShareWithLocations,
	}

	optionsStmt, err := tx.PrepareContext(ctx, optionsSQL)
	if err != nil {
		err = fmt.Errorf("error at add share kyou info options sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	defer func() {
		err := optionsStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	for key, value := range insertValuesMap {
		gkill_log.LogSQL(ctx, optionsSQL)
		queryArgs := []any{
			kyouShareInfo.ShareID,
			key,
			value,
		}
		gkill_log.LogSQLQuery(ctx, optionsSQL, queryArgs)
		_, err = optionsStmt.ExecContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at add share kyou info options sql: %w", err)
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

func (m *shareKyouInfoDAOSQLite3Impl) UpdateKyouShareInfo(ctx context.Context, kyouShareInfo *ShareKyouInfo) (bool, error) {
	m.m.Lock()
	defer m.m.Unlock()
	sql := `
UPDATE SHARE_KYOU_INFO SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  SHARE_TITLE = ?,
  FIND_QUERY_JSON = ?,
  VIEW_TYPE = ?
WHERE SHARE_ID = ?
`
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
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

	gkill_log.LogSQL(ctx, sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update kyou share info sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []any{
		kyouShareInfo.ID,
		kyouShareInfo.UserID,
		kyouShareInfo.Device,
		kyouShareInfo.ShareTitle,
		kyouShareInfo.FindQueryJSON,
		kyouShareInfo.ViewType,
		kyouShareInfo.ShareID,
	}
	if gkill_log.TraceSQLEnabled(ctx) {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", queryArgs))
	}
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	updateOptionsSQL := `
UPDATE SHARE_KYOU_INFO_OPTIONS SET
  VALUE = ?
WHERE SHARE_ID = ?
AND KEY = ?
`
	checkExistSQL := `
SELECT COUNT(*)
FROM SHARE_KYOU_INFO_OPTIONS
WHERE SHARE_ID = ?
AND KEY = ?
`

	insertSQL := `
INSERT INTO SHARE_KYOU_INFO_OPTIONS (
  SHARE_ID,
  KEY,
  VALUE
) VALUES (
  ?,
  ?,
  ?
)
`

	updateOptionStmt, err := tx.PrepareContext(ctx, updateOptionsSQL)
	if err != nil {
		err = fmt.Errorf("error at update share kyou info options sql: %w", err)
		return false, err
	}
	defer func() {
		err := updateOptionStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	checkExistStmt, err := tx.PrepareContext(ctx, checkExistSQL)
	if err != nil {
		err = fmt.Errorf("error at pre get share kyou info options sql: %w", err)
		return false, err
	}
	defer func() {
		err := checkExistStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	insertValuesMap := map[string]any{
		"IS_SHARE_TIME_ONLY":      kyouShareInfo.IsShareTimeOnly,
		"IS_SHARE_WITH_TAGS":      kyouShareInfo.IsShareWithTags,
		"IS_SHARE_WITH_TEXTS":     kyouShareInfo.IsShareWithTexts,
		"IS_SHARE_WITH_TIMEISS":   kyouShareInfo.IsShareWithTimeIss,
		"IS_SHARE_WITH_LOCATIONS": kyouShareInfo.IsShareWithLocations,
	}

	insertStmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		err = fmt.Errorf("error at add share kyou info sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	defer func() {
		err := insertStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	// レコード自体が存在しなかったらいれる
	for key, value := range insertValuesMap {
		gkill_log.LogSQL(ctx, sql)
		queryArgs := []any{
			kyouShareInfo.ShareID,
			key,
		}
		gkill_log.LogSQLQuery(ctx, checkExistSQL, queryArgs)
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
			gkill_log.LogSQL(ctx, insertSQL)
			queryArgs := []any{
				kyouShareInfo.ShareID,
				key,
				value,
			}
			gkill_log.LogSQLQuery(ctx, insertSQL, queryArgs)
			_, err = insertStmt.ExecContext(ctx, queryArgs...)

			if err != nil {
				err = fmt.Errorf("error at add share kyou info options sql: %w", err)
				err = fmt.Errorf("error at query :%w", err)
				return false, err
			}
		} else {
			gkill_log.LogSQL(ctx, updateOptionsSQL)
			queryArgs := []any{
				value,
				kyouShareInfo.ShareID,
				key,
			}
			gkill_log.LogSQLQuery(ctx, updateOptionsSQL, queryArgs)
			_, err = updateOptionStmt.ExecContext(ctx, queryArgs...)

			if err != nil {
				err = fmt.Errorf("error at update share kyou info options sql: %w", err)
				err = fmt.Errorf("error at query :%w", err)
				return false, err
			}
		}
	}

	// 更新する
	gkill_log.LogSQL(ctx, sql)
	queryArgs = []any{
		kyouShareInfo.ID,
		kyouShareInfo.UserID,
		kyouShareInfo.Device,
		kyouShareInfo.ShareTitle,
		kyouShareInfo.FindQueryJSON,
		kyouShareInfo.ViewType,
		kyouShareInfo.ShareID,
	}
	gkill_log.LogSQLQuery(ctx, sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at update share kyou info options sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		return false, err
	}
	isCommitted = true
	return true, nil
}

func (m *shareKyouInfoDAOSQLite3Impl) DeleteKyouShareInfo(ctx context.Context, shareID string) (bool, error) {
	m.m.Lock()
	defer m.m.Unlock()
	sql := `
DELETE FROM SHARE_KYOU_INFO
WHERE SHARE_ID = ?
`
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
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

	gkill_log.LogSQL(ctx, sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete kyou share info sql: %w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := []any{
		shareID,
	}
	if gkill_log.TraceSQLEnabled(ctx) {
		slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", queryArgs))
	}
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	optionsSQL := `
DELETE FROM SHARE_KYOU_INFO_OPTIONS
WHERE SHARE_ID = ?
`
	gkill_log.LogSQL(ctx, optionsSQL)
	stmt, err = tx.PrepareContext(ctx, optionsSQL)
	if err != nil {
		err = fmt.Errorf("error at delete share kyou info options sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs = []any{
		shareID,
	}
	gkill_log.LogSQLQuery(ctx, optionsSQL, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at delete share kyou info options sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error at commit: %w", err)
		return false, err
	}
	isCommitted = true
	return true, nil
}

func (m *shareKyouInfoDAOSQLite3Impl) Close(ctx context.Context) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.db.Close()
}

func checkAndResolveDataSchemaShareKyouInfoDAO(ctx context.Context, db *sql.DB) (isOld bool, oldVerDAO ShareKyouInfoDAO, err error) {
	schemaVersionKey := "SCHEMA_VERSION_SHARE_KYOU_INFO"
	currentSchemaVersion := CURRENT_SCHEMA_VERSION_SHARE_KYOU_INFO_DAO

	// テーブルとインデックスがなければ作る
	createTableSQL := `
CREATE TABLE IF NOT EXISTS GKILL_META_INFO (
  KEY NOT NULL,
  VALUE,
  PRIMARY KEY(KEY)
);`
	gkill_log.LogSQL(ctx, createTableSQL)
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

	gkill_log.LogSQL(ctx, createTableSQL)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info table: %w", err)
		return false, nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_GKILL_META_INFO ON GKILL_META_INFO (KEY);`
	gkill_log.LogIndexSQL(ctx, indexSQL)
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

	gkill_log.LogIndexSQL(ctx, indexSQL)
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create gkill meta info index: %w", err)
		return false, nil, err
	}

	// スキーマのバージョンを取得する
	selectSchemaVersionSQL := `
SELECT 
  VALUE
FROM GKILL_META_INFO
WHERE KEY = ?
`
	gkill_log.LogSQL(ctx, selectSchemaVersionSQL)
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
	queryArgs := []any{schemaVersionKey}
	gkill_log.LogSQLQuery(ctx, selectSchemaVersionSQL, queryArgs)
	err = selectSchemaVersionStmt.QueryRowContext(ctx, queryArgs...).Scan(&dbSchemaVersion)
	if err != nil {
		// データがなかったら今のバージョンをいれる
		if errors.Is(err, sql.ErrNoRows) {
			insertCurrentVersionSQL := `
INSERT INTO GKILL_META_INFO(KEY, VALUE)
VALUES(?, ?)`
			gkill_log.LogSQL(ctx, insertCurrentVersionSQL)
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
			queryArgs := []any{schemaVersionKey, currentSchemaVersion}
			gkill_log.LogSQLQuery(ctx, insertCurrentVersionSQL, queryArgs)
			_, err = insertCurrentVersionStmt.ExecContext(ctx, queryArgs...)
			if err != nil {
				err = fmt.Errorf("error at get schema version sql: %w", err)
				err = fmt.Errorf("error at query :%w", err)
				return false, nil, err
			}

			queryArgs = []any{schemaVersionKey}
			gkill_log.LogSQLQuery(ctx, selectSchemaVersionSQL, queryArgs)
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
			if err := migrateShareKyouInfoSchemaFrom100(ctx, db, schemaVersionKey, currentSchemaVersion); err != nil {
				return true, nil, err
			}
			return false, nil, nil
		}
		err = fmt.Errorf("invalid db schema version %s", dbSchemaVersion)
		return true, nil, err
	}
	// ここまで 過去バージョンのスキーマだった場合の対応

	return false, nil, nil
}

// migrateShareKyouInfoSchemaFrom100 はスキーマ1.0.0のSHARE_KYOU_INFOを1.1.0へ移行する。
//
// FIND_QUERY_JSON に保存されている旧形式の FindQuery JSON（use_* フラグ入り）を
// null判定の新形式へ書き換える（find.MigrateLegacyFindQueryJSON）。
// 共有URLは外部に配布済みで再発行できないため、読み出し側に互換層を置くのではなく
// 保存データそのものを移行する。
// パース不能な行は警告ログを出してスキップする（旧実装でも読み出し時にUnmarshalで
// 失敗するだけの壊れ行なので、移行で起動を止めない）。
func migrateShareKyouInfoSchemaFrom100(ctx context.Context, db *sql.DB, schemaVersionKey string, currentSchemaVersion string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error at begin tx for share kyou info schema migration: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at rollback share kyou info schema migration", "error", err)
			}
		}
	}()

	// checkAndResolve は CREATE TABLE より前に走るため、
	// 版番号だけが残った不完全DBでも壊れないようテーブルの存在を確認する
	tableCount := 0
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'SHARE_KYOU_INFO'`).Scan(&tableCount)
	if err != nil {
		return fmt.Errorf("error at check SHARE_KYOU_INFO table existence: %w", err)
	}

	if tableCount != 0 {
		// LIKE は候補を絞る最適化で、旧形式かどうかの正確な判定はウォーカーが行う
		selectSQL := `SELECT ID, FIND_QUERY_JSON FROM SHARE_KYOU_INFO WHERE FIND_QUERY_JSON LIKE '%"use_%'`
		gkill_log.LogSQL(ctx, selectSQL)
		rows, err := tx.QueryContext(ctx, selectSQL)
		if err != nil {
			return fmt.Errorf("error at select legacy find query json for share kyou info schema migration: %w", err)
		}
		type migratedRow struct {
			id   string
			json string
		}
		migratedRows := []migratedRow{}
		for rows.Next() {
			id := ""
			findQueryJSON := ""
			if err := rows.Scan(&id, &findQueryJSON); err != nil {
				rows.Close()
				return fmt.Errorf("error at scan legacy find query json for share kyou info schema migration: %w", err)
			}
			migrated, changed, err := find.MigrateLegacyFindQueryJSON([]byte(findQueryJSON))
			if err != nil {
				slog.Log(ctx, gkill_log.Debug, "共有クエリJSONの移行をスキップしました（パース不能）", "id", id, "error", err)
				continue
			}
			if changed {
				migratedRows = append(migratedRows, migratedRow{id: id, json: string(migrated)})
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("error at iterate legacy find query json for share kyou info schema migration: %w", err)
		}
		rows.Close()

		updateSQL := `UPDATE SHARE_KYOU_INFO SET FIND_QUERY_JSON = ? WHERE ID = ?`
		for _, row := range migratedRows {
			if gkill_log.TraceSQLEnabled(ctx) {
				slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", updateSQL), "id", row.id)
			}
			if _, err := tx.ExecContext(ctx, updateSQL, row.json, row.id); err != nil {
				return fmt.Errorf("error at update find query json for share kyou info schema migration id = %s: %w", row.id, err)
			}
		}
	}

	updateVersionSQL := `UPDATE GKILL_META_INFO SET VALUE = ? WHERE KEY = ?`
	gkill_log.LogSQL(ctx, updateVersionSQL)
	if _, err := tx.ExecContext(ctx, updateVersionSQL, currentSchemaVersion, schemaVersionKey); err != nil {
		return fmt.Errorf("error at update schema version for share kyou info schema migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit share kyou info schema migration: %w", err)
	}
	committed = true
	return nil
}
