package account_state

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type cachedLatestDataRepositoryAddressSQLite3Impl struct {
	m                              *sync.Mutex
	latestDataRepositoryAddressDAO LatestDataRepositoryAddressDAO
	cache                          map[string]*LatestDataRepositoryAddress
}

func NewCachedLatestDataRepositoryAddressSQLite3Impl(latestDataRepositoryAddressDAO LatestDataRepositoryAddressDAO) (CachedLatestDataRepositoryAddressDAO, error) {
	cachedLatestDataRepositoryAddress := &cachedLatestDataRepositoryAddressSQLite3Impl{
		m:                              &sync.Mutex{},
		latestDataRepositoryAddressDAO: latestDataRepositoryAddressDAO,
	}
	cachedLatestDataRepositoryAddress.UpdateCache(context.Background())

	return cachedLatestDataRepositoryAddress, nil
}

func (l *cachedLatestDataRepositoryAddressSQLite3Impl) GetAllLatestDataRepositoryAddresses(ctx context.Context) (map[string]*LatestDataRepositoryAddress, error) {
	// l.m.Lock()
	// defer l.m.Unlock()
	return maps.Clone(l.cache), nil
}

func (l *cachedLatestDataRepositoryAddressSQLite3Impl) GetLatestDataRepositoryAddressesByRepName(ctx context.Context, repName string) (map[string]*LatestDataRepositoryAddress, error) {
	// l.m.Lock()
	// defer l.m.Unlock()
	latestDataRepositoryAddresses := map[string]*LatestDataRepositoryAddress{}
	for _, latestDataRepositoryAddress := range l.cache {
		if latestDataRepositoryAddress.LatestDataRepositoryName == repName {
			latestDataRepositoryAddresses[latestDataRepositoryAddress.TargetID] = latestDataRepositoryAddress
		}
	}
	return latestDataRepositoryAddresses, nil
}

func (l *cachedLatestDataRepositoryAddressSQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, targetID string) (*LatestDataRepositoryAddress, error) {
	// l.m.Lock()
	// defer l.m.Unlock()
	if latestDataRepositoryAddress, exist := l.cache[targetID]; exist {
		return latestDataRepositoryAddress, nil
	}
	return nil, nil
}

func (l *cachedLatestDataRepositoryAddressSQLite3Impl) GetLatestDataRepositoryAddressByUpdateTimeAfter(ctx context.Context, updateTime time.Time, limit int64) (map[string]*LatestDataRepositoryAddress, error) {
	// l.m.Lock()
	// defer l.m.Unlock()
	latestDataRepositoryAddresses := map[string]*LatestDataRepositoryAddress{}
	for _, latestDataRepositoryAddress := range l.cache {
		if latestDataRepositoryAddress.DataUpdateTime.After(updateTime) {
			latestDataRepositoryAddresses[latestDataRepositoryAddress.TargetID] = latestDataRepositoryAddress
		}
	}
	return latestDataRepositoryAddresses, nil
}

func (l *cachedLatestDataRepositoryAddressSQLite3Impl) AddOrUpdateLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	l.cache[latestDataRepositoryAddress.TargetID] = latestDataRepositoryAddress
	return l.latestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress)
}

func (l *cachedLatestDataRepositoryAddressSQLite3Impl) AddOrUpdateLatestDataRepositoryAddresses(ctx context.Context, latestDataRepositoryAddresses []*LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
		l.cache[latestDataRepositoryAddress.TargetID] = latestDataRepositoryAddress
	}
	return l.latestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddresses(ctx, latestDataRepositoryAddresses)
}

func (l *cachedLatestDataRepositoryAddressSQLite3Impl) DeleteLatestDataRepositoryAddress(ctx context.Context, latestDataRepositoryAddress *LatestDataRepositoryAddress) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	delete(l.cache, latestDataRepositoryAddress.TargetID)
	return l.latestDataRepositoryAddressDAO.DeleteLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress)
}

func (l *cachedLatestDataRepositoryAddressSQLite3Impl) DeleteAllLatestDataRepositoryAddress(ctx context.Context) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	l.cache = map[string]*LatestDataRepositoryAddress{}
	return l.latestDataRepositoryAddressDAO.DeleteAllLatestDataRepositoryAddress(ctx)
}

func (l *cachedLatestDataRepositoryAddressSQLite3Impl) DeleteLatestDataRepositoryAddressInRep(ctx context.Context, repName string) (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	deleteTargetIDs := []string{}
	for targetID, latestDataRepositoryAddress := range l.cache {
		if latestDataRepositoryAddress.LatestDataRepositoryName == repName {
			deleteTargetIDs = append(deleteTargetIDs, targetID)
		}
	}
	for _, targetID := range deleteTargetIDs {
		delete(l.cache, targetID)
	}
	return l.latestDataRepositoryAddressDAO.DeleteLatestDataRepositoryAddressInRep(ctx, repName)
}

func (l *cachedLatestDataRepositoryAddressSQLite3Impl) UpdateLatestDataRepositoryAddressesData(ctx context.Context, latestDataRepositoryAddresses []*LatestDataRepositoryAddress) error {
	for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
		l.cache[latestDataRepositoryAddress.TargetID] = latestDataRepositoryAddress
	}
	return l.latestDataRepositoryAddressDAO.UpdateLatestDataRepositoryAddressesData(ctx, latestDataRepositoryAddresses)
}

func (l *cachedLatestDataRepositoryAddressSQLite3Impl) UpdateCache(ctx context.Context) error {
	l.m.Lock()
	defer l.m.Unlock()
	latestDataRepositoryAddresses, err := l.latestDataRepositoryAddressDAO.GetAllLatestDataRepositoryAddresses(ctx)
	if err != nil {
		err = fmt.Errorf("error at update cache at cached latest data repository address sqlite3 impl: %w", err)
		return err
	}
	l.cache = latestDataRepositoryAddresses
	return nil
}

func (l *cachedLatestDataRepositoryAddressSQLite3Impl) Close(ctx context.Context) error {
	return l.latestDataRepositoryAddressDAO.Close(ctx)
}
