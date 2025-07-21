package account_state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type LatestDataRepositoryAddress struct {
	IsDeleted                              bool      `json:"is_deleted"`
	TargetID                               string    `json:"target_id"`
	LatestDataRepositoryName               string    `json:"latest_data_repository_name"`
	DataUpdateTime                         time.Time `json:"data_update_time"`
	LatestDataRepositoryAddressUpdatedTime time.Time `json:"latest_data_repository_address_updated_time"`
}

func (l *LatestDataRepositoryAddress) ContentHash() string {
	seed := []byte(fmt.Sprintf("%t_%s_%s_%s", l.IsDeleted, l.TargetID, l.LatestDataRepositoryName, l.DataUpdateTime.Format(sqlite3impl.TimeLayout)))
	sum := sha256.Sum256(seed)
	return hex.EncodeToString(sum[:])
}
