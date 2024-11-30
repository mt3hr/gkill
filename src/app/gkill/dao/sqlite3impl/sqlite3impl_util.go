package sqlite3impl

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
)

const TimeLayout = "2006-01-02T15:04:05-07:00"

func EscapeSQLite(str string) string {
	return strings.ReplaceAll(str, "'", "''")
}

func GenerateFindSQLCommon(query *find.FindQuery, whereCounter *int) (string, error) {
	sql := ""

	// jsonからパースする

	// 削除済みであるかどうかのSQL追記
	isDeleted := false
	if query.IsDeleted != nil {
		isDeleted = *query.IsDeleted
	}
	if isDeleted {
		if *whereCounter != 0 {
			sql += " AND "
		}
		sql += "IS_DELETED = TRUE"
		*whereCounter++
	} else {
		if *whereCounter != 0 {
			sql += " AND "
		}
		sql += "IS_DELETED = FALSE"
		*whereCounter++
	}

	// id検索である場合のSQL追記
	useIDs := false
	ids := []string{}
	if query.UseIDs != nil {
		useIDs = *query.UseIDs
	}
	if query.IDs != nil {
		ids = *query.IDs
	}

	if useIDs {
		if *whereCounter != 0 {
			sql += " AND "
		}
		sql += " ID IN ("
		for i, id := range ids {
			sql += fmt.Sprintf("'%s'", id)
			if i != len(ids)-1 {
				sql += ", "
			}
		}
		sql += ")"
	}

	// 日付範囲指定ありの場合
	useCalendar := false
	var calendarStartDate *time.Time
	var calendarEndDate *time.Time
	if query.UseCalendar != nil {
		useCalendar = *query.UseCalendar
	}
	if query.CalendarStartDate != nil {
		calendarStartDate = query.CalendarStartDate
	}
	if query.CalendarEndDate != nil {
		calendarEndDate = query.CalendarEndDate
	}

	if useCalendar {
		// 開始日時を指定するSQLを追記
		if calendarStartDate != nil {
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += EscapeSQLite(fmt.Sprintf("datetime(RELATED_TIME, 'localtime') >= datetime('%s', 'localtime')", calendarStartDate.Format(TimeLayout)))
			*whereCounter++
		}

		// 終了日時を指定するSQLを追記
		if calendarEndDate != nil {
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += EscapeSQLite(fmt.Sprintf("datetime(RELATED_TIME, 'localtime') <= datetime('%s', 'localtime')", calendarEndDate.Format(TimeLayout)))
			*whereCounter++
		}
	}
	return sql, nil
}

func GenerateNewID() string {
	return uuid.New().String()
}
