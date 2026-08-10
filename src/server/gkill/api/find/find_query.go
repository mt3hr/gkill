// Package find は検索条件(FindQuery)と関連列挙型の型定義。
package find

import (
	"time"
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
	IsDeleted                   bool         `json:"is_deleted"`
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
	HideTimeIsTags              []string     `json:"hide_timeis_tags"`
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
	PlaingTime                  *time.Time   `json:"plaing_time"`
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
	// THUMBNAIL_IMAGE は base64 で埋め込まれており、実データでは1行あたり平均406KB・
	// 最大10MBで、227行の合計が90MBに達します。
	// サムネイルを使わない呼び出し（AIクライアント向けのMCP経路、
	// キャッシュ再構築など）では、DBから読む段階で外すために使います。
	//
	// FAVICON_IMAGE は対象外です。こちらは合計0.10MB・1行あたり平均0.5KBしかなく、
	// 外す意味がないため常に取得します。
	//
	// JSONには出しません（クライアントから指定させる項目ではないため）。
	ExcludeURLogThumbnailImage bool `json:"-"`
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
