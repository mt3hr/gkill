package gpslogs

import "time"

type GoogleLocationHistoryData struct {
	Locations []*Location `json:"locations"`
}
type Location struct {
	Timestamp   string    `json:"timestamp"`
	LatitudeE7  int       `json:"latitudeE7"`
	LongitudeE7 int       `json:"longitudeE7"`
	Time        time.Time `json:"-"`
}
