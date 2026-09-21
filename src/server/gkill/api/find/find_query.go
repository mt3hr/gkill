// Package find は検索条件(FindQuery)と関連列挙型の型定義。
//
// use_* フラグを廃して null 判定へ一本化した理由と却下案:
// documents/adr/0106-find-query-null-semantics.md
package find

// 編集前に読む: .claude/skills/gkill-find-query/SKILL.md（この領域の不変条件の正本）

import (
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find_word"
)

// FindQuery は Kyou 検索の条件。
//
// フィルタグループの有効/無効は値の null 判定で表す:
//   - nil（JSON では null またはキー欠落）= フィルタ未使用
//   - 非nilの空スライス（JSON では []）= フィルタ有効だが空指定
//     （Tags/Reps/IDs/RepTypes/TimeIsTags は0件、TimeIsWords は「任意のTimeIsに覆われたKyou」）
//
// 複数フィールドで構成されるグループの有効判定は HasXxxFilter ヘルパーを使うこと。
type FindQuery struct {
	UpdateCache                 bool         `json:"update_cache"`
	RepTypes                    []string     `json:"rep_types"`
	IDs                         []string     `json:"ids"`
	Words                       []string     `json:"words"`
	WordsAnd                    bool         `json:"words_and"`
	NotWords                    []string     `json:"not_words"`
	Reps                        []string     `json:"reps"`
	Tags                        []string     `json:"tags"`
	HideTags                    []string     `json:"hide_tags"`
	TagsAnd                     bool         `json:"tags_and"`
	TimeIsWords                 []string     `json:"timeis_words"`
	TimeIsNotWords              []string     `json:"timeis_not_words"`
	TimeIsWordsAnd              bool         `json:"timeis_words_and"`
	TimeIsTags                  []string     `json:"timeis_tags"`
	TimeIsTagsAnd               bool         `json:"timeis_tags_and"`
	CalendarStartDate           *time.Time   `json:"calendar_start_date"`
	CalendarEndDate             *time.Time   `json:"calendar_end_date"`
	MapRadius                   *float64     `json:"map_radius"`
	MapLatitude                 *float64     `json:"map_latitude"`
	MapLongitude                *float64     `json:"map_longitude"`
	IncludeCreateMi             bool         `json:"include_create_mi"`
	IncludeCheckMi              bool         `json:"include_check_mi"`
	IncludeLimitMi              bool         `json:"include_limit_mi"`
	IncludeStartMi              bool         `json:"include_start_mi"`
	IncludeEndMi                bool         `json:"include_end_mi"`
	IncludeEndTimeIs            bool         `json:"include_end_timeis"`
	PlayingTime                 *time.Time   `json:"playing_time"`
	UpdateTime                  *time.Time   `json:"update_time"`
	IsImageOnly                 bool         `json:"is_image_only"`
	ForMi                       bool         `json:"for_mi"`
	PeriodOfTimeStartTimeSecond *int64       `json:"period_of_time_start_time_second"`
	PeriodOfTimeEndTimeSecond   *int64       `json:"period_of_time_end_time_second"`
	PeriodOfTimeWeekOfDays      []WeekOfDays `json:"period_of_time_week_of_days"`
	MiBoardName                 *string      `json:"mi_board_name"`
	MiCheckState                MiCheckState `json:"mi_check_state"`
	MiSortType                  MiSortType   `json:"mi_sort_type"`
	OnlyLatestData              bool         `json:"only_latest_data"`
	IncludeDeletedData          bool         `json:"include_deleted_data"`

	// ExcludeURLogThumbnailImage は URLog の THUMBNAIL_IMAGE を取得しないことを指示します。
	//
	// THUMBNAIL_IMAGE は base64 で埋め込まれており、実データでは1行あたり平均数百KB・
	// 最大10MBで、数百行の合計が数十MBに達します。
	// サムネイルを使わない呼び出し（AIクライアント向けのMCP経路、
	// キャッシュ再構築など）では、DBから読む段階で外すために使います。
	//
	// FAVICON_IMAGE は対象外です。こちらは合計0.10MB・1行あたり平均0.5KBしかなく、
	// 外す意味がないため常に取得します。
	//
	// JSONには出しません（クライアントから指定させる項目ではないため）。
	ExcludeURLogThumbnailImage bool `json:"-"`

	// WordsSkipIDMatch は肯定語（Words）で ID 列を照合しないことを指示します。
	//
	// 通常の肯定語は「対象列に含む OR ID が語で始まる」で、除外語は ID を見ない。
	// 除外語を含む記録を落とすために除外語を**肯定語として**再検索する内部クエリ
	// （find_filter.go の除外語の再検査）では、この旗が無いと「ID が除外語で始まる記録」まで
	// 除外されて「除外語は ID を見ない」が破れる（`-a` で 1/16 の記録が消える）。
	//
	// SQL 側は sqlite3impl.GenerateFindSQLCommon が、Go 側は find_word.MatchLoweredWords に
	// 空の id を渡すことで従う。
	// JSONには出しません（クライアントから指定させる項目ではないため）。
	WordsSkipIDMatch bool `json:"-"`
}

// WithNormalizedWords は Words / NotWords / TimeIsWords / TimeIsNotWords の各語から前後の空白を落とし、
// 空になった語を捨てた浅いコピーを返します（find_word.NormalizeWords。nil は nil のまま、非nil は空でも非nil のまま）。
//
// Kyou 検索の入口（FindFilter.FindKyous）で1回だけ通します。空文字の語を通すと SQL の LIKE '%%' が
// 全件に一致し、除外語なら全件が消える。画面のパーサは空語を作らないが、MCP や API の直叩きでは届く。
//
// 呼び出し元の FindQuery とそのスライスは書き換えません（query は共有されるため。
// dao/reps/shared_find_query_mutation_test.go の約束）。
func (q *FindQuery) WithNormalizedWords() *FindQuery {
	copied := *q
	copied.Words = find_word.NormalizeWords(q.Words)
	copied.NotWords = find_word.NormalizeWords(q.NotWords)
	copied.TimeIsWords = find_word.NormalizeWords(q.TimeIsWords)
	copied.TimeIsNotWords = find_word.NormalizeWords(q.TimeIsNotWords)
	return &copied
}

// HasWordFilter はキーワード検索グループが有効かを返す。
// Words / NotWords のどちらかが非nilなら有効（両方空スライスならSQL条件なし=素通し）。
func (q *FindQuery) HasWordFilter() bool {
	return q.Words != nil || q.NotWords != nil
}

// HasTimeIsFilter はTimeIs検索グループが有効かを返す。
// TimeIsWords が空スライスの場合は「任意のTimeIsに覆われたKyouのみ」の意味になる。
func (q *FindQuery) HasTimeIsFilter() bool {
	return q.TimeIsWords != nil || q.TimeIsNotWords != nil
}

// HasCalendarFilter はカレンダー期間フィルタが有効かを返す。
func (q *FindQuery) HasCalendarFilter() bool {
	return q.CalendarStartDate != nil || q.CalendarEndDate != nil
}

// HasMapFilter は地図範囲フィルタが有効かを返す。緯度・経度・半径の3つが揃って初めて有効。
func (q *FindQuery) HasMapFilter() bool {
	return q.MapRadius != nil && q.MapLatitude != nil && q.MapLongitude != nil
}

// HasPeriodOfTimeFilter は時間帯フィルタが有効かを返す。
// PeriodOfTimeWeekOfDays は nil=曜日制限なし / 空スライス=0件 / 全曜日=制限なし。
func (q *FindQuery) HasPeriodOfTimeFilter() bool {
	return q.PeriodOfTimeStartTimeSecond != nil || q.PeriodOfTimeEndTimeSecond != nil ||
		q.PeriodOfTimeWeekOfDays != nil
}
