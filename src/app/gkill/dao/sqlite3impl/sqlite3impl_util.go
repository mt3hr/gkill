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

func GenerateFindSQLCommon(query *find.FindQuery, whereCounter *int, onlyLatestData bool, relatedTimeColumnName string, findWordTargetColumns []string, findWordUseLike bool, ignoreFindWord bool, appendGroupBy bool, appendOrderBy bool, queryArgs *[]interface{}) (string, error) {
	sql := ""

	// WHERE
	// id検索である場合のSQL追記
	useIDs := false
	ids := []string{}
	if query.UseIDs != nil {
		useIDs = *query.UseIDs
	}
	if query.IDs != nil {
		ids = *query.IDs
	}

	if useIDs && len(ids) != 0 {
		if *whereCounter != 0 {
			sql += " AND "
		}
		sql += " ID IN ("
		for i, id := range ids {
			sql += " ? "
			*queryArgs = append(*queryArgs, id)
			if i != len(ids)-1 {
				sql += ", "
			}
			*whereCounter++
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

	// UPDATE_TIMEか、Calendarの条件をSQLに追記
	if query.UseUpdateTime != nil && *query.UseUpdateTime && query.UpdateTime != nil {
		if query.UpdateTime != nil {
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += fmt.Sprintf("datetime(%s, 'localtime') = datetime(?, 'localtime')", "UPDATE_TIME")
			*queryArgs = append(*queryArgs, ((*query.UpdateTime).Format(TimeLayout)))
			*whereCounter++
		}
	} else if useCalendar {
		// 開始日時を指定するSQLを追記
		if calendarStartDate != nil {
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += fmt.Sprintf("datetime(%s, 'localtime') >= datetime(?, 'localtime')", relatedTimeColumnName)
			*queryArgs = append(*queryArgs, calendarStartDate.Format(TimeLayout))
			*whereCounter++
		}

		// 終了日時を指定するSQLを追記
		if calendarEndDate != nil {
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += fmt.Sprintf("datetime(%s, 'localtime') <= datetime(?, 'localtime')", relatedTimeColumnName)
			*queryArgs = append(*queryArgs, calendarEndDate.Format(TimeLayout))
			*whereCounter++
		}
	}

	// ワードand検索である場合のSQL追記
	if query.UseWords != nil && *query.UseWords {
		if query.Words != nil && len(*query.Words) != 0 {
			if query.WordsAnd != nil && *query.WordsAnd {
				if *whereCounter != 0 {
					sql += " AND "
				}
				for j, findWordTargetColumnName := range findWordTargetColumns {
					if j == 0 {
						sql += " ( "
					} else {
						sql += " AND "
					}

					for i, word := range *query.Words {
						if i == 0 {
							sql += " ( "
						} else {
							sql += " AND "
						}
						if findWordUseLike {
							sql += fmt.Sprintf("LOWER(%s) LIKE LOWER(?)", findWordTargetColumnName)
							*queryArgs = append(*queryArgs, "%"+word+"%")
						} else {
							sql += fmt.Sprintf("LOWER(%s) = LOWER(?)", findWordTargetColumnName)
							*queryArgs = append(*queryArgs, word)
						}
						if i == len(*query.Words)-1 {
							sql += " ) "
						}
						*whereCounter++
					}

					if j == len(findWordTargetColumns)-1 {
						sql += " ) "
					}
				}
			} else {
				// ワードor検索である場合のSQL追記
				if *whereCounter != 0 {
					sql += " AND "
				}
				for j, findWordTargetColumnName := range findWordTargetColumns {
					if j == 0 {
						sql += " ( "
					} else {
						sql += " OR "
					}

					for i, word := range *query.Words {
						if i == 0 {
							sql += " ( "
						} else {
							sql += " OR "
						}
						if findWordUseLike {
							sql += fmt.Sprintf("LOWER(%s) LIKE LOWER(?)", findWordTargetColumnName)
							*queryArgs = append(*queryArgs, "%"+word+"%")
						} else {
							sql += fmt.Sprintf("LOWER(%s) = LOWER(?)", findWordTargetColumnName)
							*queryArgs = append(*queryArgs, word)
						}
						if i == len(*query.Words)-1 {
							sql += " ) "
						}
						*whereCounter++
					}

					if j == len(findWordTargetColumns)-1 {
						sql += " ) "
					}
				}
			}
		}

		if query.NotWords != nil && len(*query.NotWords) != 0 {
			// notワードを除外するSQLを追記
			if *whereCounter != 0 {
				sql += " AND "
			}
			for j, findWordTargetColumnName := range findWordTargetColumns {
				if j == 0 {
					sql += " ( "
				} else {
					sql += " AND "
				}

				for i, notWord := range *query.NotWords {
					if i == 0 {
						sql += " ( "
					} else {
						sql += " AND "
					}
					if findWordUseLike {
						sql += fmt.Sprintf("LOWER(%s) NOT LIKE LOWER(?)", findWordTargetColumnName)
						*queryArgs = append(*queryArgs, "%"+notWord+"%")
					} else {
						sql += fmt.Sprintf("LOWER(%s) <> LOWER(?)", findWordTargetColumnName)
						*queryArgs = append(*queryArgs, notWord)
					}
					if i == len(*query.NotWords)-1 {
						sql += " ) "
					}
					*whereCounter++
				}

				if j == len(findWordTargetColumns)-1 {
					sql += " ) "
				}
			}
		}

		// ワード指定ありで検索対象列がない場合は全部false
		if ignoreFindWord && len(findWordTargetColumns) == 0 {
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += " 1 = 0 "
			*whereCounter++
		}
	}
	if *whereCounter == 0 {
		sql += " 0 = 0 "
	}
	*whereCounter++

	// 全部取得するのであればGROUP BYする前に返す
	if !onlyLatestData {
		if appendOrderBy {
			sql += fmt.Sprintf(" ORDER BY %s ", relatedTimeColumnName)
		}
		return sql, nil
	}

	// GROUP BY
	if appendGroupBy {
		groupByCounter := 0
		sql += " GROUP BY "

		// IDでGROUP BYする。
		if groupByCounter != 0 {
			sql += ", "
		}
		sql += " ID "
		groupByCounter++

		// HAVING
		havingCount := 0
		sql += " HAVING "

		// 最新のレコードのみ取得
		if havingCount != 0 {
			sql += " AND "
		}
		sql += " datetime(UPDATE_TIME, 'localtime') = MAX(datetime(UPDATE_TIME, 'localtime')) "
		havingCount++

		// 削除済みであるかどうかのSQL追記
		// Repをまたぐことがあるのでここでは判定しない
		// FindFilterで判定する
		/*
			isDeleted := false
			if query.IsDeleted != nil {
				isDeleted = *query.IsDeleted
			}
			if havingCount != 0 {
				sql += "AND "
			}
			sql += " IS_DELETED = ? "
			if isDeleted {
				*queryArgs = append(*queryArgs, true)
			} else {
				*queryArgs = append(*queryArgs, false)
			}
		*/
	}

	// ORDER BY
	if appendOrderBy {
		sql += fmt.Sprintf(" ORDER BY %s ", relatedTimeColumnName)
	}

	return sql, nil
}

func GenerateNewID() string {
	return uuid.New().String()
}
