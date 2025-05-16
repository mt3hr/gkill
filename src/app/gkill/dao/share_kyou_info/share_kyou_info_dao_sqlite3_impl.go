package share_kyou_info

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type shareKyouInfoDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewShareKyouInfoDAOSQLite3Impl(ctx context.Context, filename string) (ShareKyouInfoDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=1&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
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
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create SHARE_KYOU_INFO table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
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
	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err = db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create SHARE_KYOU_INFO_OPTIONS table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create SHARE_KYOU_INFO_OPTIONS table to %s: %w", filename, err)
		return nil, err
	}

	return &shareKyouInfoDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}

var shareKyouInfoDefaultValue = map[string]interface{}{
	"IS_SHARE_TIME_ONLY":      false,
	"IS_SHARE_WITH_TAGS":      false,
	"IS_SHARE_WITH_TEXTS":     false,
	"IS_SHARE_WITH_TIMEISS":   false,
	"IS_SHARE_WITH_LOCATIONS": false,
}

func (m *shareKyouInfoDAOSQLite3Impl) GetAllKyouShareInfos(ctx context.Context) ([]*ShareKyouInfo, error) {
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

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all kyou share infos sql: %w", err)
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
			err = fmt.Errorf("error at scan kyou share info: %w", err)
			if err != nil {
				return nil, err
			}
			kyouShareInfos = append(kyouShareInfos, kyouShareInfo)
		}
	}
	return kyouShareInfos, nil
}

func (m *shareKyouInfoDAOSQLite3Impl) GetKyouShareInfos(ctx context.Context, userID string, device string) ([]*ShareKyouInfo, error) {
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

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get kyou share infos sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
		device,
	}
	gkill_log.TraceSQL.Printf("%#v", queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

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
	return kyouShareInfos, nil
}

func (m *shareKyouInfoDAOSQLite3Impl) GetKyouShareInfo(ctx context.Context, sharedID string) (*ShareKyouInfo, error) {
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

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get kyou share infos sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		sharedID,
	}
	gkill_log.TraceSQL.Printf("%#v", queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

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
	if len(kyouShareInfos) == 0 {
		return nil, nil
	}
	return kyouShareInfos[0], nil
}

func (m *shareKyouInfoDAOSQLite3Impl) AddKyouShareInfo(ctx context.Context, kyouShareInfo *ShareKyouInfo) (bool, error) {
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
	tx, err := m.db.Begin()
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add kyou share info sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		kyouShareInfo.ID,
		kyouShareInfo.UserID,
		kyouShareInfo.Device,
		kyouShareInfo.ShareTitle,
		kyouShareInfo.ShareID,
		kyouShareInfo.FindQueryJSON,
		kyouShareInfo.ViewType,
	}
	gkill_log.TraceSQL.Printf("%#v", queryArgs)
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
	insertValuesMap := map[string]interface{}{
		"IS_SHARE_TIME_ONLY":      kyouShareInfo.IsShareTimeOnly,
		"IS_SHARE_WITH_TAGS":      kyouShareInfo.IsShareWithTags,
		"IS_SHARE_WITH_TEXTS":     kyouShareInfo.IsShareWithTexts,
		"IS_SHARE_WITH_TIMEISS":   kyouShareInfo.IsShareWithTimeIss,
		"IS_SHARE_WITH_LOCATIONS": kyouShareInfo.IsShareWithLocations,
	}
	for key, value := range insertValuesMap {
		gkill_log.TraceSQL.Printf("sql: %s", optionsSQL)
		stmt, err := tx.PrepareContext(ctx, optionsSQL)
		if err != nil {
			err = fmt.Errorf("error at add share kyou info options sql: %w", err)
			err = fmt.Errorf("error at query :%w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

		queryArgs := []interface{}{
			kyouShareInfo.ShareID,
			key,
			value,
		}
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", optionsSQL, queryArgs)
		_, err = stmt.ExecContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at add share kyou info options sql: %w", err)
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

func (m *shareKyouInfoDAOSQLite3Impl) UpdateKyouShareInfo(ctx context.Context, kyouShareInfo *ShareKyouInfo) (bool, error) {
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
	tx, err := m.db.Begin()
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update kyou share info sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		kyouShareInfo.ID,
		kyouShareInfo.UserID,
		kyouShareInfo.Device,
		kyouShareInfo.ShareTitle,
		kyouShareInfo.FindQueryJSON,
		kyouShareInfo.ViewType,
		kyouShareInfo.ShareID,
	}
	gkill_log.TraceSQL.Printf("%#v", queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	optionsSQL := `
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
	insertValuesMap := map[string]interface{}{
		"IS_SHARE_TIME_ONLY":      kyouShareInfo.IsShareTimeOnly,
		"IS_SHARE_WITH_TAGS":      kyouShareInfo.IsShareWithTags,
		"IS_SHARE_WITH_TEXTS":     kyouShareInfo.IsShareWithTexts,
		"IS_SHARE_WITH_TIMEISS":   kyouShareInfo.IsShareWithTimeIss,
		"IS_SHARE_WITH_LOCATIONS": kyouShareInfo.IsShareWithLocations,
	}

	// レコード自体が存在しなかったらいれる
	for key, value := range insertValuesMap {
		gkill_log.TraceSQL.Printf("sql: %s", sql)
		stmt, err := tx.PrepareContext(ctx, checkExistSQL)
		if err != nil {
			err = fmt.Errorf("error at pre get share kyou info options sql: %w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

		queryArgs := []interface{}{
			kyouShareInfo.ShareID,
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
				err = fmt.Errorf("error at add share kyou info sql: %w", err)
				err = fmt.Errorf("error at query :%w", err)
				rollbackErr := tx.Rollback()
				if rollbackErr != nil {
					err = fmt.Errorf("%w: %w", err, rollbackErr)
				}
				return false, err
			}
			defer stmt.Close()

			queryArgs := []interface{}{
				kyouShareInfo.ShareID,
				key,
				value,
			}
			gkill_log.TraceSQL.Printf("sql: %s query: %#v", insertSQL, queryArgs)
			_, err = stmt.ExecContext(ctx, queryArgs...)

			if err != nil {
				err = fmt.Errorf("error at add share kyou info options sql: %w", err)
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
	for key, value := range insertValuesMap {
		gkill_log.TraceSQL.Printf("sql: %s", optionsSQL)
		stmt, err := tx.PrepareContext(ctx, optionsSQL)
		if err != nil {
			err = fmt.Errorf("error at update share kyou info options sql: %w", err)
			err = fmt.Errorf("error at query :%w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

		queryArgs := []interface{}{
			value,
			kyouShareInfo.ShareID,
			key,
		}
		gkill_log.TraceSQL.Printf("sql: %s query: %#v", optionsSQL, queryArgs)
		_, err = stmt.ExecContext(ctx, queryArgs...)
		if err != nil {
			err = fmt.Errorf("error at update share kyou info options sql: %w", err)
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

func (m *shareKyouInfoDAOSQLite3Impl) DeleteKyouShareInfo(ctx context.Context, shareID string) (bool, error) {
	sql := `
DELETE FROM SHARE_KYOU_INFO
WHERE SHARE_ID = ?
`
	tx, err := m.db.Begin()
	if err != nil {
		err = fmt.Errorf("error at begin: %w", err)
		return false, err
	}

	gkill_log.TraceSQL.Printf("sql: %s", sql)
	stmt, err := tx.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete kyou share info sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		shareID,
	}
	gkill_log.TraceSQL.Printf("%#v", queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}

	optionsSQL := `
DELETE FROM SHARE_KYOU_INFO_OPTIONS
WHERE SHARE_ID = ?
`
	gkill_log.TraceSQL.Printf("sql: %s", optionsSQL)
	stmt, err = tx.PrepareContext(ctx, optionsSQL)
	if err != nil {
		err = fmt.Errorf("error at delete share kyou info options sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}
	defer stmt.Close()

	queryArgs = []interface{}{
		shareID,
	}
	gkill_log.TraceSQL.Printf("sql: %s query: %#v", optionsSQL, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at delete share kyou info options sql: %w", err)
		err = fmt.Errorf("error at query :%w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
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

func (m *shareKyouInfoDAOSQLite3Impl) Close(ctx context.Context) error {
	return m.db.Close()
}
