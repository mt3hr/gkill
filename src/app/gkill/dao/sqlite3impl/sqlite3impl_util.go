package sqlite3impl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

const TimeLayout = "2006-01-02T15:04:05-07:00"

func EscapeSQLite(str string) string {
	return strings.ReplaceAll(str, "'", "''")
}

func GetSQLiteDBConnection(ctx context.Context, filename string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "file:"+filename+"?_timeout=6000&_synchronous=1&_journal=DELETE")
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	return db, err
}

func GenerateFindSQLCommon(query *find.FindQuery, tableName string, tableNameAlias string, whereCounter *int, onlyLatestData bool, relatedTimeColumnName string, findWordTargetColumns []string, findWordUseLike bool, ignoreFindWord bool, appendOrderBy bool, ignoreCase bool, queryArgs *[]interface{}) (string, error) {
	sql := ""

	// CASE無視（大文字小文字無視）の場合はLOWERをいれる
	lower := ""
	if ignoreCase {
		lower = "LOWER"
	}

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

	if useIDs {
		if len(ids) != 0 {
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
		} else {
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += " 0 = 1 "
		}
	}
	if *whereCounter == 0 {
		sql += " 0 = 0 "
	}
	*whereCounter++

	// ワードand検索である場合のSQL追記
	if query.UseWords != nil && *query.UseWords {
		// ワード指定ありで検索対象列がない場合は全部false
		if ignoreFindWord && len(findWordTargetColumns) == 0 {
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += " 1 = 0 "
			*whereCounter++
		} else {

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
								sql += fmt.Sprintf("%s(%s) LIKE %s(?)", lower, findWordTargetColumnName, lower)
								*queryArgs = append(*queryArgs, "%"+word+"%")
							} else {
								sql += fmt.Sprintf("%s(%s) = %s(?)", lower, findWordTargetColumnName, lower)
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
								sql += fmt.Sprintf("%s(%s) LIKE %s(?)", lower, findWordTargetColumnName, lower)
								*queryArgs = append(*queryArgs, "%"+word+"%")
							} else {
								sql += fmt.Sprintf("%s(%s) = %s(?)", lower, findWordTargetColumnName, lower)
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
							sql += fmt.Sprintf("%s(%s) NOT LIKE %s(?)", lower, findWordTargetColumnName, lower)
							*queryArgs = append(*queryArgs, "%"+notWord+"%")
						} else {
							sql += fmt.Sprintf("%s(%s) <> %s(?)", lower, findWordTargetColumnName, lower)
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
		}
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
			if strings.HasSuffix(relatedTimeColumnName, "_UNIX") { // UNIXついてればキャッシュでしょ（適当）
				if *whereCounter != 0 {
					sql += " AND "
				}
				sql += fmt.Sprintf("%s = ?", "UPDATE_TIME_UNIX")
				*queryArgs = append(*queryArgs, ((*query.UpdateTime).Unix()))
				*whereCounter++
			} else {
				if *whereCounter != 0 {
					sql += " AND "
				}
				sql += fmt.Sprintf("datetime(%s, 'localtime') = datetime(?, 'localtime')", "UPDATE_TIME")
				*queryArgs = append(*queryArgs, ((*query.UpdateTime).Format(TimeLayout)))
				*whereCounter++
			}
		}
	} else if useCalendar {
		// 開始日時を指定するSQLを追記
		if calendarStartDate != nil {
			if strings.HasSuffix(relatedTimeColumnName, "_UNIX") {
				if *whereCounter != 0 {
					sql += " AND "
				}
				sql += fmt.Sprintf("%s >= ?", relatedTimeColumnName)
				*queryArgs = append(*queryArgs, calendarStartDate.Unix())
				*whereCounter++
			} else {
				if *whereCounter != 0 {
					sql += " AND "
				}
				sql += fmt.Sprintf("datetime(%s, 'localtime') >= datetime(?, 'localtime')", relatedTimeColumnName)
				*queryArgs = append(*queryArgs, calendarStartDate.Format(TimeLayout))
				*whereCounter++
			}
		}

		// 終了日時を指定するSQLを追記
		if calendarEndDate != nil {
			if strings.HasSuffix(relatedTimeColumnName, "_UNIX") {
				if *whereCounter != 0 {
					sql += " AND "
				}
				sql += fmt.Sprintf("%s <= ?", relatedTimeColumnName)
				*queryArgs = append(*queryArgs, calendarEndDate.Unix())
				*whereCounter++
			} else {
				if *whereCounter != 0 {
					sql += " AND "
				}
				sql += fmt.Sprintf("datetime(%s, 'localtime') <= datetime(?, 'localtime')", relatedTimeColumnName)
				*queryArgs = append(*queryArgs, calendarEndDate.Format(TimeLayout))
				*whereCounter++
			}
		}
	}

	// 時間範囲指定ありの場合
	usePeriodOfTime := false
	var periodOfStartTimeSecond *int64
	var periodOfEndTimeSecond *int64
	if query.UsePeriodOfTime != nil {
		usePeriodOfTime = *query.UsePeriodOfTime
	}
	if query.PeriodOfTimeStartTimeSecond != nil {
		periodOfStartTimeSecond = query.PeriodOfTimeStartTimeSecond
	}
	if query.PeriodOfTimeEndTimeSecond != nil {
		periodOfEndTimeSecond = query.PeriodOfTimeEndTimeSecond
	}

	// 時間帯比較用
	timeExpr := ""
	if strings.HasSuffix(relatedTimeColumnName, "_UNIX") {
		timeExpr = "strftime('%H:%M:%S', datetime(" + relatedTimeColumnName + ", 'unixepoch', 'localtime'))"
	} else {
		timeExpr = "strftime('%H:%M:%S', datetime(" + relatedTimeColumnName + ", 'localtime'))"
	}
	argExpr := "strftime('%H:%M:%S', datetime(?, 'localtime'))"

	if usePeriodOfTime {
		// start/end を両方指定している場合は「ひとかたまり」で付ける
		if periodOfStartTimeSecond != nil && periodOfEndTimeSecond != nil {
			if *whereCounter != 0 {
				sql += " AND "
			}

			st := time.Unix(*periodOfStartTimeSecond, 0).In(time.Local)
			et := time.Unix(*periodOfEndTimeSecond, 0).In(time.Local)
			stSec := st.Hour()*3600 + st.Minute()*60 + st.Second()
			etSec := et.Hour()*3600 + et.Minute()*60 + et.Second()

			sql += " ( "
			sql += timeExpr + " >= " + argExpr
			*queryArgs = append(*queryArgs, st.Format(TimeLayout))

			if stSec > etSec {
				// 夜跨ぎ
				sql += " OR "
			} else {
				// 通常
				sql += " AND "
			}

			sql += timeExpr + " <= " + argExpr
			*queryArgs = append(*queryArgs, et.Format(TimeLayout))
			sql += " ) "

			*whereCounter++
		} else if periodOfStartTimeSecond != nil {
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += timeExpr + " >= " + argExpr
			*queryArgs = append(*queryArgs, time.Unix(*periodOfStartTimeSecond, 0).In(time.Local).Format(TimeLayout))
			*whereCounter++
		} else if periodOfEndTimeSecond != nil {
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += timeExpr + " <= " + argExpr
			*queryArgs = append(*queryArgs, time.Unix(*periodOfEndTimeSecond, 0).In(time.Local).Format(TimeLayout))
			*whereCounter++
		}

		// 曜日判定
		if len(query.PeriodOfTimeWeekOfDays) == 0 {
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += " 0 = 1 "
			*whereCounter++
		} else if len(query.PeriodOfTimeWeekOfDays) != 7 {
			weekExpr := ""
			if strings.HasSuffix(relatedTimeColumnName, "_UNIX") {
				weekExpr = "strftime('%w', datetime(" + relatedTimeColumnName + ", 'unixepoch', 'localtime'))"
			} else {
				weekExpr = "strftime('%w', datetime(" + relatedTimeColumnName + ", 'localtime'))"
			}

			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += " " + weekExpr + " IN ( "
			for i, w := range query.PeriodOfTimeWeekOfDays {
				sql += fmt.Sprintf("'%d'", w)
				if i != len(query.PeriodOfTimeWeekOfDays)-1 {
					sql += ", "
				}
			}
			sql += " ) "
			*whereCounter++
		}
	}

	// 最新のレコード判定
	if onlyLatestData {
		if strings.HasSuffix(relatedTimeColumnName, "_UNIX") { // UNIXついてればキャッシュでしょ（適当）
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += fmt.Sprintf(" UPDATE_TIME_UNIX = ( SELECT MAX(UPDATE_TIME_UNIX) FROM %s AS INNER_TABLE WHERE ID = %s.ID )", tableName, tableNameAlias)
			*whereCounter++
		} else {
			if *whereCounter != 0 {
				sql += " AND "
			}
			sql += fmt.Sprintf(" UPDATE_TIME = ( SELECT MAX(UPDATE_TIME) FROM %s AS INNER_TABLE WHERE ID = %s.ID )", tableName, tableNameAlias)
			*whereCounter++
		}
	}

	// 削除済みであるかどうかのSQL追記
	// Repをまたぐことがあるのでここでは判定しない
	// FindFilterで判定する

	if *whereCounter == 0 {
		sql += " 0 = 0 "
	}

	// ORDER BY
	if appendOrderBy {
		sql += fmt.Sprintf(" ORDER BY %s DESC ", relatedTimeColumnName)
	}

	return sql, nil
}

func GenerateNewID() string {
	return uuid.New().String()
}
