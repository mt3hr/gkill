package account_state

import (
	"context"

	_ "github.com/mattn/go-sqlite3"
)

type LatestDataRepositoryAddressDAO interface {
	GetAllLatestDataRepositoryAddresses(ctx context.Context) (map[string]*LatestDataRepositoryAddress, error)
	GetLatestDataRepositoryAddressesByRepName(ctx context.Context, repName string) (map[string]*LatestDataRepositoryAddress, error)
	GetLatestDataRepositoryAddress(ctx context.Context, targetID string) (*LatestDataRepositoryAddress, error)
	AddLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error)
	AddLatestDataRepositoryAddresses(ctx context.Context, latestDataRepositoryAddresses []*LatestDataRepositoryAddress) (bool, error)
	UpdateLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error)
	UpdateOrAddLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error)
	UpdateOrAddLatestDataRepositoryAddresses(ctx context.Context, latestDataRepositoryAddresses []*LatestDataRepositoryAddress) (bool, error)
	DeleteLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error)
	DeleteAllLatestDataRepositoryAddress(ctx context.Context) (bool, error)
	DeleteLatestDataRepositoryAddressInRep(ctx context.Context, repName string) (bool, error)
	Close(ctx context.Context) error
}
