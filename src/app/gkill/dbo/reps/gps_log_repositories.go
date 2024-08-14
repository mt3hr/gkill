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

func (g *GPSLogRepositories) GetAllGPSLogs(ctx context.Context) []*GPSLog {
	// ˅

	// ˄
}

func (g *GPSLogRepositories) GetGPSLogs(ctx context.Context, startTime time.Time, endTime time.Time) []*GPSLog {
	// ˅

	// ˄
}

func (g *GPSLogRepositories) GetPath(ctx context.Context, id string) string {
	// ˅

	// ˄
}

func (g *GPSLogRepositories) GetRepName(ctx context.Context) string {
	// ˅

	// ˄
}

func (g *GPSLogRepositories) UpdateCache(ctx context.Context) {
	// ˅

	// ˄
}

// ˅

// ˄
