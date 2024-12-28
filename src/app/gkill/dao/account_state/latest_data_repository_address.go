package account_state

import (
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type LatestDataRepositoryAddress struct {
	IsDeleted                bool      `json:"is_deleted"`
	TargetID                 string    `json:"target_id"`
	LatestDataRepositoryName string    `json:"latest_data_repository_name"`
	DataUpdateTime           time.Time `json:"data_update_time"`
}
