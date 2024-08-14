// ˅
package reps

import (
	"context"
	"time"
)

// ˄

type GPSLogRepositories struct {
	// ˅

	// ˄

	gpsLogRepositories []GPSLogRepository

	// ˅

	// ˄
}

func (g *GPSLogRepositories) GetAllGPSLogs(ctx context.Context) ([]*GPSLog, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (g *GPSLogRepositories) GetGPSLogs(ctx context.Context, startTime time.Time, endTime time.Time) ([]*GPSLog, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (g *GPSLogRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (g *GPSLogRepositories) GetRepName(ctx context.Context) (string, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (g *GPSLogRepositories) UpdateCache(ctx context.Context) error {
	// ˅
	panic("notImplements")
	// ˄
}

// ˅

// ˄
