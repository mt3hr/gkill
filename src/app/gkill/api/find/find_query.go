package find

import (
	"time"
)

type FindQuery struct {
	rawFindQueryString string
	UpdateCache        *bool
	IsDeleted          *bool
	UseIDs             *bool
	IDs                *[]string
	UseWords           *bool
	Words              *[]string
	WordsAnd           *bool
	NotWords           *[]string
	Reps               *[]string
	Tags               *[]string
	TagsAnd            *bool
	TimeIsWords        *[]string
	TimeIsNotWords     *[]string
	TimeIsWordsAnd     *bool
	TimeIsTags         *[]string
	TimeIsTagsAnd      *bool
	UseCalendar        *bool
	CalendarStartDate  *time.Time
	CalendarEndDate    *time.Time
	MapRadius          *float64
	MapLatitude        *float64
	MapLongitude       *float64
	IncludeCheckMi     *bool
	IncludeLimitMi     *bool
	IncludeStartMi     *bool
	IncludeEndMi       *bool
	IncludeEndTimeIs   *bool
	UsePlaing          *bool
	PlaingTime         *time.Time
}
