package api

import "time"

type GkillVersionData struct {
	CommitHash string    `json:"commit_hash"`
	BuildTime  time.Time `json:"build_time"`
	Version    string    `json:"version"`
}
