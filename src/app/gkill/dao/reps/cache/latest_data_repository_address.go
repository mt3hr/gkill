package gkill_cache

import (
	"time"
)

type LatestDataRepositoryAddress struct {
	IsDeleted                              bool      `json:"is_deleted"`
	TargetID                               string    `json:"target_id"`
	TargetIDInData                         *string   `json:"target_id_in_data"`
	LatestDataRepositoryName               string    `json:"latest_data_repository_name"`
	DataUpdateTime                         time.Time `json:"data_update_time"`
	LatestDataRepositoryAddressUpdatedTime time.Time `json:"latest_data_repository_address_updated_time"`
}
