package sqlite3impl

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const TimeLayout = time.RFC3339

func EscapeSQLite(str string) string {
	return strings.ReplaceAll(str, "'", "''")
}

func GenerateFindSQLCommon(queryJSON string, whereCounter *int) (string, error) {
	var err error
	sql := ""

	// jsonからパースする
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json %s: %w", queryJSON, err)
		return "", err
	}

	// 削除済みであるかどうかのSQL追記
	if queryMap["is_deleted"] == fmt.Sprintf("%t", true) {
		if *whereCounter != 0 {
			sql += " AND "
		}
		sql += "IS_DELETED = 'TRUE'"
	} else {
		if *whereCounter != 0 {
			sql += " AND "
		}
		sql += "IS_DELETED = 'FALSE'"
	}

	// id検索である場合のSQL追記
	if queryMap["use_ids"] == fmt.Sprintf("%t", true) {
		ids := []string{}
		err := json.Unmarshal([]byte(queryMap["ids"]), ids)
		if err != nil {
			err = fmt.Errorf("error at parse ids %s: %w", ids, err)
			return "", nil
		}

		if *whereCounter != 0 {
			sql += " AND "
		}
		sql += "ID IN ("
		for i, id := range ids {
			sql += fmt.Sprintf("'%s'", id)
			if i != len(ids)-1 {
				sql += ", "
			}
		}
		sql += ")"
	}

	// 日付範囲指定ありの場合
	if queryMap["use_calendar"] == fmt.Sprintf("%t", true) {
		// 開始日時を指定するSQLを追記
		if queryMap["calendar_start_date"] != "" {
			startDate := &time.Time{}
			err = json.Unmarshal([]byte(queryMap["calendar_start_date"]), startDate)
			if err != nil {
				err = fmt.Errorf("error at parse calendar start date %s: %w", queryMap["calendar_start_date"])
				return "", err
			}

			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += EscapeSQLite(fmt.Sprintf("datetime(RELATED_TIME, 'localtime') >= datetime('%s', 'localtime')", startDate.Format(TimeLayout)))
			*whereCounter++
		}

		// 終了日時を指定するSQLを追記
		if queryMap["calendar_end_date"] != "" {
			endDate := &time.Time{}
			err = json.Unmarshal([]byte(queryMap["calendar_end_date"]), endDate)
			if err != nil {
				err = fmt.Errorf("error at parse calendar end date %s: %w", queryMap["calendar_end_date"])
				return "", err
			}

			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += EscapeSQLite(fmt.Sprintf("datetime(RELATED_TIME, 'localtime') <= datetime('%s', 'localtime')", endDate.Format(TimeLayout)))
			*whereCounter++
		}
	}
	return sql, nil
}

func GenerateNewID() string {
	return uuid.New().String()
}
