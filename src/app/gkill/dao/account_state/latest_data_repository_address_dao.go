package account_state

import (
	"context"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type LatestDataRepositoryAddressDAO interface {
	GetAllLatestDataRepositoryAddresses(ctx context.Context) (map[string]*LatestDataRepositoryAddress, error)
	GetLatestDataRepositoryAddressesByRepName(ctx context.Context, repName string) (map[string]*LatestDataRepositoryAddress, error)
	GetLatestDataRepositoryAddress(ctx context.Context, targetID string) (*LatestDataRepositoryAddress, error)
	GetLatestDataRepositoryAddressByUpdateTimeAfter(ctx context.Context, updateTime time.Time, limit int64) (map[string]*LatestDataRepositoryAddress, error)
	AddOrUpdateLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error)
	AddOrUpdateLatestDataRepositoryAddresses(ctx context.Context, latestDataRepositoryAddresses []*LatestDataRepositoryAddress) (bool, error)
	DeleteLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error)
	DeleteAllLatestDataRepositoryAddress(ctx context.Context) (bool, error)
	DeleteLatestDataRepositoryAddressInRep(ctx context.Context, repName string) (bool, error)
	UpdateLatestDataRepositoryAddressesData(ctx context.Context, latestDataRepositoryAddresses []*LatestDataRepositoryAddress) error
	Close(ctx context.Context) error
}
