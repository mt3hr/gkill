package find

import (
	"time"
)

type FindQuery struct {
	UpdateCache       *bool         `json:"update_cache"`
	IsDeleted         *bool         `json:"is_deleted"`
	UseTags           *bool         `json:"use_tags"`
	UseReps           *bool         `json:"use_reps"`
	UseRepTypes       *bool         `json:"use_rep_types"`
	RepTypes          *[]string     `json:"rep_types"`
	UseIDs            *bool         `json:"use_ids"`
	IDs               *[]string     `json:"ids"`
	UseWords          *bool         `json:"use_words"`
	Words             *[]string     `json:"words"`
	WordsAnd          *bool         `json:"words_and"`
	NotWords          *[]string     `json:"not_words"`
	Reps              *[]string     `json:"reps"`
	Tags              *[]string     `json:"tags"`
	TagsAnd           *bool         `json:"tags_and"`
	UseTimeIs         *bool         `json:"use_timeis"`
	TimeIsWords       *[]string     `json:"timeis_words"`
	TimeIsNotWords    *[]string     `json:"timeis_not_words"`
	TimeIsWordsAnd    *bool         `json:"timeis_words_and"`
	UseTimeIsTags     *bool         `json:"use_timeis_tags"`
	TimeIsTags        *[]string     `json:"timeis_tags"`
	TimeIsTagsAnd     *bool         `json:"timeis_tags_and"`
	UseCalendar       *bool         `json:"use_calendar"`
	CalendarStartDate *time.Time    `json:"calendar_start_date"`
	CalendarEndDate   *time.Time    `json:"calendar_end_date"`
	UseMap            *bool         `json:"use_map"`
	MapRadius         *float64      `json:"map_radius"`
	MapLatitude       *float64      `json:"map_latitude"`
	MapLongitude      *float64      `json:"map_longitude"`
	IncludeCreateMi   *bool         `json:"include_create_mi"`
	IncludeCheckMi    *bool         `json:"include_check_mi"`
	IncludeLimitMi    *bool         `json:"include_limit_mi"`
	IncludeStartMi    *bool         `json:"include_start_mi"`
	IncludeEndMi      *bool         `json:"include_end_mi"`
	IncludeEndTimeIs  *bool         `json:"include_end_timeis"`
	UsePlaing         *bool         `json:"use_plaing"`
	PlaingTime        *time.Time    `json:"plaing_time"`
	UseUpdateTime     *bool         `json:"use_update_time"`
	UpdateTime        **time.Time   `json:"update_time"`
	IsImageOnly       *bool         `json:"is_image_only"`
	ForMi             *bool         `json:"for_mi"`
	UseMiBoardName    *bool         `json:"use_mi_board_name"`
	MiBoardName       *string       `json:"mi_board_name"`
	MiCheckState      *MiCheckState `json:"mi_check_state"`
	MiSortType        *MiSortType   `json:"mi_sort_type"`
}
