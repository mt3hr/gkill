// ˅
package reps

import (
	"context"
	"time"
)

// ˄

type GPSLogRepository interface {
	GetAllGPSLogs(ctx context.Context) []*GPSLog

	GetGPSLogs(ctx context.Context, startTime time.Time, endTime time.Time) []*GPSLog

	GetPath(ctx context.Context, id string) string

	GetRepName(ctx context.Context) string

	UpdateCache(ctx context.Context)

	// ˅

	// ˄
}

// ˅

// ˄
