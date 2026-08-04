package reps

import (
	"context"
	sqllib "database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
)

// MiReKyouのSQL組み立てとスキャンを担う共通処理です。
// 実体(MIREKYOUテーブル)とキャッシュ(インメモリテーブル)で
// 列名・値の形式を揃えているため、テーブル名だけ差し替えて同じSQLを使えます。

// miReKyouColumns はMiReKyouテーブルの列定義です。実体とキャッシュで共通です。
const miReKyouColumns = `
  IS_DELETED NOT NULL,
  ID NOT NULL,
  TARGET_ID NOT NULL,
  IS_CHECKED NOT NULL,
  BOARD_NAME NOT NULL,
  LIMIT_TIME,
  ESTIMATE_START_TIME,
  ESTIMATE_END_TIME,
  CREATE_TIME NOT NULL,
  CREATE_APP NOT NULL,
  CREATE_USER NOT NULL,
  CREATE_DEVICE NOT NULL,
  UPDATE_TIME NOT NULL,
  UPDATE_APP NOT NULL,
  UPDATE_DEVICE NOT NULL,
  UPDATE_USER NOT NULL
`

// miReKyouInsertColumnNames はINSERT対象の列名です。
const miReKyouInsertColumnNames = `
  IS_DELETED,
  ID,
  TARGET_ID,
  IS_CHECKED,
  BOARD_NAME,
  LIMIT_TIME,
  ESTIMATE_START_TIME,
  ESTIMATE_END_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_USER,
  CREATE_DEVICE,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER
`

// miReKyouInsertPlaceHolders はINSERTのプレースホルダです。
const miReKyouInsertPlaceHolders = `?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?`

// miReKyouSelectColumns はMiReKyou取得用のSELECT列です。末尾にDATA_TYPEが続きます。
const miReKyouSelectColumns = `
  IS_DELETED,
  ID,
  TARGET_ID,
  IS_CHECKED,
  BOARD_NAME,
  LIMIT_TIME,
  ESTIMATE_START_TIME,
  ESTIMATE_END_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
`

// miReKyouKyouSelectColumns はKyou取得用のSELECT列です。RELATED_TIMEは射影ごとに差し替えます。
const miReKyouKyouSelectColumns = `
  IS_DELETED,
  ID,
  %s AS RELATED_TIME,
  CREATE_TIME,
  CREATE_APP,
  CREATE_DEVICE,
  CREATE_USER,
  UPDATE_TIME,
  UPDATE_APP,
  UPDATE_DEVICE,
  UPDATE_USER,
  ? AS REP_NAME,
  '%s' AS DATA_TYPE
`

// miReKyouProjection はMiReKyouの5射影の定義です。
// Miと同じく、作成/チェック/期限/開始/終了の5つの時刻でKyouを射影します。
type miReKyouProjection struct {
	// dataType はKyouのDataTypeです。
	dataType string
	// relatedTimeColumn はRELATED_TIMEに使う列です。
	relatedTimeColumn string
	// notNullColumn は射影の対象になる条件(列がNULLでないこと)を表す列です。
	notNullColumn string
	// isInclude はFindQueryで射影が有効かどうかを返します。
	isInclude func(query *find.FindQuery) bool
}

var miReKyouProjections = []miReKyouProjection{
	{
		dataType:          "mirekyou_create",
		relatedTimeColumn: "CREATE_TIME",
		notNullColumn:     "CREATE_TIME",
		isInclude:         func(query *find.FindQuery) bool { return query.IncludeCreateMi },
	},
	{
		dataType:          "mirekyou_check",
		relatedTimeColumn: "UPDATE_TIME",
		notNullColumn:     "IS_CHECKED",
		isInclude:         func(query *find.FindQuery) bool { return query.IncludeCheckMi },
	},
	{
		dataType:          "mirekyou_limit",
		relatedTimeColumn: "LIMIT_TIME",
		notNullColumn:     "LIMIT_TIME",
		isInclude:         func(query *find.FindQuery) bool { return query.IncludeLimitMi },
	},
	{
		dataType:          "mirekyou_start",
		relatedTimeColumn: "ESTIMATE_START_TIME",
		notNullColumn:     "ESTIMATE_START_TIME",
		isInclude:         func(query *find.FindQuery) bool { return query.IncludeStartMi },
	},
	{
		dataType:          "mirekyou_end",
		relatedTimeColumn: "ESTIMATE_END_TIME",
		notNullColumn:     "ESTIMATE_END_TIME",
		isInclude:         func(query *find.FindQuery) bool { return query.IncludeEndMi },
	},
}

// generateMiReKyouWhere は射影ごとのWHERE句とパラメータを組み立てます。
// MiReKyouはタイトルを持たないためワード検索はSQLでは行いません。
// (検索対象列が空のままUseWordsをGenerateFindSQLCommonへ渡すと "1 = 0" になり全件落ちるため、
//
//	ワード条件を落としたクエリでSQLを組み立て、ワード判定はターゲットKyou側へ委譲します)
func generateMiReKyouWhere(query *find.FindQuery, projection miReKyouProjection, onlyLatestData bool, repName string, tableName string) (string, []any, error) {
	queryWithoutWords := *query
	queryWithoutWords.UseWords = false
	queryWithoutWords.Words = nil
	queryWithoutWords.NotWords = nil

	queryArgs := []any{repName}
	whereCounter := 0
	findWordTargetColumns := []string{}
	ignoreFindWord := true
	appendOrderBy := false
	findWordUseLike := true
	ignoreCase := true
	whereSQL, err := sqlite3impl.GenerateFindSQLCommon(&queryWithoutWords, tableName, tableName, &whereCounter, onlyLatestData, projection.relatedTimeColumn, findWordTargetColumns, findWordUseLike, ignoreFindWord, appendOrderBy, ignoreCase, &queryArgs)
	if err != nil {
		return "", nil, err
	}
	whereSQL = projection.notNullColumn + " IS NOT NULL AND " + whereSQL
	if query.UseMiBoardName {
		whereSQL += " AND "
		whereSQL += " BOARD_NAME = ? "
		queryArgs = append(queryArgs, query.MiBoardName)
	}
	return whereSQL, queryArgs, nil
}

// resolveMiReKyouOnlyLatest は射影ごとの「最新のみ」指定を決めます。
// onlyLatestOverride が非nilならすべての射影でその値を使い、
// nilなら検索時の既定(作成射影はquery.OnlyLatestData、他は常に最新)に従います。
func resolveMiReKyouOnlyLatest(query *find.FindQuery, projection miReKyouProjection, onlyLatestOverride *bool) bool {
	if onlyLatestOverride != nil {
		return *onlyLatestOverride
	}
	if projection.dataType != "mirekyou_create" {
		return true
	}
	return query.OnlyLatestData
}

// buildMiReKyouKyouSQL は5射影のKyou取得SQLを組み立てます。
// filterProjections がtrueの場合はFindQueryのIncludeXxxMiで射影を絞ります。
func buildMiReKyouKyouSQL(query *find.FindQuery, repName string, tableName string, filterProjections bool, onlyLatestOverride *bool) (string, []any, error) {
	sqlSegments := []string{}
	queryArgs := []any{}
	for _, projection := range miReKyouProjections {
		if filterProjections && !projection.isInclude(query) {
			continue
		}
		onlyLatestData := resolveMiReKyouOnlyLatest(query, projection, onlyLatestOverride)
		whereSQL, args, err := generateMiReKyouWhere(query, projection, onlyLatestData, repName, tableName)
		if err != nil {
			return "", nil, err
		}
		selectSQL := fmt.Sprintf(miReKyouKyouSelectColumns, projection.relatedTimeColumn, projection.dataType)
		sqlSegments = append(sqlSegments, "SELECT "+selectSQL+" FROM "+tableName+" WHERE "+whereSQL)
		queryArgs = append(queryArgs, args...)
	}
	if len(sqlSegments) == 0 {
		return "", nil, nil
	}
	return strings.Join(sqlSegments, " UNION "), queryArgs, nil
}

// buildMiReKyouSQL は5射影のMiReKyou取得SQLを組み立てます。
func buildMiReKyouSQL(query *find.FindQuery, repName string, tableName string, filterProjections bool, onlyLatestOverride *bool) (string, []any, error) {
	sqlSegments := []string{}
	queryArgs := []any{}
	for _, projection := range miReKyouProjections {
		if filterProjections && !projection.isInclude(query) {
			continue
		}
		onlyLatestData := resolveMiReKyouOnlyLatest(query, projection, onlyLatestOverride)
		whereSQL, args, err := generateMiReKyouWhere(query, projection, onlyLatestData, repName, tableName)
		if err != nil {
			return "", nil, err
		}
		selectSQL := "SELECT " + miReKyouSelectColumns + "  '" + projection.dataType + "' AS DATA_TYPE FROM " + tableName
		sqlSegments = append(sqlSegments, selectSQL+" WHERE "+whereSQL)
		queryArgs = append(queryArgs, args...)
	}
	if len(sqlSegments) == 0 {
		return "", nil, nil
	}
	return strings.Join(sqlSegments, " UNION "), queryArgs, nil
}

// buildMiReKyouSingleProjectionSQL は作成射影のみでMiReKyouを取得するSQLを組み立てます。
// ID指定の取得など、射影を分ける必要がない場面で使います。
func buildMiReKyouSingleProjectionSQL(query *find.FindQuery, repName string, tableName string, onlyLatestData bool) (string, []any, error) {
	projection := miReKyouProjections[0]
	whereSQL, queryArgs, err := generateMiReKyouWhere(query, projection, onlyLatestData, repName, tableName)
	if err != nil {
		return "", nil, err
	}
	sql := "SELECT " + miReKyouSelectColumns + "  '" + projection.dataType + "' AS DATA_TYPE FROM " + tableName + " WHERE " + whereSQL
	return sql, queryArgs, nil
}

// scanMiReKyouKyous はKyou取得SQLの結果を読み取ります。
func scanMiReKyouKyous(ctx context.Context, rows *sqllib.Rows, repName string) ([]Kyou, error) {
	kyous := []Kyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kyou := Kyou{}
			kyou.RepName = repName
			relatedTimeStr, createTimeStr, updateTimeStr := "", "", ""

			err := rows.Scan(
				&kyou.IsDeleted,
				&kyou.ID,
				&relatedTimeStr,
				&createTimeStr,
				&kyou.CreateApp,
				&kyou.CreateDevice,
				&kyou.CreateUser,
				&updateTimeStr,
				&kyou.UpdateApp,
				&kyou.UpdateDevice,
				&kyou.UpdateUser,
				&kyou.RepName,
				&kyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan mirekyou: %w", err)
				return nil, err
			}

			kyou.RelatedTime, err = time.Parse(sqlite3impl.TimeLayout, relatedTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse related time %s in MIREKYOU: %w", relatedTimeStr, err)
				return nil, err
			}
			kyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in MIREKYOU: %w", createTimeStr, err)
				return nil, err
			}
			kyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in MIREKYOU: %w", updateTimeStr, err)
				return nil, err
			}
			kyous = append(kyous, kyou)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return kyous, nil
}

// scanMiReKyous はMiReKyou取得SQLの結果を読み取ります。
func scanMiReKyous(ctx context.Context, rows *sqllib.Rows, repName string) ([]MiReKyou, error) {
	mirekyous := []MiReKyou{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			mirekyou := MiReKyou{}
			mirekyou.RepName = repName
			createTimeStr, updateTimeStr := "", ""
			limitTime, estimateStartTime, estimateEndTime := sqllib.NullString{}, sqllib.NullString{}, sqllib.NullString{}

			err := rows.Scan(
				&mirekyou.IsDeleted,
				&mirekyou.ID,
				&mirekyou.TargetID,
				&mirekyou.IsChecked,
				&mirekyou.BoardName,
				&limitTime,
				&estimateStartTime,
				&estimateEndTime,
				&createTimeStr,
				&mirekyou.CreateApp,
				&mirekyou.CreateDevice,
				&mirekyou.CreateUser,
				&updateTimeStr,
				&mirekyou.UpdateApp,
				&mirekyou.UpdateDevice,
				&mirekyou.UpdateUser,
				&mirekyou.RepName,
				&mirekyou.DataType,
			)
			if err != nil {
				err = fmt.Errorf("error at scan mirekyou: %w", err)
				return nil, err
			}

			mirekyou.CreateTime, err = time.Parse(sqlite3impl.TimeLayout, createTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse create time %s in MIREKYOU: %w", createTimeStr, err)
				return nil, err
			}
			mirekyou.UpdateTime, err = time.Parse(sqlite3impl.TimeLayout, updateTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse update time %s in MIREKYOU: %w", updateTimeStr, err)
				return nil, err
			}
			if limitTime.Valid {
				parsedLimitTime, _ := time.Parse(sqlite3impl.TimeLayout, limitTime.String)
				mirekyou.LimitTime = &parsedLimitTime
			}
			if estimateStartTime.Valid {
				parsedEstimateStartTime, _ := time.Parse(sqlite3impl.TimeLayout, estimateStartTime.String)
				mirekyou.EstimateStartTime = &parsedEstimateStartTime
			}
			if estimateEndTime.Valid {
				parsedEstimateEndTime, _ := time.Parse(sqlite3impl.TimeLayout, estimateEndTime.String)
				mirekyou.EstimateEndTime = &parsedEstimateEndTime
			}
			mirekyous = append(mirekyous, mirekyou)
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("error at iterate rows: %w", err)
		return nil, err
	}
	return mirekyous, nil
}

// miReKyouInsertArgs はINSERTのパラメータを組み立てます。
func miReKyouInsertArgs(mirekyou MiReKyou) []any {
	return []any{
		mirekyou.IsDeleted,
		mirekyou.ID,
		mirekyou.TargetID,
		mirekyou.IsChecked,
		mirekyou.BoardName,
		nullableTimeString(mirekyou.LimitTime),
		nullableTimeString(mirekyou.EstimateStartTime),
		nullableTimeString(mirekyou.EstimateEndTime),
		mirekyou.CreateTime.Format(sqlite3impl.TimeLayout),
		mirekyou.CreateApp,
		mirekyou.CreateUser,
		mirekyou.CreateDevice,
		mirekyou.UpdateTime.Format(sqlite3impl.TimeLayout),
		mirekyou.UpdateApp,
		mirekyou.UpdateDevice,
		mirekyou.UpdateUser,
	}
}

// nullableTimeString はnilならnil、そうでなければSQLite用の時刻文字列を返します。
func nullableTimeString(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(sqlite3impl.TimeLayout)
}

// miReKyouTargetFilter はターゲットKyouの存在とワード検索の結果です。
type miReKyouTargetFilter struct {
	// allowAll はターゲット解決を行わずすべて通すかどうかです。
	// TX中の一時リポジトリのようにリポジトリ横断の解決ができない場面で使います。
	allowAll bool
	// reps は未削除で実在するターゲットIDを引くためのリポジトリ群です。
	// 全件をmapに写し取らないのは、入れ子になった委譲のたびに
	// アドレス表を全表スキャンすることになるためです。
	reps *GkillRepositories
	// useWordFilter はワード検索を行うかどうかです。
	useWordFilter bool
	// wordMatchTargetIDs はワード検索にヒットしたターゲットIDです。
	// メモと共有しているので書き換えてはいけません。
	wordMatchTargetIDs map[string]bool
}

// isMatch はMiReKyouが検索にヒットするかを返します。
func (f *miReKyouTargetFilter) isMatch(targetID string) bool {
	if f.allowAll {
		return true
	}
	latestDataRepositoryAddress, exist := f.reps.GetLatestDataRepositoryAddress(targetID)
	if !exist || latestDataRepositoryAddress.IsDeleted {
		return false
	}
	if f.useWordFilter && !f.wordMatchTargetIDs[targetID] {
		return false
	}
	return true
}

// newMiReKyouTargetFilter はターゲットKyouの存在確認とワード検索を行います。
// MiReKyouはタイトルを持たないため、ワード検索はターゲットKyouへ委譲します。
func newMiReKyouTargetFilter(ctx context.Context, repsWithoutMiReKyou *GkillRepositories, query *find.FindQuery) (*miReKyouTargetFilter, error) {
	if repsWithoutMiReKyou == nil {
		// リポジトリ群を辿れない場合はターゲット解決を行わない
		return &miReKyouTargetFilter{allowAll: true}, nil
	}

	if err := repsWithoutMiReKyou.EnsureLatestDataRepositoryAddresses(ctx); err != nil {
		err = fmt.Errorf("error at get all latest data repository addresses: %w", err)
		return nil, err
	}

	filter := &miReKyouTargetFilter{
		reps:               repsWithoutMiReKyou,
		wordMatchTargetIDs: map[string]bool{},
	}

	// ワードフィルタ: UseWordsが有効な場合、Targetに対してワード検索を実行しマッチしたIDを収集する
	filter.useWordFilter = isWordFilterEnabled(query)
	if filter.useWordFilter {
		// repsWithoutMiReKyou.Reps は cloneRepositoriesWithoutMiReKyou が
		// collectTargetDataRepositories() の結果を入れたものなので、同じ集合
		wordMatchTargetIDs, err := resolveMiReKyouWordMatchTargetIDs(ctx, repsWithoutMiReKyou.collectTargetDataRepositories(), query)
		if err != nil {
			return nil, err
		}
		filter.wordMatchTargetIDs = wordMatchTargetIDs
	}
	return filter, nil
}
