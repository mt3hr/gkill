package sqlite3impl

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"

	"database/sql"

	_ "modernc.org/sqlite"
)

const TimeLayout = "2006-01-02T15:04:05-07:00"

// sensitiveLogColumns はTraceSQLログで値をマスクする機密カラム名です。
var sensitiveLogColumns = map[string]struct{}{
	"GOOGLE_MAP_API_KEY":             {},
	"GKILL_NOTIFICATION_PRIVATE_KEY": {},
	"PASSWORD_SHA256":                {},
	"PASSWORD_HASH":                  {},
	"PASSWORD_RESET_TOKEN":           {},
}

// MaskSensitiveValueForLog は機密カラムの値をログ用に "***" へ置き換えます。
// SQLの実行引数には使わず、TraceSQLログ出力にのみ使うこと。
func MaskSensitiveValueForLog(key string, value any) any {
	if _, ok := sensitiveLogColumns[key]; ok {
		return "***"
	}
	return value
}

func EscapeSQLite(str string) string {
	return strings.ReplaceAll(str, "'", "''")
}

// sqliteDataDSNParams は実データDBを開くときのPRAGMAです。
//
// journal_mode は DELETE のまま変えないこと。
// WALにすると -wal / -shm のサイドカーができ、.db 単体をコピーしても
// 未チェックポイントの内容が落ちる。gkillはrepを端末ごとの .db ファイルとして
// 持ち回る作りで、バックアップも単純なファイルコピーで済ませたいので、
// ここは意図的にWALを使わない。（キャッシュ側は別で、そちらはWALを使っている）
//
// synchronous も NORMAL のまま変えない。耐久性の意味が変わるため。
//
// cache_size / temp_store / mmap_size は未設定だったので足した。
//   - cache_size: 既定は -2000 (2MB)。接続ごとの上限で、実体ファイル数ぶん
//     積み上がりうるので控えめに 8MB にしてある
//   - temp_store: 既定は FILE。ORDER BY の一時ソートがディスクに落ちるのを防ぐ
//   - mmap_size: 大きいDBで効果が大きい。実測(90MBのURLog.dbを全走査)で
//     156ms -> 5.6ms。小さいDBでは差が出ない。
//     ページをOSキャッシュからSQLiteのバッファへ複写しなくて済むため。
const sqliteDataDSNParams = "?_pragma=busy_timeout(6000)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=journal_mode(DELETE)" +
	"&_pragma=cache_size(-8000)" +
	"&_pragma=temp_store(MEMORY)" +
	"&_pragma=mmap_size(268435456)"

func GetSQLiteDBConnection(ctx context.Context, filename string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+filename+sqliteDataDSNParams)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	// 既定の MaxIdleConns は2しかなく、同時2本を超えた接続は使い終わると即閉じられる。
	// そのたびにSQLiteのページキャッシュが捨てられるので、
	// 同時に開ける本数と同じだけ保持させる。
	db.SetMaxOpenConns(runtime.NumCPU())
	db.SetMaxIdleConns(runtime.NumCPU())
	db.SetConnMaxLifetime(0)
	return db, err
}

// GenerateFindSQLCommon は全リポジトリ共通の検索WHERE句（と必要ならORDER BY）を組み立てます。
//
// SQLの組み立てには strings.Builder を使うこと。
// ここは `ID IN (?, ?, ...)` を検索対象repの全ID数ぶん展開するため、
// `sql += ...` のループだと連結のたびに全体をコピーしてO(n^2)になります。
// 実測で1,122件=6.5ms / 7,047件=102ms / 10万件=27秒かかっていました。
func GenerateFindSQLCommon(query *find.FindQuery, tableName string, tableNameAlias string, whereCounter *int, onlyLatestData bool, relatedTimeColumnName string, findWordTargetColumns []string, findWordUseLike bool, ignoreFindWord bool, appendOrderBy bool, ignoreCase bool, queryArgs *[]any) (string, error) {
	sqlBuilder := &strings.Builder{}
	// ID列挙とワード条件で膨らむぶんを概算で先に確保しておく
	sqlBuilder.Grow(256 + len(query.IDs)*5 + len(findWordTargetColumns)*(len(query.Words)+len(query.NotWords))*96)

	// CASE無視（大文字小文字無視）の場合はLOWERをいれる
	lower := ""
	if ignoreCase {
		lower = "LOWER"
	}

	// WHERE
	// id検索である場合のSQL追記
	useIDs := false
	ids := []string{}
	useIDs = query.UseIDs
	ids = query.IDs

	if useIDs {
		if len(ids) != 0 {
			if *whereCounter != 0 {
				sqlBuilder.WriteString(" AND ")
			}
			sqlBuilder.WriteString(" ID IN (")
			for i, id := range ids {
				sqlBuilder.WriteString(" ? ")
				*queryArgs = append(*queryArgs, id)
				if i != len(ids)-1 {
					sqlBuilder.WriteString(", ")
				}
				*whereCounter++
			}
			sqlBuilder.WriteString(")")
		} else {
			if *whereCounter != 0 {
				sqlBuilder.WriteString(" AND ")
			}
			sqlBuilder.WriteString(" 0 = 1 ")
			// ここでカウンタを進めないと、直後の「条件が1つも無ければ 0 = 0 」が
			// 区切りなしで続いて " 0 = 1  0 = 0 " という構文エラーになる。
			// 空のrep（最新版アドレスが1件も無いrep）を検索すると必ず踏む。
			*whereCounter++
		}
	}
	if *whereCounter == 0 {
		sqlBuilder.WriteString(" 0 = 0 ")
	}
	*whereCounter++

	// ワードand検索である場合のSQL追記
	if query.UseWords {
		// ワード指定ありで検索対象列がない場合は全部false
		if ignoreFindWord && len(findWordTargetColumns) == 0 {
			if *whereCounter != 0 {
				sqlBuilder.WriteString(" AND ")
			}
			sqlBuilder.WriteString(" 1 = 0 ")
			*whereCounter++
		} else {
			if len(query.Words) != 0 {
				if query.WordsAnd {
					if *whereCounter != 0 {
						sqlBuilder.WriteString(" AND ")
					}
					for j, findWordTargetColumnName := range findWordTargetColumns {
						if j == 0 {
							sqlBuilder.WriteString(" ( ")
						} else {
							sqlBuilder.WriteString(" AND ")
						}

						for i, word := range query.Words {
							if i == 0 {
								sqlBuilder.WriteString(" ( ")
							} else {
								sqlBuilder.WriteString(" AND ")
							}
							if findWordUseLike {
								fmt.Fprintf(sqlBuilder, "%s(%s) LIKE %s(?)", lower, findWordTargetColumnName, lower)
								*queryArgs = append(*queryArgs, "%"+word+"%")

								sqlBuilder.WriteString(" OR ")

								fmt.Fprintf(sqlBuilder, "%s(%s) LIKE %s(?)", lower, "ID", lower)
								*queryArgs = append(*queryArgs, "%"+word+"%")
							} else {
								fmt.Fprintf(sqlBuilder, "%s(%s) = %s(?)", lower, findWordTargetColumnName, lower)
								*queryArgs = append(*queryArgs, "%"+word+"%")

								sqlBuilder.WriteString(" OR ")

								fmt.Fprintf(sqlBuilder, "%s(%s) = %s(?)", lower, "ID", lower)
								*queryArgs = append(*queryArgs, "%"+word+"%")

							}
							if i == len(query.Words)-1 {
								sqlBuilder.WriteString(" ) ")
							}
							*whereCounter++
						}

						if j == len(findWordTargetColumns)-1 {
							sqlBuilder.WriteString(" ) ")
						}
					}
				} else {
					// ワードor検索である場合のSQL追記
					if *whereCounter != 0 {
						sqlBuilder.WriteString(" AND ")
					}
					for j, findWordTargetColumnName := range findWordTargetColumns {
						if j == 0 {
							sqlBuilder.WriteString(" ( ")
						} else {
							sqlBuilder.WriteString(" OR ")
						}

						for i, word := range query.Words {
							if i == 0 {
								sqlBuilder.WriteString(" ( ")
							} else {
								sqlBuilder.WriteString(" OR ")
							}
							if findWordUseLike {
								fmt.Fprintf(sqlBuilder, "%s(%s) LIKE %s(?)", lower, findWordTargetColumnName, lower)
								*queryArgs = append(*queryArgs, "%"+word+"%")

								sqlBuilder.WriteString(" OR ")

								fmt.Fprintf(sqlBuilder, "%s(%s) LIKE %s(?)", lower, "ID", lower)
								*queryArgs = append(*queryArgs, "%"+word+"%")
							} else {
								fmt.Fprintf(sqlBuilder, "%s(%s) = %s(?)", lower, findWordTargetColumnName, lower)
								*queryArgs = append(*queryArgs, word)

								sqlBuilder.WriteString(" OR ")

								fmt.Fprintf(sqlBuilder, "%s(%s) = %s(?)", lower, "ID", lower)
								*queryArgs = append(*queryArgs, "%"+word+"%")
							}
							if i == len(query.Words)-1 {
								sqlBuilder.WriteString(" ) ")
							}
							*whereCounter++
						}

						if j == len(findWordTargetColumns)-1 {
							sqlBuilder.WriteString(" ) ")
						}
					}
				}
			}

			if len(query.NotWords) != 0 {
				// notワードを除外するSQLを追記
				if *whereCounter != 0 {
					sqlBuilder.WriteString(" AND ")
				}
				for j, findWordTargetColumnName := range findWordTargetColumns {
					if j == 0 {
						sqlBuilder.WriteString(" ( ")
					} else {
						sqlBuilder.WriteString(" AND ")
					}

					for i, notWord := range query.NotWords {
						if i == 0 {
							sqlBuilder.WriteString(" ( ")
						} else {
							sqlBuilder.WriteString(" AND ")
						}
						// 肯定側は「対象列かIDのどちらかに一致」なのでORでよいが、
						// 否定側はド・モルガンによりANDでなければならない。
						//   NOT(COL LIKE ? OR ID LIKE ?) = COL NOT LIKE ? AND ID NOT LIKE ?
						// ここがORだったころは、IDがUUIDで検索語を含むことは実質ないため
						// 右辺が常に真になり、除外がまったく効いていなかった。
						if findWordUseLike {
							fmt.Fprintf(sqlBuilder, "%s(%s) NOT LIKE %s(?)", lower, findWordTargetColumnName, lower)
							*queryArgs = append(*queryArgs, "%"+notWord+"%")

							sqlBuilder.WriteString(" AND ")

							fmt.Fprintf(sqlBuilder, "%s(%s) NOT LIKE %s(?)", lower, "ID", lower)
							*queryArgs = append(*queryArgs, "%"+notWord+"%")
						} else {
							fmt.Fprintf(sqlBuilder, "%s(%s) <> %s(?)", lower, findWordTargetColumnName, lower)
							*queryArgs = append(*queryArgs, notWord)

							sqlBuilder.WriteString(" AND ")

							fmt.Fprintf(sqlBuilder, "%s(%s) <> %s(?)", lower, "ID", lower)
							*queryArgs = append(*queryArgs, notWord)
						}
						if i == len(query.NotWords)-1 {
							sqlBuilder.WriteString(" ) ")
						}
						*whereCounter++
					}

					if j == len(findWordTargetColumns)-1 {
						sqlBuilder.WriteString(" ) ")
					}
				}
			}
		}
	}

	// 日付範囲指定ありの場合
	useCalendar := false
	var calendarStartDate *time.Time
	var calendarEndDate *time.Time
	useCalendar = query.UseCalendar
	if query.CalendarStartDate != nil {
		calendarStartDate = query.CalendarStartDate
	}
	if query.CalendarEndDate != nil {
		calendarEndDate = query.CalendarEndDate
	}

	// UPDATE_TIMEか、Calendarの条件をSQLに追記
	if query.UseUpdateTime {
		if strings.HasSuffix(relatedTimeColumnName, "_UNIX") { // UNIXついてればキャッシュでしょ（適当）
			if *whereCounter != 0 {
				sqlBuilder.WriteString(" AND ")
			}
			fmt.Fprintf(sqlBuilder, "%s = ?", "UPDATE_TIME_UNIX")
			*queryArgs = append(*queryArgs, ((query.UpdateTime).Unix()))
			*whereCounter++
		} else {
			if *whereCounter != 0 {
				sqlBuilder.WriteString(" AND ")
			}
			fmt.Fprintf(sqlBuilder, "unixepoch(%s) = unixepoch(?)", "UPDATE_TIME")
			*queryArgs = append(*queryArgs, ((query.UpdateTime).Format(TimeLayout)))
			*whereCounter++
		}
	} else if useCalendar {
		// 開始日時を指定するSQLを追記
		if calendarStartDate != nil {
			if strings.HasSuffix(relatedTimeColumnName, "_UNIX") {
				if *whereCounter != 0 {
					sqlBuilder.WriteString(" AND ")
				}
				fmt.Fprintf(sqlBuilder, "%s >= ?", relatedTimeColumnName)
				*queryArgs = append(*queryArgs, calendarStartDate.Unix())
				*whereCounter++
			} else {
				if *whereCounter != 0 {
					sqlBuilder.WriteString(" AND ")
				}
				fmt.Fprintf(sqlBuilder, "unixepoch(%s) >= unixepoch(?)", relatedTimeColumnName)
				*queryArgs = append(*queryArgs, calendarStartDate.Format(TimeLayout))
				*whereCounter++
			}
		}

		// 終了日時を指定するSQLを追記
		if calendarEndDate != nil {
			if strings.HasSuffix(relatedTimeColumnName, "_UNIX") {
				if *whereCounter != 0 {
					sqlBuilder.WriteString(" AND ")
				}
				fmt.Fprintf(sqlBuilder, "%s <= ?", relatedTimeColumnName)
				*queryArgs = append(*queryArgs, calendarEndDate.Unix())
				*whereCounter++
			} else {
				if *whereCounter != 0 {
					sqlBuilder.WriteString(" AND ")
				}
				fmt.Fprintf(sqlBuilder, "unixepoch(%s) <= unixepoch(?)", relatedTimeColumnName)
				*queryArgs = append(*queryArgs, calendarEndDate.Format(TimeLayout))
				*whereCounter++
			}
		}
	}

	// 時間範囲指定ありの場合
	usePeriodOfTime := false
	var periodOfStartTimeSecond *int64
	var periodOfEndTimeSecond *int64
	usePeriodOfTime = query.UsePeriodOfTime
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
				sqlBuilder.WriteString(" AND ")
			}

			st := time.Unix(*periodOfStartTimeSecond, 0).In(time.Local)
			et := time.Unix(*periodOfEndTimeSecond, 0).In(time.Local)
			stSec := st.Hour()*3600 + st.Minute()*60 + st.Second()
			etSec := et.Hour()*3600 + et.Minute()*60 + et.Second()

			sqlBuilder.WriteString(" ( ")
			sqlBuilder.WriteString(timeExpr + " >= " + argExpr)
			*queryArgs = append(*queryArgs, st.Format(TimeLayout))

			if stSec > etSec {
				// 夜跨ぎ
				sqlBuilder.WriteString(" OR ")
			} else {
				// 通常
				sqlBuilder.WriteString(" AND ")
			}

			sqlBuilder.WriteString(timeExpr + " <= " + argExpr)
			*queryArgs = append(*queryArgs, et.Format(TimeLayout))
			sqlBuilder.WriteString(" ) ")

			*whereCounter++
		} else if periodOfStartTimeSecond != nil {
			if *whereCounter != 0 {
				sqlBuilder.WriteString(" AND ")
			}
			sqlBuilder.WriteString(timeExpr + " >= " + argExpr)
			*queryArgs = append(*queryArgs, time.Unix(*periodOfStartTimeSecond, 0).In(time.Local).Format(TimeLayout))
			*whereCounter++
		} else if periodOfEndTimeSecond != nil {
			if *whereCounter != 0 {
				sqlBuilder.WriteString(" AND ")
			}
			sqlBuilder.WriteString(timeExpr + " <= " + argExpr)
			*queryArgs = append(*queryArgs, time.Unix(*periodOfEndTimeSecond, 0).In(time.Local).Format(TimeLayout))
			*whereCounter++
		}

		// 曜日判定
		if len(query.PeriodOfTimeWeekOfDays) == 0 {
			if *whereCounter != 0 {
				sqlBuilder.WriteString(" AND ")
			}
			sqlBuilder.WriteString(" 0 = 1 ")
			*whereCounter++
		} else if len(query.PeriodOfTimeWeekOfDays) != 7 {
			weekExpr := ""
			if strings.HasSuffix(relatedTimeColumnName, "_UNIX") {
				weekExpr = "strftime('%w', datetime(" + relatedTimeColumnName + ", 'unixepoch', 'localtime'))"
			} else {
				weekExpr = "strftime('%w', datetime(" + relatedTimeColumnName + ", 'localtime'))"
			}

			if *whereCounter != 0 {
				sqlBuilder.WriteString(" AND ")
			}
			sqlBuilder.WriteString(" " + weekExpr + " IN ( ")
			for i, w := range query.PeriodOfTimeWeekOfDays {
				fmt.Fprintf(sqlBuilder, "'%d'", w)
				if i != len(query.PeriodOfTimeWeekOfDays)-1 {
					sqlBuilder.WriteString(", ")
				}
			}
			sqlBuilder.WriteString(" ) ")
			*whereCounter++
		}
	}

	// 最新のレコード判定
	if onlyLatestData {
		if strings.HasSuffix(relatedTimeColumnName, "_UNIX") { // UNIXついてればキャッシュでしょ（適当）
			if *whereCounter != 0 {
				sqlBuilder.WriteString(" AND ")
			}
			fmt.Fprintf(sqlBuilder, " UPDATE_TIME_UNIX = ( SELECT MAX(UPDATE_TIME_UNIX) FROM %s AS INNER_TABLE WHERE ID = %s.ID )", tableName, tableNameAlias)
			*whereCounter++
		} else {
			if *whereCounter != 0 {
				sqlBuilder.WriteString(" AND ")
			}
			fmt.Fprintf(sqlBuilder, " UPDATE_TIME = ( SELECT UPDATE_TIME FROM %s AS INNER_TABLE WHERE INNER_TABLE.ID = %s.ID ORDER BY datetime(INNER_TABLE.UPDATE_TIME) DESC LIMIT 1 )", tableName, tableNameAlias)
			*whereCounter++
		}
	}

	// 削除済みであるかどうかのSQL追記
	// Repをまたぐことがあるのでここでは判定しない
	// FindFilterで判定する

	if *whereCounter == 0 {
		sqlBuilder.WriteString(" 0 = 0 ")
	}

	// ORDER BY
	//
	// 文字列の時刻列は unixepoch() で並べる。
	// RELATED_TIME はオフセット付きRFC3339で、実データにも +00:00 と +09:00 が
	// 混在しているため、文字列のまま並べると時系列にならない。
	// またWHERE側と式を揃えることで unixepoch(列) の式インデックスが
	// 並び替えにも使われ、一時ソートが不要になる。
	if appendOrderBy {
		if strings.HasSuffix(relatedTimeColumnName, "_UNIX") {
			fmt.Fprintf(sqlBuilder, " ORDER BY %s DESC ", relatedTimeColumnName)
		} else {
			fmt.Fprintf(sqlBuilder, " ORDER BY unixepoch(%s) DESC ", relatedTimeColumnName)
		}
	}

	return sqlBuilder.String(), nil
}

// EnsureUnixepochIndex は文字列の時刻列に対する式インデックスを作成します。
//
// 時刻列はRFC3339の文字列で入っており、実データでもオフセットが混在しています
// (TAG.RELATED_TIME は +00:00 が6,194行 / +09:00 が853行)。
// そのため範囲比較も並び替えも unixepoch() を通す必要がありますが、
// 列に関数を適用すると通常のインデックスは一切使われません。
// 式そのものにインデックスを張ることで SCAN が SEARCH になります。
//
// GenerateFindSQLCommon が生成する式と完全に一致していなければ使われません。
// CAST を挟む・'auto' を足す・strftime('%s',...) で書くといった些細な違いでも
// エラーにならず黙って全走査に戻るので、SQL側を変えるときは必ず
// EXPLAIN QUERY PLAN に SEARCH が出ることを確認してください。
//
// なお 'localtime' 修飾子は非決定的とみなされ、式インデックスには使えません。
func EnsureUnixepochIndex(ctx context.Context, db *sql.DB, tableName string, timeColumnNames ...string) error {
	for _, timeColumnName := range timeColumnNames {
		indexName := "INDEX_" + tableName + "_" + timeColumnName + "_UNIXEPOCH"
		indexSQL := fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s ON %s (unixepoch(%s) DESC);",
			QuoteIdent(indexName), QuoteIdent(tableName), timeColumnName,
		)
		slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
		if _, err := db.ExecContext(ctx, indexSQL); err != nil {
			return fmt.Errorf("error at create unixepoch index %s on %s: %w", indexName, tableName, err)
		}
	}
	return nil
}

// EnsureUnixColumnIndex はキャッシュ側の _UNIX 整数列に索引を作成します。
//
// キャッシュ表の既存索引は (ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX) と
// ID が先頭なので、時刻範囲の絞り込みにも ORDER BY にも使えません。
// 時刻列を先頭にした索引を別途張ります。
func EnsureUnixColumnIndex(ctx context.Context, db *sql.DB, tableName string, columnNames ...string) error {
	for _, columnName := range columnNames {
		indexName := "INDEX_" + tableName + "_" + columnName + "_ONLY"
		indexSQL := fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s ON %s (%s DESC);",
			QuoteIdent(indexName), QuoteIdent(tableName), columnName,
		)
		slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
		if _, err := db.ExecContext(ctx, indexSQL); err != nil {
			return fmt.Errorf("error at create unix column index %s on %s: %w", indexName, tableName, err)
		}
	}
	return nil
}

// EnsureTxIDIndex はトランザクション用一時表に (TX_ID, USER_ID, DEVICE) の索引を作成します。
//
// 一時表への問い合わせは13種すべてが
// `WHERE TX_ID = ? AND USER_ID = ? AND DEVICE = ?` の形をしているのに、
// 既存の索引はどれも ID か TARGET_ID が先頭で TX_ID を含んでいませんでした。
// commit_tx は1コミットにつき13repへ Get...ByTXID と DeleteByTXID を呼ぶので、
// 1コミットあたり最低26回の全表スキャンになります。
//
// 一時表は利用者単位の共有DBで、複数デバイス・複数トランザクションの行が同居するため、
// 行数に比例して劣化します。
func EnsureTxIDIndex(ctx context.Context, db *sql.DB, tableName string) error {
	indexName := "INDEX_" + tableName + "_TX_ID"
	indexSQL := fmt.Sprintf(
		"CREATE INDEX IF NOT EXISTS %s ON %s (TX_ID, USER_ID, DEVICE);",
		QuoteIdent(indexName), QuoteIdent(tableName),
	)
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	if _, err := db.ExecContext(ctx, indexSQL); err != nil {
		return fmt.Errorf("error at create tx id index %s on %s: %w", indexName, tableName, err)
	}
	return nil
}

func GenerateNewID() string {
	return uuid.New().String()
}

func DeleteAllIndex(db *sql.DB) error {
	rows, err := db.Query(`
SELECT name
FROM sqlite_master
WHERE type = 'index'
  AND name NOT LIKE 'sqlite_%'
  AND sql IS NOT NULL
ORDER BY name;
`)
	if err != nil {
		return fmt.Errorf("query indexes: %w", err)
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return fmt.Errorf("scan index name: %w", err)
		}
		names = append(names, n)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate indexes: %w", err)
	}

	if len(names) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
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

	if _, err := tx.Exec(`PRAGMA foreign_keys=OFF;`); err != nil {
		return fmt.Errorf("pragma foreign_keys: %w", err)
	}
	if _, err := tx.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		return fmt.Errorf("pragma busy_timeout: %w", err)
	}

	for _, n := range names {
		stmt := fmt.Sprintf(`DROP INDEX IF EXISTS %s;`, QuoteIdent(n))
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("drop index %q: %w", n, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	isCommitted = true
	return nil
}

func Optimize(db *sql.DB) error {
	if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		return fmt.Errorf("pragma busy_timeout: %w", err)
	}

	// REINDEX and ANALYZE can be run normally
	if _, err := db.Exec(`REINDEX;`); err != nil {
		return fmt.Errorf("REINDEX: %w", err)
	}
	if _, err := db.Exec(`ANALYZE;`); err != nil {
		return fmt.Errorf("ANALYZE: %w", err)
	}
	if _, err := db.Exec(`PRAGMA optimize;`); err != nil {
		return fmt.Errorf("PRAGMA optimize: %w", err)
	}

	// VACUUM should be outside any transaction
	if _, err := db.Exec(`VACUUM;`); err != nil {
		return fmt.Errorf("VACUUM: %w", err)
	}
	return nil
}

func QuoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
