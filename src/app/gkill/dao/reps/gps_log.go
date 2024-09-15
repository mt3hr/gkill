package reps

import "time"

type GPSLog struct {
	RelatedTime time.Time `json:"related_time"`

	Longitude float64 `json:"longitude"`

	Latitude float64 `json:"latitude"`
}
