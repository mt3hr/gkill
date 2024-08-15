// ˅
package reps

import (
	"context"
	"time"
)

// ˄

type gpsLogRepositoryDirectoryImpl struct {
	// ˅

	// ˄
}

// ˅
func (g *gpsLogRepositoryDirectoryImpl) GetAllGPSLogs(ctx context.Context) ([]*GPSLog, error) {
	panic("notImplements")
}

func (g *gpsLogRepositoryDirectoryImpl) GetGPSLogs(ctx context.Context, startTime time.Time, endTime time.Time) ([]*GPSLog, error) {
	panic("notImplements")
}

func (g *gpsLogRepositoryDirectoryImpl) GetPath(ctx context.Context, id string) (string, error) {
	panic("notImplements")
}

func (g *gpsLogRepositoryDirectoryImpl) GetRepName(ctx context.Context) (string, error) {
	panic("notImplements")
}

func (g *gpsLogRepositoryDirectoryImpl) UpdateCache(ctx context.Context) error {
	panic("notImplements")
}

// ˄
